package doctrinecompiler

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// fsOpsTestSentinels are sentinel errors used by the fsOps-seam
// tests below. Each test asserts that the returned compile error
// chain exposes the relevant sentinel via errors.Is.
var (
	errFsSimLstat     = errors.New("simulated Lstat failure")
	errFsSimReadDir   = errors.New("simulated ReadDir failure")
	errFsSimRemove    = errors.New("simulated Remove failure")
	errFsSimMkdirAll  = errors.New("simulated MkdirAll failure")
	errFsSimWriteFile = errors.New("simulated WriteFile failure")
)

// swapRuntimeFsOps replaces runtimeFsOps for the duration of a test
// and returns a restore function. Tests use this hook to inject
// deterministic failures into the rollback path. Production code
// must not call this.
func swapRuntimeFsOps(ops fsOps) func() {
	prev := runtimeFsOps
	runtimeFsOps = ops
	return func() { runtimeFsOps = prev }
}

// withRuntimeFsOps temporarily replaces runtimeFsOps for the
// duration of t and restores it on cleanup.
func withRuntimeFsOps(t *testing.T, ops fsOps) {
	t.Helper()
	restore := swapRuntimeFsOps(ops)
	t.Cleanup(restore)
}

// fsOpsOverridingOne returns an fsOps that wraps runtimeFsOps but
// replaces a single function with the supplied replacement. This is
// how the seam-coverage tests exercise a specific syscall in
// isolation while keeping the rest of the rollback path real.
func fsOpsOverridingOne(target string, override func(fsOps) fsOps) fsOps {
	// The override is constructed by the caller from runtimeFsOps
	// and a chosen per-call replacement; this helper exists so the
	// call sites are uniform.
	return override(runtimeFsOps)
}

// runCompileWithFailAfterN drives Compile with FailAfterN=1 and a
// configured fsOps replacement. The target is a fresh tempdir.
func runCompileWithFailAfterN(t *testing.T, ops fsOps) error {
	t.Helper()
	pack, prof := freshPackProfile(t)
	target := newEmptyTarget(t)
	withRuntimeFsOps(t, ops)
	_, err := Compile(pack, prof, target, CompilerOptions{
		CompilerVersion: "0.1.0",
		FailAfterN:      1,
	})
	return err
}

// TestRollbackApplyOnlyFailure asserts the no-rollback-failure
// path: the apply fails, the rollback runs cleanly with the
// production fsOps, and the returned error wraps ErrApplyFailed
// but not ErrRollbackFailed.
func TestRollbackApplyOnlyFailure(t *testing.T) {
	err := runCompileWithFailAfterN(t, runtimeFsOps)
	if err == nil {
		t.Fatalf("expected injected failure")
	}
	if !errors.Is(err, ErrApplyFailed) {
		t.Errorf("errors.Is(err, ErrApplyFailed) is false: %v", err)
	}
	if errors.Is(err, ErrRollbackFailed) {
		t.Errorf("rollback should have succeeded: %v", err)
	}
}

// TestRollbackLstatFailureJoined proves that a deterministic
// Lstat failure during created-file rollback is joined into the
// rollback error chain.
func TestRollbackLstatFailureJoined(t *testing.T) {
	ops := fsOpsOverridingOne("", func(base fsOps) fsOps {
		return fsOps{
			Lstat: func(path string) (os.FileInfo, error) {
				return nil, errFsSimLstat
			},
			ReadDir:   base.ReadDir,
			Remove:    base.Remove,
			MkdirAll:  base.MkdirAll,
			WriteFile: base.WriteFile,
		}
	})
	err := runCompileWithFailAfterN(t, ops)
	if err == nil {
		t.Fatalf("expected injected failure")
	}
	if !errors.Is(err, ErrApplyFailed) {
		t.Errorf("errors.Is(err, ErrApplyFailed) is false: %v", err)
	}
	if !errors.Is(err, ErrRollbackFailed) {
		t.Errorf("errors.Is(err, ErrRollbackFailed) is false: %v", err)
	}
	if !errors.Is(err, errFsSimLstat) {
		t.Errorf("errors.Is(err, errFsSimLstat) is false: %v", err)
	}
}

