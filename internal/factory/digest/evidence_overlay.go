// SPDX-License-Identifier: Apache-2.0

// Package digest: evidence_overlay.go implements the local evidence
// overlay introduced by ACT-LEAMAS-LOCAL-EVIDENCE-OVERLAY-BINARY-MVP01.
//
// The overlay is an explicitly scoped, opt-in side-channel that
// surfaces ignored-untracked files alongside the ordinary subject
// digest. The MVP does not promote ignored evidence into the
// subject manifest. The two domains stay separate:
//
//	subject  = Git-visible change set
//	evidence = explicitly authorized, ignored-untracked local state
//
// All path-safety, symlink, redaction, and budget invariants from
// ACT SS7, 8, 9, 10 are enforced here. The collector uses Git as
// the truth source for "what is ignored-untracked" -- never
// re-implements `.gitignore` semantics from the filesystem.
package digest

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Conservative aggregate and per-file byte caps for the MVP.
// These are deliberately engineering choices, not closure
// invariants. A later ACT may formalize configurable limits.
const (
	MaxEvidenceAggregateBytes int64 = 8 * 1024 * 1024 // 8 MiB
	MaxEvidencePerFileBytes   int64 = 1024 * 1024     // 1 MiB
)

// EvidenceRoot is one requested root, classified by the path
// safety pre-flight. Rejected roots carry a short Reason so the
// CLI can surface an actionable diagnostic; the collection step
// silently drops rejected roots and the renderer still surfaces
// them so reviewers can see what was refused.
type EvidenceRoot struct {
	// Path is the canonical, repo-relative slash form of the
	// requested root, used as the Git pathspec.
	Path string
	// Rejected is true when the path failed path-safety.
	Rejected bool
	// Reason describes why a rejected root was rejected.
	Reason string
}

// EvidenceFile is one ignored-untracked path surfaced under an
// authorized root. Path is repo-relative, slash-separated.
//
// Identity is the os.FileInfo captured by the collector at
// inspect time. It is internal package state used by the
// renderer to verify that the filesystem object actually read
// is the same object that was authorized at collect time.
// It MUST NOT be exposed to the rendered digest or the
// external contract.
type EvidenceFile struct {
	Path     string
	Size     int64
	Identity os.FileInfo
}

// EvidenceOverlay is the rendered overlay: the surviving or
// rejected roots, the sorted list of files, the running
// aggregate byte total used to enforce MaxEvidenceAggregateBytes,
// and the canonical repository root that bound the collection.
// RepoRoot MUST be populated by CollectEvidenceOverlay so the
// renderer never reconstructs filesystem authority from the
// process current working directory.
type EvidenceOverlay struct {
	RepoRoot   string
	Roots      []EvidenceRoot
	Files      []EvidenceFile
	Omitted    []EvidenceOmission
	Truncated  bool
	Aggregated int64
}

// EvidenceOmission is one excluded path with a stable reason
// code. The renderer uses Omitted to surface any file Git
// classified as ignored-untracked under an authorized root but
// refused at collection time. Safety caps, symlinks, special
// files, and inspect failures all produce a row here; nothing is
// silently dropped.
type EvidenceOmission struct {
	Path   string
	Reason string
}

// Stable evidence-omission reason codes. These are the only
// values a caller should rely on; the renderer emits them
// verbatim so downstream reviewers can parse without prose.
const (
	OmitInspectFailed  = "inspect_failed"
	OmitSymlink        = "symlink"
	OmitNonRegular     = "non_regular"
	OmitFileTooLarge   = "file_too_large"
	OmitAggregateLimit = "aggregate_limit"
)

// evidenceRejectedPrefix prefixes a rejection reason so callers
// can surface an actionable diagnostic without parsing prose.
const evidenceRejectedPrefix = "evidence:"

// EvidenceAggregateTotal reports the live aggregate byte total for
// the overlay; used by tests to verify the safety ceiling.
func (o EvidenceOverlay) EvidenceAggregateTotal() int64 { return o.Aggregated }

// IsEvidenceRequested reports whether the caller asked for any
// --evidence-path roots. It is the canonical gate used by the
// digest renderers to decide whether to include the overlay
// section.
func IsEvidenceRequested(opts Options) bool {
	return len(opts.EvidencePaths) > 0
}

