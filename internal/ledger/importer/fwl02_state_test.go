package importer

import (
	"testing"

	"github.com/s1onique/leamas/internal/ledger/domain"
)

// TestF2UnresolvedReconciliationIsPersistedAsDurableState witnesses the
// W3 contract.
//
// The contract: an unresolved source record is PERSISTED as durable
// migration state. It is never a console warning only.
func TestF2UnresolvedReconciliationIsPersistedAsDurableState(t *testing.T) {
	source := SourceBoard{
		Source: "f2-durable",
		Items: []SourceItem{
			{SourceID: "W-01", EpicID: "WS-A", Title: "first", Status: SourceStatusOpen,
				DependsOn: []string{"W-MISSING"}},
		},
	}
	s := newTestStore(t)
	plan := BuildPlan(source, domain.Ledger{SchemaVersion: domain.LedgerSchemaVersion})
	if _, err := Apply(s, plan, "tx-f2-durable"); err != nil {
		t.Fatalf("apply: %v", err)
	}
	reloaded, err := s.Load()
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	if len(reloaded.Migrations) == 0 {
		t.Fatal("the unresolved reconciliation was not persisted as durable state")
	}
	found := false
	for _, rec := range reloaded.Migrations {
		if rec.Outcome == "unresolved" && rec.ScopedID == "WS-A/W-01" {
			found = true
		}
	}
	if !found {
		t.Fatalf("the unresolved reconciliation is not in the durable state: %+v", reloaded.Migrations)
	}
}

// TestF2DurableMigrationStateIsIdempotent witnesses the W3 contract.
//
// The contract: replaying the migration does not create a SECOND durable
// reconciliation record for the same source record.
func TestF2DurableMigrationStateIsIdempotent(t *testing.T) {
	source := SourceBoard{
		Source: "f2-durable-idem",
		Items: []SourceItem{
			{SourceID: "W-01", EpicID: "WS-A", Title: "first", Status: SourceStatusOpen,
				DependsOn: []string{"W-MISSING"}},
		},
	}
	s := newTestStore(t)
	first := BuildPlan(source, domain.Ledger{SchemaVersion: domain.LedgerSchemaVersion})
	if _, err := Apply(s, first, "tx-f2-durable-1"); err != nil {
		t.Fatalf("apply: %v", err)
	}
	afterFirst, err := s.Load()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	second := BuildPlan(source, afterFirst)
	if _, err := Apply(s, second, "tx-f2-durable-2"); err != nil {
		t.Fatalf("apply the replay: %v", err)
	}
	afterSecond, err := s.Load()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if len(afterSecond.Migrations) != len(afterFirst.Migrations) {
		t.Fatalf("the replay duplicated the durable migration state: %d vs %d",
			len(afterSecond.Migrations), len(afterFirst.Migrations))
	}
}

// TestF2PersistedReceiptComparesTheActualPersistedState witnesses the W5
// contract.
//
// The contract: BuildReceipt does NOT claim persisted conservation from
// planned records alone. The persisted receipt compares the ACTUAL
// persisted state.
func TestF2PersistedReceiptComparesTheActualPersistedState(t *testing.T) {
	source := SourceBoard{
		Source: "f2-receipt-persisted",
		Items: []SourceItem{
			{SourceID: "W-01", EpicID: "WS-A", Title: "first", Status: SourceStatusOpen,
				AcceptanceCriteria: []string{"criterion one"}},
		},
	}
	s := newTestStore(t)
	plan := BuildPlan(source, domain.Ledger{SchemaVersion: domain.LedgerSchemaVersion})
	if _, err := Apply(s, plan, "tx-f2-receipt"); err != nil {
		t.Fatalf("apply: %v", err)
	}
	persisted, err := s.Load()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	receipt := BuildPersistedReceipt(
		domain.Ledger{SchemaVersion: domain.LedgerSchemaVersion}, plan, persisted)
	if !receipt.Conserved {
		t.Fatalf("the persisted state is reported as not conserved: %v", receipt.Discrepancies)
	}

	// A persisted state that LOST the obligation is reported.
	lost := domain.Ledger{SchemaVersion: domain.LedgerSchemaVersion}
	receipt = BuildPersistedReceipt(
		domain.Ledger{SchemaVersion: domain.LedgerSchemaVersion}, plan, lost)
	if receipt.Conserved {
		t.Fatal("a persisted state that lost the work item is reported as conserved")
	}
	if len(receipt.Discrepancies) == 0 {
		t.Fatal("the lost work item was not reported as a discrepancy")
	}
}

// TestF2PersistedReceiptReportsALostCriterion witnesses the W5 contract.
//
// The contract: a persisted state that lost an acceptance criterion is
// reported.
func TestF2PersistedReceiptReportsALostCriterion(t *testing.T) {
	source := SourceBoard{
		Source: "f2-receipt-criterion",
		Items: []SourceItem{
			{SourceID: "W-01", EpicID: "WS-A", Title: "first", Status: SourceStatusOpen,
				AcceptanceCriteria: []string{"criterion one"}},
		},
	}
	s := newTestStore(t)
	plan := BuildPlan(source, domain.Ledger{SchemaVersion: domain.LedgerSchemaVersion})
	if _, err := Apply(s, plan, "tx-f2-receipt-crit"); err != nil {
		t.Fatalf("apply: %v", err)
	}
	persisted, err := s.Load()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	// The persisted state loses the criterion.
	for i := range persisted.Items {
		persisted.Items[i].AcceptanceCriteria = nil
	}
	receipt := BuildPersistedReceipt(
		domain.Ledger{SchemaVersion: domain.LedgerSchemaVersion}, plan, persisted)
	if receipt.Conserved {
		t.Fatal("a persisted state that lost the criterion is reported as conserved")
	}
	if receipt.Criteria != 1 {
		t.Fatalf("the receipt did not count the conserved criterion: %d", receipt.Criteria)
	}
}
