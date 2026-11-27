package domain

import "sort"

// SourceRef is a byte-exact reference to a source location.
//
// It binds a claim to an exact byte range of an exact source revision. The
// line range is one-based and inclusive. The digest covers the exact
// original bytes of that range, including the newline bytes.
type SourceRef struct {
	Path      string `json:"path"`
	StartLine int    `json:"start_line"`
	EndLine   int    `json:"end_line"`
	SHA256    string `json:"sha256"`
	// SourceHead binds the reference to a source revision when it differs
	// from the parent source manifest head.
	SourceHead string `json:"source_head,omitempty"`
}

// WorkItem is a typed engineering obligation.
type WorkItem struct {
	SchemaVersion int      `json:"schema_version"`
	EpicID        EpicID   `json:"epic_id"`
	ItemID        ItemID   `json:"item_id"`
	Title         string   `json:"title"`
	Summary       string   `json:"summary,omitempty"`
	Kind          ItemKind `json:"kind"`

	Lifecycle  LifecycleState  `json:"lifecycle"`
	Acceptance AcceptanceState `json:"acceptance"`
	Priority   Priority        `json:"priority,omitempty"`
	Quality    EvidenceQuality `json:"evidence_quality,omitempty"`

	Owner    string `json:"owner,omitempty"`
	Revision int    `json:"revision"`
	Archived bool   `json:"archived,omitempty"`

	// AcceptanceCriteria lists the concrete obligations that must hold
	// for the item to be accepted.
	AcceptanceCriteria []string `json:"acceptance_criteria,omitempty"`

	// RequiresLiveGate marks an item whose acceptance needs a live
	// qualification run. A structural pass never satisfies it.
	RequiresLiveGate bool `json:"requires_live_gate,omitempty"`

	// ExitCriteria are the epic-scoped exit criteria owned by this item.
	ExitCriteria []string `json:"exit_criteria,omitempty"`

	// Workstream preserves the EXACT source workstream identity of the
	// imported record.
	//
	// A historical board may group its work items under a parent epic and
	// a narrower workstream. The canonical epic scope keeps the parent
	// identity, so the workstream identity would be lost without this
	// explicit field. It is conserved verbatim: it is never derived from
	// the item identity.
	Workstream string `json:"workstream,omitempty"`

	// SourceFingerprint is the immutable normalized fingerprint of the
	// SOURCE record the work item was imported from.
	//
	// It covers the normalized source content, INCLUDING the normalized
	// source relationship graph. It is never rewritten by a legitimate
	// canonical evolution: the fingerprint describes the source, not the
	// canonical record.
	//
	// It is what lets an import distinguish an exact historical replay
	// from a changed historical source without comparing the canonical
	// record: a canonical evolution does not change the fingerprint, a
	// changed source does.
	SourceFingerprint string `json:"source_fingerprint,omitempty"`

	Dependencies []Dependency      `json:"dependencies,omitempty"`
	Evidence     []EvidenceRef     `json:"evidence,omitempty"`
	Successors   []ScopedItemID    `json:"successors,omitempty"`
	Provenance   *Provenance       `json:"provenance,omitempty"`
	Reopen       []ReopenCondition `json:"reopen_conditions,omitempty"`

	// SourceBlockers are the EXPLICIT blockers the source declared for the
	// record.
	//
	// They are conserved verbatim: a blocker the source stated is a
	// load-bearing fact about the obligation. It is NOT a dependency edge:
	// an explicit blocker prevents readiness even when every dependency
	// resolves.
	SourceBlockers []string `json:"source_blockers,omitempty"`

	// SourceCurrentACT is the ACT identity the source associated with the
	// record as the CURRENT execution.
	//
	// It is conserved as a historical association. It never asserts that
	// the ACT is still running and never authorizes execution.
	SourceCurrentACT string `json:"source_current_act,omitempty"`

	// SourceRelatedACTs are the historical ACT identities the source
	// associated with the record.
	//
	// They are conserved verbatim: a historical association stays
	// discoverable without ever re-entering the live execution history.
	SourceRelatedACTs []string `json:"source_related_acts,omitempty"`

	// SourceEvidenceLocators are the source locators of the historical
	// evidence of the record.
	//
	// They are LOCATORS, not registered evidence records: they never
	// satisfy an acceptance criterion.
	SourceEvidenceLocators []string `json:"source_evidence_locators,omitempty"`

	// TerminalDisposition preserves the original terminal disposition
	// when the record is archived.
	TerminalDisposition string `json:"terminal_disposition,omitempty"`

	// SourceStatus preserves the EXACT source status the record was
	// imported from. It is conserved verbatim: an unknown source status
	// stays unknown instead of being coerced into a known lifecycle
	// state.
	SourceStatus string `json:"source_status,omitempty"`

	// SourceRow preserves the original row provenance of the imported
	// record: the identity it carried in the source and the row it came
	// from.
	SourceRow string `json:"source_row,omitempty"`

	// SourceLine preserves the original line locator of the imported
	// record. It is zero when the source carried no locator.
	SourceLine int `json:"source_line,omitempty"`

	// SourceDetail preserves the original free-form source detail of the
	// imported record.
	SourceDetail string `json:"source_detail,omitempty"`

	// SourceRefs are the byte-exact source locations the imported record
	// was derived from. They preserve the provenance of the interpretation.
	SourceRefs []SourceRef `json:"source_refs,omitempty"`

	// SourceQualification preserves the QUALIFICATION ASSERTION the source
	// made about the obligation.
	//
	// It is a historical assertion, never an acceptance decision: the
	// canonical acceptance state is established by the acceptance authority
	// alone. A source assertion of ACCEPTED never sets the acceptance state.
	SourceQualification string `json:"source_qualification,omitempty"`

	// SourceLifecycle preserves the NORMALIZED lifecycle the source
	// asserted for the record.
	//
	// It is distinct from the canonical lifecycle state: an UNRESOLVED
	// source lifecycle stays distinguishable from an intentional DEFERRED
	// one after the import.
	SourceLifecycle string `json:"source_lifecycle,omitempty"`

	// CriteriaCoverage records whether the acceptance criteria of the
	// source were recovered completely. It distinguishes "there are none"
	// from "they were not recovered".
	CriteriaCoverage string `json:"criteria_coverage,omitempty"`

	// DependencyCoverage records whether the dependencies of the source
	// were recovered completely.
	DependencyCoverage string `json:"dependency_coverage,omitempty"`

	// AcceptedBy records the explicit engineering acceptance decision: who
	// accepted the item and on what basis. It is the scoped, explicit act
	// that the mechanical artifact verification alone can never perform.
	//
	// It is EMPTY for a non-accepted item. A mechanically verified
	// artifact never fills it: only the explicit acceptance operation does.
	AcceptedBy string `json:"accepted_by,omitempty"`

	// AcceptanceBasis records the engineering justification of the
	// acceptance decision. It is never authority on its own.
	AcceptanceBasis string `json:"acceptance_basis,omitempty"`

	// AcceptanceSourceHead binds the acceptance decision to the
	// repository/source revision it was made against. It is empty only
	// when the acceptance is not bound to a source revision.
	//
	// It is the source-identity binding of the CLAIM: an acceptance that
	// requires source binding must name the source revision it was made
	// against, so a later source change is detectable.
	AcceptanceSourceHead string `json:"acceptance_source_head,omitempty"`
}

