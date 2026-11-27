package importer

import (
	"testing"

	"github.com/s1onique/leamas/internal/ledger/domain"
)

// This file qualifies the FWL02-C conservation discriminators against the
// REAL production seams (BuildPlan, Apply, BuildPersistedReceipt, store).
//
// C1 — the COMPLETE expected-versus-persisted comparison.
//
// The receipt must compare the WHOLE record families: the epic scopes, the
// work items, the indexed historical ACT executions and the durable
// migration reconciliation records. An independent deletion or mutation of
// each load-bearing record family must invalidate the receipt.

// conservationFixture returns a source board with an epic, a work item, a
// historical ACT and an explicit unresolved reference.
func conservationFixture() SourceBoard {
	return SourceBoard{
		Source: "fwl02-c-source",
		Items: []SourceItem{{
			SourceID:           "W-01",
			EpicID:             "EPIC-A",
			Title:              "the first obligation",
			Description:        "the obligation must survive the import",
			Status:             SourceStatusOpen,
			AcceptanceCriteria: []string{"criterion one"},
			SourceRefs: []domain.SourceRef{{
				Path: "board.md", StartLine: 1, EndLine: 2,
				SHA256: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			}},
			CriteriaCoverage:    "COMPLETE",
			DependencyCoverage:  "COMPLETE",
			Blockers:            []string{"the source states this work is blocked"},
			CurrentACT:          "ACT-CURRENT-01",
			RelatedACTs:         []string{"ACT-EXAMPLE-01"},
			EvidenceLocators:    []string{".factory/evidence/act-example-01.md"},
			ReopenConditions:    []string{"the qualification is rerun"},
			SourceLifecycle:     "OPEN",
			SourceQualification: "UNKNOWN",
		}},
		Acts: []SourceACT{{
			SourceID:         "ACT-EXAMPLE-01",
			OriginalVerdict:  "PASS_STRUCTURAL_ONLY",
			EpicID:           "EPIC-A",
			WorkItemIDs:      []string{"W-01"},
			EvidenceLocators: []string{".factory/evidence/act-example-01.md"},
			Reconciliation:   "UNAMBIGUOUS",
		}},
	}
}

// applyFixture applies the conservation fixture and returns the persisted
// state.
func applyFixture(t *testing.T, source SourceBoard) (Plan, domain.Ledger) {
	t.Helper()
	s := newTestStore(t)
	plan := BuildPlan(source, domain.Ledger{SchemaVersion: domain.LedgerSchemaVersion})
	if _, err := Apply(s, plan, "tx-fwl02-c"); err != nil {
		t.Fatalf("apply: %v", err)
	}
	persisted, err := s.Load()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	return plan, persisted
}

// TestC1IntactPersistedStateIsConserved proves the baseline: the intact
// persisted state is conserved.
func TestC1IntactPersistedStateIsConserved(t *testing.T) {
	source := conservationFixture()
	plan, persisted := applyFixture(t, source)
	receipt := BuildPersistedReceipt(
		domain.Ledger{SchemaVersion: domain.LedgerSchemaVersion}, plan, persisted)
	if !receipt.Conserved {
		t.Fatalf("the intact persisted state is reported as not conserved: %v", receipt.Discrepancies)
	}
}

// TestC1DeletedHistoricalACTInvalidatesConservation proves the receipt
// compares the historical ACT records.
//
// The contract: a persisted state that LOST an indexed historical ACT record
// is NOT conserved.
func TestC1DeletedHistoricalACTInvalidatesConservation(t *testing.T) {
	source := conservationFixture()
	plan, persisted := applyFixture(t, source)
	deleted := persisted
	deleted.Acts = nil
	receipt := BuildPersistedReceipt(
		domain.Ledger{SchemaVersion: domain.LedgerSchemaVersion}, plan, deleted)
	if receipt.Conserved {
		t.Fatal("a persisted state that lost the historical ACT is reported as conserved")
	}
}

