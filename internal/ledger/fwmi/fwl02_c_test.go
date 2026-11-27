package fwmi

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/s1onique/leamas/internal/ledger/domain"
	"github.com/s1onique/leamas/internal/ledger/importer"
	"github.com/s1onique/leamas/internal/ledger/store"
)

// This file qualifies the FWL02-C interchange discriminators against the REAL
// production seams (Load, Adapt, importer.BuildPlan, importer.Apply, the
// canonical store and the domain readiness projection).
//
// C2 — the complete field conservation of the interchange wire model.
// C3 — the deterministic reconciliation identity.
// C4 — the explicit epic inventory.

// loadMultiEpic loads the bounded multi-epic fixture.
func loadMultiEpic(t *testing.T) *Package {
	t.Helper()
	pkg, err := Load(writePackage(t, multiEpicPackage()))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	return pkg
}

// newTestStore returns an initialized canonical store in a temporary
// directory.
func newTestStore(t *testing.T) store.Store {
	t.Helper()
	s := store.New(filepath.Join(t.TempDir(), "ledger"))
	if err := s.Init(); err != nil {
		t.Fatalf("init: %v", err)
	}
	return s
}

// TestC2ExplicitBlockerSurvivesPersistenceAndPreventsReadiness witnesses C2.
//
// The contract: an EXPLICIT blocker the source stated survives the import,
// it stays visible in the blocker projection, and it PREVENTS readiness even
// though every dependency resolves.
func TestC2ExplicitBlockerSurvivesPersistenceAndPreventsReadiness(t *testing.T) {
	pkg := loadMultiEpic(t)
	adapted, err := Adapt(pkg)
	if err != nil {
		t.Fatalf("Adapt: %v", err)
	}
	plan := importer.BuildPlan(adapted.Board, domain.Ledger{SchemaVersion: domain.LedgerSchemaVersion})
	if len(plan.Items) != 1 {
		t.Fatalf("planned items = %d, want 1", len(plan.Items))
	}
	blocker := "the source states that the upstream decision is pending"
	if len(plan.Items[0].SourceBlockers) != 1 || plan.Items[0].SourceBlockers[0] != blocker {
		t.Fatalf("the explicit blocker was dropped: %v", plan.Items[0].SourceBlockers)
	}

	s := newTestStore(t)
	if _, err := importer.Apply(s, plan, "tx-c2"); err != nil {
		t.Fatalf("apply: %v", err)
	}
	persisted, err := s.Load()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	item, ok := persisted.ItemByScoped(domain.NewScopedItemID("EPIC-ARCHITECTURE", "E8"))
	if !ok {
		t.Fatal("the imported work item was not persisted")
	}
	if len(item.SourceBlockers) != 1 || item.SourceBlockers[0] != blocker {
		t.Fatalf("the persisted blocker was lost: %v", item.SourceBlockers)
	}
	blockers := domain.BlockersFor(persisted, item)
	shown := false
	for _, reported := range blockers {
		if reported == blocker {
			shown = true
		}
	}
	if !shown {
		t.Fatalf("the blocker projection does not show the explicit blocker: %v", blockers)
	}
	if domain.IsReady(persisted, item, false) {
		t.Fatal("the explicitly blocked work item is ready")
	}
}

// TestC2HistoricalAssociationsSurvivePersistence witnesses C2 for the
// historical associations, the evidence locators and the reopen conditions.
func TestC2HistoricalAssociationsSurvivePersistence(t *testing.T) {
	pkg := loadMultiEpic(t)
	adapted, err := Adapt(pkg)
	if err != nil {
		t.Fatalf("Adapt: %v", err)
	}
	plan := importer.BuildPlan(adapted.Board, domain.Ledger{SchemaVersion: domain.LedgerSchemaVersion})
	item := plan.Items[0]
	if item.SourceCurrentACT != "ACT-CURRENT-01" {
		t.Errorf("current ACT = %q", item.SourceCurrentACT)
	}
	if len(item.SourceRelatedACTs) != 1 || item.SourceRelatedACTs[0] != "ACT-EXAMPLE-01" {
		t.Errorf("related ACTs = %v", item.SourceRelatedACTs)
	}
	if len(item.SourceEvidenceLocators) != 1 {
		t.Errorf("evidence locators = %v", item.SourceEvidenceLocators)
	}
	if len(item.Reopen) != 1 || item.Reopen[0].Detail != "the upstream decision is taken" {
		t.Errorf("reopen conditions = %v", item.Reopen)
	}
	if len(item.Dependencies) != 1 {
		t.Fatalf("dependencies = %v", item.Dependencies)
	}
	if item.Dependencies[0].RequiredOutcome != "E9 is accepted" {
		t.Errorf("the required dependency outcome was lost: %q", item.Dependencies[0].RequiredOutcome)
	}
}

