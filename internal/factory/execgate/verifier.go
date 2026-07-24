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
	"sort"
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

// AllowedFiles are specific files where direct exec calls are permitted.
// Only the execution gateway itself and factory infrastructure require process APIs.
//
// The adversarial harness files are test-only. CORRECTION05 renamed
// adversarial_harness_executor.go to _test.go so it no longer appears
// in the production binary. The entry here keeps the allow-list bound
// to the renamed path so the renamed file's exec.Command("go", "build", ...)
// call is still recognised by the gate as a legitimate test build-step.
var AllowedFiles = map[string]bool{
	"internal/execution/executor.go":                          true,
	"internal/execution/git.go":                               true,
	"internal/execution/testlong.go":                          true,
	"internal/execution/exectest/outcome.go":                  true,
	"internal/execution/exectest/bounded_output.go":           true,
	"internal/execution/exectest/environment.go":              true,
	"internal/execution/exectest/run_make.go":                 true,
	"internal/execution/exectest/request.go":                  true,
	"internal/execution/adversarial_harness_test.go":          true,
	"internal/execution/adversarial_harness_executor_test.go": true,
	"internal/execution/adversarial_helper_build_test.go":     true,
	"internal/execution/retained_pipe_raw_test.go":            true,
	"internal/factory/gate/gate.go":                           true,
	"internal/factory/gate/gate_failure_output_test.go":       true,
	"internal/factory/gate/gate_failure_execution_test.go":    true,
	"internal/factory/gate/toolchain.go":                      true,
	"internal/factory/gate/subject_identity.go":               true,
	"internal/factory/gate/platform_sampler.go":               true,
	"internal/factory/digest/git.go":                          true,
	"internal/factory/digest/digest_auto_test.go":             true,
	"internal/factory/digest/auto_range.go":                   true,
	"internal/factory/digest/auto_range_git.go":               true,
	"internal/factory/authority/authority_test.go":            true,
	"internal/factory/authority/authority_extra_test.go":      true,
	"internal/factory/authority/bootstrap.go":                 true,
	"internal/factory/authority/checker.go":                   true,
	"internal/factory/dupcode/baseline_verify.go":             true,
	"internal/factory/githooks/check.go":                      true,
	"internal/factory/githooks/check_test.go":                 true,
	"internal/factory/githooks/hook_functional_test.go":       true,
	"internal/factory/github/check.go":                        true,
	"internal/factory/llmfriendly/check.go":                   true,
	"internal/factory/output/outputcontract.go":               true,
	"internal/factory/staticbinary/check.go":                  true,
	"internal/factory/doctrinecompiler/subprocess_test.go":    true,
	"cmd/leamas/runtime_smoke_test.go":                        true,
	"cmd/leamas/version_cli_test.go":                          true,
	"cmd/leamas/gate_summary_schema_subprocess_test.go":       true,
	"cmd/leamas/factory_close_subprocess_test.go":             true,
}

// AllowedImports are packages that may only be imported by test files.
// These packages wrap os/exec and should not be used in production code.
var AllowedImports = map[string]bool{
	"github.com/s1onique/leamas/internal/execution/exectest": true,
}

// CheckRepo scans the repository for forbidden exec patterns.
func CheckRepo(root string) []checks.Finding {
	var findings []checks.Finding

	// Scan cmd/ and internal/
	dirs := []string{"cmd", "internal"}

	for _, dir := range dirs {
		scanPath := filepath.Join(root, dir)
		if _, err := os.Stat(scanPath); os.IsNotExist(err) {
			continue
		}

		err := filepath.WalkDir(scanPath, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}

			if d.IsDir() {
				name := d.Name()
				if name == "testdata" || name == "vendor" || name == ".git" {
					return filepath.SkipDir
				}
				return nil
			}

			// Only check .go files
			if !strings.HasSuffix(path, ".go") {
				return nil
			}

			relPath, _ := filepath.Rel(root, path)

			// Skip only exact allowed files
			if AllowedFiles[filepath.ToSlash(relPath)] {
				return nil
			}

			findings = append(findings, checkFile(relPath, path)...)
			return nil
		})
		if err != nil {
			continue
		}
	}

	sort.Slice(findings, func(i, j int) bool {
		if findings[i].Path != findings[j].Path {
			return findings[i].Path < findings[j].Path
		}
		return findings[i].Message < findings[j].Message
	})

	return findings
}

