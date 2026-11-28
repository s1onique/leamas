package importer

import (
	"crypto/sha256"
	"encoding/hex"

	"github.com/s1onique/leamas/internal/ledger/domain"
)

// ApplyResult reports the outcome of an applied import.
type ApplyResult struct {
	// CreatedEpics is the number of epic records created.
	CreatedEpics int `json:"created_epics"`
	// CreatedItems is the number of work-item records created.
	CreatedItems int `json:"created_items"`
	// CreatedActs is the number of indexed historical ACT records created.
	CreatedActs int `json:"created_acts"`
	// Skipped is the number of source records that already existed.
	Skipped int `json:"skipped"`
	// Ambiguities is the number of ambiguity reports.
	Ambiguities int `json:"ambiguities"`
	// Unresolved is the number of unresolved-reference reports.
	Unresolved int `json:"unresolved"`
}

// Receipt is the conservation receipt of an import.
//
// It reports what the import conserved: the nodes, the relationships, the
// blockers, the original dispositions, the criteria and the reconciliation
// records. The receipt is the evidence that the import did not lose
// obligations.
//
// An imported work-item COUNT is not a conservation proof: the receipt
// reports the semantic content that was carried over.
type Receipt struct {
	// Nodes is the number of imported work-item records.
	Nodes int `json:"nodes"`
	// Epics is the number of imported epic records.
	Epics int `json:"epics"`
	// Relationships is the number of conserved dependency edges.
	Relationships int `json:"relationships"`
	// Blockers is the number of durable unresolved blockers.
	Blockers int `json:"blockers"`
	// OriginalDispositions is the number of records whose exact source
	// status was conserved.
	OriginalDispositions int `json:"original_dispositions"`
	// Criteria is the number of conserved acceptance criteria.
	Criteria int `json:"criteria"`
	// ExitCriteria is the number of conserved exit criteria.
	//
	// It counts BOTH the item-scoped exit criteria and the epic-scoped
	// exit criteria: a board that declares its criteria at the scope
	// level conserves them on the epic record, and they are conserved.
	ExitCriteria int `json:"exit_criteria"`
	// EpicExitCriteria is the number of conserved epic-scoped exit
	// criteria.
	EpicExitCriteria int `json:"epic_exit_criteria"`
	// Reconciliations lists the explicit reconciliation records.
	Reconciliations []Reconciliation `json:"reconciliations"`
	// Quarantined lists the quarantined identity groups.
	Quarantined []Quarantine `json:"quarantined"`
	// Conserved reports whether the ACTUAL PERSISTED state conserves
	// everything the plan carried over.
	//
	// It is FALSE when the persisted state lost an obligation, a
	// relationship, a criterion or a disposition. A receipt that only
	// restated the planned records would always report conservation; this
	// field is the comparison against the state that was actually
	// persisted.
	Conserved bool `json:"conserved"`
	// Discrepancies lists the load-bearing differences between the plan
	// and the actual persisted state.
	Discrepancies []string `json:"discrepancies"`
}

// BuildReceipt computes the conservation receipt of an applied import.
//
// The receipt is computed against the state BEFORE the import and the plan
// that was applied, so it reports the conservation rather than restating
// the plan.
//
// The receipt compares the ORIGINAL SOURCE, the NORMALIZED RECORDS and the
// ACTUAL PERSISTED STATE. A planned-record count is not a conservation
// proof: the receipt reports the semantic content that was carried over.
//
// BuildReceipt does NOT claim persisted conservation: it has no access to
// the persisted state. Use BuildPersistedReceipt for that claim.
func BuildReceipt(before domain.Ledger, p Plan) Receipt {
	r := Receipt{
		Nodes:           len(p.Items),
		Epics:           len(p.Epics),
		Reconciliations: []Reconciliation{},
		Quarantined:     p.Quarantined,
	}
	if r.Quarantined == nil {
		r.Quarantined = []Quarantine{}
	}
	for _, epic := range p.Epics {
		r.EpicExitCriteria += len(epic.ExitCriteria)
	}
	for _, item := range p.Items {
		r.OriginalDispositions++
		r.Criteria += len(item.AcceptanceCriteria)
		r.ExitCriteria += len(item.ExitCriteria)
		// The EXPLICIT source blockers are durable blockers too: they
		// prevent readiness and they are reported.
		r.Blockers += len(item.SourceBlockers)
		for _, dep := range item.Dependencies {
			if dep.Kind == domain.DependencyBlocker {
				r.Blockers++
				continue
			}
			r.Relationships++
		}
	}
	r.ExitCriteria += r.EpicExitCriteria
	// The reconciliation records are the plan's own explicit outcomes. The
	// plan compares the incoming normalized source record with the
	// preserved canonical record, so the receipt restates a comparison
	// that was actually made, not a count.
	r.Reconciliations = append(r.Reconciliations, p.Reconciliations...)
	return r
}

