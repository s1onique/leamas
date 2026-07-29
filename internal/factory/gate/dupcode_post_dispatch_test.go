// SPDX-License-Identifier: Apache-2.0

package gate

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// protectedCallNames is the set of symbol names that must NOT appear in the
// post-Dispatch region of any typed entry-point function. They map to
// method calls or factory invocations that would re-execute protected work.
var protectedCallNames = []string{
	"binder.run",
	"BindRunner",
	"deps.NewRunner",
	"NewDupcodeRunner",
	"LoadBaseline",
	"RunCheckReport",
	"VerifyBaseline",
	"WriteBaseline",
	"CompareToBaseline",
}

// typedEntryPointInternal lists the package-internal typed dispatch
// entry-point functions whose body must own exactly one
// dispatcher.Dispatch call and must contain no post-Dispatch protected
// call.
var typedEntryPointInternal = []string{
	"dispatchDupcodeVerifyTypedWith",
	"dispatchDupcodeBaselineVerifyTypedWith",
	"dispatchDupcodeUpdateBaselineTypedWith",
}

// typedEntryPointPublic lists the public typed dispatch entry-point
// functions. Each must delegate exactly once to its corresponding
// internal *With function and must contain no direct call to a
// protected operation.
var typedEntryPointPublic = []string{
	"DispatchDupcodeVerifyTyped",
	"DispatchDupcodeBaselineVerifyTyped",
	"DispatchDupcodeUpdateBaselineTyped",
}

// TestDupcodeTypedEntryPointsNoPostDispatchProtectedCalls is a fail-closed
// structural guard over the typed dispatch entry points. The test parses
// every production Go file in the gate package with go/parser, locates
// each expected entry-point declaration (FAIL on missing or duplicate),
// walks its body to find the dispatcher.Dispatch statement, and asserts
// that no protected call appears in any subsequent top-level statement.
// Public wrappers are also required to delegate to the matching
// internal *With function and to contain no direct protected call.
func TestDupcodeTypedEntryPointsNoPostDispatchProtectedCalls(t *testing.T) {
	root, err := findModuleRootForGateTest()
	if err != nil {
		t.Fatal(err)
	}
	srcDir := filepath.Join(root, "internal", "factory", "gate")

	type parsed struct {
		fset *token.FileSet
		file *ast.File
	}
	files := map[string]parsed{}
	entries, err := os.ReadDir(srcDir)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".go") || strings.HasSuffix(e.Name(), "_test.go") {
			continue
		}
		path := filepath.Join(srcDir, e.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, path, data, parser.AllErrors)
		if err != nil {
			t.Fatalf("parse %s: %v", e.Name(), err)
		}
		files[e.Name()] = parsed{fset: fset, file: f}
	}

	type declHit struct {
		file string
		pos  token.Pos
		body *ast.BlockStmt
	}
	hits := map[string][]declHit{}
	for fname, p := range files {
		for _, decl := range p.file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok {
				continue
			}
			hits[fn.Name.Name] = append(hits[fn.Name.Name], declHit{
				file: fname,
				pos:  fn.Pos(),
				body: fn.Body,
			})
		}
	}

	checkExactlyOne := func(name string) declHit {
		t.Helper()
		found := hits[name]
		if len(found) == 0 {
			t.Errorf("required declaration %q is missing", name)
			return declHit{}
		}
		if len(found) > 1 {
			t.Errorf("required declaration %q appears %d times, want exactly 1 (in %s)",
				name, len(found), found[0].file)
			return declHit{}
		}
		return found[0]
	}

	// Internal *With entry points: must own exactly one
	// dispatcher.Dispatch statement and contain no protected call in
	// any subsequent top-level statement.
	for _, name := range typedEntryPointInternal {
		hit := checkExactlyOne(name)
		if hit.body == nil {
			continue
		}
		// Find the statement that calls dispatcher.Dispatch(...).
		dispatchStmtIdx := -1
		for i, stmt := range hit.body.List {
			if stmtHasDispatchCall(stmt) {
				dispatchStmtIdx = i
				break
			}
		}
		if dispatchStmtIdx < 0 {
			t.Errorf("%s: dispatcher.Dispatch call not found", name)
			continue
		}
		// Walk all SUBSEQUENT top-level statements. Anything inside the
		// dispatcher.Dispatch call (e.g. argument sub-expressions) is
		// excluded because the dispatch statement itself is the boundary.
		for j := dispatchStmtIdx + 1; j < len(hit.body.List); j++ {
			ast.Inspect(hit.body.List[j], func(n ast.Node) bool {
				if n == nil {
					return true
				}
				call, ok := n.(*ast.CallExpr)
				if !ok {
					return true
				}
				for _, bad := range protectedCallNames {
					if callContains(call, bad) {
						t.Errorf("%s: post-Dispatch statement at index %d contains protected call %q",
							name, j, bad)
					}
				}
				return true
			})
		}
	}

	// Public wrappers: must delegate exactly once to the matching
	// internal *With function and contain no direct call to a protected
	// operation anywhere in the wrapper body.
	for i, name := range typedEntryPointPublic {
		hit := checkExactlyOne(name)
		if hit.body == nil {
			continue
		}
		expectedInternal := typedEntryPointInternal[i]
		delegateCount := 0
		ast.Inspect(hit.body, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			sel, ok := call.Fun.(*ast.SelectorExpr)
			if ok && sel.Sel.Name == expectedInternal {
				delegateCount++
				return true
			}
			ident, ok := call.Fun.(*ast.Ident)
			if !ok || ident.Name != expectedInternal {
				return true
			}
			delegateCount++
			return true
		})
		if delegateCount != 1 {
			t.Errorf("%s: must delegate exactly once to %s, got %d calls",
				name, expectedInternal, delegateCount)
		}
		ast.Inspect(hit.body, func(n ast.Node) bool {
			if n == nil {
				return true
			}
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			for _, bad := range protectedCallNames {
				if callContains(call, bad) {
					t.Errorf("%s: directly calls protected operation %q", name, bad)
				}
			}
			return true
		})
	}
}

// stmtHasDispatchCall reports whether the top-level statement contains a
// call to dispatcher.Dispatch(...). The call may be the statement's
// expression, or be embedded in an assignment like
//
//	Dispatch: dispatcher.Dispatch(...)
func stmtHasDispatchCall(stmt ast.Stmt) bool {
	found := false
	ast.Inspect(stmt, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok || sel.Sel.Name != "Dispatch" {
			return true
		}
		// Bound receiver is "dispatcher" (the local variable in the
		// typed entry points).
		ident, ok := sel.X.(*ast.Ident)
		if !ok {
			return true
		}
		if ident.Name == "dispatcher" {
			found = true
			return false
		}
		return true
	})
	return found
}

func callContains(call *ast.CallExpr, name string) bool {
	switch f := call.Fun.(type) {
	case *ast.Ident:
		return f.Name == name
	case *ast.SelectorExpr:
		if f.Sel.Name == name {
			return true
		}
	}
	return false
}

func findModuleRootForGateTest() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	dir := cwd
	for i := 0; i < 8; i++ {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return "", fmt.Errorf("go.mod not found")
}
