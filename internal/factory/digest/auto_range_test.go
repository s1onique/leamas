// SPDX-License-Identifier: Apache-2.0

// Package digest provides targeted digest generation for Git repositories.
//
// auto_range_test.go is the executable contract for ACT-LEAMAS-FACTORY-DIGEST-AUTO-ACT-RANGE01.
// The tests build small Git repositories with synthetic histories that
// mirror the regression fixture described in the ACT, then assert that
// ResolveAutoMode selects the expected authoritative range and that
// the digest surface area renders the documented LIFECYCLE metadata.
package digest

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// commitAt runs `git commit --allow-empty -m msg` and returns the new HEAD OID.
func commitAt(t *testing.T, dir, msg string) string {
	t.Helper()
	runGit(t, dir, "commit", "--allow-empty", "-m", msg)
	out, err := runGitValueTrimmed(dir, "rev-parse", "HEAD")
	if err != nil {
		t.Fatalf("rev-parse HEAD: %v", err)
	}
	return out
}

// commitFile writes content under dir/name and commits it. Returns the new HEAD OID.
func commitFile(t *testing.T, dir, name, content, msg string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}
	runGit(t, dir, "add", name)
	return commitAt(t, dir, msg)
}

// commitChange modifies an existing tracked file and commits. Returns new HEAD OID.
func commitChange(t *testing.T, dir, name, content, msg string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}
	runGit(t, dir, "add", name)
	return commitAt(t, dir, msg)
}

// makeAnnotatedTag creates an annotated tag tagName pointing at HEAD.
func makeAnnotatedTag(t *testing.T, dir, tagName, body string) {
	t.Helper()
	runGit(t, dir, "tag", "--annotate", "--cleanup=verbatim", "--message", body, tagName)
}

// TestResolveAutoMode_DirtyProducesDirtyMode verifies the legacy
// dirty-tree behaviour is unchanged.
func TestResolveAutoMode_DirtyProducesDirtyMode(t *testing.T) {
	dir := t.TempDir()
	initGit(t, dir)

	commitFile(t, dir, "tracked.txt", "v1\n", "initial")

	if err := os.WriteFile(filepath.Join(dir, "tracked.txt"), []byte("v1\nmodified\n"), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}

	got, err := ResolveAutoMode(dir)
	if err != nil {
		t.Fatalf("ResolveAutoMode: %v", err)
	}
	if got.Mode != ModeDirty {
		t.Fatalf("Mode = %s, want dirty", got.Mode)
	}
	if !strings.Contains(got.Reason, "working tree") {
		t.Fatalf("Reason = %q, want working-tree note", got.Reason)
	}
}

// TestResolveAutoMode_SingleImplementationCommitFallsBack verifies
// that a clean HEAD with no ACT artifacts resolves to HEAD~1..HEAD.
func TestResolveAutoMode_SingleImplementationCommitFallsBack(t *testing.T) {
	dir := t.TempDir()
	initGit(t, dir)

	parent := commitFile(t, dir, "file1.txt", "v1\n", "first commit")
	commitFile(t, dir, "file2.txt", "v2\n", "second commit")

	got, err := ResolveAutoMode(dir)
	if err != nil {
		t.Fatalf("ResolveAutoMode: %v", err)
	}
	if got.Mode != ModeRange {
		t.Fatalf("Mode = %s, want range", got.Mode)
	}
	// The resolver returns the literal "HEAD~1..HEAD" range for
	// the verified_single_commit fallback to keep the digest
	// surface stable; RangeBase and RangeHead still carry full OIDs.
	wantRange := parent + ".." + got.HeadCommit
	if got.Range != "HEAD~1..HEAD" && got.Range != wantRange {
		t.Fatalf("Range = %s, want HEAD~1..HEAD or %s", got.Range, wantRange)
	}
	if got.BaseCommit != parent {
		t.Fatalf("BaseCommit = %s, want %s", got.BaseCommit, parent)
	}
	if got.RangeStrategy() != StrategyVerifiedSingleCommit {
		t.Fatalf("RangeStrategy = %s, want verified_single_commit", got.RangeStrategy())
	}
}

// TestResolveAutoMode_InitialCommitUsesEmptyTreeBaseline verifies the
// initial-commit edge case still works.
func TestResolveAutoMode_InitialCommitUsesEmptyTreeBaseline(t *testing.T) {
	dir := t.TempDir()
	initGit(t, dir)

	commitFile(t, dir, "first.txt", "first\n", "first commit")

	got, err := ResolveAutoMode(dir)
	if err != nil {
		t.Fatalf("ResolveAutoMode: %v", err)
	}
	if got.Mode != ModeRange {
		t.Fatalf("Mode = %s, want range", got.Mode)
	}
	if !strings.HasPrefix(got.Range, emptyTreeBaseline) {
		t.Fatalf("Range = %s, want empty-tree baseline prefix %s", got.Range, emptyTreeBaseline)
	}
}

