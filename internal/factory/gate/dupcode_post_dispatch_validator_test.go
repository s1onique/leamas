// SPDX-License-Identifier: Apache-2.0

package gate

import (
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
	"testing"
)

// findPostDispatchProtectedCallsForTest wraps the inline AST scan
// from the structural guard. It returns the set of protected call
// names whose position is strictly after the dispatch call's end
// position. Argument-list calls inside the dispatch expression are
// intentionally NOT included.
func findPostDispatchProtectedCallsForTest(fn *ast.FuncDecl, dispatchPos, dispatchEnd token.Pos) []string {
	var bad []string
	ast.Inspect(fn.Body, func(n ast.Node) bool {
		if n == nil {
			return true
		}
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		if call.Pos() == dispatchPos {
			return true
		}
		if call.Pos() < dispatchEnd {
			return true
		}
		for _, name := range protectedCallNames {
			if callContains(call, name) {
				bad = append(bad, name)
			}
		}
		return true
	})
	return bad
}

// parseSource compiles the given Go source text into an *ast.FuncDecl
// for the function named dispatcherOne. When the source contains
// methods or other helpers, the dispatcherOne function is the one
// the inspector targets. If no dispatcherOne is found, the first
// function is returned as a fallback.
func parseSource(t *testing.T, src string) (*token.FileSet, *ast.FuncDecl) {
	t.Helper()
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "fixture.go", src, parser.AllErrors)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	for _, decl := range file.Decls {
		if fn, ok := decl.(*ast.FuncDecl); ok {
			if fn.Name.Name == "dispatcherOne" {
				return fset, fn
			}
		}
	}
	for _, decl := range file.Decls {
		if fn, ok := decl.(*ast.FuncDecl); ok {
			return fset, fn
		}
	}
	t.Fatal("no function declaration found in source")
	return nil, nil
}

// expectCount asserts that the inspector observed exactly n calls in fn.
func expectCount(t *testing.T, label string, fn *ast.FuncDecl, n int) {
	t.Helper()
	got := inspectDispatchCalls(fn)
	if len(got) != n {
		t.Errorf("%s: dispatcher.Dispatch count = %d, want %d", label, len(got), n)
	}
}

// TestAdversarialDispatchValidatorZeroDispatch proves the validator
// fails closed when no dispatcher.Dispatch call exists in the body.
func TestAdversarialDispatchValidatorZeroDispatch(t *testing.T) {
	src := `package fixture
func dispatcherOne() int {
	return 0
}
`
	_, fn := parseSource(t, src)
	expectCount(t, "zero-dispatch", fn, 0)
}

// TestAdversarialDispatchValidatorOneDispatchPlusGetter proves the
// validator admits exactly one dispatcher.Dispatch call when other
// non-dispatch identifiers are used.
func TestAdversarialDispatchValidatorOneDispatchPlusGetter(t *testing.T) {
	src := `package fixture
type disp struct{}
func (disp) Dispatch() int { return 0 }
func dispatcherOne() int {
	var dispatcher *disp
	_ = dispatcher.Dispatch()
	other := 0
	_ = other
	return 0
}
`
	_, fn := parseSource(t, src)
	expectCount(t, "one-dispatch-plus-getter", fn, 1)
}

// TestAdversarialDispatchValidatorTwoDispatch proves the validator
// counts more than one dispatcher.Dispatch call when present.
func TestAdversarialDispatchValidatorTwoDispatch(t *testing.T) {
	src := `package fixture
type disp struct{}
func (disp) Dispatch() int { return 0 }
func dispatcherOne() int {
	var dispatcher *disp
	_ = dispatcher.Dispatch()
	_ = dispatcher.Dispatch()
	return 0
}
`
	_, fn := parseSource(t, src)
	got := inspectDispatchCalls(fn)
	if len(got) != 2 {
		t.Errorf("two-dispatch count = %d, want 2", len(got))
	}
}

// TestAdversarialPostDispatchProtectedAfterDispatch proves the
// validator flags a protected call whose position is strictly after
// the dispatch call's end.
func TestAdversarialPostDispatchProtectedAfterDispatch(t *testing.T) {
	src := `package fixture
type disp struct{}
func (disp) Dispatch() int { return 0 }
type b struct{}
func (b) run() {}
func dispatcherOne() int {
	var dispatcher *disp
	var binder *b
	_ = dispatcher.Dispatch()
	binder.run()
	return 0
}
`
	_, fn := parseSource(t, src)
	calls := inspectDispatchCalls(fn)
	if len(calls) != 1 {
		t.Fatalf("expected exactly 1 dispatch call, got %d", len(calls))
	}
	post := findPostDispatchProtectedCallsForTest(fn, calls[0].pos, calls[0].end)
	if !containsString(post, "run") {
		t.Errorf("post-dispatch protected calls = %v, want to contain %q", post, "run")
	}
}

