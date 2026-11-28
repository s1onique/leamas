package importer

import (
	"testing"

	"github.com/s1onique/leamas/internal/ledger/domain"
	"github.com/s1onique/leamas/internal/ledger/store"
)

// This file qualifies the FWL02-C2 RESIDUAL conservation corrections
// against the REAL production seams (BuildPlan, Apply, BuildPersistedReceipt,
// store).
//
// R1 — the UNMODIFIED canonical record families. The receipt must compare the
//      evidence records, the decision records and the knowledge references
//      too: a persisted state that lost one of them is NOT conserved.
//
// R2 — the UNRESOLVED RELATIONSHIP metadata. The durable migration record of
//      an unresolved relationship must carry the COMPLETE relationship: its
//      kind, its required outcome and its byte-exact source locations.
//
// The two counterexamples are reproduced against the production seams, not
// against a synthetic fixture.

const (
	// residualEvidenceID is the identity of the pre-existing evidence record.
	residualEvidenceID = domain.EvidenceID("EVIDENCE-RESIDUAL-01")
	// residualDecisionID is the identity of the pre-existing decision record.
	residualDecisionID = domain.DecisionID("DECISION-RESIDUAL-01")
	// residualKnowledgeID is the identity of the pre-existing knowledge reference.
	residualKnowledgeID = domain.KnowledgeID("KNOWLEDGE-RESIDUAL-01")
)

// residualEvidence returns the pre-existing evidence record.
func residualEvidence() domain.EvidenceRefRecord {
	return domain.EvidenceRefRecord{
		SchemaVersion: domain.LedgerSchemaVersion,
		EvidenceID:    residualEvidenceID,
		Kind:          "artifact",
		Title:         "the pre-existing evidence",
		RelativePath:  ".factory/evidence/residual.md",
		SHA256:        "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
		Quality:       domain.EvidenceQualityVerified,
		Revision:      3,
	}
}

// residualDecision returns the pre-existing decision record.
func residualDecision() domain.Decision {
	return domain.Decision{
		SchemaVersion: domain.LedgerSchemaVersion,
		DecisionID:    residualDecisionID,
		Title:         "the pre-existing decision",
		Rationale:     "the decision was taken before the import",
		Revision:      2,
	}
}

// residualKnowledge returns the pre-existing knowledge reference.
func residualKnowledge() domain.Knowledge {
	return domain.Knowledge{
		SchemaVersion: domain.LedgerSchemaVersion,
		KnowledgeID:   residualKnowledgeID,
		Title:         "the pre-existing knowledge",
		Summary:       "the reusable reference was recorded before the import",
		RelativePath:  "docs/knowledge/residual.md",
		Revision:      1,
	}
}

// residualSeedStore returns a store that carries the three UNMODIFIED
// canonical record families, together with the seeded state.
//
// The records are persisted through the REAL store seam: the receipt must
// compare the state that was actually written to disk.
func residualSeedStore(t *testing.T) (store.Store, domain.Ledger) {
	t.Helper()
	s := newTestStore(t)
	txn, err := s.BeginMutation("tx-c2-seed")
	if err != nil {
		t.Fatalf("begin: %v", err)
	}
	if err := txn.PutEvidence(residualEvidence()); err != nil {
		t.Fatalf("put evidence: %v", err)
	}
	if err := txn.PutDecision(residualDecision()); err != nil {
		t.Fatalf("put decision: %v", err)
	}
	if err := txn.PutKnowledge(residualKnowledge()); err != nil {
		t.Fatalf("put knowledge: %v", err)
	}
	if err := txn.Commit(); err != nil {
		t.Fatalf("commit: %v", err)
	}
	seeded, err := s.Load()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	return s, seeded
}

// residualApply seeds the unmodified record families, applies the import to
// the SAME store and returns the pre-import state, the plan and the
// persisted state.
//
// The import runs against the seeded store, so the persisted state really
// carries the seeded records: the receipt compares the state that was
// actually written to disk.
func residualApply(t *testing.T) (domain.Ledger, Plan, domain.Ledger) {
	t.Helper()
	s, before := residualSeedStore(t)
	plan := BuildPlan(conservationFixture(), before)
	if _, err := Apply(s, plan, "tx-r1-apply"); err != nil {
		t.Fatalf("apply: %v", err)
	}
	persisted, err := s.Load()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if len(persisted.Evidence) != 1 || len(persisted.Decisions) != 1 ||
		len(persisted.Knowledge) != 1 {
		t.Fatalf("the seeded records did not survive the import: %+v", persisted)
	}
	return before, plan, persisted
}

