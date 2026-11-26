// SPDX-License-Identifier: Apache-2.0

// Package digest: evidence_budget_patch_endpoints.go holds the
// COMPARISON ENDPOINT RESOLUTION of the canonical patch specification.
//
// ACT-LEAMAS-RANGE-PATCH-BUDGET-AUTHORITY-01 AMENDMENT-01 (A2) and
// FINAL AUTHORITY CORRECTION (G3)
//
// The comparison endpoints are the AUTHORITY of the patch. A patch
// without resolved endpoints is not a patch of the historical range at
// all: `git diff` without comparison endpoints compares the mutable
// WORKTREE against the index and SUCCEEDS.
//
// The companion file evidence_budget_patch_spec.go holds the canonical
// PatchSpec itself.
package digest

import "strings"

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
//
// FINAL AUTHORITY CORRECTION (G3): the fallback endpoints are the
// caller's LITERAL revision expression. They are NOT a comparison
// authority: the caller must check HasComparison before executing the
// specification, so an unresolved endpoint produces explicit
// unavailable evidence rather than a successful comparison against
// mutable worktree state.
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
