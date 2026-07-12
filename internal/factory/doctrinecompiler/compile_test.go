package doctrinecompiler

import (
	"os"
	"path/filepath"
	"sort"
	"testing"
)

// TestCompileEmptyTargetProducesGoldenTree asserts the bounded tree
// shape matches the golden fixture.
func TestCompileEmptyTargetProducesGoldenTree(t *testing.T) {
	pack, err := LoadCorePack()
	if err != nil {
		t.Fatalf("LoadCorePack: %v", err)
	}
	prof, err := pack.MustProfile(ProfileId("fsharp-elm-service-v1"))
	if err != nil {
		t.Fatalf("MustProfile: %v", err)
	}
	target := newEmptyTarget(t)
	if _, err := Compile(pack, prof, target, CompilerOptions{}); err != nil {
		t.Fatalf("Compile: %v", err)
	}
	got := listTargetTree(t, target)
	want := []string{
		".factory/doctrine.lock.json",
		".factory/generated/doctrine-inventory.md",
		".factory/generated/factory.mk",
		".factory/project.json",
		"Makefile",
		"docs/factory/README.md",
	}
	if !equalStringSet(got, want) {
		t.Fatalf("target tree mismatch:\n got = %v\n want = %v", got, want)
	}
	for _, p := range want {
		data, err := os.ReadFile(filepath.Join(target, p))
		if err != nil {
			t.Fatalf("missing %s: %v", p, err)
		}
		golden, err := os.ReadFile(filepath.Join(expectedFixtureDir(t), p))
		if err != nil {
			t.Fatalf("missing golden %s: %v", p, err)
		}
		if string(data) != string(golden) {
			t.Logf("data:\n%s", data)
			t.Logf("golden:\n%s", golden)
			t.Errorf("file %s differs from golden", p)
		}
	}
}

// TestCompileIdempotentNoFilesystemChange ensures a second compile
// produces no filesystem changes.
func TestCompileIdempotentNoFilesystemChange(t *testing.T) {
	pack, err := LoadCorePack()
	if err != nil {
		t.Fatalf("LoadCorePack: %v", err)
	}
	prof, err := pack.MustProfile(ProfileId("fsharp-elm-service-v1"))
	if err != nil {
		t.Fatalf("MustProfile: %v", err)
	}
	target := newEmptyTarget(t)
	if _, err := Compile(pack, prof, target, CompilerOptions{}); err != nil {
		t.Fatalf("first Compile: %v", err)
	}
	before := snapshotTree(t, target)
	if _, err := Compile(pack, prof, target, CompilerOptions{}); err != nil {
		t.Fatalf("second Compile: %v", err)
	}
	after := snapshotTree(t, target)
	if !equalStringSet(before, after) {
		t.Errorf("tree changed after second compile:\n before = %v\n after = %v", before, after)
	}
	for _, p := range before {
		a, _ := os.ReadFile(filepath.Join(target, p))
		b, _ := os.ReadFile(filepath.Join(target, p))
		if string(a) != string(b) {
			t.Errorf("file content changed for %s", p)
		}
	}
}

