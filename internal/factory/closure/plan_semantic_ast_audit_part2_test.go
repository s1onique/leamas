package closure

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"strings"
	"testing"
)

// TestSemanticASTAuditProhibitedDeclarationsFixtures proves the
// detector catches prohibited function declarations.
func TestSemanticASTAuditProhibitedDeclarationsFixtures(t *testing.T) {
	cases := []struct {
		name    string
		src     string
		wantErr bool
	}{
		{
			name: "semanticDiagnostic-prohibited",
			src: `package fixture
func semanticDiagnostic(msg string) string {
	return "diagnostic: " + msg
}`,
			wantErr: true,
		},
		{
			name: "semanticPathFromError-prohibited",
			src: `package fixture
func semanticPathFromError(err error) string {
	return err.Error()
}`,
			wantErr: true,
		},
		{
			name: "extractDuplicateCheckIDPath-prohibited",
			src: `package fixture
func extractDuplicateCheckIDPath(err error) string {
	return err.Error()
}`,
			wantErr: true,
		},
		{
			name: "allowed-function-name",
			src: `package fixture
func extractPath(data string) string {
	return data
}`,
			wantErr: false,
		},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			file, err := parser.ParseFile(token.NewFileSet(), "fixture.go", tc.src, 0)
			if err != nil {
				t.Fatalf("parse fixture: %v", err)
			}
			prohibited := []string{
				"semanticDiagnostic",
				"semanticPathFromError",
				"extractDuplicateCheckIDPath",
			}
			var gotErr bool
			for _, name := range prohibited {
				if hasFunctionDecl(file, name) {
					gotErr = true
					break
				}
			}
			if gotErr != tc.wantErr {
				if tc.wantErr {
					t.Fatalf("detector must reject %s: prohibited declaration not found", tc.name)
				}
				t.Fatalf("detector must NOT reject %s: fixture incorrectly flagged", tc.name)
			}
		})
	}
}

// auditFileForErrErrorParsing walks the AST of file and reports
// whether any strings or regexp operation is applied to an
// err.Error() call. The check is fail-closed on parser errors.
func auditFileForErrErrorParsing(file *ast.File, path string) error {
	var diagnostics []string
	ast.Inspect(file, func(n ast.Node) bool {
		// Look for err.Error() call expressions
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		if !isErrorDotErrorCall(call) {
			return true
		}
		// Find the enclosing expression that uses the result
		// We need to check if this is passed to a strings or regexp function
		enclosingErr := findEnclosingStringRegexpUse(file, call)
		if enclosingErr != "" {
			diagnostics = append(diagnostics, fmt.Sprintf("%s: err.Error() passed to %s", path, enclosingErr))
		}
		return true
	})
	if len(diagnostics) > 0 {
		return fmt.Errorf("audit failed:\n  %s", strings.Join(diagnostics, "\n  "))
	}
	return nil
}

// isErrorDotErrorCall reports whether expr is a call expression
// of the form err.Error() where err is an identifier.
func isErrorDotErrorCall(call *ast.CallExpr) bool {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	if sel.Sel.Name != "Error" {
		return false
	}
	_, ok = sel.X.(*ast.Ident)
	if !ok {
		return false
	}
	return true
}

// findEnclosingStringRegexpUse examines the parent nodes of errCall
// to detect if the result is passed to a strings or regexp function.
// Returns the name of the offending function if found, empty string otherwise.
func findEnclosingStringRegexpUse(file *ast.File, errCall *ast.CallExpr) string {
	var enclosingFunc string
	ast.Inspect(file, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		// Check if errCall is an argument to this call
		for _, arg := range call.Args {
			if referencesNode(errCall, arg) {
				fnName := extractCalleeName(call)
				if fnName != "" && isForbiddenStringFunc(fnName) {
					enclosingFunc = "strings." + fnName
					return false
				}
				if fnName != "" && isForbiddenRegexpFunc(fnName) {
					enclosingFunc = "regexp." + fnName
					return false
				}
			}
		}
		return true
	})
	return enclosingFunc
}

// referencesNode reports whether the expr subtree contains or equals target.
func referencesNode(target ast.Node, expr ast.Expr) bool {
	found := false
	ast.Inspect(expr, func(n ast.Node) bool {
		if n == target {
			found = true
			return false
		}
		return true
	})
	return found
}

