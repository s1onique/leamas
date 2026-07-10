// Package execgate provides AST-based verification tests.
package execgate

import (
	"os"
	"path/filepath"
	"testing"
)

// TestCheckRepoExcludesAllowedFiles tests that allowed files are not flagged.
func TestCheckRepoExcludesAllowedFiles(t *testing.T) {
	root := "../../../.."
	findings := CheckRepo(root)

	// Filter to only check for forbidden exec calls
	var forbiddenFindings []string
	for _, f := range findings {
		if f.Kind == "forbidden_exec_call" {
			forbiddenFindings = append(forbiddenFindings, f.Path+": "+f.Message)
		}
	}

	// Allowed files should not appear in findings
	for _, f := range forbiddenFindings {
		if filepath.HasPrefix(f, "internal/execution/executor_unix.go") {
			t.Errorf("executor_unix.go should not have findings: %s", f)
		}
		if filepath.HasPrefix(f, "internal/execution/executor_windows.go") {
			t.Errorf("executor_windows.go should not have findings: %s", f)
		}
		if filepath.HasPrefix(f, "internal/execution/process_unix.go") {
			t.Errorf("process_unix.go should not have findings: %s", f)
		}
	}
}

// TestCheckFileDetectsForbiddenCalls tests detection of forbidden calls.
func TestCheckFileDetectsForbiddenCalls(t *testing.T) {
	// Create a temporary file with forbidden calls
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test_forbidden.go")

	content := `package test

import (
	"os/exec"
)

func test() {
	exec.Command("ls") // This should be flagged
}
`
	if err := os.WriteFile(tmpFile, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	findings := CheckFile(tmpFile)

	if len(findings) == 0 {
		t.Error("expected to find forbidden exec.Command call")
	}

	found := false
	for _, f := range findings {
		if f.Kind == "forbidden_exec_call" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected to find forbidden_exec_call kind")
	}
}

// TestCheckFileWithAlias tests detection with import aliases.
func TestCheckFileWithAlias(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test_alias.go")

	content := `package test

import osexec "os/exec"

func test() {
	osexec.Command("ls") // This should be flagged
}
`
	if err := os.WriteFile(tmpFile, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	findings := CheckFile(tmpFile)

	found := false
	for _, f := range findings {
		if f.Kind == "forbidden_exec_call" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected to find forbidden exec.Command call with alias")
	}
}

// TestCheckFileDetectsSyscall tests detection of syscall calls.
func TestCheckFileDetectsSyscall(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test_syscall.go")

	content := `package test

import "syscall"

func test() {
	syscall.Exec([]byte("/bin/ls"), nil, nil) // This should be flagged
}
`
	if err := os.WriteFile(tmpFile, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	findings := CheckFile(tmpFile)

	found := false
	for _, f := range findings {
		if f.Kind == "forbidden_exec_call" && f.Message != "" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected to find forbidden syscall call")
	}
}

// TestCheckFileDetectsStartProcess tests detection of os.StartProcess.
func TestCheckFileDetectsStartProcess(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test_startprocess.go")

	content := `package test

import "os"

func test() {
	attr := &os.ProcAttr{}
	os.StartProcess("/bin/ls", nil, attr) // This should be flagged
}
`
	if err := os.WriteFile(tmpFile, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	findings := CheckFile(tmpFile)

	found := false
	for _, f := range findings {
		if f.Kind == "forbidden_exec_call" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected to find forbidden os.StartProcess call")
	}
}

// TestCheckFileAcceptsValidCode tests that valid code doesn't trigger findings.
func TestCheckFileAcceptsValidCode(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test_valid.go")

	content := `package test

import "fmt"

func test() {
	fmt.Println("hello")
}
`
	if err := os.WriteFile(tmpFile, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	findings := CheckFile(tmpFile)

	for _, f := range findings {
		if f.Kind == "forbidden_exec_call" {
			t.Errorf("valid code should not have forbidden findings: %s", f.Message)
		}
	}
}

// TestCheckFileDetectsDotImport tests detection of dot imports.
func TestCheckFileDetectsDotImport(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test_dotimport.go")

	content := `package test

import . "os/exec"

func test() {
	Command("ls")
}
`
	if err := os.WriteFile(tmpFile, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	findings := CheckFile(tmpFile)

	// Dot imports are flagged
	found := false
	for _, f := range findings {
		if f.Kind == "dot_import_forbidden" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected to find dot import forbidden")
	}
}

// TestCheckFileHandlesParseError tests handling of parse errors.
func TestCheckFileHandlesParseError(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test_parse_error.go")

	content := `package test
func test() { // missing closing brace
`
	if err := os.WriteFile(tmpFile, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	findings := CheckFile(tmpFile)

	found := false
	for _, f := range findings {
		if f.Kind == "parse_error" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected to find parse error")
	}
}

// TestFindigsAreDeterministic tests that findings are sorted.
func TestFindigsAreDeterministic(t *testing.T) {
	root := "../../../.."

	// Run multiple times and check consistency
	findings1 := CheckRepo(root)
	findings2 := CheckRepo(root)

	if len(findings1) != len(findings2) {
		t.Fatalf("inconsistent finding count: %d vs %d", len(findings1), len(findings2))
	}

	for i := range findings1 {
		if findings1[i].Path != findings2[i].Path {
			t.Errorf("path mismatch at index %d: %s vs %s", i, findings1[i].Path, findings2[i].Path)
		}
	}
}

// TestCheckFileDetectsLookPath tests detection of exec.LookPath.
func TestCheckFileDetectsLookPath(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test_lookpath.go")

	content := `package test

import "os/exec"

func test() {
	exec.LookPath("ls") // This should be flagged
}
`
	if err := os.WriteFile(tmpFile, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	findings := CheckFile(tmpFile)

	found := false
	for _, f := range findings {
		if f.Kind == "forbidden_exec_call" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected to find forbidden exec.LookPath call")
	}
}