// TestC1DeletedMigrationRecordInvalidatesConservation proves the receipt
// compares the durable migration reconciliation records.
//
// The contract: a persisted state that LOST a reconciliation record is NOT
// conserved.
func TestC1DeletedMigrationRecordInvalidatesConservation(t *testing.T) {
	source := conservationFixture()
	// The source carries an unresolved reference, so the import produces a
	// durable migration record.
	source.Items[0].DependsOn = []string{"W-MISSING"}
	plan, persisted := applyFixture(t, source)
	if len(persisted.Migrations) == 0 {
		t.Fatal("the unresolved reference produced no durable migration record")
	}
	deleted := persisted
	deleted.Migrations = nil
	receipt := BuildPersistedReceipt(
		domain.Ledger{SchemaVersion: domain.LedgerSchemaVersion}, plan, deleted)
	if receipt.Conserved {
		t.Fatal("a persisted state that lost the migration record is reported as conserved")
	}
}

// TestC1ClearedSourceFieldsInvalidateConservation proves the receipt
// compares the SOURCE-PRESERVED fields of a work item.
//
// The contract: a persisted state whose summary, source references or
// dependency coverage were cleared is NOT conserved.
func TestC1ClearedSourceFieldsInvalidateConservation(t *testing.T) {
	source := conservationFixture()
	plan, persisted := applyFixture(t, source)
	cases := []struct {
		name   string
		mutate func(item *domain.WorkItem)
	}{
		{"summary", func(item *domain.WorkItem) { item.Summary = "" }},
		{"source refs", func(item *domain.WorkItem) { item.SourceRefs = nil }},
		{"dependency coverage", func(item *domain.WorkItem) { item.DependencyCoverage = "" }},
		{"explicit blocker", func(item *domain.WorkItem) { item.SourceBlockers = nil }},
		{"current ACT", func(item *domain.WorkItem) { item.SourceCurrentACT = "" }},
		{"related ACTs", func(item *domain.WorkItem) { item.SourceRelatedACTs = nil }},
		{"evidence locators", func(item *domain.WorkItem) { item.SourceEvidenceLocators = nil }},
		{"reopen conditions", func(item *domain.WorkItem) { item.Reopen = nil }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mutated := persisted
			mutated.Items = append([]domain.WorkItem{}, persisted.Items...)
			tc.mutate(&mutated.Items[0])
			receipt := BuildPersistedReceipt(
				domain.Ledger{SchemaVersion: domain.LedgerSchemaVersion}, plan, mutated)
			if receipt.Conserved {
				t.Fatalf("a persisted state that lost the %s is reported as conserved", tc.name)
			}
		})
	}
}

// TestC1UnexpectedRecordInvalidatesConservation proves the receipt detects
// an UNEXPECTED addition.
//
// The contract: a persisted state that carries a record the import never
// planned is NOT conserved.
func TestC1UnexpectedRecordInvalidatesConservation(t *testing.T) {
	source := conservationFixture()
	plan, persisted := applyFixture(t, source)
	unexpected := persisted
	unexpected.Items = append([]domain.WorkItem{}, persisted.Items...)
	unexpected.Items = append(unexpected.Items, domain.WorkItem{
		SchemaVersion: domain.LedgerSchemaVersion,
		EpicID:        "EPIC-A",
		ItemID:        "W-UNEXPECTED",
		Title:         "an unexpected obligation",
		Kind:          domain.ItemKindTask,
		Lifecycle:     domain.LifecycleOpen,
		Acceptance:    domain.AcceptancePending,
		Revision:      1,
	})
	receipt := BuildPersistedReceipt(
		domain.Ledger{SchemaVersion: domain.LedgerSchemaVersion}, plan, unexpected)
	if receipt.Conserved {
		t.Fatal("a persisted state with an unexpected record is reported as conserved")
	}
}

