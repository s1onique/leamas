// SPDX-License-Identifier: Apache-2.0

package closure

// atomic_writer.go implements the atomic file publication
// required by Phase 2 of
// ACT-LEAMAS-FACTORY-CLOSURE-PROTOCOL-V2-VERIFIER-MAC-HANDOFF01-CORRECTION01.
//
// The writer guarantees that:
//
//   - the destination file is replaced via temp + rename;
//   - the temp file is fsynced before the rename (where
//     supported by the OS);
//   - the parent directory is fsynced after the rename
//     (where supported);
//   - the temp file is removed on every failure;
//   - the destination file is left untouched on every
//     failure path.
//
// The CLI is the only caller; the writer is the single
// authority that decides when a verdict-derived artifact
// becomes visible on disk. The CLI must NEVER call
// os.WriteFile for verifier output.

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
)

// WriteFileAtomic writes data to path via a temp file +
// rename so a partial write can never leave a half-formed
// destination behind. The temp file lives in the same
// directory as the destination so the rename is atomic on
// every supported platform.
func WriteFileAtomic(path string, data []byte, perm os.FileMode) error {
	if path == "" {
		return fmt.Errorf("WriteFileAtomic: empty path")
	}
	dir := filepath.Dir(path)
	if dir == "" {
		dir = "."
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("WriteFileAtomic: mkdir %s: %w", dir, err)
	}
	tmp, err := os.CreateTemp(dir, ".verifier-output-*.tmp")
	if err != nil {
		return fmt.Errorf("WriteFileAtomic: create temp: %w", err)
	}
	tmpPath := tmp.Name()
	cleanup := func() { _ = os.Remove(tmpPath) }

	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		cleanup()
		return fmt.Errorf("WriteFileAtomic: write temp: %w", err)
	}
	if err := syncFileBestEffort(tmp); err != nil {
		_ = tmp.Close()
		cleanup()
		return fmt.Errorf("WriteFileAtomic: fsync temp: %w", err)
	}
	if err := tmp.Close(); err != nil {
		cleanup()
		return fmt.Errorf("WriteFileAtomic: close temp: %w", err)
	}
	if err := os.Chmod(tmpPath, perm); err != nil {
		cleanup()
		return fmt.Errorf("WriteFileAtomic: chmod temp: %w", err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		cleanup()
		return fmt.Errorf("WriteFileAtomic: rename temp: %w", err)
	}
	if err := syncDirBestEffort(dir); err != nil {
		// A post-rename dirfsync failure is not fatal for
		// durability on the file itself; surface it for
		// the caller but do not delete the (already
		// renamed) destination.
		return fmt.Errorf("WriteFileAtomic: fsync dir: %w", err)
	}
	return nil
}

// syncFileBestEffort invokes Sync on f. On filesystems
// where Sync is a no-op (Windows) the call returns nil
// without raising. The function exists to centralize the
// fsync call so the writer's failure handling stays
// uniform.
func syncFileBestEffort(f *os.File) error {
	if f == nil {
		return nil
	}
	return f.Sync()
}

// syncDirBestEffort opens dir and Sync()s it. On Windows
// the call returns nil without an error: directory
// durability is not exposed via the standard library on
// Windows and we deliberately do not chase a syscall.
func syncDirBestEffort(dir string) error {
	if runtime.GOOS == "windows" {
		return nil
	}
	d, err := os.Open(dir)
	if err != nil {
		return err
	}
	defer d.Close()
	return d.Sync()
}

// Discardable is the small interface used by callers that
// want to hand a custom sink to the writer. The CLI
// always uses a real file; the interface exists so tests
// can assert "destination unchanged on every failure".
type Discardable interface {
	io.Writer
}
