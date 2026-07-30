// SPDX-License-Identifier: Apache-2.0

package forbidden

import (
	"strings"
	"testing"

	"github.com/s1onique/leamas/internal/factory/checks"
)

// namedCallerFixture builds a single-policy fixture with a raw protected
// symbol and a caller package whose source is supplied. The caller
// package path and protected symbol are returned so tests can construct
// approvals.
func namedCallerFixture(t *testing.T, callerSource string) (*canonicalFixture, ProtectedSymbol, string) {
	t.Helper()
	fixture := newCanonicalFixture(t)
	fixture.write("protected/protected.go", `package protected
func Cap() {}
func Other() {}
type Runner struct{}
func (*Runner) Run() {}
func (*Runner) Other() {}
`)
	fixture.write("caller/caller.go", callerSource)
	protected := fixtureSymbol(
		AuthorityLayerRaw,
		fixture.packagePath("protected"),
		"Cap",
		ProtectedPackageFunction,
		"",
	)
	return fixture, protected, fixture.packagePath("caller")
}

// namedCallerMethodSymbols returns the two protected symbols needed for
// a method-approval test (the function and the method).
func namedCallerMethodSymbols(fixture *canonicalFixture) ([]ProtectedSymbol, ProtectedSymbol, ProtectedSymbol) {
	protected := fixtureSymbol(AuthorityLayerRaw, fixture.packagePath("protected"), "Cap", ProtectedPackageFunction, "")
	method := fixtureSymbol(AuthorityLayerAdapter, fixture.packagePath("protected"), "Run", ProtectedMethod, "Runner")
	return []ProtectedSymbol{protected, method}, protected, method
}

// hasFindingKind reports whether any finding has the given kind.
func hasFindingKind(findings []checks.Finding, kind string) bool {
	for _, finding := range findings {
		if finding.Kind == kind {
			return true
		}
	}
	return false
}

// Phase 5 regression — approved package function direct call:
//
//	accepted
//	no anonymous-caller finding
//	cardinality satisfied
func TestNamedCallerPackageFunctionDirectCall(t *testing.T) {
	fixture, protected, callerPkg := namedCallerFixture(t, `package caller
import p "example.test/policy/protected"
func Allowed() { p.Cap() }
`)
	approval := fixtureApproval(callerPkg, "Allowed", "", CallerKindPackageFunction, protected, refDirectCall)
	result := fixture.run([]ProtectedSymbol{protected}, []ApprovedCaller{approval})

	for _, kind := range []string{
		"authority_policy_anonymous_caller",
		"dupcode_bypass",
		"authority_policy_stale_approval",
		"authority_policy_edge_cardinality_mismatch",
		"dupcode_protected_function_value",
	} {
		rejectFindingKind(t, result.Findings, kind)
	}
	if result.Stats.ValidatedApprovals != 1 {
		t.Errorf("validated approvals = %d, want 1", result.Stats.ValidatedApprovals)
	}
	if len(result.ObservedEdges) != 1 {
		t.Fatalf("observed edges = %d, want 1", len(result.ObservedEdges))
	}
	if result.ObservedEdges[0].Caller.Function != "Allowed" {
		t.Errorf("caller function = %q, want Allowed", result.ObservedEdges[0].Caller.Function)
	}
	if result.ObservedEdges[0].Caller.Kind != CallerKindPackageFunction {
		t.Errorf("caller kind = %q, want package_function", result.ObservedEdges[0].Caller.Kind)
	}
}

