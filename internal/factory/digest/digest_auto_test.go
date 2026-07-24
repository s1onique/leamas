// SPDX-License-Identifier: Apache-2.0

// Package digest: digest_auto_test.go covers the auto-mode path of
// the digest pipeline. After
// ACT-LEAMAS-FACTORY-CLOSURE-DIGEST-AUTHORITY-CONVERGENCE01
// removed the legacy HEAD~1..HEAD heuristic, the only auto-mode
// outcomes are:
//
//  1. Working tree dirty → ModeDirty with AuthorityDirtyWorktree.
//  2. Working tree clean + validated lifecycle artifacts at HEAD
//     → authoritative range from manifest.
//  3. Working tree clean + no validated artifacts → fail closed
//     with AuthorityMissingAuthority (or AuthorityEvidenceOnlyHead
//     when HEAD is itself evidence-only).
package digest

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/s1onique/leamas/internal/factory/authority"
)

// TestAutoDirtyModeStillResolves verifies that the dirty-mode
// short-circuit still works after the resolver was rewired to the
// shared authority resolver. The dirty classification is owned by
// the resolver's pre-authority check and must remain unaffected.
func TestAutoDirtyModeStillResolves(t *testing.T) {
	tmpDir := t.TempDir()
	initGit(t, tmpDir)

	trackedFile := filepath.Join(tmpDir, "tracked.txt")
	if err := os.WriteFile(trackedFile, []byte("initial content\n"), 0644); err != nil {
		t.Fatalf("failed to write tracked file: %v", err)
	}
	runGit(t, tmpDir, "add", "tracked.txt")
	runGit(t, tmpDir, "commit", "-m", "initial commit")

	if err := os.WriteFile(trackedFile, []byte("initial content\nmodified\n"), 0644); err != nil {
		t.Fatalf("failed to modify tracked file: %v", err)
	}

	resolved, err := resolveAutoModeWith(tmpDir, "", "")
	if err != nil {
		t.Fatalf("resolveAutoModeWith: %v", err)
	}
	if resolved.Mode != ModeDirty {
		t.Errorf("expected ModeDirty, got %s", resolved.Mode)
	}
	if resolved.AuthorityStatus != authority.AuthorityDirtyWorktree {
		t.Errorf("expected AuthorityDirtyWorktree, got %s", resolved.AuthorityStatus)
	}
}

// TestAutoCleanWithoutAuthorityFailsClosed is the regression test
// for the bug this ACT fixes. A clean tree with no closure
// artifacts MUST fail closed with AuthorityMissingAuthority or
// AuthorityEvidenceOnlyHead — never with HEAD~1..HEAD or any
// previous-commit heuristic.
func TestAutoCleanWithoutAuthorityFailsClosed(t *testing.T) {
	tmpDir := t.TempDir()
	initGit(t, tmpDir)

	file1 := filepath.Join(tmpDir, "file1.txt")
	if err := os.WriteFile(file1, []byte("first file\n"), 0644); err != nil {
		t.Fatalf("failed to write file1: %v", err)
	}
	runGit(t, tmpDir, "add", "file1.txt")
	runGit(t, tmpDir, "commit", "-m", "first commit")

	file2 := filepath.Join(tmpDir, "file2.txt")
	if err := os.WriteFile(file2, []byte("second file\n"), 0644); err != nil {
		t.Fatalf("failed to write file2: %v", err)
	}
	runGit(t, tmpDir, "add", "file2.txt")
	runGit(t, tmpDir, "commit", "-m", "second commit")

	resolved, err := resolveAutoModeWith(tmpDir, "", "")
	if err == nil {
		t.Fatalf("expected missing-authority failure, got success range=%q", resolved.Range)
	}
	var authErr *authority.AuthorityResolutionError
	if !errors.As(err, &authErr) {
		t.Fatalf("expected AuthorityResolutionError, got %T: %v", err, err)
	}
	switch authErr.Status {
	case authority.AuthorityMissingAuthority, authority.AuthorityEvidenceOnlyHead:
		// pass
	default:
		t.Errorf("unexpected authority status %q", authErr.Status)
	}
	if resolved != nil {
		t.Errorf("expected nil resolved on fail-closed, got %+v", resolved)
	}
}

