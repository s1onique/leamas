package importer

import (
	"path/filepath"
	"testing"

	"github.com/s1onique/leamas/internal/ledger/domain"
	"github.com/s1onique/leamas/internal/ledger/store"
)

// This file records the RED reproduction of the import-graph and provenance
// conservation counterexamples against the real BuildPlan/Apply path.

// TestAmendmentResolvedDependencyBecomesTypedEdge reproduces counterexample
// 3: loss of a resolved imported dependency.
//
// The contract: every dependency that resolves against the imported identity
// set becomes a canonical typed edge on the imported work item.
func TestAmendmentResolvedDependencyBecomesTypedEdge(t *testing.T) {
	source := SourceBoard{
		Source: "src",
		Items: []SourceItem{
			{SourceID: "W-01", EpicID: "WS-A", Title: "first", Status: SourceStatusOpen},
			{SourceID: "W-02", EpicID: "WS-A", Title: "second", Status: SourceStatusOpen,
				DependsOn: []string{"W-01"}},
		},
	}

	p := BuildPlan(source, domain.Ledger{SchemaVersion: domain.LedgerSchemaVersion})

	var target domain.WorkItem
	for _, it := range p.Items {
		if it.ItemID == "W-02" {
			target = it
		}
	}
	if target.ItemID == "" {
		t.Fatal("W-02 was not imported")
	}
	if len(target.Dependencies) == 0 {
		t.Fatal("resolved dependency edge was lost: want one typed edge")
	}
	dep := target.Dependencies[0]
	if dep.Kind != domain.DependencyRequires {
		t.Fatalf("expected a blocking typed edge, got kind %q", dep.Kind)
	}
	if dep.Ref != "WS-A/W-01" {
		t.Fatalf("expected dependency on WS-A/W-01, got %q", dep.Ref.String())
	}
}

// TestAmendmentUnresolvedDependencyBlocksImport reproduces counterexample 4:
// an unresolved dependency becoming executable.
//
// The contract: an unresolved blocking dependency is a durable blocker. It
// must never become an executable work item.
func TestAmendmentUnresolvedDependencyBlocksImport(t *testing.T) {
	source := SourceBoard{
		Source: "src",
		Items: []SourceItem{
			{SourceID: "W-01", EpicID: "WS-A", Title: "first", Status: SourceStatusOpen,
				DependsOn: []string{"W-MISSING"}},
		},
	}

	p := BuildPlan(source, domain.Ledger{SchemaVersion: domain.LedgerSchemaVersion})

	if len(p.Unresolved) == 0 {
		t.Fatal("unresolved dependency was not reported")
	}
	if len(p.Items) == 0 {
		t.Fatal("W-01 was not imported at all")
	}
	// The imported item must carry a durable blocker, not an executable
	// dependency that silently resolves to nothing.
	item := p.Items[0]
	hasBlocker := false
	for _, dep := range item.Dependencies {
		if dep.Ref == "WS-A/W-MISSING" {
			hasBlocker = true
		}
	}
	if !hasBlocker {
		t.Fatal("unresolved dependency did not become a durable blocker")
	}
}

// TestAmendmentConflictingSourceRecordsAreQuarantined reproduces
// counterexample 5: first-winner selection of conflicting historical
// records.
//
// The contract: conflicting source records are quarantined as a complete
// identity group. Input order cannot select a winner.
func TestAmendmentConflictingSourceRecordsAreQuarantined(t *testing.T) {
	source := SourceBoard{
		Source: "src",
		Items: []SourceItem{
			{SourceID: "W-01", EpicID: "WS-A", Title: "first title", Status: SourceStatusOpen},
			{SourceID: "W-01", EpicID: "WS-A", Title: "second title", Status: SourceStatusClosed},
		},
	}

	p := BuildPlan(source, domain.Ledger{SchemaVersion: domain.LedgerSchemaVersion})

	if len(p.Items) != 0 {
		t.Fatalf("conflicting records acquired authority: %d items imported", len(p.Items))
	}
	if len(p.Ambiguities) == 0 {
		t.Fatal("conflicting source records were not quarantined as an ambiguity")
	}
	if len(p.Quarantined) == 0 {
		t.Fatal("no quarantine record was produced for the conflicting identity group")
	}
}

