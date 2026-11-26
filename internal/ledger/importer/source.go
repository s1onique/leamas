// Package importer implements the dry-run import and reconciliation
// primitive for existing Factory boards.
//
// The importer is deliberately conservative:
//
//   - it preserves the source record identity and its provenance;
//   - it reports ambiguities and unresolved references explicitly;
//   - it never promotes a conversational claim into a repository-verified
//     fact;
//   - it is idempotent: importing the same source twice produces no
//     unintended canonical delta;
//   - an unknown source status remains durably non-executable until it is
//     explicitly reconciled;
//   - a changed historical source is reported as an explicit reconciliation,
//     never applied as a silent overwrite of the canonical work.
package importer

import (
	"errors"
	"strings"

	"github.com/s1onique/leamas/internal/ledger/domain"
)

// ErrAmbiguousImport is returned when an import cannot proceed because of
// unresolved load-bearing ambiguity.
var ErrAmbiguousImport = errors.New("ambiguous import")

// SourceStatus is a lifecycle status observed in a source board.
type SourceStatus string

// Source statuses recognised by the importer.
//
// The list is the SET of statuses the importer understands. A status that
// is not in this list is an UNKNOWN status: it is conserved verbatim and
// the imported record stays durably non-executable until it is explicitly
// reconciled.
const (
	SourceStatusOpen               SourceStatus = "open"
	SourceStatusInProgress         SourceStatus = "in_progress"
	SourceStatusBlocked            SourceStatus = "blocked"
	SourceStatusClosed             SourceStatus = "closed"
	SourceStatusClosedInsufficient SourceStatus = "closed_insufficient"
	SourceStatusDone               SourceStatus = "done"
	SourceStatusUnfinished         SourceStatus = "unfinished"
	SourceStatusDeferred           SourceStatus = "deferred"
	SourceStatusPartial            SourceStatus = "partial"
	SourceStatusQueued             SourceStatus = "queued"
	SourceStatusInvestigate        SourceStatus = "investigate"
	SourceStatusHypothesis         SourceStatus = "hypothesis"
	SourceStatusLiveWitness        SourceStatus = "live_witness"
	SourceStatusFinalPreflight     SourceStatus = "final_preflight"
)

// IsKnownSourceStatus reports whether the importer understands the status.
//
// The comparison is case-INSENSITIVE: the original board spells its
// statuses in UPPERCASE, while the repository-native board spells them in
// lower case. The SPELLING of the status is conserved verbatim on the
// imported record; only the RECOGNITION is case-insensitive.
//
// An unknown status is never coerced into a known lifecycle state: the
// record keeps the exact source status and stays non-executable.
func IsKnownSourceStatus(s SourceStatus) bool {
	switch NormalizeSourceStatus(s) {
	case SourceStatusOpen, SourceStatusInProgress, SourceStatusBlocked,
		SourceStatusClosed, SourceStatusClosedInsufficient, SourceStatusDone,
		SourceStatusUnfinished, SourceStatusDeferred, SourceStatusPartial,
		SourceStatusQueued, SourceStatusInvestigate, SourceStatusHypothesis,
		SourceStatusLiveWitness, SourceStatusFinalPreflight:
		return true
	}
	return false
}

// NormalizeSourceStatus returns the canonical spelling of a source status.
//
// It is the RECOGNITION form only. The exact source spelling is conserved
// verbatim on the imported record.
func NormalizeSourceStatus(s SourceStatus) SourceStatus {
	return SourceStatus(strings.ToLower(string(s)))
}

// IsUnfinishedSourceStatus reports whether the status denotes work that is
// not finished.
//
// A CLOSED_INSUFFICIENT status is NOT finished work: the investigation was
// closed without establishing the obligation. It is a disposition, not a
// completion.
//
// The unfinished CLASS of the original board excludes the CLOSED records
// and the CLOSED_INSUFFICIENT records: the insufficient investigation is
// its own disposition class, and it is reported separately so the
// disposition is never hidden inside the unfinished count.
func IsUnfinishedSourceStatus(s SourceStatus) bool {
	switch NormalizeSourceStatus(s) {
	case SourceStatusClosed, SourceStatusDone, SourceStatusClosedInsufficient:
		return false
	}
	return true
}

