package importer

import (
	"encoding/json"

	"github.com/s1onique/leamas/internal/ledger/domain"
)

// reconcileMigrations reconciles the durable migration records that the
// SOURCE carries with the canonical migration state.
//
// The three outcomes are explicit:
//
//   - an IDENTICAL record is a no-op: the canonical state already carries it
//     and the replay changes nothing;
//   - a CHANGED record NEVER overwrites the prior durable migration state:
//     the prior record is conserved and the change is reported as an
//     explicit reconciliation;
//   - a NEW record is created.
//
// A durable migration record is the persisted evidence that a source record
// could not be reconciled. Overwriting it would erase that evidence, so the
// prior state is always preserved.
func reconcileMigrations(p *Plan, existing domain.Ledger, records []domain.MigrationRecord) {
	prior := map[string]domain.MigrationRecord{}
	for _, rec := range existing.Migrations {
		prior[rec.MigrationID] = rec
	}
	for _, rec := range records {
		before, ok := prior[rec.MigrationID]
		if !ok {
			p.AdditionalMigrations = append(p.AdditionalMigrations, rec)
			p.Reconciliations = append(p.Reconciliations, Reconciliation{
				ScopedID: rec.MigrationID,
				Outcome:  ReconciliationCreated,
				Detail:   "the reconciliation record did not exist in the canonical state",
			})
			continue
		}
		if sameMigrationContent(before, rec) {
			p.Reconciliations = append(p.Reconciliations, Reconciliation{
				ScopedID: rec.MigrationID,
				Outcome:  ReconciliationUnchanged,
				Detail:   "the replay produced no canonical delta",
			})
			continue
		}
		p.Reconciliations = append(p.Reconciliations, Reconciliation{
			ScopedID: rec.MigrationID,
			Outcome:  ReconciliationSourceChanged,
			Detail: "the reconciliation record changed; the prior durable migration state is conserved " +
				"and the change requires explicit reconciliation",
		})
	}
}

// sameMigrationContent reports whether a canonical migration record carries
// the same content as the preserved one.
//
// The comparison uses the canonical serialization of the record: every field
// participates, so a changed reason, outcome, detail or source provenance is
// a change.
func sameMigrationContent(a, b domain.MigrationRecord) bool {
	left, err := json.Marshal(a)
	if err != nil {
		return false
	}
	right, err := json.Marshal(b)
	if err != nil {
		return false
	}
	return string(left) == string(right)
}
