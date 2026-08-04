package closure

import (
	"go/parser"
	"go/token"
	"testing"
)

// plan_semantic_ast_audit_test.go verifies that the closure package
// never constructs semantic paths or identities by parsing error
// messages. The audit uses the Go AST to detect:
//
//   - String operations on err.Error(): HasPrefix, Contains, Index,
//     Split from the strings package
//   - Regexp-based matching: MustCompile, Compile, Match, MatchString,
//     Find, FindString, FindAll
//   - Prohibited declarations: semanticDiagnostic, semanticPathFromError,
//     extractDuplicateCheckIDPath
//
// This test exists because error message parsing is fragile, non-
// deterministic across locales, and creates hidden coupling to
// error formatting. The closure package uses typed semantic errors
// that carry exact JSON Pointer paths via PlanDiagnostics(); parsing
// err.Error() bypasses that machinery entirely.

// TestSemanticASTAuditNoErrErrorParsing verifies that no production
// source file uses err.Error() as input to semantic path or identity
// construction. The audit is fail-closed: any use of strings functions
// or regexp operations on err.Error() triggers a test failure.
func TestSemanticASTAuditNoErrErrorParsing(t *testing.T) {
	roots, err := loadSemanticAuditSources(t)
	if err != nil {
		t.Fatalf("load sources: %v", err)
	}
	for _, file := range roots {
		path := shortFile(token.NewFileSet(), file)
		if err := auditFileForErrErrorParsing(file, path); err != nil {
			t.Error(err)
		}
	}
}

// TestSemanticASTAuditNoProhibitedDeclarations verifies that the
// prohibited function declarations are absent from all production
// source files. These functions were identified as anti-patterns
// for semantic path construction and must not be reintroduced.
func TestSemanticASTAuditNoProhibitedDeclarations(t *testing.T) {
	roots, err := loadSemanticAuditSources(t)
	if err != nil {
		t.Fatalf("load sources: %v", err)
	}
	prohibited := []string{
		"semanticDiagnostic",
		"semanticPathFromError",
		"extractDuplicateCheckIDPath",
	}
	for _, file := range roots {
		path := shortFile(token.NewFileSet(), file)
		for _, name := range prohibited {
			if hasFunctionDecl(file, name) {
				t.Errorf("file %s contains prohibited declaration %q", path, name)
			}
		}
	}
}

// TestSemanticASTAuditAdversarialPositiveFixtures proves the detector
// catches each forbidden pattern. Each fixture exercises a specific
// strings or regexp operation on err.Error() that must be rejected.
func TestSemanticASTAuditAdversarialPositiveFixtures(t *testing.T) {
	cases := []struct {
		name string
		src  string
	}{
		{
			name: "errError-HasPrefix",
			src: `package fixture
import "errors"
import "strings"
func parseError(err error) bool {
	return strings.HasPrefix(err.Error(), "validation failed")
}`,
		},
		{
			name: "errError-Contains",
			src: `package fixture
import "errors"
import "strings"
func checkError(err error) bool {
	return strings.Contains(err.Error(), "path:")
}`,
		},
		{
			name: "errError-Index",
			src: `package fixture
import "errors"
import "strings"
func findPath(err error) int {
	return strings.Index(err.Error(), "/checks/")
}`,
		},
		{
			name: "errError-Split",
			src: `package fixture
import "errors"
import "strings"
func splitError(err error) []string {
	return strings.Split(err.Error(), ":")
}`,
		},
		{
			name: "errError-MustCompile",
			src: `package fixture
import "errors"
import "regexp"
func matchError(err error) bool {
	re := regexp.MustCompile("path:.*")
	return re.MatchString(err.Error())
}`,
		},
		{
			name: "errError-Compile-Match",
			src: `package fixture
import "errors"
import "regexp"
func matchError(err error) bool {
	re, _ := regexp.Compile("check.*failed")
	return re.Match(err.Error())
}`,
		},
		{
			name: "errError-FindString",
			src: `package fixture
import "errors"
import "regexp"
func extractPath(err error) string {
	re := regexp.MustCompile(` + "`" + `/[^/]+/[^/]+` + "`" + `)
	return re.FindString(err.Error())
}`,
		},
		{
			name: "errError-Find",
			src: `package fixture
import "errors"
import "regexp"
func findMatch(err error) []byte {
	re := regexp.MustCompile("error at.*")
	return re.Find([]byte(err.Error()))
}`,
		},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			file, err := parser.ParseFile(token.NewFileSet(), "fixture.go", tc.src, 0)
			if err != nil {
				t.Fatalf("parse fixture: %v", err)
			}
			err = auditFileForErrErrorParsing(file, "fixture.go")
			if err == nil {
				t.Fatalf("detector must reject %s: audit passed but should have failed", tc.name)
			}
		})
	}
}

// TestSemanticASTAuditAdversarialNegativeFixtures proves the detector
// accepts legitimate patterns that must not be flagged. Each fixture
// represents valid code that happens to use strings or regexp but
// not on err.Error().
func TestSemanticASTAuditAdversarialNegativeFixtures(t *testing.T) {
	cases := []struct {
		name string
		src  string
	}{
		{
			name: "strings-on-regular-string",
			src: `package fixture
import "strings"
func check(s string) bool {
	return strings.HasPrefix(s, "validation")
}`,
		},
		{
			name: "strings-on-json-pointer",
			src: `package fixture
import "strings"
func validatePath(path string) bool {
	return strings.Contains(path, "/checks/")
}`,
		},
		{
			name: "regexp-on-constant",
			src: `package fixture
import "regexp"
var checkRE = regexp.MustCompile("^[a-z]+$")
func isValid(s string) bool {
	return checkRE.MatchString(s)
}`,
		},
		{
			name: "regexp-on-struct-field",
			src: `package fixture
import "regexp"
type Result struct {
	Message string
}
var pathRE = regexp.MustCompile(` + "`" + `/[^/]+/[^/]+` + "`" + `)
func extractPath(r Result) string {
	return pathRE.FindString(r.Message)
}`,
		},
		{
			name: "errError-but-not-strings",
			src: `package fixture
import "errors"
func getMessage(err error) string {
	return err.Error()
}`,
		},
		{
			name: "errError-with-strings-but-not-path",
			src: `package fixture
import "errors"
import "strings"
func getLength(err error) int {
	return len(err.Error())
}
func upperError(err error) string {
	return strings.ToUpper(err.Error())
}`,
		},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			file, err := parser.ParseFile(token.NewFileSet(), "fixture.go", tc.src, 0)
			if err != nil {
				t.Fatalf("parse fixture: %v", err)
			}
			err = auditFileForErrErrorParsing(file, "fixture.go")
			if err != nil {
				t.Fatalf("detector must NOT reject %s: audit failed but should have passed: %v", tc.name, err)
			}
		})
	}
}
