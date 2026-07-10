// SPDX-License-Identifier: Apache-2.0

package digest

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestPublicSurfaceDelta_RealRemoval verifies that real symbol removals are still detected.
func TestPublicSurfaceDelta_RealRemoval(t *testing.T) {
	tmpDir := t.TempDir()
	initGit(t, tmpDir)
	runGit(t, tmpDir, "commit", "--allow-empty", "-m", "initial commit")

	pkgDir := filepath.Join(tmpDir, "pkg", "example")
	if err := os.MkdirAll(pkgDir, 0755); err != nil {
		t.Fatalf("failed to create pkg dir: %v", err)
	}

	goFile := filepath.Join(pkgDir, "example.go")
	beforeContent := `package example

// PublicFunc is a public function.
func PublicFunc() {}

// RemovedPublicFunc was a public function.
func RemovedPublicFunc() {}
`
	if err := os.WriteFile(goFile, []byte(beforeContent), 0644); err != nil {
		t.Fatalf("failed to write go file: %v", err)
	}
	runGit(t, tmpDir, "add", ".")
	runGit(t, tmpDir, "commit", "-m", "add example package")

	afterContent := `package example

// PublicFunc is a public function.
func PublicFunc() {}
`
	if err := os.WriteFile(goFile, []byte(afterContent), 0644); err != nil {
		t.Fatalf("failed to update go file: %v", err)
	}
	runGit(t, tmpDir, "add", ".")
	runGit(t, tmpDir, "commit", "-m", "remove RemovedPublicFunc")

	output, err := Generate(Options{
		RepoRoot: tmpDir,
		Mode:     ModeRange,
		Range:    "HEAD~1..HEAD",
	})
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	if !strings.Contains(output, "- pkg.example.RemovedPublicFunc(func)") {
		t.Error("Expected RemovedPublicFunc to be in removed section")
	}
	if !strings.Contains(output, "symbols_removed=1") {
		t.Errorf("Expected symbols_removed=1, got: %s", extractField(output, "symbols_removed"))
	}
}

// TestPublicSurfaceDelta_RealAddition verifies that real symbol additions are still detected.
func TestPublicSurfaceDelta_RealAddition(t *testing.T) {
	tmpDir := t.TempDir()
	initGit(t, tmpDir)
	runGit(t, tmpDir, "commit", "--allow-empty", "-m", "initial commit")

	pkgDir := filepath.Join(tmpDir, "pkg", "example")
	if err := os.MkdirAll(pkgDir, 0755); err != nil {
		t.Fatalf("failed to create pkg dir: %v", err)
	}

	goFile := filepath.Join(pkgDir, "example.go")
	beforeContent := `package example

// ExistingFunc is an existing function.
func ExistingFunc() {}
`
	if err := os.WriteFile(goFile, []byte(beforeContent), 0644); err != nil {
		t.Fatalf("failed to write go file: %v", err)
	}
	runGit(t, tmpDir, "add", ".")
	runGit(t, tmpDir, "commit", "-m", "add example package")

	afterContent := `package example

// ExistingFunc is an existing function.
func ExistingFunc() {}

// AddedPublicFunc is a new public function.
func AddedPublicFunc() {}
`
	if err := os.WriteFile(goFile, []byte(afterContent), 0644); err != nil {
		t.Fatalf("failed to update go file: %v", err)
	}
	runGit(t, tmpDir, "add", ".")
	runGit(t, tmpDir, "commit", "-m", "add AddedPublicFunc")

	output, err := Generate(Options{
		RepoRoot: tmpDir,
		Mode:     ModeRange,
		Range:    "HEAD~1..HEAD",
	})
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	if !strings.Contains(output, "- pkg.example.AddedPublicFunc(func)") {
		t.Error("Expected AddedPublicFunc to be in added section")
	}
	if !strings.Contains(output, "symbols_added=1") {
		t.Errorf("Expected symbols_added=1, got: %s", extractField(output, "symbols_added"))
	}
}

// TestPublicSurfaceDelta_SignatureChange verifies that signature changes are still detected.
func TestPublicSurfaceDelta_SignatureChange(t *testing.T) {
	tmpDir := t.TempDir()
	initGit(t, tmpDir)
	runGit(t, tmpDir, "commit", "--allow-empty", "-m", "initial commit")

	pkgDir := filepath.Join(tmpDir, "pkg", "example")
	if err := os.MkdirAll(pkgDir, 0755); err != nil {
		t.Fatalf("failed to create pkg dir: %v", err)
	}

	goFile := filepath.Join(pkgDir, "example.go")
	beforeContent := `package example

// PublicFunc has an old signature.
func PublicFunc(a string) error {
	return nil
}
`
	if err := os.WriteFile(goFile, []byte(beforeContent), 0644); err != nil {
		t.Fatalf("failed to write go file: %v", err)
	}
	runGit(t, tmpDir, "add", ".")
	runGit(t, tmpDir, "commit", "-m", "add example package")

	afterContent := `package example

// PublicFunc has a new signature.
func PublicFunc(a string, strict bool) error {
	return nil
}
`
	if err := os.WriteFile(goFile, []byte(afterContent), 0644); err != nil {
		t.Fatalf("failed to update go file: %v", err)
	}
	runGit(t, tmpDir, "add", ".")
	runGit(t, tmpDir, "commit", "-m", "change PublicFunc signature")

	output, err := Generate(Options{
		RepoRoot: tmpDir,
		Mode:     ModeRange,
		Range:    "HEAD~1..HEAD",
	})
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	if !strings.Contains(output, "- pkg.example.PublicFunc(func)") {
		t.Error("Expected PublicFunc to be in modified section")
	}
	if !strings.Contains(output, "symbols_modified=1") {
		t.Errorf("Expected symbols_modified=1, got: %s", extractField(output, "symbols_modified"))
	}
}

