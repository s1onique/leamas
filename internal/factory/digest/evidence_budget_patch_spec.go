// SPDX-License-Identifier: Apache-2.0

// Package digest: evidence_budget_patch_spec.go defines the CANONICAL
// PATCH SPECIFICATION for range-mode evidence.
//
// # ACT-LEAMAS-RANGE-PATCH-BUDGET-AUTHORITY-01 AMENDMENT-01
//
// The predecessor of this ACT generated a patch with one Git argument
// vector and published a "RetrievalPath:" line reconstructed by hand.
// Those two were NOT the same command:
//
//   - generation enabled rename/copy detection and scoped the diff to
//     BOTH the old and the new path;
//   - retrieval used neither, so it reproduced a different (often far
//     larger) patch, or a different patch entirely.
//
// Independently reproduced in a disposable repository: for a pure
// rename of a 20,000-line file, the destination-only retrieval command
// produced 20131 bytes while the rename-aware generation command
// produced 99 bytes.
//
// The canonical PatchSpec removes that divergence. ONE specification
// drives BOTH consumers:
//
//	Generate  -> spec.Args()          (executes the patch)
//	Publish   -> spec.RetrievalArgv() (serializes the SAME patch)
//
// The specification additionally pins:
//
//   - IMMUTABLE resolved comparison endpoints, so the retrieval
//     command keeps working after HEAD advances;
//   - LITERAL pathspec semantics (":(literal)"), so a path containing
//     glob metacharacters matches exactly one file instead of a
//     wildcard set;
//   - the rename/copy source path, so the base blob identity of a
//     renamed file resolves to its true preimage.
//
// The package deliberately owns no policy. It describes ONE patch.
package digest

import (
	"strings"
)

// PatchSpec is the canonical, immutable description of the unified
// patch for exactly one range file.
//
// A PatchSpec is a VALUE. Once constructed it never changes, and the
// same value always yields the same patch bytes from Git. That is what
// makes the published retrieval command a faithful reproduction of the
// embedded evidence.
type PatchSpec struct {
	// Base is the immutable base endpoint OID. Empty when the
	// endpoint could not be resolved.
	Base string
	// Head is the immutable head endpoint OID. Empty when the
	// endpoint could not be resolved.
	Head string
	// BaseFallback is the base endpoint as it appeared in the
	// caller's revision expression. It is used only when Base is
	// empty (resolution failure).
	BaseFallback string
	// HeadFallback is the head endpoint as it appeared in the
	// caller's revision expression. Used only when Head is empty.
	HeadFallback string
	// DotDot selects the two-dot comparison semantics ("A..B").
	DotDot bool
	// Path is the destination (post-change) path.
	Path string
	// From is the source (pre-change) path for a rename or copy.
	// Empty for ordinary modifications.
	From string
	// RenameSource is the pre-change path used for the BASE blob
	// identity lookup. For a rename or copy this is From; for an
	// ordinary modification it is Path.
	RenameSource string
}

// newPatchSpec builds the canonical specification for one range file.
//
// base/head are the RESOLVED comparison endpoints (see
// resolveRangeEndpoints). baseFallback/headFallback are the caller's
// original revision expression endpoints, retained only as a
// fallback when resolution failed.
//
// The three-dot form "A...B" is resolved by resolveRangeEndpoints to
// the merge base, so the specification always carries an explicit
// base and head and never has to re-derive three-dot semantics at
// retrieval time.
func newPatchSpec(base, head, baseFallback, headFallback string,
	dotDot bool, f RangeFile) PatchSpec {
	renameSource := f.Path
	if f.From != "" {
		renameSource = f.From
	}
	return PatchSpec{
		Base:         base,
		Head:         head,
		BaseFallback: baseFallback,
		HeadFallback: headFallback,
		DotDot:       dotDot,
		Path:         f.Path,
		From:         f.From,
		RenameSource: renameSource,
	}
}

// comparison returns the revision expression Git compares.
//
// When both endpoints are resolved, the expression is built from
// immutable OIDs. This is the property that keeps the published
// retrieval command valid AFTER HEAD advances: "HEAD~1..HEAD" would
// silently start describing a different range, while
// "<base-oid>..<head-oid>" always describes the same one.
func (s PatchSpec) comparison() string {
	base, head := s.Base, s.Head
	if base == "" {
		base = s.BaseFallback
	}
	if head == "" {
		head = s.HeadFallback
	}
	if base == "" || head == "" {
		return ""
	}
	if s.DotDot {
		return base + "..." + head
	}
	return base + ".." + head
}

