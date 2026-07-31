package closure

import (
	"bufio"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// plan_contract_source_structure_test.go enforces the package's
// own LLM-friendly invariants at test-time and provides a
// fail-closed naming audit for the typed-stage seam. The audit
// walks every test function in the package and reports any
// function whose name carries the typed-stage marker but whose
// body does not provide AST evidence of the typed-decoder seam.

// maxLineLength mirrors the LLM-friendly gate threshold (240
// chars per line). Any test file in the closure package whose
// lines exceed this threshold trips the test.
const maxLineLength = 240

// typedStageNamingMarkers are the substrings whose presence in
// a test function's name signals a claim that the test exercises
// the typed-stage seam. The strings are stored at package
// scope so they do not appear inside any test function body,
// where they would self-trip the audit. The marker set covers
// the camelCase, snake_case, and space-separated forms.
var typedStageNamingMarkers = []string{"TypedFailure", "typed_failure", "typed failure"}

// TestSourceStructureNoLongLines walks every Go file in the
// closure package directory and asserts that no line exceeds
// the LLM-friendly line-length threshold. This proves the
// "split long inline fixtures into small focused files"
// requirement is mechanically satisfied.
func TestSourceStructureNoLongLines(t *testing.T) {
	files, err := filepath.Glob("./*.go")
	if err != nil {
		t.Fatalf("glob: %v", err)
	}
	if len(files) == 0 {
		t.Fatalf("no Go files found in closure package directory")
	}
	for _, path := range files {
		scanFile(t, path)
	}
}

// TestSourceStructureNoAllowlistInLLMFriendly asserts the
// LLM-friendly verifier has not grown an allowlist entry for
// any closure-package source file. The existing
// isCanonicalClosurePlan exemption is restricted to
// docs/closure-plans/*.json; adding a similar exemption that
// names any closure-package Go file would mask a real
// LLM-friendliness regression.
func TestSourceStructureNoAllowlistInLLMFriendly(t *testing.T) {
	data, err := os.ReadFile("../llmfriendly/check.go")
	if err != nil {
		t.Fatalf("read llmfriendly/check.go: %v", err)
	}
	content := string(data)
	markers := []string{
		"internal/factory/closure/",
		"plan_contract_",
		"closure package",
	}
	for _, marker := range markers {
		if strings.Contains(content, marker) {
			t.Fatalf("llmfriendly/check.go mentions %q; that is a forbidden allowlist for the closure package", marker)
		}
	}
}

// TestSourceStructureNamingAudit audits every test function in
// the closure package whose name carries the typed-stage marker.
// Each such test MUST reach the typed-decode stage by referring
// to a typed-decoder binding. The audit walks the parsed AST
// directly so a misleading name without AST evidence of the
// seam is reported. The audit's own name does not contain the
// typed-stage marker, so it does not trip its own filter.
// Parser errors fail the audit; the audit cannot silently pass
// on a malformed source.
func TestSourceStructureNamingAudit(t *testing.T) {
	files, err := filepath.Glob("./*_test.go")
	if err != nil {
		t.Fatalf("glob: %v", err)
	}
	for _, path := range files {
		body, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		violations, err := findNamingViolations(string(body))
		if err != nil {
			t.Fatalf("parse %s: %v", path, err)
		}
		if len(violations) > 0 {
			t.Fatalf("%s contains naming violations %v", path, violations)
		}
	}
}

// TestSourceStructureNamingAuditAdversarialFixtures exercises
// the documented audit outcomes. Each fixture is parsed
// in-memory; they never enter the production source tree. The
// audit accepts a typed-failure test only when the body
// actually invokes the dep-aware composed pipeline; the
// test rejects any test that merely constructs a dependency
// bundle or calls a sentinel decoder.
func TestSourceStructureNamingAuditAdversarialFixtures(t *testing.T) {
	t.Run("deps-declaration-only-rejected", func(t *testing.T) {
		src := `package fixture
import "testing"
func TestDepsOnlyTypedFailure(t *testing.T) {
	_ = composedValidationDeps{DecodeTyped: nil}
}`
		violations := mustRunAudit(t, src)
		if !containsStringInList(violations, "TestDepsOnlyTypedFailure") {
			t.Fatalf("expected TestDepsOnlyTypedFailure to be rejected; got %v", violations)
		}
	})
	t.Run("sentinel-call-only-rejected", func(t *testing.T) {
		src := `package fixture
import "testing"
func TestSentinelOnlyTypedFailure(t *testing.T) {
	sentinelTypedDecode(nil, noopCompositionObserver{})
}`
		violations := mustRunAudit(t, src)
		if !containsStringInList(violations, "TestSentinelOnlyTypedFailure") {
			t.Fatalf("expected TestSentinelOnlyTypedFailure to be rejected; got %v", violations)
		}
	})
	t.Run("dependency-aware-composed-call-accepted", func(t *testing.T) {
		src := `package fixture
import "testing"
func TestComposedCallTypedFailure(t *testing.T) {
	deps := composedValidationDeps{DecodeTyped: sentinelTypedDecode}
	obs := noopCompositionObserver{}
	_ = validatePlanComposedWithObserverAndDeps(nil, obs, deps)
}`
		violations := mustRunAudit(t, src)
		if containsStringInList(violations, "TestComposedCallTypedFailure") {
			t.Fatalf("expected TestComposedCallTypedFailure to be accepted; got %v", violations)
		}
	})
	t.Run("dependency-aware-wrapper-call-accepted", func(t *testing.T) {
		src := `package fixture
import "testing"
func TestWrapperCallTypedFailure(t *testing.T) {
	deps := composedValidationDeps{DecodeTyped: sentinelTypedDecode}
	obs := noopCompositionObserver{}
	_, _ = validatePlanStructuralAndSemanticWith(nil, obs, deps)
}`
		violations := mustRunAudit(t, src)
		if containsStringInList(violations, "TestWrapperCallTypedFailure") {
			t.Fatalf("expected TestWrapperCallTypedFailure to be accepted; got %v", violations)
		}
	})
	t.Run("structural-test-without-typed-failure-ignored", func(t *testing.T) {
		src := `package fixture
import "testing"
func TestOrdinaryStructuralFailure(t *testing.T) {
}`
		violations := mustRunAudit(t, src)
		if len(violations) != 0 {
			t.Fatalf("ordinary structural failure name should be ignored; got %v", violations)
		}
	})
	t.Run("duplicate-declaration-rejected", func(t *testing.T) {
		src := `package fixture
import "testing"
func TestDuplicateTypedFailure(t *testing.T) {
	_ = validatePlanComposedWithObserverAndDeps(nil, noopCompositionObserver{}, composedValidationDeps{})
}
func TestDuplicateTypedFailure(t *testing.T) {
	_ = validatePlanComposedWithObserverAndDeps(nil, noopCompositionObserver{}, composedValidationDeps{})
}`
		violations := mustRunAudit(t, src)
		if !containsStringInList(violations, "TestDuplicateTypedFailure") {
			t.Fatalf("expected TestDuplicateTypedFailure to be rejected; got %v", violations)
		}
	})
	t.Run("parse-failure-rejected", func(t *testing.T) {
		src := `package fixture
this is not valid Go {`
		_, err := findNamingViolations(src)
		if err == nil {
			t.Fatalf("parse failure must return error")
		}
	})
}

// TestSourceStructureNamingAuditNilBodyRejected constructs a
// fake FuncDecl with Body=nil and asserts the audit rejects it.
// This case cannot arise from parsing real Go source because
// every parsed FuncDecl has a body; the audit must still be
// fail-closed so a future tooling change that constructs
// body-less declarations is caught.
func TestSourceStructureNamingAuditNilBodyRejected(t *testing.T) {
	fakeDecl := &ast.FuncDecl{
		Name: &ast.Ident{Name: "TestNilBodyTypedFailure"},
		Type: &ast.FuncType{
			Params: &ast.FieldList{List: []*ast.Field{{
				Names: []*ast.Ident{{Name: "t"}},
				Type:  &ast.Ident{Name: "testingT"},
			}}},
		},
		Body: nil,
	}
	violations := findNamingViolationsInDecls([]ast.Decl{fakeDecl})
	if !containsStringInList(violations, "TestNilBodyTypedFailure") {
		t.Fatalf("expected TestNilBodyTypedFailure to be rejected; got %v", violations)
	}
}

// findNamingViolations parses the file body and returns every
// function declaration whose name carries the typed-stage
// marker but whose body fails the seam, duplicate, or nil-body
// checks. Parser errors fail closed.
func findNamingViolations(body string) ([]string, error) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "body.go", body, parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("parse: %w", err)
	}
	return findNamingViolationsInDecls(file.Decls), nil
}

