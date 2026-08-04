// SPDX-License-Identifier: Apache-2.0

package forbidden

import (
	"os"
	"path/filepath"
	"testing"
)

const testModulePath = "github.com/s1onique/leamas"

func createTestModule(t *testing.T, dir string) {
	goMod := `module github.com/s1onique/leamas

go 1.21
`
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte(goMod), 0644); err != nil {
		t.Fatalf("failed to write go.mod: %v", err)
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
	createTestModule(t, dir)

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

func TestProtectedSymbolsMap(t *testing.T) {
	m := ProtectedSymbolsMap()
	if len(m) == 0 {
		t.Error("ProtectedSymbolsMap should not be empty")
	}

	expected := []string{
		"github.com/s1onique/leamas/internal/factory/dupcode.CheckRepo",
		"github.com/s1onique/leamas/internal/factory/dupcode.CheckReport",
	}
	for _, key := range expected {
		if !m[key] {
			t.Errorf("expected %q in ProtectedSymbolsMap", key)
		}
	}
}

func TestIsApprovedCaller(t *testing.T) {
	// Approved *DupcodeRunner.RunCheckRepo caller should return true
	caller := CallerIdentity{
		PackagePath: "github.com/s1onique/leamas/internal/factory/protectedverifier",
		Function:    "RunCheckRepo",
		Receiver:    "DupcodeRunner",
		Kind:        "method",
	}
	if !IsApprovedCaller(caller, ProtectedSymbol{
		Layer:       AuthorityLayerRaw,
		PackagePath: "github.com/s1onique/leamas/internal/factory/dupcode",
		Name:        "CheckRepo",
		Kind:        ProtectedPackageFunction,
	}) {
		t.Error("approved *DupcodeRunner.RunCheckRepo caller should return true")
	}

	// Unapproved function should return false
	if IsApprovedCaller(
		CallerIdentity{PackagePath: "github.com/s1onique/leamas/internal/factory/gate", Function: "AnyFunc", Receiver: "", Kind: "package_function"},
		ProtectedSymbol{Layer: AuthorityLayerRaw, PackagePath: "github.com/s1onique/leamas/internal/factory/dupcode", Name: "CheckRepo", Kind: ProtectedPackageFunction},
	) {
		t.Error("unapproved caller should return false")
	}
}

func TestFindProtectedSymbol(t *testing.T) {
	sym := FindProtectedSymbol("github.com/s1onique/leamas/internal/factory/dupcode", "CheckRepo")
	if sym == nil {
		t.Error("expected to find CheckRepo symbol")
	}

	nonexistent := FindProtectedSymbol("github.com/s1onique/leamas/internal/factory/dupcode", "NonExistent")
	if nonexistent != nil {
		t.Error("expected nil for nonexistent symbol")
	}
}

func TestCanonicalPolicy_LoadsPackagesFromRepoRoot(t *testing.T) {
	dir := t.TempDir()

	pkgDir := filepath.Join(dir, "internal", "app")
	if err := os.MkdirAll(pkgDir, 0755); err != nil {
		t.Fatalf("failed to create pkg dir: %v", err)
	}

	fixture := filepath.Join(pkgDir, "main.go")
	content := `package app
func Run() {}`
	if err := os.WriteFile(fixture, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write fixture: %v", err)
	}

	if err := os.MkdirAll(filepath.Join(dir, ".git"), 0755); err != nil {
		t.Fatalf("failed to create .git dir: %v", err)
	}
	createTestModule(t, dir)

	findings := CanonicalCheckDupcodeBypass(dir, testModulePath)
	for _, f := range findings {
		if f.Kind == "dupcode_package_load_error" {
			t.Errorf("unexpected package load error: %s", f.Message)
		}
	}
}

func TestCanonicalPolicy_FailClosedOnPackageErrors(t *testing.T) {
	// This test ensures that package errors trigger fail-closed behavior.
	// We use a non-existent path which should produce canonical_root_error.
	findings := CanonicalCheckDupcodeBypass("/nonexistent", testModulePath)
	if len(findings) == 0 {
		t.Error("expected findings for nonexistent path")
	}
	foundCanonicalRoot := false
	for _, f := range findings {
		if f.Kind == "canonical_root_error" {
			foundCanonicalRoot = true
			break
		}
	}
	if !foundCanonicalRoot {
		t.Error("expected canonical_root_error finding")
	}
}

func TestCallerIdentityEnforcedStrict(t *testing.T) {
	// An empty Function in ApprovedCallers must NOT match.
	if IsApprovedCaller(
		CallerIdentity{PackagePath: "github.com/s1onique/leamas/internal/factory/protectedverifier", Function: ""},
		ProtectedSymbol{PackagePath: "github.com/s1onique/leamas/internal/factory/dupcode", Name: "CheckRepo", Kind: ProtectedPackageFunction},
	) {
		t.Error("empty Function should not be approved (no wildcards)")
	}
}