// extractCalleeName returns the identifier name of a call's function,
// or empty string if it's not a simple identifier.
func extractCalleeName(call *ast.CallExpr) string {
	ident, ok := call.Fun.(*ast.Ident)
	if !ok {
		// Also handle selector expressions for package-qualified calls
		if sel, ok := call.Fun.(*ast.SelectorExpr); ok {
			if ident, ok := sel.X.(*ast.Ident); ok {
				_ = ident // package name is unused here
				return sel.Sel.Name
			}
		}
		return ""
	}
	return ident.Name
}

// forbiddenStringFuncs is the set of strings package functions
// that are forbidden when applied to err.Error() for semantic
// path or identity construction.
var forbiddenStringFuncs = map[string]bool{
	"HasPrefix": true,
	"Contains":  true,
	"Index":     true,
	"Split":     true,
	"SplitN":    true,
	// Note: ToUpper, ToLower, Trim, etc. are allowed as they don't
	// extract semantic information from the error message structure.
}

// isForbiddenStringFunc reports whether the named function is
// forbidden when used on err.Error().
func isForbiddenStringFunc(name string) bool {
	return forbiddenStringFuncs[name]
}

// forbiddenRegexpFuncs is the set of regexp package functions
// that are forbidden when applied to err.Error() for semantic
// path or identity construction.
var forbiddenRegexpFuncs = map[string]bool{
	"MustCompile":   true,
	"Compile":       true,
	"Match":         true,
	"MatchString":   true,
	"Find":          true,
	"FindString":    true,
	"FindAll":       true,
	"FindAllString": true,
}

// isForbiddenRegexpFunc reports whether the named function is
// forbidden when used on err.Error().
func isForbiddenRegexpFunc(name string) bool {
	return forbiddenRegexpFuncs[name]
}

// hasFunctionDecl reports whether file declares a function named name.
func hasFunctionDecl(file *ast.File, name string) bool {
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok {
			continue
		}
		if fn.Name.Name == name {
			return true
		}
	}
	return false
}

// loadSemanticAuditSources parses every non-test Go file in the
// closure package directory and returns the AST roots. This is
// the same helper used by the bounded parser authority tests.
func loadSemanticAuditSources(t *testing.T) ([]*ast.File, error) {
	t.Helper()
	entries, err := os.ReadDir(".")
	if err != nil {
		return nil, fmt.Errorf("read dir: %w", err)
	}
	var paths []string
	for _, entry := range entries {
		name := entry.Name()
		if strings.HasSuffix(name, ".go") && !strings.HasSuffix(name, "_test.go") {
			paths = append(paths, name)
		}
	}
	fset := token.NewFileSet()
	var roots []*ast.File
	for _, path := range paths {
		file, err := parser.ParseFile(fset, path, nil, 0)
		if err != nil {
			return nil, fmt.Errorf("parse %s: %w", path, err)
		}
		roots = append(roots, file)
	}
	return roots, nil
}

// ----------------------------------------------------------------------------
// Local Alias Taint Tracking
//
// This section implements function-local taint tracking for error message
// sources. Within each function body, we track identifiers that have been
// directly assigned from err.Error() as "tainted". Tainted identifiers remain
// tainted until they are reassigned from a non-tainted expression.
//
// This is NOT whole-program taint analysis. It is bounded to:
//   - Direct assignments within the same function scope
//   - Simple identifier-to-identifier reassignments
//   - Does not track data flow across function boundaries
//   - Does not handle aliasing through map/chan/slice operations
//   - Does not propagate through type conversions like []byte(msg)
//
// The purpose is to catch common patterns like:
//   msg := err.Error()
//   if strings.Contains(msg, "...") { ... }
// ----------------------------------------------------------------------------

// taintTracker tracks identifiers tainted by err.Error() assignments.
type taintTracker struct {
	// tainted contains identifiers known to hold values derived from err.Error()
	tainted map[string]bool
}

// newTaintTracker creates a fresh taint tracker for a function body.
func newTaintTracker() *taintTracker {
	return &taintTracker{
		tainted: make(map[string]bool),
	}
}

// isTainted reports whether ident is currently marked as tainted.
func (tt *taintTracker) isTainted(ident string) bool {
	return tt.tainted[ident]
}

// markTainted marks ident as tainted (derived from err.Error()).
func (tt *taintTracker) markTainted(ident string) {
	tt.tainted[ident] = true
}

// unmarkTainted removes ident from the tainted set.
func (tt *taintTracker) unmarkTainted(ident string) {
	delete(tt.tainted, ident)
}