// findNamingViolationsInDecls inspects every function
// declaration in decls. The audit fails closed when a function
// whose name carries the typed-stage marker:
//   - has a duplicate declaration (more than one FuncDecl
//     sharing the name);
//   - has a nil body (the audit cannot prove seam usage);
//   - has a body whose AST does not exercise the typed-decoder
//     seam.
//
// The audit does NOT require the body to repeat the marker
// text — a function whose name claims a typed-stage scenario
// must show AST evidence of the seam regardless of how the body
// is written.
func findNamingViolationsInDecls(decls []ast.Decl) []string {
	var violations []string
	nameCount := make(map[string]int)
	for _, decl := range decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok {
			continue
		}
		nameCount[fn.Name.Name]++
	}
	for _, decl := range decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok {
			continue
		}
		name := fn.Name.Name
		if !containsAny(name, typedStageNamingMarkers...) {
			continue
		}
		if nameCount[name] > 1 {
			violations = append(violations, name)
			continue
		}
		if fn.Body == nil {
			violations = append(violations, name)
			continue
		}
		if !bodyExercisesTypedDecoder(fn.Body) {
			violations = append(violations, name)
		}
	}
	return violations
}

// bodyExercisesTypedDecoder walks the AST of body and reports
// whether the function actually invokes the dep-aware composed
// pipeline. A test that merely constructs a
// composedValidationDeps composite literal, references the
// DecodeTyped selector, or calls a sentinel decoder does NOT
// prove the typed stage ran; the audit requires an actual call
// to validatePlanComposedWithObserverAndDeps or
// validatePlanStructuralAndSemanticWith so the test proves it
// executed the seam.
func bodyExercisesTypedDecoder(body *ast.BlockStmt) bool {
	if body == nil {
		return false
	}
	var found bool
	ast.Inspect(body, func(n ast.Node) bool {
		if found {
			return false
		}
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		ident, ok := call.Fun.(*ast.Ident)
		if !ok {
			return true
		}
		if ident.Name == "validatePlanComposedWithObserverAndDeps" ||
			ident.Name == "validatePlanStructuralAndSemanticWith" {
			found = true
		}
		return true
	})
	return found
}

