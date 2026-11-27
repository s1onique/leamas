// Package domain provides the pure domain model for the Factory Work
// Ledger.
//
// This package has no filesystem, clock, network, database or process
// dependencies. It defines the typed records, the identity rules, the
// semantic validation rules and the deterministic projections that the
// ledger storage layer and the CLI render on top of.
//
// The separation is deliberate: every load-bearing semantic rule lives
// here exactly once so the CLI, the importer and the tests all agree.
package domain

// LedgerSchemaVersion is the version of the canonical record wire format.
const LedgerSchemaVersion = 1

// EpicID is a stable scoped epic identity.
type EpicID string

// ItemID is a stable scoped work-item identity.
type ItemID string

// ActID is a stable ACT execution record identity.
type ActID string

// EvidenceID is a stable evidence reference identity.
type EvidenceID string

// DecisionID is a stable decision identity.
type DecisionID string

// KnowledgeID is a stable reusable knowledge reference identity.
type KnowledgeID string

// ScopedItemID is the scoped identity of a work item: "<EPIC-ID>/<ITEM-ID>".
type ScopedItemID string

// Scope returns the epic scope of a scoped item identity.
func (s ScopedItemID) Scope() EpicID {
	for i := 0; i < len(s); i++ {
		if s[i] == '/' {
			return EpicID(s[:i])
		}
	}
	return ""
}

// Local returns the item-local part of a scoped item identity.
func (s ScopedItemID) Local() ItemID {
	for i := 0; i < len(s); i++ {
		if s[i] == '/' {
			return ItemID(s[i+1:])
		}
	}
	return ItemID(s)
}

// String renders the scoped identity.
func (s ScopedItemID) String() string { return string(s) }

// LifecycleState enumerates the workflow position of a record.
type LifecycleState string

// Lifecycle states.
const (
	LifecycleOpen       LifecycleState = "open"
	LifecycleInProgress LifecycleState = "in_progress"
	LifecycleBlocked    LifecycleState = "blocked"
	LifecycleClosed     LifecycleState = "closed"
	LifecycleArchived   LifecycleState = "archived"
)

// IsValidLifecycleState reports whether state is a known lifecycle state.
func IsValidLifecycleState(state LifecycleState) bool {
	switch state {
	case LifecycleOpen, LifecycleInProgress, LifecycleBlocked,
		LifecycleClosed, LifecycleArchived:
		return true
	}
	return false
}

// AcceptanceState enumerates whether the engineering obligations of a
// work item are satisfied. It is deliberately separate from lifecycle.
type AcceptanceState string

// Acceptance states.
const (
	AcceptancePending   AcceptanceState = "pending"
	AcceptancePartial   AcceptanceState = "partial"
	AcceptanceAccepted  AcceptanceState = "accepted"
	AcceptanceRejected  AcceptanceState = "rejected"
	AcceptanceDeferred  AcceptanceState = "deferred"
	AcceptanceWithdrawn AcceptanceState = "withdrawn"
)

// IsValidAcceptanceState reports whether state is a known acceptance state.
func IsValidAcceptanceState(state AcceptanceState) bool {
	switch state {
	case AcceptancePending, AcceptancePartial, AcceptanceAccepted,
		AcceptanceRejected, AcceptanceDeferred, AcceptanceWithdrawn:
		return true
	}
	return false
}

// IsTerminalAcceptance reports whether the acceptance state is settled.
func IsTerminalAcceptance(state AcceptanceState) bool {
	switch state {
	case AcceptanceAccepted, AcceptanceRejected, AcceptanceWithdrawn:
		return true
	}
	return false
}

// Priority enumerates ordering intent.
type Priority string

// Priorities.
const (
	PriorityP0 Priority = "P0"
	PriorityP1 Priority = "P1"
	PriorityP2 Priority = "P2"
	PriorityP3 Priority = "P3"
)

// IsValidPriority reports whether p is a known priority.
func IsValidPriority(p Priority) bool {
	switch p {
	case PriorityP0, PriorityP1, PriorityP2, PriorityP3:
		return true
	}
	return false
}

// rank returns the deterministic sort rank of a priority (lower first).
func (p Priority) rank() int {
	switch p {
	case PriorityP0:
		return 0
	case PriorityP1:
		return 1
	case PriorityP2:
		return 2
	case PriorityP3:
		return 3
	}
	return 4
}

// EvidenceQuality enumerates how strong the attached evidence is.
type EvidenceQuality string

