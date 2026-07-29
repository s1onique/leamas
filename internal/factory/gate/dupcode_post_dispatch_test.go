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
//
// Names are compared against the LAST selector segment of the call (the
// method name on a SelectorExpr, or the bare identifier on a plain
// function call). For example, "binder.run" matches the bare
// identifier "binder.run" only when the receiver is itself named that
// way; in production code the relevant calls are dispatched through a
// receiver (e.g. binder.run() appears as SelectorExpr {Sel: "run"}).
var protectedCallNames = []string{
	"run",
	"BindRunner",
	"NewRunner",
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

// dispatchCallInfo records the precise location of a single
// dispatcher.Dispatch call. The token interval is the exclusive
// boundary inside which the dispatch call executes; any protected
// call observed at a position overlapping that interval is a
// structural violation.
type dispatchCallInfo struct {
	pos       token.Pos
	end       token.Pos
	stmtStart token.Pos
}

// inspectDispatchCalls walks every *ast.CallExpr in fn and returns the
// slice of dispatchCallInfo records for dispatcher.Dispatch calls.
// The receiver must be the bare identifier "dispatcher"; this is the
// canonical local variable name used by every typed entry point.
func inspectDispatchCalls(fn *ast.FuncDecl) []dispatchCallInfo {
	var out []dispatchCallInfo
	if fn.Body == nil {
		return out
	}
	ast.Inspect(fn.Body, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok || sel.Sel.Name != "Dispatch" {
			return true
		}
		ident, ok := sel.X.(*ast.Ident)
		if !ok || ident.Name != "dispatcher" {
			return true
		}
		out = append(out, dispatchCallInfo{
			pos:       call.Pos(),
			end:       call.End(),
			stmtStart: n.Pos(),
		})
		return true
	})
	return out
}

// findStmtForPos returns the index of the top-level statement in
// body.List whose token range contains pos, or -1 when no enclosing
// statement is found.
func findStmtForPos(body *ast.BlockStmt, pos token.Pos) int {
	for i, s := range body.List {
		if pos >= s.Pos() && pos <= s.End() {
			return i
		}
	}
	return -1
}

// TestDupcodeTypedEntryPointsNoPostDispatchProtectedCalls is a fail-closed
// structural guard over the typed dispatch entry points. The test parses
// every production Go file in the gate package with go/parser, locates
// each expected entry-point declaration (FAIL on missing, duplicate, or
// nil body), counts ALL dispatcher.Dispatch calls (FAIL if not exactly
// one), and asserts that no protected call overlaps the dispatch call's
// token interval. Public wrappers must delegate exactly once.
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
		fset *token.FileSet
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
				fset: p.fset,
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
		if found[0].body == nil {
			t.Errorf("required declaration %q has nil body", name)
			return declHit{}
		}
		return found[0]
	}

	// Internal *With entry points: must own exactly one
	// dispatcher.Dispatch call (count ALL such calls, not just the
	// first). Any second dispatch call fails. Any protected call whose
	// position overlaps the unique dispatch call's token interval
	// (including the SAME statement, a return expression, a deferred
	// call, a nested closure, or a branch following the dispatch) is
	// a structural violation.
	for _, name := range typedEntryPointInternal {
		hit := checkExactlyOne(name)
		if hit.body == nil {
			continue
		}
		tuple := declHitTuple{
			file: hit.file,
			fset: hit.fset,
			pos:  hit.pos,
			body: hit.body,
		}
		calls := inspectDispatchCalls(fnFromDecl(name, tuple))
		if len(calls) == 0 {
			t.Errorf("%s: dispatcher.Dispatch call not found", name)
			continue
		}
		if len(calls) > 1 {
			t.Errorf("%s: dispatcher.Dispatch called %d times, want exactly 1", name, len(calls))
			continue
		}
		dispatchCall := calls[0]
		// Reject any protected call whose position is strictly AFTER
		// the dispatch call's end position. Argument-list calls
		// (e.g. binder.BindRunner() inside dispatcher.Dispatch(...))
		// are syntactically nested INSIDE the dispatch call, so their
		// Pos() is <= dispatch.end. Those are legitimate production
		// patterns and must not be flagged here.
		ast.Inspect(hit.body, func(n ast.Node) bool {
			if n == nil {
				return true
			}
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			// Skip the dispatch call itself.
			if call.Pos() == dispatchCall.pos {
				return true
			}
			// Calls inside the dispatch argument list have Pos()
			// strictly less than dispatch.end. We only flag calls
			// AFTER dispatch.end. Anything inside dispatch(...):
			// executed before dispatch returns.
			if call.Pos() < dispatchCall.end {
				return true
			}
			for _, bad := range protectedCallNames {
				if callContains(call, bad) {
					pos := hit.fset.Position(call.Pos())
					t.Errorf("%s: protected call %q at line %d column %d occurs after dispatch call end",
						name, bad, pos.Line, pos.Column)
				}
			}
			return true
		})
		// Additionally: require no protected call in any top-level
		// statement that begins at or after the dispatch call's
		// enclosing statement. This catches protected calls that come
		// after the dispatch even when they don't overlap the same
		// line (i.e. on a later statement).
		dispatchStmtIdx := findStmtForPos(hit.body, dispatchCall.pos)
		if dispatchStmtIdx < 0 {
			continue
		}
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
	// internal *With function and contain no direct call to a
	// protected operation anywhere in the wrapper body.
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

// declHitTuple is a serializable representation of a typed-entry-point
// declaration. It mirrors the local type in the test function but is
// hoisted to package scope so fnFromDecl can use it.
type declHitTuple struct {
	file string
	fset *token.FileSet
	pos  token.Pos
	body *ast.BlockStmt
}

// fnFromDecl reconstructs an *ast.FuncDecl from the gathered declHit
// (which holds only the body). It is used by the dispatcher-call
// inspector. The reconstruction is local to this file and never
// escapes.
func fnFromDecl(name string, hit declHitTuple) *ast.FuncDecl {
	return &ast.FuncDecl{
		Name: &ast.Ident{NamePos: hit.pos, Name: name},
		Body: hit.body,
	}
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