// TestR1UnmodifiedRecordFamiliesAreCompared proves the R1 baseline.
//
// The contract: the intact persisted state — carrying the evidence, the
// decision and the knowledge reference — is conserved.
func TestR1UnmodifiedRecordFamiliesAreCompared(t *testing.T) {
	before, plan, persisted := residualApply(t)
	receipt := BuildPersistedReceipt(before, plan, persisted)
	if !receipt.Conserved {
		t.Fatalf("the intact persisted state is reported as not conserved: %v",
			receipt.Discrepancies)
	}
}

// TestR1DeletedUnmodifiedRecordInvalidatesConservation proves the R1
// counterexamples against the production seam.
//
// The contract: a persisted state that LOST an evidence record, a decision
// record or a knowledge reference is NOT conserved. The three families
// participate in the comparison exactly like the epic, the item, the ACT and
// the migration families.
func TestR1DeletedUnmodifiedRecordInvalidatesConservation(t *testing.T) {
	before, plan, persisted := residualApply(t)
	cases := []struct {
		name   string
		mutate func(l *domain.Ledger)
	}{
		{"evidence", func(l *domain.Ledger) { l.Evidence = nil }},
		{"decision", func(l *domain.Ledger) { l.Decisions = nil }},
		{"knowledge", func(l *domain.Ledger) { l.Knowledge = nil }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mutated := persisted
			tc.mutate(&mutated)
			receipt := BuildPersistedReceipt(before, plan, mutated)
			if receipt.Conserved {
				t.Fatalf("a persisted state that lost the %s is reported as conserved",
					tc.name)
			}
			if len(receipt.Discrepancies) == 0 {
				t.Fatalf("the lost %s is not reported as a discrepancy", tc.name)
			}
		})
	}
}

// TestR1UnauthorizedAdditionInvalidatesConservation proves the R1 addition
// discriminator.
//
// The contract: a persisted state that carries a record the import never
// planned is NOT conserved — for the unmodified families too.
func TestR1UnauthorizedAdditionInvalidatesConservation(t *testing.T) {
	before, plan, persisted := residualApply(t)
	cases := []struct {
		name   string
		mutate func(l *domain.Ledger)
	}{
		{"evidence", func(l *domain.Ledger) {
			l.Evidence = append(append([]domain.EvidenceRefRecord{}, l.Evidence...),
				domain.EvidenceRefRecord{
					SchemaVersion: domain.LedgerSchemaVersion,
					EvidenceID:    "EVIDENCE-UNEXPECTED-01",
					Kind:          "artifact",
					Title:         "an unexpected evidence record",
					Revision:      1,
				})
		}},
		{"decision", func(l *domain.Ledger) {
			l.Decisions = append(append([]domain.Decision{}, l.Decisions...),
				domain.Decision{
					SchemaVersion: domain.LedgerSchemaVersion,
					DecisionID:    "DECISION-UNEXPECTED-01",
					Title:         "an unexpected decision record",
					Revision:      1,
				})
		}},
		{"knowledge", func(l *domain.Ledger) {
			l.Knowledge = append(append([]domain.Knowledge{}, l.Knowledge...),
				domain.Knowledge{
					SchemaVersion: domain.LedgerSchemaVersion,
					KnowledgeID:   "KNOWLEDGE-UNEXPECTED-01",
					Title:         "an unexpected knowledge reference",
					Revision:      1,
				})
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mutated := persisted
			tc.mutate(&mutated)
			receipt := BuildPersistedReceipt(before, plan, mutated)
			if receipt.Conserved {
				t.Fatalf("a persisted state with an unexpected %s is reported as conserved",
					tc.name)
			}
		})
	}
}

// TestR1UnauthorizedChangeInvalidatesConservation proves the R1 change
// discriminator.
//
// The contract: an unauthorized change of an unmodified record is NOT
// conserved: the comparison covers every semantic field of the record.
func TestR1UnauthorizedChangeInvalidatesConservation(t *testing.T) {
	before, plan, persisted := residualApply(t)
	cases := []struct {
		name   string
		mutate func(l *domain.Ledger)
	}{
		{"evidence title", func(l *domain.Ledger) {
			l.Evidence = append([]domain.EvidenceRefRecord{}, l.Evidence...)
			l.Evidence[0].Title = "a rewritten evidence record"
		}},
		{"decision rationale", func(l *domain.Ledger) {
			l.Decisions = append([]domain.Decision{}, l.Decisions...)
			l.Decisions[0].Rationale = "a rewritten rationale"
		}},
		{"knowledge summary", func(l *domain.Ledger) {
			l.Knowledge = append([]domain.Knowledge{}, l.Knowledge...)
			l.Knowledge[0].Summary = "a rewritten summary"
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mutated := persisted
			tc.mutate(&mutated)
			receipt := BuildPersistedReceipt(before, plan, mutated)
			if receipt.Conserved {
				t.Fatalf("an unauthorized change of the %s is reported as conserved",
					tc.name)
			}
		})
	}
}