// Scoped returns the scoped identity of the work item.
func (w WorkItem) Scoped() ScopedItemID {
	return ScopedItemID(string(w.EpicID) + "/" + string(w.ItemID))
}

// ActExecution is an ACT execution record bound to a work item.
type ActExecution struct {
	SchemaVersion int          `json:"schema_version"`
	ActID         ActID        `json:"act_id"`
	ItemID        ScopedItemID `json:"item_id,omitempty"`
	// ItemIDs are all work items the ACT addresses. An ACT may address
	// several work items; each keeps its own acceptance state and its own
	// retained obligation.
	ItemIDs []ScopedItemID `json:"item_ids,omitempty"`

	// UnresolvedItemIDs are the work-item associations of the ACT that did
	// NOT resolve to a known work item.
	//
	// They are conserved EXPLICITLY: an association the historical index
	// carried is never silently dropped because its target is unknown.
	// They are not work-item obligations: they are the preserved,
	// unresolved association.
	UnresolvedItemIDs []string   `json:"unresolved_item_ids,omitempty"`
	Title             string     `json:"title,omitempty"`
	Verdict           ActVerdict `json:"verdict"`
	Revision          int        `json:"revision"`
	Archived          bool       `json:"archived,omitempty"`

	// TerminalDisposition preserves the original terminal disposition.
	TerminalDisposition string `json:"terminal_disposition,omitempty"`

	// RetainedObligation records that the terminal verdict did NOT retire
	// the linked engineering obligation. It is the explicit, machine
	// readable form of "a terminal ACT must not automatically close its
	// associated engineering obligations".
	RetainedObligation string `json:"retained_obligation,omitempty"`

	// Evidence references bound to this ACT execution.
	Evidence []EvidenceRef `json:"evidence,omitempty"`

	// Note is a free-form operator note. It is never authority.
	Note string `json:"note,omitempty"`

	// OriginalVerdict preserves the ORIGINAL verdict of the historical
	// execution attempt.
	//
	// It is conserved verbatim. It is never an acceptance decision about
	// the engineering obligation the ACT touched.
	OriginalVerdict string `json:"original_verdict,omitempty"`

	// EvidenceLocators are the source locators of the historical evidence
	// of the execution attempt. They are locators, not registered evidence
	// records: they never satisfy an acceptance criterion.
	EvidenceLocators []string `json:"evidence_locators,omitempty"`

	// SourceRefs are the byte-exact source locations the historical record
	// was indexed from.
	SourceRefs []SourceRef `json:"source_refs,omitempty"`

	// Reconciliation records how confidently the historical identity was
	// reconciled with the source.
	Reconciliation string `json:"reconciliation,omitempty"`

	Provenance *Provenance `json:"provenance,omitempty"`
}

