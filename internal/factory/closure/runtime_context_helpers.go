// SPDX-License-Identifier: Apache-2.0

// Package closure - runtime_context_helpers.go contains the small
// filesystem and SHA-1 helpers used by runtime_context_resolver.go
// and the binary authority code path.
//
// The helpers are kept in a dedicated file so the resolver stays
// focused on identity resolution and so the binary builder can
// reuse the same primitives without circular imports.

package closure

import (
	"crypto/sha1"
	"encoding/hex"
	"os"
	"path/filepath"
)

// writeTempBytes writes data to a freshly created temp file under
// the system temp directory and returns the absolute path. The
// caller is responsible for calling removeTemp on the returned
// path. A failure to create the file produces an error so the
// resolver can fall back to localSHA1Hex.
func writeTempBytes(data []byte) (string, error) {
	file, err := os.CreateTemp("", "leamas-runtime-*.bin")
	if err != nil {
		return "", err
	}
	path := file.Name()
	if _, err := file.Write(data); err != nil {
		_ = file.Close()
		_ = os.Remove(path)
		return "", err
	}
	if err := file.Close(); err != nil {
		_ = os.Remove(path)
		return "", err
	}
	return path, nil
}

// removeTemp removes the supplied temp path. Errors are ignored
// because the caller has no useful recovery path.
func removeTemp(path string) {
	if path == "" {
		return
	}
	if dir := filepath.Dir(path); dir != "" && dir != "." {
		_ = os.Remove(path)
	}
}

// localSHA1Hex computes the SHA-1 of the supplied bytes locally.
// It is the deterministic fallback when git hash-object is
// unavailable. The encoding matches `git hash-object`: lowercase
// hex of the SHA-1 digest.
func localSHA1Hex(data []byte) string {
	sum := sha1.Sum(data)
	return hex.EncodeToString(sum[:])
}
