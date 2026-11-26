package importer

import (
	"testing"

	"github.com/s1onique/leamas/internal/ledger/domain"
)

// This file qualifies the FWL01 amendment import-conservation witnesses.
//
// Each witness drives the REAL BuildPlan path and asserts the REQUIRED
// behaviour.

// TestWitnessUnknownStatusStaysNonExecutable proves that an unknown source
// status remains durably non-executable.
func TestWitnessUnknownStatusStaysNonExecutable(t *testing.T) {
	src := SourceBoard{
		Source: "src",
		Items: []SourceItem{{
			SourceID: "W-01", EpicID: "WS-A", Title: "unknown status",
			Status: SourceStatus("TOTALLY_UNKNOWN"),
		}},
	}
	p := BuildPlan(src, domain.Ledger{SchemaVersion: domain.LedgerSchemaVersion})
	if len(p.Items) != 1 {
		t.Fatalf("expected 1 imported record, got %d", len(p.Items))
	}
	item := p.Items[0]
	if item.SourceStatus != "TOTALLY_UNKNOWN" {
		t.Fatalf("the unknown source status was not conserved verbatim: %q", item.SourceStatus)
	}
	if item.Acceptance != domain.AcceptanceDeferred {
		t.Fatalf("the unknown-status record is executable: %q", item.Acceptance)
	}
	// The unresolved status is reported.
	found := false
	for _, u := range p.Unresolved {
		if u.Kind == "unknown_source_status" {
			found = true
		}
	}
	if !found {
		t.Fatalf("the unknown source status was not reported: %+v", p.Unresolved)
	}
	// The readiness projection refuses it.
	l := domain.Ledger{SchemaVersion: domain.LedgerSchemaVersion, Items: p.Items}
	if ready := domain.SelectReady(l, domain.ReadyOptions{}); len(ready) != 0 {
		t.Fatalf("the unknown-status record became executable: %d ready items", len(ready))
	}
}

// TestWitnessChangedSourceProducesExplicitReconciliation proves that a
// changed historical source is reported as an explicit reconciliation and
// never overwrites the canonical work.
func TestWitnessChangedSourceProducesExplicitReconciliation(t *testing.T) {
	src := SourceBoard{
		Source: "src",
		Items: []SourceItem{{
			SourceID: "W-01", EpicID: "WS-A", Title: "original title",
			Status: SourceStatusOpen, Priority: "P1", SourceRow: "W-01\toriginal title",
		}},
	}
	existing := domain.Ledger{
		SchemaVersion: domain.LedgerSchemaVersion,
		Items: []domain.WorkItem{{
			SchemaVersion: domain.LedgerSchemaVersion,
			EpicID:        "WS-A",
			ItemID:        "W-01",
			Title:         "original title",
			Kind:          domain.ItemKindTask,
			Lifecycle:     domain.LifecycleOpen,
			Acceptance:    domain.AcceptancePending,
			Priority:      domain.PriorityP1,
			SourceStatus:  string(SourceStatusOpen),
			SourceRow:     "W-01\toriginal title",
			Revision:      1,
		}},
	}

	// An EXACT replay produces no canonical delta.
	exact := BuildPlan(src, existing)
	if len(exact.Items) != 0 {
		t.Fatalf("the exact replay produced %d records", len(exact.Items))
	}
	if !hasOutcome(exact, "WS-A/W-01", ReconciliationUnchanged) {
		t.Fatalf("the exact replay was not reported as unchanged: %+v", exact.Reconciliations)
	}

	// A CHANGED replay is an explicit reconciliation.
	src.Items[0].Title = "CHANGED title"
	changed := BuildPlan(src, existing)
	if len(changed.Items) != 0 {
		t.Fatalf("the changed source overwrote the canonical work: %d records", len(changed.Items))
	}
	if !hasOutcome(changed, "WS-A/W-01", ReconciliationSourceChanged) {
		t.Fatalf("the changed source was not reported: %+v", changed.Reconciliations)
	}
}

// TestWitnessEverySourceIdentityIsAccountedFor proves that every source
// identity receives exactly one reconciliation record.
func TestWitnessEverySourceIdentityIsAccountedFor(t *testing.T) {
	src := SourceBoard{
		Source: "src",
		Items: []SourceItem{
			{SourceID: "W-01", EpicID: "WS-A", Title: "one", Status: SourceStatusOpen},
			{SourceID: "W-02", EpicID: "WS-A", Title: "two", Status: SourceStatusClosed},
			{SourceID: "W-03", EpicID: "WS-B", Title: "three", Status: SourceStatusBlocked},
		},
	}
	p := BuildPlan(src, domain.Ledger{SchemaVersion: domain.LedgerSchemaVersion})
	seen := map[string]int{}
	for _, rec := range p.Reconciliations {
		seen[rec.ScopedID]++
	}
	for _, item := range src.Items {
		key := item.EpicID + "/" + item.SourceID
		if seen[key] != 1 {
			t.Fatalf("the source identity %q has %d reconciliation records", key, seen[key])
		}
	}
	if len(p.Reconciliations) != len(src.Items) {
		t.Fatalf("expected %d reconciliation records, got %d", len(src.Items), len(p.Reconciliations))
	}
}

