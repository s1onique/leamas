// SPDX-License-Identifier: Apache-2.0

// Package digest provides targeted digest generation for Git repositories.
//
// auto_range_extra_test.go covers additional resolver scenarios:
// fail-closed semantics for docs-only HEAD, stale generator diagnosis,
// digest inventory matching `git diff --name-status`, and the
// invariant that the auto-selected range includes production/test
// files referenced by the close report.
package digest

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/s1onique/leamas/internal/version"
)

// TestResolveAutoMode_DocsOnlyHeadFailsClosed verifies that HEAD is
// a docs-only commit AND references earlier implementation -> the
// resolver refuses to silently produce a docs-only digest.
func TestResolveAutoMode_DocsOnlyHeadFailsClosed(t *testing.T) {
	dir := t.TempDir()
	initGit(t, dir)

	commitChange(t, dir, "tracked.txt", "v1\n", "baseline")
	commitChange(t, dir, "tracked.txt", "v1\nimp\n", "implementation")

	// HEAD is purely documentation with NO close-report structured table.
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("# notes\n"), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}
	runGit(t, dir, "add", "README.md")
	commitAt(t, dir, "docs: notes only")

	_, err := ResolveAutoMode(dir)
	if err == nil {
		t.Fatalf("ResolveAutoMode succeeded; expected docs-only failure")
	}
	if !errors.Is(err, ErrNoACTAuthority) {
		t.Fatalf("error %v, want ErrNoACTAuthority", err)
	}
}

// TestResolveAutoMode_StaleBinaryDiagnosed verifies that a binary
// with an unknown LEAMAS_COMMIT is diagnosed as stale.
func TestResolveAutoMode_StaleBinaryDiagnosed(t *testing.T) {
	dir := t.TempDir()
	initGit(t, dir)

	commitChange(t, dir, "tracked.txt", "v1\n", "baseline")
	commitChange(t, dir, "tracked.txt", "v1\nimp\n", "implementation")

	// Force a stale binary by overriding version.Commit to an
	// unrelated commit that is not an ancestor of HEAD.
	prevCommit := version.Commit
	t.Cleanup(func() {
		version.Commit = prevCommit
	})
	// Pick an OID that cannot be an ancestor: 0000...0001.
	version.Commit = "0000000000000000000000000000000000000001"

	got, err := ResolveAutoMode(dir)
	if err != nil {
		t.Fatalf("ResolveAutoMode: %v", err)
	}
	if !got.GeneratorStale {
		t.Fatalf("GeneratorStale = false, want true")
	}
	if got.StaleReason == "" {
		t.Fatalf("StaleReason empty")
	}
}

// TestResolveAutoMode_AutoRangeContainsImplementation verifies the
// ACCEPTANCE criterion that the auto-selected range includes at
// least one production or test file when the closure report claims
// production changes.
func TestResolveAutoMode_AutoRangeContainsImplementation(t *testing.T) {
	dir := t.TempDir()
	initGit(t, dir)

	commitChange(t, dir, "tracked.txt", "v1\n", "baseline")
	os.MkdirAll(filepath.Join(dir, "cmd"), 0755)
	commitChange(t, dir, "cmd/foo.go", "package x\n", "implementation")
	commitChange(t, dir, "cmd/foo_test.go", "package x\n", "test addition")

	actID := "ACT-LEAMAS-FACTORY-DIGEST-AUTO-ACT-INV01"
	prev, _ := runGitValueTrimmed(dir, "rev-parse", "HEAD^")
	closeReport := fmt.Sprintf("# Close Report: %s\n\n## Implementation Range\n\n| Identity | Commit |\n|----------|--------|\n| BASE | %s |\n| Subject (HEAD) | HEAD |\n\n## Files\n\n- cmd/foo.go\n- cmd/foo_test.go\n", actID, prev)
	if err := os.MkdirAll(filepath.Join(dir, "docs/close-reports"), 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "docs/close-reports/"+actID+".md"),
		[]byte(closeReport), 0644); err != nil {
		t.Fatalf("write report: %v", err)
	}
	runGit(t, dir, "add", "docs/close-reports/"+actID+".md")
	commitAt(t, dir, actID+": close report")

	out, err := Generate(Options{
		RepoRoot: dir,
		Mode:     ModeAuto,
		Output:   filepath.Join(dir, "digest.md"),
	})
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if !strings.Contains(out, "cmd/foo.go") {
		t.Fatalf("digest missing cmd/foo.go:\n%s", out)
	}
	if !strings.Contains(out, "cmd/foo_test.go") {
		t.Fatalf("digest missing cmd/foo_test.go:\n%s", out)
	}
}

