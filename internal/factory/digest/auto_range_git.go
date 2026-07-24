// SPDX-License-Identifier: Apache-2.0

package digest

import (
	"fmt"
	"os/exec"
	"strings"
)

// readHeadBlob returns the contents of path at the given commit.
// Returns an error if the file does not exist at that commit.
func readHeadBlob(repoRoot, commit, path string) ([]byte, error) {
	return runGitBytes(repoRoot, "show", commit+":"+path)
}

// runGitValueTrimmed runs `git <args>` and returns trimmed stdout.
func runGitValueTrimmed(repoRoot string, args ...string) (string, error) {
	out, err := runGitBytes(repoRoot, args...)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// runGitOutput runs `git <args>` and returns the raw stdout.
func runGitOutput(repoRoot string, args ...string) (string, error) {
	out, err := runGitBytes(repoRoot, args...)
	if err != nil {
		return "", err
	}
	return string(out), nil
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

// listTreeNames returns the file names under dir at the given commit.
func listTreeNames(repoRoot, commit, dir string) ([]string, error) {
	out, err := runGitBytes(repoRoot, "ls-tree", "--name-only", commit+"^{tree}", "--", dir)
	if err != nil {
		return nil, nil
	}
	var names []string
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if line = strings.TrimSpace(line); line != "" {
			names = append(names, line)
		}
	}
	return names, nil
}

// mustResolveOID resolves a short or full OID to a full OID, or ""
// when resolution fails.
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
