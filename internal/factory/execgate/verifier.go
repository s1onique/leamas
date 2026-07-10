// Package execgate provides AST-based verification that all process execution
// flows through the execution gateway.
package execgate

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"

	"github.com/s1onique/leamas/internal/factory/checks"
)

// ForbiddenCall represents a forbidden exec call pattern.
type ForbiddenCall struct {
	Package  string
	Function string
	Desc     string
}

// ForbiddenCalls lists all forbidden exec calls.
var ForbiddenCalls = []ForbiddenCall{
	{"os/exec", "Command", "exec.Command must not be called outside internal/execution"},
	{"os/exec", "CommandContext", "exec.CommandContext must not be called outside internal/execution"},
	{"os", "StartProcess", "os.StartProcess must not be called outside internal/execution"},
	{"syscall", "ForkExec", "syscall.ForkExec must not be called outside internal/execution"},
	{"syscall", "Exec", "syscall.Exec must not be called outside internal/execution"},
}

// AllowedDirs are directories where direct exec calls are permitted.
var AllowedDirs = []string{
	"internal/execution",
	"internal/execution/adapters",
	"internal/factory", // Factory verifiers are tooling infrastructure
}

// CheckRepo scans the repository for forbidden exec patterns.
func CheckRepo(root string) []checks.Finding {
	var findings []checks.Finding

	// Scan cmd/ and internal/ (except execution gateway)
	dirs := []string{"cmd", "internal"}

	for _, dir := range dirs {
		scanPath := filepath.Join(root, dir)
		if _, err := os.Stat(scanPath); os.IsNotExist(err) {
			continue
		}

		err := filepath.WalkDir(scanPath, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return nil
			}

			if d.IsDir() {
				// Skip testdata and vendor
				name := d.Name()
				if name == "testdata" || name == "vendor" || name == ".git" {
					return filepath.SkipDir
				}
				// Skip execution package - it's allowed to use exec
				if strings.HasPrefix(path, filepath.Join(root, "internal/execution")) {
					return nil
				}
				return nil
			}

			// Only check .go files
			if !strings.HasSuffix(path, ".go") {
				return nil
			}

			// Skip test files
			if strings.HasSuffix(path, "_test.go") {
				return nil
			}

			findings = append(findings, checkFile(path)...)
			return nil
		})
		if err != nil {
			continue
		}
	}

	checks.SortFindings(findings)
	return findings
}

// checkFile checks a single file for forbidden exec calls.
func checkFile(path string) []checks.Finding {
	var findings []checks.Finding

	data, err := os.ReadFile(path)
	if err != nil {
		return findings
	}

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, path, data, parser.AllErrors)
	if err != nil {
		return findings
	}

	relPath, _ := filepath.Rel(".", path)

	ast.Inspect(file, func(n ast.Node) bool {
		switch node := n.(type) {
		case *ast.CallExpr:
			findings = append(findings, checkCallExpr(fset, relPath, node)...)
		}
		return true
	})

	return findings
}

// checkCallExpr checks a call expression for forbidden exec calls.
func checkCallExpr(fset *token.FileSet, path string, node *ast.CallExpr) []checks.Finding {
	var findings []checks.Finding

	// Check for exec.Command and exec.CommandContext
	if sel, ok := node.Fun.(*ast.SelectorExpr); ok {
		if ident, ok := sel.X.(*ast.Ident); ok {
			pkg := ident.Name
			fn := sel.Sel.Name

			if pkg == "exec" && (fn == "Command" || fn == "CommandContext") {
				// Check if we're in internal/execution
				if !isInAllowedDir(path) {
					findings = append(findings, checks.Finding{
						Path:     path,
						Kind:     "forbidden_exec_call",
						Message:  fmt.Sprintf("exec.%s must not be called outside internal/execution", fn),
						Severity: checks.SeverityError,
					})
				}
			}
		}
	}

	return findings
}

// isInAllowedDir checks if a path is in an allowed directory.
func isInAllowedDir(path string) bool {
	for _, dir := range AllowedDirs {
		if strings.HasPrefix(path, dir) || strings.HasPrefix(path, "./"+dir) {
			return true
		}
	}
	return false
}

// CheckFile scans a single file for forbidden exec patterns.
func CheckFile(path string) []checks.Finding {
	return checkFile(path)
}