// TestR2UnresolvedRelationshipMetadataSurvivesPersistence proves the R2
// contract through the REAL production seams.
//
// The contract: an unresolved RELATIONSHIP keeps its COMPLETE metadata in the
// durable migration state — its relationship kind, its required outcome and
// its byte-exact source locations — through Apply, an independent reload and
// the persisted conservation verification.
func TestR2UnresolvedRelationshipMetadataSurvivesPersistence(t *testing.T) {
	refs := []domain.SourceRef{{
		Path:      "board.md",
		StartLine: 11,
		EndLine:   11,
		SHA256:    "cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc",
	}}
	source := SourceBoard{
		Source: "r2-unresolved",
		Items: []SourceItem{{
			SourceID: "W-01",
			EpicID:   "EPIC-A",
			Title:    "the first obligation",
			Status:   SourceStatusOpen,
			DependsOnRelations: []SourceRelation{{
				Kind:            "DEPENDS_ON",
				Target:          "W-MISSING",
				RequiredOutcome: "the merge must land first",
				SourceRefs:      refs,
			}},
		}},
	}
	plan := BuildPlan(source, domain.Ledger{SchemaVersion: domain.LedgerSchemaVersion})
	if len(plan.Unresolved) != 1 {
		t.Fatalf("the plan reports %d unresolved references, want 1", len(plan.Unresolved))
	}
	unresolved := plan.Unresolved[0]
	if unresolved.RelationshipKind != "DEPENDS_ON" {
		t.Fatalf("the plan lost the relationship kind: %q", unresolved.RelationshipKind)
	}
	if unresolved.RequiredOutcome != "the merge must land first" {
		t.Fatalf("the plan lost the required outcome: %q", unresolved.RequiredOutcome)
	}
	if len(unresolved.SourceRefs) != 1 {
		t.Fatalf("the plan lost the source locations: %d", len(unresolved.SourceRefs))
	}

	// The durable migration record carries the COMPLETE relationship.
	records := plan.MigrationRecords()
	var durable *domain.MigrationRecord
	for i := range records {
		if records[i].Outcome == "unresolved" && records[i].ScopedID == "EPIC-A/W-01" {
			durable = &records[i]
		}
	}
	if durable == nil {
		t.Fatalf("the unresolved relationship produced no durable migration record: %+v",
			records)
	}
	if durable.RelationshipKind != "DEPENDS_ON" {
		t.Fatalf("the durable migration record lost the relationship kind: %q",
			durable.RelationshipKind)
	}
	if durable.RequiredOutcome != "the merge must land first" {
		t.Fatalf("the durable migration record lost the required outcome: %q",
			durable.RequiredOutcome)
	}
	if len(durable.RelationshipSourceRefs) != 1 ||
		durable.RelationshipSourceRefs[0].Path != "board.md" ||
		durable.RelationshipSourceRefs[0].SHA256 != refs[0].SHA256 {
		t.Fatalf("the durable migration record lost the source locations: %+v",
			durable.RelationshipSourceRefs)
	}

	// The metadata survives Apply and an INDEPENDENT reload.
	s := newTestStore(t)
	if _, err := Apply(s, plan, "tx-r2"); err != nil {
		t.Fatalf("apply: %v", err)
	}
	reloaded, err := s.Load()
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	var persisted *domain.MigrationRecord
	for i := range reloaded.Migrations {
		if reloaded.Migrations[i].Outcome == "unresolved" &&
			reloaded.Migrations[i].ScopedID == "EPIC-A/W-01" {
			persisted = &reloaded.Migrations[i]
		}
	}
	if persisted == nil {
		t.Fatalf("the unresolved relationship is not in the durable state: %+v",
			reloaded.Migrations)
	}
	if persisted.RelationshipKind != "DEPENDS_ON" {
		t.Fatalf("the persisted relationship kind was lost: %q", persisted.RelationshipKind)
	}
	if persisted.RequiredOutcome != "the merge must land first" {
		t.Fatalf("the persisted required outcome was lost: %q", persisted.RequiredOutcome)
	}
	if len(persisted.RelationshipSourceRefs) != 1 ||
		persisted.RelationshipSourceRefs[0].SHA256 != refs[0].SHA256 {
		t.Fatalf("the persisted source locations were lost: %+v",
			persisted.RelationshipSourceRefs)
	}

	// The persisted conservation verification accepts the state.
	receipt := BuildPersistedReceipt(
		domain.Ledger{SchemaVersion: domain.LedgerSchemaVersion}, plan, reloaded)
	if !receipt.Conserved {
		t.Fatalf("the persisted unresolved relationship is reported as not conserved: %v",
			receipt.Discrepancies)
	}
}