// TestResolveAutoMode_DigestAgreesWithGitDiff verifies that the
// files reported in the digest match `git diff --name-status <range>`.
func TestResolveAutoMode_DigestAgreesWithGitDiff(t *testing.T) {
	dir := t.TempDir()
	initGit(t, dir)

	commitChange(t, dir, "tracked.txt", "v1\n", "baseline")
	os.MkdirAll(filepath.Join(dir, "cmd"), 0755)
	commitChange(t, dir, "cmd/foo.go", "package x\n", "implementation")
	commitChange(t, dir, "cmd/foo_test.go", "package x\n", "test addition")

	actID := "ACT-LEAMAS-FACTORY-DIGEST-AUTO-ACT-AGREE01"
	prev, _ := runGitValueTrimmed(dir, "rev-parse", "HEAD^")
	closeReport := fmt.Sprintf("# Close Report: %s\n\n## Implementation Range\n\n| Identity | Commit |\n|----------|--------|\n| BASE | %s |\n| Subject (HEAD) | HEAD |\n\n## Files\n\n- cmd/foo.go\n- cmd/foo_test.go\n", actID, prev)
	if err := os.MkdirAll(filepath.Join(dir, "docs/close-reports"), 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "docs/close-reports/"+actID+".md"),
		[]byte(closeReport), 0644); err != nil {
		t.Fatalf("write report: %v", err)
	}
	runGit(t, dir, "add", "docs/close-reports/"+actID+".md")
	commitAt(t, dir, actID+": close report")

	resolved, err := ResolveAutoMode(dir)
	if err != nil {
		t.Fatalf("ResolveAutoMode: %v", err)
	}
	if resolved.Mode != ModeRange {
		t.Fatalf("Mode = %s, want range", resolved.Mode)
	}

	gitFiles := gitNameStatus(t, dir, resolved.Range)
	digestFiles := digestNameStatus(GenerateOrFatal(t, dir))
	if !sameFileSet(gitFiles, digestFiles) {
		t.Fatalf("git diff vs digest mismatch:\n  git: %v\n  digest: %v", gitFiles, digestFiles)
	}
}

// TestResolveAutoMode_AnnotatedTagAuthoritative verifies the
// annotated ACT tag strategy.
func TestResolveAutoMode_AnnotatedTagAuthoritative(t *testing.T) {
	dir := t.TempDir()
	initGit(t, dir)

	baseline := commitChange(t, dir, "tracked.txt", "v1\n", "baseline")
	commitChange(t, dir, "tracked.txt", "v1\nfreeze\n", "freeze")
	subject := commitChange(t, dir, "tracked.txt", "v1\nfreeze\nsubject\n", "subject")
	freeze := commitChange(t, dir, "tracked.txt", "v1\nfreeze\nsubject\nplan\n", "plan freeze")
	commitChange(t, dir, "tracked.txt", "v1\nfreeze\nsubject\nplan\ntag\n", "tag work")
	closureHead := commitChange(t, dir, "tracked.txt", "v1\nfreeze\nsubject\nplan\ntag\nclosure\n", "closure work")

	actID := "ACT-LEAMAS-FACTORY-DIGEST-AUTO-ACT-TAG01"
	tagBody := fmt.Sprintf("LEAMAS_CLOSURE_TAG_CONTRACT_VERSION: 1\n"+
		"act_id: %s\n"+
		"verdict: pass\n"+
		"subject_commit_oid: %s\n"+
		"subject_tree_oid: 0000000000000000000000000000000000000000\n"+
		"closure_commit_oid: %s\n"+
		"closure_tree_oid: 0000000000000000000000000000000000000000\n"+
		"freeze_commit: %s\n",
		actID, subject, closureHead, freeze)
	makeAnnotatedTag(t, dir, "act/"+actID, tagBody)

	got, err := ResolveAutoMode(dir)
	if err != nil {
		t.Fatalf("ResolveAutoMode: %v", err)
	}
	if got.RangeStrategy() != StrategyAnnotatedActTag {
		t.Fatalf("RangeStrategy = %s, want annotated_act_tag", got.RangeStrategy())
	}
	if got.LifecycleFreeze != freeze {
		t.Fatalf("LifecycleFreeze = %s, want %s", got.LifecycleFreeze, freeze)
	}
	if got.BaseCommit == baseline {
		t.Fatalf("BaseCommit = %s, must exclude unrelated baseline", got.BaseCommit)
	}
}