// TestC2UnresolvedHistoricalAssociationIsPreserved witnesses C2 for the
// unresolved historical association.
//
// The contract: an ACT association that does not resolve is conserved
// EXPLICITLY: it is never dropped because its target is unknown.
func TestC2UnresolvedHistoricalAssociationIsPreserved(t *testing.T) {
	pkg := loadMultiEpic(t)
	adapted, err := Adapt(pkg)
	if err != nil {
		t.Fatalf("Adapt: %v", err)
	}
	plan := importer.BuildPlan(adapted.Board, domain.Ledger{SchemaVersion: domain.LedgerSchemaVersion})
	if len(plan.Acts) != 1 {
		t.Fatalf("planned historical ACTs = %d, want 1", len(plan.Acts))
	}
	act := plan.Acts[0]
	if len(act.UnresolvedItemIDs) != 1 || act.UnresolvedItemIDs[0] != "EPIC-ARCHITECTURE/E-MISSING" {
		t.Fatalf("the unresolved association was not preserved: %v", act.UnresolvedItemIDs)
	}
}

// TestC2EffectiveSourceHeadIsRetained witnesses C2 for the source binding.
//
// The contract: a source reference that names no revision of its own is bound
// to the source HEAD of the package, and a reference that names one keeps
// it. A persisted reference therefore always resolves against a concrete
// source revision.
func TestC2EffectiveSourceHeadIsRetained(t *testing.T) {
	files := multiEpicPackage()
	files["epics/EPIC-ARCHITECTURE.json"] = strings.Replace(multiEpicArchitecture,
		`"sha256": "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"}`,
		`"sha256": "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
         "source_head": "1111111111111111111111111111111111111111"}`, 1)
	pkg, err := Load(writePackage(t, files))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	adapted, err := Adapt(pkg)
	if err != nil {
		t.Fatalf("Adapt: %v", err)
	}
	refs := adapted.Board.Items[0].SourceRefs
	if len(refs) != 1 {
		t.Fatalf("source refs = %v", refs)
	}
	if refs[0].SourceHead != "1111111111111111111111111111111111111111" {
		t.Errorf("the explicit source HEAD was lost: %q", refs[0].SourceHead)
	}

	// A reference that names no revision of its own inherits the package
	// HEAD.
	files = multiEpicPackage()
	pkg, err = Load(writePackage(t, files))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	adapted, err = Adapt(pkg)
	if err != nil {
		t.Fatalf("Adapt: %v", err)
	}
	refs = adapted.Board.Items[0].SourceRefs
	if len(refs) != 1 {
		t.Fatalf("source refs = %v", refs)
	}
	if refs[0].SourceHead != "29774ba05851f9dd5a46a33a90d963ff3ef286d5" {
		t.Errorf("the effective source HEAD was lost: %q", refs[0].SourceHead)
	}
}

// TestC2ImportedProvenanceRemainsImported witnesses M2 of the FWL02
// contract.
//
// The contract: an imported record keeps the IMPORTED provenance until it is
// independently verified. The adapter never promotes an imported record to
// a repository-verified fact.
func TestC2ImportedProvenanceRemainsImported(t *testing.T) {
	pkg := loadMultiEpic(t)
	adapted, err := Adapt(pkg)
	if err != nil {
		t.Fatalf("Adapt: %v", err)
	}
	plan := importer.BuildPlan(adapted.Board, domain.Ledger{SchemaVersion: domain.LedgerSchemaVersion})
	item := plan.Items[0]
	if item.Provenance == nil {
		t.Fatal("the imported record carries no provenance")
	}
	if item.Provenance.Confidence != string(domain.ProvenanceImported) {
		t.Errorf("provenance confidence = %q, want the imported confidence",
			item.Provenance.Confidence)
	}
	if item.Provenance.Source != "clinemm" {
		t.Errorf("provenance source = %q, want the package identity", item.Provenance.Source)
	}
}

