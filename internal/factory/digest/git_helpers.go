// SPDX-License-Identifier: Apache-2.0

// Package digest: git_helpers.go preserves the small set of
// subprocess helpers consumed by resolve.go and lifecycle_render.go
// after the legacy HEAD~1..HEAD auto-range resolver was removed.
//
// These helpers were previously bundled in auto_range_git.go and
// are now isolated here so the new authority-driven resolve path
// can keep using them without re-introducing the heuristic.
package digest

import (
	"fmt"
	"os/exec"
	"strings"
)

// runGitValueTrimmed runs `git <args>` and returns trimmed stdout.
func runGitValueTrimmed(repoRoot string, args ...string) (string, error) {
	out, err := runGitBytes(repoRoot, args...)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// runGitBytes runs `git <args>` and returns the captured stdout bytes.
func runGitBytes(repoRoot string, args ...string) ([]byte, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = repoRoot
	out, err := cmd.Output()
	if err != nil {
		return out, fmt.Errorf("git %s: %w", strings.Join(args, " "), err)
	}
	return out, nil
}

// mustResolveOID resolves a short or full OID to a full OID, or ""
// when resolution fails. It mirrors the helper the legacy
// auto-range used and is preserved so digest_status_*_test.go
// can continue to validate range semantics.
func mustResolveOID(repoRoot, oid string) string {
	oid = strings.TrimSpace(oid)
	if oid == "" {
		return ""
	}
	out, err := runGitValueTrimmed(repoRoot, "rev-parse", "--verify", "--end-of-options", oid+"^{commit}")
	if err != nil {
		return ""
	}
	if !fullOIDPattern.MatchString(out) {
		return ""
	}
	return strings.ToLower(out)
}

// shortSHA returns the first 12 hex chars of an OID. When the input
// is empty, returns "unknown".
func shortSHA(oid string) string {
	oid = strings.TrimSpace(oid)
	if oid == "" {
		return "unknown"
	}
	if len(oid) <= 12 {
		return oid
	}
	return oid[:12]
}