// Phase 5 regression — approved method direct call:
//
//	accepted
//	no anonymous-caller finding
//	cardinality satisfied
func TestNamedCallerMethodDirectCall(t *testing.T) {
	fixture, _, callerPkg := namedCallerFixture(t, `package caller
import p "example.test/policy/protected"
type Runner struct{}
func (r *Runner) Allowed(r2 *p.Runner) { r2.Run() }
`)
	symbols, _, method := namedCallerMethodSymbols(fixture)
	approval := fixtureApproval(callerPkg, "Allowed", "Runner", CallerKindMethod, method, refDirectCall)
	result := fixture.run(symbols, []ApprovedCaller{approval})

	for _, kind := range []string{
		"authority_policy_anonymous_caller",
		"dupcode_bypass",
		"authority_policy_stale_approval",
		"authority_policy_edge_cardinality_mismatch",
		"dupcode_protected_function_value",
		"dupcode_adapter_bypass",
	} {
		rejectFindingKind(t, result.Findings, kind)
	}
	if result.Stats.ValidatedApprovals != 1 {
		t.Errorf("validated approvals = %d, want 1", result.Stats.ValidatedApprovals)
	}
}

// Phase 5 regression — approved package function as function value:
//
//	accepted when explicitly configured
func TestNamedCallerPackageFunctionValue(t *testing.T) {
	fixture, protected, callerPkg := namedCallerFixture(t, `package caller
import p "example.test/policy/protected"
func Allowed() { _ = p.Other }
`)
	symbols := []ProtectedSymbol{
		protected,
		fixtureSymbol(AuthorityLayerRaw, fixture.packagePath("protected"), "Other", ProtectedPackageFunction, ""),
	}
	other := symbols[1]
	approval := fixtureApproval(callerPkg, "Allowed", "", CallerKindPackageFunction, other, refFunctionValue)
	result := fixture.run(symbols, []ApprovedCaller{approval})

	for _, kind := range []string{
		"authority_policy_anonymous_caller",
		"dupcode_bypass",
		"dupcode_protected_function_value",
		"authority_policy_stale_approval",
		"authority_policy_edge_cardinality_mismatch",
	} {
		rejectFindingKind(t, result.Findings, kind)
	}
	if result.Stats.ValidatedApprovals != 1 {
		t.Errorf("validated approvals = %d, want 1", result.Stats.ValidatedApprovals)
	}
}

// Phase 5 regression — approved method (function value) when explicitly
// configured:
//
//	accepted
func TestNamedCallerMethodValue(t *testing.T) {
	fixture, _, callerPkg := namedCallerFixture(t, `package caller
import p "example.test/policy/protected"
type Runner struct{}
func (r *Runner) Allowed(r2 *p.Runner) { _ = r2.Run }
`)
	symbols, _, method := namedCallerMethodSymbols(fixture)
	approval := fixtureApproval(callerPkg, "Allowed", "Runner", CallerKindMethod, method, refMethodValue)
	result := fixture.run(symbols, []ApprovedCaller{approval})

	for _, kind := range []string{
		"authority_policy_anonymous_caller",
		"dupcode_bypass",
		"dupcode_protected_function_value",
		"authority_policy_stale_approval",
		"authority_policy_edge_cardinality_mismatch",
	} {
		rejectFindingKind(t, result.Findings, kind)
	}
	if result.Stats.ValidatedApprovals != 1 {
		t.Errorf("validated approvals = %d, want 1", result.Stats.ValidatedApprovals)
	}
}

// Phase 5 regression — unapproved named package function:
//
//	rejected through existing policy (no anonymous-caller finding)
func TestNamedCallerUnapprovedPackageFunction(t *testing.T) {
	fixture, protected, callerPkg := namedCallerFixture(t, `package caller
import p "example.test/policy/protected"
func Unapproved() { p.Cap() }
func Approved()  { _ = p.Cap }
`)
	other := fixtureSymbol(AuthorityLayerRaw, fixture.packagePath("protected"), "Other", ProtectedPackageFunction, "")
	// Approve the function-value reference inside Approved, but leave
	// Unapproved's direct call unapproved. The unapproved direct call
	// must produce a dupcode_bypass.
	approval := fixtureApproval(callerPkg, "Approved", "", CallerKindPackageFunction, other, refFunctionValue)
	result := fixture.run([]ProtectedSymbol{protected, other}, []ApprovedCaller{approval})
	rejectFindingKind(t, result.Findings, "authority_policy_anonymous_caller")
	if !hasFindingKind(result.Findings, "dupcode_bypass") {
		t.Errorf("expected dupcode_bypass for unapproved call: %v", findingKinds(result.Findings))
	}
}