// TestResolveAutoMode_CloseReportSelectsImplementationRange is the
// C07-C09 regression fixture: a CORRECTION09 commit that only
// contains the close-report markdown file, but the close report
// references C07, C08, and the whitespace fix as the implementation
// range.
func TestResolveAutoMode_CloseReportSelectsImplementationRange(t *testing.T) {
	dir := t.TempDir()
	initGit(t, dir)

	base := commitChange(t, dir, "tracked.txt", "v1\n", "baseline")
	c07 := commitChange(t, dir, "tracked.txt", "v1\nc07\n", "C07 implementation")
	c08 := commitChange(t, dir, "tracked.txt", "v1\nc07\nc08\n", "C08 implementation")
	hygiene := commitChange(t, dir, "tracked.txt", "v1\nc07\nc08\nhygiene\n", "hygiene fix")

	// HEAD introduces a close-report markdown for the wrapping ACT.
	actID := "ACT-LEAMAS-FACTORY-DIGEST-AUTO-ACT-TEST01"
	closeReport := fmt.Sprintf(`# Close Report: %s

## Implementation Range

| Identity | Commit |
|----------|--------|
| BASE | %s |
| C07 | %s |
| C08 | %s |
| Subject (HEAD) | %s |

Verified ordering: C07 is ancestor of C08.
`, actID, base, c07, c08, hygiene)
	if err := os.MkdirAll(filepath.Join(dir, "docs/close-reports"), 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "docs/close-reports/"+actID+".md"),
		[]byte(closeReport), 0644); err != nil {
		t.Fatalf("write report: %v", err)
	}
	runGit(t, dir, "add", "docs/close-reports/"+actID+".md")
	commitAt(t, dir, actID+": close-report only")

	got, err := ResolveAutoMode(dir)
	if err != nil {
		t.Fatalf("ResolveAutoMode: %v", err)
	}
	if got.Mode != ModeRange {
		t.Fatalf("Mode = %s, want range", got.Mode)
	}
	if got.RangeStrategy() != StrategyActCommitTrailers {
		t.Fatalf("RangeStrategy = %s, want act_commit_trailers", got.RangeStrategy())
	}
	if got.ActID != actID {
		t.Fatalf("ActID = %s, want %s", got.ActID, actID)
	}
	wantBase := base
	if got.BaseCommit != wantBase {
		t.Fatalf("BaseCommit = %s, want %s", got.BaseCommit, wantBase)
	}
	if got.HeadCommit == "" {
		t.Fatalf("HeadCommit empty")
	}
	if !strings.HasSuffix(got.Range, got.HeadCommit) {
		t.Fatalf("Range %s does not end with HEAD %s", got.Range, got.HeadCommit)
	}
	if len(got.IncludedCommits) < 4 {
		t.Fatalf("IncludedCommits = %v, want >= 4 (C07, C08, hygiene, C09)", got.IncludedCommits)
	}
}

// TestResolveAutoMode_DocumentationClosureIncludesImplementation
// verifies that an implementation commit followed by a documentation
// closure commit includes both.
func TestResolveAutoMode_DocumentationClosureIncludesImplementation(t *testing.T) {
	dir := t.TempDir()
	initGit(t, dir)

	base := commitChange(t, dir, "tracked.txt", "v1\n", "baseline")
	impl := commitChange(t, dir, "tracked.txt", "v1\nimpl\n", "implementation")

	actID := "ACT-LEAMAS-FACTORY-DIGEST-AUTO-ACT-DOC01"
	closeReport := fmt.Sprintf(`# Close Report: %s

## Implementation Range

| Identity | Commit |
|----------|--------|
| BASE | %s |
| Subject (HEAD) | %s |

`, actID, base, impl)
	if err := os.MkdirAll(filepath.Join(dir, "docs/close-reports"), 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "docs/close-reports/"+actID+".md"),
		[]byte(closeReport), 0644); err != nil {
		t.Fatalf("write report: %v", err)
	}
	runGit(t, dir, "add", "docs/close-reports/"+actID+".md")
	commitAt(t, dir, actID+": docs-only closure")

	got, err := ResolveAutoMode(dir)
	if err != nil {
		t.Fatalf("ResolveAutoMode: %v", err)
	}
	if got.RangeStrategy() != StrategyActCommitTrailers {
		t.Fatalf("RangeStrategy = %s, want act_commit_trailers", got.RangeStrategy())
	}
	// Expect both commits in the range.
	if len(got.IncludedCommits) < 2 {
		t.Fatalf("IncludedCommits = %v, want >= 2", got.IncludedCommits)
	}
}

