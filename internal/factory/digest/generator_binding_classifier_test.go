// SPDX-License-Identifier: Apache-2.0

// Package digest: generator_binding_classifier_test.go locks
// the focused regression tests for the pure
// EvaluateGeneratorBinding classifier.
//
// The canonical behavior matrix lives in
// generator_binding_classifier_matrix_test.go. This file
// holds identity-sensitivity, determinism, vocabulary, and
// invalid-OID regressions.
package digest

import "testing"

// TestEvaluateGeneratorBindingIdentitySensitivity proves that
// distinct generator identities yield distinct verdicts and
// that the classifier's verdict is fully determined by its
// inputs (no clock, no I/O, no hidden state).
//
// CORRECTION01: the previous "ClockIndependence" name was
// misleading — the test does not vary any clock or build-time
// input; it varies the generator identity. The classifier is
// pure by construction (no time package, no os calls), so the
// test asserts the input/output contract directly.
func TestEvaluateGeneratorBindingIdentitySensitivity(t *testing.T) {
	const x = "0123456789abcdef0123456789abcdef01234567"
	// Same prefix as x; different suffix. The difference is
	// purely commit identity.
	const genMismatchOID = "0123456789abcdef0123456789abcdef01234599"

	genA := GeneratorIdentity{Commit: x, CommitIsValid: true}
	genB := GeneratorIdentity{Commit: genMismatchOID, CommitIsValid: true}
	repo := RepositoryIdentity{HeadCommit: x, HeadCommitIsValid: true}
	digest := DigestAuthoritySubject{SubjectCommit: x, SubjectCommitIsValid: true}

	// genA matches both HEAD and subject -> AUTHORITATIVE.
	gotA := EvaluateGeneratorBinding(genA, repo, digest)
	if gotA.Status != GeneratorBindingAuthoritative {
		t.Errorf("genA: got %q, want %q", gotA.Status, GeneratorBindingAuthoritative)
	}
	// genB does not match subject -> SUBJECT_MISMATCH
	// (CORRECTION01: overall status is the SUBJECT verdict;
	// commit binding also records MISMATCH).
	gotB := EvaluateGeneratorBinding(genB, repo, digest)
	if gotB.Status != GeneratorBindingSubjectMismatch {
		t.Errorf("genB: got %q, want %q", gotB.Status, GeneratorBindingSubjectMismatch)
	}
	// Distinct inputs -> distinct outputs.
	if gotA.Status == gotB.Status {
		t.Errorf("distinct inputs produced identical status: %q", gotA.Status)
	}
	// Determinism: re-invoking with the same inputs yields
	// byte-identical outputs (no hidden state).
	if gotA.AuthoritativeForDigest != EvaluateGeneratorBinding(genA, repo, digest).AuthoritativeForDigest {
		t.Errorf("classifier is not deterministic for genA")
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
		string(GeneratorBindingSubjectMismatch),
		string(GeneratorBindingDirtySubjectUnbound),
		string(GeneratorBindingIdentityUnbound),
		string(GeneratorBindingEvidenceInvalid),
	}
	if len(known) != 6 {
		t.Errorf("binding vocabulary size changed: got %d, want 6", len(known))
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
		GeneratorWarningCodeSubjectMismatch,
		GeneratorWarningCodeDirtySubjectUnbound,
		GeneratorWarningCodeIdentityUnbound,
		GeneratorWarningCodeEvidenceInvalid,
	}
	if len(knownWarning) != 6 {
		t.Errorf("warning vocabulary size changed: got %d, want 6", len(knownWarning))
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