// TestPublicSurfaceDelta_FieldRemovalStillReported verifies that field removal is still detected.
func TestPublicSurfaceDelta_FieldRemovalStillReported(t *testing.T) {
	tmpDir := t.TempDir()
	initGit(t, tmpDir)
	runGit(t, tmpDir, "commit", "--allow-empty", "-m", "initial commit")

	pkgDir := filepath.Join(tmpDir, "pkg", "example")
	if err := os.MkdirAll(pkgDir, 0755); err != nil {
		t.Fatalf("failed to create pkg dir: %v", err)
	}

	goFile := filepath.Join(pkgDir, "example.go")
	beforeContent := `package example

// MyStruct is a struct.
type MyStruct struct {
	Name string
	Value int
}
`
	if err := os.WriteFile(goFile, []byte(beforeContent), 0644); err != nil {
		t.Fatalf("failed to write go file: %v", err)
	}
	runGit(t, tmpDir, "add", ".")
	runGit(t, tmpDir, "commit", "-m", "add example package")

	afterContent := `package example

// MyStruct is a struct.
type MyStruct struct {
	Name string
}
`
	if err := os.WriteFile(goFile, []byte(afterContent), 0644); err != nil {
		t.Fatalf("failed to update go file: %v", err)
	}
	runGit(t, tmpDir, "add", ".")
	runGit(t, tmpDir, "commit", "-m", "remove Value field")

	output, err := Generate(Options{
		RepoRoot: tmpDir,
		Mode:     ModeRange,
		Range:    "HEAD~1..HEAD",
	})
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	if !strings.Contains(output, "- pkg.example.MyStruct.Value.field(MyStruct)") {
		t.Error("Expected MyStruct.Value field to be in removed section")
	}
}

// TestMergeExports verifies that exports from multiple files in the same package are merged.
func TestMergeExports(t *testing.T) {
	tests := []struct {
		name         string
		pkgExports   map[symbolKey]symbolInfo
		newExports   map[symbolKey]symbolInfo
		wantCount    int
		wantContains []symbolKey
	}{
		{
			name:       "empty pkg exports",
			pkgExports: make(map[symbolKey]symbolInfo),
			newExports: map[symbolKey]symbolInfo{
				{Name: "Foo", Kind: "func"}: {Key: symbolKey{Name: "Foo", Kind: "func"}, Signature: "func Foo()"},
			},
			wantCount:    1,
			wantContains: []symbolKey{{Name: "Foo", Kind: "func"}},
		},
		{
			name: "multiple files add to same package",
			pkgExports: map[symbolKey]symbolInfo{
				{Name: "Foo", Kind: "func"}: {Key: symbolKey{Name: "Foo", Kind: "func"}, Signature: "func Foo()"},
			},
			newExports: map[symbolKey]symbolInfo{
				{Name: "Bar", Kind: "func"}: {Key: symbolKey{Name: "Bar", Kind: "func"}, Signature: "func Bar()"},
			},
			wantCount:    2,
			wantContains: []symbolKey{{Name: "Foo", Kind: "func"}, {Name: "Bar", Kind: "func"}},
		},
		{
			name: "duplicate symbol not overwritten",
			pkgExports: map[symbolKey]symbolInfo{
				{Name: "Foo", Kind: "func"}: {Key: symbolKey{Name: "Foo", Kind: "func"}, Signature: "func Foo()"},
			},
			newExports: map[symbolKey]symbolInfo{
				{Name: "Foo", Kind: "func"}: {Key: symbolKey{Name: "Foo", Kind: "func"}, Signature: "func Foo()"},
			},
			wantCount:    1,
			wantContains: []symbolKey{{Name: "Foo", Kind: "func"}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mergeExports(tt.pkgExports, tt.newExports)
			if len(tt.pkgExports) != tt.wantCount {
				t.Errorf("mergeExports() = %d, want %d", len(tt.pkgExports), tt.wantCount)
			}
			for _, want := range tt.wantContains {
				if _, exists := tt.pkgExports[want]; !exists {
					t.Errorf("mergeExports() missing: %v", want)
				}
			}
		})
	}
}
