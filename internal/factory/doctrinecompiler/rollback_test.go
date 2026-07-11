package doctrinecompiler

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

// treeSnapshot captures a deterministic snapshot of a target's
// contents. It is reused across rollback tests so each test asserts
// against the same structural shape (relative paths, entry kinds,
// file bytes, and permission bits where the platform supports them).
type treeSnapshot struct {
	files map[string]treeFile
}

type treeFile struct {
	kind  string // "file" or "symlink"
	mode  os.FileMode
	bytes []byte
}

// snapshotTreeBytes walks target and records the full pre-compile
// state for later comparison.
func snapshotTreeBytes(target string) (*treeSnapshot, error) {
	out := &treeSnapshot{files: make(map[string]treeFile)}
	err := filepath.WalkDir(target, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		rel, _ := filepath.Rel(target, p)
		if rel == "." {
			return nil
		}
		rel = filepath.ToSlash(rel)
		info, ierr := d.Info()
		if ierr != nil {
			return ierr
		}
		if info.Mode()&os.ModeSymlink != 0 {
			out.files[rel] = treeFile{kind: "symlink"}
			return nil
		}
		data, rerr := os.ReadFile(p)
		if rerr != nil {
			return rerr
		}
		out.files[rel] = treeFile{
			kind:  "file",
			mode:  info.Mode().Perm(),
			bytes: data,
		}
		return nil
	})
	return out, err
}

// equal reports whether two snapshots are byte- and mode-identical.
func (s *treeSnapshot) equal(other *treeSnapshot) bool {
	if len(s.files) != len(other.files) {
		return false
	}
	for k, a := range s.files {
		b, ok := other.files[k]
		if !ok {
			return false
		}
		if a.kind != b.kind {
			return false
		}
		if a.kind == "file" {
			if a.mode != b.mode {
				return false
			}
			if string(a.bytes) != string(b.bytes) {
				return false
			}
		}
	}
	return true
}

// TestCompileRollbackFailsClosedAfterMutation proves that an injected
// mid-apply failure aborts the compile and restores the pre-compile
// state exactly (contents + modes + path set), leaving the previous
// lock in place. The fault is injected AFTER at least one mutation
// has succeeded via FailAfterN=1, and the rollback path runs cleanly.
func TestCompileRollbackFailsClosedAfterMutation(t *testing.T) {
	pack, prof := freshPackProfile(t)
	target := newEmptyTarget(t)
	if _, err := Compile(pack, prof, target, CompilerOptions{CompilerVersion: "0.1.0"}); err != nil {
		t.Fatalf("initial Compile: %v", err)
	}
	mk := filepath.Join(target, ".factory/generated/factory.mk")
	original, err := os.ReadFile(mk)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	tampered := append(original, []byte("\n# TAMPERED\n")...)
	if err := os.WriteFile(mk, tampered, 0o600); err != nil {
		t.Fatalf("write tampered: %v", err)
	}
	preMode := modeOf(t, mk)
	before, err := snapshotTreeBytes(target)
	if err != nil {
		t.Fatalf("snapshot before: %v", err)
	}
	_, err = Compile(pack, prof, target, CompilerOptions{
		CompilerVersion: "0.1.0",
		FailAfterN:      1,
	})
	if err == nil {
		t.Fatalf("expected injected failure to abort compile")
	}
	if !errors.Is(err, ErrApplyFailed) {
		t.Errorf("returned error is not ErrApplyFailed: %v", err)
	}
	if errors.Is(err, ErrRollbackFailed) {
		t.Errorf("rollback should have succeeded but ErrRollbackFailed is present: %v", err)
	}
	after, err := snapshotTreeBytes(target)
	if err != nil {
		t.Fatalf("snapshot after: %v", err)
	}
	if !before.equal(after) {
		t.Errorf("rollback did not restore pre-compile state byte-for-byte")
	}
	if got := modeOf(t, mk); got != preMode {
		t.Errorf("rollback did not restore mode: got=%v want=%v", got, preMode)
	}
	lockPath := filepath.Join(target, ".factory/doctrine.lock.json")
	if _, err := os.Stat(lockPath); err != nil {
		t.Fatalf("prior lock vanished after rollback: %v", err)
	}
}

// TestCompileRollbackEmptyTargetCleanup proves that an injected
// failure on an empty target leaves no transaction-created files or
// non-root directories behind.
func TestCompileRollbackEmptyTargetCleanup(t *testing.T) {
	pack, prof := freshPackProfile(t)
	target := newEmptyTarget(t)
	entries, _ := os.ReadDir(target)
	if len(entries) != 0 {
		t.Fatalf("target not empty before compile")
	}
	_, err := Compile(pack, prof, target, CompilerOptions{
		CompilerVersion: "0.1.0",
		FailAfterN:      1,
	})
	if err == nil {
		t.Fatalf("expected injected failure")
	}
	if !errors.Is(err, ErrApplyFailed) {
		t.Errorf("returned error is not ErrApplyFailed: %v", err)
	}
	leftover, _ := os.ReadDir(target)
	for _, e := range leftover {
		t.Errorf("rollback left child in target root: %s", e.Name())
	}
}

