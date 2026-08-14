// SPDX-License-Identifier: Apache-2.0

// Package digest: gate_evidence_binding.go declares the typed
// surface of the gate-evidence binding classifier. The pure
// classifier itself lives in gate_evidence_classifier.go.
//
// Doctrine:
//
//	PRESENCE != AUTHORITY.
//
// A file merely existing inside the working tree (or in a
// historical artifact) does not mean its verdict describes the
// state captured by the digest. Authority is established only
// when the gate summary's execution identity matches the digest's
// resolved subject identity.
//
// The classifier is pure:
//
//   - it reads no filesystem;
//   - it spawns no Git subprocesses;
//   - it consults no clock;
//   - it never mutates its inputs.
//
// All required identities MUST be resolved by the caller before
// EvaluateGateEvidenceBinding is invoked. The classifier returns
// a typed GateEvidenceBinding record that downstream renderers
// translate into the visible GATE_SUMMARY section.
//
// The classifier is intentionally fail-closed: any ambiguity
// produces authoritative_for_digest=false. The vocabulary is
// stable and machine-readable.
package digest

// GateBindingStatus is the typed verdict of the binding
// classifier. The string values are part of the digest surface
// and must remain stable.
type GateBindingStatus string

const (
	// BindingAuthoritative: state and scope predicates both
	// pass. The gate summary is the authoritative verdict
	// for the digest.
	BindingAuthoritative GateBindingStatus = "AUTHORITATIVE"

	// BindingStateMatchScopeUnbound: repository state matches
	// but the digest's ACT/scope identity cannot be verified
	// from the gate evidence. The summary is not authoritative
	// for the digest even though it is not yet stale.
	BindingStateMatchScopeUnbound GateBindingStatus = "STATE_MATCH_SCOPE_UNBOUND"

	// BindingStateMismatch: gate execution identity does not
	// match the digest subject identity.
	BindingStateMismatch GateBindingStatus = "STATE_MISMATCH"

	// BindingScopeMismatch: state matches but the scope/ACT
	// identities explicitly disagree.
	BindingScopeMismatch GateBindingStatus = "SCOPE_MISMATCH"

	// BindingDirtySubjectUnbound: digest represents a dirty
	// (uncommitted) subject but the gate evidence binds only
	// to a commit/tree identity. A commit-only proof cannot
	// identify uncommitted working-tree content.
	BindingDirtySubjectUnbound GateBindingStatus = "DIRTY_SUBJECT_UNBOUND"

	// BindingLegacyUnbound: the gate summary carries no
	// execution identity (v1 wire shape). Authority cannot be
	// established; the verdict is preserved as historical
	// evidence only.
	BindingLegacyUnbound GateBindingStatus = "LEGACY_UNBOUND"

	// BindingEvidenceInvalid: the summary is unparseable or
	// internally inconsistent. Authority cannot be derived.
	BindingEvidenceInvalid GateBindingStatus = "EVIDENCE_INVALID"

	// BindingNotApplicable: no gate summary was discovered.
	// Distinct from LEGACY_UNBOUND: absence is not a stale
	// unbound artifact.
	BindingNotApplicable GateBindingStatus = "NOT_APPLICABLE"
)

// StateBindingStatus is the state component of the binding
// verdict. Comparison of the gate summary's execution identity
// with the digest's resolved subject identity.
type StateBindingStatus string

const (
	StateMatch    StateBindingStatus = "MATCH"
	StateMismatch StateBindingStatus = "MISMATCH"
	StateUnbound  StateBindingStatus = "UNBOUND"
)

// ScopeBindingStatus is the scope component of the binding
// verdict. Comparison of the gate summary's scope/ACT identity
// with the digest's resolved ACT identity.
type ScopeBindingStatus string

const (
	ScopeMatch         ScopeBindingStatus = "MATCH"
	ScopeMismatch      ScopeBindingStatus = "MISMATCH"
	ScopeUnbound       ScopeBindingStatus = "UNBOUND"
	ScopeNotApplicable ScopeBindingStatus = "NOT_APPLICABLE"
)

// GateSummaryIdentity is the binding-relevant extract of a
// normalized Gate Summary. The renderer and the digest pipeline
// populate this struct from the existing decoded Summary; the
// binding classifier never inspects the wire format directly.
//
// Empty strings mean the corresponding field is absent on the
// gate summary (for example, the v1 wire shape carries no
// execution identity). The classifier treats empty strings as
// a missing field.
type GateSummaryIdentity struct {
	// SchemaVersion is the numeric schema version of the
	// source gate summary. 0 means the summary was not
	// readable.
	SchemaVersion int

	// GeneratedAt is preserved for diagnostic rendering.
	// It MUST NOT establish authority.
	GeneratedAt string

	// OverallStatus is the literal reported verdict (pass,
	// fail, skip, unavailable). It is rendered unchanged as
	// reported_overall_status when the binding is not
	// AUTHORITATIVE.
	OverallStatus string

	// ExecutionHeadOID is the commit OID the gate ran
	// against. Empty means the field is absent on the source.
	ExecutionHeadOID string

	// ExecutionTreeOID is the working-tree OID the gate ran
	// against. Optional but preferred.
	ExecutionTreeOID string

	// SubjectTreeOID is the post-run tree the gate captured.
	// Optional but preferred.
	SubjectTreeOID string

	// ScopeID is the bounded verification scope (ACT-ID).
	ScopeID string

	// ParentAct is the parent ACT identifier.
	ParentAct string

	// WorktreeCleanBefore reports whether the producer
	// observed a clean working tree before running.
	WorktreeCleanBefore    bool
	WorktreeCleanBeforeSet bool

	// WorktreeCleanAfter reports whether the producer
	// observed a clean working tree after running.
	WorktreeCleanAfter    bool
	WorktreeCleanAfterSet bool

	// HasExecutionIdentity reports whether the gate summary
	// exposed any execution identity at all. v1 returns
	// false; v2/v3 return true when execution_head_oid is
	// populated.
	HasExecutionIdentity bool

	// HasScopeIdentity reports whether the gate summary
	// exposed any scope identity (scope_id or parent_act).
	HasScopeIdentity bool
}

