// SPDX-License-Identifier: Apache-2.0

// Package digest: lifecycle_render_correction01_test.go proves
// the CORRECTION01 invariants for ACT-LEAMAS-DIGEST-GENERATOR-WORKTREE-STALE-AUTHORITY01.
//
// The decisive regressions locked here:
//
//  1. Historical range A..B with ambient HEAD=C: when the
//     generator's embedded commit is B, the digest subject is
//     B, and ambient HEAD is C, the generator MUST be
//     AUTHORITATIVE for the digest subject. The legacy
//     GENERATOR_STALE signal MUST be true (commit ≠ HEAD), and
//     the new GENERATOR_AUTHORITATIVE_FOR_DIGEST signal MUST be
//     true. The two signals MUST coexist.
//
//  2. The renderer MUST use the explicit-range right endpoint
//     (B), NOT ambient HEAD (C), when computing the digest
//     subject for SUBJECT_BINDING.
//
//  3. A --B--C(HEAD) repository with a generator built from B
//     (the right endpoint) and an explicit --range A..B must
//     render:
//     GENERATOR_BINDING_STATUS: AUTHORITATIVE
//     GENERATOR_AUTHORITATIVE_FOR_DIGEST: true
//     GENERATOR_STALE: true
//     GENERATOR_COMMIT_MATCHES_HEAD: false
//     GENERATOR_COMMIT_BINDING: MISMATCH
//     GENERATOR_SUBJECT_BINDING: MATCH
//
//  4. The same digest subject (B) with a generator built from
//     ambient HEAD=C must render:
//     GENERATOR_BINDING_STATUS: SUBJECT_MISMATCH
//     GENERATOR_AUTHORITATIVE_FOR_DIGEST: false
//     GENERATOR_STALE: false
//     GENERATOR_COMMIT_BINDING: MATCH
//     GENERATOR_SUBJECT_BINDING: MISMATCH
//
// These tests drive the full resolver path through
// ResolveAutoModeExplicitRange and RenderLifecycle. The proof
// is end-to-end: hermetic repositories, real resolver, real
// renderer.
package digest

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/s1onique/leamas/internal/factory/authority"
)

// buildHistoricalFixture creates a 3-commit repository:
//
//	A -- B -- C(HEAD)
//
// and returns (repo, a, b, c). The fixture is hermetic: the
// worktree is clean and HEAD is C.
func buildHistoricalFixture(t *testing.T) (repo, a, b, c string) {
	t.Helper()
	repo = t.TempDir()
	initGit(t, repo)
	mustWriteFile(t, filepath.Join(repo, "file.txt"), "v0\n")
	runGit(t, repo, "add", "file.txt")
	runGit(t, repo, "commit", "-m", "A: initial")
	a = runGit(t, repo, "rev-parse", "HEAD")

	mustWriteFile(t, filepath.Join(repo, "file.txt"), "v1\n")
	runGit(t, repo, "add", "file.txt")
	runGit(t, repo, "commit", "-m", "B: bump")
	b = runGit(t, repo, "rev-parse", "HEAD")

	mustWriteFile(t, filepath.Join(repo, "file.txt"), "v2\n")
	runGit(t, repo, "add", "file.txt")
	runGit(t, repo, "commit", "-m", "C: bump again")
	c = runGit(t, repo, "rev-parse", "HEAD")
	return repo, a, b, c
}

// TestRenderLifecycleCorrection01_GeneratorAtSubject_Authoritative
// drives the full resolver path: ResolveAutoModeExplicitRange
// with range "A..B" on an A--B--C(HEAD) fixture, with the
// generator's embedded commit equal to B. The verdict MUST
// be AUTHORITATIVE for the digest subject, even though the
// binary is "stale" relative to ambient HEAD=C.
//
// This is the decisive CORRECTION01 regression.
func TestRenderLifecycleCorrection01_GeneratorAtSubject_Authoritative(t *testing.T) {
	repo, a, b, c := buildHistoricalFixture(t)
	_ = c // ambient HEAD is C; the digest subject is B.

	// Drive the resolver explicitly. The resolved mode will
	// have AuthorityStatus=AuthorityExplicitRange,
	// LifecycleSubjectRange=B, HeadCommit=C.
	resolved, err := ResolveAutoModeExplicitRange(repo, "", a+".."+b)
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if resolved.LifecycleSubjectRange != b {
		t.Fatalf("LifecycleSubjectRange = %q, want %q",
			resolved.LifecycleSubjectRange, b)
	}
	if resolved.HeadCommit == "" {
		t.Fatalf("HeadCommit not populated by resolver")
	}

	// Build a synthetic digest where the embedded generator
	// commit is B (the resolved subject). The renderer's job is
	// to bind against the explicit-range right endpoint.
	mode := *resolved
	mode.GeneratorCommit = b
	// The real resolveAutoModeWith computed GeneratorStale from
	// the running binary's embedded commit vs HEAD; we override
	// it to match the simulated "generator=B" identity.
	mode.GeneratorStale = b != mode.HeadCommit
	mode.StaleReason = "embedded leamas commit does not match repository HEAD"
	// HeadCommit remains ambient HEAD (C). The renderer must
	// resolve the subject from LifecycleSubjectRange (B) before
	// falling back to LifecycleSubject or HeadCommit.
	rendered := RenderLifecycle(&mode)

	for _, want := range []string{
		"GENERATOR_COMMIT_MATCHES_HEAD: false",
		"GENERATOR_BINDING_STATUS: AUTHORITATIVE",
		"GENERATOR_COMMIT_BINDING: MISMATCH",
		"GENERATOR_SUBJECT_BINDING: MATCH",
		"GENERATOR_AUTHORITATIVE_FOR_DIGEST: true",
		"GENERATOR_WARNING_CODE: none",
		"GENERATOR_STALE: true",
		"GENERATOR_STALE_BASIS: commit_vs_repository_head",
	} {
		if !strings.Contains(rendered, want) {
			t.Errorf("CORRECTION01 historical-range authoritative render missing %q:\n%s",
				want, rendered)
		}
	}
}

