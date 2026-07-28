// SPDX-License-Identifier: Apache-2.0

package forbidden

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDupcodeBypassPolicy_RejectsDotImport(t *testing.T) {
	// Create a temp file with dot import
	dir := t.TempDir()
	fixture := filepath.Join(dir, "dot_import.go")
	content := `package main

import . "github.com/s1onique/leamas/internal/factory/dupcode"

func main() {}
`
	if err := os.WriteFile(fixture, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write fixture: %v", err)
	}

	policy := NewDupcodeBypassPolicy()
	err := policy.CheckFile(fixture)
	if err == nil {
		t.Error("expected error for dot import")
	}
}

func TestDupcodeBypassPolicy_RejectsAliasedImport(t *testing.T) {
	// Create a temp file with alias import to protected package
	dir := t.TempDir()
	fixture := filepath.Join(dir, "aliased_import.go")
	content := `package unauthorized

import dc "github.com/s1onique/leamas/internal/factory/dupcode"

func CheckSomething() {
	_ = dc.CheckRepo
}
`
	if err := os.WriteFile(fixture, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write fixture: %v", err)
	}

	policy := NewDupcodeBypassPolicy()
	err := policy.CheckFile(fixture)
	if err == nil {
		t.Error("expected error for aliased import from unauthorized package")
	}
}

func TestDupcodeBypassPolicy_RejectsDirectCall(t *testing.T) {
	// Create a temp file with direct call from unauthorized package
	dir := t.TempDir()
	fixture := filepath.Join(dir, "direct_call.go")
	content := `package unauthorized

import "github.com/s1onique/leamas/internal/factory/dupcode"

func RunCheck() {
	dupcode.CheckRepo(".")
}
`
	if err := os.WriteFile(fixture, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write fixture: %v", err)
	}

	policy := NewDupcodeBypassPolicy()
	err := policy.CheckFile(fixture)
	if err == nil {
		t.Error("expected error for direct call from unauthorized package")
	}
}

func TestDupcodeBypassPolicy_AllowsCanonicalAdapter(t *testing.T) {
	// Create a temp file in the canonical adapter directory structure
	dir := t.TempDir()
	// Create the proper directory structure: internal/factory/protectedverifier/
	adapterDir := filepath.Join(dir, "internal", "factory", "protectedverifier")
	if err := os.MkdirAll(adapterDir, 0755); err != nil {
		t.Fatalf("failed to create adapter directory: %v", err)
	}

	fixture := filepath.Join(adapterDir, "adapter.go")
	content := `package protectedverifier

import "github.com/s1onique/leamas/internal/factory/dupcode"

func RunCheck(root string) {
	dupcode.CheckRepo(root)
}
`
	if err := os.WriteFile(fixture, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write fixture: %v", err)
	}

	// Change to temp dir so relative paths work
	oldCwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get cwd: %v", err)
	}
	defer os.Chdir(oldCwd)

	if err := os.Chdir(dir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}

	policy := NewDupcodeBypassPolicy()
	err = policy.CheckFile(fixture)
	if err != nil {
		t.Errorf("unexpected error for canonical adapter: %v", err)
	}
}

func TestDupcodeBypassPolicy_AllowsDupcodePackage(t *testing.T) {
	// Create a temp file in the dupcode directory structure
	dir := t.TempDir()
	// Create the proper directory structure: internal/factory/dupcode/
	dupcodeDir := filepath.Join(dir, "internal", "factory", "dupcode")
	if err := os.MkdirAll(dupcodeDir, 0755); err != nil {
		t.Fatalf("failed to create dupcode directory: %v", err)
	}

	fixture := filepath.Join(dupcodeDir, "internal.go")
	content := `package dupcode

import "os"

func CheckRepo(root string) error {
	return nil
}

func WriteBaseline(path string, report Report) error {
	return nil
}
`
	if err := os.WriteFile(fixture, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write fixture: %v", err)
	}

	// Change to temp dir so relative paths work
	oldCwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get cwd: %v", err)
	}
	defer os.Chdir(oldCwd)

	if err := os.Chdir(dir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}

	policy := NewDupcodeBypassPolicy()
	err = policy.CheckFile(fixture)
	if err != nil {
		t.Errorf("unexpected error for dupcode package: %v", err)
	}
}

func TestDupcodeBypassPolicy_CurrentRepository(t *testing.T) {
	// Check the current repository for bypasses using CheckDupcodeBypass
	// This tests the fail-closed repository-wide scan
	findings := CheckDupcodeBypass(".")

	// Filter out findings that are expected (allowed packages)
	var unexpectedFindings []string
	for _, f := range findings {
		// Skip allowed paths - gate package is allowed
		if f.Kind == "dupcode_bypass_walk_error" || f.Kind == "dupcode_bypass_scan_error" {
			// These might be expected if certain directories don't exist
			continue
		}
		// Check if the finding is from an allowed directory
		if f.Kind == "dupcode_bypass" {
			// Report unexpected bypass findings
			unexpectedFindings = append(unexpectedFindings, f.Path+": "+f.Message)
		}
	}

	if len(unexpectedFindings) > 0 {
		t.Errorf("unexpected bypass findings:\n%s", unexpectedFindings)
	}
}

func TestDupcodeBypassPolicy_RejectsSelectorCall(t *testing.T) {
	// Create a temp file with selector call
	dir := t.TempDir()
	fixture := filepath.Join(dir, "selector_call.go")
	content := `package somethingservice

import "github.com/s1onique/leamas/internal/factory/dupcode"

func Process() {
	dupcode.CheckReport(".", dupcode.DefaultConfig())
}
`
	if err := os.WriteFile(fixture, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write fixture: %v", err)
	}

	policy := NewDupcodeBypassPolicy()
	err := policy.CheckFile(fixture)
	if err == nil {
		t.Error("expected error for selector call from unauthorized package")
	}
}

func TestDupcodeBypassPolicy_RejectsBaselineWrite(t *testing.T) {
	// Create a temp file with baseline write
	dir := t.TempDir()
	fixture := filepath.Join(dir, "baseline_write.go")
	content := `package badactor

import "github.com/s1onique/leamas/internal/factory/dupcode"

func Update() {
	dupcode.WriteBaseline(".factory/dupcode-baseline.json", dupcode.Report{})
}
`
	if err := os.WriteFile(fixture, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write fixture: %v", err)
	}

	policy := NewDupcodeBypassPolicy()
	err := policy.CheckFile(fixture)
	if err == nil {
		t.Error("expected error for baseline write from unauthorized package")
	}
}
