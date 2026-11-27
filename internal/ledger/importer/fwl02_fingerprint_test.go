package importer

import (
	"testing"

	"github.com/s1onique/leamas/internal/ledger/domain"
)

// TestF2CanonicalEvolutionDoesNotChangeTheSourceFingerprint witnesses F2.
//
// The contract: a legitimate CANONICAL EVOLUTION does not change the
// immutable source fingerprint. The unchanged historical source therefore
// still replays as unchanged.
func TestF2CanonicalEvolutionDoesNotChangeTheSourceFingerprint(t *testing.T) {
	source := SourceBoard{
		Source: "f2-evolution",
		Items: []SourceItem{
			{SourceID: "W-01", EpicID: "WS-A", Title: "first", Status: SourceStatusOpen},
		},
	}
	s := newTestStore(t)
	first := BuildPlan(source, domain.Ledger{SchemaVersion: domain.LedgerSchemaVersion})
	if _, err := Apply(s, first, "tx-f2-evo-1"); err != nil {
		t.Fatalf("apply: %v", err)
	}
	after, err := s.Load()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	prior := after.Items[0]
	fingerprint := prior.SourceFingerprint
	if fingerprint == "" {
		t.Fatal("the imported record carries no source fingerprint")
	}

	txn, err := s.BeginMutation("tx-f2-evo-2")
	if err != nil {
		t.Fatalf("begin: %v", err)
	}
	evolved := prior
	evolved.Title = "evolved canonical title"
	evolved.Lifecycle = domain.LifecycleInProgress
	evolved.Revision++
	if err := txn.PutItem(evolved); err != nil {
		t.Fatalf("put: %v", err)
	}
	if err := txn.Commit(); err != nil {
		t.Fatalf("commit: %v", err)
	}
	afterEvolution, err := s.Load()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if afterEvolution.Items[0].SourceFingerprint != fingerprint {
		t.Fatal("the canonical evolution rewrote the source fingerprint")
	}

	plan := BuildPlan(source, afterEvolution)
	if len(plan.Items) != 0 {
		t.Fatalf("the unchanged historical source overwrote the evolved canonical work: %d records",
			len(plan.Items))
	}
	for _, rec := range plan.Reconciliations {
		if rec.ScopedID == "WS-A/W-01" && rec.Outcome != ReconciliationUnchanged {
			t.Fatalf("the unchanged historical source was reported as %q", rec.Outcome)
		}
	}
}

// TestF2ChangedSourceDoesNotOverwriteEvolvedCanonicalWork witnesses F2.
//
// The contract: a changed source record must NOT silently overwrite the
// evolved canonical work.
func TestF2ChangedSourceDoesNotOverwriteEvolvedCanonicalWork(t *testing.T) {
	source := SourceBoard{
		Source: "f2-overwrite",
		Items: []SourceItem{
			{SourceID: "W-01", EpicID: "WS-A", Title: "first", Status: SourceStatusOpen},
		},
	}
	s := newTestStore(t)
	first := BuildPlan(source, domain.Ledger{SchemaVersion: domain.LedgerSchemaVersion})
	if _, err := Apply(s, first, "tx-f2-ow-1"); err != nil {
		t.Fatalf("apply: %v", err)
	}
	after, err := s.Load()
	if err != nil {
		t.Fatalf("load: %v", err)
	}

	changed := SourceBoard{
		Source: "f2-overwrite",
		Items: []SourceItem{
			{SourceID: "W-01", EpicID: "WS-A", Title: "CHANGED", Status: SourceStatusOpen},
		},
	}
	plan := BuildPlan(changed, after)
	if len(plan.Items) != 0 {
		t.Fatalf("the changed source overwrote the canonical work: %d records", len(plan.Items))
	}
	final, err := s.Load()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if final.Items[0].Title != "first" {
		t.Fatalf("the canonical title was overwritten: %q", final.Items[0].Title)
	}
}