// DigestAuthority is the binding-relevant extract of the
// digest's already-resolved subject. The digest pipeline
// populates this from the existing ResolvedMode and tooling
// identity; the classifier never invokes Git or reads
// filesystem state.
type DigestAuthority struct {
	// Mode is the resolved digest mode. "dirty" implies
	// the working tree has uncommitted changes.
	Mode string

	// Range is the resolved range expression (when applicable).
	Range string

	// SubjectCommitOID is the commit OID that the digest
	// represents as its subject. For clean committed ranges
	// this is the resolved right endpoint. For dirty mode
	// it is empty (uncommitted content has no commit OID).
	SubjectCommitOID string

	// SubjectTreeOID is the tree OID that the digest
	// represents. For dirty mode it is empty.
	SubjectTreeOID string

	// ActID is the resolved ACT identifier. Empty when
	// no lifecycle artifact identifies an ACT at the
	// digest subject.
	ActID string

	// Dirty is true when the digest represents a dirty
	// working tree (uncommitted changes).
	Dirty bool
}

// GateEvidenceBinding is the typed output of the binding
// classifier. It is the single source of truth for whether
// the discovered gate summary is authoritative for the
// digest.
//
// All fields are populated deterministically. Renderers
// MUST translate the typed fields into the documented
// GATE_SUMMARY surface and MUST NOT silently override them.
type GateEvidenceBinding struct {
	// Status is the binding verdict. Distinct from
	// OverallStatus, which is the raw gate verdict.
	Status GateBindingStatus

	// SourceValidity is the third independent dimension
	// that distinguishes ABSENT from INVALID. Callers
	// MUST surface this in the rendered digest so the
	// reader can distinguish "no evidence present" from
	// "evidence present but malformed".
	SourceValidity GateSourceValidity

	// StateBinding is the state component.
	StateBinding StateBindingStatus

	// ScopeBinding is the scope component.
	ScopeBinding ScopeBindingStatus

	// AuthoritativeForDigest is the strict boolean that
	// callers must consult. True ONLY when every required
	// predicate passed.
	AuthoritativeForDigest bool

	// WarningCode is the stable machine-readable code
	// documenting why authority failed. "none" when the
	// summary is authoritative.
	WarningCode string

	// TreeBinding records the additional tree OID
	// comparison when both the gate summary and the digest
	// expose a tree OID. The classifier never promotes
	// same-tree evidence to authority; the field is
	// purely informational.
	TreeBinding StateBindingStatus
}

// Stable warning codes. These are part of the digest surface
// and must remain stable. Prefixed with GateBinding* to avoid
// collision with the WarningCodeNone / RangeScope warning codes
// declared in range_scope.go (same package).
const (
	GateBindingWarningCodeNone                   = "none"
	GateBindingWarningCodeLegacyUnbound          = "GATE_SUMMARY_LEGACY_UNBOUND"
	GateBindingWarningCodeStateMismatch          = "GATE_SUMMARY_STATE_MISMATCH"
	GateBindingWarningCodeScopeMismatch          = "GATE_SUMMARY_SCOPE_MISMATCH"
	GateBindingWarningCodeDirtySubjectUnbound    = "GATE_SUMMARY_DIRTY_SUBJECT_UNBOUND"
	GateBindingWarningCodeInvalidBinding         = "GATE_SUMMARY_INVALID_BINDING"
	GateBindingWarningCodeStateMatchScopeUnbound = "GATE_SUMMARY_STATE_MATCH_SCOPE_UNBOUND"
	GateBindingWarningCodeNotApplicable          = "GATE_SUMMARY_NOT_APPLICABLE"
)

// GateSourceValidity is the third independent dimension of
// the binding classifier. It distinguishes absence from
// invalidity, which the ACT requires to remain semantically
// distinct.
type GateSourceValidity int

const (
	// SourceAbsent: no gate summary file was discovered.
	// The classifier returns NOT_APPLICABLE.
	SourceAbsent GateSourceValidity = iota

	// SourceInvalid: a gate summary file was discovered but
	// is unparseable, malformed, or fails structural
	// normalization. The classifier returns EVIDENCE_INVALID.
	SourceInvalid

	// SourceValid: a gate summary file was discovered and
	// parsed successfully. The classifier proceeds to the
	// binding evaluation.
	SourceValid
)

// String renders the source validity as a stable string.
func (s GateSourceValidity) String() string {
	switch s {
	case SourceAbsent:
		return "ABSENT"
	case SourceInvalid:
		return "INVALID"
	case SourceValid:
		return "VALID"
	}
	return "UNKNOWN"
}
