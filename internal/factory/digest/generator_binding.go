// SPDX-License-Identifier: Apache-2.0

// Package digest: generator_binding.go declares the typed surface
// of the generator<->digest-subject binding classifier required by
// ACT-LEAMAS-DIGEST-GENERATOR-WORKTREE-STALE-AUTHORITY01.
//
// Doctrine:
//
//	COMMIT IDENTITY PROVES COMMITTED STATE.
//	IT DOES NOT PROVE UNCOMMITTED SOURCE STATE.
//	GENERATOR AUTHORITY MUST BIND TO THE DIGEST SUBJECT,
//	NOT MERELY TO AMBIENT HEAD.
//
// The classifier separates two distinct questions that were
// previously conflated under a single GENERATOR_STALE flag:
//
//  1. Does the running binary's embedded commit equal the
//     repository HEAD?  (commit-vs-HEAD freshness)
//
//  2. Is the running binary provably authoritative for the
//     complete source subject the digest represents?
//     (commit-vs-digest-subject authority)
//
// The first question is a freshness signal that has always been
// the (legacy) GENERATOR_STALE field's intent. The second question
// is what reviewers actually need. This ACT makes both signals
// independently visible; GENERATOR_STALE is preserved with its
// legacy semantics and explicitly labeled as the legacy
// commit-vs-repository-HEAD signal.
//
// The classifier is pure:
//
//   - it reads no filesystem;
//   - it spawns no Git subprocesses;
//   - it consults no clock;
//   - it never mutates its inputs.
//
// All required identities MUST be resolved by the caller before
// EvaluateGeneratorBinding is invoked. The classifier returns a
// typed GeneratorBinding record that downstream renderers
// translate into the visible lifecycle surface.
//
// The classifier is intentionally fail-closed: any ambiguity
// produces authoritative_for_digest=false. The vocabulary is
// stable and machine-readable.
package digest

// GeneratorBindingStatus is the typed verdict of the
// generator<->digest-subject binding classifier. The string
// values are part of the digest surface and must remain stable.
type GeneratorBindingStatus string

const (
	// GeneratorBindingAuthoritative: both the commit-binding and
	// the subject-binding predicates pass. The generator binary
	// is provably authoritative for the digest subject.
	GeneratorBindingAuthoritative GeneratorBindingStatus = "AUTHORITATIVE"

	// GeneratorBindingCommitMismatch: the generator's embedded
	// commit does not equal the repository HEAD. The generator
	// cannot be authoritative for any digest that resolves
	// against the current working tree.
	GeneratorBindingCommitMismatch GeneratorBindingStatus = "COMMIT_MISMATCH"

	// GeneratorBindingSubjectMismatch (CORRECTION01): the
	// generator's embedded commit does not equal the digest
	// subject commit. Distinct from COMMIT_MISMATCH:
	// COMMIT_MISMATCH records ambient-HEAD drift (the legacy
	// GENERATOR_STALE signal); SUBJECT_MISMATCH records actual
	// authority failure (the new GENERATOR_AUTHORITATIVE_FOR_DIGEST
	// signal). They may legitimately differ for historical-range
	// digests.
	GeneratorBindingSubjectMismatch GeneratorBindingStatus = "SUBJECT_MISMATCH"

	// GeneratorBindingDirtySubjectUnbound: the generator's
	// embedded commit equals the repository HEAD, but the digest
	// subject includes uncommitted source state (tracked-dirty,
	// staged, untracked, or mixed). The generator binary may
	// actually contain the change, but the digest cannot prove
	// it from commit identity alone. Authority is not proven.
	GeneratorBindingDirtySubjectUnbound GeneratorBindingStatus = "DIRTY_SUBJECT_UNBOUND"

	// GeneratorBindingIdentityUnbound: the generator's embedded
	// commit is not available (unset / unknown / empty). No
	// authority claim can be derived. Distinct from
	// COMMIT_MISMATCH: here there is nothing to compare.
	GeneratorBindingIdentityUnbound GeneratorBindingStatus = "IDENTITY_UNBOUND"

	// GeneratorBindingEvidenceInvalid: the generator identity is
	// available but malformed (not a valid Git OID). The
	// classifier never silently promotes unknown values.
	GeneratorBindingEvidenceInvalid GeneratorBindingStatus = "EVIDENCE_INVALID"
)

// GeneratorStateBindingStatus is the per-axis verdict. It mirrors
// the StateBindingStatus vocabulary established by
// gate_evidence_binding.go so the digest surface remains coherent
// across binding classifiers.
type GeneratorStateBindingStatus string

const (
	GeneratorStateMatch    GeneratorStateBindingStatus = "MATCH"
	GeneratorStateMismatch GeneratorStateBindingStatus = "MISMATCH"
	GeneratorStateUnbound  GeneratorStateBindingStatus = "UNBOUND"
)

