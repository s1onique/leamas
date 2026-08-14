// SPDX-License-Identifier: Apache-2.0

// Package digest: gate_evidence_parent_act_test.go covers the
// CORRECTION02 parent-act regression at the binding-function level.
// The matrix test row 18 covers the same invariant at the
// classifier level.
package digest

import (
	"testing"
)

// TestGateSummary_DigestScopeBindingParentActOnly locks the
// pure scope-binding function: parent_act does NOT match
// digestActID when scope_id is empty. This is the lowest-level
// regression for the CORRECTION02 fix.
func TestGateSummary_DigestScopeBindingParentActOnly(t *testing.T) {
	t.Parallel()
	gate := GateSummaryIdentity{
		ScopeID:   "",
		ParentAct: testActA,
	}
	digest := DigestAuthority{
		ActID: testActA, // matches parent_act
	}

	got := digestScopeBinding(gate, digest)
	if got != ScopeUnbound {
		t.Errorf("digestScopeBinding with empty scope_id: got %v want %v", got, ScopeUnbound)
	}

	// Negative assertion: must NOT be ScopeMatch.
	if got == ScopeMatch {
		t.Errorf("parent_act MUST NOT establish scope match")
	}
}