// ValidateEvidenceRoots canonicalises each requested path, rejects
// any root that violates path safety, and returns the surviving
// roots. The function is fail-closed: ANY rejection (absolute
// path, `..` segment, `.git`, symlink chain) returns a non-nil
// error so the digest cannot partially authorize.
//
// Rules enforced:
//
//   - empty / whitespace-only paths are rejected
//   - absolute paths (POSIX and Windows forms) are rejected
//   - any segment equal to `..` is rejected, even when
//     `filepath.Clean` would normalize inside the repository;
//     the rejection is segment-aware and runs BEFORE Clean
//   - `.git` and any descendant of it are rejected as
//     Git metadata; ordinary working-tree files like
//     `.gitignore` are NOT metadata and are accepted
//   - each existing component of the path must be a non-symlink,
//     regular directory or a non-existent leaf; symlinks anywhere
//     along the existing chain fail closed
//
// Filenames that merely contain the `..` character sequence
// (for example `report..json`) are NOT rejected; the rule only
// applies to path segments that resolve to `..`.
//
// Duplicates are deduplicated; the surviving ordering follows the
// first occurrence so command-line order is preserved.
//
// On ANY rejection the function returns a non-nil error that
// aggregates every reason. The returned slice still lists the
// offending roots (with Rejected=true) for diagnostic use.
func ValidateEvidenceRoots(repoRoot string, requested []string) ([]EvidenceRoot, error) {
	if repoRoot == "" {
		return nil, fmt.Errorf("%s non-empty repoRoot required", evidenceRejectedPrefix)
	}
	cleanRoot := filepath.Clean(repoRoot)
	absRoot, absErr := filepath.Abs(cleanRoot)
	if absErr != nil {
		return nil, fmt.Errorf("%s resolve repoRoot: %w", evidenceRejectedPrefix, absErr)
	}
	canonicalRoot := absRoot
	if resolved, eerr := filepath.EvalSymlinks(absRoot); eerr == nil {
		canonicalRoot = filepath.Clean(resolved)
	}

	var (
		roots   []EvidenceRoot
		errs    []string
		seenSet = make(map[string]bool)
	)
	for _, raw := range requested {
		r := strings.TrimSpace(raw)
		if r == "" {
			errs = append(errs, fmt.Sprintf("%s empty --evidence-path", evidenceRejectedPrefix))
			roots = append(roots, EvidenceRoot{Path: r, Rejected: true, Reason: "empty path"})
			continue
		}
		if filepath.IsAbs(r) || strings.HasPrefix(r, "/") || strings.HasPrefix(r, `\`) {
			errs = append(errs, fmt.Sprintf(
				"%s refusing absolute evidence path %q", evidenceRejectedPrefix, raw))
			roots = append(roots, EvidenceRoot{Path: r, Rejected: true, Reason: "absolute path"})
			continue
		}

		// C5: segment-aware `..` rejection. The check runs BEFORE
		// `filepath.Clean` so `ignored-a/../etc` cannot sneak
		// past the lexical containment guard by normalizing to
		// `etc`. Filenames merely containing `..` as a literal
		// sequence (for example `report..json`) are unaffected
		// because their path segments never equal `..`.
		if hasDotDotSegment(r) {
			errs = append(errs, fmt.Sprintf(
				"%s refusing escape %q", evidenceRejectedPrefix, raw))
			roots = append(roots, EvidenceRoot{Path: r, Rejected: true, Reason: "path traversal"})
			continue
		}

		// C4: `.git` and any descendant are Git metadata; reject
		// them. `.gitignore` is an ordinary working-tree file
		// (NOT metadata) and is therefore accepted.
		cleaned := filepath.ToSlash(filepath.Clean(r))
		if cleaned == ".git" || strings.HasPrefix(cleaned, ".git/") {
			errs = append(errs, fmt.Sprintf(
				"%s refusing Git metadata path %q", evidenceRejectedPrefix, raw))
			roots = append(roots, EvidenceRoot{Path: cleaned, Rejected: true, Reason: "Git metadata"})
			continue
		}

		// Symlink walk: walk up the path from the canonical root;
		// any existing component that resolves through a symlink
		// fails closed. Non-existent leaves are acceptable because
		// the user may have supplied an evidence root they are
		// about to populate.
		abandoned := filepath.Join(canonicalRoot, filepath.FromSlash(cleaned))
		if err := rejectSymlinkChain(canonicalRoot, abandoned); err != nil {
			errs = append(errs, err.Error())
			roots = append(roots, EvidenceRoot{Path: cleaned, Rejected: true, Reason: "symlink"})
			continue
		}

		if seenSet[cleaned] {
			continue
		}
		seenSet[cleaned] = true
		roots = append(roots, EvidenceRoot{Path: cleaned})
	}

	if len(errs) > 0 {
		return roots, fmt.Errorf("%s rejected %d evidence path(s): %s",
			evidenceRejectedPrefix, len(errs), strings.Join(errs, "; "))
	}
	return roots, nil
}

// hasDotDotSegment is implemented in evidence_overlay_path.go.
// It reports whether any slash-separated segment of `raw` is
// exactly `..`, and is the segment-aware C5 rejector.

// rejectSymlinkChain is implemented in evidence_overlay_path.go.
// It walks every existing component of a target path and
// rejects any segment that resolves through a symlink.

// splitPathSegments is implemented in evidence_overlay_path.go.
// It returns the slash-aware segments of a relative path
// without "." or empty entries.

// CollectEvidenceOverlay runs the per-root Git query and walks the
// filesystem to confirm each reported file is a non-symlink,
// regular, in-repository file at or under one of the requested
// roots. The output is sorted by repository-relative path.
// Rejected roots are reported in the overlay slice but
// contribute no files; callers may surface them via rendering.
//
// Files larger than MaxEvidencePerFileBytes are recorded as
// EvidenceOmission{OmitFileTooLarge}. Inspection failures are
// recorded with a stable reason code. Once the running aggregate
// hits MaxEvidenceAggregateBytes the per-root walk stops adding
// files and sets overlay.Truncated=true so the renderer can
// surface incompleteness truthfully. The overlay.RepoRoot is
// always populated so the renderer never reconstructs filesystem
// authority from the caller's cwd.
func CollectEvidenceOverlay(repoRoot string, roots []EvidenceRoot) (EvidenceOverlay, error) {
	if repoRoot == "" {
		return EvidenceOverlay{Roots: roots}, fmt.Errorf("%s non-empty repoRoot required", evidenceRejectedPrefix)
	}
	canonicalRoot := filepath.Clean(repoRoot)
	if resolved, eerr := filepath.EvalSymlinks(repoRoot); eerr == nil {
		canonicalRoot = filepath.Clean(resolved)
	}

	overlay := EvidenceOverlay{RepoRoot: canonicalRoot, Roots: roots}
	seen := make(map[string]bool)
	for _, root := range roots {
		if root.Rejected {
			continue
		}
		gitOut, gerr := RunGit(canonicalRoot, []string{
			"ls-files", "-z", "--others", "--ignored",
			"--exclude-standard", "--", root.Path,
		})
		if gerr != nil {
			return overlay, fmt.Errorf("%s git ls-files for root %q: %w",
				evidenceRejectedPrefix, root.Path, gerr)
		}
		for _, p := range splitNULList(gitOut) {
			if p == "" {
				continue
			}
			if seen[p] {
				continue
			}
			// Strict containment: only file paths at or
			// under the requested root qualify. Git's
			// pathspec is anchored; defensive check guards
			// against non-default `git config` quirks.
			if root.Path != "." && !strings.HasPrefix(p, root.Path+"/") && p != root.Path {
				continue
			}
			seen[p] = true
			info, inspectErr := inspectEvidenceFile(canonicalRoot, p)
			if inspectErr != nil {
				// C3 + C8: never drop silently. The
				// classifier uses typed sentinel
				// errors so this decision is
				// independent of error-string content.
				reason := OmitInspectFailed
				switch {
				case InspectSymlink(inspectErr):
					reason = OmitSymlink
				case InspectNonRegular(inspectErr):
					reason = OmitNonRegular
				}
				overlay.Omitted = append(overlay.Omitted, EvidenceOmission{
					Path: p, Reason: reason,
				})
				continue
			}
			if info.Size > MaxEvidencePerFileBytes {
				overlay.Omitted = append(overlay.Omitted, EvidenceOmission{
					Path: p, Reason: OmitFileTooLarge,
				})
				continue
			}
			if overlay.Aggregated+info.Size > MaxEvidenceAggregateBytes {
				// C3: aggregate cap reached.
				// Record this file as omitted so the
				// reviewer can see we stopped, and
				// mark the overlay truncated.
				overlay.Omitted = append(overlay.Omitted, EvidenceOmission{
					Path: p, Reason: OmitAggregateLimit,
				})
				overlay.Truncated = true
				continue
			}
			overlay.Files = append(overlay.Files, EvidenceFile{
				Path:     p,
				Size:     info.Size,
				Identity: info.Info,
			})
			overlay.Aggregated += info.Size
		}
	}
	sort.Slice(overlay.Files, func(i, j int) bool {
		return overlay.Files[i].Path < overlay.Files[j].Path
	})
	sort.Slice(overlay.Omitted, func(i, j int) bool {
		return overlay.Omitted[i].Path < overlay.Omitted[j].Path
	})
	return overlay, nil
}

// evidenceEntry is the in-memory form of a single path Git listed
// under an evidence root. Only regular files survive inspect.
// Info carries the Lstat result so the collector can hand the
// inode-bearing os.FileInfo to the renderer, which will use
// os.SameFile to confirm the object actually opened is still
// the same object that was authorized at collect time.
type evidenceEntry struct {
	Size int64
	Info os.FileInfo
}

// inspectEvidenceFile is declared in evidence_overlay_inspect.go.
// It is the production-classify primitive used by the collector
// to convert a relative path into a typed inspection result.