// TestResolveAutoMode_ManifestAuthoritativeRange verifies that a
// closure manifest drives the range selection when both a manifest
// and a close report exist.
func TestResolveAutoMode_ManifestAuthoritativeRange(t *testing.T) {
	dir := t.TempDir()
	initGit(t, dir)

	// F = freeze (plan-freeze commit, manifest records it)
	// S = subject (implementation)
	// C = closure (manifest file committed at HEAD)
	baseline := commitChange(t, dir, "tracked.txt", "v1\n", "baseline")
	commitChange(t, dir, "tracked.txt", "v1\nfreeze\n", "freeze commit")
	subject := commitChange(t, dir, "tracked.txt", "v1\nfreeze\nsubject\n", "subject")
	commitChange(t, dir, "tracked.txt", "v1\nfreeze\nsubject\nextra\n", "extra")
	freeze := commitChange(t, dir, "tracked.txt", "v1\nfreeze\nsubject\nextra\nplan\n", "manifest freeze")
	commitChange(t, dir, "tracked.txt", "v1\nfreeze\nsubject\nextra\nplan\nclosure\n", "closure work")

	actID := "ACT-LEAMAS-FACTORY-DIGEST-AUTO-ACT-MANIFEST01"
	manifest := fmt.Sprintf(`{
  "contract_version": 1,
  "act_id": "%s",
  "plan": {
    "sha256": "f0000000000000000000000000000000000000000000000000000000000000f",
    "path": "docs/closure-plans/%s.json"
  },
  "plan_freeze": {
    "freeze_commit": "%s",
    "plan_path": "docs/closure-plans/%s.json",
    "plan_blob_oid": "0000000000000000000000000000000000000000",
    "plan_sha256": "f0000000000000000000000000000000000000000000000000000000000000f",
    "subject_commit": "%s"
  },
  "subject": {
    "commit_oid": "%s",
    "tree_oid": "0000000000000000000000000000000000000000"
  },
  "verdict": "pass"
}
`, actID, actID, freeze, actID, subject, subject)
	if err := os.MkdirAll(filepath.Join(dir, "docs/closure-manifests"), 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "docs/closure-manifests/"+actID+".json"),
		[]byte(manifest), 0644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	runGit(t, dir, "add", "docs/closure-manifests/"+actID+".json")
	commitAt(t, dir, actID+": manifest-only closure")

	got, err := ResolveAutoMode(dir)
	if err != nil {
		t.Fatalf("ResolveAutoMode: %v", err)
	}
	if got.RangeStrategy() != StrategyClosureManifest {
		t.Fatalf("RangeStrategy = %s, want closure_manifest", got.RangeStrategy())
	}
	if got.LifecycleFreeze != freeze {
		t.Fatalf("LifecycleFreeze = %s, want %s", got.LifecycleFreeze, freeze)
	}
	if got.LifecycleSubject != subject {
		t.Fatalf("LifecycleSubject = %s, want %s", got.LifecycleSubject, subject)
	}
	// Range base should be freeze^, not the historical baseline.
	wantBase, err := runGitValueTrimmed(dir, "rev-parse", freeze+"^")
	if err != nil {
		t.Fatalf("resolve freeze^: %v", err)
	}
	if got.BaseCommit != wantBase {
		t.Fatalf("BaseCommit = %s, want %s (freeze^)", got.BaseCommit, wantBase)
	}
	if got.BaseCommit == baseline {
		t.Fatalf("BaseCommit = %s, must not be the unrelated baseline", got.BaseCommit)
	}
}

// TestResolveAutoMode_MultipleActsFailsClosed verifies that two
// distinct ACTs at HEAD cause fail-closed.
func TestResolveAutoMode_MultipleActsFailsClosed(t *testing.T) {
	dir := t.TempDir()
	initGit(t, dir)

	commitChange(t, dir, "tracked.txt", "v1\n", "baseline")
	base1 := commitChange(t, dir, "tracked.txt", "v1\nA\n", "actA baseline")
	base2 := commitChange(t, dir, "tracked.txt", "v1\nA\nB\n", "actB baseline")

	// Build two unrelated close reports and commit them together.
	actA := "ACT-LEAMAS-FACTORY-DIGEST-AUTO-ACT-AMBIGUOUS-A"
	actB := "ACT-LEAMAS-FACTORY-DIGEST-AUTO-ACT-AMBIGUOUS-B"
	if err := os.MkdirAll(filepath.Join(dir, "docs/close-reports"), 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	reportA := fmt.Sprintf("# Close Report: %s\n\n## Implementation Range\n\n| Identity | Commit |\n|----------|--------|\n| BASE | %s |\n", actA, base1)
	reportB := fmt.Sprintf("# Close Report: %s\n\n## Implementation Range\n\n| Identity | Commit |\n|----------|--------|\n| BASE | %s |\n", actB, base2)
	if err := os.WriteFile(filepath.Join(dir, "docs/close-reports/"+actA+".md"), []byte(reportA), 0644); err != nil {
		t.Fatalf("write reportA: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "docs/close-reports/"+actB+".md"), []byte(reportB), 0644); err != nil {
		t.Fatalf("write reportB: %v", err)
	}
	runGit(t, dir, "add", "docs/close-reports/"+actA+".md", "docs/close-reports/"+actB+".md")
	commitAt(t, dir, "ambiguous: two ACT close reports in one commit")

	_, err := ResolveAutoMode(dir)
	if err == nil {
		t.Fatalf("ResolveAutoMode succeeded; expected ambiguous error")
	}
	if !errors.Is(err, ErrAmbiguousRange) {
		t.Fatalf("error %v, want ErrAmbiguousRange", err)
	}
	if !strings.Contains(err.Error(), actA) || !strings.Contains(err.Error(), actB) {
		t.Fatalf("error %v does not list both candidates", err)
	}
}
