package domain

import "sort"

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

	Dependencies []Dependency      `json:"dependencies,omitempty"`
	Evidence     []EvidenceRef     `json:"evidence,omitempty"`
	Successors   []ScopedItemID    `json:"successors,omitempty"`
	Provenance   *Provenance       `json:"provenance,omitempty"`
	Reopen       []ReopenCondition `json:"reopen_conditions,omitempty"`

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
	ItemIDs  []ScopedItemID `json:"item_ids,omitempty"`
	Title    string         `json:"title,omitempty"`
	Verdict  ActVerdict     `json:"verdict"`
	Revision int            `json:"revision"`
	Archived bool           `json:"archived,omitempty"`

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

// Ledger is the full canonical state of one repository ledger.
type Ledger struct {
	SchemaVersion int                 `json:"schema_version"`
	Epics         []Epic              `json:"epics,omitempty"`
	Items         []WorkItem          `json:"items,omitempty"`
	Acts          []ActExecution      `json:"acts,omitempty"`
	Evidence      []EvidenceRefRecord `json:"evidence,omitempty"`
	Decisions     []Decision          `json:"decisions,omitempty"`
	Knowledge     []Knowledge         `json:"knowledge,omitempty"`
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