// IsTerminal reports whether the ACT execution reached a terminal verdict.
func (a ActExecution) IsTerminal() bool {
	return IsTerminalActVerdict(a.Verdict)
}

// EvidenceRefRecord is a repository-owned evidence reference.
type EvidenceRefRecord struct {
	SchemaVersion int             `json:"schema_version"`
	EvidenceID    EvidenceID      `json:"evidence_id"`
	Kind          string          `json:"kind"`
	Title         string          `json:"title"`
	RelativePath  string          `json:"relative_path,omitempty"`
	SHA256        string          `json:"sha256,omitempty"`
	ItemID        ScopedItemID    `json:"item_id,omitempty"`
	Quality       EvidenceQuality `json:"quality,omitempty"`
	Revision      int             `json:"revision"`
	Archived      bool            `json:"archived,omitempty"`
	Provenance    *Provenance     `json:"provenance,omitempty"`

	// Class is the explicit evidence class. It separates artifact
	// integrity from claim binding: a verified artifact of the structural
	// class can never establish a live qualification.
	Class EvidenceClass `json:"class,omitempty"`

	// Criterion is the acceptance criterion this evidence supports. It is
	// the explicit criterion/scope association.
	Criterion string `json:"criterion,omitempty"`

	// ByteSize is the verified size of the artifact in bytes. It is
	// recorded by the verification pass, never by the caller.
	ByteSize int64 `json:"byte_size,omitempty"`

	// SourceHead binds the evidence to the repository/source HEAD it was
	// produced against. It is empty only when the evidence is not bound to
	// a source revision.
	SourceHead string `json:"source_head,omitempty"`

	// Verified records that the artifact identity was actually checked
	// against the repository: the file exists at the declared relative
	// path, its byte size matches and its SHA-256 matches.
	Verified bool `json:"verified,omitempty"`

	// VerifiedAt is the verification timestamp. It is never authority.
	VerifiedAt string `json:"verified_at,omitempty"`
}

// Decision is a durable recorded decision.
type Decision struct {
	SchemaVersion int          `json:"schema_version"`
	DecisionID    DecisionID   `json:"decision_id"`
	Title         string       `json:"title"`
	Rationale     string       `json:"rationale,omitempty"`
	ItemID        ScopedItemID `json:"item_id,omitempty"`
	Supersedes    []DecisionID `json:"supersedes,omitempty"`
	Revision      int          `json:"revision"`
	Archived      bool         `json:"archived,omitempty"`
	Provenance    *Provenance  `json:"provenance,omitempty"`
}

// Knowledge is a reusable knowledge reference.
type Knowledge struct {
	SchemaVersion int          `json:"schema_version"`
	KnowledgeID   KnowledgeID  `json:"knowledge_id"`
	Title         string       `json:"title"`
	Summary       string       `json:"summary,omitempty"`
	RelativePath  string       `json:"relative_path,omitempty"`
	ItemID        ScopedItemID `json:"item_id,omitempty"`
	Revision      int          `json:"revision"`
	Archived      bool         `json:"archived,omitempty"`
	Provenance    *Provenance  `json:"provenance,omitempty"`
}

