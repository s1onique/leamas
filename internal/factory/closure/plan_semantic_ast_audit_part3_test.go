package closure

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"regexp"
	"strings"
	"testing"
)

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

// auditFileForLocalTaintParsing audits a single file for tainted identifier
// usage within function scopes. It tracks local assignments from err.Error()
// and rejects forbidden operations on tainted identifiers.
//
// This analysis is function-local only and does not perform whole-program
// data flow analysis.
func auditFileForLocalTaintParsing(file *ast.File, path string) error {
	var diagnostics []string

	// Visit each function declaration
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Body == nil {
			continue
		}

		tracker := newTaintTracker()
		diags := auditFunctionBody(fn.Body, tracker)
		for _, d := range diags {
			diagnostics = append(diagnostics, fmt.Sprintf("%s: %s", path, d))
		}
	}

	if len(diagnostics) > 0 {
		return fmt.Errorf("local taint audit failed:\n  %s", strings.Join(diagnostics, "\n  "))
	}
	return nil
}

// auditFunctionBody analyzes a function body for tainted identifier usage.
// Returns a list of diagnostic messages for any violations found.
func auditFunctionBody(body *ast.BlockStmt, tracker *taintTracker) []string {
	var diagnostics []string

	// First pass: update taint state based on assignments
	for _, stmt := range body.List {
		processAssignment(stmt, tracker)
	}

	// Second pass: check for forbidden operations
	for _, stmt := range body.List {
		diags := checkForbiddenOperations(stmt, tracker)
		diagnostics = append(diagnostics, diags...)
	}

	return diagnostics
}

// processAssignment updates taint state based on assignment statements.
// Handles:
//   - msg := err.Error()  (taint the identifier)
//   - msg = err.Error()   (taint the identifier)
//   - msg = "safe"        (untaint if reassigned from literal)
func processAssignment(stmt ast.Stmt, tracker *taintTracker) {
	// Handle short var declarations: msg := expr
	if assign, ok := stmt.(*ast.AssignStmt); ok {
		for i, lhs := range assign.Lhs {
			ident, ok := lhs.(*ast.Ident)
			if !ok {
				continue
			}

			// For assignments, Rhs is a slice, so we need to index it
			if i >= len(assign.Rhs) {
				continue
			}
			rhs := assign.Rhs[i]

			if isErrErrorCall(rhs) {
				tracker.markTainted(ident.Name)
			} else if isSafeLiteral(rhs) {
				// Reassignment from a safe literal un-taints the variable
				tracker.unmarkTainted(ident.Name)
			} else if srcIdent, ok := rhs.(*ast.Ident); ok {
				// Simple assignment: ident = otherIdent
				if tracker.isTainted(srcIdent.Name) {
					tracker.markTainted(ident.Name)
				} else {
					// Assigning from non-tainted source un-taints
					tracker.unmarkTainted(ident.Name)
				}
			}
		}
	}
}

// isErrErrorCall reports whether expr is a call to err.Error() where err is an identifier.
func isErrErrorCall(expr ast.Expr) bool {
	call, ok := expr.(*ast.CallExpr)
	if !ok {
		return false
	}
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	if sel.Sel.Name != "Error" {
		return false
	}
	_, ok = sel.X.(*ast.Ident)
	return ok
}

// isSafeLiteral reports whether expr is a literal value (string, etc).
// These do not carry taint from error messages.
func isSafeLiteral(expr ast.Expr) bool {
	switch expr.(type) {
	case *ast.BasicLit:
		return true
	default:
		return false
	}
}

// checkForbiddenOperations checks if any identifier in a statement is
// a tainted identifier passed to forbidden strings or regexp functions.
func checkForbiddenOperations(stmt ast.Stmt, tracker *taintTracker) []string {
	var diagnostics []string

	ast.Inspect(stmt, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}

		// Check each argument
		for _, arg := range call.Args {
			if ident, ok := arg.(*ast.Ident); ok {
				if tracker.isTainted(ident.Name) {
					fnName := extractCalleeName(call)
					if fnName != "" && isForbiddenStringFunc(fnName) {
						diagnostics = append(diagnostics,
							fmt.Sprintf("tainted identifier %q passed to strings.%s", ident.Name, fnName))
					}
					if fnName != "" && isForbiddenRegexpFunc(fnName) {
						diagnostics = append(diagnostics,
							fmt.Sprintf("tainted identifier %q passed to regexp.%s", ident.Name, fnName))
					}
				}
			}

			// Also check if err.Error() is passed directly
			if isErrErrorCall(arg) {
				fnName := extractCalleeName(call)
				if fnName != "" && isForbiddenStringFunc(fnName) {
					diagnostics = append(diagnostics,
						fmt.Sprintf("err.Error() passed directly to strings.%s", fnName))
				}
				if fnName != "" && isForbiddenRegexpFunc(fnName) {
					diagnostics = append(diagnostics,
						fmt.Sprintf("err.Error() passed directly to regexp.%s", fnName))
				}
			}
		}

		return true
	})

	return diagnostics
}

