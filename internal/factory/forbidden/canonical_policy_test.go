// SPDX-License-Identifier: Apache-2.0

package forbidden

import (
	"os"
	"path/filepath"
	"testing"
)

const testModulePath = "github.com/s1onique/leamas"

func TestCanonicalPolicy_RejectsNestedBypass(t *testing.T) {
	dir := t.TempDir()
	bypassDir := filepath.Join(dir, "cmd", "a", "b", "c")
	if err := os.MkdirAll(bypassDir, 0755); err != nil {
		t.Fatalf("failed to create nested dir: %v", err)
	}
	fixture := filepath.Join(bypassDir, "bypass.go")
	content := `package bypass
import "github.com/s1onique/leamas/internal/factory/dupcode"
func Run() { dupcode.CheckRepo(".") }`
	if err := os.WriteFile(fixture, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write fixture: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(dir, ".git"), 0755); err != nil {
		t.Fatalf("failed to create .git dir: %v", err)
	}
	findings := CanonicalCheckDupcodeBypass(dir, testModulePath)
	found := false
	for _, f := range findings {
		if f.Kind == "dupcode_bypass" && filepath.Base(f.Path) == "bypass.go" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected to find nested bypass in cmd/a/b/c/bypass.go")
	}
}

func TestCanonicalPolicy_RejectsSiblingsBypass(t *testing.T) {
	dir := t.TempDir()
	pkgA := filepath.Join(dir, "internal", "a")
	pkgB := filepath.Join(dir, "internal", "b")
	if err := os.MkdirAll(pkgA, 0755); err != nil {
		t.Fatalf("failed to create pkgA: %v", err)
	}
	if err := os.MkdirAll(pkgB, 0755); err != nil {
		t.Fatalf("failed to create pkgB: %v", err)
	}
	fixtureA := filepath.Join(pkgA, "bypass.go")
	contentA := `package a
import "github.com/s1onique/leamas/internal/factory/dupcode"
func Run() { dupcode.CheckReport(".", dupcode.DefaultConfig()) }`
	if err := os.WriteFile(fixtureA, []byte(contentA), 0644); err != nil {
		t.Fatalf("failed to write fixtureA: %v", err)
	}
	fixtureB := filepath.Join(pkgB, "clean.go")
	contentB := `package b
func Run() {}`
	if err := os.WriteFile(fixtureB, []byte(contentB), 0644); err != nil {
		t.Fatalf("failed to write fixtureB: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(dir, ".git"), 0755); err != nil {
		t.Fatalf("failed to create .git dir: %v", err)
	}
	findings := CanonicalCheckDupcodeBypass(dir, testModulePath)
	found := false
	for _, f := range findings {
		if f.Kind == "dupcode_bypass" && filepath.Base(f.Path) == "bypass.go" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected to find sibling bypass in internal/a/bypass.go")
	}
}

func TestCanonicalPolicy_RejectsAliasedImport(t *testing.T) {
	dir := t.TempDir()
	fixture := filepath.Join(dir, "cmd", "app", "aliased.go")
	if err := os.MkdirAll(filepath.Dir(fixture), 0755); err != nil {
		t.Fatalf("failed to create dir: %v", err)
	}
	content := `package app
import dc "github.com/s1onique/leamas/internal/factory/dupcode"
func Run() { dc.CheckRepo(".") }`
	if err := os.WriteFile(fixture, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write fixture: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(dir, ".git"), 0755); err != nil {
		t.Fatalf("failed to create .git dir: %v", err)
	}
	findings := CanonicalCheckDupcodeBypass(dir, testModulePath)
	found := false
	for _, f := range findings {
		if f.Kind == "dupcode_bypass" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected to find aliased import bypass")
	}
}

func TestCanonicalPolicy_RejectsDotImport(t *testing.T) {
	dir := t.TempDir()
	fixture := filepath.Join(dir, "cmd", "app", "dotimport.go")
	if err := os.MkdirAll(filepath.Dir(fixture), 0755); err != nil {
		t.Fatalf("failed to create dir: %v", err)
	}
	content := `package app
import . "github.com/s1onique/leamas/internal/factory/dupcode"
func Run() { CheckRepo(".") }`
	if err := os.WriteFile(fixture, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write fixture: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(dir, ".git"), 0755); err != nil {
		t.Fatalf("failed to create .git dir: %v", err)
	}
	findings := CanonicalCheckDupcodeBypass(dir, testModulePath)
	found := false
	for _, f := range findings {
		if f.Kind == "dupcode_bypass" && f.Message != "" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected to find dot import bypass")
	}
}

func TestCanonicalPolicy_IgnoresComments(t *testing.T) {
	dir := t.TempDir()
	fixture := filepath.Join(dir, "cmd", "app", "comment.go")
	if err := os.MkdirAll(filepath.Dir(fixture), 0755); err != nil {
		t.Fatalf("failed to create dir: %v", err)
	}
	content := `package app
// dupcode.CheckRepo should not be flagged
func Run() { /* dupcode.CheckReport */ }`
	if err := os.WriteFile(fixture, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write fixture: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(dir, ".git"), 0755); err != nil {
		t.Fatalf("failed to create .git dir: %v", err)
	}
	findings := CanonicalCheckDupcodeBypass(dir, testModulePath)
	for _, f := range findings {
		if f.Kind == "dupcode_bypass" {
			t.Errorf("comment-only mention should not trigger bypass: %s", f.Message)
		}
	}
}

func TestCanonicalPolicy_AllowsCanonicalAdapter(t *testing.T) {
	dir := t.TempDir()
	adapterDir := filepath.Join(dir, "internal", "factory", "protectedverifier")
	if err := os.MkdirAll(adapterDir, 0755); err != nil {
		t.Fatalf("failed to create adapter dir: %v", err)
	}
	fixture := filepath.Join(adapterDir, "adapter.go")
	content := `package protectedverifier
import "github.com/s1onique/leamas/internal/factory/dupcode"
func RunCheck(root string) { dupcode.CheckRepo(root) }`
	if err := os.WriteFile(fixture, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write fixture: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(dir, ".git"), 0755); err != nil {
		t.Fatalf("failed to create .git dir: %v", err)
	}
	findings := CanonicalCheckDupcodeBypass(dir, testModulePath)
	for _, f := range findings {
		if f.Kind == "dupcode_bypass" {
			t.Errorf("canonical adapter should be allowed: %s", f.Message)
		}
	}
}

func TestCanonicalPolicy_DeterministicFindings(t *testing.T) {
	dir := t.TempDir()
	fixture := filepath.Join(dir, "cmd", "app", "main.go")
	if err := os.MkdirAll(filepath.Dir(fixture), 0755); err != nil {
		t.Fatalf("failed to create dir: %v", err)
	}
	content := `package app
func Run() {}`
	if err := os.WriteFile(fixture, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write fixture: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(dir, ".git"), 0755); err != nil {
		t.Fatalf("failed to create .git dir: %v", err)
	}
	findings1 := CanonicalCheckDupcodeBypass(dir, testModulePath)
	findings2 := CanonicalCheckDupcodeBypass(dir, testModulePath)
	if len(findings1) != len(findings2) {
		t.Errorf("finding count differs: %d vs %d", len(findings1), len(findings2))
	}
	for i := range findings1 {
		if findings1[i].Path != findings2[i].Path {
			t.Errorf("finding path differs at index %d: %s vs %s", i, findings1[i].Path, findings2[i].Path)
		}
	}
}

func TestCanonicalPolicy_RejectsNonRepository(t *testing.T) {
	dir := t.TempDir()
	findings := CanonicalCheckDupcodeBypass(dir, testModulePath)
	found := false
	for _, f := range findings {
		if f.Kind == "canonical_root_error" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected canonical_root_error for non-repository path")
	}
}

func TestCanonicalPolicy_RejectsNonexistentPath(t *testing.T) {
	findings := CanonicalCheckDupcodeBypass("/nonexistent/path/to/repo", testModulePath)
	found := false
	for _, f := range findings {
		if f.Kind == "canonical_root_error" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected canonical_root_error for nonexistent path")
	}
}