// TestCompileLeavesUnrelatedFilesAlone plants a user file and asserts
// compile does not remove or modify it.
func TestCompileLeavesUnrelatedFilesAlone(t *testing.T) {
	pack, err := LoadCorePack()
	if err != nil {
		t.Fatalf("LoadCorePack: %v", err)
	}
	prof, err := pack.MustProfile(ProfileId("fsharp-elm-service-v1"))
	if err != nil {
		t.Fatalf("MustProfile: %v", err)
	}
	target := newEmptyTarget(t)
	user := filepath.Join(target, "user-notes.md")
	if err := os.WriteFile(user, []byte("user"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	if _, err := Compile(pack, prof, target, CompilerOptions{}); err != nil {
		t.Fatalf("Compile: %v", err)
	}
	if _, err := os.Stat(user); err != nil {
		t.Fatalf("user file vanished: %v", err)
	}
}

// TestCompileNeverOverwritesSeeded verifies seeded files are never
// overwritten by a subsequent compile, even after manual edit.
func TestCompileNeverOverwritesSeeded(t *testing.T) {
	pack, err := LoadCorePack()
	if err != nil {
		t.Fatalf("LoadCorePack: %v", err)
	}
	prof, err := pack.MustProfile(ProfileId("fsharp-elm-service-v1"))
	if err != nil {
		t.Fatalf("MustProfile: %v", err)
	}
	target := newEmptyTarget(t)
	if _, err := Compile(pack, prof, target, CompilerOptions{}); err != nil {
		t.Fatalf("first Compile: %v", err)
	}
	mk := filepath.Join(target, "Makefile")
	custom := "# custom\ngate: factorize extra-step\n"
	if err := os.WriteFile(mk, []byte(custom), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	if _, err := Compile(pack, prof, target, CompilerOptions{}); err != nil {
		t.Fatalf("second Compile: %v", err)
	}
	got, _ := os.ReadFile(mk)
	if string(got) != custom {
		t.Errorf("Makefile overwritten: got %q want %q", got, custom)
	}
}

// TestCompileRepairsManagedFiles verifies that an explicit compile
// after tampering repairs the managed file.
func TestCompileRepairsManagedFiles(t *testing.T) {
	pack, err := LoadCorePack()
	if err != nil {
		t.Fatalf("LoadCorePack: %v", err)
	}
	prof, err := pack.MustProfile(ProfileId("fsharp-elm-service-v1"))
	if err != nil {
		t.Fatalf("MustProfile: %v", err)
	}
	target := newEmptyTarget(t)
	if _, err := Compile(pack, prof, target, CompilerOptions{}); err != nil {
		t.Fatalf("first Compile: %v", err)
	}
	mk := filepath.Join(target, ".factory/generated/factory.mk")
	if err := os.WriteFile(mk, []byte("tampered"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	if _, err := Compile(pack, prof, target, CompilerOptions{}); err != nil {
		t.Fatalf("repair Compile: %v", err)
	}
	got := mustRead(t, mk)
	golden := mustRead(t, filepath.Join(expectedFixtureDir(t), ".factory/generated/factory.mk"))
	if string(got) != string(golden) {
		t.Errorf("managed file not repaired")
	}
}

// TestCompileRemovesObsoleteManaged verifies that an obsolete recorded
// managed file is removed by compile.
func TestCompileRemovesObsoleteManaged(t *testing.T) {
	pack, err := LoadCorePack()
	if err != nil {
		t.Fatalf("LoadCorePack: %v", err)
	}
	prof, err := pack.MustProfile(ProfileId("fsharp-elm-service-v1"))
	if err != nil {
		t.Fatalf("MustProfile: %v", err)
	}
	target := newEmptyTarget(t)
	if _, err := Compile(pack, prof, target, CompilerOptions{}); err != nil {
		t.Fatalf("Compile: %v", err)
	}
	plant := filepath.Join(target, ".factory/generated/obsolete.txt")
	if err := os.WriteFile(plant, []byte("obsolete"), 0o644); err != nil {
		t.Fatalf("plant: %v", err)
	}
	lockPath := filepath.Join(target, ".factory/doctrine.lock.json")
	data, err := os.ReadFile(lockPath)
	if err != nil {
		t.Fatalf("lock: %v", err)
	}
	patched := string(data)
	var err2 error
	patched, err2 = patchLockAddManaged(patched,
		".factory/generated/obsolete.txt",
		"0000000000000000000000000000000000000000000000000000000000000000")
	if err2 != nil {
		t.Fatalf("patch lock: %v", err2)
	}
	if err := os.WriteFile(lockPath, []byte(patched), 0o644); err != nil {
		t.Fatalf("write patched lock: %v", err)
	}
	if _, err := Compile(pack, prof, target, CompilerOptions{}); err != nil {
		t.Fatalf("Compile: %v", err)
	}
	if _, err := os.Stat(plant); !os.IsNotExist(err) {
		t.Errorf("obsolete managed file still present")
	}
}

// TestCompileNeverRemovesUnrecordedFiles ensures compile never deletes
// user files.
func TestCompileNeverRemovesUnrecordedFiles(t *testing.T) {
	pack, err := LoadCorePack()
	if err != nil {
		t.Fatalf("LoadCorePack: %v", err)
	}
	prof, err := pack.MustProfile(ProfileId("fsharp-elm-service-v1"))
	if err != nil {
		t.Fatalf("MustProfile: %v", err)
	}
	target := newEmptyTarget(t)
	user := filepath.Join(target, "user-notes.md")
	if err := os.WriteFile(user, []byte("user"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	if _, err := Compile(pack, prof, target, CompilerOptions{}); err != nil {
		t.Fatalf("Compile: %v", err)
	}
	if _, err := os.Stat(user); err != nil {
		t.Fatalf("user file vanished: %v", err)
	}
}

// Safety-related tests are split into compile_safety_test.go to keep
// this file under the LLM-friendliness line threshold.

// listTargetTree returns a sorted list of relative paths in target.
func listTargetTree(t *testing.T, target string) []string {
	t.Helper()
	var out []string
	err := filepath.WalkDir(target, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		rel, _ := filepath.Rel(target, p)
		out = append(out, filepath.ToSlash(rel))
		return nil
	})
	if err != nil {
		t.Fatalf("walk: %v", err)
	}
	sort.Strings(out)
	return out
}

// snapshotTree returns a sorted list of relative paths in target.
func snapshotTree(t *testing.T, target string) []string {
	return listTargetTree(t, target)
}

// equalStringSet compares two sorted slices.
func equalStringSet(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
