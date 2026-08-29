// SPDX-License-Identifier: Apache-2.0

//go:build unix

// Package digest: evidence_overlay_open_unix.go defines the
// platform-specific safe-open primitive used by the renderer.
// On Unix systems the renderer MUST open evidence files with
// O_NONBLOCK so that a pathname previously inspected as a
// regular file but later replaced by a FIFO (or other special
// file) cannot block the render thread waiting for a writer.
//
// The traversal-resistance property is preserved: the open
// is performed against an *os.Root via OpenFile (Go >= 1.24),
// which retains the root-confinement guarantee while also
// letting us pass the O_NONBLOCK flag. After the nonblocking
// open, the renderer must Stat the file, reject non-regular
// objects, and verify SameFile against the collected identity
// before any read.
package digest

import (
	"fmt"
	"os"
	"syscall"
)

// safeOpenReadOnlyNonblock opens rel through root with the
// O_NONBLOCK flag set, so a FIFO/socket/device substitution
// between collection and render cannot block waiting for a
// writer. The caller MUST Stat the returned file and verify
// it is regular (and matches the collected identity) before
// reading any bytes.
//
// On Unix the call resolves to root.OpenFile with the
// platform O_NONBLOCK flag; on non-Unix platforms see
// evidence_overlay_open_windows.go.
func safeOpenReadOnlyNonblock(root *os.Root, rel string) (*os.File, error) {
	if root == nil {
		return nil, fmt.Errorf("safe-open: nil root")
	}
	if rel == "" {
		return nil, fmt.Errorf("safe-open: empty relative path")
	}
	return root.OpenFile(rel, os.O_RDONLY|syscall.O_NONBLOCK, 0)
}
