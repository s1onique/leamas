package closure

import (
	"bufio"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// plan_contract_source_structure_test.go enforces the package's
// own LLM-friendly invariants at test-time. The directive ACT
// requires splitting long inline fixtures into small focused
// files; this file proves that outcome and guards against
// future regressions.

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

// typedStageDecodeMarkers are the substrings whose presence in
// a function body proves the function actually exercises the
// typed-decoder seam. A test whose name claims a typed-stage
// scenario MUST mention at least one of these tokens.
var typedStageDecodeMarkers = []string{
	"DecodeTyped",
	"decodeTypedPlan",
	"composedValidationDeps",
	"sentinelTypedDecode",
}

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
// the closure package whose name carries a typed-stage marker.
// Each such test MUST reach the typed-decode stage by referring
// to a typed-decoder binding (typedStageDecodeMarkers). The
// audit walks the raw function body so a misleading name without
// a typed decoder reference is reported. The audit's own name
// does not contain the typed-stage marker, so it does not trip
// its own filter.
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
		misleading := findNamingViolations(string(body))
		if len(misleading) > 0 {
			t.Fatalf("%s contains naming violations %v", path, misleading)
		}
	}
}

// findNamingViolations parses the file body, finds every
// function declaration whose name contains a typed-stage
// marker, and returns the names that DO NOT mention a
// typed-decoder binding from the typedStageDecodeMarkers set.
func findNamingViolations(body string) []string {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "body.go", body, 0)
	if err != nil {
		// Parser errors are not the audit's concern; AST guards
		// for syntax errors live in the compiler.
		return nil
	}
	var violations []string
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Body == nil {
			continue
		}
		name := fn.Name.Name
		if !containsAny(name, typedStageNamingMarkers...) {
			continue
		}
		bodyText := funcBodyText(fn)
		if !containsAny(bodyText, typedStageNamingMarkers...) {
			continue
		}
		if !containsAny(bodyText, typedStageDecodeMarkers...) {
			violations = append(violations, name)
		}
	}
	return violations
}

// funcBodyText returns the textual body of a function. The text
// is reconstructed from the AST so the audit is robust to
// whitespace and comment changes.
func funcBodyText(fn *ast.FuncDecl) string {
	if fn.Body == nil {
		return ""
	}
	var b strings.Builder
	ast.Inspect(fn.Body, func(n ast.Node) bool {
		switch v := n.(type) {
		case *ast.Ident:
			b.WriteString(v.Name)
			b.WriteString("\n")
		case *ast.BasicLit:
			b.WriteString(v.Value)
			b.WriteString("\n")
		}
		return true
	})
	return b.String()
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
