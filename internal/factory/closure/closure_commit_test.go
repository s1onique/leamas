// SPDX-License-Identifier: Apache-2.0

package closure

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// setupClosureCommitRepo creates a fresh repo with one initial commit and
// returns its absolute path. The returned repo has no working-tree files
// beyond seed.txt; tests add canonical closure artifacts before commit.
func setupClosureCommitRepo(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	for _, args := range [][]string{
		{"init", "-b", "main"},
		{"config", "user.email", "closure@example.invalid"},
		{"config", "user.name", "Closure Test"},
	} {
		if out, err := runGitValue(context.Background(), RealGit{}, root, args...); err != nil {
			t.Fatalf("git %v: %v: %s", args, err, out)
		}
	}
	if err := os.WriteFile(filepath.Join(root, "subject.txt"), []byte("subject\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{{"add", "subject.txt"}, {"commit", "-m", "subject"}} {
		if out, err := runGitValue(context.Background(), RealGit{}, root, args...); err != nil {
			t.Fatalf("git %v: %v: %s", args, err, out)
		}
	}
	return root
}

func writeCanonicalClosureArtifacts(t *testing.T, root, actID, plan, manifest, report string) (planPath, manifestPath, reportPath string) {
	t.Helper()
	planPath = "docs/closure-plans/" + actID + ".json"
	manifestPath = "docs/closure-manifests/" + actID + ".json"
	reportPath = "docs/close-reports/" + actID + ".md"
	for _, p := range []string{planPath, manifestPath, reportPath} {
		if err := os.MkdirAll(filepath.Join(root, filepath.Dir(p)), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(root, planPath), []byte(plan), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, manifestPath), []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, reportPath), []byte(report), 0o644); err != nil {
		t.Fatal(err)
	}
	return planPath, manifestPath, reportPath
}

func commitAll(t *testing.T, root, message string) (hash string) {
	t.Helper()
	if out, err := runGitValue(context.Background(), RealGit{}, root, "add", "docs"); err != nil {
		t.Fatalf("git add: %v: %s", err, out)
	}
	if out, err := runGitValue(context.Background(), RealGit{}, root, "commit", "-m", message); err != nil {
		t.Fatalf("git commit: %v: %s", err, out)
	}
	hash, err := runGitValue(context.Background(), RealGit{}, root, "rev-parse", "HEAD")
	if err != nil {
		t.Fatal(err)
	}
	return hash
}

func TestClosureCommitDerivedFromExactArtifactBlobs(t *testing.T) {
	root := setupClosureCommitRepo(t)
	subject, err := runGitValue(context.Background(), RealGit{}, root, "rev-parse", "HEAD")
	if err != nil {
		t.Fatal(err)
	}
	planPath, manifestPath, reportPath := writeCanonicalClosureArtifacts(t, root,
		"ACT-LEAMAS-CLOSURE-COMMIT01",
		"{\"act_id\":\"ACT-LEAMAS-CLOSURE-COMMIT01\"}\n",
		"{\"act_id\":\"ACT-LEAMAS-CLOSURE-COMMIT01\",\"verdict\":\"pass\"}\n",
		"# report\n",
	)
	closure := commitAll(t, root, "closure")
	planBlob, _ := runGitValue(context.Background(), RealGit{}, root, "rev-parse", closure+":"+planPath)
	manifestBlob, _ := runGitValue(context.Background(), RealGit{}, root, "rev-parse", closure+":"+manifestPath)
	reportBlob, _ := runGitValue(context.Background(), RealGit{}, root, "rev-parse", closure+":"+reportPath)
	match, err := FindClosureCommit(context.Background(), ClosureCommitSelector{
		CanonicalPaths:    []string{planPath, manifestPath, reportPath},
		ExpectedBlobOIDs:  []string{planBlob, manifestBlob, reportBlob},
		SubjectCommitOID:  subject,
		RunRepositoryRoot: root,
	})
	if err != nil {
		t.Fatalf("FindClosureCommit: %v", err)
	}
	if match.CommitOID != closure {
		t.Fatalf("expected %s got %s", closure, match.CommitOID)
	}
}

func TestClosureCommitMessageAloneIsInsufficient(t *testing.T) {
	root := setupClosureCommitRepo(t)
	subject, err := runGitValue(context.Background(), RealGit{}, root, "rev-parse", "HEAD")
	if err != nil {
		t.Fatal(err)
	}
	// The selector with an empty path set must fail at the API boundary
	// without performing any ref lookup.
	_, err = FindClosureCommit(context.Background(), ClosureCommitSelector{
		CanonicalPaths:    nil,
		ExpectedBlobOIDs:  nil,
		SubjectCommitOID:  subject,
		RunRepositoryRoot: root,
	})
	if err == nil || !strings.Contains(err.Error(), "at least one canonical path") {
		t.Fatalf("expected selector rejection, got %v", err)
	}
}