// TestF2DependencyEdgeIsPersisted witnesses F2.
//
// The contract: a resolved source dependency becomes a canonical typed
// edge that survives persistence and reload.
func TestF2DependencyEdgeIsPersisted(t *testing.T) {
	source := SourceBoard{
		Source: "f2-edge",
		Items: []SourceItem{
			{SourceID: "W-01", EpicID: "WS-A", Title: "first", Status: SourceStatusOpen},
			{SourceID: "W-02", EpicID: "WS-A", Title: "second", Status: SourceStatusOpen,
				DependsOn: []string{"W-01"}},
		},
	}
	s := newTestStore(t)
	plan := BuildPlan(source, domain.Ledger{SchemaVersion: domain.LedgerSchemaVersion})
	if _, err := Apply(s, plan, "tx-f2-edge"); err != nil {
		t.Fatalf("apply: %v", err)
	}
	reloaded, err := s.Load()
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	item, ok := reloaded.ItemByScoped("WS-A/W-02")
	if !ok {
		t.Fatal("the dependent record was not persisted")
	}
	if len(item.Dependencies) != 1 {
		t.Fatalf("the dependency edge was not persisted: %d edges", len(item.Dependencies))
	}
	if item.Dependencies[0].Ref != "WS-A/W-01" {
		t.Fatalf("unexpected dependency edge: %q", item.Dependencies[0].Ref.String())
	}
}

// TestF2ReplayedDependencyEdgeIsNotDuplicated witnesses F2.
//
// The contract: an exact replay of a source with a dependency graph
// produces no canonical delta: no duplicated edge.
func TestF2ReplayedDependencyEdgeIsNotDuplicated(t *testing.T) {
	source := SourceBoard{
		Source: "f2-dup",
		Items: []SourceItem{
			{SourceID: "W-01", EpicID: "WS-A", Title: "first", Status: SourceStatusOpen},
			{SourceID: "W-02", EpicID: "WS-A", Title: "second", Status: SourceStatusOpen,
				DependsOn: []string{"W-01"}},
		},
	}
	s := newTestStore(t)
	first := BuildPlan(source, domain.Ledger{SchemaVersion: domain.LedgerSchemaVersion})
	if _, err := Apply(s, first, "tx-f2-dup-1"); err != nil {
		t.Fatalf("apply: %v", err)
	}
	after, err := s.Load()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	plan := BuildPlan(source, after)
	if len(plan.Items) != 0 {
		t.Fatalf("the replay produced a canonical delta: %d records", len(plan.Items))
	}
	if _, err := Apply(s, plan, "tx-f2-dup-2"); err != nil {
		t.Fatalf("apply the replay: %v", err)
	}
	final, err := s.Load()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	item, _ := final.ItemByScoped("WS-A/W-02")
	if len(item.Dependencies) != 1 {
		t.Fatalf("the replay duplicated the dependency edge: %d edges", len(item.Dependencies))
	}
}

// TestF2ReceiptCountsTheConservedRelationship witnesses F2.
//
// The contract: the conservation receipt reports the conserved
// relationship, not only the record count.
func TestF2ReceiptCountsTheConservedRelationship(t *testing.T) {
	source := SourceBoard{
		Source: "f2-receipt",
		Items: []SourceItem{
			{SourceID: "W-01", EpicID: "WS-A", Title: "first", Status: SourceStatusOpen},
			{SourceID: "W-02", EpicID: "WS-A", Title: "second", Status: SourceStatusOpen,
				DependsOn: []string{"W-01"}},
		},
	}
	plan := BuildPlan(source, domain.Ledger{SchemaVersion: domain.LedgerSchemaVersion})
	receipt := BuildReceipt(domain.Ledger{SchemaVersion: domain.LedgerSchemaVersion}, plan)
	if receipt.Relationships != 1 {
		t.Fatalf("the receipt did not count the conserved relationship: %d", receipt.Relationships)
	}
	if receipt.Nodes != 2 {
		t.Fatalf("the receipt did not count the conserved nodes: %d", receipt.Nodes)
	}
}

