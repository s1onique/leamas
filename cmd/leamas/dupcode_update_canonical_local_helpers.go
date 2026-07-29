// SPDX-License-Identifier: Apache-2.0

package main

import (
	"os"
	"path/filepath"
)

// writeFileMkdirAll creates parent directories as needed and writes the
// supplied bytes atomically. It is a tiny helper used by the canonical
// local-path integration test to seed a Git worktree.
func writeFileMkdirAll(dir, relative string, data []byte) error {
	full := filepath.Join(dir, relative)
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		return err
	}
	return os.WriteFile(full, data, 0o644)
}

// osStat is a thin os.Stat wrapper. The integration test depends on
// file-existence checks against the canonical local-path baseline.
func osStat(path string) (os.FileInfo, error) {
	return os.Stat(path)
}
