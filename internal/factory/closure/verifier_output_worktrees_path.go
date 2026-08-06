// SPDX-License-Identifier: Apache-2.0

package closure

// verifier_output_worktrees_path.go isolates the
// canonicalization helper from the inventory observation path
// in verifier_output_worktrees.go. Splitting along the path-
// resolution boundary keeps each file under the LLM-
// friendliness 400-line threshold while preserving the
// canonical-only invariant from CORRECTION02B.

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// canonicalizeWorktreeRoot converts a raw porcelain worktree
// path into a canonical absolute form. The function NEVER
// falls back to a lexical form: relative paths are rejected,
// and paths whose canonical form cannot be resolved (broken
// symlink, non-existent, permission-denied) are rejected so
// the inventory cannot silently underflow.
//
// The function returns ErrRelativeWorktree when the upstream
// output was relative, ErrUnresolvableWorktree when the path
// does not exist or points through a broken symlink.
func canonicalizeWorktreeRoot(raw string) (string, error) {
	if !filepath.IsAbs(raw) {
		return "", fmt.Errorf("relative worktree path %q (must be absolute)", raw)
	}
	if _, err := os.Lstat(raw); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", fmt.Errorf("unresolvable worktree path %q: %w", raw, os.ErrNotExist)
		}
		return "", fmt.Errorf("stat worktree path %q: %w", raw, err)
	}
	resolved, err := filepath.EvalSymlinks(raw)
	if err != nil {
		return "", fmt.Errorf("eval symlinks on %q: %w", raw, err)
	}
	if !filepath.IsAbs(resolved) {
		return "", fmt.Errorf("resolved worktree path %q is not absolute", resolved)
	}
	return filepath.Clean(resolved), nil
}
