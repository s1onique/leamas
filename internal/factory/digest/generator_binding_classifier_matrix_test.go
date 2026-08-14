// SPDX-License-Identifier: Apache-2.0

// Package digest: generator_binding_classifier_matrix_test.go
// contains the canonical input/verdict matrix for the pure
// EvaluateGeneratorBinding classifier required by
// ACT-LEAMAS-DIGEST-GENERATOR-WORKTREE-STALE-AUTHORITY01.
//
// Every row corresponds to one numbered case from ACT §37
// (and CORRECTION01 additions). The matrix exhausts the
// canonical cases and a few adversarial variations.
//
// Split from generator_binding_classifier_test.go so the
// matrix test (the largest single fixture) lives in its own
// file under the LLM-friendliness 400-line guidance.
package digest

import "testing"

// TestEvaluateGeneratorBindingMatrix drives the canonical
// matrix. Each row names the case, the input identities, and
// the expected verdict.
func TestEvaluateGeneratorBindingMatrix(t *testing.T) {
	const x = "0123456789abcdef0123456789abcdef01234567"
	const y = "fedcba9876543210fedcba9876543210fedcba98"
	const b = "1111111111111111111111111111111111111111"
	const c = "2222222222222222222222222222222222222222"
	const garbage = "not-a-valid-oid"

	mk := func(commit string, valid bool) GeneratorIdentity {
		return GeneratorIdentity{Commit: commit, CommitIsValid: valid}
	}
	repoMk := func(commit string, valid bool) RepositoryIdentity {
		return RepositoryIdentity{HeadCommit: commit, HeadCommitIsValid: valid}
	}
	digestMk := func(commit string, valid bool, dirty bool) DigestAuthoritySubject {
		return DigestAuthoritySubject{
			SubjectCommit:        commit,
			SubjectCommitIsValid: valid,
			Dirty:                dirty,
		}
	}

	cases := []struct {
		name              string
		generator         GeneratorIdentity
		repo              RepositoryIdentity
		digest            DigestAuthoritySubject
		wantStatus        GeneratorBindingStatus
		wantCommitBind    GeneratorStateBindingStatus
		wantSubjectBind   GeneratorStateBindingStatus
		wantMatchesHead   bool
		wantAuthoritative bool
		wantWarning       string
	}{
		// Row 1: clean generator, clean committed subject.
		{
			name:              "clean_equals_HEAD_committed",
			generator:         mk(x, true),
			repo:              repoMk(x, true),
			digest:            digestMk(x, true, false),
			wantStatus:        GeneratorBindingAuthoritative,
			wantCommitBind:    GeneratorStateMatch,
			wantSubjectBind:   GeneratorStateMatch,
			wantMatchesHead:   true,
			wantAuthoritative: true,
			wantWarning:       GeneratorWarningCodeNone,
		},
		// Row 2: clean generator mismatch. Generator (y) differs
		// from both HEAD (x) and the digest subject (x). Under
		// CORRECTION01 the overall verdict is SUBJECT_MISMATCH
		// (the binary does not correspond to the digest
		// subject). Commit binding also records MISMATCH
		// (legacy freshness signal). The overall status is
		// the SUBJECT verdict because that is the binding's
		// reference point.
		{
			name:              "clean_generator_mismatch_HEAD",
			generator:         mk(y, true),
			repo:              repoMk(x, true),
			digest:            digestMk(x, true, false),
			wantStatus:        GeneratorBindingSubjectMismatch,
			wantCommitBind:    GeneratorStateMismatch,
			wantSubjectBind:   GeneratorStateMismatch,
			wantMatchesHead:   false,
			wantAuthoritative: false,
			wantWarning:       GeneratorWarningCodeSubjectMismatch,
		},
		// Row 3: dirty tracked subject.
		{
			name:              "dirty_tracked_unbound",
			generator:         mk(x, true),
			repo:              repoMk(x, true),
			digest:            digestMk(x, true, true),
			wantStatus:        GeneratorBindingDirtySubjectUnbound,
			wantCommitBind:    GeneratorStateMatch,
			wantSubjectBind:   GeneratorStateUnbound,
			wantMatchesHead:   true,
			wantAuthoritative: false,
			wantWarning:       GeneratorWarningCodeDirtySubjectUnbound,
		},
		// Row 4: staged-only subject.
		{
			name:              "staged_only_unbound",
			generator:         mk(x, true),
			repo:              repoMk(x, true),
			digest:            digestMk(x, true, true),
			wantStatus:        GeneratorBindingDirtySubjectUnbound,
			wantCommitBind:    GeneratorStateMatch,
			wantSubjectBind:   GeneratorStateUnbound,
			wantMatchesHead:   true,
			wantAuthoritative: false,
			wantWarning:       GeneratorWarningCodeDirtySubjectUnbound,
		},
		// Row 5: untracked-only subject.
		{
			name:              "untracked_only_unbound",
			generator:         mk(x, true),
			repo:              repoMk(x, true),
			digest:            digestMk(x, true, true),
			wantStatus:        GeneratorBindingDirtySubjectUnbound,
			wantCommitBind:    GeneratorStateMatch,
			wantSubjectBind:   GeneratorStateUnbound,
			wantMatchesHead:   true,
			wantAuthoritative: false,
			wantWarning:       GeneratorWarningCodeDirtySubjectUnbound,
		},
		// Row 6: mixed dirty subject.
		{
			name:              "mixed_dirty_unbound",
			generator:         mk(x, true),
			repo:              repoMk(x, true),
			digest:            digestMk(x, true, true),
			wantStatus:        GeneratorBindingDirtySubjectUnbound,
			wantCommitBind:    GeneratorStateMatch,
			wantSubjectBind:   GeneratorStateUnbound,
			wantMatchesHead:   true,
			wantAuthoritative: false,
			wantWarning:       GeneratorWarningCodeDirtySubjectUnbound,
		},
		// Row 7 (case A): historical range A..B. Generator
		// commit equals the resolved subject (B); ambient HEAD
		// is C. Subject is committed.
		//
		// CORRECTION01 doctrine: the overall verdict is driven
		// by the SUBJECT axis (generator ↔ digest subject),
		// NOT the ambient-HEAD axis. Commit binding is
		// MISMATCH (legacy freshness signal), but the digest
		// subject authority is MATCH. The verifier MUST see:
		//
		//	GENERATOR_BINDING_STATUS: AUTHORITATIVE
		//	GENERATOR_COMMIT_BINDING:  MISMATCH   (legacy stale)
		//	GENERATOR_SUBJECT_BINDING: MATCH      (new authority)
		//	GENERATOR_AUTHORITATIVE_FOR_DIGEST: true
		//	GENERATOR_STALE: true                 (legacy field)
		//	GENERATOR_COMMIT_MATCHES_HEAD: false
		{
			name:              "historical_range_generator_matches_subject_only",
			generator:         mk(b, true),
			repo:              repoMk(c, true),
			digest:            digestMk(b, true, false),
			wantStatus:        GeneratorBindingAuthoritative,
			wantCommitBind:    GeneratorStateMismatch,
			wantSubjectBind:   GeneratorStateMatch,
			wantMatchesHead:   false,
			wantAuthoritative: true,
			wantWarning:       GeneratorWarningCodeNone,
		},
		// Row 7 (case B): historical range whose right
		// endpoint coincides with ambient HEAD (B == B).
		// Generator commit also equals B. Both axes match;
		// verdict is AUTHORITATIVE.
		{
			name:              "historical_range_generator_and_subject_at_HEAD",
			generator:         mk(b, true),
			repo:              repoMk(b, true),
			digest:            digestMk(b, true, false),
			wantStatus:        GeneratorBindingAuthoritative,
			wantCommitBind:    GeneratorStateMatch,
			wantSubjectBind:   GeneratorStateMatch,
			wantMatchesHead:   true,
			wantAuthoritative: true,
			wantWarning:       GeneratorWarningCodeNone,
		},
		// Row 8 (CORRECTION01): historical range where the
		// generator commit equals ambient HEAD (C) but NOT
		// the resolved subject (B). The legacy freshness
		// signal is fine; the new authority signal fails
		// because the binary does not correspond to the
		// resolved subject. The overall verdict is now
		// SUBJECT_MISMATCH (distinct from COMMIT_MISMATCH).
		{
			name:              "historical_range_generator_at_HEAD_not_subject",
			generator:         mk(c, true),
			repo:              repoMk(c, true),
			digest:            digestMk(b, true, false),
			wantStatus:        GeneratorBindingSubjectMismatch,
			wantCommitBind:    GeneratorStateMatch,
			wantSubjectBind:   GeneratorStateMismatch,
			wantMatchesHead:   true,
			wantAuthoritative: false,
			wantWarning:       GeneratorWarningCodeSubjectMismatch,
		},
		// Row 8b (CORRECTION01): historical range where
		// generator differs from BOTH the subject AND ambient
		// HEAD. Subject binding fails (SUBJECT_MISMATCH); the
		// commit binding also records MISMATCH (legacy
		// freshness signal), but the overall verdict is the
		// subject verdict because the digest subject is the
		// binding's reference point.
		{
			name:              "historical_range_generator_mismatches_both",
			generator:         mk(x, true),
			repo:              repoMk(c, true),
			digest:            digestMk(b, true, false),
			wantStatus:        GeneratorBindingSubjectMismatch,
			wantCommitBind:    GeneratorStateMismatch,
			wantSubjectBind:   GeneratorStateMismatch,
			wantMatchesHead:   false,
			wantAuthoritative: false,
			wantWarning:       GeneratorWarningCodeSubjectMismatch,
		},
		// Row 9: missing generator identity.
		{
			name:              "missing_generator_identity",
			generator:         mk("", false),
			repo:              repoMk(x, true),
			digest:            digestMk(x, true, false),
			wantStatus:        GeneratorBindingIdentityUnbound,
			wantCommitBind:    GeneratorStateUnbound,
			wantSubjectBind:   GeneratorStateUnbound,
			wantMatchesHead:   false,
			wantAuthoritative: false,
			wantWarning:       GeneratorWarningCodeIdentityUnbound,
		},
		// Row 10: invalid generator identity.
		{
			name:              "invalid_generator_identity",
			generator:         mk(garbage, false),
			repo:              repoMk(x, true),
			digest:            digestMk(x, true, false),
			wantStatus:        GeneratorBindingEvidenceInvalid,
			wantCommitBind:    GeneratorStateUnbound,
			wantSubjectBind:   GeneratorStateUnbound,
			wantMatchesHead:   false,
			wantAuthoritative: false,
			wantWarning:       GeneratorWarningCodeEvidenceInvalid,
		},
		// Row 11: unresolved digest subject (empty SubjectCommit).
		{
			name:              "unresolved_digest_subject",
			generator:         mk(x, true),
			repo:              repoMk(x, true),
			digest:            digestMk("", false, false),
			wantStatus:        GeneratorBindingIdentityUnbound,
			wantCommitBind:    GeneratorStateMatch,
			wantSubjectBind:   GeneratorStateUnbound,
			wantMatchesHead:   true,
			wantAuthoritative: false,
			wantWarning:       GeneratorWarningCodeIdentityUnbound,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			got := EvaluateGeneratorBinding(tc.generator, tc.repo, tc.digest)
			if got.Status != tc.wantStatus {
				t.Errorf("Status: got %q, want %q", got.Status, tc.wantStatus)
			}
			if got.CommitBinding != tc.wantCommitBind {
				t.Errorf("CommitBinding: got %q, want %q", got.CommitBinding, tc.wantCommitBind)
			}
			if got.SubjectBinding != tc.wantSubjectBind {
				t.Errorf("SubjectBinding: got %q, want %q", got.SubjectBinding, tc.wantSubjectBind)
			}
			if got.CommitMatchesHead != tc.wantMatchesHead {
				t.Errorf("CommitMatchesHead: got %t, want %t", got.CommitMatchesHead, tc.wantMatchesHead)
			}
			if got.AuthoritativeForDigest != tc.wantAuthoritative {
				t.Errorf("AuthoritativeForDigest: got %t, want %t", got.AuthoritativeForDigest, tc.wantAuthoritative)
			}
			if got.WarningCode != tc.wantWarning {
				t.Errorf("WarningCode: got %q, want %q", got.WarningCode, tc.wantWarning)
			}
		})
	}
}
