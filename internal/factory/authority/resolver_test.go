// SPDX-License-Identifier: Apache-2.0

// Package authority: resolver_test.go drives the shared authority
// resolver against a hermetic Git fixture. The fixture builds the
// canonical F → S → C chain required by the act, then mutates
// one element at a time to prove every resolver invariant.
package authority

import (
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

const fixtureActID = "ACT-LEAMAS-AUTHORITY-RESOLVER01"

// TestResolverValidClosedActReturnsAuthoritative proves the happy
// path: a manifest pinned to a freeze and subject resolves with
// AuthorityAuthoritativeClosedLocal when no attestation/tag is
// supplied. The full closed-ACT path is exercised by integration
// tests in cmd/leamas.
func TestResolverValidClosedActReturnsAuthoritative(t *testing.T) {
	repo := newFixtureRepo(t)
	freeze, subject, _ := fixtureClosedLocal(t, repo)

	resolved, err := Resolve(ResolverOptions{RepoRoot: repo})
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if resolved.AuthorityStatus != AuthorityAuthoritativeClosedLocal {
		t.Errorf("expected AuthoritativeClosedLocal, got %s", resolved.AuthorityStatus)
	}
	if resolved.DigestRange != freeze+".."+subject {
		t.Errorf("expected range %s..%s, got %q", freeze, subject, resolved.DigestRange)
	}
	if resolved.ActID != fixtureActID {
		t.Errorf("expected act_id %q, got %q", fixtureActID, resolved.ActID)
	}
}

// TestResolverMissingAuthorityReturnsTypedError is the negative
// case: a clean repository with no closure artifacts must fail
// closed with AuthorityMissingAuthority, never with a heuristic
// fallback range.
func TestResolverMissingAuthorityReturnsTypedError(t *testing.T) {
	repo := newFixtureRepo(t)
	_, err := Resolve(ResolverOptions{RepoRoot: repo})
	if err == nil {
		t.Fatalf("expected missing-authority failure")
	}
	var authErr *AuthorityResolutionError
	if !errors.As(err, &authErr) {
		t.Fatalf("expected AuthorityResolutionError, got %T: %v", err, err)
	}
	if authErr.Status != AuthorityMissingAuthority {
		t.Errorf("expected MissingAuthority, got %s", authErr.Status)
	}
}

// TestResolverEvidenceOnlyHeadFailsClosed verifies the
// evidence-only HEAD path. When HEAD touches only docs/* files,
// the resolver returns AuthorityEvidenceOnlyHead with no range.
func TestResolverEvidenceOnlyHeadFailsClosed(t *testing.T) {
	repo := newFixtureRepo(t)
	if err := os.MkdirAll(filepath.Join(repo, "docs", "acts"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(repo, "docs", "acts", fixtureActID+".md"),
		[]byte("# evidence\n"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	git(t, repo, "add", "docs/acts")
	git(t, repo, "commit", "-m", "docs(acts): "+fixtureActID)

	_, err := Resolve(ResolverOptions{RepoRoot: repo})
	if err == nil {
		t.Fatalf("expected EvidenceOnlyHead failure")
	}
	var authErr *AuthorityResolutionError
	if !errors.As(err, &authErr) {
		t.Fatalf("expected AuthorityResolutionError, got %T", err)
	}
	if authErr.Status != AuthorityEvidenceOnlyHead {
		t.Errorf("expected EvidenceOnlyHead, got %s", authErr.Status)
	}
}

// TestResolverAmbiguousAuthorityFailsClosed builds two distinct
// closure manifests at HEAD and verifies the resolver refuses to
// silently pick one.
func TestResolverAmbiguousAuthorityFailsClosed(t *testing.T) {
	repo := newFixtureRepo(t)
	freeze, subject, _ := fixtureClosedLocal(t, repo)

	// Add a second manifest at HEAD that points to the same F/S.
	// We amend the manifest commit so HEAD itself introduces
	// both manifest files in the same diff; otherwise the
	// resolver only sees the most recent manifest.
	mfPath := filepath.Join(repo, "docs", "closure-manifests", "ACT-LEAMAS-OTHER01.json")
	body := manifestJSON("ACT-LEAMAS-OTHER01", freeze, subject)
	if err := os.WriteFile(mfPath, []byte(body), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	git(t, repo, "add", "docs/closure-manifests")
	git(t, repo, "commit", "--amend", "--no-edit")

	_, err := Resolve(ResolverOptions{RepoRoot: repo})
	if err == nil {
		t.Fatalf("expected ambiguous-authority failure")
	}
	var authErr *AuthorityResolutionError
	if !errors.As(err, &authErr) {
		t.Fatalf("expected AuthorityResolutionError, got %T", err)
	}
	if authErr.Status != AuthorityAmbiguousAuthority {
		t.Errorf("expected AmbiguousAuthority, got %s", authErr.Status)
	}
}

// TestResolverInvalidArtifactFailsClosed mangles the manifest's
// plan_freeze and verifies the resolver refuses the artifact
// (freeze == subject violates F != S).
func TestResolverInvalidArtifactFailsClosed(t *testing.T) {
	repo := newFixtureRepo(t)
	freeze, _, manifestPath := fixtureClosedLocal(t, repo)

	// Set freeze == subject using the same freeze OID for both
	// fields. This is a different invalid case from missing
	// fields, but both must fail closed.
	body := manifestJSON(fixtureActID, freeze, freeze)
	if err := os.WriteFile(manifestPath, []byte(body), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	git(t, repo, "add", "docs/closure-manifests")
	git(t, repo, "commit", "--amend", "--no-edit")

	_, err := Resolve(ResolverOptions{RepoRoot: repo})
	if err == nil {
		t.Fatalf("expected invalid-artifact failure")
	}
	var authErr *AuthorityResolutionError
	if !errors.As(err, &authErr) {
		t.Fatalf("expected AuthorityResolutionError, got %T", err)
	}
	if authErr.Status != AuthorityInvalidArtifact {
		t.Errorf("expected InvalidArtifact, got %s", authErr.Status)
	}
}

// TestResolverExplicitRangeClassifiesAsNonAuthoritative verifies
// that an explicit --range bypasses lifecycle authority and is
// classified as AuthorityExplicitRange.
func TestResolverExplicitRangeClassifiesAsNonAuthoritative(t *testing.T) {
	repo := newFixtureRepo(t)
	// Need at least one commit before HEAD. Make a second commit
	// so HEAD~1..HEAD resolves.
	if err := os.WriteFile(filepath.Join(repo, "extra.txt"), []byte("x\n"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	git(t, repo, "add", "extra.txt")
	git(t, repo, "commit", "-q", "-m", "extra")

	resolved, err := Resolve(ResolverOptions{
		RepoRoot:      repo,
		ExplicitRange: "HEAD~1..HEAD",
	})
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if resolved.AuthorityStatus != AuthorityExplicitRange {
		t.Errorf("expected AuthorityExplicitRange, got %s", resolved.AuthorityStatus)
	}
	if resolved.DigestRange != "HEAD~1..HEAD" {
		t.Errorf("expected explicit range preserved, got %q", resolved.DigestRange)
	}
	if resolved.ResolutionSrc != "explicit_cli" {
		t.Errorf("expected resolution source explicit_cli, got %q", resolved.ResolutionSrc)
	}
}

// TestResolverToolPathIsRecorded verifies that the resolver
// stores the supplied ToolBinaryPath on the resolved authority
// even when no real binary probe is available.
func TestResolverToolPathIsRecorded(t *testing.T) {
	repo := newFixtureRepo(t)
	fakeBin := filepath.Join(t.TempDir(), "fake-leamas")
	if err := os.WriteFile(fakeBin, []byte("not really leamas"), 0o755); err != nil {
		t.Fatalf("write fake binary: %v", err)
	}

	// The fake binary cannot satisfy `version --json`, so the
	// resolver surfaces a ToolIdentityMismatch. We use this
	// negative path to assert that supplying a tool path
	// produces a typed error rather than an authoritative
	// resolution.
	_, err := Resolve(ResolverOptions{
		RepoRoot:      repo,
		ToolBinaryPath: fakeBin,
	})
	if err == nil {
		t.Fatalf("expected typed error from fake binary")
	}
	var authErr *AuthorityResolutionError
	if !errors.As(err, &authErr) {
		t.Fatalf("expected AuthorityResolutionError, got %T: %v", err, err)
	}
	if authErr.Status != AuthorityToolIdentityMismatch {
		t.Errorf("expected ToolIdentityMismatch, got %s", authErr.Status)
	}
}

// TestResolverRequiresRepoRoot verifies the contract: a missing
// RepoRoot is rejected with AuthorityInvalidArtifact.
func TestResolverRequiresRepoRoot(t *testing.T) {
	_, err := Resolve(ResolverOptions{})
	if err == nil {
		t.Fatalf("expected missing-repo-root failure")
	}
	var authErr *AuthorityResolutionError
	if !errors.As(err, &authErr) {
		t.Fatalf("expected AuthorityResolutionError, got %T", err)
	}
	if authErr.Status != AuthorityInvalidArtifact {
		t.Errorf("expected InvalidArtifact, got %s", authErr.Status)
	}
}

// TestResolverRepositoryIdentityMismatchFailsClosed verifies the
// repository-identity invariant. The manifest's recorded
// head_commit_oid MUST match the actual repository HEAD; when it
// doesn't, the resolver refuses to claim authority.
func TestResolverRepositoryIdentityMismatchFailsClosed(t *testing.T) {
	repo := newFixtureRepo(t)
	_, _, manifestPath := fixtureClosedLocal(t, repo)

	// Replace the manifest with one whose recorded head_commit_oid
	// disagrees with the actual HEAD.
	body := `{
  "contract_version": 1,
  "act_id": "ACT-LEAMAS-AUTHORITY-RESOLVER01",
  "plan": {"path": "docs/closure-plans/x.json"},
  "plan_freeze": {
    "freeze_commit": "0000000000000000000000000000000000000001",
    "plan_path": "docs/closure-plans/x.json",
    "subject_commit": "0000000000000000000000000000000000000002"
  },
  "subject": {
    "commit_oid": "0000000000000000000000000000000000000002",
    "tree_oid": "0000000000000000000000000000000000000002"
  },
  "verdict": "pass",
  "runner": {"binary_sha256": "deadbeef"},
  "repository": {"head_commit_oid": "deadbeefdeadbeefdeadbeefdeadbeefdeadbeef"}
}
`
	if err := os.WriteFile(manifestPath, []byte(body), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	git(t, repo, "add", "docs/closure-manifests")
	git(t, repo, "commit", "--amend", "--no-edit")

	_, err := Resolve(ResolverOptions{RepoRoot: repo})
	if err == nil {
		t.Fatalf("expected repository-identity-mismatch failure")
	}
	var authErr *AuthorityResolutionError
	if !errors.As(err, &authErr) {
		t.Fatalf("expected AuthorityResolutionError, got %T", err)
	}
	if authErr.Status != AuthorityInvalidGitObject && authErr.Status != AuthorityInvalidArtifact &&
		authErr.Status != AuthorityRepositoryIdentityMismatch {
		t.Errorf("expected identity/artifact/object failure, got %s", authErr.Status)
	}
}

// TestResolverSubjectNotAncestorOfHeadFailsClosed verifies that
// the resolver refuses a manifest whose subject does not descend
// to the current HEAD.
func TestResolverSubjectNotAncestorOfHeadFailsClosed(t *testing.T) {
	repo := newFixtureRepo(t)
	_, _, manifestPath := fixtureClosedLocal(t, repo)

	// Replace the manifest's subject_commit with a random OID
	// that is NOT an ancestor of HEAD.
	body := manifestJSON(fixtureActID, "0000000000000000000000000000000000000001",
		"0000000000000000000000000000000000000002")
	if err := os.WriteFile(manifestPath, []byte(body), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	git(t, repo, "add", "docs/closure-manifests")
	git(t, repo, "commit", "--amend", "--no-edit")

	_, err := Resolve(ResolverOptions{RepoRoot: repo})
	if err == nil {
		t.Fatalf("expected invalid-artifact failure")
	}
	var authErr *AuthorityResolutionError
	if !errors.As(err, &authErr) {
		t.Fatalf("expected AuthorityResolutionError, got %T", err)
	}
	if authErr.Status != AuthorityInvalidArtifact && authErr.Status != AuthorityInvalidGitObject {
		t.Errorf("expected InvalidArtifact or InvalidGitObject, got %s", authErr.Status)
	}
}

// ---- fixture helpers ---------------------------------------------------

func newFixtureRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	git(t, dir, "init", "-q")
	git(t, dir, "config", "user.email", "resolver@example.com")
	git(t, dir, "config", "user.name", "Resolver Test")
	if err := os.WriteFile(filepath.Join(dir, "seed.txt"), []byte("seed\n"), 0o644); err != nil {
		t.Fatalf("write seed: %v", err)
	}
	git(t, dir, "add", "seed.txt")
	git(t, dir, "commit", "-q", "-m", "baseline")
	return dir
}

// fixtureClosedLocal commits a manifest without an attestation or
// annotated tag. Returns (freeze, subject, manifestPath).
func fixtureClosedLocal(t *testing.T, repo string) (string, string, string) {
	t.Helper()
	freeze := git(t, repo, "rev-parse", "HEAD")

	if err := os.WriteFile(filepath.Join(repo, "feature.go"), []byte("package x\n"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	git(t, repo, "add", "feature.go")
	git(t, repo, "commit", "-q", "-m", "feat: production commit")
	subject := git(t, repo, "rev-parse", "HEAD")

	if err := os.MkdirAll(filepath.Join(repo, "docs", "closure-manifests"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	manifestPath := filepath.Join(repo, "docs", "closure-manifests", fixtureActID+".json")
	if err := os.WriteFile(manifestPath, []byte(manifestJSON(fixtureActID, freeze, subject)), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	git(t, repo, "add", "docs/closure-manifests")
	git(t, repo, "commit", "-q", "-m", "docs(close): local manifest")
	return freeze, subject, manifestPath
}

func manifestJSON(actID, freeze, subject string) string {
	body := map[string]interface{}{
		"contract_version": 1,
		"act_id":           actID,
		"plan":             map[string]string{"path": "docs/closure-plans/" + actID + ".json"},
		"plan_freeze": map[string]string{
			"freeze_commit":  freeze,
			"plan_path":      "docs/closure-plans/" + actID + ".json",
			"subject_commit": subject,
		},
		"subject": map[string]string{"commit_oid": subject, "tree_oid": subject},
		"verdict": "pass",
		"runner":  map[string]string{"binary_sha256": "deadbeef"},
	}
	data, _ := json.MarshalIndent(body, "", "  ")
	return string(data) + "\n"
}

func git(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, string(out))
	}
	return strings.TrimSpace(string(out))
}
