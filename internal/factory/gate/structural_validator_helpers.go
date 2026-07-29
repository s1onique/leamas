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
)

// dispatchCallInfo records a single dispatcher.Dispatch call's token
// interval and its direct argument expressions. It is the validator's
// internal representation.
type dispatchCallInfo struct {
	pos  token.Pos
	end  token.Pos
	args []ast.Expr
}

// dispatchCallsFor returns every dispatcher.Dispatch call in fn that
// uses the bare identifier "dispatcher" as the receiver.
func dispatchCallsFor(fn *ast.FuncDecl) []dispatchCallInfo {
	var out []dispatchCallInfo
	if fn == nil || fn.Body == nil {
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
			pos:  call.Pos(),
			end:  call.End(),
			args: call.Args,
		})
		return true
	})
	return out
}

// callName returns the canonical name used for protected-call and
// allowed-call matching: the last selector segment for SelectorExpr
// calls, the bare identifier for plain function calls, or "" when the
// call is on something we do not match (e.g. method expressions).
func callName(call *ast.CallExpr) string {
	switch f := call.Fun.(type) {
	case *ast.Ident:
		return f.Name
	case *ast.SelectorExpr:
		return f.Sel.Name
	}
	return ""
}

// callReceiverName returns the receiver identifier of a selector
// call expression (e.g. `binder` for `binder.BindRunner()`) or "" for
// a bare identifier call. It is used by the validator to confirm the
// receiver identity against AllowedReceivers.
func callReceiverName(call *ast.CallExpr) string {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return ""
	}
	ident, ok := sel.X.(*ast.Ident)
	if !ok {
		return ""
	}
	return ident.Name
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

// parseGatePackageFiles parses every production .go file under
// internal/factory/gate and returns them as a name-indexed map. The
// validator runs against this map. It is unexported because the
// gate package's own structural-validation tests are the only
// consumer; production callers have no reason to expose the AST.
func parseGatePackageFiles() (map[string]*ast.File, *token.FileSet, error) {
	root, err := findModuleRootForGateTest()
	if err != nil {
		return nil, nil, err
	}
	srcDir := filepath.Join(root, "internal", "factory", "gate")
	files := map[string]*ast.File{}
	fset := token.NewFileSet()
	entries, err := osReadDir(srcDir)
	if err != nil {
		return nil, nil, err
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".go") || strings.HasSuffix(e.Name(), "_test.go") {
			continue
		}
		path := filepath.Join(srcDir, e.Name())
		data, err := osReadFile(path)
		if err != nil {
			return nil, nil, err
		}
		f, err := parser.ParseFile(fset, path, data, parser.AllErrors)
		if err != nil {
			return nil, nil, err
		}
		files[e.Name()] = f
	}
	return files, fset, nil
}

// findModuleRootForGateTest walks upward from the current working
// directory until it finds go.mod. The structural validator uses this
// to locate internal/factory/gate during tests.
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
