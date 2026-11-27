package importer

import (
	"testing"

	"github.com/s1onique/leamas/internal/ledger/domain"
)

// This file qualifies the FWL02-C terminal conservation corrections against
// the REAL production seams (BuildPlan, Apply, BuildPersistedReceipt, store).
//
// D1 — the conservation BOUNDARY: the expected state is the pre-import
//      canonical state plus the explicitly authorized plan, and every
//      semantic field of every record participates in the comparison.
//
// D2 — the CHANGED reconciliation replay: a changed durable migration record
//      never overwrites the prior durable migration state.

// migrationFixture returns a source board that carries one durable migration
// record.
func migrationFixture(detail string) SourceBoard {
	return SourceBoard{
		Source: "fwl02-d-source",
		Items: []SourceItem{{
			SourceID: "W-01", EpicID: "EPIC-A", Title: "the first obligation",
			Status: SourceStatusOpen,
		}},
		Migrations: []domain.MigrationRecord{{
			SchemaVersion:    domain.LedgerSchemaVersion,
			MigrationID:      "MIGRATION-FWMI-D2",
			SourceRepository: "clinemm",
			SourceHead:       "29774ba05851f9dd5a46a33a90d963ff3ef286d5",
			ScopedID:         "EPIC-A/X,EPIC-B/X",
			Outcome:          "POSSIBLE_CROSS_LINK",
			Detail:           detail,
			Revision:         1,
		}},
	}
}

// TestD2ChangedMigrationRecordIsPreservedAndReported witnesses D2.
//
// The contract: a CHANGED durable migration record never overwrites the
// prior durable migration state. The prior record is conserved, the change
// is reported as an explicit reconciliation, and the conservation receipt
// reports the import as conserved.
func TestD2ChangedMigrationRecordIsPreservedAndReported(t *testing.T) {
	s := newTestStore(t)
	empty := domain.Ledger{SchemaVersion: domain.LedgerSchemaVersion}
	first := BuildPlan(migrationFixture("UNRESOLVED: the original reason"), empty)
	if len(first.AdditionalMigrations) != 1 {
		t.Fatalf("planned migration records = %d, want 1", len(first.AdditionalMigrations))
	}
	if _, err := Apply(s, first, "tx-d2-1"); err != nil {
		t.Fatalf("apply: %v", err)
	}
	after, err := s.Load()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	prior, ok := migrationByID(after, "MIGRATION-FWMI-D2")
	if !ok {
		t.Fatal("the durable migration record was not persisted")
	}
	if prior.Detail != "UNRESOLVED: the original reason" {
		t.Fatalf("the persisted detail = %q", prior.Detail)
	}

	// The reconciliation record CHANGED.
	plan := BuildPlan(migrationFixture("UNRESOLVED: a changed reason"), after)
	if len(plan.AdditionalMigrations) != 0 {
		t.Fatalf("the changed reconciliation record overwrites the prior state: %+v",
			plan.AdditionalMigrations)
	}
	reported := false
	for _, rec := range plan.Reconciliations {
		if rec.ScopedID == "MIGRATION-FWMI-D2" && rec.Outcome == ReconciliationSourceChanged {
			reported = true
		}
	}
	if !reported {
		t.Fatalf("the changed reconciliation record was not reported: %+v", plan.Reconciliations)
	}
	if _, err := Apply(s, plan, "tx-d2-2"); err != nil {
		t.Fatalf("apply the replay: %v", err)
	}
	persisted, err := s.Load()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	conserved, ok := migrationByID(persisted, "MIGRATION-FWMI-D2")
	if !ok {
		t.Fatal("the prior durable migration state was lost")
	}
	if conserved.Detail != "UNRESOLVED: the original reason" {
		t.Fatalf("the prior durable migration state was overwritten: %q", conserved.Detail)
	}
	receipt := BuildPersistedReceipt(after, plan, persisted)
	if !receipt.Conserved {
		t.Fatalf("the changed reconciliation replay is reported as not conserved: %v",
			receipt.Discrepancies)
	}
}

// TestD2IdenticalMigrationRecordIsANoOp witnesses D2 for the identical
// record.
//
// The contract: an IDENTICAL reconciliation record produces no canonical
// delta. The replay does not create a second durable record.
func TestD2IdenticalMigrationRecordIsANoOp(t *testing.T) {
	s := newTestStore(t)
	empty := domain.Ledger{SchemaVersion: domain.LedgerSchemaVersion}
	source := migrationFixture("UNRESOLVED: the original reason")
	first := BuildPlan(source, empty)
	if _, err := Apply(s, first, "tx-d2-noop-1"); err != nil {
		t.Fatalf("apply: %v", err)
	}
	after, err := s.Load()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	replay := BuildPlan(source, after)
	if len(replay.AdditionalMigrations) != 0 {
		t.Fatalf("the identical replay created a durable record: %+v", replay.AdditionalMigrations)
	}
	if _, err := Apply(s, replay, "tx-d2-noop-2"); err != nil {
		t.Fatalf("apply the replay: %v", err)
	}
	persisted, err := s.Load()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if len(persisted.Migrations) != len(after.Migrations) {
		t.Fatalf("the identical replay duplicated a durable record: %d != %d",
			len(persisted.Migrations), len(after.Migrations))
	}
	receipt := BuildPersistedReceipt(after, replay, persisted)
	if !receipt.Conserved {
		t.Fatalf("the identical replay is reported as not conserved: %v", receipt.Discrepancies)
	}
}

// migrationByID returns the durable migration record with the given identity.
func migrationByID(l domain.Ledger, id string) (domain.MigrationRecord, bool) {
	for _, rec := range l.Migrations {
		if rec.MigrationID == id {
			return rec, true
		}
	}
	return domain.MigrationRecord{}, false
}