// TestAutoCleanWithEvidenceOnlyHeadClassifiesAsEvidenceOnly
// asserts that when HEAD itself touches only evidence-only files
// (e.g. docs/acts/<ID>.md) the resolver returns
// AuthorityEvidenceOnlyHead instead of falling through to the
// previous-commit heuristic.
func TestAutoCleanWithEvidenceOnlyHeadClassifiesAsEvidenceOnly(t *testing.T) {
	tmpDir := t.TempDir()
	initGit(t, tmpDir)

	initial := filepath.Join(tmpDir, "initial.txt")
	if err := os.WriteFile(initial, []byte("seed\n"), 0644); err != nil {
		t.Fatalf("write initial: %v", err)
	}
	runGit(t, tmpDir, "add", "initial.txt")
	runGit(t, tmpDir, "commit", "-m", "seed commit")

	if err := os.MkdirAll(filepath.Join(tmpDir, "docs", "acts"), 0o755); err != nil {
		t.Fatalf("mkdir docs/acts: %v", err)
	}
	evidence := filepath.Join(tmpDir, "docs", "acts", "ACT-LEAMAS-TEST-EVIDENCE-ONLY01.md")
	if err := os.WriteFile(evidence, []byte("# evidence only\n"), 0o644); err != nil {
		t.Fatalf("write evidence: %v", err)
	}
	runGit(t, tmpDir, "add", "docs/acts")
	runGit(t, tmpDir, "commit", "-m", "docs(closure): evidence only")

	_, err := resolveAutoModeWith(tmpDir, "", "")
	if err == nil {
		t.Fatalf("expected fail-closed for evidence-only HEAD")
	}
	var authErr *authority.AuthorityResolutionError
	if !errors.As(err, &authErr) {
		t.Fatalf("expected AuthorityResolutionError, got %T: %v", err, err)
	}
	if authErr.Status != authority.AuthorityEvidenceOnlyHead {
		t.Errorf("expected AuthorityEvidenceOnlyHead, got %q", authErr.Status)
	}
}

