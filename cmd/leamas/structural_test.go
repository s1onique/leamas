// SPDX-License-Identifier: Apache-2.0

package main

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestCmdNoProtectedVerifierImport is a real structural AST test. It parses
// every production .go file under cmd/leamas and fails if any file imports
// the protectedverifier package or otherwise references adapter symbols.
//
// The previous version of this test (in dupcode_cli_output_test.go) was a
// comment-only no-op. This test parses actual source code and inspects
// imports and selector expressions.
func TestCmdNoProtectedVerifierImport(t *testing.T) {
	prohibitedImport := "github.com/s1onique/leamas/internal/factory/protectedverifier"

	root, err := findModuleRoot()
	if err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(root, "cmd", "leamas")

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}

		path := filepath.Join(dir, name)
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		fset := token.NewFileSet()
		file, err := parser.ParseFile(fset, path, data, parser.AllErrors)
		if err != nil {
			t.Fatalf("parse %s: %v", name, err)
		}

		for _, imp := range file.Imports {
			importPath, err := unquoteImport(imp)
			if err != nil {
				t.Fatalf("%s: unquote import: %v", name, err)
			}
			if importPath == prohibitedImport {
				t.Errorf("%s: imports prohibited package %q", name, prohibitedImport)
			}
		}

		// Detect `protectedverifier.Symbol` selector references — these would
		// indicate the cmd layer is reaching into the adapter directly.
		ast.Inspect(file, func(n ast.Node) bool {
			sel, ok := n.(*ast.SelectorExpr)
			if !ok {
				return true
			}
			x, ok := sel.X.(*ast.Ident)
			if !ok {
				return true
			}
			if x.Name == "protectedverifier" {
				t.Errorf("%s: line %d: references protectedverifier.%s",
					name, fset.Position(sel.Pos()).Line, sel.Sel.Name)
			}
			return true
		})
	}
}

// TestCmdNoExecutableFactorySurface proves the cmd layer does not construct
// production RunnerFactory closures. A func literal whose body references
// protectedverifier.* is a candidate for an executable factory.
func TestCmdNoExecutableFactorySurface(t *testing.T) {
	root, err := findModuleRoot()
	if err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(root, "cmd", "leamas")

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		path := filepath.Join(dir, name)
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		fset := token.NewFileSet()
		file, err := parser.ParseFile(fset, path, data, parser.AllErrors)
		if err != nil {
			t.Fatalf("parse %s: %v", name, err)
		}
		ast.Inspect(file, func(n ast.Node) bool {
			fn, ok := n.(*ast.FuncLit)
			if !ok {
				return true
			}
			// Heuristic: a func literal whose body contains a protectedverifier
			// selector expression is a candidate for an executable factory.
			foundProtected := false
			ast.Inspect(fn, func(inner ast.Node) bool {
				sel, ok := inner.(*ast.SelectorExpr)
				if !ok {
					return true
				}
				x, ok := sel.X.(*ast.Ident)
				if !ok {
					return true
				}
				if x.Name == "protectedverifier" {
					foundProtected = true
				}
				return true
			})
			if foundProtected {
				t.Errorf("%s: func literal at line %d references protectedverifier",
					name, fset.Position(fn.Pos()).Line)
			}
			return true
		})
	}
}

// findModuleRoot walks up to find go.mod.
func findModuleRoot() (string, error) {
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
	return "", os.ErrNotExist
}

// unquoteImport unquotes an import path value.
func unquoteImport(imp *ast.ImportSpec) (string, error) {
	if imp.Path == nil {
		return "", os.ErrInvalid
	}
	v := imp.Path.Value
	if len(v) < 2 {
		return "", os.ErrInvalid
	}
	if v[0] == '"' && v[len(v)-1] == '"' {
		return v[1 : len(v)-1], nil
	}
	if v[0] == '`' && v[len(v)-1] == '`' {
		return v[1 : len(v)-1], nil
	}
	return v, nil
}
