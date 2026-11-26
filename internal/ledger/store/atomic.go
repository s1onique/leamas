package store

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

// WriteFileAtomic writes data to path via a temp file plus rename.
//
// The destination is replaced atomically: a partial write can never leave
// a half-formed destination behind. The temp file lives in the same
// directory as the destination so the rename is atomic on every supported
// platform.
func WriteFileAtomic(path string, data []byte) error {
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
	tmp, err := os.CreateTemp(dir, ".ledger-*.tmp")
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
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		cleanup()
		return fmt.Errorf("WriteFileAtomic: fsync temp: %w", err)
	}
	if err := tmp.Close(); err != nil {
		cleanup()
		return fmt.Errorf("WriteFileAtomic: close temp: %w", err)
	}
	if err := os.Chmod(tmpPath, 0o644); err != nil {
		cleanup()
		return fmt.Errorf("WriteFileAtomic: chmod temp: %w", err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		cleanup()
		return fmt.Errorf("WriteFileAtomic: rename temp: %w", err)
	}
	return syncDirBestEffort(dir)
}

// syncDirBestEffort opens dir and syncs it. On Windows directory
// durability is not exposed via the standard library, so the call is a
// no-op there.
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