// TestRenderLifecycleCorrection01_GeneratorAtHEAD_NotSubject
// drives the same fixture but with the generator's embedded
// commit equal to ambient HEAD (C). The legacy freshness
// signal is fine (commit equals HEAD), but the digest subject
// is B and the binary does NOT correspond to B. The verdict
// MUST be SUBJECT_MISMATCH, AUTHORITATIVE=false.
func TestRenderLifecycleCorrection01_GeneratorAtHEAD_NotSubject(t *testing.T) {
	repo, a, b, c := buildHistoricalFixture(t)

	resolved, err := ResolveAutoModeExplicitRange(repo, "", a+".."+b)
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if resolved.LifecycleSubjectRange != b {
		t.Fatalf("LifecycleSubjectRange = %q, want %q",
			resolved.LifecycleSubjectRange, b)
	}

	mode := *resolved
	mode.GeneratorCommit = c // binary built from HEAD, not from B
	// Reset legacy stale flag to match simulated identity:
	// generator=C, HEAD=non-C so the legacy freshness signal
	// fires for this case.
	mode.GeneratorStale = c != mode.HeadCommit
	rendered := RenderLifecycle(&mode)

	for _, want := range []string{
		"GENERATOR_COMMIT_MATCHES_HEAD: true",
		"GENERATOR_BINDING_STATUS: SUBJECT_MISMATCH",
		"GENERATOR_COMMIT_BINDING: MATCH",
		"GENERATOR_SUBJECT_BINDING: MISMATCH",
		"GENERATOR_AUTHORITATIVE_FOR_DIGEST: false",
		"GENERATOR_WARNING_CODE: GENERATOR_SUBJECT_MISMATCH",
		"GENERATOR_STALE: false",
	} {
		if !strings.Contains(rendered, want) {
			t.Errorf("CORRECTION01 historical-range non-authoritative render missing %q:\n%s",
				want, rendered)
		}
	}
}