// GeneratorIdentity records the resolved commit identity of the
// running generator binary, plus a classification of its validity.
// The classifier never spawns subprocesses; callers resolve and
// validate these fields before invoking EvaluateGeneratorBinding.
type GeneratorIdentity struct {
	// Commit is the binary's embedded VCS commit (full 40-char
	// OID), or "" when no commit was embedded.
	Commit string
	// CommitIsValid distinguishes "" / "unknown" (identity is
	// not available) from a non-empty value that is not a
	// syntactically valid OID (evidence is invalid).
	CommitIsValid bool
}

// RepositoryIdentity records the resolved commit identity of the
// repository HEAD. The classifier never spawns subprocesses;
// callers resolve these fields before invoking
// EvaluateGeneratorBinding.
type RepositoryIdentity struct {
	// HeadCommit is the full 40-char OID of the repository HEAD,
	// or "" when HEAD could not be resolved.
	HeadCommit string
	// HeadCommitIsValid distinguishes "" (not resolved) from a
	// non-empty value that is not a syntactically valid OID.
	HeadCommitIsValid bool
}

// DigestAuthoritySubject records the canonical subject the digest
// represents. For clean committed subjects, SubjectCommit equals
// repository HEAD. For explicit-range subjects, SubjectCommit is
// the resolved right endpoint of the range, which may differ from
// ambient HEAD. For dirty subjects, Dirty is true and the digest
// subject includes uncommitted source state not represented by
// any commit OID.
type DigestAuthoritySubject struct {
	// SubjectCommit is the full 40-char OID of the digest
	// subject, or "" when the subject could not be resolved.
	SubjectCommit string
	// SubjectCommitIsValid distinguishes "" (not resolved) from
	// a non-empty value that is not a syntactically valid OID.
	SubjectCommitIsValid bool
	// Dirty is true when the digest subject includes
	// uncommitted source state (tracked-dirty, staged, untracked,
	// or mixed). The classifier uses this flag to choose the
	// correct subject-binding verdict.
	Dirty bool
}

// GeneratorBinding is the typed output of the binding classifier.
// It is the single source of truth for whether the generator
// binary is authoritative for the digest subject.
//
// All fields are populated deterministically. Renderers MUST
// translate the typed fields into the documented lifecycle
// surface and MUST NOT silently override them.
type GeneratorBinding struct {
	// Status is the overall binding verdict. Distinct from
	// CommitMatchesHead, which is only the legacy
	// commit-vs-HEAD freshness signal.
	Status GeneratorBindingStatus

	// CommitBinding is the commit-binding component verdict.
	// MATCH iff the generator commit equals repository HEAD.
	// MISMATCH when the commits differ. UNBOUND when either
	// identity is not available.
	CommitBinding GeneratorStateBindingStatus

	// SubjectBinding is the subject-binding component verdict.
	// MATCH iff the generator commit equals the digest subject
	// AND the subject is not dirty. UNBOUND when the digest
	// subject is dirty (no commit-only proof can establish
	// authority for uncommitted state). MISMATCH when commits
	// differ. UNBOUND when either identity is not available.
	SubjectBinding GeneratorStateBindingStatus

	// CommitMatchesHead is the strict boolean that captures the
	// legacy GENERATOR_STALE semantics inverted. True when the
	// generator commit equals repository HEAD; false otherwise.
	// This is the digest surface for the legacy
	// GENERATOR_STALE flag and is preserved for compatibility.
	CommitMatchesHead bool

	// AuthoritativeForDigest is the strict boolean that callers
	// must consult. True ONLY when every required predicate
	// passed AND the digest subject is committed (not dirty)
	// AND every identity was available and valid.
	AuthoritativeForDigest bool

	// WarningCode is the stable machine-readable code
	// documenting why authority failed. "none" when the
	// binding is authoritative.
	WarningCode string
}

// Stable warning codes. These are part of the digest surface and
// must remain stable. Prefixed with Generator* to avoid collision
// with the gate-evidence and range-scope warning codes declared
// elsewhere in this package.
const (
	GeneratorWarningCodeNone                = "none"
	GeneratorWarningCodeCommitMismatch      = "GENERATOR_COMMIT_MISMATCH"
	GeneratorWarningCodeSubjectMismatch     = "GENERATOR_SUBJECT_MISMATCH"
	GeneratorWarningCodeDirtySubjectUnbound = "GENERATOR_DIRTY_SUBJECT_UNBOUND"
	GeneratorWarningCodeIdentityUnbound     = "GENERATOR_IDENTITY_UNBOUND"
	GeneratorWarningCodeEvidenceInvalid     = "GENERATOR_EVIDENCE_INVALID"
)
