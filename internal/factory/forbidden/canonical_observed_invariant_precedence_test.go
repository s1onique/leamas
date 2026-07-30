// SPDX-License-Identifier: Apache-2.0

package forbidden

import (
	"testing"

	"github.com/s1onique/leamas/internal/factory/checks"
)

// TestInvPrecedenceAnonymousEdgeHasNilCallerObject is the structural
// invariant that the real traversal never produces an anonymous edge
// with a non-nil callerObject. The function literal has no declared
// name in the package symbol table, so callerForUse returns nil.
func TestInvPrecedenceAnonymousEdgeHasNilCallerObject(t *testing.T) {
	fixture := newCanonicalFixture(t)
	protectedPkg := fixture.packagePath("protected")
	fixture.write("protected/protected.go", `package protected
func Cap() {}
`)
	fixture.write("caller/caller.go", `package caller
import p "example.test/policy/protected"
func Allowed() {
	func() {
		p.Cap()
	}()
}
`)
	symbol := fixtureSymbol(AuthorityLayerRaw, protectedPkg, "Cap", ProtectedPackageFunction, "")
	approval := fixtureApproval(fixture.packagePath("caller"), "Allowed", "", CallerKindPackageFunction, symbol, refDirectCall)
	result := fixture.run([]ProtectedSymbol{symbol}, []ApprovedCaller{approval})

	for _, edge := range result.ObservedEdges {
		if edge.Caller.Kind == CallerKindFunctionLiteral {
			if edge.callerObject != nil {
				t.Errorf("anonymous edge has non-nil callerObject: %#v", edge)
			}
			if !edge.hasOuterCaller {
				t.Errorf("anonymous edge missing outer caller: %#v", edge)
			}
		}
	}
}

// TestInvPrecedenceValidAnonymousDirectCallPreserved locks the
// preservation of the anonymous-caller finding when the observed
// reference class is internally valid. The implementation must remain
// on the anonymous-caller path and not regress into the invalid-class
// branch.
func TestInvPrecedenceValidAnonymousDirectCallPreserved(t *testing.T) {
	fixture := newCanonicalFixture(t)
	protectedPkg := fixture.packagePath("protected")
	fixture.write("protected/protected.go", `package protected
func Cap() {}
`)
	fixture.write("caller/caller.go", `package caller
import p "example.test/policy/protected"
func Allowed() {
	func() {
		p.Cap()
	}()
}
`)
	symbol := fixtureSymbol(AuthorityLayerRaw, protectedPkg, "Cap", ProtectedPackageFunction, "")
	approval := fixtureApproval(fixture.packagePath("caller"), "Allowed", "", CallerKindPackageFunction, symbol, refDirectCall)
	result := fixture.run([]ProtectedSymbol{symbol}, []ApprovedCaller{approval})

	if got := findingKindCount(result.Findings, "authority_policy_anonymous_caller"); got != 1 {
		t.Errorf("anonymous_caller count = %d, want 1", got)
	}
	if got := findingKindCount(result.Findings, "authority_policy_observed_reference_class_invalid"); got != 0 {
		t.Errorf("observed_reference_class_invalid count = %d, want 0", got)
	}
	if got := findingKindCount(result.Findings, "dupcode_bypass"); got != 0 {
		t.Errorf("dupcode_bypass count = %d, want 0", got)
	}
}

// TestInvPrecedenceValidAnonymousFunctionValuePreserved locks the
// preservation of FUNCTION_VALUE classification through the invariant
// precedence. The anonymous-caller finding is emitted with the
// FUNCTION_VALUE class.
func TestInvPrecedenceValidAnonymousFunctionValuePreserved(t *testing.T) {
	fixture := newCanonicalFixture(t)
	protectedPkg := fixture.packagePath("protected")
	fixture.write("protected/protected.go", `package protected
func Cap() {}
`)
	fixture.write("caller/caller.go", `package caller
import p "example.test/policy/protected"
func Allowed() {
	func() {
		f := p.Cap
		_ = f
	}()
}
`)
	symbol := fixtureSymbol(AuthorityLayerRaw, protectedPkg, "Cap", ProtectedPackageFunction, "")
	approval := fixtureApproval(fixture.packagePath("caller"), "Allowed", "", CallerKindPackageFunction, symbol, refDirectCall)
	result := fixture.run([]ProtectedSymbol{symbol}, []ApprovedCaller{approval})

	if got := findingKindCount(result.Findings, "authority_policy_anonymous_caller"); got != 1 {
		t.Errorf("anonymous_caller count = %d, want 1", got)
	}
	functionValueEdge := 0
	for _, edge := range result.ObservedEdges {
		if edge.Caller.Kind == CallerKindFunctionLiteral &&
			edge.ReferenceClass == refFunctionValue {
			functionValueEdge++
		}
	}
	if functionValueEdge != 1 {
		t.Errorf("function-value edges inside literal = %d, want 1", functionValueEdge)
	}
}

