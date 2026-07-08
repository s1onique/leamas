package staticbinary

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCheckStaticBinaryIntent(t *testing.T) {
	tmpDir := t.TempDir()

	// Test with missing go.mod
	findings := CheckStaticBinaryIntent(tmpDir)
	foundGoModError := false
	for _, f := range findings {
		if f.Path == "go.mod" && f.Kind == "missing" {
			foundGoModError = true
			break
		}
	}
	if !foundGoModError {
		t.Error("expected missing go.mod error")
	}

	// Create go.mod
	goModPath := filepath.Join(tmpDir, "go.mod")
	if err := os.WriteFile(goModPath, []byte("module test\ngo 1.21"), 0644); err != nil {
		t.Fatal(err)
	}

	// Create cmd/leamas/main.go
	mainDir := filepath.Join(tmpDir, "cmd", "leamas")
	if err := os.MkdirAll(mainDir, 0755); err != nil {
		t.Fatal(err)
	}
	mainPath := filepath.Join(mainDir, "main.go")
	if err := os.WriteFile(mainPath, []byte("package main\nfunc main() {}"), 0644); err != nil {
		t.Fatal(err)
	}

	// Now should not have go.mod error
	findings = CheckStaticBinaryIntent(tmpDir)
	for _, f := range findings {
		if f.Path == "go.mod" && f.Kind == "missing" {
			t.Error("should not have go.mod error after creating it")
		}
	}
}

func TestCheckRepo(t *testing.T) {
	// Just verify it runs
	findings := CheckRepo(".")
	if findings == nil {
		t.Error("CheckRepo should return findings, not nil")
	}
}
