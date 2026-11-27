package importer

import (
	"testing"

	"github.com/s1onique/leamas/internal/ledger/domain"
)

// This file qualifies the ADVERSARIAL DISCRIMINATORS of the FWMI v1
// interchange contract against the REAL production seams (BuildPlan, Apply,
// BuildPersistedReceipt, store).
//
// D3 — deleting a persisted epic must invalidate conservation.
// D4 — changing a persisted priority must invalidate conservation.
// D5 — a changed historical replay must not silently erase the previous ACT
//      records.
//
// These are the discriminators that a permissive adapter fails silently: they
// are only observable through the persisted state, never through a plan.

// TestD3DeletedPersistedEpicInvalidatesConservation witnesses D3.
//
// The contract: a persisted state that LOST a planned epic is NOT conserved.
// The conservation receipt compares the ACTUAL PERSISTED state, so the loss
// of an epic scope is reported as an explicit discrepancy.
func TestD3DeletedPersistedEpicInvalidatesConservation(t *testing.T) {
	source := SourceBoard{
		Source:           "d3-epic",
		Items:            []SourceItem{{SourceID: "W-01", EpicID: "WS-A", Title: "first", Status: SourceStatusOpen}},
		EpicExitCriteria: map[string][]string{"WS-A": {"criterion one"}},
	}

	s := newTestStore(t)
	plan := BuildPlan(source, domain.Ledger{SchemaVersion: domain.LedgerSchemaVersion})
	if len(plan.Epics) != 1 {
		t.Fatalf("expected 1 planned epic, got %d", len(plan.Epics))
	}
	if _, err := Apply(s, plan, "tx-d3"); err != nil {
		t.Fatalf("apply: %v", err)
	}
	persisted, err := s.Load()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	receipt := BuildPersistedReceipt(
		domain.Ledger{SchemaVersion: domain.LedgerSchemaVersion}, plan, persisted)
	if !receipt.Conserved {
		t.Fatalf("the intact persisted state is reported as not conserved: %v", receipt.Discrepancies)
	}

	// The persisted epic is DELETED. The scope is gone.
	deleted := persisted
	deleted.Epics = nil
	receipt = BuildPersistedReceipt(
		domain.Ledger{SchemaVersion: domain.LedgerSchemaVersion}, plan, deleted)
	if receipt.Conserved {
		t.Fatal("a persisted state that lost the epic is reported as conserved")
	}
	found := false
	for _, d := range receipt.Discrepancies {
		if d == "the persisted state lost the epic WS-A" {
			found = true
		}
	}
	if !found {
		t.Fatalf("the lost epic is not reported as a discrepancy: %v", receipt.Discrepancies)
	}
}

// TestD4ChangedPersistedPriorityInvalidatesConservation witnesses D4.
//
// The contract: a persisted work item whose PRIORITY changed is NOT conserved.
// The priority is a load-bearing part of the imported obligation: a silent
// priority change reorders the work and is reported as a discrepancy.
func TestD4ChangedPersistedPriorityInvalidatesConservation(t *testing.T) {
	source := SourceBoard{
		Source: "d4-priority",
		Items: []SourceItem{
			{SourceID: "W-01", EpicID: "WS-A", Title: "first", Status: SourceStatusOpen, Priority: "P1"},
		},
	}

	s := newTestStore(t)
	plan := BuildPlan(source, domain.Ledger{SchemaVersion: domain.LedgerSchemaVersion})
	if _, err := Apply(s, plan, "tx-d4"); err != nil {
		t.Fatalf("apply: %v", err)
	}
	persisted, err := s.Load()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	receipt := BuildPersistedReceipt(
		domain.Ledger{SchemaVersion: domain.LedgerSchemaVersion}, plan, persisted)
	if !receipt.Conserved {
		t.Fatalf("the intact persisted state is reported as not conserved: %v", receipt.Discrepancies)
	}

	// The persisted priority is CHANGED.
	changed := persisted
	changed.Items = append([]domain.WorkItem{}, persisted.Items...)
	changed.Items[0].Priority = domain.PriorityP3
	receipt = BuildPersistedReceipt(
		domain.Ledger{SchemaVersion: domain.LedgerSchemaVersion}, plan, changed)
	if receipt.Conserved {
		t.Fatal("a persisted state that changed the priority is reported as conserved")
	}
	found := false
	for _, d := range receipt.Discrepancies {
		if d == "the persisted work item WS-A/W-01 changed its priority" {
			found = true
		}
	}
	if !found {
		t.Fatalf("the changed priority is not reported as a discrepancy: %v", receipt.Discrepancies)
	}
}

