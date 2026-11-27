package fwmi

import (
	"strings"
	"testing"

	"github.com/s1onique/leamas/internal/ledger/domain"
	"github.com/s1onique/leamas/internal/ledger/importer"
)

// TestAdaptPreservesSourceFacts proves the adaptation conserves the
// load-bearing facts of the interchange package.
func TestAdaptPreservesSourceFacts(t *testing.T) {
	pkg, err := Load(writePackage(t, validPackage()))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	adapted, err := Adapt(pkg)
	if err != nil {
		t.Fatalf("Adapt: %v", err)
	}
	if adapted.Stats.Items != 1 {
		t.Fatalf("items = %d, want 1", adapted.Stats.Items)
	}
	if adapted.Stats.HistoricalActs != 1 {
		t.Fatalf("historical acts = %d, want 1", adapted.Stats.HistoricalActs)
	}
	if adapted.Stats.ReconciliationRecords != 1 {
		t.Fatalf("reconciliation records = %d, want 1", adapted.Stats.ReconciliationRecords)
	}
	item := adapted.Board.Items[0]
	if item.SourceLifecycle != LifecycleOpen {
		t.Errorf("source lifecycle = %q", item.SourceLifecycle)
	}
	if item.SourceQualification != QualificationUnknown {
		t.Errorf("source qualification = %q", item.SourceQualification)
	}
	if item.CriteriaCoverage != CoverageUnknown {
		t.Errorf("criteria coverage = %q", item.CriteriaCoverage)
	}
	if len(item.SourceRefs) != 1 || item.SourceRefs[0].StartLine != 120 {
		t.Errorf("source refs = %+v", item.SourceRefs)
	}
	if item.SourceRow != "OPEN" {
		t.Errorf("original status = %q, want the verbatim source status", item.SourceRow)
	}
	if len(adapted.Board.Acts) != 1 {
		t.Fatalf("acts = %d, want 1", len(adapted.Board.Acts))
	}
	if adapted.Board.Acts[0].OriginalVerdict != "PASS_STRUCTURAL_ONLY" {
		t.Errorf("original verdict = %q", adapted.Board.Acts[0].OriginalVerdict)
	}
}

// TestAdaptKeepsUnresolvedDistinctFromDeferred proves an UNRESOLVED source
// lifecycle never becomes an intentional deferral.
func TestAdaptKeepsUnresolvedDistinctFromDeferred(t *testing.T) {
	files := validPackage()
	files["epics/EPIC-ARCHITECTURE.json"] = strings.Replace(validEpic,
		`"original_status": "OPEN",
      "lifecycle": "OPEN"`, `"lifecycle": "UNRESOLVED"`, 1)
	pkg, err := Load(writePackage(t, files))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	adapted, err := Adapt(pkg)
	if err != nil {
		t.Fatalf("Adapt: %v", err)
	}
	if adapted.Board.Items[0].Status == importer.SourceStatusDeferred {
		t.Fatal("an unresolved lifecycle must not become a deferral")
	}
	if importer.IsKnownSourceStatus(adapted.Board.Items[0].Status) {
		t.Fatalf("status %q must stay unknown", adapted.Board.Items[0].Status)
	}
}

// TestAdaptSeparatesBlockingDependenciesFromCrossLinks proves only a
// DEPENDS_ON relationship becomes an execution prerequisite.
func TestAdaptSeparatesBlockingDependenciesFromCrossLinks(t *testing.T) {
	files := validPackage()
	files["epics/EPIC-ARCHITECTURE.json"] = strings.Replace(validEpic,
		`"dependencies": []`,
		`"dependencies": [
          {"kind": "DEPENDS_ON", "target": "EPIC-ARCHITECTURE/E9", "source_refs": []},
          {"kind": "CROSS_LINK", "target": "EPIC-OTHER/X", "source_refs": []}
        ]`, 1)
	pkg, err := Load(writePackage(t, files))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	adapted, err := Adapt(pkg)
	if err != nil {
		t.Fatalf("Adapt: %v", err)
	}
	item := adapted.Board.Items[0]
	if len(item.DependsOn) != 1 || item.DependsOn[0] != "EPIC-ARCHITECTURE/E9" {
		t.Errorf("depends_on = %v", item.DependsOn)
	}
	if len(item.Relations) != 1 || item.Relations[0].Kind != RelCrossLink {
		t.Errorf("relations = %+v", item.Relations)
	}
}

// TestAdaptedPlanKeepsUnresolvedItemNonExecutable proves an unresolved
// source status keeps its record durably non-executable and explicitly
// reported.
func TestAdaptedPlanKeepsUnresolvedItemNonExecutable(t *testing.T) {
	files := validPackage()
	files["epics/EPIC-ARCHITECTURE.json"] = strings.Replace(validEpic,
		`"original_status": "OPEN",
      "lifecycle": "OPEN"`, `"lifecycle": "UNRESOLVED"`, 1)
	pkg, err := Load(writePackage(t, files))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	adapted, err := Adapt(pkg)
	if err != nil {
		t.Fatalf("Adapt: %v", err)
	}
	plan := importer.BuildPlan(adapted.Board, domain.Ledger{})
	if len(plan.Items) != 1 {
		t.Fatalf("planned items = %d, want 1", len(plan.Items))
	}
	item := plan.Items[0]
	if item.Acceptance != domain.AcceptanceDeferred {
		t.Errorf("acceptance = %q, want a non-executable state", item.Acceptance)
	}
	found := false
	for _, u := range plan.Unresolved {
		if u.Kind == "unknown_source_status" {
			found = true
		}
	}
	if !found {
		t.Errorf("unresolved = %+v, want an explicit unknown source status report", plan.Unresolved)
	}
}