func TestClosureCommitMustDescendFromSubject(t *testing.T) {
	root := setupClosureCommitRepo(t)
	planPath, manifestPath, reportPath := writeCanonicalClosureArtifacts(t, root,
		"ACT-LEAMAS-CLOSURE-COMMIT-DESCENT01",
		"{\"act_id\":\"ACT-LEAMAS-CLOSURE-COMMIT-DESCENT01\"}\n",
		"{\"act_id\":\"ACT-LEAMAS-CLOSURE-COMMIT-DESCENT01\",\"verdict\":\"pass\"}\n",
		"# report\n",
	)
	_ = commitAll(t, root, "files")
	planBlob, _ := runGitValue(context.Background(), RealGit{}, root, "rev-parse", "HEAD:"+planPath)
	manifestBlob, _ := runGitValue(context.Background(), RealGit{}, root, "rev-parse", "HEAD:"+manifestPath)
	reportBlob, _ := runGitValue(context.Background(), RealGit{}, root, "rev-parse", "HEAD:"+reportPath)
	// Use a subject that is unrelated to the closure commit.
	_, err := FindClosureCommit(context.Background(), ClosureCommitSelector{
		CanonicalPaths:    []string{planPath, manifestPath, reportPath},
		ExpectedBlobOIDs:  []string{planBlob, manifestBlob, reportBlob},
		SubjectCommitOID:  strings.Repeat("0", 40),
		RunRepositoryRoot: root,
	})
	if err == nil || !strings.Contains(err.Error(), "subject") {
		t.Fatalf("expected subject descent rejection, got %v", err)
	}
}

func TestCurrentHeadMayDescendFromClosureCommit(t *testing.T) {
	root := setupClosureCommitRepo(t)
	planPath, manifestPath, reportPath := writeCanonicalClosureArtifacts(t, root,
		"ACT-LEAMAS-CLOSURE-COMMIT-DESCEND02",
		"{\"act_id\":\"ACT-LEAMAS-CLOSURE-COMMIT-DESCEND02\"}\n",
		"{\"act_id\":\"ACT-LEAMAS-CLOSURE-COMMIT-DESCEND02\",\"verdict\":\"pass\"}\n",
		"# report\n",
	)
	closure := commitAll(t, root, "closure")
	if out, err := runGitValue(context.Background(), RealGit{}, root, "commit", "--allow-empty", "-m", "descendant"); err != nil {
		t.Fatalf("git commit: %v: %s", err, out)
	}
	subject, _ := runGitValue(context.Background(), RealGit{}, root, "rev-parse", "HEAD~2")
	planBlob, _ := runGitValue(context.Background(), RealGit{}, root, "rev-parse", closure+":"+planPath)
	manifestBlob, _ := runGitValue(context.Background(), RealGit{}, root, "rev-parse", closure+":"+manifestPath)
	reportBlob, _ := runGitValue(context.Background(), RealGit{}, root, "rev-parse", closure+":"+reportPath)
	match, err := FindClosureCommit(context.Background(), ClosureCommitSelector{
		CanonicalPaths:    []string{planPath, manifestPath, reportPath},
		ExpectedBlobOIDs:  []string{planBlob, manifestBlob, reportBlob},
		SubjectCommitOID:  subject,
		RunRepositoryRoot: root,
	})
	if err != nil {
		t.Fatalf("FindClosureCommit: %v", err)
	}
	if match.CommitOID != closure {
		t.Fatalf("closure commit must be selectable even when HEAD descends from it")
	}
}

func TestExactBinaryParityDetectsMismatch(t *testing.T) {
	manifest := RunnerIdentity{LeamasVersion: "0.1.0", BinarySHA256: strings.Repeat("a", 64), VCSRevision: strings.Repeat("b", 40), VCSModified: false}
	live := RunnerIdentity{LeamasVersion: "0.1.0", BinarySHA256: strings.Repeat("a", 64), VCSRevision: strings.Repeat("b", 40), VCSModified: false}
	parity := AssertExactBinaryParity(manifest, live, strings.Repeat("a", 64))
	if !parity.Compatible {
		t.Fatalf("matching identities were rejected: %+v", parity)
	}
	liveModified := live
	liveModified.VCSModified = true
	parity = AssertExactBinaryParity(manifest, liveModified, strings.Repeat("a", 64))
	if parity.Compatible {
		t.Fatalf("VCSModified mismatch not detected: %+v", parity)
	}
}