// Evidence qualities.
const (
	EvidenceQualityNone       EvidenceQuality = "none"
	EvidenceQualityDeclared   EvidenceQuality = "declared"
	EvidenceQualityReferenced EvidenceQuality = "referenced"
	EvidenceQualityVerified   EvidenceQuality = "verified"
)

// IsValidEvidenceQuality reports whether q is a known evidence quality.
func IsValidEvidenceQuality(q EvidenceQuality) bool {
	switch q {
	case EvidenceQualityNone, EvidenceQualityDeclared,
		EvidenceQualityReferenced, EvidenceQualityVerified:
		return true
	}
	return false
}

// EvidenceClass enumerates WHAT KIND of claim an artifact supports.
//
// Evidence quality answers "how strongly is this artifact bound?". The
// evidence class answers "what kind of claim may this artifact support?".
// The two are independent: a perfectly verified structural artifact is
// still not a live qualification.
type EvidenceClass string

// Evidence classes.
const (
	// EvidenceClassStructural is a structural or synthetic artifact. It can
	// support a structural criterion but never a live qualification.
	EvidenceClassStructural EvidenceClass = "structural"
	// EvidenceClassLiveQualification is the output of an actual live
	// qualification run. Only this class can satisfy a live qualification
	// criterion.
	EvidenceClassLiveQualification EvidenceClass = "live_qualification"
	// EvidenceClassReview is a human or agent review record.
	EvidenceClassReview EvidenceClass = "review"
)

// IsValidEvidenceClass reports whether c is a known evidence class.
func IsValidEvidenceClass(c EvidenceClass) bool {
	switch c {
	case EvidenceClassStructural, EvidenceClassLiveQualification,
		EvidenceClassReview:
		return true
	}
	return false
}

// SatisfiesLiveQualification reports whether the class may establish a
// live qualification.
//
// This is the mechanical form of the "structural evidence is not live
// qualification" rule. Mechanical artifact verification is NOT autonomous
// verification of an engineering claim: only an explicitly classified live
// qualification artifact can support a live qualification criterion.
func (c EvidenceClass) SatisfiesLiveQualification() bool {
	return c == EvidenceClassLiveQualification
}

// ActVerdict enumerates the terminal verdict of an ACT execution record.
type ActVerdict string

// ACT verdicts.
const (
	ActVerdictPending     ActVerdict = "pending"
	ActVerdictClosed      ActVerdict = "closed"
	ActVerdictClosedInsuf ActVerdict = "closed_insufficient"
	ActVerdictFailed      ActVerdict = "failed"
	ActVerdictAbandoned   ActVerdict = "abandoned"
)

// IsValidActVerdict reports whether v is a known ACT verdict.
func IsValidActVerdict(v ActVerdict) bool {
	switch v {
	case ActVerdictPending, ActVerdictClosed, ActVerdictClosedInsuf,
		ActVerdictFailed, ActVerdictAbandoned:
		return true
	}
	return false
}

// IsTerminalActVerdict reports whether v is a terminal ACT verdict.
func IsTerminalActVerdict(v ActVerdict) bool {
	switch v {
	case ActVerdictClosed, ActVerdictClosedInsuf, ActVerdictFailed,
		ActVerdictAbandoned:
		return true
	}
	return false
}

// DependencyKind enumerates the typed dependency relations.
type DependencyKind string

// Dependency kinds.
const (
	DependencyBlocks   DependencyKind = "blocks"
	DependencyRequires DependencyKind = "requires"
	DependencyRelates  DependencyKind = "relates"
	// DependencyBlocker is a durable unresolved blocker. It points at an
	// identity that does not exist yet. It is allowed to dangle precisely
	// because the dangling reference is what keeps the dependent item
	// non-executable.
	DependencyBlocker DependencyKind = "blocker"
)

// IsValidDependencyKind reports whether k is a known dependency kind.
func IsValidDependencyKind(k DependencyKind) bool {
	switch k {
	case DependencyBlocks, DependencyRequires, DependencyRelates,
		DependencyBlocker:
		return true
	}
	return false
}

// IsBlocking reports whether the dependency kind blocks progress.
func (k DependencyKind) IsBlocking() bool {
	return k == DependencyBlocks || k == DependencyRequires || k == DependencyBlocker
}

// IsResolvable reports whether the dependency reference must resolve to an
// existing work item.
//
// A durable blocker is the explicit exception: an unresolved blocker is a
// conserved obligation, not a dangling mistake.
func (k DependencyKind) IsResolvable() bool {
	return k != DependencyBlocker
}

