package closure

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
	"testing"
)

// plan_contract_bounded_parser_authority_test.go enforces the
// single-caller invariant for parseClosurePlanDocument and the
// no-duplicate-size-check invariant for every public byte-entry
// API. It walks every Go source file in the package via go/ast
// and asserts the documented caller topology. The companion
// plan_contract_bounded_parser_authority_pkg_test.go holds the
// shared AST inspection helpers the tests rely on.

// TestBoundedParserAuthoritySingleCaller proves that the only
// production caller of parseClosurePlanDocument is
// parseBoundedClosurePlanDocument (in
// plan_contract_validation_bounded.go). The function is otherwise
// private and tests must reach it through the bounded entry
// point so the size bound is enforced.
func TestBoundedParserAuthoritySingleCaller(t *testing.T) {
	calls := collectCallsInPackage(t, "parseClosurePlanDocument")
	if len(calls) != 1 {
		var names []string
		for _, c := range calls {
			names = append(names, c.function)
		}
		t.Fatalf("parseClosurePlanDocument callers = %v, want exactly 1", names)
	}
	if calls[0].function != "parseBoundedClosurePlanDocument" {
		t.Fatalf("parseClosurePlanDocument caller = %s, want parseBoundedClosurePlanDocument", calls[0].function)
	}
	if !strings.HasSuffix(calls[0].file, "plan_contract_validation_bounded.go") {
		t.Fatalf("parseClosurePlanDocument caller file = %s, want plan_contract_validation_bounded.go", calls[0].file)
	}
}

// TestBoundedParserAuthorityNoPublicDuplicateSizeCheck proves that
// no exported byte-entry API calls parseClosurePlanDocument
// directly (which would bypass MaxPlanBytes). Public byte-entry
// APIs must reach the parser only through
// parseBoundedClosurePlanDocument so the size bound is enforced.
func TestBoundedParserAuthorityNoPublicDuplicateSizeCheck(t *testing.T) {
	for _, fn := range collectPublicByteEntryFunctions(t) {
		for _, call := range fn.calls {
			if call == "parseClosurePlanDocument" {
				t.Fatalf("public byte-entry %s in %s must not call parseClosurePlanDocument directly; route through parseBoundedClosurePlanDocument",
					fn.name, fn.file)
			}
		}
	}
}

// TestBoundedParserAuthorityNoDuplicateSizeCheck walks every
// production byte-entry and internal pipeline entry point and
// asserts the original *ast.FuncDecl.Body contains no binary
// comparison equivalent to `len(data) > MaxPlanBytes`. The
// detector fails closed on a missing declaration, duplicate
// declarations, a nil body, or a parser error. The only place
// that size check is allowed is the bounded parser itself.
func TestBoundedParserAuthorityNoDuplicateSizeCheck(t *testing.T) {
	entries := sizeBoundCandidateEntries()
	roots, err := loadClosureSources(t)
	if err != nil {
		t.Fatalf("load sources: %v", err)
	}
	for _, entry := range entries {
		if entry == "parseBoundedClosurePlanDocument" {
			continue
		}
		fn, err := loadUniqueFunctionDecl(roots, entry)
		if err != nil {
			t.Fatalf("function %s: %v", entry, err)
		}
		if containsMaxPlanBytesComparison(fn.Body) {
			t.Fatalf("function %s contains a duplicate MaxPlanBytes comparison; route through parseBoundedClosurePlanDocument instead",
				entry)
		}
	}
}

// TestBoundedParserAuthorityAdversarialSizeCheckFixtures exercises
// four adversarial comparison orientations to prove the
// detector rejects each form independently. The fixtures are
// constructed in-memory; they never enter the production source
// tree.
func TestBoundedParserAuthorityAdversarialSizeCheckFixtures(t *testing.T) {
	cases := []struct {
		name string
		src  string
	}{
		{
			name: "len-data-greater",
			src: `package fixture
func F(data []byte) {
	if len(data) > MaxPlanBytes {
		return
	}
}`,
		},
		{
			name: "len-data-greater-equal",
			src: `package fixture
func F(data []byte) {
	if len(data) >= MaxPlanBytes {
		return
	}
}`,
		},
		{
			name: "max-less-len",
			src: `package fixture
func F(data []byte) {
	if MaxPlanBytes < len(data) {
		return
	}
}`,
		},
		{
			name: "max-less-equal-len",
			src: `package fixture
func F(data []byte) {
	if MaxPlanBytes <= len(data) {
		return
	}
}`,
		},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			body := parseFixtureBody(t, tc.src)
			if !containsMaxPlanBytesComparison(body) {
				t.Fatalf("detector must reject %s orientation", tc.name)
			}
		})
	}
}

// parseFixtureBody parses a Go source fragment into an AST body
// for adversarial testing. The fragment must declare exactly one
// function named `F`; the returned body is `F`'s body.
func parseFixtureBody(t *testing.T, src string) *ast.BlockStmt {
	t.Helper()
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "fixture.go", src, 0)
	if err != nil {
		t.Fatalf("parse fixture: %v", err)
	}
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok {
			continue
		}
		if fn.Name.Name != "F" {
			continue
		}
		if fn.Body == nil {
			t.Fatalf("fixture %q has no body", src)
		}
		return fn.Body
	}
	t.Fatalf("fixture %q declares no function F", src)
	return nil
}

// sizeBoundCandidateEntries returns every entry point the ACT
// pins as a candidate for size-bound duplication. The list is
// authoritative; add new candidates here only after the ACT
// grows them.
func sizeBoundCandidateEntries() []string {
	return []string{
		"DecodePlan",
		"LoadPlanFromBytes",
		"ValidatePlanStructural",
		"ValidatePlanComposed",
		"ValidatePlanStructuralAndSemantic",
		"validatePlanComposedWithObserver",
		"validatePlanComposedWithObserverAndDeps",
		"validatePlanStructuralAndSemanticWith",
		"validatePlanStructuralWithObserver",
	}
}

// loadUniqueFunctionDecl returns the unique function declaration
// with the given name across the supplied source roots. It fails
// closed when zero declarations match, when more than one
// declaration matches, or when the matched declaration has a
// nil body. The fail-closed contract guarantees the test cannot
// silently pass on a missing or duplicated entry point.
func loadUniqueFunctionDecl(roots []*ast.File, name string) (*ast.FuncDecl, error) {
	var matches []*ast.FuncDecl
	for _, file := range roots {
		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok {
				continue
			}
			if fn.Name.Name != name {
				continue
			}
			matches = append(matches, fn)
		}
	}
	if len(matches) == 0 {
		return nil, fmt.Errorf("function %s not found", name)
	}
	if len(matches) > 1 {
		return nil, fmt.Errorf("function %s has %d declarations, want exactly 1", name, len(matches))
	}
	fn := matches[0]
	if fn.Body == nil {
		return nil, fmt.Errorf("function %s has no body", name)
	}
	return fn, nil
}

// strings is imported indirectly via collectCallsInPackage which
// uses strings.HasSuffix. Importing strings here keeps the test
// file self-contained even when its helpers move.
var _ = strings.HasSuffix
