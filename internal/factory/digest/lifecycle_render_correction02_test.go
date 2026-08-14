// SPDX-License-Identifier: Apache-2.0

// Package digest: lifecycle_render_correction02_test.go locks
// the CORRECTION02 invariants for ACT-LEAMAS-DIGEST-GENERATOR-WORKTREE-STALE-AUTHORITY01.
//
// The decisive regressions locked here:
//
//  1. AuthorityExplicitRange with unresolved RangeSubjectEnd
//     MUST fail closed: the renderer MUST NOT fall back to
//     ambient HEAD for the digest subject. An unresolved
//     explicit-range right endpoint produces
//     IDENTITY_UNBOUND with AUTHORITATIVE=false, regardless of
//     whether the generator's embedded commit happens to equal
//     ambient HEAD.
//
//  2. The three-dot form `A...B` is REJECTED at the resolver
//     boundary. The resolver MUST NOT silently interpret it as
//     `A..B` (which would resolve the right endpoint to `B`
//     via `rev-parse --verify ".B^{commit}"`).
package digest

import (
	"strings"
	"testing"

	"github.com/s1onique/leamas/internal/factory/authority"
)

// TestRenderLifecycleCorrection02_UnresolvedExplicitRangeFailsClosed
// is the CORRECTION02 decisive regression.
//
// Without the fix: when AuthorityExplicitRange and
// LifecycleSubjectRange are empty, the renderer would fall back
// to ambient HEAD as the digest subject. If the generator
// commit happens to equal ambient HEAD (the most common
// development case), the renderer would produce
// AUTHORITATIVE=true and SUBJECT_BINDING=MATCH — falsely
// certifying authority for a digest whose subject was never
// actually resolved.
//
// With the fix: the renderer refuses to fall back to HEAD for
// AuthorityExplicitRange, the subject stays empty, the
// classifier reports IDENTITY_UNBOUND, and AUTHORITATIVE=false.
func TestRenderLifecycleCorrection02_UnresolvedExplicitRangeFailsClosed(t *testing.T) {
	const x = "0123456789abcdef0123456789abcdef01234567"

	mode := &ResolvedMode{
		HeadCommit:            x,
		GeneratorCommit:       x, // equal to HEAD; the dangerous case
		LifecycleSubject:      "",
		LifecycleSubjectRange: "", // unresolved explicit-range right endpoint
		AuthorityStatus:       authority.AuthorityExplicitRange,
		IsClean:               true,
	}
	rendered := RenderLifecycle(mode)

	for _, want := range []string{
		"GENERATOR_COMMIT_MATCHES_HEAD: true", // commit equals HEAD
		"GENERATOR_BINDING_STATUS: IDENTITY_UNBOUND",
		"GENERATOR_COMMIT_BINDING: MATCH",
		"GENERATOR_SUBJECT_BINDING: UNBOUND",
		"GENERATOR_AUTHORITATIVE_FOR_DIGEST: false",
		"GENERATOR_WARNING_CODE: GENERATOR_IDENTITY_UNBOUND",
		"GENERATOR_STALE: false",
	} {
		if !strings.Contains(rendered, want) {
			t.Errorf("CORRECTION02 unresolved-explicit-range render missing %q:\n%s",
				want, rendered)
		}
	}

	// AUTHORITY_STATUS must surface ExplicitRange so reviewers
	// can see the resolver classification that triggered the
	// fail-closed path.
	if !strings.Contains(rendered, "AUTHORITY_STATUS: ExplicitRange") {
		t.Errorf("CORRECTION02 expected AUTHORITY_STATUS: ExplicitRange in render:\n%s",
			rendered)
	}
}