// TestLocalAliasTaintTrackingPositiveFixtures tests that the local taint
// tracker correctly rejects tainted identifier usage.
func TestLocalAliasTaintTrackingPositiveFixtures(t *testing.T) {
	cases := []struct {
		name    string
		src     string
		wantErr bool
	}{
		{
			name: "direct-errError-to-strings.Contains-rejected",
			src: `package fixture
import "strings"
import "errors"
func check(err error) bool {
	return strings.Contains(err.Error(), "path:")
}`,
			wantErr: true,
		},
		{
			name: "msg-assigned-errError-strings.Contains-rejected",
			src: `package fixture
import "strings"
import "errors"
func check(err error) bool {
	msg := err.Error()
	return strings.Contains(msg, "path:")
}`,
			wantErr: true,
		},
		{
			name: "msg-assigned-errError-strings.SplitN-rejected",
			src: `package fixture
import "strings"
import "errors"
func split(err error) []string {
	msg := err.Error()
	return strings.SplitN(msg, ":", 2)
}`,
			wantErr: true,
		},
		{
			name: "msg-reassigned-errError-strings.SplitN-rejected",
			src: `package fixture
import "strings"
import "errors"
func split(err error) []string {
	msg := "initial"
	msg = err.Error()
	return strings.SplitN(msg, ":", 2)
}`,
			wantErr: true,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			file, err := parser.ParseFile(token.NewFileSet(), "fixture.go", tc.src, 0)
			if err != nil {
				t.Fatalf("parse fixture: %v", err)
			}
			// Check both direct usage and local taint tracking
			err = auditFileForErrErrorParsing(file, "fixture.go")
			err2 := auditFileForLocalTaintParsing(file, "fixture.go")
			gotErr := (err != nil) || (err2 != nil)
			if gotErr != tc.wantErr {
				if tc.wantErr {
					t.Fatalf("detector must reject %s: both audits passed but should have failed", tc.name)
				}
				t.Fatalf("detector must NOT reject %s: audit incorrectly flagged", tc.name)
			}
		})
	}
}

// TestLocalAliasTaintTrackingNegativeFixtures tests that the local taint
// tracker correctly allows legitimate patterns.
func TestLocalAliasTaintTrackingNegativeFixtures(t *testing.T) {
	cases := []struct {
		name    string
		src     string
		wantErr bool
	}{
		{
			name: "msg-sanitized-strings.Contains-accepted",
			src: `package fixture
import "strings"
import "errors"
func check(err error) bool {
	msg := err.Error()
	msg = "safe"
	return strings.Contains(msg, "path:")
}`,
			wantErr: false,
		},
		{
			name: "ordinary-string-alias-accepted",
			src: `package fixture
import "strings"
func check(data string) bool {
	msg := data
	return strings.Contains(msg, "path:")
}`,
			wantErr: false,
		},
		{
			name: "tainted-used-with-safe-function-accepted",
			src: `package fixture
import "strings"
import "errors"
func check(err error) int {
	msg := err.Error()
	return len(msg)
}`,
			wantErr: false,
		},
		{
			name: "safe-string-to-tainted-accepted",
			src: `package fixture
import "strings"
import "errors"
func check(err error) bool {
	safe := "constant"
	msg := safe
	return strings.Contains(msg, "path:")
}`,
			wantErr: false,
		},
		{
			name: "string-split-on-safe-data-accepted",
			src: `package fixture
import "strings"
func split(data string) []string {
	return strings.SplitN(data, ":", 2)
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
			err = auditFileForLocalTaintParsing(file, "fixture.go")
			gotErr := (err != nil)
			if gotErr != tc.wantErr {
				if tc.wantErr {
					t.Fatalf("detector must reject %s: audit passed but should have failed", tc.name)
				}
				t.Fatalf("detector must NOT reject %s: audit incorrectly flagged: %v", tc.name, err)
			}
		})
	}
}

// keep lint happy
var _ = regexp.Compile