// TestCompileRollbackRestoresRemovedManaged verifies that a managed
// file removed by the apply is restored when a later mutation fails.
func TestCompileRollbackRestoresRemovedManaged(t *testing.T) {
	pack, prof := freshPackProfile(t)
	target := newEmptyTarget(t)
	if _, err := Compile(pack, prof, target, CompilerOptions{CompilerVersion: "0.1.0"}); err != nil {
		t.Fatalf("initial Compile: %v", err)
	}
	plant := filepath.Join(target, ".factory/generated/obsolete.txt")
	original := []byte("user-managed")
	if err := os.WriteFile(plant, original, 0o600); err != nil {
		t.Fatalf("plant: %v", err)
	}
	lockPath := filepath.Join(target, ".factory/doctrine.lock.json")
	data, err := os.ReadFile(lockPath)
	if err != nil {
		t.Fatalf("lock: %v", err)
	}
	patched, err := patchLockAddManaged(string(data),
		".factory/generated/obsolete.txt",
		"0000000000000000000000000000000000000000000000000000000000000000")
	if err != nil {
		t.Fatalf("patch lock: %v", err)
	}
	if err := os.WriteFile(lockPath, []byte(patched), 0o644); err != nil {
		t.Fatalf("write lock: %v", err)
	}
	pre, err := snapshotTreeBytes(target)
	if err != nil {
		t.Fatalf("snapshot pre: %v", err)
	}
	_, err = Compile(pack, prof, target, CompilerOptions{
		CompilerVersion: "0.1.0",
		FailAfterN:      1,
	})
	if err == nil {
		t.Fatalf("expected injected failure")
	}
	if !errors.Is(err, ErrApplyFailed) {
		t.Errorf("returned error is not ErrApplyFailed: %v", err)
	}
	if _, err := os.Stat(plant); err != nil {
		t.Errorf("removed managed file was not restored: %v", err)
	}
	got, _ := os.ReadFile(plant)
	if string(got) != string(original) {
		t.Errorf("removed managed file contents not restored")
	}
	post, err := snapshotTreeBytes(target)
	if err != nil {
		t.Fatalf("snapshot post: %v", err)
	}
	if !pre.equal(post) {
		t.Errorf("tree differs after rollback")
	}
}

// TestCompileRollbackPreservesSeeded verifies a user-modified seeded
// file is preserved when a later managed mutation fails.
func TestCompileRollbackPreservesSeeded(t *testing.T) {
	pack, prof := freshPackProfile(t)
	target := newEmptyTarget(t)
	if _, err := Compile(pack, prof, target, CompilerOptions{CompilerVersion: "0.1.0"}); err != nil {
		t.Fatalf("initial Compile: %v", err)
	}
	mk := filepath.Join(target, "Makefile")
	custom := "# user override\ngate: factorize custom-step\n"
	if err := os.WriteFile(mk, []byte(custom), 0o600); err != nil {
		t.Fatalf("write seeded: %v", err)
	}
	preMode := modeOf(t, mk)
	doc := filepath.Join(target, ".factory/generated/doctrine-inventory.md")
	if err := os.WriteFile(doc, []byte("tampered"), 0o644); err != nil {
		t.Fatalf("tamper: %v", err)
	}
	_, err := Compile(pack, prof, target, CompilerOptions{
		CompilerVersion: "0.1.0",
		FailAfterN:      1,
	})
	if err == nil {
		t.Fatalf("expected injected failure")
	}
	if !errors.Is(err, ErrApplyFailed) {
		t.Errorf("returned error is not ErrApplyFailed: %v", err)
	}
	got, _ := os.ReadFile(mk)
	if string(got) != custom {
		t.Errorf("seeded Makefile altered: got %q want %q", got, custom)
	}
	if modeOf(t, mk) != preMode {
		t.Errorf("seeded Makefile mode altered")
	}
}

// TestCompileSuccessDeterministicNoInjection verifies a clean compile
// without injection produces the deterministic final projection.
func TestCompileSuccessDeterministicNoInjection(t *testing.T) {
	pack, prof := freshPackProfile(t)
	target := newEmptyTarget(t)
	if _, err := Compile(pack, prof, target, CompilerOptions{CompilerVersion: "0.1.0"}); err != nil {
		t.Fatalf("Compile: %v", err)
	}
	want := []string{
		".factory/doctrine.lock.json",
		".factory/generated/doctrine-inventory.md",
		".factory/generated/factory.mk",
		".factory/project.json",
		"Makefile",
		"docs/factory/README.md",
	}
	got := listTargetTree(t, target)
	if !equalStringSet(got, want) {
		t.Errorf("tree mismatch: got=%v want=%v", got, want)
	}
}
