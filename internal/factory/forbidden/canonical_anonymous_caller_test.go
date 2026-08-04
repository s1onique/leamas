// SPDX-License-Identifier: Apache-2.0

package forbidden

import (
	"strings"
	"testing"
)

// anonymousCallerFixture builds a single-policy fixture with a raw
// protected symbol and a caller package whose source is supplied. The
// caller package path is returned so tests can construct approvals and
// assertions.
func anonymousCallerFixture(t *testing.T, callerSource string) (*canonicalFixture, ProtectedSymbol, string) {
	t.Helper()
	fixture := newCanonicalFixture(t)
	fixture.write("protected/protected.go", `package protected
func Cap() {}
type Runner struct{}
func (*Runner) Run() {}
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

// anonymousCallerHarness executes one anonymous-caller fixture and
// returns the canonical result. The supplied approval is configured
// against the outer named function or method so any leakage of the
// outer approval onto the anonymous edge would be observable.
func anonymousCallerHarness(
	t *testing.T,
	callerSource string,
	approvalFunction string,
	approvalReceiver string,
	approvalKind string,
) canonicalResult {
	t.Helper()
	fixture, protected, callerPkg := anonymousCallerFixture(t, callerSource)
	approval := fixtureApproval(callerPkg, approvalFunction, approvalReceiver, approvalKind, protected, refDirectCall)
	return fixture.run([]ProtectedSymbol{protected}, []ApprovedCaller{approval})
}

// requireAnonymousCallerVerified enforces the invariant contract for
// every adversarial fixture in Phase 4.
//
// Anonymous-caller contracts:
//
//	exactly one anonymous-caller finding
//	no ordinary dupcode_bypass (no duplicate cascade for the same edge)
//	no cardinality mismatch on the outer approval (the anonymous edge
//	  contributes zero matches to the outer approval)
//
// The outer approval may legitimately be reported as stale when the
// only source edge is the anonymous one. That is the truthful outcome
// for the fixture-only scenario; production never sees anonymous
// edges because the protectedverifier adapter is the only caller.
func requireAnonymousCallerVerified(t *testing.T, result canonicalResult) {
	t.Helper()

	anonymousCount := 0
	bypassCount := 0
	functionLiteralEdges := 0
	for _, finding := range result.Findings {
		switch finding.Kind {
		case "authority_policy_anonymous_caller":
			anonymousCount++
			message := finding.Message
			if !strings.Contains(message, "anonymous caller (function literal)") {
				t.Errorf("anonymous finding missing scope classification: %q", message)
			}
			if !strings.Contains(message, "outer approval not inherited") {
				t.Errorf("anonymous finding missing outer-not-inherited phrase: %q", message)
			}
			if !strings.Contains(message, "->") {
				t.Errorf("anonymous finding missing callee identity: %q", message)
			}
			if !strings.Contains(message, "[DIRECT_CALL]") && !strings.Contains(message, "[FUNCTION_VALUE]") {
				t.Errorf("anonymous finding missing reference class: %q", message)
			}
		case "dupcode_bypass":
			bypassCount++
		}
	}
	for _, edge := range result.ObservedEdges {
		if edge.Caller.Kind == CallerKindFunctionLiteral {
			functionLiteralEdges++
		}
	}
	if anonymousCount != functionLiteralEdges {
		t.Errorf("anonymous findings (%d) != function literal edges (%d)", anonymousCount, functionLiteralEdges)
	}
	if anonymousCount < 1 {
		t.Errorf("anonymous caller finding count = %d, want >= 1", anonymousCount)
	}
	if bypassCount != 0 {
		t.Errorf("dupcode_bypass count = %d, want 0 (no ordinary bypass for anonymous edge)", bypassCount)
	}
	for _, kind := range []string{
		"authority_policy_edge_cardinality_mismatch",
	} {
		rejectFindingKind(t, result.Findings, kind)
	}
}

// Phase 4.1 — Immediately invoked closure.
func TestAnonymousCallerImmediatelyInvokedClosure(t *testing.T) {
	result := anonymousCallerHarness(
		t,
		`package caller
import p "example.test/policy/protected"
func Approved() {
	func() {
		p.Cap()
	}()
}
`,
		"Approved",
		"",
		CallerKindPackageFunction,
	)
	requireAnonymousCallerVerified(t, result)

	// The outer caller is captured for diagnostics.
	functionLiteralEdges := 0
	for _, edge := range result.ObservedEdges {
		if edge.Caller.Kind == CallerKindFunctionLiteral &&
			edge.hasOuterCaller &&
			edge.outerCaller.Function == "Approved" {
			functionLiteralEdges++
		}
	}
	if functionLiteralEdges != 1 {
		t.Errorf("function literal use with outer Approved = %d, want 1", functionLiteralEdges)
	}
}

// Phase 4.2 — Escaping closure.
func TestAnonymousCallerEscapingClosure(t *testing.T) {
	result := anonymousCallerHarness(
		t,
		`package caller
import p "example.test/policy/protected"
func publish(_ func()) {}
func Approved() {
	escaped := func() {
		p.Cap()
	}
	publish(escaped)
}
`,
		"Approved",
		"",
		CallerKindPackageFunction,
	)
	requireAnonymousCallerVerified(t, result)
}

// Phase 4.3 — Callback argument.
func TestAnonymousCallerCallbackArgument(t *testing.T) {
	result := anonymousCallerHarness(
		t,
		`package caller
import p "example.test/policy/protected"
func register(_ func()) {}
func Approved() {
	register(func() {
		p.Cap()
	})
}
`,
		"Approved",
		"",
		CallerKindPackageFunction,
	)
	requireAnonymousCallerVerified(t, result)
}

// Phase 4.4 — Returned callback.
func TestAnonymousCallerReturnedCallback(t *testing.T) {
	result := anonymousCallerHarness(
		t,
		`package caller
import p "example.test/policy/protected"
func Approved() func() {
	return func() {
		p.Cap()
	}
}
`,
		"Approved",
		"",
		CallerKindPackageFunction,
	)
	requireAnonymousCallerVerified(t, result)
}

// Phase 4.5 — Nested anonymous closures.
func TestAnonymousCallerNestedAnonymousClosures(t *testing.T) {
	result := anonymousCallerHarness(
		t,
		`package caller
import p "example.test/policy/protected"
func Approved() {
	func() {
		func() {
			p.Cap()
		}()
	}()
}
`,
		"Approved",
		"",
		CallerKindPackageFunction,
	)
	requireAnonymousCallerVerified(t, result)

	// Exactly one anonymous edge, with the nearest enclosing scope
	// selected (the inner function literal). The outer caller capture
	// identifies the nearest NAMED declaration, which is Approved.
	for _, edge := range result.ObservedEdges {
		if edge.Caller.Kind == CallerKindFunctionLiteral {
			if !edge.hasOuterCaller {
				t.Errorf("nested anonymous edge missing outer caller: %#v", edge)
			}
			if edge.outerCaller.Kind != CallerKindPackageFunction ||
				edge.outerCaller.Function != "Approved" {
				t.Errorf("nested anonymous outer caller = %v, want Approved", edge.outerCaller)
			}
		}
	}
}

// Phase 4.6 — Closure inside an approved method.
func TestAnonymousCallerInApprovedMethod(t *testing.T) {
	fixture, protected, callerPkg := anonymousCallerFixture(t, `package caller
import p "example.test/policy/protected"
type Runner struct{}
func (r *Runner) ApprovedMethod() {
	_ = r
	func() {
		p.Cap()
	}()
}
`)
	approval := fixtureApproval(callerPkg, "ApprovedMethod", "Runner", CallerKindMethod, protected, refDirectCall)
	result := fixture.run([]ProtectedSymbol{protected}, []ApprovedCaller{approval})
	requireAnonymousCallerVerified(t, result)

	// The outer caller capture identifies the nearest named method.
	for _, edge := range result.ObservedEdges {
		if edge.Caller.Kind == CallerKindFunctionLiteral {
			if !edge.hasOuterCaller ||
				edge.outerCaller.Kind != CallerKindMethod ||
				edge.outerCaller.Function != "ApprovedMethod" {
				t.Errorf("method-edge outer caller = %v, want ApprovedMethod", edge.outerCaller)
			}
		}
	}
}

// Phase 4.7 — Function value capture inside a closure.
func TestAnonymousCallerFunctionValueInsideClosure(t *testing.T) {
	result := anonymousCallerHarness(
		t,
		`package caller
import p "example.test/policy/protected"
func Approved() {
	callback := func() {
		f := p.Cap
		_ = f
	}
	_ = callback
}
`,
		"Approved",
		"",
		CallerKindPackageFunction,
	)
	requireAnonymousCallerVerified(t, result)

	// The reference class for p.Cap captured as a function value is
	// FUNCTION_VALUE, not DIRECT_CALL. The anonymous-caller finding
	// must still be emitted with the FUNCTION_VALUE class.
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

// TestAnonymousCallerMultipleLitsAreIndependent asserts that two
// independent function literals in the same enclosing function each
// produce one anonymous-caller finding. The outer approval is neither
// matched nor poisoned; it is reported as stale because no legitimate
// source edge exists for it (the only edges are anonymous).
func TestAnonymousCallerMultipleLitsAreIndependent(t *testing.T) {
	result := anonymousCallerHarness(
		t,
		`package caller
import p "example.test/policy/protected"
func Approved() {
	func() {
		p.Cap()
	}()
	func() {
		p.Cap()
	}()
}
`,
		"Approved",
		"",
		CallerKindPackageFunction,
	)
	requireAnonymousCallerVerified(t, result)

	anonymousCount := 0
	for _, finding := range result.Findings {
		if finding.Kind == "authority_policy_anonymous_caller" {
			anonymousCount++
		}
	}
	if anonymousCount != 2 {
		t.Errorf("anonymous caller findings = %d, want 2", anonymousCount)
	}
	for _, kind := range []string{
		"dupcode_bypass",
		"authority_policy_edge_cardinality_mismatch",
	} {
		rejectFindingKind(t, result.Findings, kind)
	}
	// Stale_approval is the truthful outcome for an outer approval
	// whose only source edge is anonymous. The presence of stale is
	// acceptable; the cardinality-mismatch cascade is the invariant.
	requireFindingKind(t, result.Findings, "authority_policy_stale_approval")
}

// TestAnonymousCallerOuterApprovalStaleExactCount locks the cardinality
// invariant: the outer approval's match count is exactly zero when the
// only source edge is anonymous. The stale_approval finding is the
// truthful outcome.
func TestAnonymousCallerOuterApprovalStaleExactCount(t *testing.T) {
	result := anonymousCallerHarness(
		t,
		`package caller
import p "example.test/policy/protected"
func Approved() {
	func() {
		p.Cap()
	}()
}
`,
		"Approved",
		"",
		CallerKindPackageFunction,
	)
	requireAnonymousCallerVerified(t, result)
	stale := 0
	for _, finding := range result.Findings {
		if finding.Kind == "authority_policy_stale_approval" {
			stale++
		}
	}
	if stale != 1 {
		t.Errorf("stale_approval = %d, want 1 (outer approval has no legitimate edge)", stale)
	}
}