// TestRenderLifecycleCorrection01_SubjectResolutionOrder pins
// the CORRECTION01 subject resolution order at the renderer
// boundary. The resolver MUST consult LifecycleSubjectRange
// before LifecycleSubject before HeadCommit.
//
// CORRECTION02 amendment: subject resolution is now
// authority-sensitive. The chain LifecycleSubjectRange ->
// LifecycleSubject -> HeadCommit applies ONLY to
// AuthorityExplicitRange. For all other authorities the chain
// is LifecycleSubject -> HeadCommit (no range fallback). The
// rows in this matrix therefore carry explicit AuthorityStatus
// values to lock the per-axis policy.
func TestRenderLifecycleCorrection01_SubjectResolutionOrder(t *testing.T) {
	const x = "0123456789abcdef0123456789abcdef01234567"
	const y = "fedcba9876543210fedcba9876543210fedcba98"
	const z = "abcdefabcdefabcdefabcdefabcdefabcdefabcd"

	cases := []struct {
		name                               string
		auth                               authority.AuthorityStatus
		rangeV, lifecycle, head, generator string
		wantSubjectBinding                 string
		wantStatus                         string
	}{
		{
			name:   "ExplicitRange_LifecycleSubjectRange_used",
			auth:   authority.AuthorityExplicitRange,
			rangeV: y, lifecycle: x, head: x, generator: y,
			wantSubjectBinding: "MATCH",
			wantStatus:         "AUTHORITATIVE",
		},
		{
			name:   "ExplicitRange_unresolved_endpoint_yields_unbound",
			auth:   authority.AuthorityExplicitRange,
			rangeV: "", lifecycle: x, head: z, generator: x,
			wantSubjectBinding: "UNBOUND",
			wantStatus:         "IDENTITY_UNBOUND",
		},
		{
			name:   "AuthoritativeClosed_falls_through_to_HeadCommit",
			auth:   authority.AuthorityAuthoritativeClosed,
			rangeV: "", lifecycle: "", head: x, generator: x,
			wantSubjectBinding: "MATCH",
			wantStatus:         "AUTHORITATIVE",
		},
		{
			name:   "AuthoritativeClosed_LifecycleSubject_wins_over_Head",
			auth:   authority.AuthorityAuthoritativeClosed,
			rangeV: "", lifecycle: x, head: z, generator: x,
			wantSubjectBinding: "MATCH",
			wantStatus:         "AUTHORITATIVE",
		},
		{
			name:   "AuthoritativeClosed_all_empty_yields_unbound",
			auth:   authority.AuthorityAuthoritativeClosed,
			rangeV: "", lifecycle: "", head: "", generator: x,
			wantSubjectBinding: "UNBOUND",
			wantStatus:         "IDENTITY_UNBOUND",
		},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			rendered := RenderLifecycle(&ResolvedMode{
				HeadCommit:            tc.head,
				GeneratorCommit:       tc.generator,
				LifecycleSubject:      tc.lifecycle,
				LifecycleSubjectRange: tc.rangeV,
				AuthorityStatus:       tc.auth,
				IsClean:               true,
			})
			if !strings.Contains(rendered,
				"GENERATOR_SUBJECT_BINDING: "+tc.wantSubjectBinding) {
				t.Errorf("subject binding mismatch:\n%s", rendered)
			}
			if !strings.Contains(rendered,
				"GENERATOR_BINDING_STATUS: "+tc.wantStatus) {
				t.Errorf("binding status mismatch:\n%s", rendered)
			}
		})
	}
}

// TestExplicitRangeRightEndpoint_Resolution locks the
// AuthorityExplicitRange path's right-endpoint resolution.
// The resolver MUST populate RangeSubjectEnd with the resolved
// full OID of the right endpoint of the explicit range.
func TestExplicitRangeRightEndpoint_Resolution(t *testing.T) {
	repo, a, b, _ := buildHistoricalFixture(t)

	resolved, err := authority.Resolve(authority.ResolverOptions{
		RepoRoot:      repo,
		ExplicitRange: a + ".." + b,
	})
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if resolved.AuthorityStatus != authority.AuthorityExplicitRange {
		t.Fatalf("status = %q, want %q",
			resolved.AuthorityStatus, authority.AuthorityExplicitRange)
	}
	if resolved.RangeSubjectEnd != b {
		t.Errorf("RangeSubjectEnd = %q, want %q (right endpoint of %q..%q)",
			resolved.RangeSubjectEnd, b, a, b)
	}
}

// TestExplicitRangeRightEndpoint_BareRev locks the bare-rev
// form (no ".."): the resolver MUST treat the entire expr as
// the right endpoint.
func TestExplicitRangeRightEndpoint_BareRev(t *testing.T) {
	repo, _, b, _ := buildHistoricalFixture(t)

	resolved, err := authority.Resolve(authority.ResolverOptions{
		RepoRoot:      repo,
		ExplicitRange: b,
	})
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if resolved.RangeSubjectEnd != b {
		t.Errorf("RangeSubjectEnd = %q, want %q (bare-rev form)",
			resolved.RangeSubjectEnd, b)
	}
}

// TestExplicitRangeRightEndpoint_MalformedFailsSoft locks the
// fail-soft contract: malformed right endpoint yields an empty
// RangeSubjectEnd but the resolution still classifies as
// AuthorityExplicitRange.
func TestExplicitRangeRightEndpoint_MalformedFailsSoft(t *testing.T) {
	repo, a, _, _ := buildHistoricalFixture(t)
	resolved, err := authority.Resolve(authority.ResolverOptions{
		RepoRoot:      repo,
		ExplicitRange: a + "..not-a-valid-rev",
	})
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if resolved.AuthorityStatus != authority.AuthorityExplicitRange {
		t.Fatalf("status = %q, want %q",
			resolved.AuthorityStatus, authority.AuthorityExplicitRange)
	}
	if resolved.RangeSubjectEnd != "" {
		t.Errorf("RangeSubjectEnd = %q, want empty (malformed rev)",
			resolved.RangeSubjectEnd)
	}
}

// helper: os.WriteFile with t.Fatal on error.
func mustWriteFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