// TestRollbackRemoveFailureJoined proves that a deterministic
// Remove failure during created-file rollback is joined into the
// rollback error chain.
func TestRollbackRemoveFailureJoined(t *testing.T) {
	ops := fsOpsOverridingOne("", func(base fsOps) fsOps {
		return fsOps{
			Lstat: base.Lstat,
			ReadDir: func(path string) ([]os.DirEntry, error) {
				// The created-file rollback Remove needs the
				// Lstat-first check to succeed. Forward Lstat so
				// the Remove path is exercised.
				if _, err := base.Lstat(path); err != nil {
					return nil, err
				}
				return []os.DirEntry{}, nil
			},
			Remove: func(path string) error {
				return errFsSimRemove
			},
			MkdirAll:  base.MkdirAll,
			WriteFile: base.WriteFile,
		}
	})
	err := runCompileWithFailAfterN(t, ops)
	if err == nil {
		t.Fatalf("expected injected failure")
	}
	if !errors.Is(err, ErrApplyFailed) {
		t.Errorf("errors.Is(err, ErrApplyFailed) is false: %v", err)
	}
	if !errors.Is(err, ErrRollbackFailed) {
		t.Errorf("errors.Is(err, ErrRollbackFailed) is false: %v", err)
	}
	if !errors.Is(err, errFsSimRemove) {
		t.Errorf("errors.Is(err, errFsSimRemove) is false: %v", err)
	}
}

// TestRollbackLstatErrNotExistIgnored proves that rollback
// directory cleanup tolerates fs.ErrNotExist from Lstat: the
// returned error must wrap ErrApplyFailed only.
func TestRollbackLstatErrNotExistIgnored(t *testing.T) {
	ops := fsOpsOverridingOne("", func(base fsOps) fsOps {
		return fsOps{
			Lstat: func(path string) (os.FileInfo, error) {
				return nil, fs.ErrNotExist
			},
			ReadDir:   base.ReadDir,
			Remove:    base.Remove,
			MkdirAll:  base.MkdirAll,
			WriteFile: base.WriteFile,
		}
	})
	err := runCompileWithFailAfterN(t, ops)
	if err == nil {
		t.Fatalf("expected injected failure")
	}
	if !errors.Is(err, ErrApplyFailed) {
		t.Errorf("errors.Is(err, ErrApplyFailed) is false: %v", err)
	}
	if errors.Is(err, ErrRollbackFailed) {
		t.Errorf("fs.ErrNotExist must be ignored: %v", err)
	}
}

// TestRollbackReadDirFailureJoined proves that a deterministic
// ReadDir failure during directory-cleanup rollback is joined.
func TestRollbackReadDirFailureJoined(t *testing.T) {
	ops := fsOpsOverridingOne("", func(base fsOps) fsOps {
		return fsOps{
			Lstat: base.Lstat,
			ReadDir: func(path string) ([]os.DirEntry, error) {
				return nil, errFsSimReadDir
			},
			Remove:    base.Remove,
			MkdirAll:  base.MkdirAll,
			WriteFile: base.WriteFile,
		}
	})
	err := runCompileWithFailAfterN(t, ops)
	if err == nil {
		t.Fatalf("expected injected failure")
	}
	if !errors.Is(err, ErrApplyFailed) {
		t.Errorf("errors.Is(err, ErrApplyFailed) is false: %v", err)
	}
	if !errors.Is(err, ErrRollbackFailed) {
		t.Errorf("errors.Is(err, ErrRollbackFailed) is false: %v", err)
	}
	if !errors.Is(err, errFsSimReadDir) {
		t.Errorf("errors.Is(err, errFsSimReadDir) is false: %v", err)
	}
}