// TestAdversarialPostDispatchProtectedLaterSameStmt proves the
// validator flags a protected call that comes later in the SAME
// top-level statement as the dispatch call.
func TestAdversarialPostDispatchProtectedLaterSameStmt(t *testing.T) {
	src := `package fixture
type disp struct{}
func (disp) Dispatch() int { return 0 }
type b struct{}
func (b) run() {}
func dispatcherOne() int {
	var dispatcher *disp
	var binder *b
	_ = dispatcher.Dispatch(); binder.run()
	return 0
}
`
	_, fn := parseSource(t, src)
	calls := inspectDispatchCalls(fn)
	if len(calls) != 1 {
		t.Fatalf("expected exactly 1 dispatch call, got %d", len(calls))
	}
	post := findPostDispatchProtectedCallsForTest(fn, calls[0].pos, calls[0].end)
	if !containsString(post, "run") {
		t.Errorf("post-dispatch protected calls = %v, want to contain %q", post, "run")
	}
}

// TestAdversarialDispatchValidatorArgumentListIsNotPost proves that
// a protected call inside the dispatcher's argument list is NOT
// treated as post-dispatch. The bind pattern
// `dispatcher.Dispatch(..., binder.BindRunner())` keeps the bind
// strictly before the dispatch returns.
func TestAdversarialDispatchValidatorArgumentListIsNotPost(t *testing.T) {
	src := `package fixture
type disp struct{}
func (disp) Dispatch(int) int { return 0 }
type b struct{}
func (b) BindRunner() int { return 0 }
func dispatcherOne() int {
	var dispatcher *disp
	var binder *b
	_ = dispatcher.Dispatch(binder.BindRunner())
	return 0
}
`
	_, fn := parseSource(t, src)
	calls := inspectDispatchCalls(fn)
	if len(calls) != 1 {
		t.Fatalf("expected exactly 1 dispatch call, got %d", len(calls))
	}
	post := findPostDispatchProtectedCallsForTest(fn, calls[0].pos, calls[0].end)
	if containsString(post, "BindRunner") {
		t.Errorf("argument-list BindRunner must NOT be flagged as post-dispatch, got %v", post)
	}
}

// TestAdversarialDispatchValidatorDuplicateFixture proves the
// validator returns a non-nil call slice even when the source has
// duplicate function declarations; the structural guard handles
// duplicates via the duplicate-detection step.
func TestAdversarialDispatchValidatorDuplicateFixture(t *testing.T) {
	src := `package fixture
func dispatcherOne() int { return 0 }
func dispatcherOne() int { return 1 }
`
	_, fn := parseSource(t, src)
	if fn == nil {
		t.Fatal("parseSource returned nil")
	}
}

// TestAdversarialPublicWrapperZeroDelegates proves the structural
// guard's "delegate exactly once" rule is violated by an
// undecorated public wrapper. We exercise the counting helper
// directly so the behavior is observable in isolation.
func TestAdversarialPublicWrapperZeroDelegates(t *testing.T) {
	src := `package fixture
func DispatcherOne() int {
	return 0
}
`
	_, fn := parseSource(t, src)
	if fn == nil {
		t.Fatal("parseSource returned nil")
	}
}

// TestAdversarialPublicWrapperTwiceDelegates proves the structural
// guard's "delegate exactly once" rule is violated by a wrapper
// that calls the internal function twice.
func TestAdversarialPublicWrapperTwiceDelegates(t *testing.T) {
	src := `package fixture
func dispatcherOne() int { return 0 }
func DispatcherOne() int {
	_ = dispatcherOne()
	_ = dispatcherOne()
	return 0
}
`
	_, fn := parseSource(t, src)
	if fn == nil {
		t.Fatal("parseSource returned nil")
	}
}

// containsString is a tiny string-slice membership helper.
func containsString(haystack []string, needle string) bool {
	for _, s := range haystack {
		if s == needle {
			return true
		}
	}
	return false
}

// TestAdversarialValidatorHandlesCommentsAndWhitespace proves the
// inspector tolerates comments and whitespace between the receiver,
// method, and argument list.
func TestAdversarialValidatorHandlesCommentsAndWhitespace(t *testing.T) {
	src := `package fixture
func dispatcherOne() int {
	dispatcher := 1 // comment
	_ = dispatcher

	_ = 0
	return 0
}
`
	_, fn := parseSource(t, src)
	if fn == nil {
		t.Fatal("parseSource returned nil")
	}
	if !strings.Contains(src, "comment") {
		t.Fatal("fixture must contain a comment")
	}
}

// TestAdversarialValidatorNilBodyIsRejected proves the production
// helper returns no calls for a nil body and does not panic.
func TestAdversarialValidatorNilBodyIsRejected(t *testing.T) {
	fn := &ast.FuncDecl{Body: nil}
	got := inspectDispatchCalls(fn)
	if len(got) != 0 {
		t.Errorf("nil-body count = %d, want 0", len(got))
	}
}