// TestAmendmentIdenticalReplayProducesNoDelta reproduces the W3
// idempotence requirement.
func TestAmendmentIdenticalReplayProducesNoDelta(t *testing.T) {
	source := SourceBoard{
		Source: "src",
		Items: []SourceItem{
			{SourceID: "W-01", EpicID: "WS-A", Title: "first", Status: SourceStatusOpen},
		},
	}
	existing := domain.Ledger{SchemaVersion: domain.LedgerSchemaVersion}

	p1 := BuildPlan(source, existing)
	if len(p1.Items) != 1 {
		t.Fatalf("expected 1 planned item, got %d", len(p1.Items))
	}

	// Replay against the state that already contains the imported item.
	after := existing
	after.Items = append(after.Items, p1.Items[0])
	p2 := BuildPlan(source, after)
	if len(p2.Items) != 0 {
		t.Fatalf("identical replay produced a canonical delta: %d items", len(p2.Items))
	}
	if len(p2.Skipped) != 1 {
		t.Fatalf("expected 1 skipped record, got %d", len(p2.Skipped))
	}
}

// TestAmendmentUnresolvedDependencyIsNotExecutable witnesses the W3
// requirement that an unresolved dependency stays non-executable.
//
// The imported item carries a durable blocker edge. The readiness
// projection must refuse to select it.
func TestAmendmentUnresolvedDependencyIsNotExecutable(t *testing.T) {
	source := SourceBoard{
		Source: "src",
		Items: []SourceItem{
			{SourceID: "W-01", EpicID: "WS-A", Title: "first", Status: SourceStatusOpen,
				DependsOn: []string{"W-MISSING"}},
		},
	}

	p := BuildPlan(source, domain.Ledger{SchemaVersion: domain.LedgerSchemaVersion})
	if len(p.Items) != 1 {
		t.Fatalf("expected 1 imported item, got %d", len(p.Items))
	}

	l := domain.Ledger{
		SchemaVersion: domain.LedgerSchemaVersion,
		Items:         p.Items,
	}
	ready := domain.SelectReady(l, domain.ReadyOptions{})
	if len(ready) != 0 {
		t.Fatalf("an item with an unresolved dependency became executable: %d ready items", len(ready))
	}
}

// TestAmendmentScopedIdentitiesRemainDistinct witnesses the W3
// requirement that scoped identities remain distinct across epics.
func TestAmendmentScopedIdentitiesRemainDistinct(t *testing.T) {
	source := SourceBoard{
		Source: "src",
		Items: []SourceItem{
			{SourceID: "W-01", EpicID: "WS-A", Title: "alpha", Status: SourceStatusOpen},
			{SourceID: "W-01", EpicID: "WS-B", Title: "bravo", Status: SourceStatusOpen},
		},
	}

	p := BuildPlan(source, domain.Ledger{SchemaVersion: domain.LedgerSchemaVersion})
	if len(p.Items) != 2 {
		t.Fatalf("expected 2 scoped records, got %d", len(p.Items))
	}
	seen := map[string]bool{}
	for _, it := range p.Items {
		id := it.Scoped().String()
		if seen[id] {
			t.Fatalf("scoped identity collapsed: %s", id)
		}
		seen[id] = true
	}
	if !seen["WS-A/W-01"] || !seen["WS-B/W-01"] {
		t.Fatalf("scoped identities are not distinct: %+v", seen)
	}
}