// TestC1CanonicalEvolutionBeforeTheReplayIsPreserved proves the receipt
// preserves a LEGITIMATE canonical evolution.
//
// The contract: a canonical evolution that happened BEFORE the import is
// part of the pre-import state, so the expected state carries it and the
// replay must not overwrite it. An evolution that happens BETWEEN the plan
// and the persistence is an unauthorized change and is reported — the two
// cases are never conflated.
func TestC1CanonicalEvolutionBeforeTheReplayIsPreserved(t *testing.T) {
	source := conservationFixture()
	s := newTestStore(t)
	empty := domain.Ledger{SchemaVersion: domain.LedgerSchemaVersion}
	first := BuildPlan(source, empty)
	if _, err := Apply(s, first, "tx-c1-evolve-1"); err != nil {
		t.Fatalf("apply: %v", err)
	}
	imported, err := s.Load()
	if err != nil {
		t.Fatalf("load: %v", err)
	}

	// The operator evolves the imported record BEFORE the replay.
	evolved := imported
	evolved.Items = append([]domain.WorkItem{}, imported.Items...)
	evolved.Items[0].Lifecycle = domain.LifecycleInProgress
	evolved.Items[0].Acceptance = domain.AcceptancePartial
	evolved.Items[0].Revision++
	txn, err := s.BeginMutation("tx-c1-evolve-2")
	if err != nil {
		t.Fatalf("begin: %v", err)
	}
	if err := txn.PutItem(evolved.Items[0]); err != nil {
		t.Fatalf("put: %v", err)
	}
	if err := txn.Commit(); err != nil {
		t.Fatalf("commit: %v", err)
	}
	before, err := s.Load()
	if err != nil {
		t.Fatalf("load: %v", err)
	}

	// The exact historical replay must not overwrite the evolved record.
	replay := BuildPlan(source, before)
	if len(replay.Items) != 0 {
		t.Fatalf("the replay overwrote the evolved record: %d records", len(replay.Items))
	}
	if _, err := Apply(s, replay, "tx-c1-evolve-3"); err != nil {
		t.Fatalf("apply the replay: %v", err)
	}
	persisted, err := s.Load()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	receipt := BuildPersistedReceipt(before, replay, persisted)
	if !receipt.Conserved {
		t.Fatalf("a legitimate canonical evolution is reported as a discrepancy: %v",
			receipt.Discrepancies)
	}
	record, ok := persisted.ItemByScoped(evolved.Items[0].Scoped())
	if !ok {
		t.Fatal("the evolved record was lost")
	}
	if record.Lifecycle != domain.LifecycleInProgress {
		t.Fatalf("the replay overwrote the evolved lifecycle: %q", record.Lifecycle)
	}
	if record.Acceptance != domain.AcceptancePartial {
		t.Fatalf("the replay overwrote the evolved acceptance: %q", record.Acceptance)
	}
}

// TestC4ChangedEpicSourceIsReportedAndNotOverwritten witnesses the epic
// inventory of the FWL02-C contract.
//
// The contract: a CHANGED epic source is detected and reported as an
// explicit reconciliation, and the canonical scope is NOT overwritten. A
// legitimate canonical evolution of the scope does not change the source
// fingerprint and is therefore never reported.
func TestC4ChangedEpicSourceIsReportedAndNotOverwritten(t *testing.T) {
	source := SourceBoard{
		Source: "fwl02-c-epic",
		Epics: []SourceEpic{{
			EpicID:       "EPIC-A",
			Title:        "the first scope",
			Lifecycle:    "OPEN",
			ExitCriteria: []string{"criterion one"},
		}},
		Items: []SourceItem{{
			SourceID: "W-01", EpicID: "EPIC-A", Title: "first", Status: SourceStatusOpen,
		}},
	}
	s := newTestStore(t)
	first := BuildPlan(source, domain.Ledger{SchemaVersion: domain.LedgerSchemaVersion})
	if len(first.Epics) != 1 {
		t.Fatalf("planned epics = %d, want 1", len(first.Epics))
	}
	if _, err := Apply(s, first, "tx-c4-1"); err != nil {
		t.Fatalf("apply: %v", err)
	}
	after, err := s.Load()
	if err != nil {
		t.Fatalf("load: %v", err)
	}

	// The epic SOURCE changed.
	changed := source
	changed.Epics = []SourceEpic{{
		EpicID:       "EPIC-A",
		Title:        "a rewritten source scope",
		Lifecycle:    "OPEN",
		ExitCriteria: []string{"criterion one"},
	}}
	plan := BuildPlan(changed, after)
	if len(plan.Epics) != 0 {
		t.Fatalf("the changed epic source overwrote the canonical scope: %d records",
			len(plan.Epics))
	}
	reported := false
	for _, rec := range plan.Reconciliations {
		if rec.ScopedID == "EPIC-A" && rec.Outcome == ReconciliationSourceChanged {
			reported = true
		}
	}
	if !reported {
		t.Fatalf("the changed epic source was not reported: %+v", plan.Reconciliations)
	}

	// An EXACT replay is unchanged.
	replay := BuildPlan(source, after)
	for _, rec := range replay.Reconciliations {
		if rec.Outcome == ReconciliationSourceChanged {
			t.Fatalf("the exact replay reported a source change: %+v", rec)
		}
	}
}

