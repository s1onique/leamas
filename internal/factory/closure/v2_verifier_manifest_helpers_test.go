// SPDX-License-Identifier: Apache-2.0

package closure

// v2_verifier_manifest_helpers_test.go provides tiny
// helpers used only by the v2 verifier ACT 3 tests.
//
// The helpers are intentionally minimal: each one wraps a
// single stdlib call so the test bodies stay focused on the
// verification contract under test.

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
)

// sha256Sum returns the lowercase hex SHA-256 digest of the
// supplied bytes.
func sha256Sum(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

// osMkdirAll creates the directory of the supplied path.
// The path may be absolute or relative to the process CWD.
func osMkdirAll(path string) error {
	return os.MkdirAll(filepath.Dir(path), 0o755)
}

// osWriteFile writes the supplied bytes to the supplied
// path with mode 0o644. The directory of the path must
// already exist.
func osWriteFile(path string, content []byte) error {
	return os.WriteFile(path, content, 0o644)
}