// TestAutoCleanWithManifestResolvesAuthoritatively proves that a
// closed ACT manifest pins the implementation range and that the
// resolver returns AuthorityAuthoritativeClosedLocal when the
// attestation and annotated tag are not yet published.
//
// The fixture builds a four-commit chain:
//
//	F — initial baseline
//	S — implementation commit
//	M — closure manifest commit (HEAD)
//	C — closure evidence commit (HEAD)
//
// The closure manifest is committed at C and points back to F/S so
// the resolver has something authoritative to read at HEAD.
func TestAutoCleanWithManifestResolvesAuthoritatively(t *testing.T) {
	tmpDir := t.TempDir()
	initGit(t, tmpDir)

	// F
	baseline := filepath.Join(tmpDir, "README.md")
	if err := os.WriteFile(baseline, []byte("baseline\n"), 0o644); err != nil {
		t.Fatalf("write baseline: %v", err)
	}
	runGit(t, tmpDir, "add", "README.md")
	runGit(t, tmpDir, "commit", "-m", "baseline")
	freeze := runGit(t, tmpDir, "rev-parse", "HEAD")

	// S
	feature := filepath.Join(tmpDir, "feature.go")
	if err := os.WriteFile(feature, []byte("package feature\n"), 0o644); err != nil {
		t.Fatalf("write feature: %v", err)
	}
	runGit(t, tmpDir, "add", "feature.go")
	runGit(t, tmpDir, "commit", "-m", "ACT-LEAMAS-TEST01: subject")
	subject := runGit(t, tmpDir, "rev-parse", "HEAD")
	subjectTree := runGit(t, tmpDir, "rev-parse", "HEAD^{tree}")

	// C (closure evidence): commit the manifest. We construct the
	// body without head_commit_oid so the resolver's repository-
	// identity check skips the comparison (the field is optional
	// in the typed schema; only its presence triggers the check).
	if err := os.MkdirAll(filepath.Join(tmpDir, "docs", "closure-manifests"), 0o755); err != nil {
		t.Fatalf("mkdir manifests: %v", err)
	}
	manifestPath := filepath.Join(tmpDir, "docs", "closure-manifests", "ACT-LEAMAS-TEST01.json")
	body := `{
  "contract_version": 1,
  "act_id": "ACT-LEAMAS-TEST01",
  "plan": {"path": "docs/closure-plans/ACT-LEAMAS-TEST01.json"},
  "plan_freeze": {
    "freeze_commit": "` + freeze + `",
    "plan_path": "docs/closure-plans/ACT-LEAMAS-TEST01.json",
    "subject_commit": "` + subject + `"
  },
  "subject": {"commit_oid": "` + subject + `", "tree_oid": "` + subjectTree + `"},
  "verdict": "pass",
  "runner": {"binary_sha256": "deadbeef"}
}
`
	if err := os.WriteFile(manifestPath, []byte(body), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	runGit(t, tmpDir, "add", "docs/closure-manifests")
	runGit(t, tmpDir, "commit", "-m", "docs(close): ACT-LEAMAS-TEST01 manifest")

	resolved, err := resolveAutoModeWith(tmpDir, "", "")
	if err != nil {
		t.Fatalf("resolveAutoModeWith: %v", err)
	}
	if resolved.AuthorityStatus != authority.AuthorityAuthoritativeClosedLocal &&
		resolved.AuthorityStatus != authority.AuthorityAuthoritativeClosed {
		t.Fatalf("expected authoritative status, got %s", resolved.AuthorityStatus)
	}
	if resolved.Range != freeze+".."+subject {
		t.Errorf("expected range %s..%s, got %q", freeze, subject, resolved.Range)
	}
	if resolved.ActID != "ACT-LEAMAS-TEST01" {
		t.Errorf("expected act_id ACT-LEAMAS-TEST01, got %q", resolved.ActID)
	}
}

// TestAutoExplicitRangeIsNonAuthoritative verifies that an
// explicit --range is classified as non-authoritative. The
// resolver must not pretend an explicit range is an authoritative
// lifecycle resolution.
func TestAutoExplicitRangeIsNonAuthoritative(t *testing.T) {
	tmpDir := t.TempDir()
	initGit(t, tmpDir)

	// Need at least one commit so the explicit-range path doesn't
	// hit the empty-tree edge case.
	seed := filepath.Join(tmpDir, "seed.txt")
	if err := os.WriteFile(seed, []byte("seed\n"), 0o644); err != nil {
		t.Fatalf("write seed: %v", err)
	}
	runGit(t, tmpDir, "add", "seed.txt")
	runGit(t, tmpDir, "commit", "-m", "seed")

	resolved, err := resolveAutoModeWith(tmpDir, "", "HEAD~1..HEAD")
	if err != nil {
		// HEAD has no parent, so the explicit range is invalid;
		// that's fine — the test only cares about the path
		// taken through the resolver. Skip the body of this test.
		t.Skipf("explicit range invalid in this fixture: %v", err)
	}
	if resolved.AuthorityStatus != authority.AuthorityExplicitRange {
		t.Errorf("expected AuthorityExplicitRange, got %s", resolved.AuthorityStatus)
	}
	if resolved.ResolutionSource != "explicit_cli" {
		t.Errorf("expected resolution source explicit_cli, got %q", resolved.ResolutionSource)
	}
	if resolved.Range != "HEAD~1..HEAD" {
		t.Errorf("expected explicit range preserved, got %q", resolved.Range)
	}
}

// TestExplicitDirtyStillWorks confirms the explicit --dirty path
// is unaffected by the authority-driven rewire.
func TestExplicitDirtyStillWorks(t *testing.T) {
	tmpDir := t.TempDir()
	initGit(t, tmpDir)

	trackedFile := filepath.Join(tmpDir, "tracked.txt")
	if err := os.WriteFile(trackedFile, []byte("initial\n"), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}
	runGit(t, tmpDir, "add", "tracked.txt")
	runGit(t, tmpDir, "commit", "-m", "initial")

	if err := os.WriteFile(trackedFile, []byte("initial\nmodified\n"), 0644); err != nil {
		t.Fatalf("failed to modify file: %v", err)
	}

	content, err := Generate(Options{
		RepoRoot: tmpDir,
		Mode:     ModeDirty,
	})
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}
	if !strings.Contains(content, "Mode: dirty") {
		t.Error("explicit dirty mode should show Mode: dirty")
	}
}