// TestWitnessConflictingSourceIdentitiesAreQuarantined proves that
// conflicting source identities never acquire executable authority.
func TestWitnessConflictingSourceIdentitiesAreQuarantined(t *testing.T) {
	src := SourceBoard{
		Source: "src",
		Items: []SourceItem{
			{SourceID: "W-01", EpicID: "WS-A", Title: "one", Status: SourceStatusOpen},
			{SourceID: "W-01", EpicID: "WS-A", Title: "two", Status: SourceStatusClosed},
		},
	}
	p := BuildPlan(src, domain.Ledger{SchemaVersion: domain.LedgerSchemaVersion})
	if len(p.Items) != 0 {
		t.Fatalf("the conflicting identity acquired authority: %d records", len(p.Items))
	}
	if len(p.Quarantined) != 1 {
		t.Fatalf("expected 1 quarantined group, got %d", len(p.Quarantined))
	}
	if !hasOutcome(p, "WS-A/W-01", ReconciliationConflict) {
		t.Fatalf("the conflict was not reported: %+v", p.Reconciliations)
	}
}

// TestWitnessMissingSourceIdentityIsReported proves that a source record
// without an identity is reported and never acquires authority.
func TestWitnessMissingSourceIdentityIsReported(t *testing.T) {
	src := SourceBoard{
		Source: "src",
		Items: []SourceItem{{
			SourceID: "", EpicID: "WS-A", Title: "no identity", Status: SourceStatusOpen,
		}},
	}
	p := BuildPlan(src, domain.Ledger{SchemaVersion: domain.LedgerSchemaVersion})
	if len(p.Items) != 0 {
		t.Fatalf("the record without an identity acquired authority: %d records", len(p.Items))
	}
	found := false
	for _, a := range p.Ambiguities {
		if a.Kind == "missing_source_id" {
			found = true
		}
	}
	if !found {
		t.Fatalf("the missing source identity was not reported: %+v", p.Ambiguities)
	}
}

// TestWitnessUnresolvedDependencyStaysNonExecutable proves that an
// unresolved dependency keeps the importing item non-executable.
func TestWitnessUnresolvedDependencyStaysNonExecutable(t *testing.T) {
	src := SourceBoard{
		Source: "src",
		Items: []SourceItem{{
			SourceID: "W-01", EpicID: "WS-A", Title: "one", Status: SourceStatusOpen,
			DependsOn: []string{"W-MISSING"},
		}},
	}
	p := BuildPlan(src, domain.Ledger{SchemaVersion: domain.LedgerSchemaVersion})
	if len(p.Items) != 1 {
		t.Fatalf("expected 1 imported record, got %d", len(p.Items))
	}
	item := p.Items[0]
	hasBlocker := false
	for _, dep := range item.Dependencies {
		if dep.Kind == domain.DependencyBlocker {
			hasBlocker = true
		}
	}
	if !hasBlocker {
		t.Fatal("the unresolved dependency did not become a durable blocker")
	}
	l := domain.Ledger{SchemaVersion: domain.LedgerSchemaVersion, Items: p.Items}
	if ready := domain.SelectReady(l, domain.ReadyOptions{}); len(ready) != 0 {
		t.Fatalf("the blocked record became executable: %d ready items", len(ready))
	}
}

// TestWitnessSourceProvenanceIsPreserved proves that the original row text,
// the source locator, the status, the priority and the workstream are
// preserved.
func TestWitnessSourceProvenanceIsPreserved(t *testing.T) {
	src := SourceBoard{
		Source: "src",
		Items: []SourceItem{{
			SourceID: "W-01", EpicID: "WS-A", Title: "one", Status: SourceStatusClosed,
			Priority: "P0", Owner: "owner-a",
			SourceRow: "W-01\tone CLOSED", SourceLine: 57, SourceDetail: "detail",
		}},
	}
	p := BuildPlan(src, domain.Ledger{SchemaVersion: domain.LedgerSchemaVersion})
	item := p.Items[0]
	if item.SourceRow != "W-01\tone CLOSED" {
		t.Fatalf("the original row text was not preserved: %q", item.SourceRow)
	}
	if item.SourceLine != 57 {
		t.Fatalf("the source locator was not preserved: %d", item.SourceLine)
	}
	if item.SourceDetail != "detail" {
		t.Fatalf("the source detail was not preserved: %q", item.SourceDetail)
	}
	if item.SourceStatus != string(SourceStatusClosed) {
		t.Fatalf("the source status was not preserved: %q", item.SourceStatus)
	}
	if item.Priority != domain.PriorityP0 {
		t.Fatalf("the source priority was not preserved: %q", item.Priority)
	}
	if item.EpicID != "WS-A" {
		t.Fatalf("the workstream was not preserved: %q", item.EpicID)
	}
	if item.Owner != "owner-a" {
		t.Fatalf("the owner was not preserved: %q", item.Owner)
	}
	if item.Provenance == nil || item.Provenance.SourceID != "W-01" {
		t.Fatalf("the source identity was not preserved: %+v", item.Provenance)
	}
}

// hasOutcome reports whether the plan carries a reconciliation with the
// given scoped identity and outcome.
func hasOutcome(p Plan, scopedID, outcome string) bool {
	for _, rec := range p.Reconciliations {
		if rec.ScopedID == scopedID && rec.Outcome == outcome {
			return true
		}
	}
	return false
}
