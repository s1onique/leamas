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
	//
	// IMPORTANT (CORRECTION01): the overall verdict is driven by
	// the SUBJECT binding (generator ↔ digest subject), NOT by the
	// commit binding (generator ↔ ambient HEAD). The doctrine is:
	//
	//	GENERATOR AUTHORITY MUST BIND TO THE DIGEST SUBJECT,
	//	NOT MERELY TO AMBIENT HEAD.
	//
	// Commit binding is recorded verbatim on the result and
	// surfaced in the digest as the legacy freshness signal, but
	// it MUST NOT short-circuit the subject verdict. A historical
	// digest (A..B) with generator=B and ambient HEAD=C is fully
	// authoritative for the digest subject, even though the
	// generator is "stale" relative to ambient HEAD. The legacy
	// GENERATOR_STALE flag captures the latter signal; the new
	// GENERATOR_AUTHORITATIVE_FOR_DIGEST captures the former.
	result.Status, result.AuthoritativeForDigest, result.WarningCode =
		composeGeneratorVerdict(generator, repo, digest, result.SubjectBinding)

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
// then the subject-binding predicates.
//
// CORRECTION01: the ambient-HEAD axis (commit binding) is
// deliberately NOT consulted here. The legacy GENERATOR_STALE
// signal is surfaced verbatim by the renderer on the
// GENERATOR_COMMIT_MATCHES_HEAD / GENERATOR_STALE fields, which
// are mechanical mirrors of the per-axis CommitBinding. The
// overall verdict reflects ONLY the digest-subject authority
// question, because that is the only question a digest consumer
// actually needs answered:
//
//	GENERATOR AUTHORITY MUST BIND TO THE DIGEST SUBJECT,
//	NOT MERELY TO AMBIENT HEAD.
//
// A historical digest (A..B) with generator=B and ambient
// HEAD=C is fully AUTHORITATIVE for the digest subject. The
// fact that the binary is "stale" relative to ambient HEAD is
// documented by GENERATOR_STALE=true alongside
// AUTHORITATIVE=true — exactly the separation the ACT
// introduced.
//
// The classifier never silently promotes an unknown value to a
// valid-authoritative result. Any non-{valid identity, valid
// identity, valid subject} combination falls through to a
// fail-closed verdict.
func composeGeneratorVerdict(
	generator GeneratorIdentity,
	repo RepositoryIdentity,
	digest DigestAuthoritySubject,
	subjectBinding GeneratorStateBindingStatus,
) (GeneratorBindingStatus, bool, string) {
	// Stage A: invalid evidence (non-empty but malformed OID).
	// If any identity is non-empty but flagged invalid, the
	// classifier reports EVIDENCE_INVALID. This is distinct
	// from "identity is missing" — the latter is reported as
	// IDENTITY_UNBOUND.
	//
	// The repo HEAD check is intentionally part of this stage:
	// when the repository HEAD identity is malformed, the
	// GENERATOR_STALE signal is not meaningful and the
	// classifier must surface the ambiguity through the
	// overall verdict. CORRECTION01 doctrine: ambient-HEAD
	// mismatches are recorded on the per-axis CommitBinding
	// field (and rendered as GENERATOR_STALE) but do NOT
	// dominate the overall status.
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

	// Stage C: subject mismatch. The generator's embedded
	// commit does not equal the digest subject commit. The
	// generator cannot be authoritative for the digest
	// subject — it was not built from that commit.
	//
	// Note: the ambient-HEAD axis (commit binding) is
	// deliberately NOT consulted here. Historical ranges
	// (A..B) with generator=B and ambient HEAD=C report
	// SUBJECT_MISMATCH here when the digest subject is
	// neither B nor anything the binary matches. When the
	// digest subject equals B and the binary equals B, the
	// subject binding is MATCH and we fall through to the
	// AUTHORITATIVE verdict — regardless of ambient HEAD=C.
	if subjectBinding == GeneratorStateMismatch {
		return GeneratorBindingSubjectMismatch, false, GeneratorWarningCodeSubjectMismatch
	}

	// Stage D: dirty subject. The generator commit equals
	// the digest subject commit, but the subject itself
	// includes uncommitted source state. Commit identity
	// cannot establish authority for uncommitted source
	// state. This is the decisive regression the ACT
	// originally fixed: an old binary built from a dirty
	// tree may actually contain the change, but the digest
	// cannot prove it from commit identity alone.
	if subjectBinding == GeneratorStateUnbound && digest.Dirty {
		return GeneratorBindingDirtySubjectUnbound, false, GeneratorWarningCodeDirtySubjectUnbound
	}

	// Stage E: every predicate passed: generator commit
	// equals the digest subject commit, the subject is not
	// dirty, and every identity was available and valid.
	//
	// Note: this does NOT require the generator commit to
	// equal the repository HEAD. The doctrine is:
	// "GENERATOR AUTHORITY MUST BIND TO THE DIGEST SUBJECT,
	// NOT MERELY TO AMBIENT HEAD."
	if subjectBinding == GeneratorStateMatch {
		return GeneratorBindingAuthoritative, true, GeneratorWarningCodeNone
	}

	// Defensive default. Any combination not covered above is
	// treated as ambiguous and reported as IDENTITY_UNBOUND.
	// The classifier never silently promotes an unrecognized
	// combination to AUTHORITATIVE.
	return GeneratorBindingIdentityUnbound, false, GeneratorWarningCodeIdentityUnbound
}