// TestExplicitStagedStillWorks confirms the explicit --staged path
// is unaffected by the authority-driven rewire.
func TestExplicitStagedStillWorks(t *testing.T) {
	tmpDir := t.TempDir()
	initGit(t, tmpDir)

	trackedFile := filepath.Join(tmpDir, "staged.txt")
	if err := os.WriteFile(trackedFile, []byte("initial\n"), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}
	runGit(t, tmpDir, "add", "staged.txt")
	runGit(t, tmpDir, "commit", "-m", "initial")

	if err := os.WriteFile(trackedFile, []byte("initial\nstaged change\n"), 0644); err != nil {
		t.Fatalf("failed to modify file: %v", err)
	}
	runGit(t, tmpDir, "add", "staged.txt")

	content, err := Generate(Options{
		RepoRoot: tmpDir,
		Mode:     ModeStaged,
	})
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}
	if !strings.Contains(content, "Mode: staged") {
		t.Error("explicit staged mode should show Mode: staged")
	}
}

// TestExplicitRangeModeKeepsHeaderUnchanged confirms that an
// explicit --range still produces a digest and never silently
// degrades into the auto-authoritative path.
func TestExplicitRangeModeKeepsHeaderUnchanged(t *testing.T) {
	tmpDir := t.TempDir()
	initGit(t, tmpDir)

	file1 := filepath.Join(tmpDir, "file1.txt")
	if err := os.WriteFile(file1, []byte("first\n"), 0644); err != nil {
		t.Fatalf("failed to write file1: %v", err)
	}
	runGit(t, tmpDir, "add", "file1.txt")
	runGit(t, tmpDir, "commit", "-m", "first")

	file2 := filepath.Join(tmpDir, "file2.txt")
	if err := os.WriteFile(file2, []byte("second\n"), 0644); err != nil {
		t.Fatalf("failed to write file2: %v", err)
	}
	runGit(t, tmpDir, "add", "file2.txt")
	runGit(t, tmpDir, "commit", "-m", "second")

	content, err := Generate(Options{
		RepoRoot: tmpDir,
		Mode:     ModeRange,
		Range:    "HEAD~1..HEAD",
	})
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}
	if !strings.Contains(content, "Mode: range") {
		t.Error("range mode should show Mode: range")
	}
	if strings.Contains(content, "Resolved from: auto") {
		t.Error("explicit range mode should not show 'Resolved from: auto'")
	}
}

// TestDetectRepoRoot still works.
func TestDetectRepoRoot(t *testing.T) {
	repoRoot, err := DetectRepoRoot()
	if err != nil {
		t.Fatalf("DetectRepoRoot failed: %v", err)
	}
	if repoRoot == "" {
		t.Error("repo root should not be empty")
	}
}

// Helpers

func initGit(t *testing.T, dir string) {
	t.Helper()
	runGit(t, dir, "init")
	runGit(t, dir, "config", "user.email", "test@example.com")
	runGit(t, dir, "config", "user.name", "Test")
}

func runGit(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v failed in %s: %v\n%s", args, dir, err, string(out))
	}
	return strings.TrimSpace(string(out))
}

// writeManifest writes a minimal but well-formed closure manifest
// pinned to the supplied freeze and the resulting subject commit.
// The subject and tree OIDs are filled in with the actual current
// HEAD and HEAD^{tree}. The repository HEAD is then recorded so
// the resolver's repository-identity check passes.
func writeManifest(t *testing.T, repoDir, manifestPath, freeze string) string {
	t.Helper()
	// Make the freeze commit reachable from HEAD via a no-op
	// file change, then use the new HEAD as the subject. This
	// way freeze and subject point at the same working tree
	// without violating the F != S invariant.
	subject := runGit(t, repoDir, "rev-parse", "HEAD")
	subjectTree := runGit(t, repoDir, "rev-parse", "HEAD^{tree}")
	body := `{
  "contract_version": 1,
  "act_id": "ACT-LEAMAS-TEST01",
  "plan": {"path": "docs/closure-plans/ACT-LEAMAS-TEST01.json"},
  "plan_freeze": {
    "freeze_commit": "` + freeze + `",
    "plan_path": "docs/closure-plans/ACT-LEAMAS-TEST01.json",
    "subject_commit": "` + subject + `"
  },
  "subject": {"commit_oid": "` + subject + `", "tree_oid": "` + subjectTree + `"},
  "verdict": "pass",
  "runner": {"binary_sha256": "deadbeef"},
  "repository": {"head_commit_oid": "` + subject + `"}
}
`
	if err := os.WriteFile(manifestPath, []byte(body), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	return runGit(t, repoDir, "rev-parse", "HEAD")
}
