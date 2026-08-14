// SPDX-License-Identifier: Apache-2.0

// Package digest: generator_binding_classifier.go implements the
// pure EvaluateGeneratorBinding function. The classifier is
// total — every input combination produces a typed
// GeneratorBinding record. It is intentionally fail-closed:
// ambiguous input always produces authoritative_for_digest=false.
//
// The classifier never reads the filesystem, never spawns Git
// subprocesses, and never consults the clock. The caller MUST
// resolve all identities before invocation.
package digest

import "strings"

// EvaluateGeneratorBinding is the pure binding classifier.
// It performs no I/O, no Git subprocesses, and no time lookup.
// Identities MUST be resolved before invocation.
//
// Inputs:
//
//	generator - the resolved commit identity of the running
//	           generator binary, plus its validity flag.
//	repo     - the resolved commit identity of the repository
//	           HEAD, plus its validity flag.
//	digest   - the resolved digest subject identity (commit +
//	           validity), plus a dirty flag recording whether the
//	           subject includes uncommitted source state.
//
// The function is total: every input combination produces a
// typed GeneratorBinding record. The returned struct is safe for
// direct use by renderers and tests.
//
// The default branches are critical: every identity field can
// be invalid (non-empty but syntactically not a valid OID). Any
// unrecognized value MUST NOT silently reach the
// authoritative-for-digest path. The doctrine is
// "AMBIGUOUS_BINDING_FAILS_CLOSED=true"; invalid inputs are
// definitionally ambiguous.
func EvaluateGeneratorBinding(generator GeneratorIdentity, repo RepositoryIdentity, digest DigestAuthoritySubject) GeneratorBinding {
	result := GeneratorBinding{
		Status:                 GeneratorBindingIdentityUnbound,
		CommitBinding:          GeneratorStateUnbound,
		SubjectBinding:         GeneratorStateUnbound,
		CommitMatchesHead:      false,
		AuthoritativeForDigest: false,
		WarningCode:            GeneratorWarningCodeIdentityUnbound,
	}

	// Stage 1: classify the per-axis bindings. We deliberately
	// compute CommitBinding and SubjectBinding independently
	// before composing the overall verdict. This matches the
	// structure of gate_evidence_classifier.go.

	result.CommitBinding = classifyCommitBinding(generator, repo)
	result.SubjectBinding = classifySubjectBinding(generator, digest)

	// Stage 2: invert the legacy freshness signal. The legacy
	// GENERATOR_STALE field uses "false" for "not stale" and
	// "true" for "stale". The new commit_matches_head surface
	// uses the affirmative "true" when commits match so the
	// semantic split is obvious to readers.
	result.CommitMatchesHead = result.CommitBinding == GeneratorStateMatch

	// Stage 3: compose the overall verdict from the per-axis
	// bindings, the dirty flag, and the validity of every
	// identity.
	result.Status, result.AuthoritativeForDigest, result.WarningCode =
		composeGeneratorVerdict(generator, repo, digest, result.CommitBinding, result.SubjectBinding)

	return result
}

// classifyCommitBinding returns MATCH iff generator commit
// equals repository HEAD (case-insensitive). Any other case is
// MISMATCH or UNBOUND depending on the validity flags.
func classifyCommitBinding(generator GeneratorIdentity, repo RepositoryIdentity) GeneratorStateBindingStatus {
	if !generator.CommitIsValid || !repo.HeadCommitIsValid {
		return GeneratorStateUnbound
	}
	if strings.EqualFold(generator.Commit, repo.HeadCommit) {
		return GeneratorStateMatch
	}
	return GeneratorStateMismatch
}

// classifySubjectBinding returns MATCH iff generator commit
// equals the digest subject commit AND the digest subject is
// not dirty (a commit-only proof cannot establish authority for
// uncommitted state).
func classifySubjectBinding(generator GeneratorIdentity, digest DigestAuthoritySubject) GeneratorStateBindingStatus {
	if !generator.CommitIsValid || !digest.SubjectCommitIsValid {
		return GeneratorStateUnbound
	}
	if digest.Dirty {
		// No commit-only proof can establish authority for a
		// dirty subject. The fact that commits match is
		// irrelevant — the subject includes uncommitted state
		// that the commit does not represent.
		return GeneratorStateUnbound
	}
	if strings.EqualFold(generator.Commit, digest.SubjectCommit) {
		return GeneratorStateMatch
	}
	return GeneratorStateMismatch
}

