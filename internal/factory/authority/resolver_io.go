// SPDX-License-Identifier: Apache-2.0

// Package authority: resolver_io.go provides the small filesystem
// and execution helpers consumed by resolver.go. The helpers are
// kept in a dedicated file so the typed-status logic in
// resolver.go remains easy to read.
package authority

import (
	"fmt"
	"io"
	"os"
)

// readFileBytes returns the full contents of path as a byte slice.
// It is intentionally tiny: the resolver only needs to hash a
// tool binary and to read a JSON manifest from disk in tests.
func readFileBytes(path string) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", path, err)
	}
	defer f.Close()
	stat, err := f.Stat()
	if err != nil {
		return nil, fmt.Errorf("stat %s: %w", path, err)
	}
	if !stat.Mode().IsRegular() {
		return nil, fmt.Errorf("%s is not a regular file", path)
	}
	const maxToolBytes = 256 << 20 // 256 MiB hard cap
	if stat.Size() > maxToolBytes {
		return nil, fmt.Errorf("%s exceeds %d bytes", path, maxToolBytes)
	}
	return io.ReadAll(f)
}
