// SPDX-License-Identifier: Apache-2.0

package main

import (
	"os"
	"path/filepath"
)

// writeFileMkdirAll is a tiny test helper that ensures the parent
// directory exists before writing a file. It exists solely for the
// canonical local-path integration test; production code does not use
// it. Same-package tests in cmd/leamas can call it directly.
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
