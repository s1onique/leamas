package language

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/s1onique/leamas/internal/factory/checks"
)

func TestCheckProductionDirs(t *testing.T) {
	tmpDir := t.TempDir()

	// Create internal directory with a non-Go file
	internalDir := filepath.Join(tmpDir, "internal")
	if err := os.MkdirAll(internalDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create a Python file in internal (forbidden)
	pyFile := filepath.Join(internalDir, "script.py")
	if err := os.WriteFile(pyFile, []byte("#!/usr/bin/env python3\nprint('hello')"), 0644); err != nil {
		t.Fatal(err)
	}

	// Create a Go file in internal (allowed)
	goFile := filepath.Join(internalDir, "main.go")
	if err := os.WriteFile(goFile, []byte("package main\nfunc main() {}"), 0644); err != nil {
		t.Fatal(err)
	}

	findings := checkProductionDirs(tmpDir)
	if len(findings) == 0 {
		t.Error("expected findings for Python file in internal/")
	}

	// Check it found the .py file
	found := false
	for _, f := range findings {
		if filepath.Ext(f.Path) == ".py" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected to find .py file in findings")
	}
}

func TestCheckForbiddenNodeFiles(t *testing.T) {
	tmpDir := t.TempDir()

	// Create package.json
	pkgFile := filepath.Join(tmpDir, "package.json")
	if err := os.WriteFile(pkgFile, []byte(`{"name": "test"}`), 0644); err != nil {
		t.Fatal(err)
	}

	findings := checkForbiddenNodeFiles(tmpDir)
	if len(findings) == 0 {
		t.Error("expected findings for package.json")
	}
}

func TestCheckShellScriptLocations(t *testing.T) {
	tmpDir := t.TempDir()

	// Create shell script in allowed directory
	scriptsDir := filepath.Join(tmpDir, "scripts")
	if err := os.MkdirAll(scriptsDir, 0755); err != nil {
		t.Fatal(err)
	}

	allowedScript := filepath.Join(scriptsDir, "build.sh")
	if err := os.WriteFile(allowedScript, []byte("#!/bin/bash\necho hello"), 0644); err != nil {
		t.Fatal(err)
	}

	// Create shell script in disallowed directory
	internalDir := filepath.Join(tmpDir, "internal")
	if err := os.MkdirAll(internalDir, 0755); err != nil {
		t.Fatal(err)
	}

	disallowedScript := filepath.Join(internalDir, "setup.sh")
	if err := os.WriteFile(disallowedScript, []byte("#!/bin/bash\necho hello"), 0644); err != nil {
		t.Fatal(err)
	}

	findings := checkShellScriptLocations(tmpDir)
	if len(findings) == 0 {
		t.Error("expected findings for shell script outside allowed directories")
	}
}

func TestCheckPythonFiles(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a Python file
	pyFile := filepath.Join(tmpDir, "script.py")
	if err := os.WriteFile(pyFile, []byte("#!/usr/bin/env python3\nprint('hello')"), 0644); err != nil {
		t.Fatal(err)
	}

	findings := CheckPythonFiles(tmpDir)
	if len(findings) == 0 {
		t.Error("expected findings for Python file")
	}
}

func TestCheckRepo(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a clean project structure
	internalDir := filepath.Join(tmpDir, "internal")
	if err := os.MkdirAll(internalDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create a Go file
	goFile := filepath.Join(internalDir, "main.go")
	if err := os.WriteFile(goFile, []byte("package main\nfunc main() {}"), 0644); err != nil {
		t.Fatal(err)
	}

	// Create scripts directory with shell script
	scriptsDir := filepath.Join(tmpDir, "scripts")
	if err := os.MkdirAll(scriptsDir, 0755); err != nil {
		t.Fatal(err)
	}

	scriptFile := filepath.Join(scriptsDir, "build.sh")
	if err := os.WriteFile(scriptFile, []byte("#!/bin/bash\necho hello"), 0644); err != nil {
		t.Fatal(err)
	}

	findings := CheckRepo(tmpDir)
	// Should have no findings for clean code
	foundErrors := false
	for _, f := range findings {
		if f.Severity == checks.SeverityError {
			foundErrors = true
			break
		}
	}
	if foundErrors {
		t.Errorf("expected no error findings for clean code, got %d", len(findings))
	}
}