// TestD5ChangedHistoricalReplayDoesNotEraseThePreviousACTRecords witnesses D5.
//
// The contract: a CHANGED historical replay does not erase the previous ACT
// records. The canonical record is conserved, the change is reported as an
// explicit reconciliation, and the previous record survives the replay.
func TestD5ChangedHistoricalReplayDoesNotEraseThePreviousACTRecords(t *testing.T) {
	source := SourceBoard{
		Source: "d5-historical",
		Items:  []SourceItem{{SourceID: "W-01", EpicID: "WS-A", Title: "first", Status: SourceStatusOpen}},
		Acts: []SourceACT{{
			SourceID:         "ACT-D5-01",
			OriginalVerdict:  "PASS_STRUCTURAL_ONLY",
			EpicID:           "WS-A",
			WorkItemIDs:      []string{"W-01"},
			EvidenceLocators: []string{".factory/evidence/act-d5-01.md"},
			Reconciliation:   "UNAMBIGUOUS",
		}},
	}

	s := newTestStore(t)
	first := BuildPlan(source, domain.Ledger{SchemaVersion: domain.LedgerSchemaVersion})
	if len(first.Acts) != 1 {
		t.Fatalf("expected 1 planned historical ACT, got %d", len(first.Acts))
	}
	if _, err := Apply(s, first, "tx-d5-1"); err != nil {
		t.Fatalf("apply: %v", err)
	}
	afterFirst, err := s.Load()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	prior, ok := afterFirst.ActByID("ACT-D5-01")
	if !ok {
		t.Fatal("the historical ACT was not persisted")
	}
	if prior.OriginalVerdict != "PASS_STRUCTURAL_ONLY" {
		t.Fatalf("the persisted original verdict changed: %q", prior.OriginalVerdict)
	}

	// The historical source CHANGED: the original verdict differs.
	changed := SourceBoard{
		Source: "d5-historical",
		Items:  []SourceItem{{SourceID: "W-01", EpicID: "WS-A", Title: "first", Status: SourceStatusOpen}},
		Acts: []SourceACT{{
			SourceID:         "ACT-D5-01",
			OriginalVerdict:  "FAILED",
			EpicID:           "WS-A",
			WorkItemIDs:      []string{"W-01"},
			EvidenceLocators: []string{".factory/evidence/act-d5-01.md"},
			Reconciliation:   "UNAMBIGUOUS",
		}},
	}
	plan := BuildPlan(changed, afterFirst)
	if len(plan.Acts) != 0 {
		t.Fatalf("the changed historical replay overwrote the canonical ACT: %d records", len(plan.Acts))
	}
	reported := false
	for _, rec := range plan.Reconciliations {
		if rec.ScopedID == "ACT-D5-01" && rec.Outcome == ReconciliationSourceChanged {
			reported = true
		}
	}
	if !reported {
		t.Fatalf("the changed historical replay was not reported: %+v", plan.Reconciliations)
	}
	if _, err := Apply(s, plan, "tx-d5-2"); err != nil {
		t.Fatalf("apply the replay: %v", err)
	}
	afterSecond, err := s.Load()
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	survived, ok := afterSecond.ActByID("ACT-D5-01")
	if !ok {
		t.Fatal("the changed historical replay ERASED the previous ACT record")
	}
	if survived.OriginalVerdict != "PASS_STRUCTURAL_ONLY" {
		t.Fatalf("the previous ACT record was rewritten by the changed replay: %q",
			survived.OriginalVerdict)
	}
}
