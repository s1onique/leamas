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

// ensure ast import is referenced (compiler-only anchor). The helpers
// package keeps the ast import even though parseGatePackageFiles uses
// parser.ParseFile directly so callers can rely on *ast.File in
// their returned map.
var _ = ast.File{}
