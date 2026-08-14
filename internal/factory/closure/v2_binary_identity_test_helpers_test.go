// SPDX-License-Identifier: Apache-2.0

package closure

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// newV2TestBinaryIdentity returns a complete deterministic fake whose path and
// digest are runtime-verifiable by the production identity validator.
//
// Note: on macOS, /var -> /private/var causes filepath.EvalSymlinks to resolve
// to a different path. We canonicalize the path so it passes the production
// "must already be symlink-resolved" check.
func newV2TestBinaryIdentity(t testing.TB) V2BinaryIdentity {
	t.Helper()
	path := filepath.Join(t.TempDir(), "leamas-test-binary")
	data := []byte("deterministic fake leamas binary identity\n")
	if err := os.WriteFile(path, data, 0o700); err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(data)

	// Canonicalize path for macOS compatibility: /var -> /private/var
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil {
		t.Fatalf("cannot resolve symlinks for test binary: %v", err)
	}
	resolved = filepath.Clean(resolved)

	return V2BinaryIdentity{
		Path:          resolved,
		SHA256:        hex.EncodeToString(sum[:]),
		VCSRevision:   strings.Repeat("7", 40),
		VCSModified:   false,
		LeamasVersion: "0.1.0+test",
	}
}

func runClosureProtocolV2ForTest(t testing.TB, ctx context.Context, req V2Request) (V2Manifest, error) {
	t.Helper()
	return RunClosureProtocolV2WithBinary(ctx, req, newV2TestBinaryIdentity(t))
}