// mustRunAudit parses src and returns the violations. Parse
// errors fail the test.
func mustRunAudit(t *testing.T, src string) []string {
	t.Helper()
	violations, err := findNamingViolations(src)
	if err != nil {
		t.Fatalf("audit parse error: %v", err)
	}
	return violations
}

// containsStringInList reports whether needle is present in
// haystack. The name is verbose to avoid a conflict with a
// different containsString helper in run_v2_authority_integration_test.go.
func containsStringInList(haystack []string, needle string) bool {
	for _, s := range haystack {
		if s == needle {
			return true
		}
	}
	return false
}

// containsAny reports whether haystack contains any of the
// needles (case-insensitive).
func containsAny(haystack string, needles ...string) bool {
	lower := strings.ToLower(haystack)
	for _, n := range needles {
		if strings.Contains(lower, strings.ToLower(n)) {
			return true
		}
	}
	return false
}

// scanFile reads path and asserts every line is at most
// maxLineLength runes long. The line counter and offending
// snippet are reported in the failure message.
func scanFile(t *testing.T, path string) {
	t.Helper()
	file, err := os.Open(path)
	if err != nil {
		t.Fatalf("open %s: %v", path, err)
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	const bufferSize = 1024 * 1024
	scanner.Buffer(make([]byte, bufferSize), bufferSize)
	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := scanner.Text()
		if len(line) <= maxLineLength {
			continue
		}
		t.Fatalf("%s line %d: %d chars (max %d): %s",
			path, lineNum, len(line), maxLineLength, truncate(line, 80))
	}
	if err := scanner.Err(); err != nil {
		t.Fatalf("%s scan: %v", path, err)
	}
}

// truncate returns the first n characters of s plus an ellipsis
// when truncation happened.
func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
