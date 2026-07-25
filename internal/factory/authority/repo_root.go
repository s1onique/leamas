// SPDX-License-Identifier: Apache-2.0

// Package authority: repo_root.go provides the deterministic
// repository-root resolver used by the executable-authority
// checks and any other consumer that needs to anchor a path to
// the Leamas source tree.
//
// The resolver replaces brittle arithmetical patterns such as
// filepath.Clean(filepath.Join(wd, "..", "..", "..")) that
// silently miss when the package test directory or the
// checkout depth differs from expectations.
package authority

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// ErrNoRepositoryRoot is returned when FindRepositoryRoot cannot
// locate a Leamas repository from the supplied starting path.
var ErrNoRepositoryRoot = errors.New("no Leamas repository root found")

// RepositorySentinels are the files that, taken together, mark
// a directory as the Leamas repository root. All sentinels must
// exist; a directory that only contains go.mod is not sufficient
// because nested modules can satisfy that signal alone.
var RepositorySentinels = []string{
	"go.mod",
	"Makefile",
	"AGENTS.md",
	"githooks/pre-push",
	".factory",
}

// FindRepositoryRoot walks up from start looking for the first
// directory containing every entry of RepositorySentinels. The
// walk terminates at the filesystem root.
//
// When start is empty, the current working directory is used. If
// start refers to a regular file rather than a directory, the
// file's parent is used instead. Symbolic links are not resolved
// before walking; a separate symlink-rooted test exercises the
// resolution behaviour.
func FindRepositoryRoot(start string) (string, error) {
	dir := start
	if dir == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return "", fmt.Errorf("getwd: %w", err)
		}
		dir = cwd
	} else {
		info, statErr := os.Stat(dir)
		if statErr == nil && !info.IsDir() {
			dir = filepath.Dir(dir)
		}
	}
	abs, err := filepath.Abs(dir)
	if err != nil {
		return "", fmt.Errorf("abs: %w", err)
	}
	dir = abs
	for {
		if hasAllSentinels(dir) {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", ErrNoRepositoryRoot
		}
		dir = parent
	}
}

// hasAllSentinels reports whether dir contains every entry of
// RepositorySentinels as a regular file or directory. The check
// uses os.Stat so it follows symlinks; sentinels are expected to
// be version-controlled files, not arbitrary user content.
func hasAllSentinels(dir string) bool {
	for _, sentinel := range RepositorySentinels {
		if _, err := os.Stat(filepath.Join(dir, sentinel)); err != nil {
			return false
		}
	}
	return true
}