// Args returns the canonical Git argument vector that produces the
// patch.
//
// The vector is:
//
//	git diff --unified=3 --find-renames=<n>% --find-copies=<n>% \
//	    <comparison> -- :(literal)<path> [:(literal)<from>]
//
// The literal pathspec magic is REQUIRED, not cosmetic: without it
// Git interprets the path as a wildcard expression and a file named
// "literal*.ts" also matches "literal-other.ts". The per-file
// evidence budget would then be applied to a patch that belongs to
// several files.
func (s PatchSpec) Args() []string {
	args := []string{"diff", "--unified=3"}
	args = append(args, detectArgs()...)
	if cmp := s.comparison(); cmp != "" {
		args = append(args, cmp)
	}
	args = append(args, "--")
	args = append(args, literalPathspec(s.Path))
	if s.From != "" && s.From != s.Path {
		args = append(args, literalPathspec(s.From))
	}
	return args
}

// rangeDiffArgs builds the argument vector for the path-scoped
// unified diff that backs the range-mode evidence budget.
//
// ACT-LEAMAS-RANGE-PATCH-BUDGET-AUTHORITY-01: the per-file budget
// applies to the ACTUAL unified patch. To measure the ACTUAL patch
// of a RENAME, the diff must be able to see BOTH the old and the
// new path. Scoping the diff to the new path alone makes Git
// synthesise a "new file" patch whose size is the whole blob, not
// the small rename patch that actually happened.
//
// AMENDMENT-01 (A2): this function is now a THIN ADAPTER over the
// canonical PatchSpec. The canonical specification owns the argument
// vector, the literal pathspec semantics, and the immutable
// endpoints; this adapter only supplies the caller's raw revision
// expression, which the canonical layer resolves.
//
// Deprecated in favour of rangePatchSpec, which produces the SAME
// argument vector together with the matching retrieval serialization.
func rangeDiffArgs(rangeSpec string, f RangeFile) []string {
	spec := rangePatchSpec(nil, "", rangeSpec, f, "", "")
	return spec.Args()
}

// rangeBaseIdentityPath returns the path used to resolve the BASE
// endpoint blob identity for `f`.
//
// AMENDMENT-01 (A3): for a rename or a copy the preimage lives at the
// SOURCE path, so the base identity must be looked up there. Looking
// it up at the destination path reports a MISSING blob even though
// the true preimage is present under the source name.
//
// For every other change kind the destination path IS the preimage
// path, so the lookup is unchanged.
func rangeBaseIdentityPath(f RangeFile) string {
	if f.From != "" && f.From != f.Path {
		return f.From
	}
	return f.Path
}

// rangePatchSpec builds the CANONICAL patch specification for one
// range file.
//
// The specification is the single authority for BOTH the generated
// patch and the published retrieval command.
//
// When runner is non-nil and the caller's revision expression names
// resolvable endpoints, the endpoints are resolved to IMMUTABLE OIDs
// so the retrieval command keeps describing the same range after HEAD
// advances. The caller may pass the already-resolved left/right
// endpoints (as the renderer does) to avoid a redundant lookup.
//
// AMENDMENT-01 (A2).
func rangePatchSpec(runner gitRunner, repoRoot, rangeSpec string,
	f RangeFile, left, right string) PatchSpec {
	base, head, baseFallback, headFallback, dotDot :=
		resolvePatchEndpoints(runner, repoRoot, rangeSpec, left, right)
	return newPatchSpec(base, head, baseFallback, headFallback,
		dotDot, f)
}

