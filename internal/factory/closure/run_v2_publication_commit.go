// SPDX-License-Identifier: Apache-2.0

package closure

import (
	"bytes"
	"context"
	"fmt"
	"strings"
)

// publishV2Refs performs the branch and tag compare-and-swap in one explicit
// Git update-ref transaction. A branch CAS or tag-creation conflict prevents
// either ref from being committed.
func publishV2Refs(ctx context.Context, git gitClient, repoRoot, branch, tagName, subject, closure, tagObject string) error {
	if branch == "" {
		return fmt.Errorf("invalid branch ref name (empty)")
	}
	if err := validateV2BranchName(ctx, git, repoRoot, branch); err != nil {
		return err
	}
	if !v2TagNamePattern.MatchString(tagName) {
		return fmt.Errorf("invalid tag ref name %q", tagName)
	}
	input := strings.Join([]string{
		"start",
		"update refs/heads/" + branch + " " + closure + " " + subject,
		"create refs/tags/" + tagName + " " + tagObject,
		"prepare",
		"commit",
		"",
	}, "\n")
	result := git.RunWithStdin(ctx, repoRoot, input, "update-ref", "--stdin")
	if result.Err != nil || result.ExitCode != 0 {
		return fmt.Errorf("publish refs transaction failed: %s", gitFailureDetail(result))
	}
	return nil
}

func validateV2BranchName(ctx context.Context, git gitClient, repoRoot, branch string) error {
	result := git.Run(ctx, repoRoot, "check-ref-format", "--branch", branch)
	if result.Err != nil || result.ExitCode != 0 {
		detail := strings.TrimSpace(string(result.Stderr))
		if detail == "" && result.Err != nil {
			detail = result.Err.Error()
		}
		return fmt.Errorf("invalid branch ref name %q: %s", branch, sanitizeDiagnostic(detail))
	}
	return nil
}

// requireExactV2ConvergenceState accepts only the interruption window created
// after refs move S→C while index and worktree still represent S. In porcelain
// v1 -z that is exactly two staged deletions (`D `), one for each canonical
// artifact. Every collision, rename, type change, third path, or worktree-side
// mutation is rejected before reset --hard can run.
func requireExactV2ConvergenceState(ctx context.Context, git gitClient, repoRoot string, expected v2ExpectedTransaction) error {
	result := git.Run(ctx, repoRoot, "status", "--porcelain=v1", "-z", "--untracked-files=all")
	if result.Err != nil || result.ExitCode != 0 {
		return fmt.Errorf("inspect worktree status: %s", gitFailureDetail(result))
	}
	canonicalPaths := []string{
		canonicalV2ManifestPath(expected.Tag.ActID),
		canonicalV2ReportPath(expected.Tag.ActID),
	}
	want := map[string]bool{canonicalPaths[0]: false, canonicalPaths[1]: false}
	records := bytes.Split(result.Stdout, []byte{0})
	if len(records) == 0 || len(records[len(records)-1]) != 0 {
		return fmt.Errorf("malformed NUL-delimited status output")
	}
	records = records[:len(records)-1]
	if len(records) != len(want) {
		return fmt.Errorf("post-ref status has %d records, want exactly %d", len(records), len(want))
	}
	for _, record := range records {
		if len(record) < 4 || record[2] != ' ' {
			return fmt.Errorf("malformed status record %q", record)
		}
		xy, path := string(record[:2]), string(record[3:])
		seen, canonical := want[path]
		if xy != "D " || !canonical || seen {
			return fmt.Errorf("unsafe post-ref status %q for %q", xy, path)
		}
		want[path] = true
	}
	for _, path := range canonicalPaths {
		if !want[path] {
			return fmt.Errorf("missing canonical staged deletion %q", path)
		}
	}
	return nil
}

func boundedConverge(ctx context.Context, git gitClient, repoRoot string, expected v2ExpectedTransaction) error {
	if err := requireExactV2ConvergenceState(ctx, git, repoRoot, expected); err != nil {
		return err
	}
	result := git.Run(ctx, repoRoot, "reset", "--hard", "HEAD")
	if result.Err != nil || result.ExitCode != 0 {
		return fmt.Errorf("converge worktree: %s", gitFailureDetail(result))
	}
	return nil
}