// MigrationRecord is a durable migration reconciliation record.
//
// It is the PERSISTED form of an unresolved reconciliation: a source
// record that could not be accounted for during the migration.
//
// A reconciliation record is durable MIGRATION STATE, never a console
// warning. An operator that finds it later can see exactly which source
// record could not be reconciled and why.
type MigrationRecord struct {
	SchemaVersion int    `json:"schema_version"`
	MigrationID   string `json:"migration_id"`
	// SourceRepository is the source repository the migration read.
	SourceRepository string `json:"source_repository,omitempty"`
	// SourceHead is the source HEAD the migration was taken at.
	SourceHead string `json:"source_head,omitempty"`
	// SourcePath is the repository-relative path of the source record.
	SourcePath string `json:"source_path,omitempty"`
	// SourceLine is the source line of the record.
	SourceLine int `json:"source_line,omitempty"`
	// SourceDigest is the content digest of the source record.
	SourceDigest string `json:"source_digest,omitempty"`
	// SourceRefs are ALL the byte-exact source locations the record was
	// derived from.
	//
	// The single source path, line and digest above are the FIRST
	// reference. The complete set is conserved here: a reconciliation
	// record that cites several locations loses none of them.
	SourceRefs []SourceRef `json:"source_refs,omitempty"`
	// ScopedID is the canonical identity the record concerns.
	ScopedID string `json:"scoped_id,omitempty"`
	// Outcome is the reconciliation outcome.
	Outcome string `json:"outcome"`
	// Detail explains the reconciliation.
	Detail string `json:"detail"`
	// Revision is the record revision.
	Revision int `json:"revision"`
}

// Ledger is the full canonical state of one repository ledger.
type Ledger struct {
	SchemaVersion int                 `json:"schema_version"`
	Epics         []Epic              `json:"epics,omitempty"`
	Items         []WorkItem          `json:"items,omitempty"`
	Acts          []ActExecution      `json:"acts,omitempty"`
	Evidence      []EvidenceRefRecord `json:"evidence,omitempty"`
	Decisions     []Decision          `json:"decisions,omitempty"`
	Knowledge     []Knowledge         `json:"knowledge,omitempty"`
	// Migrations are the durable migration reconciliation records.
	//
	// They are the persisted evidence that an unresolved source record
	// was NOT silently dropped.
	Migrations []MigrationRecord `json:"migrations,omitempty"`
}

// ItemByScoped returns the work item with the given scoped identity.
func (l Ledger) ItemByScoped(id ScopedItemID) (WorkItem, bool) {
	for _, it := range l.Items {
		if it.Scoped() == id {
			return it, true
		}
	}
	return WorkItem{}, false
}

// ActByID returns the ACT execution record with the given identity.
func (l Ledger) ActByID(id ActID) (ActExecution, bool) {
	for _, a := range l.Acts {
		if a.ActID == id {
			return a, true
		}
	}
	return ActExecution{}, false
}

// EvidenceByID returns the evidence record with the given identity.
func (l Ledger) EvidenceByID(id EvidenceID) (EvidenceRefRecord, bool) {
	for _, e := range l.Evidence {
		if e.EvidenceID == id {
			return e, true
		}
	}
	return EvidenceRefRecord{}, false
}

// SortItems orders work items deterministically: by epic, then priority,
// then item identity.
func SortItems(items []WorkItem) {
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].EpicID != items[j].EpicID {
			return items[i].EpicID < items[j].EpicID
		}
		ri, rj := items[i].Priority.rank(), items[j].Priority.rank()
		if ri != rj {
			return ri < rj
		}
		return items[i].ItemID < items[j].ItemID
	})
}

// SortActs orders ACT execution records deterministically by identity.
func SortActs(acts []ActExecution) {
	sort.SliceStable(acts, func(i, j int) bool {
		return acts[i].ActID < acts[j].ActID
	})
}

// SortEvidence orders evidence records deterministically by identity.
func SortEvidence(records []EvidenceRefRecord) {
	sort.SliceStable(records, func(i, j int) bool {
		return records[i].EvidenceID < records[j].EvidenceID
	})
}