// resolvePatchEndpoints determines the immutable comparison endpoints
// and the fallback endpoints for a range expression.
//
// Precedence:
//
//  1. The already-resolved left/right supplied by the caller (the
//     renderer resolves them once per range).
//  2. A fresh resolution of the caller's revision expression, when a
//     runner is available.
//  3. The literal endpoints of the revision expression as a fallback.
//
// The three-dot form "A...B" is reported with DotDot=true so the
// canonical comparison keeps the merge-base semantics Git defines for
// it.
func resolvePatchEndpoints(runner gitRunner, repoRoot, rangeSpec,
	left, right string) (base, head, baseFallback, headFallback string,
	dotDot bool) {
	baseFallback, headFallback, ok := splitRangeSpec(rangeSpec)
	if !ok {
		baseFallback, headFallback = left, right
	}
	dotDot = strings.Contains(rangeSpec, "...")

	resolvedLeft, resolvedRight := left, right
	if resolvedLeft == "" || resolvedRight == "" {
		if runner != nil && repoRoot != "" && rangeSpec != "" {
			if l, r, resolved := resolveRangeEndpoints(runner, repoRoot,
				rangeSpec); resolved {
				resolvedLeft, resolvedRight = l, r
			}
		}
	}
	if resolvedLeft == "" || resolvedRight == "" {
		return "", "", baseFallback, headFallback, dotDot
	}

	// AMENDMENT-01 (A2): pin the endpoints to IMMUTABLE object IDs
	// so the published retrieval command keeps describing the same
	// historical range after HEAD advances.
	return resolveImmutableOID(runner, repoRoot, resolvedLeft),
		resolveImmutableOID(runner, repoRoot, resolvedRight),
		baseFallback, headFallback, dotDot
}

// formatRangeRetrieval returns the canonical retrieval serialization
// for the WHOLE range.
//
// AMENDMENT-01 (A2): the range-level retrieval command reproduces
// every patch the range contains. It uses the SAME comparison
// endpoints and the SAME diff options as the per-file
// specifications, so it is the whole-range counterpart of the
// per-file retrieval path.
//
// The command is scoped to the range only (no pathspec), because the
// whole range is what it reproduces.
//
// The per-file retrieval path is the PATHSCOPING suffix that is
// appended to this command.
func formatRangeRetrieval(base, head string, dotDot bool) string {
	cmp := ""
	if base != "" && head != "" {
		if dotDot {
			cmp = base + "..." + head
		} else {
			cmp = base + ".." + head
		}
	}
	args := []string{"diff", "--unified=3"}
	args = append(args, detectArgs()...)
	if cmp != "" {
		args = append(args, cmp)
	}
	parts := make([]string, 0, len(args))
	for _, a := range args {
		parts = append(parts, shellQuoteArg(a))
	}
	return "git " + strings.Join(parts, " ")
}

// literalPathspecMagic is Git's pathspec magic prefix that disables
// wildcard and colon-prefix interpretation.
const literalPathspecMagic = ":(literal)"

// literalPathspec returns the Git pathspec that matches EXACTLY one
// path.
//
// Git's ":(literal)" magic disables wildcard and colon-prefix
// interpretation for the remainder of the pathspec.
func literalPathspec(path string) string {
	return literalPathspecMagic + path
}

// resolveImmutableOID resolves a revision expression to its
// immutable object ID.
//
// AMENDMENT-01 (A2): the published retrieval command must keep
// describing the SAME historical range after HEAD advances. A
// symbolic endpoint such as "HEAD~1" silently starts describing a
// DIFFERENT range once the next commit lands, so the canonical
// specification pins the resolved OID instead.
//
// The resolver returns the input unchanged when Git cannot resolve it
// (an unborn branch, a malformed expression); the caller then falls
// back to the literal expression.
func resolveImmutableOID(runner gitRunner, repoRoot, rev string) string {
	if runner == nil || repoRoot == "" || rev == "" {
		return rev
	}
	out, err := runner.Run(repoRoot, []string{
		"rev-parse", "--verify", rev + "^{commit}"})
	if err != nil {
		return rev
	}
	oid := strings.TrimSpace(out)
	if !isHexOID(oid) {
		return rev
	}
	return oid
}

// isShellSafeWord reports whether `a` can be rendered as a bare shell
// word without any quoting.
//
// The safe set is restricted to characters that Git accepts literally
// in a pathspec and that the shell treats as ordinary word
// constituents. Anything else (spaces, quotes, glob characters, the
// ":(...)" magic prefix, control bytes) forces the quoted form.
func isShellSafeWord(a string) bool {
	for i := 0; i < len(a); i++ {
		c := a[i]
		switch {
		case c >= 'a' && c <= 'z':
		case c >= 'A' && c <= 'Z':
		case c >= '0' && c <= '9':
		case c == '_' || c == '-' || c == '.' || c == '/' || c == '+' ||
			c == '=' || c == '@' || c == ',' || c == ':' || c == '%':
		default:
			return false
		}
	}
	// A leading dash would be read as an option. The pathspec magic
	// prefix already prevents that for paths, but keep the rule
	// absolute.
	return !strings.HasPrefix(a, "-")
}