// TestC1UnauthorizedChangeToAFreshlyImportedRecordInvalidatesConservation
// witnesses the conservation BOUNDARY.
//
// The contract: the expected state is the pre-import state plus the
// explicitly authorized plan. Every semantic field of a freshly imported
// record participates in the comparison: an unauthorized change to the
// lifecycle state, the acceptance state, the archival state, the criteria or
// the provenance is a conservation failure.
func TestC1UnauthorizedChangeToAFreshlyImportedRecordInvalidatesConservation(t *testing.T) {
	source := conservationFixture()
	plan, persisted := applyFixture(t, source)
	cases := []struct {
		name   string
		mutate func(item *domain.WorkItem)
	}{
		{"lifecycle", func(item *domain.WorkItem) { item.Lifecycle = domain.LifecycleClosed }},
		{"acceptance", func(item *domain.WorkItem) { item.Acceptance = domain.AcceptanceDeferred }},
		{"archived", func(item *domain.WorkItem) { item.Archived = true }},
		{"extra criterion", func(item *domain.WorkItem) {
			item.AcceptanceCriteria = append(item.AcceptanceCriteria, "an invented criterion")
		}},
		{"provenance", func(item *domain.WorkItem) {
			if item.Provenance == nil {
				item.Provenance = &domain.Provenance{}
			}
			item.Provenance.Confidence = string(domain.ProvenanceRepository)
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mutated := persisted
			mutated.Items = append([]domain.WorkItem{}, persisted.Items...)
			tc.mutate(&mutated.Items[0])
			receipt := BuildPersistedReceipt(
				domain.Ledger{SchemaVersion: domain.LedgerSchemaVersion}, plan, mutated)
			if receipt.Conserved {
				t.Fatalf("an unauthorized change of the %s is reported as conserved", tc.name)
			}
		})
	}
}

// TestC1UnauthorizedChangeToAPreExistingRecordInvalidatesConservation proves
// the receipt detects an UNAUTHORIZED change to a pre-existing record.
//
// The contract: a record that existed before the import and that the import
// did not plan must be preserved verbatim.
func TestC1UnauthorizedChangeToAPreExistingRecordInvalidatesConservation(t *testing.T) {
	source := conservationFixture()
	s := newTestStore(t)
	before := domain.Ledger{
		SchemaVersion: domain.LedgerSchemaVersion,
		Items: []domain.WorkItem{{
			SchemaVersion: domain.LedgerSchemaVersion,
			EpicID:        "EPIC-A",
			ItemID:        "W-LEGACY",
			Title:         "a pre-existing obligation",
			Kind:          domain.ItemKindTask,
			Lifecycle:     domain.LifecycleOpen,
			Acceptance:    domain.AcceptancePending,
			Revision:      1,
		}},
	}
	if _, err := Apply(s, Plan{Items: before.Items}, "tx-legacy"); err != nil {
		t.Fatalf("apply the legacy record: %v", err)
	}
	plan := BuildPlan(source, before)
	if _, err := Apply(s, plan, "tx-fwl02-c-2"); err != nil {
		t.Fatalf("apply: %v", err)
	}
	persisted, err := s.Load()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	mutated := persisted
	mutated.Items = append([]domain.WorkItem{}, persisted.Items...)
	for i := range mutated.Items {
		if mutated.Items[i].ItemID == "W-LEGACY" {
			mutated.Items[i].Title = "a rewritten pre-existing obligation"
		}
	}
	receipt := BuildPersistedReceipt(before, plan, mutated)
	if receipt.Conserved {
		t.Fatal("an unauthorized change to a pre-existing record is reported as conserved")
	}
}