// Phase 5 regression — unapproved named method:
//
//	rejected through existing policy
func TestNamedCallerUnapprovedMethod(t *testing.T) {
	fixture, _, _ := namedCallerFixture(t, `package caller
import p "example.test/policy/protected"
type Runner struct{}
func (r *Runner) Unapproved(r2 *p.Runner) { r2.Run() }
`)
	symbols, _, _ := namedCallerMethodSymbols(fixture)
	result := fixture.run(symbols, nil)
	rejectFindingKind(t, result.Findings, "authority_policy_anonymous_caller")
	if !hasFindingKind(result.Findings, "dupcode_bypass") && !hasFindingKind(result.Findings, "dupcode_adapter_bypass") {
		t.Errorf("expected dupcode_bypass or dupcode_adapter_bypass for unapproved method: %v", findingKinds(result.Findings))
	}
	methodEdges := 0
	for _, edge := range result.ObservedEdges {
		if edge.Caller.Kind == CallerKindMethod &&
			edge.Caller.Function == "Unapproved" {
			methodEdges++
		}
	}
	if methodEdges != 1 {
		t.Errorf("Unapproved method edges = %d, want 1", methodEdges)
	}
}

// Phase 6 — function-literal approval is rejected by the strict schema.
func TestSchemaRejectsFunctionLiteralApproval(t *testing.T) {
	fixture, protected, callerPkg := namedCallerFixture(t, `package caller
import p "example.test/policy/protected"
func Allowed() { p.Cap() }
`)
	bogus := fixtureApproval(callerPkg, "<func-literal:1:1>", "", CallerKindFunctionLiteral, protected, refDirectCall)
	issues := validateApprovalSchema(bogus)
	if len(issues) == 0 {
		t.Fatalf("function-literal approval must fail schema; got none")
	}
	rejected := false
	for _, issue := range issues {
		if issue.Kind == "authority_policy_approval_caller_kind_invalid" {
			rejected = true
		}
	}
	if !rejected {
		t.Errorf("expected caller_kind_invalid issue, got %+v", issues)
	}
	result := fixture.run([]ProtectedSymbol{protected}, []ApprovedCaller{bogus})
	if !hasFindingKind(result.Findings, "authority_policy_approval_caller_kind_invalid") {
		t.Errorf("expected schema finding for function-literal approval: %v", findingKinds(result.Findings))
	}
}

// Phase 6 — source-coordinate approval is rejected by the strict schema.
func TestSchemaRejectsSourceCoordinateApproval(t *testing.T) {
	fixture, protected, callerPkg := namedCallerFixture(t, `package caller
import p "example.test/policy/protected"
func Allowed() { p.Cap() }
`)
	bogus := fixtureApproval(callerPkg, "Allowed@10:15", "", CallerKindPackageFunction, protected, refDirectCall)
	issues := validateApprovalSchema(bogus)
	if len(issues) == 0 {
		t.Fatalf("source-coordinate approval must fail schema; got none")
	}
	rejected := false
	for _, issue := range issues {
		if issue.Kind == "authority_policy_approval_identity_invalid" {
			rejected = true
		}
	}
	if !rejected {
		t.Errorf("expected identity_invalid issue, got %+v", issues)
	}
	result := fixture.run([]ProtectedSymbol{protected}, []ApprovedCaller{bogus})
	if !hasFindingKind(result.Findings, "authority_policy_approval_identity_invalid") {
		t.Errorf("expected schema finding for source-coordinate approval: %v", findingKinds(result.Findings))
	}
}

