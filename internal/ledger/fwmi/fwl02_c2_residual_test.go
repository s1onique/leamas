package fwmi

import (
	"strings"
	"testing"

	"github.com/s1onique/leamas/internal/ledger/domain"
	"github.com/s1onique/leamas/internal/ledger/importer"
)

// This file qualifies the FWL02-C2 RESIDUAL corrections at the INTERCHANGE
// seam: the FWMI package is loaded, adapted, planned, applied, reloaded and
// verified against the persisted canonical state.
//
// R2 — the UNRESOLVED RELATIONSHIP metadata must survive the whole chain:
//      the interchange package declares a dependency that does not resolve,
//      and the durable migration state must keep the COMPLETE relationship —
//      its kind, the outcome it required and the byte-exact source locations
//      of the edge.
//
// R1 — the receipt must compare the UNMODIFIED canonical record families, so
//      an evidence record, a decision record or a knowledge reference that
//      the import did not touch still participates in the conservation proof.

// unresolvedDependencyPackage returns a package whose work item declares an
// UNRESOLVED dependency with its complete relationship metadata.
//
// The target of the dependency does not exist in the package: the edge cannot
// resolve and becomes an unresolved relationship.
func unresolvedDependencyPackage(t *testing.T) *Package {
	t.Helper()
	files := multiEpicPackage()
	files["epics/EPIC-ARCHITECTURE.json"] = strings.Replace(
		multiEpicArchitecture,
		`"required_outcome": "E9 is accepted", "source_refs": []}`,
		`"required_outcome": "E9 is accepted",
          "source_refs": [{"path": ".factory/epics/architecture.md", "start_line": 130,
            "end_line": 130, "sha256": "dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd"}]}`,
		1)
	pkg, err := Load(writePackage(t, files))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	return pkg
}

// TestR2InterchangeUnresolvedRelationshipSurvivesPersistence proves the R2
// contract at the interchange seam.
//
// The contract: an unresolved RELATIONSHIP of an interchange package keeps
// its COMPLETE metadata in the durable migration state — the relationship
// kind, the outcome it required and the byte-exact source locations of the
// edge — through Adapt, Apply, an independent reload and the persisted
// conservation verification.
func TestR2InterchangeUnresolvedRelationshipSurvivesPersistence(t *testing.T) {
	pkg := unresolvedDependencyPackage(t)
	adapted, err := Adapt(pkg)
	if err != nil {
		t.Fatalf("Adapt: %v", err)
	}
	plan := importer.BuildPlan(adapted.Board,
		domain.Ledger{SchemaVersion: domain.LedgerSchemaVersion})
	// The fixture also reports the unresolved historical ACT association.
	// The RELATIONSHIP under qualification is the unresolved DEPENDS_ON edge.
	var relationship *importer.UnresolvedReference
	for i := range plan.Unresolved {
		if plan.Unresolved[i].RelationshipKind == "DEPENDS_ON" {
			relationship = &plan.Unresolved[i]
		}
	}
	if relationship == nil {
		t.Fatalf("the plan reports no unresolved dependency relationship: %+v",
			plan.Unresolved)
	}
	unresolved := *relationship
	if unresolved.RequiredOutcome != "E9 is accepted" {
		t.Fatalf("the plan lost the required outcome: %q", unresolved.RequiredOutcome)
	}
	if len(unresolved.SourceRefs) != 1 {
		t.Fatalf("the plan lost the source locations of the edge: %+v", unresolved.SourceRefs)
	}

	s := newTestStore(t)
	if _, err := importer.Apply(s, plan, "tx-fwmi-c2-r2"); err != nil {
		t.Fatalf("apply: %v", err)
	}
	reloaded, err := s.Load()
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	var durable *domain.MigrationRecord
	for i := range reloaded.Migrations {
		if reloaded.Migrations[i].Outcome == "unresolved" {
			durable = &reloaded.Migrations[i]
		}
	}
	if durable == nil {
		t.Fatalf("the unresolved relationship is not in the durable state: %+v",
			reloaded.Migrations)
	}
	if durable.RelationshipKind != "DEPENDS_ON" {
		t.Fatalf("the durable record lost the relationship kind: %q",
			durable.RelationshipKind)
	}
	if durable.RequiredOutcome != "E9 is accepted" {
		t.Fatalf("the durable record lost the required outcome: %q",
			durable.RequiredOutcome)
	}
	if len(durable.RelationshipSourceRefs) != 1 ||
		durable.RelationshipSourceRefs[0].Path != ".factory/epics/architecture.md" {
		t.Fatalf("the durable record lost the source locations of the edge: %+v",
			durable.RelationshipSourceRefs)
	}

	receipt := importer.BuildPersistedReceipt(
		domain.Ledger{SchemaVersion: domain.LedgerSchemaVersion}, plan, reloaded)
	if !receipt.Conserved {
		t.Fatalf("the persisted unresolved relationship is reported as not conserved: %v",
			receipt.Discrepancies)
	}
}

// TestR1InterchangeReceiptComparesUnmodifiedFamilies proves the R1 contract
// at the interchange seam.
//
// The contract: an evidence record that the import did not touch still
// participates in the conservation proof: a persisted state that lost it is
// NOT conserved.
func TestR1InterchangeReceiptComparesUnmodifiedFamilies(t *testing.T) {
	pkg := loadMultiEpic(t)
	adapted, err := Adapt(pkg)
	if err != nil {
		t.Fatalf("Adapt: %v", err)
	}
	plan := importer.BuildPlan(adapted.Board,
		domain.Ledger{SchemaVersion: domain.LedgerSchemaVersion})
	s := newTestStore(t)
	if _, err := importer.Apply(s, plan, "tx-fwmi-c2-r1"); err != nil {
		t.Fatalf("apply: %v", err)
	}
	reloaded, err := s.Load()
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	before := domain.Ledger{SchemaVersion: domain.LedgerSchemaVersion}

	// The persisted state carries an evidence record the import never
	// planned: the receipt must report the unauthorized addition.
	withEvidence := reloaded
	withEvidence.Evidence = []domain.EvidenceRefRecord{{
		SchemaVersion: domain.LedgerSchemaVersion,
		EvidenceID:    "EVIDENCE-FWMI-C2-01",
		Kind:          "artifact",
		Title:         "an evidence record the import never planned",
		Revision:      1,
	}}
	receipt := importer.BuildPersistedReceipt(before, plan, withEvidence)
	if receipt.Conserved {
		t.Fatal("a persisted state with an unauthorized evidence record is reported as conserved")
	}
}
