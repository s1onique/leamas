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
	// Create a temp file in the canonical adapter package
	dir := t.TempDir()
	fixture := filepath.Join(dir, "canonical_adapter.go")
	content := `package protectedverifier

import "github.com/s1onique/leamas/internal/factory/dupcode"

func RunCheck(root string) {
	dupcode.CheckRepo(root)
}
`
	if err := os.WriteFile(fixture, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write fixture: %v", err)
	}

	policy := NewDupcodeBypassPolicy()
	err := policy.CheckFile(fixture)
	if err != nil {
		t.Errorf("unexpected error for canonical adapter: %v", err)
	}
}

func TestDupcodeBypassPolicy_AllowsDupcodePackage(t *testing.T) {
	// Create a temp file in the dupcode package itself
	dir := t.TempDir()
	fixture := filepath.Join(dir, "dupcode_internal.go")
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

	policy := NewDupcodeBypassPolicy()
	err := policy.CheckFile(fixture)
	if err != nil {
		t.Errorf("unexpected error for dupcode package: %v", err)
	}
}

func TestDupcodeBypassPolicy_CurrentRepository(t *testing.T) {
	// Check the current repository for bypasses
	policy := NewDupcodeBypassPolicy()

	// Walk the internal/factory directory
	factoryDir := filepath.Join("..", "..", "internal", "factory")
	if _, err := os.Stat(factoryDir); os.IsNotExist(err) {
		// Try from repo root
		factoryDir = filepath.Join("internal", "factory")
	}

	if _, err := os.Stat(factoryDir); err != nil {
		t.Skip("factory directory not found, skipping repository check")
	}

	// Check gate package - this should be allowed
	gateDir := filepath.Join(factoryDir, "gate")
	if _, err := os.Stat(gateDir); err == nil {
		err := policy.CheckPackageDir(gateDir)
		if err != nil {
			t.Errorf("gate package check failed: %v", err)
		}
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
