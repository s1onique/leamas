// SPDX-License-Identifier: Apache-2.0

package closure

import (
	"context"
	"fmt"
	"strings"
	"testing"
)

func TestPublishV2RefsCommitsBranchAndAnnotatedTagTogether(t *testing.T) {
	repo, subject, tree := prepareObjectTransactionRepository(t, ObjectFormatSHA1)
	closure := oidGit(t, repo, "commit-tree", tree, "-p", subject, "-m", "closure")
	tagName := canonicalV2TagName(objectTransactionActID)
	tagBytes := fmt.Sprintf("object %s\ntype commit\ntag %s\ntagger Test <test@example.invalid> 1 +0000\n\nprepared\n", closure, tagName)
	tag := rawGitStdinStdout(t, repo, tagBytes, "mktag")
	if err := publishV2Refs(context.Background(), RealGit{}, repo, "main", tagName, subject, closure, tag); err != nil {
		t.Fatal(err)
	}
	if got := oidGit(t, repo, "rev-parse", "refs/heads/main"); got != closure {
		t.Fatalf("branch = %s", got)
	}
	if got := oidGit(t, repo, "rev-parse", "refs/tags/"+tagName+"^{tag}"); got != tag {
		t.Fatalf("tag = %s", got)
	}
}

func TestPublishV2RefsCASConflictChangesNeitherRef(t *testing.T) {
	repo, subject, tree := prepareObjectTransactionRepository(t, ObjectFormatSHA1)
	closure := oidGit(t, repo, "commit-tree", tree, "-p", subject, "-m", "closure")
	moved := oidGit(t, repo, "commit-tree", tree, "-p", subject, "-m", "moved")
	rawGitStdout(t, repo, "update-ref", "refs/heads/main", moved, subject)
	tagName := canonicalV2TagName(objectTransactionActID)
	tagBytes := fmt.Sprintf("object %s\ntype commit\ntag %s\ntagger Test <test@example.invalid> 1 +0000\n\nprepared\n", closure, tagName)
	tag := rawGitStdinStdout(t, repo, tagBytes, "mktag")
	if err := publishV2Refs(context.Background(), RealGit{}, repo, "main", tagName, subject, closure, tag); err == nil {
		t.Fatal("stale branch CAS was accepted")
	}
	if got := oidGit(t, repo, "rev-parse", "refs/heads/main"); got != moved {
		t.Fatalf("branch changed to %s", got)
	}
	result := RealGit{}.Run(context.Background(), repo, "show-ref", "--verify", "--quiet", "refs/tags/"+tagName)
	if result.ExitCode == 0 {
		t.Fatal("tag ref was created despite branch conflict")
	}
}

func oidGit(t *testing.T, repo string, args ...string) string {
	t.Helper()
	return strings.TrimSpace(string(rawGitStdout(t, repo, args...)))
}

func rawGitStdinStdout(t *testing.T, repo, stdin string, args ...string) string {
	t.Helper()
	result := RealGit{}.RunWithStdin(context.Background(), repo, stdin, args...)
	if result.Err != nil || result.ExitCode != 0 {
		t.Fatalf("git %v: %s", args, result.Stderr)
	}
	return strings.TrimSpace(string(result.Stdout))
}
