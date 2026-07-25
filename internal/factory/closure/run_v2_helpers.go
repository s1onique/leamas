// SPDX-License-Identifier: Apache-2.0

package closure

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
)

// validateV2Plan validates a closure plan for v2 protocol.
func validateV2Plan(plan Plan) error {
	if plan.ContractVersion != ContractVersionV1 {
		return fmt.Errorf("unsupported contract_version %d", plan.ContractVersion)
	}
	if !actIDPattern.MatchString(plan.ActID) {
		return fmt.Errorf("invalid act_id %q", plan.ActID)
	}
	if plan.Baseline.CommitOID == "" || plan.Baseline.TreeOID == "" {
		return fmt.Errorf("baseline.commit_oid and baseline.tree_oid are required")
	}
	if len(plan.Checks) == 0 {
		return fmt.Errorf("at least one check is required")
	}
	return nil
}

// resolveGitObjectWithFormat resolves a git object and validates its format.
func resolveGitObjectWithFormat(ctx context.Context, git gitClient, root, expression string, format ObjectFormat) (string, error) {
	oid, err := runGitValue(ctx, git, root, "rev-parse", "--verify", "--end-of-options", expression)
	if err != nil {
		return "", err
	}
	if err := ValidateOIDWithFormat("Git object", oid, format); err != nil {
		return "", err
	}
	return oid, nil
}

// verifySingleParent uses `git rev-list --parents -n 1` to verify that
// commit has exactly one parent and returns that parent. Strict parsing:
// exactly one non-empty output line, exactly two fields, first field
// equals the requested commit, and both OIDs validated against the
// repository object format.
func verifySingleParent(ctx context.Context, git gitClient, root, commit string, format ObjectFormat) (string, error) {
	result := git.Run(ctx, root, "rev-list", "--parents", "-n", "1", commit)
	if result.Err != nil || result.ExitCode != 0 {
		return "", fmt.Errorf("git rev-list --parents failed: %v: %s", result.Err, result.Stderr)
	}
	lines := bytesSplitLines(result.Stdout)
	if len(lines) == 0 {
		return "", fmt.Errorf("rev-list returned no output")
	}
	if len(lines) > 1 {
		return "", fmt.Errorf("rev-list returned %d lines; want exactly 1", len(lines))
	}
	fields := bytesSplitFields(lines[0])
	if len(fields) == 0 {
		return "", fmt.Errorf("rev-list returned empty line")
	}
	if len(fields) == 1 {
		return "", fmt.Errorf("commit has no parent (root commit)")
	}
	if len(fields) > 2 {
		return "", fmt.Errorf("commit is a merge with %d parents", len(fields)-1)
	}
	if string(fields[0]) != commit {
		return "", fmt.Errorf("rev-list returned %q, expected %q", string(fields[0]), commit)
	}
	if err := ValidateOIDWithFormat("rev-list commit", string(fields[0]), format); err != nil {
		return "", err
	}
	if err := ValidateOIDWithFormat("rev-list parent", string(fields[1]), format); err != nil {
		return "", err
	}
	return string(fields[1]), nil
}

// evidenceDirectoryPath returns the final deterministic evidence path.
func evidenceDirectoryPath(repoRoot, actID, subjectCommit string) string {
	return filepath.Join(repoRoot, ".factory", "closure-evidence", actID, subjectCommit)
}

// createEvidenceStagingDirectory creates a same-parent staging directory.
// The final deterministic path remains absent until qualification succeeds.
func createEvidenceStagingDirectory(repoRoot, actID, subjectCommit string) (string, error) {
	parent := filepath.Dir(evidenceDirectoryPath(repoRoot, actID, subjectCommit))
	if err := os.MkdirAll(parent, 0o700); err != nil {
		return "", fmt.Errorf("create evidence parent: %w", err)
	}
	if _, err := os.Lstat(evidenceDirectoryPath(repoRoot, actID, subjectCommit)); err == nil {
		return "", fmt.Errorf("final evidence directory already exists")
	} else if !os.IsNotExist(err) {
		return "", fmt.Errorf("inspect final evidence directory: %w", err)
	}
	staging, err := os.MkdirTemp(parent, ".staging-"+subjectCommit+"-")
	if err != nil {
		return "", fmt.Errorf("create evidence staging directory: %w", err)
	}
	if err := os.Chmod(staging, 0o700); err != nil {
		_ = os.RemoveAll(staging)
		return "", fmt.Errorf("secure evidence staging directory: %w", err)
	}
	return staging, nil
}
