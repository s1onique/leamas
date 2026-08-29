// SPDX-License-Identifier: Apache-2.0

// Package digest: evidence_overlay_path.go isolates the path-
// walk primitives used by ValidateEvidenceRoots. Splitting them
// out keeps the validation orchestration in
// evidence_overlay.go under the LLM-friendly 400-line ceiling.
package digest

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// hasDotDotSegment reports whether any slash-separated segment
// of `raw` is exactly `..`. Trailing slashes and platform
// separators are normalized through filepath.ToSlash so the
// check works consistently on POSIX and Windows paths. The
// function deliberately rejects segments like `a/../b` even
// when `filepath.Clean` would normalize them inside the
// repository; the goal is the stricter MVP safety policy.
func hasDotDotSegment(raw string) bool {
	if raw == "" {
		return false
	}
	normalized := filepath.ToSlash(raw)
	for _, seg := range strings.Split(normalized, "/") {
		if seg == ".." {
			return true
		}
	}
	return false
}

// rejectSymlinkChain walks every existing component of `target`
// (interpreted relative to `canonicalRoot`) and rejects any that
// resolves through a symlink. The walk stops at the first missing
// component (the leaf may not exist yet).
//
// The chain check uses Lstat per segment; macOS users (where
// `/var` is a symlink to `/private/var`) get the canonical form
// because the repository root itself is resolved before the walk.
func rejectSymlinkChain(canonicalRoot, target string) error {
	rel, rerr := filepath.Rel(canonicalRoot, target)
	if rerr != nil {
		return fmt.Errorf("%s cannot resolve %q relative to repo root: %w",
			evidenceRejectedPrefix, target, rerr)
	}
	if rel == "." {
		// Targeting the repo root itself is allowed as a
		// degenerate evidence root (collects the entire
		// ignored tree under .gitignore / .git/info/exclude).
		return nil
	}
	cursor := canonicalRoot
	for _, seg := range splitPathSegments(rel) {
		cursor = filepath.Join(cursor, seg)
		lst, lerr := os.Lstat(cursor)
		if lerr != nil {
			if errors.Is(lerr, os.ErrNotExist) {
				return nil
			}
			return fmt.Errorf("%s cannot inspect %q: %w",
				evidenceRejectedPrefix, cursor, lerr)
		}
		if lst.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("%s refusing evidence path that traverses symlink %q",
				evidenceRejectedPrefix, cursor)
		}
	}
	return nil
}

// splitPathSegments splits a relative path into its slash-aware
// segments without producing "." or "" entries; it tolerates the
// platform separator so paths produced on Windows survive on Linux
// during cross-platform test runs.
func splitPathSegments(rel string) []string {
	if rel == "" || rel == "." {
		return nil
	}
	parts := strings.Split(filepath.ToSlash(rel), "/")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p == "" || p == "." {
			continue
		}
		out = append(out, p)
	}
	return out
}
