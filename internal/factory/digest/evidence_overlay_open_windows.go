// SPDX-License-Identifier: Apache-2.0

//go:build windows

// Package digest: evidence_overlay_open_windows.go defines
// the platform-specific safe-open primitive used by the
// renderer.
//
// Windows does not have Unix FIFO semantics in the same form;
// named pipes and other special objects follow different
// blocking rules and the O_NONBLOCK equivalent is irrelevant
// in the same way. The renderer therefore uses an ordinary
// read-only root.Open call, and the post-open Stat/SameFile
// check still rejects substitutions before any read.
//
// The cross-platform invariant is the SAME FILE OBJECT:
//
//	opened.Mode().IsRegular() && os.SameFile(opened, collected)
//
// before any content read.
package digest

import (
	"fmt"
	"os"
)

// safeOpenReadOnlyNonblock opens rel through root in
// read-only mode. On Windows there is no O_NONBLOCK analogue
// needed because the post-open regular-file check rejects
// any substituted special object before any blocking read
// can be attempted.
func safeOpenReadOnlyNonblock(root *os.Root, rel string) (*os.File, error) {
	if root == nil {
		return nil, fmt.Errorf("safe-open: nil root")
	}
	if rel == "" {
		return nil, fmt.Errorf("safe-open: empty relative path")
	}
	return root.OpenFile(rel, os.O_RDONLY, 0)
}
