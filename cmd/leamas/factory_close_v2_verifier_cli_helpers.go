// SPDX-License-Identifier: Apache-2.0

package main

// factory_close_v2_verifier_cli_helpers.go isolates the small
// helpers (sortedV2DiagsForText, canonicalWorktreesFrom,
// isV2VerifierOutputPathError) used by the verifier CLI from
// the top-level command dispatcher in
// factory_close_v2_verifier_cli.go. Splitting along the
// helper boundary keeps each file under the LLM-friendliness
// 400-line threshold.

import (
	"sort"

	"github.com/s1onique/leamas/internal/factory/closure"
)

// canonicalWorktreesFrom converts the internal inventory into
// the public CanonicalWorktree slice expected by
// PrepareVerifierOutput. The slice is reused verbatim so the
// CLI never has to canonicalize the inventory twice.
func canonicalWorktreesFrom(inv closure.RepositoryWorktreeInventory) []closure.CanonicalWorktree {
	roots := inv.RootsView()
	out := make([]closure.CanonicalWorktree, len(roots))
	for i, r := range roots {
		out[i] = closure.CanonicalWorktree{Path: r}
	}
	return out
}

// isV2VerifierOutputPathError reports whether err
// originates from the output-path resolver.
func isV2VerifierOutputPathError(err error) bool {
	var vErr *closure.V2VerifierError
	if !isErrorsAs(err, &vErr) {
		return false
	}
	for _, d := range vErr.Diags {
		if d.Code == closure.V2VerifierOutputPathNotDetached {
			return true
		}
	}
	return false
}

// sortedV2DiagsForText returns the diagnostics in a
// deterministic order for text rendering: by Code, then by
// PropertyName, then by Message.
func sortedV2DiagsForText(in closure.V2VerifierDiagnostics) closure.V2VerifierDiagnostics {
	out := make(closure.V2VerifierDiagnostics, len(in))
	copy(out, in)
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Code != out[j].Code {
			return string(out[i].Code) < string(out[j].Code)
		}
		if out[i].PropertyName != out[j].PropertyName {
			return out[i].PropertyName < out[j].PropertyName
		}
		return out[i].Message < out[j].Message
	})
	return out
}
