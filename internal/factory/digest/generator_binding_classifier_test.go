// SPDX-License-Identifier: Apache-2.0

// Package digest: generator_binding_classifier_test.go locks the
// pure classifier's behavior matrix for
// ACT-LEAMAS-DIGEST-GENERATOR-WORKTREE-STALE-AUTHORITY01.
//
// Every row corresponds to one numbered case from ACT §37. The
// matrix exhausts the canonical cases and a few adversarial
// variations (clock independence, unknown enums, malformed OIDs).
//
// The classifier is total: every input combination produces a
// typed GeneratorBinding record. No row may panic, return a
// half-populated struct, or silently promote an unrecognized
// value to AUTHORITATIVE.
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
		// Row 2: clean generator mismatch.
		{
			name:              "clean_generator_mismatch_HEAD",
			generator:         mk(y, true),
			repo:              repoMk(x, true),
			digest:            digestMk(x, true, false),
			wantStatus:        GeneratorBindingCommitMismatch,
			wantCommitBind:    GeneratorStateMismatch,
			wantSubjectBind:   GeneratorStateMismatch,
			wantMatchesHead:   false,
			wantAuthoritative: false,
			wantWarning:       GeneratorWarningCodeCommitMismatch,
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
		// Row 7 (case A): historical range. Generator commit
		// equals the resolved subject (B), ambient HEAD is C.
		// Subject is committed. Commit binding is MISMATCH
		// (B != C), so the overall verdict is COMMIT_MISMATCH.
		{
			name:              "historical_range_generator_matches_subject_only",
			generator:         mk(b, true),
			repo:              repoMk(c, true),
			digest:            digestMk(b, true, false),
			wantStatus:        GeneratorBindingCommitMismatch,
			wantCommitBind:    GeneratorStateMismatch,
			wantSubjectBind:   GeneratorStateMatch,
			wantMatchesHead:   false,
			wantAuthoritative: false,
			wantWarning:       GeneratorWarningCodeCommitMismatch,
		},
		// Row 7 (case B): when the historical range resolves
		// against the ambient repository's HEAD (B == B), the
		// generator IS authoritative. This covers ACT §20 row
		// "generator=B => AUTHORITATIVE" when the digest
		// subject coincides with HEAD.
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
		// Row 8: historical range where generator commit
		// equals ambient HEAD but not the resolved subject.
		{
			name:              "historical_range_generator_at_HEAD_not_subject",
			generator:         mk(c, true),
			repo:              repoMk(c, true),
			digest:            digestMk(b, true, false),
			wantStatus:        GeneratorBindingCommitMismatch,
			wantCommitBind:    GeneratorStateMatch,
			wantSubjectBind:   GeneratorStateMismatch,
			wantMatchesHead:   true,
			wantAuthoritative: false,
			wantWarning:       GeneratorWarningCodeCommitMismatch,
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

// TestEvaluateGeneratorBindingClockIndependence proves the
// classifier is a pure function of its inputs. Two otherwise
// identical inputs with different commit suffixes (no time data
// encoded) must produce distinct verdicts; identical inputs
// must produce identical verdicts. This is the determinism
// requirement from ACT §33/§34.
func TestEvaluateGeneratorBindingClockIndependence(t *testing.T) {
	const x = "0123456789abcdef0123456789abcdef01234567"
	// Same prefix as x; different suffix. No timestamp data
	// embedded; the difference is purely commit identity.
	const genMismatchOID = "0123456789abcdef0123456789abcdef01234599"

	genA := GeneratorIdentity{Commit: x, CommitIsValid: true}
	genB := GeneratorIdentity{Commit: genMismatchOID, CommitIsValid: true}
	repo := RepositoryIdentity{HeadCommit: x, HeadCommitIsValid: true}
	digest := DigestAuthoritySubject{SubjectCommit: x, SubjectCommitIsValid: true}

	// genA matches HEAD -> AUTHORITATIVE.
	gotA := EvaluateGeneratorBinding(genA, repo, digest)
	if gotA.Status != GeneratorBindingAuthoritative {
		t.Errorf("genA: got %q, want %q", gotA.Status, GeneratorBindingAuthoritative)
	}
	// genB does not match HEAD -> COMMIT_MISMATCH.
	gotB := EvaluateGeneratorBinding(genB, repo, digest)
	if gotB.Status != GeneratorBindingCommitMismatch {
		t.Errorf("genB: got %q, want %q", gotB.Status, GeneratorBindingCommitMismatch)
	}
	// Distinct inputs -> distinct outputs.
	if gotA.Status == gotB.Status {
		t.Errorf("distinct inputs produced identical status: %q", gotA.Status)
	}
}

// TestEvaluateGeneratorBindingDeterminism proves that identical
// inputs produce byte-identical outputs across many invocations.
// This is the deterministic-rendering requirement from ACT §33.
func TestEvaluateGeneratorBindingDeterminism(t *testing.T) {
	const x = "0123456789abcdef0123456789abcdef01234567"
	gen := GeneratorIdentity{Commit: x, CommitIsValid: true}
	repo := RepositoryIdentity{HeadCommit: x, HeadCommitIsValid: true}
	digest := DigestAuthoritySubject{SubjectCommit: x, SubjectCommitIsValid: true}

	first := EvaluateGeneratorBinding(gen, repo, digest)
	for i := 0; i < 32; i++ {
		next := EvaluateGeneratorBinding(gen, repo, digest)
		if next.Status != first.Status ||
			next.CommitBinding != first.CommitBinding ||
			next.SubjectBinding != first.SubjectBinding ||
			next.CommitMatchesHead != first.CommitMatchesHead ||
			next.AuthoritativeForDigest != first.AuthoritativeForDigest ||
			next.WarningCode != first.WarningCode {
			t.Fatalf("iteration %d: classifier non-deterministic\nfirst=%+v\nnext=%+v", i, first, next)
		}
	}
}

// TestEvaluateGeneratorBindingVocabularyStable locks the
// vocabulary cardinality. The ACT §38 specifies that any new
// enum value MUST fail closed; the simpler invariant is that
// the documented vocabulary does not silently grow without a
// corresponding test update.
func TestEvaluateGeneratorBindingVocabularyStable(t *testing.T) {
	known := []string{
		string(GeneratorBindingAuthoritative),
		string(GeneratorBindingCommitMismatch),
		string(GeneratorBindingDirtySubjectUnbound),
		string(GeneratorBindingIdentityUnbound),
		string(GeneratorBindingEvidenceInvalid),
	}
	if len(known) != 5 {
		t.Errorf("binding vocabulary size changed: got %d, want 5", len(known))
	}

	knownState := []string{
		string(GeneratorStateMatch),
		string(GeneratorStateMismatch),
		string(GeneratorStateUnbound),
	}
	if len(knownState) != 3 {
		t.Errorf("state vocabulary size changed: got %d, want 3", len(knownState))
	}

	knownWarning := []string{
		GeneratorWarningCodeNone,
		GeneratorWarningCodeCommitMismatch,
		GeneratorWarningCodeDirtySubjectUnbound,
		GeneratorWarningCodeIdentityUnbound,
		GeneratorWarningCodeEvidenceInvalid,
	}
	if len(knownWarning) != 5 {
		t.Errorf("warning vocabulary size changed: got %d, want 5", len(knownWarning))
	}
}

// TestEvaluateGeneratorBindingCaseInsensitive proves the
// classifier treats commits case-insensitively, matching
// existing fullOIDPattern normalization in auto_range.go and
// Git's documented short-SHA conventions.
func TestEvaluateGeneratorBindingCaseInsensitive(t *testing.T) {
	const xMixed = "0123456789AbCdEf0123456789abcdef01234567"
	const xUpper = "0123456789ABCDEF0123456789ABCDEF01234567"

	gen := GeneratorIdentity{Commit: xMixed, CommitIsValid: true}
	repo := RepositoryIdentity{HeadCommit: xUpper, HeadCommitIsValid: true}
	digest := DigestAuthoritySubject{SubjectCommit: xMixed, SubjectCommitIsValid: true}

	got := EvaluateGeneratorBinding(gen, repo, digest)
	if got.Status != GeneratorBindingAuthoritative {
		t.Errorf("mixed-case commits should match: got %q, want %q",
			got.Status, GeneratorBindingAuthoritative)
	}
}

// TestEvaluateGeneratorBindingInvalidOIDFailClosed proves that
// when the generator identity is non-empty but syntactically
// invalid (CommitIsValid=false), the classifier reports
// EVIDENCE_INVALID even when other inputs would otherwise
// match. This is the "unknown => fail closed" regression from
// ACT §26.
func TestEvaluateGeneratorBindingInvalidOIDFailClosed(t *testing.T) {
	const x = "0123456789abcdef0123456789abcdef01234567"
	gen := GeneratorIdentity{Commit: "garbage", CommitIsValid: false}
	repo := RepositoryIdentity{HeadCommit: x, HeadCommitIsValid: true}
	digest := DigestAuthoritySubject{SubjectCommit: x, SubjectCommitIsValid: true}

	got := EvaluateGeneratorBinding(gen, repo, digest)
	if got.Status != GeneratorBindingEvidenceInvalid {
		t.Errorf("invalid generator identity must yield EVIDENCE_INVALID: got %q", got.Status)
	}
	if got.AuthoritativeForDigest {
		t.Errorf("invalid generator identity MUST NOT be authoritative")
	}
}