// SourceItem is one source work-item record.
type SourceItem struct {
	// SourceID is the identity of the record in the source board. It is
	// preserved verbatim through the import.
	SourceID string `json:"source_id"`
	// EpicID is the source workstream the record belongs to.
	EpicID string `json:"epic_id"`
	// Title is the source title.
	Title string `json:"title"`
	// Status is the observed source status. It is conserved.
	Status SourceStatus `json:"status"`
	// Priority is the observed source priority.
	Priority string `json:"priority,omitempty"`
	// Owner is the observed source owner.
	Owner string `json:"owner,omitempty"`
	// AcceptanceCriteria are the source acceptance criteria.
	AcceptanceCriteria []string `json:"acceptance_criteria,omitempty"`
	// ExitCriteria are the source exit criteria.
	ExitCriteria []string `json:"exit_criteria,omitempty"`
	// DependsOn lists the source dependency references.
	DependsOn []string `json:"depends_on,omitempty"`
	// SourceConfidence records whether the source record is a
	// repository-verified fact or a conversational claim.
	SourceConfidence domain.ProvenanceConfidence `json:"source_confidence,omitempty"`
	// SourceRow preserves the original row text of the source record.
	SourceRow string `json:"source_row,omitempty"`
	// SourceLine preserves the original line locator of the source record.
	SourceLine int `json:"source_line,omitempty"`
	// SourceDetail preserves the original free-form source detail.
	SourceDetail string `json:"source_detail,omitempty"`
}

// SourceBoard is the parsed source board.
type SourceBoard struct {
	// Source identifies the board the records came from.
	Source string `json:"source"`
	// EpicTitles maps a workstream identity to its title.
	EpicTitles map[string]string `json:"epic_titles,omitempty"`
	// Items are the source work-item records.
	Items []SourceItem `json:"items"`
	// ExitCriteria are the board-scoped exit criteria that are not owned by
	// a single work item. They are conserved as well.
	ExitCriteria []string `json:"exit_criteria,omitempty"`
}

// Ambiguity is an unresolved source ambiguity.
type Ambiguity struct {
	// SourceID is the source record the ambiguity belongs to.
	SourceID string `json:"source_id"`
	// Kind classifies the ambiguity.
	Kind string `json:"kind"`
	// Detail explains the ambiguity.
	Detail string `json:"detail"`
}

// UnresolvedReference is a source reference that did not resolve.
type UnresolvedReference struct {
	// SourceID is the source record carrying the reference.
	SourceID string `json:"source_id"`
	// Ref is the unresolved reference.
	Ref string `json:"ref"`
	// Kind classifies the reference.
	Kind string `json:"kind"`
}

// Reconciliation outcomes.
//
// They distinguish the states a replayed source record can be in
// relative to the canonical ledger.
const (
	// ReconciliationUnchanged means the replay produced no canonical delta.
	ReconciliationUnchanged = "unchanged"
	// ReconciliationCreated means the identity did not exist yet.
	ReconciliationCreated = "created"
	// ReconciliationConflict means the source record differs from the
	// existing canonical record. The existing record is NOT
	// overwritten: the conflict is reported.
	ReconciliationConflict = "conflict"
	// ReconciliationSourceChanged means the source record changed while the
	// canonical record kept its identity. The canonical record is NOT
	// overwritten: the change is reported for explicit reconciliation.
	ReconciliationSourceChanged = "source_changed"
	// ReconciliationUnresolved means the source record could not be
	// reconciled: its identity, status or dependency is unresolved. It
	// never acquires executable authority.
	ReconciliationUnresolved = "unresolved"
)

// Reconciliation is one explicit reconciliation outcome for an existing
// canonical identity.
type Reconciliation struct {
	// ScopedID is the reconciled work-item identity.
	ScopedID string `json:"scoped_id"`
	// Outcome classifies the reconciliation.
	Outcome string `json:"outcome"`
	// Detail explains the outcome.
	Detail string `json:"detail"`
}

// Plan is the deterministic import plan.
type Plan struct {
	// Epics are the epic records to create.
	Epics []domain.Epic `json:"epics"`
	// Items are the work-item records to create.
	Items []domain.WorkItem `json:"items"`
	// Ambiguities are the explicit ambiguity reports.
	Ambiguities []Ambiguity `json:"ambiguities"`
	// Unresolved are the explicit unresolved-reference reports.
	Unresolved []UnresolvedReference `json:"unresolved"`
	// Quarantined are the source identity groups that were quarantined
	// because their source records conflicted. A quarantined group never
	// acquires authority and never appears as an imported record.
	Quarantined []Quarantine `json:"quarantined"`
	// Skipped lists the source records that already exist and therefore
	// produce no canonical delta.
	Skipped []string `json:"skipped"`
	// SkippedEpics lists the epic scopes that already exist in the
	// canonical state. Their existing metadata is conserved: the import
	// does not overwrite it.
	SkippedEpics []string `json:"skipped_epics,omitempty"`
	// Reconciliations are the explicit per-identity reconciliation
	// outcomes. Every source identity is accounted for in exactly one
	// reconciliation record.
	Reconciliations []Reconciliation `json:"reconciliations,omitempty"`
}

// Quarantine is a complete source identity group that was withheld from
// the canonical import because its source records conflicted.
//
// The whole group is quarantined: input order must never select a winner.
type Quarantine struct {
	// SourceID is the source identity of the quarantined group.
	SourceID string `json:"source_id"`
	// Reason explains why the group was quarantined.
	Reason string `json:"reason"`
	// Records lists the conflicting source record titles, in source order.
	Records []string `json:"records"`
}
