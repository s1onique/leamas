package fwmi

// EpicRecord is one epic of the interchange package.
type EpicRecord struct {
	SchemaVersion  int         `json:"schema_version"`
	EpicID         string      `json:"epic_id"`
	Title          string      `json:"title"`
	OriginalStatus *string     `json:"original_status"`
	Lifecycle      string      `json:"lifecycle"`
	Priority       *string     `json:"priority"`
	ExitCriteria   []string    `json:"exit_criteria"`
	Items          []WorkItem  `json:"items"`
	SourceRefs     []SourceRef `json:"source_refs"`
}

// WorkItem is one reconciled obligation of the interchange package.
//
// The qualification and coverage fields are what distinguish an ABSENT
// dependency from a NOT-YET-RECOVERED dependency. Both are preserved.
type WorkItem struct {
	SchemaVersion      int            `json:"schema_version"`
	ItemID             string         `json:"item_id"`
	Title              string         `json:"title"`
	Description        string         `json:"description"`
	Workstream         *string        `json:"workstream"`
	OriginalStatus     *string        `json:"original_status"`
	Lifecycle          string         `json:"lifecycle"`
	Priority           *string        `json:"priority"`
	Qualification      string         `json:"qualification"`
	Criteria           []string       `json:"criteria"`
	CriteriaCoverage   string         `json:"criteria_coverage"`
	Dependencies       []Relationship `json:"dependencies"`
	DependencyCoverage string         `json:"dependency_coverage"`
	Blockers           []string       `json:"blockers"`
	CurrentACT         *string        `json:"current_act"`
	RelatedACTs        []string       `json:"related_acts"`
	EvidenceLocators   []string       `json:"evidence_locators"`
	ReopenConditions   []string       `json:"reopen_conditions"`
	SourceRefs         []SourceRef    `json:"source_refs"`
}

// HistoricalRecord is one indexed historical ACT execution.
//
// The original verdict is preserved verbatim. A historical verdict is never
// an acceptance decision about the engineering obligation.
type HistoricalRecord struct {
	SchemaVersion    int         `json:"schema_version"`
	ACTID            string      `json:"act_id"`
	OriginalVerdict  *string     `json:"original_verdict"`
	EpicID           *string     `json:"epic_id"`
	WorkItemIDs      []string    `json:"work_item_ids"`
	EvidenceLocators []string    `json:"evidence_locators"`
	SourceRefs       []SourceRef `json:"source_refs"`
	Reconciliation   string      `json:"reconciliation"`
}

// HistoricalIndex is the deterministic historical ACT index.
type HistoricalIndex struct {
	SchemaVersion int                `json:"schema_version"`
	Records       []HistoricalRecord `json:"records"`
}

// ReconciliationRecord is one explicit ambiguity, conflict or cross-link.
//
// An unresolved reconciliation is preserved as such: it is never silently
// resolved by the importer.
type ReconciliationRecord struct {
	SchemaVersion int         `json:"schema_version"`
	Kind          string      `json:"kind"`
	Identities    []string    `json:"identities"`
	Resolution    string      `json:"resolution"`
	Reason        string      `json:"reason"`
	SourceRefs    []SourceRef `json:"source_refs"`
}

// ReconciliationFile is the reconciliation manifest.
type ReconciliationFile struct {
	SchemaVersion int                    `json:"schema_version"`
	Records       []ReconciliationRecord `json:"records"`
}
