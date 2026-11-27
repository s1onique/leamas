package importer

import (
	"testing"

	"github.com/s1onique/leamas/internal/ledger/domain"
)

// This file qualifies the FWL02 importer corrections against the REAL
// production seams (BuildPlan, Apply, store).
//
// F1 — epic exit criteria and workstream identity conservation.
// F2 — dependency-only changed source replay.

// TestF1EpicExitCriteriaSurvivePersistenceAndReload witnesses F1.
//
// The contract: the epic exit criteria of the source board retain their
// EXACT content in the canonical state, and they survive an independent
// reload.
func TestF1EpicExitCriteriaSurvivePersistenceAndReload(t *testing.T) {
	criteria := []string{
		"Reasoning and tool continuity qualified across required provider routes",
		"Explicit thinking configuration reaches the actual provider request",
	}
	source := SourceBoard{
		Source:           "f1-source",
		Items:            []SourceItem{{SourceID: "W-01", EpicID: "WS-A", Title: "first", Status: SourceStatusOpen}},
		EpicExitCriteria: map[string][]string{"WS-A": criteria},
	}

	plan := BuildPlan(source, domain.Ledger{SchemaVersion: domain.LedgerSchemaVersion})
	if len(plan.Epics) != 1 {
		t.Fatalf("expected 1 planned epic, got %d", len(plan.Epics))
	}
	if len(plan.Epics[0].ExitCriteria) != len(criteria) {
		t.Fatalf("the planned epic lost the exit criteria: %d of %d",
			len(plan.Epics[0].ExitCriteria), len(criteria))
	}

	receipt := BuildReceipt(domain.Ledger{SchemaVersion: domain.LedgerSchemaVersion}, plan)
	if receipt.EpicExitCriteria != len(criteria) {
		t.Fatalf("the receipt lost the epic exit criteria: %d of %d",
			receipt.EpicExitCriteria, len(criteria))
	}
	if receipt.ExitCriteria != len(criteria) {
		t.Fatalf("the receipt did not count the epic exit criteria: %d of %d",
			receipt.ExitCriteria, len(criteria))
	}

	s := newTestStore(t)
	if _, err := Apply(s, plan, "tx-f1"); err != nil {
		t.Fatalf("apply: %v", err)
	}
	reloaded, err := s.Load()
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	if len(reloaded.Epics) != 1 {
		t.Fatalf("expected 1 persisted epic, got %d", len(reloaded.Epics))
	}
	persisted := reloaded.Epics[0].ExitCriteria
	if len(persisted) != len(criteria) {
		t.Fatalf("the persisted epic lost the exit criteria: %d of %d",
			len(persisted), len(criteria))
	}
	for i := range criteria {
		if persisted[i] != criteria[i] {
			t.Fatalf("the persisted exit criterion %d changed: %q != %q",
				i, persisted[i], criteria[i])
		}
	}
}

// TestF1BoardScopedExitCriteriaConserveOnSingleScopeEpic witnesses F1 for
// a board that declares its exit criteria at the board level.
//
// The contract: a board with a single epic scope owns its board-scoped
// exit criteria. They are conserved on that scope's epic record.
func TestF1BoardScopedExitCriteriaConserveOnSingleScopeEpic(t *testing.T) {
	criteria := []string{"criterion one", "criterion two"}
	source := SourceBoard{
		Source:       "f1-board",
		Items:        []SourceItem{{SourceID: "W-01", EpicID: "WS-A", Title: "first"}},
		ExitCriteria: criteria,
	}

	plan := BuildPlan(source, domain.Ledger{SchemaVersion: domain.LedgerSchemaVersion})
	if len(plan.Epics) != 1 {
		t.Fatalf("expected 1 planned epic, got %d", len(plan.Epics))
	}
	if len(plan.Epics[0].ExitCriteria) != len(criteria) {
		t.Fatalf("the board-scoped exit criteria were lost: %d of %d",
			len(plan.Epics[0].ExitCriteria), len(criteria))
	}
}

// TestF2DependencyOnlySourceChangeIsDetected witnesses F2.
//
// The contract: a source record whose DEPENDENCY GRAPH changed is a
// CHANGED HISTORICAL SOURCE. The replay reports the change explicitly and
// never overwrites the evolved canonical work.
func TestF2DependencyOnlySourceChangeIsDetected(t *testing.T) {
	source := SourceBoard{
		Source: "f2-source",
		Items: []SourceItem{
			{SourceID: "W-01", EpicID: "WS-A", Title: "first", Status: SourceStatusOpen},
			{SourceID: "W-02", EpicID: "WS-A", Title: "second", Status: SourceStatusOpen},
		},
	}

	s := newTestStore(t)
	first := BuildPlan(source, domain.Ledger{SchemaVersion: domain.LedgerSchemaVersion})
	if _, err := Apply(s, first, "tx-f2-1"); err != nil {
		t.Fatalf("apply: %v", err)
	}
	after, err := s.Load()
	if err != nil {
		t.Fatalf("load: %v", err)
	}

	changed := SourceBoard{
		Source: "f2-source",
		Items: []SourceItem{
			{SourceID: "W-01", EpicID: "WS-A", Title: "first", Status: SourceStatusOpen},
			{SourceID: "W-02", EpicID: "WS-A", Title: "second", Status: SourceStatusOpen,
				DependsOn: []string{"W-01"}},
		},
	}
	plan := BuildPlan(changed, after)
	changedReported := false
	for _, rec := range plan.Reconciliations {
		if rec.ScopedID == "WS-A/W-02" && rec.Outcome == ReconciliationSourceChanged {
			changedReported = true
		}
	}
	if !changedReported {
		t.Fatalf("the dependency-only source change was not reported: %+v", plan.Reconciliations)
	}
	if len(plan.Items) != 0 {
		t.Fatalf("the changed source overwrote the canonical work: %d records", len(plan.Items))
	}
}

// TestF2ExactReplayIsUnchangedAfterDependencyImport witnesses F2.
//
// The contract: an EXACT historical replay is unchanged, even when the
// source carries a dependency graph.
func TestF2ExactReplayIsUnchangedAfterDependencyImport(t *testing.T) {
	source := SourceBoard{
		Source: "f2-replay",
		Items: []SourceItem{
			{SourceID: "W-01", EpicID: "WS-A", Title: "first", Status: SourceStatusOpen},
			{SourceID: "W-02", EpicID: "WS-A", Title: "second", Status: SourceStatusOpen,
				DependsOn: []string{"W-01"}},
		},
	}
	s := newTestStore(t)
	first := BuildPlan(source, domain.Ledger{SchemaVersion: domain.LedgerSchemaVersion})
	if _, err := Apply(s, first, "tx-f2-replay-1"); err != nil {
		t.Fatalf("apply: %v", err)
	}
	after, err := s.Load()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	plan := BuildPlan(source, after)
	for _, rec := range plan.Reconciliations {
		if rec.Outcome != ReconciliationUnchanged && rec.Outcome != ReconciliationCreated {
			t.Fatalf("the exact replay was not unchanged: %+v", rec)
		}
	}
	if len(plan.Items) != 0 {
		t.Fatalf("the exact replay produced a canonical delta: %d records", len(plan.Items))
	}
}