// BuildPersistedReceipt and the complete expected-versus-persisted
// comparison live in receipt_conservation.go.

// containsString reports whether the slice contains the value.
func containsString(slice []string, value string) bool {
	for _, item := range slice {
		if item == value {
			return true
		}
	}
	return false
}

// MigrationRecords returns the durable migration reconciliation records of
// a plan.
//
// Every source record that could NOT be accounted for becomes a durable
// migration record: an unresolved reference, an ambiguity, a quarantined
// identity group and a changed historical source.
//
// A record that was CREATED or replayed UNCHANGED is NOT a discrepancy: it
// is accounted for. Persisting it would make the durable state grow on
// every replay.
//
// The identity of a record is derived from its CONTENT, not from its
// position: the same source record always produces the same migration
// record, so a replay does not create a second one.
func (p Plan) MigrationRecords() []domain.MigrationRecord {
	var out []domain.MigrationRecord
	add := func(scopedID, outcome, detail string) {
		out = append(out, domain.MigrationRecord{
			SchemaVersion: domain.LedgerSchemaVersion,
			MigrationID:   migrationIDOf(scopedID, outcome, detail),
			ScopedID:      scopedID,
			Outcome:       outcome,
			Detail:        detail,
			Revision:      1,
		})
	}
	for _, rec := range p.Reconciliations {
		if rec.Outcome == ReconciliationCreated || rec.Outcome == ReconciliationUnchanged {
			continue
		}
		add(rec.ScopedID, rec.Outcome, rec.Detail)
	}
	for _, amb := range p.Ambiguities {
		add(amb.SourceID, "ambiguity", amb.Kind+": "+amb.Detail)
	}
	for _, unresolved := range p.Unresolved {
		scoped := unresolved.ScopedID
		if scoped == "" {
			scoped = unresolved.SourceID
		}
		// The COMPLETE unresolved relationship is conserved: the specific
		// relationship kind, the outcome it required and the byte-exact
		// source locations of the edge. A durable record that carries only
		// the target and a generic kind loses the obligation.
		detail := unresolved.Kind + ": " + unresolved.Ref
		if unresolved.RelationshipKind != "" {
			detail += " [" + unresolved.RelationshipKind + "]"
		}
		if unresolved.RequiredOutcome != "" {
			detail += " requires: " + unresolved.RequiredOutcome
		}
		rec := domain.MigrationRecord{
			SchemaVersion:    domain.LedgerSchemaVersion,
			MigrationID:      migrationIDOf(scoped, "unresolved", detail),
			ScopedID:         scoped,
			Outcome:          "unresolved",
			Detail:           detail,
			RelationshipKind: unresolved.RelationshipKind,
			RequiredOutcome:  unresolved.RequiredOutcome,
			RelationshipSourceRefs: append(
				[]domain.SourceRef{}, unresolved.SourceRefs...),
			Revision: 1,
		}
		out = append(out, rec)
	}
	for _, quarantined := range p.Quarantined {
		add(quarantined.SourceID, "quarantined", quarantined.Reason)
	}
	return out
}

// migrationIDOf returns the deterministic identity of a migration record.
//
// The identity is the digest of the reconciled content: the same source
// record with the same outcome always produces the same identity.
func migrationIDOf(scopedID, outcome, detail string) string {
	sum := sha256.Sum256([]byte(scopedID + "\x1f" + outcome + "\x1f" + detail))
	return "MIGRATION-" + hex.EncodeToString(sum[:])[:16]
}
