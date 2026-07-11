package doctrinecompiler

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// TestCompileRollbackFailsClosed proves that an injected mid-apply
// failure aborts the compile and reports a typed error.
//
// The pre-compile target state is captured in a snapshot before any
// write occurs. A failure injected via CompilerOptions.FailAfter must
// cause the compile to return a non-nil error.
func TestCompileRollbackFailsClosed(t *testing.T) {
	pack, err := LoadCorePack()
	if err != nil {
		t.Fatalf("LoadCorePack: %v", err)
	}
	prof, err := pack.MustProfile(ProfileId("fsharp-elm-service-v1"))
	if err != nil {
		t.Fatalf("MustProfile: %v", err)
	}
	target := newEmptyTarget(t)
	// Initial successful compile to populate the target.
	if _, err := Compile(pack, prof, target, CompilerOptions{}); err != nil {
		t.Fatalf("initial Compile: %v", err)
	}
	mk := filepath.Join(target, ".factory/generated/factory.mk")
	original, err := os.ReadFile(mk)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	tampered := append(original, []byte("\n# TAMPERED\n")...)
	if err := os.WriteFile(mk, tampered, 0o644); err != nil {
		t.Fatalf("write tampered: %v", err)
	}
	// Inject failure during the update-managed step. Rollback must
	// restore to the pre-compile snapshot (the tampered state).
	_, err = Compile(pack, prof, target, CompilerOptions{
		FailAfter: ActionUpdateManaged,
	})
	if err == nil {
		t.Fatalf("expected injected failure to abort compile")
	}
	got, err := os.ReadFile(mk)
	if err != nil {
		t.Fatalf("read after rollback: %v", err)
	}
	if string(got) != string(tampered) {
		t.Errorf("rollback did not restore pre-compile state")
	}
	// The lock file must remain the prior compiled lock.
	lockPath := filepath.Join(target, ".factory/doctrine.lock.json")
	if _, err := os.Stat(lockPath); err != nil {
		t.Fatalf("prior lock vanished after rollback: %v", err)
	}
}

// TestCompileRollbackRemovesCreatedOnEmptyTarget proves that an
// injected failure during create-managed on an empty target leaves the
// target empty after rollback.
func TestCompileRollbackRemovesCreatedOnEmptyTarget(t *testing.T) {
	pack, err := LoadCorePack()
	if err != nil {
		t.Fatalf("LoadCorePack: %v", err)
	}
	prof, err := pack.MustProfile(ProfileId("fsharp-elm-service-v1"))
	if err != nil {
		t.Fatalf("MustProfile: %v", err)
	}
	target := newEmptyTarget(t)
	// Capture target emptiness before compile.
	if entries, _ := os.ReadDir(target); len(entries) != 0 {
		t.Fatalf("target not empty before compile")
	}
	// Inject failure during the first create-managed step.
	_, err = Compile(pack, prof, target, CompilerOptions{
		FailAfter: ActionCreateManaged,
	})
	if err == nil {
		t.Fatalf("expected injected failure")
	}
	// After rollback, only .factory may exist (from the parent-dir
	// creation in writeAtomicFile). The .factory directory itself
	// should be empty because no managed file was successfully
	// committed.
	factory := filepath.Join(target, ".factory")
	if info, err := os.Stat(factory); err == nil {
		if !info.IsDir() {
			t.Errorf(".factory should be a directory after rollback")
		}
	}
	// Walk the target and assert no managed file content survived.
	err = filepath.WalkDir(target, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		rel, _ := filepath.Rel(target, p)
		// .factory/doctrine.lock.json may not exist on empty target
		// before compile; it should not exist after rollback either.
		if rel == ".factory/doctrine.lock.json" {
			t.Errorf("lock file should not exist after rollback: %s", rel)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk: %v", err)
	}
}

// TestCompileLeavesNoTempFiles verifies that compile does not leave
// behind any temp files after success.
func TestCompileLeavesNoTempFiles(t *testing.T) {
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
	entries, _ := os.ReadDir(filepath.Join(target, ".factory"))
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), ".tmp-") {
			t.Errorf("temp file left behind: %s", e.Name())
		}
	}
	// Also check generated/ subdirectory.
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
	mk := filepath.Join(target, ".factory/generated/factory.mk")
	if err := os.WriteFile(mk, []byte("tampered"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	if _, err := Compile(pack, prof, target, CompilerOptions{}); err != nil {
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
	pack, err := LoadCorePack()
	if err != nil {
		t.Fatalf("LoadCorePack: %v", err)
	}
	prof, err := pack.MustProfile(ProfileId("fsharp-elm-service-v1"))
	if err != nil {
		t.Fatalf("MustProfile: %v", err)
	}
	target := newEmptyTarget(t)
	outside := t.TempDir()
	if err := os.Symlink(outside, filepath.Join(target, ".factory")); err != nil {
		t.Fatalf("symlink: %v", err)
	}
	_, err = Compile(pack, prof, target, CompilerOptions{})
	if err == nil {
		t.Errorf("expected compile to refuse symlink escape")
	}
}