// TestRenderLifecycleCorrection02_ResolvedExplicitRangeIsAuthoritative
// pins the positive case: when RangeSubjectEnd is resolved and
// matches the generator commit, the verdict is AUTHORITATIVE
// even though the generator commit does NOT equal ambient HEAD
// (this is the CORRECTION01 historical-range case, asserted
// here as a regression guard for CORRECTION02's stricter
// fallback policy).
func TestRenderLifecycleCorrection02_ResolvedExplicitRangeIsAuthoritative(t *testing.T) {
	const b = "1111111111111111111111111111111111111111"
	const c = "2222222222222222222222222222222222222222"

	mode := &ResolvedMode{
		HeadCommit:            c, // ambient HEAD is C
		GeneratorCommit:       b, // generator built from B (subject)
		LifecycleSubjectRange: b, // explicit-range right endpoint resolved to B
		AuthorityStatus:       authority.AuthorityExplicitRange,
		GeneratorStale:        true, // B != C: legacy freshness signal fires
		IsClean:               true,
	}
	rendered := RenderLifecycle(mode)

	for _, want := range []string{
		"GENERATOR_COMMIT_MATCHES_HEAD: false",
		"GENERATOR_BINDING_STATUS: AUTHORITATIVE",
		"GENERATOR_COMMIT_BINDING: MISMATCH",
		"GENERATOR_SUBJECT_BINDING: MATCH",
		"GENERATOR_AUTHORITATIVE_FOR_DIGEST: true",
		"GENERATOR_WARNING_CODE: none",
		"GENERATOR_STALE: true",
	} {
		if !strings.Contains(rendered, want) {
			t.Errorf("CORRECTION02 resolved-explicit-range authoritative render missing %q:\n%s",
				want, rendered)
		}
	}
}

// TestRenderLifecycleCorrection02_NonExplicitRangeKeepsHEADFallback
// pins the negative case: for non-explicit authorities (auto
// mode, manifest-derived, etc.) the documented fallback chain
// LifecycleSubject -> HeadCommit MUST continue to apply. Only
// AuthorityExplicitRange is exempt from the HEAD fallback.
//
// This guards against an over-correction that would suppress
// the HEAD fallback universally.
func TestRenderLifecycleCorrection02_NonExplicitRangeKeepsHEADFallback(t *testing.T) {
	const x = "0123456789abcdef0123456789abcdef01234567"

	mode := &ResolvedMode{
		HeadCommit:            x,
		GeneratorCommit:       x,
		LifecycleSubject:      "",                                     // not populated
		LifecycleSubjectRange: "",                                     // not populated
		AuthorityStatus:       authority.AuthorityAuthoritativeClosed, // NOT explicit
		IsClean:               true,
	}
	rendered := RenderLifecycle(mode)

	for _, want := range []string{
		"GENERATOR_COMMIT_MATCHES_HEAD: true",
		"GENERATOR_BINDING_STATUS: AUTHORITATIVE",
		"GENERATOR_COMMIT_BINDING: MATCH",
		"GENERATOR_SUBJECT_BINDING: MATCH",
		"GENERATOR_AUTHORITATIVE_FOR_DIGEST: true",
		"GENERATOR_WARNING_CODE: none",
		"GENERATOR_STALE: false",
	} {
		if !strings.Contains(rendered, want) {
			t.Errorf("CORRECTION02 non-explicit-range HEAD fallback missing %q:\n%s",
				want, rendered)
		}
	}
}

// TestExplicitRangeRightEndpoint_ThreeDotRejected pins the
// CORRECTION02 contract that the three-dot form is rejected
// at the resolver boundary. The range-scope diagnostic
// already rejects `A...B`; the resolver MUST follow the same
// policy.
func TestExplicitRangeRightEndpoint_ThreeDotRejected(t *testing.T) {
	repo, a, b, _ := buildHistoricalFixture(t)

	resolved, err := authority.Resolve(authority.ResolverOptions{
		RepoRoot:      repo,
		ExplicitRange: a + "..." + b, // explicit three-dot
	})
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if resolved.AuthorityStatus != authority.AuthorityExplicitRange {
		t.Fatalf("status = %q, want %q",
			resolved.AuthorityStatus, authority.AuthorityExplicitRange)
	}
	// CORRECTION02: three-dot MUST NOT silently resolve to B.
	if resolved.RangeSubjectEnd != "" {
		t.Errorf("RangeSubjectEnd = %q, want empty (three-dot rejected)",
			resolved.RangeSubjectEnd)
	}
	// And the digest range is preserved verbatim so the
	// downstream CLI surface surfaces the original artifact.
	if resolved.DigestRange != a+"..."+b {
		t.Errorf("DigestRange = %q, want %q",
			resolved.DigestRange, a+"..."+b)
	}
}
