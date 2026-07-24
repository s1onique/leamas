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
	"path/filepath"
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