// TestRollbackMkdirAllFailureJoined proves that a deterministic
// MkdirAll failure during mutReplace/mutRemove recreation is
// joined. MutRemove and mutReplace use the seam's MkdirAll to
// re-create the parent directory before writing the prior bytes.
func TestRollbackMkdirAllFailureJoined(t *testing.T) {
	ops := fsOpsOverridingOne("", func(base fsOps) fsOps {
		return fsOps{
			Lstat:     base.Lstat,
			ReadDir:   base.ReadDir,
			Remove:    base.Remove,
			MkdirAll:  func(path string, mode os.FileMode) error { return errFsSimMkdirAll },
			WriteFile: base.WriteFile,
		}
	})
	// We need an existing file that the apply would remove so the
	// rollback tries to recreate it.
	pack, prof := freshPackProfile(t)
	target := newEmptyTarget(t)
	withRuntimeFsOps(t, ops)
	if _, err := Compile(pack, prof, target, CompilerOptions{CompilerVersion: "0.1.0"}); err != nil {
		t.Fatalf("initial Compile: %v", err)
	}
	plant := filepath.Join(target, ".factory/generated/obsolete.txt")
	if err := os.WriteFile(plant, []byte("orig"), 0o644); err != nil {
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
	_, err = Compile(pack, prof, target, CompilerOptions{
		CompilerVersion: "0.1.0",
		FailAfterN:      1,
	})
	if err == nil {
		t.Fatalf("expected injected failure")
	}
	if !errors.Is(err, ErrApplyFailed) {
		t.Errorf("errors.Is(err, ErrApplyFailed) is false: %v", err)
	}
	if !errors.Is(err, ErrRollbackFailed) {
		t.Errorf("errors.Is(err, ErrRollbackFailed) is false: %v", err)
	}
	if !errors.Is(err, errFsSimMkdirAll) {
		t.Errorf("errors.Is(err, errFsSimMkdirAll) is false: %v", err)
	}
}

// TestRollbackWriteFileFailureJoined proves that a deterministic
// WriteFile failure during mutReplace/mutRemove recreation is
// joined. The file already exists, so MkdirAll on its existing
// parent succeeds and the WriteFile replacement fails.
func TestRollbackWriteFileFailureJoined(t *testing.T) {
	ops := fsOpsOverridingOne("", func(base fsOps) fsOps {
		return fsOps{
			Lstat:    base.Lstat,
			ReadDir:  base.ReadDir,
			Remove:   base.Remove,
			MkdirAll: base.MkdirAll,
			WriteFile: func(path string, data []byte, mode os.FileMode) (string, error) {
				return "", errFsSimWriteFile
			},
		}
	})
	pack, prof := freshPackProfile(t)
	target := newEmptyTarget(t)
	withRuntimeFsOps(t, ops)
	if _, err := Compile(pack, prof, target, CompilerOptions{CompilerVersion: "0.1.0"}); err != nil {
		t.Fatalf("initial Compile: %v", err)
	}
	plant := filepath.Join(target, ".factory/generated/obsolete.txt")
	if err := os.WriteFile(plant, []byte("orig"), 0o644); err != nil {
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
	_, err = Compile(pack, prof, target, CompilerOptions{
		CompilerVersion: "0.1.0",
		FailAfterN:      1,
	})
	if err == nil {
		t.Fatalf("expected injected failure")
	}
	if !errors.Is(err, ErrApplyFailed) {
		t.Errorf("errors.Is(err, ErrApplyFailed) is false: %v", err)
	}
	if !errors.Is(err, ErrRollbackFailed) {
		t.Errorf("errors.Is(err, ErrRollbackFailed) is false: %v", err)
	}
	if !errors.Is(err, errFsSimWriteFile) {
		t.Errorf("errors.Is(err, errFsSimWriteFile) is false: %v", err)
	}
}

// TestRollbackErrorMessage declares restoration incomplete when
// rollback fails. This guards the human-readable failure signal.
func TestRollbackErrorMessage(t *testing.T) {
	ops := fsOpsOverridingOne("", func(base fsOps) fsOps {
		return fsOps{
			Lstat: func(path string) (os.FileInfo, error) {
				return nil, errFsSimLstat
			},
			ReadDir:   base.ReadDir,
			Remove:    base.Remove,
			MkdirAll:  base.MkdirAll,
			WriteFile: base.WriteFile,
		}
	})
	err := runCompileWithFailAfterN(t, ops)
	if err == nil {
		t.Fatalf("expected injected failure")
	}
	if !strings.Contains(err.Error(), "INCOMPLETE") {
		t.Errorf("error message does not declare restoration incomplete: %v", err)
	}
}
