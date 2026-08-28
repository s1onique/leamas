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
type EvidenceFile struct {
	Path string
	Size int64
}

// EvidenceOverlay is the rendered overlay: the surviving or
// rejected roots, the sorted list of files, and the running
// aggregate byte total used to enforce MaxEvidenceAggregateBytes.
type EvidenceOverlay struct {
	Roots      []EvidenceRoot
	Files      []EvidenceFile
	Aggregated int64
}

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
// roots plus per-root rejection reasons.
//
// Rules enforced:
//
//   - empty / whitespace-only paths are rejected
//   - absolute paths (POSIX and Windows forms) are rejected
//   - filepath.Clean followed by lexical containment: `..`,
//     leading `..`, and `.git` (or any descendant) are rejected
//   - each existing component of the path must be a non-symlink,
//     regular directory or a non-existent leaf; symlinks anywhere
//     along the existing chain fail closed
//
// Duplicates are deduplicated; the surviving ordering follows the
// first occurrence so command-line order is preserved.
//
// If ANY root is rejected the function returns a non-nil error
// that aggregates each rejection reason. Callers that want
// partial success must inspect the returned slice directly.
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

		// Lexical containment: refuse escapes and `.git` early so a
		// pathological input never reaches a filesystem probe.
		cleaned := filepath.Clean(r)
		if cleaned == ".." || strings.HasPrefix(cleaned, "../") {
			errs = append(errs, fmt.Sprintf(
				"%s refusing escape %q", evidenceRejectedPrefix, raw))
			roots = append(roots, EvidenceRoot{Path: cleaned, Rejected: true, Reason: "path traversal"})
			continue
		}
		if cleaned == ".git" || strings.HasPrefix(cleaned, ".git/") ||
			cleaned == ".gitignore" || strings.HasPrefix(cleaned, ".gitignore/") {
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

		// Surviving roots are slash-canonicalised.
		canon := filepath.ToSlash(cleaned)
		if seenSet[canon] {
			continue
		}
		seenSet[canon] = true
		roots = append(roots, EvidenceRoot{Path: canon})
	}

	if len(errs) > 0 {
		return roots, fmt.Errorf("%s rejected %d evidence path(s): %s",
			evidenceRejectedPrefix, len(errs), strings.Join(errs, "; "))
	}
	return roots, nil
}

// rejectSymlinkChain walks every existing component of `target`
// (interpreted relative to `canonicalRoot`) and rejects any that
// resolves through a symlink. The walk stops at the first missing
// component (the leaf may not exist yet).
//
// The chain check uses Lstat per segment; macOS users (where
// `/var` is a symlink to `/private/var`) get the canonical form
// because the repository root itself is resolved before the walk.
func rejectSymlinkChain(canonicalRoot, target string) error {
	rel, rerr := filepath.Rel(canonicalRoot, target)
	if rerr != nil {
		return fmt.Errorf("%s cannot resolve %q relative to repo root: %w",
			evidenceRejectedPrefix, target, rerr)
	}
	if rel == "." {
		// Targeting the repo root itself is allowed as a
		// degenerate evidence root (collects the entire
		// ignored tree under .gitignore / .git/info/exclude).
		return nil
	}
	cursor := canonicalRoot
	for _, seg := range splitPathSegments(rel) {
		cursor = filepath.Join(cursor, seg)
		lst, lerr := os.Lstat(cursor)
		if lerr != nil {
			if errors.Is(lerr, os.ErrNotExist) {
				return nil
			}
			return fmt.Errorf("%s cannot inspect %q: %w",
				evidenceRejectedPrefix, cursor, lerr)
		}
		if lst.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("%s refusing evidence path that traverses symlink %q",
				evidenceRejectedPrefix, cursor)
		}
	}
	return nil
}

// splitPathSegments splits a relative path into its slash-aware
// segments without producing "." or "" entries; it tolerates the
// platform separator so paths produced on Windows survive on Linux
// during cross-platform test runs.
func splitPathSegments(rel string) []string {
	if rel == "" || rel == "." {
		return nil
	}
	parts := strings.Split(filepath.ToSlash(rel), "/")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p == "" || p == "." {
			continue
		}
		out = append(out, p)
	}
	return out
}

// CollectEvidenceOverlay runs the per-root Git query and walks the
// filesystem to confirm each reported file is a non-symlink,
// regular, in-repository file at or under one of the requested
// roots. The output is sorted by repository-relative path.
// Rejected roots are reported in the overlay slice but
// contribute no files; callers may surface them via rendering.
//
// Files larger than MaxEvidencePerFileBytes are skipped; once the
// running aggregate hits MaxEvidenceAggregateBytes the per-root
// walk stops adding files. The caps guarantee that a single
// evidence overlay cannot accidentally ingest an unbounded
// ignored namespace.
func CollectEvidenceOverlay(repoRoot string, roots []EvidenceRoot) (EvidenceOverlay, error) {
	if repoRoot == "" {
		return EvidenceOverlay{Roots: roots}, fmt.Errorf("%s non-empty repoRoot required", evidenceRejectedPrefix)
	}
	canonicalRoot := filepath.Clean(repoRoot)
	if resolved, eerr := filepath.EvalSymlinks(repoRoot); eerr == nil {
		canonicalRoot = filepath.Clean(resolved)
	}

	overlay := EvidenceOverlay{Roots: roots}
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
			info, ierr := inspectEvidenceFile(canonicalRoot, p)
			if ierr != nil {
				// Skip un-inspectable entries (race,
				// permission, special file). The
				// remaining overlay is still truthful.
				continue
			}
			if info.Size > MaxEvidencePerFileBytes {
				// Skip oversize: too large to safely
				// preview under MVP budgets.
				continue
			}
			if overlay.Aggregated+info.Size > MaxEvidenceAggregateBytes {
				// Hit the aggregate cap; stop adding
				// more files to preserve the safety
				// guarantee for this root.
				break
			}
			overlay.Files = append(overlay.Files, EvidenceFile{
				Path: p,
				Size: info.Size,
			})
			overlay.Aggregated += info.Size
			seen[p] = true
		}
	}
	sort.Slice(overlay.Files, func(i, j int) bool {
		return overlay.Files[i].Path < overlay.Files[j].Path
	})
	return overlay, nil
}

// evidenceEntry is the in-memory form of a single path Git listed
// under an evidence root. Only regular files survive inspect.
type evidenceEntry struct {
	Size int64
}

// inspectEvidenceFile classifies a single relative path under
// canonicalRoot. Only non-symlink, regular files survive;
// symlinks, directories, FIFOs, sockets, device nodes, and
// missing entries are silently dropped.
func inspectEvidenceFile(canonicalRoot, relPath string) (evidenceEntry, error) {
	full := filepath.Join(canonicalRoot, filepath.FromSlash(relPath))
	lst, lerr := os.Lstat(full)
	if lerr != nil {
		return evidenceEntry{}, lerr
	}
	mode := lst.Mode()
	if mode&os.ModeSymlink != 0 {
		return evidenceEntry{}, fmt.Errorf("symlink: %s", relPath)
	}
	if !mode.IsRegular() {
		return evidenceEntry{}, fmt.Errorf("non-regular: %s", relPath)
	}
	return evidenceEntry{Size: lst.Size()}, nil
}