// composeGeneratorVerdict composes the overall status from the
// per-axis bindings, the dirty flag, and the validity flags.
// The ordering matters: invalid evidence is checked first
// (EVIDENCE_INVALID), then missing identity (IDENTITY_UNBOUND),
// then the binding predicates.
//
// The classifier never silently promotes an unknown value to a
// valid-authoritative result. Any non-{valid identity, valid
// identity, valid subject} combination falls through to a
// fail-closed verdict.
func composeGeneratorVerdict(
	generator GeneratorIdentity,
	repo RepositoryIdentity,
	digest DigestAuthoritySubject,
	commitBinding GeneratorStateBindingStatus,
	subjectBinding GeneratorStateBindingStatus,
) (GeneratorBindingStatus, bool, string) {
	// Stage A: invalid evidence (non-empty but malformed OID).
	// If any identity is non-empty but flagged invalid, the
	// classifier reports EVIDENCE_INVALID. This is distinct
	// from "identity is missing" — the latter is reported as
	// IDENTITY_UNBOUND.
	anyIdentityMalformed := (generator.Commit != "" && !generator.CommitIsValid) ||
		(repo.HeadCommit != "" && !repo.HeadCommitIsValid) ||
		(digest.SubjectCommit != "" && !digest.SubjectCommitIsValid)
	if anyIdentityMalformed {
		return GeneratorBindingEvidenceInvalid, false, GeneratorWarningCodeEvidenceInvalid
	}

	// Stage B: missing identity. Either commit is empty,
	// repository HEAD is empty, or digest subject is empty.
	anyIdentityMissing := generator.Commit == "" ||
		repo.HeadCommit == "" ||
		digest.SubjectCommit == ""
	if anyIdentityMissing {
		return GeneratorBindingIdentityUnbound, false, GeneratorWarningCodeIdentityUnbound
	}

	// Stage C: commit mismatch. If the generator commit does
	// not equal repository HEAD, the generator cannot be
	// authoritative for any digest that resolves against the
	// current working tree. This takes precedence over the
	// subject binding verdict: an explicit-range subject may
	// still match the generator commit while ambient HEAD does
	// not, but we report COMMIT_MISMATCH because the legacy
	// GENERATOR_STALE signal is also stale. The ACT specifies
	// commit-mismatch as the dominant verdict.
	if commitBinding == GeneratorStateMismatch {
		return GeneratorBindingCommitMismatch, false, GeneratorWarningCodeCommitMismatch
	}

	// Stage D: dirty subject. Commit binding is MATCH, but the
	// subject itself is dirty. Commit identity cannot establish
	// authority for uncommitted source state. This is the
	// decisive regression the ACT fixes: an old binary built
	// from a dirty tree may actually contain the change, but
	// the digest cannot prove it from commit identity alone.
	if subjectBinding == GeneratorStateUnbound && digest.Dirty {
		return GeneratorBindingDirtySubjectUnbound, false, GeneratorWarningCodeDirtySubjectUnbound
	}

	// Stage E: explicit-range subject mismatch. Generator
	// commit equals HEAD, but the digest subject is a
	// historical range endpoint that differs from HEAD. We
	// classify COMMIT_MISMATCH because the generator is not
	// authoritative for the historical subject — it was built
	// from a different commit.
	if subjectBinding == GeneratorStateMismatch {
		return GeneratorBindingCommitMismatch, false, GeneratorWarningCodeCommitMismatch
	}

	// Stage F: every predicate passed: generator commit
	// equals HEAD, generator commit equals subject, subject is
	// not dirty, every identity was available and valid.
	if commitBinding == GeneratorStateMatch && subjectBinding == GeneratorStateMatch {
		return GeneratorBindingAuthoritative, true, GeneratorWarningCodeNone
	}

	// Defensive default. Any combination not covered above is
	// treated as ambiguous and reported as IDENTITY_UNBOUND.
	// The classifier never silently promotes an unrecognized
	// combination to AUTHORITATIVE.
	return GeneratorBindingIdentityUnbound, false, GeneratorWarningCodeIdentityUnbound
}