// The contract: two DISTINCT reconciliation records with the same kind and
// the same resolution do NOT share an identity, and neither overwrites the
// other.
func TestC3DistinctReconciliationRecordsKeepDistinctIdentities(t *testing.T) {
	pkg := loadMultiEpic(t)
	adapted, err := Adapt(pkg)
	if err != nil {
		t.Fatalf("Adapt: %v", err)
	}
	if len(adapted.Migrations) != 2 {
		t.Fatalf("migration records = %d, want 2", len(adapted.Migrations))
	}
	if adapted.Migrations[0].MigrationID == adapted.Migrations[1].MigrationID {
		t.Fatalf("the distinct reconciliation records share one identity: %q",
			adapted.Migrations[0].MigrationID)
	}
	for _, migration := range adapted.Migrations {
		if migration.SourceRepository != "clinemm" {
			t.Errorf("the source repository was lost: %q", migration.SourceRepository)
		}
		if migration.SourceHead == "" {
			t.Error("the effective source HEAD was lost")
		}
		if migration.Outcome != "POSSIBLE_CROSS_LINK" {
			t.Errorf("outcome = %q", migration.Outcome)
		}
		if !strings.Contains(migration.Detail, "UNRESOLVED") {
			t.Errorf("the resolution was lost: %q", migration.Detail)
		}
		if migration.ScopedID == "" {
			t.Error("the affected records were lost")
		}
	}
}

// TestC3ExactReplayDoesNotDuplicateMigrationRecords witnesses C3.
//
// The contract: an EXACT replay produces the same identities and therefore
// no second durable record.
func TestC3ExactReplayDoesNotDuplicateMigrationRecords(t *testing.T) {
	pkg := loadMultiEpic(t)
	first, err := Adapt(pkg)
	if err != nil {
		t.Fatalf("Adapt: %v", err)
	}
	second, err := Adapt(loadMultiEpic(t))
	if err != nil {
		t.Fatalf("Adapt (replay): %v", err)
	}
	if len(first.Migrations) != len(second.Migrations) {
		t.Fatalf("the replay produced a different number of records: %d != %d",
			len(first.Migrations), len(second.Migrations))
	}
	for i := range first.Migrations {
		if first.Migrations[i].MigrationID != second.Migrations[i].MigrationID {
			t.Fatalf("the replay produced a different identity: %q != %q",
				first.Migrations[i].MigrationID, second.Migrations[i].MigrationID)
		}
	}
}

// TestC3ConflictingReconciliationIdentityIsRejected witnesses C3.
//
// The contract: two reconciliation records that claim the SAME identity with
// a DIFFERENT reason are a conflict, and the conflict is rejected BEFORE
// publication.
func TestC3ConflictingReconciliationIdentityIsRejected(t *testing.T) {
	files := multiEpicPackage()
	files[ReconciliationFileName] = strings.Replace(multiEpicReconciliation,
		`"identities": ["EPIC-C/Y", "EPIC-D/Y"]`,
		`"identities": ["EPIC-A/X", "EPIC-B/X"]`, 1)
	pkg, err := Load(writePackage(t, files))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if _, err := Adapt(pkg); err == nil {
		t.Fatal("the conflicting reconciliation identity was accepted")
	}
}