// Dependency is a typed reference from one work item to another.
type Dependency struct {
	Kind DependencyKind `json:"kind"`
	Ref  ScopedItemID   `json:"ref"`
	Note string         `json:"note,omitempty"`

	// RequiredOutcome states WHAT the relationship requires, when the
	// source declared it.
	//
	// It is conserved verbatim: the outcome a relationship demands is
	// load-bearing content of the edge, not a comment.
	RequiredOutcome string `json:"required_outcome,omitempty"`

	// SourceRefs are the byte-exact source locations the relationship was
	// derived from.
	SourceRefs []SourceRef `json:"source_refs,omitempty"`
}

// ProvenanceConfidence enumerates how authoritative a source record is.
type ProvenanceConfidence string

// Provenance confidences.
const (
	// ProvenanceRepository means the source is repository-verified state.
	ProvenanceRepository ProvenanceConfidence = "repository_verified"
	// ProvenanceConversational means the source is a conversational claim
	// that has NOT been promoted to a repository-verified fact.
	ProvenanceConversational ProvenanceConfidence = "conversational"
	// ProvenanceImported means the source came through the importer.
	ProvenanceImported ProvenanceConfidence = "imported"
)

// IsValidProvenanceConfidence reports whether c is a known confidence.
func IsValidProvenanceConfidence(c ProvenanceConfidence) bool {
	switch c {
	case ProvenanceRepository, ProvenanceConversational, ProvenanceImported:
		return true
	}
	return false
}

// Provenance records where a record came from.
type Provenance struct {
	Source     string `json:"source"`
	SourceID   string `json:"source_id,omitempty"`
	ImportedAt string `json:"imported_at,omitempty"`
	Confidence string `json:"confidence,omitempty"`
}

// EvidenceRef binds a work item to an evidence reference.
type EvidenceRef struct {
	EvidenceID EvidenceID      `json:"evidence_id"`
	Quality    EvidenceQuality `json:"quality"`
	Note       string          `json:"note,omitempty"`
	// Class is the explicit evidence class of the referenced artifact. It
	// determines which kind of claim the artifact may support.
	Class EvidenceClass `json:"class,omitempty"`
	// Criterion is the acceptance criterion this evidence supports. It is
	// the explicit criterion/scope association: evidence that names no
	// criterion supports no criterion.
	Criterion string `json:"criterion,omitempty"`
}

// ReopenCondition records an applicable condition that permits reopening
// an archived record.
type ReopenCondition struct {
	Kind   string `json:"kind"`
	Detail string `json:"detail,omitempty"`
}

// Epic is a scoped body of work.
type Epic struct {
	SchemaVersion int         `json:"schema_version"`
	EpicID        EpicID      `json:"epic_id"`
	Title         string      `json:"title"`
	Summary       string      `json:"summary,omitempty"`
	Status        string      `json:"status,omitempty"`
	Priority      Priority    `json:"priority,omitempty"`
	Provenance    *Provenance `json:"provenance,omitempty"`
	Archived      bool        `json:"archived,omitempty"`
	Revision      int         `json:"revision"`

	// ExitCriteria are the epic-scoped exit criteria of the scope.
	//
	// A historical board may declare exit criteria that belong to the
	// scope as a whole rather than to a single work item. They are
	// conserved verbatim: their exact content survives normalization,
	// planning, persistence and an independent reload.
	ExitCriteria []string `json:"exit_criteria,omitempty"`

	// SourceRefs are the byte-exact source locations the epic record was
	// derived from.
	SourceRefs []SourceRef `json:"source_refs,omitempty"`

	// SourceLifecycle is the NORMALIZED lifecycle the source asserted for
	// the epic. It is conserved distinctly from the canonical state.
	SourceLifecycle string `json:"source_lifecycle,omitempty"`

	// SourceFingerprint is the immutable normalized fingerprint of the
	// SOURCE epic record.
	//
	// It covers the declared metadata, the exit criteria and the source
	// provenance of the scope. It is what lets an import distinguish an
	// exact historical replay from a changed historical epic source
	// without comparing the canonical record: a legitimate canonical
	// evolution of the scope does not change the fingerprint, a changed
	// source does.
	SourceFingerprint string `json:"source_fingerprint,omitempty"`
}

// ItemKind enumerates the kinds of work item.
type ItemKind string

// Item kinds.
const (
	ItemKindTask   ItemKind = "task"
	ItemKindGate   ItemKind = "gate"
	ItemKindSpike  ItemKind = "spike"
	ItemKindDefect ItemKind = "defect"
)

// IsValidItemKind reports whether k is a known item kind.
func IsValidItemKind(k ItemKind) bool {
	switch k {
	case ItemKindTask, ItemKindGate, ItemKindSpike, ItemKindDefect:
		return true
	}
	return false
}