// checkFile checks a single file for forbidden exec calls.
func checkFile(relPath, path string) []checks.Finding {
	var findings []checks.Finding

	// Determine if this is a test file
	isTestFile := strings.HasSuffix(path, "_test.go")

	data, err := os.ReadFile(path)
	if err != nil {
		findings = append(findings, checks.Finding{
			Path:     relPath,
			Kind:     "file_read_error",
			Message:  fmt.Sprintf("failed to read file: %v", err),
			Severity: checks.SeverityError,
		})
		return findings
	}

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, path, data, parser.AllErrors)
	if err != nil {
		findings = append(findings, checks.Finding{
			Path:     relPath,
			Kind:     "parse_error",
			Message:  fmt.Sprintf("failed to parse file: %v", err),
			Severity: checks.SeverityError,
		})
		return findings
	}

	// Build import alias map
	imports := buildImportMap(file)

	ast.Inspect(file, func(n ast.Node) bool {
		switch node := n.(type) {
		case *ast.CallExpr:
			findings = append(findings, checkCallExpr(relPath, fset, node, imports)...)
		case *ast.ImportSpec:
			// Check dot imports
			if node.Path != nil && node.Name != nil && node.Name.Name == "." {
				findings = append(findings, checks.Finding{
					Path:     relPath,
					Kind:     "dot_import_forbidden",
					Message:  fmt.Sprintf("dot imports are forbidden in this package"),
					Severity: checks.SeverityError,
				})
			}

			// Check for forbidden imports (exectest may only be imported by test files)
			if node.Path != nil {
				importPath := strings.Trim(node.Path.Value, `"`)
				if isRestrictedImport(importPath) && !isTestFile {
					findings = append(findings, checks.Finding{
						Path:     relPath,
						Kind:     "forbidden_import",
						Message:  fmt.Sprintf("forbidden import: %s may only be imported by _test.go files", importPath),
						Severity: checks.SeverityError,
					})
				}
			}
		}
		return true
	})

	return findings
}

// buildImportMap builds a map from import aliases to package paths.
func buildImportMap(file *ast.File) map[string]string {
	imports := make(map[string]string)

	// Add standard aliases
	imports["exec"] = "os/exec"
	imports["osexec"] = "os/exec"
	imports["os"] = "os"
	imports["syscall"] = "syscall"

	for _, spec := range file.Imports {
		if spec.Path == nil {
			continue
		}
		path := strings.Trim(spec.Path.Value, `"`)

		var alias string
		if spec.Name != nil {
			alias = spec.Name.Name
		} else {
			// Derive alias from path
			parts := strings.Split(path, "/")
			alias = parts[len(parts)-1]
		}

		imports[alias] = path
	}

	return imports
}

// checkCallExpr checks a call expression for forbidden exec calls.
func checkCallExpr(path string, fset *token.FileSet, node *ast.CallExpr, imports map[string]string) []checks.Finding {
	var findings []checks.Finding

	// Handle selector expressions: pkg.Function()
	if sel, ok := node.Fun.(*ast.SelectorExpr); ok {
		if ident, ok := sel.X.(*ast.Ident); ok {
			pkg := ident.Name
			fn := sel.Sel.Name

			// Resolve alias to package path
			if resolved, ok := imports[pkg]; ok {
				pkg = resolved
			}

			// Check for forbidden patterns
			if isForbiddenCall(pkg, fn) {
				findings = append(findings, checks.Finding{
					Path:     path,
					Kind:     "forbidden_exec_call",
					Message:  fmt.Sprintf("forbidden: %s.%s - %s", pkg, fn, forbiddenDesc(pkg, fn)),
					Severity: checks.SeverityError,
				})
			}
		}
	}

	// Handle direct function calls where function is assigned from package
	// e.g., cmd := exec.Command; cmd(...)
	if ident, ok := node.Fun.(*ast.Ident); ok {
		fnName := ident.Name
		if isForbiddenFunction(fnName) {
			findings = append(findings, checks.Finding{
				Path:     path,
				Kind:     "forbidden_exec_call",
				Message:  fmt.Sprintf("forbidden: %s() - direct exec function usage", fnName),
				Severity: checks.SeverityError,
			})
		}
	}

	return findings
}

// isForbiddenCall checks if a package.function combination is forbidden.
func isForbiddenCall(pkg, fn string) bool {
	switch pkg {
	case "os/exec":
		return fn == "Command" || fn == "CommandContext" || fn == "LookPath"
	case "os":
		return fn == "StartProcess"
	case "syscall":
		return fn == "ForkExec" || fn == "Exec"
	}
	return false
}

// isForbiddenFunction checks if a bare function name is forbidden.
func isForbiddenFunction(fn string) bool {
	switch fn {
	case "Command", "CommandContext", "LookPath":
		return true
	}
	return false
}

// forbiddenDesc returns the description for a forbidden call.
func forbiddenDesc(pkg, fn string) string {
	switch pkg {
	case "os/exec":
		return "exec.Command/CommandContext must not be called outside internal/execution"
	case "os":
		return "os.StartProcess must not be called outside internal/execution"
	case "syscall":
		return "syscall.ForkExec/Exec must not be called outside internal/execution"
	}
	return "forbidden execution call"
}

// isRestrictedImport returns true if the import path is restricted to test files only.
func isRestrictedImport(importPath string) bool {
	return AllowedImports[importPath]
}

// CheckFile scans a single file for forbidden exec patterns.
func CheckFile(path string) []checks.Finding {
	relPath := path
	if strings.HasPrefix(path, "./") {
		relPath = strings.TrimPrefix(path, "./")
	}
	return checkFile(relPath, path)
}