// TestD2ChangedReconciliationReplayPreservesThePriorRecord witnesses the
// changed reconciliation replay at the interchange seam.
//
// The contract: a reconciliation record whose REASON changed in a second
// package does not overwrite the prior durable migration state. The prior
// record is conserved, the change is reported as an explicit reconciliation
// and the conservation receipt reports the replay as conserved.
func TestD2ChangedReconciliationReplayPreservesThePriorRecord(t *testing.T) {
	s := newTestStore(t)
	empty := domain.Ledger{SchemaVersion: domain.LedgerSchemaVersion}

	adapted, err := Adapt(loadMultiEpic(t))
	if err != nil {
		t.Fatalf("Adapt: %v", err)
	}
	first := importer.BuildPlan(adapted.Board, empty)
	if len(first.AdditionalMigrations) != 2 {
		t.Fatalf("planned migration records = %d, want 2", len(first.AdditionalMigrations))
	}
	if _, err := importer.Apply(s, first, "tx-d2-fwmi-1"); err != nil {
		t.Fatalf("apply: %v", err)
	}
	after, err := s.Load()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	priorID := adapted.Migrations[0].MigrationID
	prior, ok := migrationByID(after, priorID)
	if !ok {
		t.Fatal("the durable migration record was not persisted")
	}

	// The SECOND package changes the reason of that reconciliation.
	files := multiEpicPackage()
	files[ReconciliationFileName] = strings.Replace(multiEpicReconciliation,
		`"reason": "Determine whether these are one obligation or two."`,
		`"reason": "The operator changed the reason of this reconciliation."`, 1)
	pkg, err := Load(writePackage(t, files))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	changed, err := Adapt(pkg)
	if err != nil {
		t.Fatalf("Adapt: %v", err)
	}
	if changed.Migrations[0].MigrationID != priorID {
		t.Fatalf("the reconciliation identity changed: %q != %q",
			changed.Migrations[0].MigrationID, priorID)
	}
	plan := importer.BuildPlan(changed.Board, after)
	for _, rec := range plan.AdditionalMigrations {
		if rec.MigrationID == priorID {
			t.Fatal("the changed reconciliation record overwrites the prior durable state")
		}
	}
	reported := false
	for _, rec := range plan.Reconciliations {
		if rec.ScopedID == priorID && rec.Outcome == importer.ReconciliationSourceChanged {
			reported = true
		}
	}
	if !reported {
		t.Fatalf("the changed reconciliation record was not reported: %+v", plan.Reconciliations)
	}
	if _, err := importer.Apply(s, plan, "tx-d2-fwmi-2"); err != nil {
		t.Fatalf("apply the replay: %v", err)
	}
	persisted, err := s.Load()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	conserved, ok := migrationByID(persisted, priorID)
	if !ok {
		t.Fatal("the prior durable migration state was lost")
	}
	if conserved.Detail != prior.Detail {
		t.Fatalf("the prior durable migration state was overwritten: %q", conserved.Detail)
	}
	receipt := importer.BuildPersistedReceipt(after, plan, persisted)
	if !receipt.Conserved {
		t.Fatalf("the changed reconciliation replay is reported as not conserved: %v",
			receipt.Discrepancies)
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

// TestP1DeclaredEpicInheritsThePackageSourceHead witnesses the bounded P1
// correction of the declared epic records.
//
// The contract: the EFFECTIVE source HEAD of a DECLARED epic record is
// retained. A reference that names no revision of its own is bound to the
// source HEAD of the package.
func TestP1DeclaredEpicInheritsThePackageSourceHead(t *testing.T) {
	files := multiEpicPackage()
	files["epics/EPIC-ARCHITECTURE.json"] = strings.Replace(multiEpicArchitecture,
		`  "source_refs": []
}`,
		`  "source_refs": [
    {"path": ".factory/epics/architecture.md", "start_line": 1, "end_line": 9,
     "sha256": "cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc"}
  ]
}`, 1)
	pkg, err := Load(writePackage(t, files))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	adapted, err := Adapt(pkg)
	if err != nil {
		t.Fatalf("Adapt: %v", err)
	}
	var declared importer.SourceEpic
	for _, epic := range adapted.Board.Epics {
		if epic.EpicID == "EPIC-ARCHITECTURE" {
			declared = epic
		}
	}
	if len(declared.SourceRefs) != 1 {
		t.Fatalf("the declared epic source refs = %v", declared.SourceRefs)
	}
	if declared.SourceRefs[0].SourceHead != "29774ba05851f9dd5a46a33a90d963ff3ef286d5" {
		t.Fatalf("the inherited source HEAD was lost: %q", declared.SourceRefs[0].SourceHead)
	}
	plan := importer.BuildPlan(adapted.Board, domain.Ledger{SchemaVersion: domain.LedgerSchemaVersion})
	for _, epic := range plan.Epics {
		if epic.EpicID != "EPIC-ARCHITECTURE" {
			continue
		}
		if len(epic.SourceRefs) != 1 {
			t.Fatalf("the planned epic source refs = %v", epic.SourceRefs)
		}
		if epic.SourceRefs[0].SourceHead != "29774ba05851f9dd5a46a33a90d963ff3ef286d5" {
			t.Fatalf("the planned epic lost the inherited source HEAD: %q",
				epic.SourceRefs[0].SourceHead)
		}
	}
}

// TestP1AllReconciliationSourceReferencesAreConserved witnesses the bounded
// P1 correction of the reconciliation records.
//
// The contract: ALL the source references of a reconciliation record are
// conserved, not only the first one.
func TestP1AllReconciliationSourceReferencesAreConserved(t *testing.T) {
	files := multiEpicPackage()
	files[ReconciliationFileName] = strings.Replace(multiEpicReconciliation,
		`      "reason": "Determine whether these are one obligation or two.",
      "source_refs": []`,
		`      "reason": "Determine whether these are one obligation or two.",
      "source_refs": [
        {"path": ".factory/board.md", "start_line": 3, "end_line": 7,
         "sha256": "dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd"},
        {"path": ".factory/index.md", "start_line": 11, "end_line": 13,
         "sha256": "eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee"}
      ]`, 1)
	pkg, err := Load(writePackage(t, files))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	adapted, err := Adapt(pkg)
	if err != nil {
		t.Fatalf("Adapt: %v", err)
	}
	for _, migration := range adapted.Migrations {
		if migration.MigrationID != adapted.Migrations[0].MigrationID {
			continue
		}
		if len(migration.SourceRefs) != 2 {
			t.Fatalf("the reconciliation source refs = %v", migration.SourceRefs)
		}
		for _, ref := range migration.SourceRefs {
			if ref.SourceHead != "29774ba05851f9dd5a46a33a90d963ff3ef286d5" {
				t.Errorf("the effective source HEAD was lost: %q", ref.SourceHead)
			}
		}
		if migration.SourcePath != ".factory/board.md" {
			t.Errorf("the first source path = %q", migration.SourcePath)
		}
	}
}

// TestC4CompletedEpicWithoutItemsIsDiscoverable witnesses C4.
//
// The contract: a source epic that carries no outstanding work item is still
// a scope of the board: it is discovered, its metadata is conserved and it
// stays discoverable after the import.
func TestC4CompletedEpicWithoutItemsIsDiscoverable(t *testing.T) {
	pkg := loadMultiEpic(t)
	adapted, err := Adapt(pkg)
	if err != nil {
		t.Fatalf("Adapt: %v", err)
	}
	plan := importer.BuildPlan(adapted.Board, domain.Ledger{SchemaVersion: domain.LedgerSchemaVersion})
	if len(plan.Epics) != 2 {
		t.Fatalf("planned epics = %d, want 2", len(plan.Epics))
	}
	var completed *domain.Epic
	for i := range plan.Epics {
		if plan.Epics[i].EpicID == "EPIC-COMPLETED" {
			completed = &plan.Epics[i]
		}
	}
	if completed == nil {
		t.Fatal("the completed epic without work items disappeared from the plan")
	}
	if completed.Title != "The completed scope" {
		t.Errorf("title = %q", completed.Title)
	}
	if completed.Status != "TERMINAL" {
		t.Errorf("the original disposition was lost: %q", completed.Status)
	}
	if len(completed.ExitCriteria) != 1 {
		t.Errorf("exit criteria = %v", completed.ExitCriteria)
	}

	s := newTestStore(t)
	if _, err := importer.Apply(s, plan, "tx-c4"); err != nil {
		t.Fatalf("apply: %v", err)
	}
	persisted, err := s.Load()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	found := false
	for _, epic := range persisted.Epics {
		if epic.EpicID == "EPIC-COMPLETED" {
			found = true
		}
	}
	if !found {
		t.Fatal("the completed epic is not discoverable after the import")
	}
}