// TestAmendmentExistingEpicMetadataIsNotOverwritten witnesses the W3
// requirement that existing epic metadata is not silently overwritten.
func TestAmendmentExistingEpicMetadataIsNotOverwritten(t *testing.T) {
	source := SourceBoard{
		Source:     "src",
		EpicTitles: map[string]string{"WS-A": "imported title"},
		Items: []SourceItem{
			{SourceID: "W-01", EpicID: "WS-A", Title: "first", Status: SourceStatusOpen},
		},
	}
	existing := domain.Ledger{
		SchemaVersion: domain.LedgerSchemaVersion,
		Epics: []domain.Epic{{
			SchemaVersion: domain.LedgerSchemaVersion,
			EpicID:        "WS-A",
			Title:         "canonical title",
			Revision:      7,
		}},
	}

	p := BuildPlan(source, existing)
	for _, e := range p.Epics {
		if e.EpicID == "WS-A" {
			t.Fatal("the import planned to overwrite the existing epic metadata")
		}
	}
}

// TestAmendmentConservationReceiptReportsSemantics witnesses the W3
// conservation receipt requirement.
//
// The receipt reports the conserved SEMANTICS, not only the imported
// record count.
func TestAmendmentConservationReceiptReportsSemantics(t *testing.T) {
	source := SourceBoard{
		Source: "src",
		Items: []SourceItem{
			{SourceID: "W-01", EpicID: "WS-A", Title: "first", Status: SourceStatusClosed,
				AcceptanceCriteria: []string{"criterion one"}},
			{SourceID: "W-02", EpicID: "WS-A", Title: "second", Status: SourceStatusOpen,
				DependsOn: []string{"W-01"}},
			{SourceID: "W-03", EpicID: "WS-A", Title: "third", Status: SourceStatusOpen,
				DependsOn: []string{"W-MISSING"}},
		},
	}
	before := domain.Ledger{SchemaVersion: domain.LedgerSchemaVersion}

	p := BuildPlan(source, before)
	r := BuildReceipt(before, p)

	if r.Nodes != 3 {
		t.Fatalf("expected 3 nodes, got %d", r.Nodes)
	}
	if r.Relationships != 1 {
		t.Fatalf("expected 1 conserved relationship, got %d", r.Relationships)
	}
	if r.Blockers != 1 {
		t.Fatalf("expected 1 durable blocker, got %d", r.Blockers)
	}
	if r.OriginalDispositions != 3 {
		t.Fatalf("expected 3 conserved dispositions, got %d", r.OriginalDispositions)
	}
	if r.Criteria != 1 {
		t.Fatalf("expected 1 conserved criterion, got %d", r.Criteria)
	}
	if len(r.Reconciliations) != 3 {
		t.Fatalf("expected 3 reconciliation records, got %d", len(r.Reconciliations))
	}
	for _, rec := range r.Reconciliations {
		if rec.Outcome != ReconciliationCreated {
			t.Fatalf("expected a 'created' reconciliation, got %q", rec.Outcome)
		}
	}
}

// TestAmendmentImportFailsClosedOnQuarantinedGroup witnesses the W3
// requirement that the import apply fails closed on unresolved load-bearing
// ambiguity.
func TestAmendmentImportFailsClosedOnQuarantinedGroup(t *testing.T) {
	source := SourceBoard{
		Source: "src",
		Items: []SourceItem{
			{SourceID: "W-01", EpicID: "WS-A", Title: "one", Status: SourceStatusOpen},
			{SourceID: "W-01", EpicID: "WS-A", Title: "two", Status: SourceStatusClosed},
		},
	}
	p := BuildPlan(source, domain.Ledger{SchemaVersion: domain.LedgerSchemaVersion})
	if len(p.Quarantined) == 0 {
		t.Fatal("expected the conflicting group to be quarantined")
	}
	st := newTestStore(t)
	if _, err := Apply(st, p, "tx-import"); err == nil {
		t.Fatal("the import applied a quarantined identity group: want rejection")
	}
}

// newTestStore returns an initialized store in a temporary directory.
func newTestStore(t *testing.T) store.Store {
	t.Helper()
	s := store.New(filepath.Join(t.TempDir(), "ledger"))
	if err := s.Init(); err != nil {
		t.Fatalf("init: %v", err)
	}
	return s
}