// TestF2StoreReplayProducesNoByteDelta witnesses F2.
//
// The contract: an exact replay produces ZERO unintended canonical byte
// delta.
func TestF2StoreReplayProducesNoByteDelta(t *testing.T) {
	source := SourceBoard{
		Source: "f2-bytes",
		Items: []SourceItem{
			{SourceID: "W-01", EpicID: "WS-A", Title: "first", Status: SourceStatusOpen},
			{SourceID: "W-02", EpicID: "WS-A", Title: "second", Status: SourceStatusOpen,
				DependsOn: []string{"W-01"}},
		},
	}
	s := newTestStore(t)
	first := BuildPlan(source, domain.Ledger{SchemaVersion: domain.LedgerSchemaVersion})
	if _, err := Apply(s, first, "tx-f2-bytes-1"); err != nil {
		t.Fatalf("apply: %v", err)
	}
	afterFirst, err := s.Load()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	replay := BuildPlan(source, afterFirst)
	if _, err := Apply(s, replay, "tx-f2-bytes-2"); err != nil {
		t.Fatalf("apply the replay: %v", err)
	}
	afterSecond, err := s.Load()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	delta := ComputeDelta(afterFirst, afterSecond)
	if !delta.Empty() {
		t.Fatalf("the replay produced a canonical delta: %s", delta.Describe())
	}
}

// TestF2UnresolvedDependencyStaysNonExecutable witnesses F2.
//
// The contract: an unresolved blocking dependency is a durable blocker. It
// never becomes an executable work item.
func TestF2UnresolvedDependencyStaysNonExecutable(t *testing.T) {
	source := SourceBoard{
		Source: "f2-blocker",
		Items: []SourceItem{
			{SourceID: "W-01", EpicID: "WS-A", Title: "first", Status: SourceStatusOpen,
				DependsOn: []string{"W-MISSING"}},
		},
	}
	plan := BuildPlan(source, domain.Ledger{SchemaVersion: domain.LedgerSchemaVersion})
	if len(plan.Unresolved) == 0 {
		t.Fatal("the unresolved dependency was not reported")
	}
	l := domain.Ledger{SchemaVersion: domain.LedgerSchemaVersion, Items: plan.Items}
	if len(domain.SelectReady(l, domain.ReadyOptions{})) != 0 {
		t.Fatal("the blocked record became executable")
	}
}

// TestF1WorkstreamIdentityIsConservedExplicitly witnesses F1.
//
// The contract: the exact source workstream identity survives the import
// explicitly. It is never derived from the item identity.
func TestF1WorkstreamIdentityIsConservedExplicitly(t *testing.T) {
	source := SourceBoard{
		Source: "f1-workstream",
		Items: []SourceItem{{
			SourceID: "W-01", EpicID: "EPIC-PARENT", Workstream: "RC",
			Title: "signed reasoning preservation", Status: SourceStatusOpen,
		}},
	}

	plan := BuildPlan(source, domain.Ledger{SchemaVersion: domain.LedgerSchemaVersion})
	if len(plan.Items) != 1 {
		t.Fatalf("expected 1 planned item, got %d", len(plan.Items))
	}
	if plan.Items[0].Workstream != "RC" {
		t.Fatalf("the workstream identity was not conserved: %q", plan.Items[0].Workstream)
	}

	s := newTestStore(t)
	if _, err := Apply(s, plan, "tx-f1-ws"); err != nil {
		t.Fatalf("apply: %v", err)
	}
	reloaded, err := s.Load()
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	if reloaded.Items[0].Workstream != "RC" {
		t.Fatalf("the workstream identity did not survive the reload: %q",
			reloaded.Items[0].Workstream)
	}
}
