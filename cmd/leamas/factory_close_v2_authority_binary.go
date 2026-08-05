// SPDX-License-Identifier: Apache-2.0

package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"

	"github.com/s1onique/leamas/internal/factory/closure"
)

// captureRunningBinaryIdentity reads the absolute path of the
// current leamas binary, resolves any symlink chain, and
// computes its SHA-256. It returns the exact file the OS is
// executing, which is the identity the manifest records.
//
// The function never returns a partial identity: on any failure
// it returns the error and an empty identity so the CLI can
// emit a typed binary_identity_invalid envelope.
func captureRunningBinaryIdentity() (closure.V2BinaryIdentity, error) {
	exe, err := os.Executable()
	if err != nil {
		return closure.V2BinaryIdentity{}, fmt.Errorf("locate leamas binary: %w", err)
	}
	resolved, err := filepath.EvalSymlinks(exe)
	if err != nil {
		return closure.V2BinaryIdentity{}, fmt.Errorf("resolve leamas binary symlinks: %w", err)
	}
	abs, err := filepath.Abs(resolved)
	if err != nil {
		return closure.V2BinaryIdentity{}, fmt.Errorf("make binary path absolute: %w", err)
	}
	abs = filepath.Clean(abs)
	data, err := os.ReadFile(abs)
	if err != nil {
		return closure.V2BinaryIdentity{}, fmt.Errorf("read leamas binary: %w", err)
	}
	sum := sha256.Sum256(data)
	revision := closure.RunningLeamasVCSRevision()
	version := closure.RunningLeamasVersion()
	return closure.V2BinaryIdentity{
		Path:          abs,
		SHA256:        hex.EncodeToString(sum[:]),
		VCSRevision:   revision,
		VCSModified:   closure.RunningLeamasVCSModified(),
		LeamasVersion: version,
	}, nil
}