// Phase 6 — wildcard approval is rejected by the strict schema.
func TestSchemaRejectsWildcardApproval(t *testing.T) {
	_, protected, callerPkg := namedCallerFixture(t, `package caller
import p "example.test/policy/protected"
func Allowed() { p.Cap() }
`)
	bogus := fixtureApproval(callerPkg, "Allowed*", "", CallerKindPackageFunction, protected, refDirectCall)
	issues := validateApprovalSchema(bogus)
	if len(issues) == 0 {
		t.Fatalf("wildcard approval must fail schema; got none")
	}
	rejected := false
	for _, issue := range issues {
		if issue.Kind == "authority_policy_approval_identity_invalid" {
			rejected = true
		}
	}
	if !rejected {
		t.Errorf("expected identity_invalid issue, got %+v", issues)
	}
}

// Phase 6 — receiver used as synthetic closure identity is rejected by
// the strict schema. A package-function approval must NOT carry a
// receiver.
func TestSchemaRejectsReceiverOnPackageFunctionApproval(t *testing.T) {
	_, protected, callerPkg := namedCallerFixture(t, `package caller
import p "example.test/policy/protected"
func Allowed() { p.Cap() }
`)
	bogus := fixtureApproval(callerPkg, "Allowed", "func-literal", CallerKindPackageFunction, protected, refDirectCall)
	issues := validateApprovalSchema(bogus)
	if len(issues) == 0 {
		t.Fatalf("receiver on package function approval must fail schema; got none")
	}
	rejected := false
	for _, issue := range issues {
		if issue.Kind == "authority_policy_approval_receiver_invalid" {
			rejected = true
		}
	}
	if !rejected {
		t.Errorf("expected receiver_invalid issue, got %+v", issues)
	}
}

// Production preservation — direct call inside an approved package
// function alongside a function literal that uses the protected
// symbol. The literal is rejected; the direct call is matched. The
// outer approval is matched exactly once.
func TestNamedCallerDirectCallAndLiteralInSameFunction(t *testing.T) {
	fixture, protected, callerPkg := namedCallerFixture(t, `package caller
import p "example.test/policy/protected"
func Allowed() {
	func() {
		p.Cap()
	}()
	p.Cap()
}
`)
	approval := fixtureApproval(callerPkg, "Allowed", "", CallerKindPackageFunction, protected, refDirectCall)
	result := fixture.run([]ProtectedSymbol{protected}, []ApprovedCaller{approval})

	if len(result.ObservedEdges) != 2 {
		t.Fatalf("observed edges = %d, want 2", len(result.ObservedEdges))
	}
	anonymous := 0
	for _, finding := range result.Findings {
		if finding.Kind == "authority_policy_anonymous_caller" {
			anonymous++
		}
	}
	if anonymous != 1 {
		t.Errorf("anonymous findings = %d, want 1", anonymous)
	}
	for _, kind := range []string{
		"authority_policy_stale_approval",
		"authority_policy_edge_cardinality_mismatch",
		"dupcode_bypass",
	} {
		rejectFindingKind(t, result.Findings, kind)
	}
	if result.Stats.ValidatedApprovals != 1 {
		t.Errorf("validated approvals = %d, want 1", result.Stats.ValidatedApprovals)
	}
}

// Production preservation — the function literal identifier is stable
// across runs and uses the position-based pattern.
func TestAnonymousCallerIdentifierRemainsStable(t *testing.T) {
	fixture, protected, callerPkg := namedCallerFixture(t, `package caller
import p "example.test/policy/protected"
func Allowed() { func() { p.Cap() }() }
`)
	approval := fixtureApproval(callerPkg, "Allowed", "", CallerKindPackageFunction, protected, refDirectCall)
	result := fixture.run([]ProtectedSymbol{protected}, []ApprovedCaller{approval})

	sawLiteral := false
	for _, edge := range result.ObservedEdges {
		if edge.Caller.Kind == CallerKindFunctionLiteral {
			sawLiteral = true
			if !strings.HasPrefix(edge.Caller.Function, "<func-literal:") {
				t.Errorf("function literal identifier = %q, want <func-literal:..> prefix", edge.Caller.Function)
			}
		}
	}
	if !sawLiteral {
		t.Errorf("no function literal observed edge")
	}
}
