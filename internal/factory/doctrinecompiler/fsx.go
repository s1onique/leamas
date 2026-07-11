package doctrinecompiler

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// readFS reads the file at path and returns its bytes.
func readFS(path string) ([]byte, error) {
	return os.ReadFile(path)
}

// writeAtomicFile replaces path with data via a same-directory temp
// file and rename. It returns the temp path on failure so callers can
// clean it up.
//
// The function refuses to overwrite a directory or to follow a symlink
// at the destination. It fails closed if the rename would cross a
// filesystem boundary, leaving the original file (if any) untouched.
func writeAtomicFile(path string, data []byte, mode os.FileMode) (tmpPath string, err error) {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("mkdir %s: %w", dir, err)
	}
	// Refuse to overwrite a directory.
	lst, err := os.Lstat(path)
	switch {
	case err == nil:
		m := lst.Mode()
		if m.IsDir() {
			return "", fmt.Errorf("refusing to overwrite directory: %s", path)
		}
		if m&os.ModeSymlink != 0 {
			return "", fmt.Errorf("refusing to overwrite symlink: %s", path)
		}
	case errors.Is(err, fs.ErrNotExist):
		// OK.
	default:
		return "", fmt.Errorf("lstat %s: %w", path, err)
	}
	// Create temp file in the same directory.
	tmp, err := os.CreateTemp(dir, ".tmp-doctrine-*")
	if err != nil {
		return "", fmt.Errorf("create temp: %w", err)
	}
	tmpName := tmp.Name()
	// Best-effort cleanup on any failure path.
	cleanup := func() {
		_ = os.Remove(tmpName)
	}
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		cleanup()
		return tmpName, fmt.Errorf("write temp: %w", err)
	}
	if err := tmp.Chmod(mode); err != nil {
		_ = tmp.Close()
		cleanup()
		return tmpName, fmt.Errorf("chmod temp: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		cleanup()
		return tmpName, fmt.Errorf("sync temp: %w", err)
	}
	if err := tmp.Close(); err != nil {
		cleanup()
		return tmpName, fmt.Errorf("close temp: %w", err)
	}
	if err := os.Rename(tmpName, path); err != nil {
		cleanup()
		return tmpName, fmt.Errorf("rename temp: %w", err)
	}
	return "", nil
}

// removeFileIfExists deletes path if it is a regular file.
//
// Refuses to follow symlinks or to remove directories. Missing files
// are not an error.
func removeFileIfExists(path string) error {
	lst, err := os.Lstat(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("lstat %s: %w", path, err)
	}
	m := lst.Mode()
	if m&os.ModeSymlink != 0 {
		return fmt.Errorf("refusing to remove symlink: %s", path)
	}
	if m.IsDir() {
		return fmt.Errorf("refusing to remove directory: %s", path)
	}
	if err := os.Remove(path); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("remove %s: %w", path, err)
	}
	return nil
}

// ensureParentDir creates the parent directory of path if missing.
func ensureParentDir(path string) error {
	return os.MkdirAll(filepath.Dir(path), 0o755)
}
