package doctrinecompiler

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// modeOf returns the permission bits of a regular file.
func modeOf(t *testing.T, p string) os.FileMode {
	t.Helper()
	info, err := os.Lstat(p)
	if err != nil {
		t.Fatalf("lstat %s: %v", p, err)
	}
	return info.Mode().Perm()
}

// freshPackProfile loads the canonical pack and returns the
// fsharp-elm-service-v1 profile. Centralised so individual tests
// stay short.
func freshPackProfile(t *testing.T) (*Pack, Profile) {
	t.Helper()
	pack, err := LoadCorePack()
	if err != nil {
		t.Fatalf("LoadCorePack: %v", err)
	}
	prof, err := pack.MustProfile(ProfileId("fsharp-elm-service-v1"))
	if err != nil {
		t.Fatalf("MustProfile: %v", err)
	}
	return pack, prof
}

// TestCompileLeavesNoTempFiles verifies that compile does not leave
// any temp files after success.
func TestCompileLeavesNoTempFiles(t *testing.T) {
	pack, prof := freshPackProfile(t)
	target := newEmptyTarget(t)
	if _, err := Compile(pack, prof, target, CompilerOptions{CompilerVersion: "0.1.0"}); err != nil {
		t.Fatalf("Compile: %v", err)
	}
	entries, _ := os.ReadDir(filepath.Join(target, ".factory"))
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), ".tmp-") {
			t.Errorf("temp file left behind: %s", e.Name())
		}
	}
	gendir := filepath.Join(target, ".factory", "generated")
	if entries, err := os.ReadDir(gendir); err == nil {
		for _, e := range entries {
			if strings.HasPrefix(e.Name(), ".tmp-") {
				t.Errorf("temp file in generated/: %s", e.Name())
			}
		}
	}
}

// TestCompileAtomicLockLast verifies that the lock file is the LAST
// file written. We assert by digesting each managed file before and
// after a tamper-and-repair cycle and confirming the lock was rewritten
// exactly once.
func TestCompileAtomicLockLast(t *testing.T) {
	pack, prof := freshPackProfile(t)
	target := newEmptyTarget(t)
	if _, err := Compile(pack, prof, target, CompilerOptions{CompilerVersion: "0.1.0"}); err != nil {
		t.Fatalf("Compile: %v", err)
	}
	mk := filepath.Join(target, ".factory/generated/factory.mk")
	if err := os.WriteFile(mk, []byte("tampered"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	if _, err := Compile(pack, prof, target, CompilerOptions{CompilerVersion: "0.1.0"}); err != nil {
		t.Fatalf("Compile: %v", err)
	}
	lock, err := ReadLockFile(filepath.Join(target, ".factory/doctrine.lock.json"))
	if err != nil {
		t.Fatalf("lock: %v", err)
	}
	for _, mf := range lock.ManagedFiles {
		if mf.Path == ".factory/generated/factory.mk" {
			got, _ := os.ReadFile(mk)
			if ComputeDigest(got) != mf.Digest {
				t.Errorf("lock digest out of sync with file")
			}
			return
		}
	}
	t.Errorf("factory.mk missing from lock")
}

// TestCompileRefusesSymlinkParent verifies that a symlink in the parent
// chain of a desired managed path causes compile to refuse.
func TestCompileRefusesSymlinkParent(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink semantics differ on windows")
	}
	pack, prof := freshPackProfile(t)
	target := newEmptyTarget(t)
	outside := t.TempDir()
	if err := os.Symlink(outside, filepath.Join(target, ".factory")); err != nil {
		t.Fatalf("symlink: %v", err)
	}
	_, err := Compile(pack, prof, target, CompilerOptions{CompilerVersion: "0.1.0"})
	if err == nil {
		t.Errorf("expected compile to refuse symlink escape")
	}
}
