// SPDX-License-Identifier: Apache-2.0

package forbidden

import (
	"slices"
	"strings"
	"testing"
)

func TestCanonicalProtectedReferenceMatrixUsesTypeIdentity(t *testing.T) {
	fixture := newCanonicalFixture(t)
	protectedPkg := fixture.packagePath("protected")
	fixture.write("protected/protected.go", `package protected

func Cap() {}
func Other() {}
var Capability = func() {}

type Runner struct{}
func (*Runner) Run() {}

func Internal() { Cap() }
`)
	fixture.write("caller/caller.go", `package caller

import p "example.test/policy/protected"

func Alias() { p.Cap() }
func FunctionValue() { _ = p.Other }
func MethodDirect(r *p.Runner) { r.Run() }
func MethodValue(r *p.Runner) { _ = r.Run }
func MethodExpression() { _ = (*p.Runner).Run }
func Outer() { func() { p.Cap() }() }

var Captured = p.Other
var CapturedVariable = p.Capability

func init() { p.Cap() }

type shadow struct{}
func (shadow) Cap() {}
func ShadowedPackageName() { p := shadow{}; p.Cap() }

var _ = "p.Cap() in a string"
// p.Cap() in a comment
`)
	fixture.write("dotcaller/dot.go", `package dotcaller
import . "example.test/policy/protected"
func Dot() { Cap() }
`)

	symbols := []ProtectedSymbol{
		fixtureSymbol(AuthorityLayerRaw, protectedPkg, "Cap", ProtectedPackageFunction, ""),
		fixtureSymbol(AuthorityLayerRaw, protectedPkg, "Other", ProtectedPackageFunction, ""),
		fixtureSymbol(AuthorityLayerAdapter, protectedPkg, "Run", ProtectedMethod, "Runner"),
		fixtureSymbol(AuthorityLayerAdapter, protectedPkg, "Capability", ProtectedPackageVariable, ""),
	}
	result := fixture.run(symbols, nil)
	if len(result.ObservedEdges) != 11 {
		t.Fatalf("observed edges = %d, want 11: %#v", len(result.ObservedEdges), result.ObservedEdges)
	}

	wantClasses := []ReferenceClass{
		refDirectCall,
		refDirectCall,
		refDirectCall,
		refDirectCall,
		refDirectCall,
		refDotImport,
		refFunctionValue,
		refFunctionValue,
		refMethodExpression,
		refMethodValue,
		refPackageVariable,
	}
	slices.Sort(wantClasses)
	if got := edgeClasses(result.ObservedEdges); !slices.Equal(got, wantClasses) {
		t.Fatalf("reference classes = %v, want %v", got, wantClasses)
	}

	// The function literal case is now its own caller scope. It must
	// NOT be attributed to the outer named function Outer. The
	// CallerKind is function_literal and the outer caller is captured
	// on the edge for diagnostics only.
	functionLiteralCount := 0
	for _, edge := range result.ObservedEdges {
		if edge.Caller.Kind == CallerKindFunctionLiteral {
			functionLiteralCount++
			if !edge.hasOuterCaller || edge.outerCaller.Function != "Outer" {
				t.Fatalf("function literal edge missing outer Outer: %#v", edge)
			}
			if !strings.HasPrefix(edge.Caller.Function, "<func-literal:") {
				t.Fatalf("function literal identifier not stable: %#v", edge.Caller)
			}
		}
		if edge.Caller.Function == "Outer" && edge.Caller.Kind == CallerKindPackageFunction {
			t.Fatalf("outer Outer must not be attributed to function literal use: %#v", edge)
		}
	}
	if functionLiteralCount != 1 {
		t.Fatalf("function literal callers = %d, want 1", functionLiteralCount)
	}

	// The function literal case below Outer's body now produces an
	// authority_policy_anonymous_caller finding instead of attributed to
	// Outer.
	requireFindingKind(t, result.Findings, "authority_policy_anonymous_caller")
	requireFindingKind(t, result.Findings, "dupcode_dot_import")

	assertReferenceCaller(t, result.ObservedEdges, "<var-init:Captured>", CallerKindVariableInitializer, refFunctionValue)
	assertReferenceCaller(t, result.ObservedEdges, "<init>", CallerKindPackageInit, refDirectCall)
	assertReferenceCaller(t, result.ObservedEdges, "Internal", CallerKindPackageFunction, refDirectCall)
	assertReferenceCaller(t, result.ObservedEdges, "Dot", CallerKindPackageFunction, refDotImport)

	for _, edge := range result.ObservedEdges {
		if edge.Caller.Function == "ShadowedPackageName" {
			t.Fatalf("shadowed identifier produced protected edge: %#v", edge)
		}
	}
}

func assertReferenceCaller(
	t *testing.T,
	edges []ObservedEdge,
	function string,
	kind string,
	class ReferenceClass,
) {
	t.Helper()
	for _, edge := range edges {
		if edge.Caller.Function == function && edge.Caller.Kind == kind && edge.ReferenceClass == class {
			return
		}
	}
	t.Fatalf("missing edge caller function=%q kind=%q class=%q in %#v", function, kind, class, edges)
}

func TestCanonicalDirectCallProducesNoDuplicateFunctionValueEdge(t *testing.T) {
	fixture, protected, _ := approvalFixture(t, `package caller
import p "example.test/policy/protected"
func One() { p.Cap() }
`)
	result := fixture.run([]ProtectedSymbol{protected}, nil)
	if len(result.ObservedEdges) != 1 {
		t.Fatalf("direct call edges = %#v, want exactly one", result.ObservedEdges)
	}
	if result.ObservedEdges[0].ReferenceClass != refDirectCall {
		t.Fatalf("direct call class = %q", result.ObservedEdges[0].ReferenceClass)
	}
}