// TestInvPrecedenceValidNamedEdgePreserved confirms named-caller
// matching and validation remain unchanged. The named direct call is
// matched; no anonymous finding; no cascade.
func TestInvPrecedenceValidNamedEdgePreserved(t *testing.T) {
	fixture, protected, callerPkg := approvalFixture(t, `package caller
import p "example.test/policy/protected"
func Allowed() { p.Cap() }
`)
	approval := fixtureApproval(callerPkg, "Allowed", "", CallerKindPackageFunction, protected, refDirectCall)
	result := fixture.run([]ProtectedSymbol{protected}, []ApprovedCaller{approval})

	if got := findingKindCount(result.Findings, "authority_policy_anonymous_caller"); got != 0 {
		t.Errorf("anonymous_caller count = %d, want 0", got)
	}
	if got := findingKindCount(result.Findings, "authority_policy_observed_reference_class_invalid"); got != 0 {
		t.Errorf("observed_reference_class_invalid count = %d, want 0", got)
	}
	if got := findingKindCount(result.Findings, "dupcode_bypass"); got != 0 {
		t.Errorf("dupcode_bypass count = %d, want 0", got)
	}
	if result.Stats.ValidatedApprovals != 1 {
		t.Errorf("validated approvals = %d, want 1", result.Stats.ValidatedApprovals)
	}
}

// TestInvPrecedenceDotImportPreserved confirms DOT_IMPORT remains
// categorically observed and reported without internal-invariant
// findings.
func TestInvPrecedenceDotImportPreserved(t *testing.T) {
	fixture := newCanonicalFixture(t)
	protectedPkg := fixture.packagePath("protected")
	fixture.write("protected/protected.go", `package protected
func Cap() {}
`)
	fixture.write("dotcaller/dot.go", `package dotcaller
import . "example.test/policy/protected"
func Dot() { Cap() }
`)
	symbol := fixtureSymbol(AuthorityLayerRaw, protectedPkg, "Cap", ProtectedPackageFunction, "")
	result := fixture.run([]ProtectedSymbol{symbol}, nil)

	if got := findingKindCount(result.Findings, "dupcode_dot_import"); got != 1 {
		t.Errorf("dupcode_dot_import count = %d, want 1", got)
	}
	if got := findingKindCount(result.Findings, "authority_policy_observed_reference_class_invalid"); got != 0 {
		t.Errorf("observed_reference_class_invalid count = %d, want 0 (DOT_IMPORT is internally valid)", got)
	}
	if got := findingKindCount(result.Findings, "authority_policy_anonymous_caller"); got != 0 {
		t.Errorf("anonymous_caller count = %d, want 0", got)
	}
}

// TestInvPrecedenceValidAnonymousEdgeOuterApprovalNotMatched locks the
// preservation of anonymous-caller policy for a valid direct call. The
// outer approval is reported as stale (no legitimate source edge) but
// is NOT matched or validated.
func TestInvPrecedenceValidAnonymousEdgeOuterApprovalNotMatched(t *testing.T) {
	fixture := newCanonicalFixture(t)
	protectedPkg := fixture.packagePath("protected")
	fixture.write("protected/protected.go", `package protected
func Cap() {}
`)
	fixture.write("caller/caller.go", `package caller
import p "example.test/policy/protected"
func Approved() {
	func() {
		p.Cap()
	}()
}
`)
	symbol := fixtureSymbol(AuthorityLayerRaw, protectedPkg, "Cap", ProtectedPackageFunction, "")
	approval := fixtureApproval(fixture.packagePath("caller"), "Approved", "", CallerKindPackageFunction, symbol, refDirectCall)
	result := fixture.run([]ProtectedSymbol{symbol}, []ApprovedCaller{approval})

	if got := findingKindCount(result.Findings, "authority_policy_anonymous_caller"); got != 1 {
		t.Errorf("anonymous_caller count = %d, want 1", got)
	}
	if got := findingKindCount(result.Findings, "authority_policy_observed_reference_class_invalid"); got != 0 {
		t.Errorf("observed_reference_class_invalid count = %d, want 0", got)
	}
	if got := findingKindCount(result.Findings, "authority_policy_stale_approval"); got != 1 {
		t.Errorf("stale_approval count = %d, want 1", got)
	}
	if result.Stats.ValidatedApprovals != 0 {
		t.Errorf("validated approvals = %d, want 0 (anonymous edge never matches)", result.Stats.ValidatedApprovals)
	}
}

// findingKindCount returns the count of checks.Finding with the given
// kind. It is independent of the canonical-finding seam used by other
// pipeline tests.
func findingKindCount(findings []checks.Finding, kind string) int {
	count := 0
	for _, finding := range findings {
		if finding.Kind == kind {
			count++
		}
	}
	return count
}
