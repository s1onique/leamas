package importer

import (
	"path/filepath"
	"testing"

	"github.com/s1onique/leamas/internal/ledger/domain"
	"github.com/s1onique/leamas/internal/ledger/store"
)

// TestImportPreservesSourceIdentityAndProvenance proves that the importer
// preserves the source record identity and its provenance.
func TestImportPreservesSourceIdentityAndProvenance(t *testing.T) {
	board := SourceBoard{
		Source: "conversation-log",
		Items: []SourceItem{{
			SourceID:         "W-01",
			EpicID:           "WS-ALPHA",
			Title:            "Claimed work",
			Status:           SourceStatusUnfinished,
			SourceConfidence: domain.ProvenanceConversational,
		}},
	}
	plan := BuildPlan(board, domain.Ledger{})
	if len(plan.Items) != 1 {
		t.Fatalf("expected one imported item, got %d", len(plan.Items))
	}
	item := plan.Items[0]
	if item.ItemID != "W-01" {
		t.Fatalf("source identity not preserved: %q", item.ItemID)
	}
	if item.Provenance == nil || item.Provenance.SourceID != "W-01" {
		t.Fatalf("source provenance not preserved: %+v", item.Provenance)
	}
	if item.Provenance.Source != "conversation-log" {
		t.Fatalf("source name not preserved: %+v", item.Provenance)
	}
}

// TestImportDoesNotPromoteConversationalClaims proves that a conversational
// claim never becomes a repository-verified fact.
func TestImportDoesNotPromoteConversationalClaims(t *testing.T) {
	board := SourceBoard{
		Source: "conversation-log",
		Items: []SourceItem{{
			SourceID:         "W-01",
			EpicID:           "WS-ALPHA",
			Title:            "Claimed done",
			Status:           SourceStatusClosed,
			SourceConfidence: domain.ProvenanceConversational,
		}},
	}
	plan := BuildPlan(board, domain.Ledger{})
	item := plan.Items[0]
	if item.Acceptance == domain.AcceptanceAccepted {
		t.Fatal("conversational claim was promoted to accepted")
	}
	if item.Quality != domain.EvidenceQualityNone {
		t.Fatalf("conversational claim acquired evidence quality %q", item.Quality)
	}
	if item.Provenance.Confidence != string(domain.ProvenanceConversational) {
		t.Fatalf("provenance confidence was rewritten: %q", item.Provenance.Confidence)
	}
}

// TestImportConservesClosedInsufficientDisposition proves that the
// CLOSED_INSUFFICIENT disposition is conserved through the import.
func TestImportConservesClosedInsufficientDisposition(t *testing.T) {
	board := SourceBoard{
		Source: "conversation-log",
		Items: []SourceItem{{
			SourceID: "W-01",
			EpicID:   "WS-ALPHA",
			Title:    "Closed without proof",
			Status:   SourceStatusClosedInsufficient,
		}},
	}
	plan := BuildPlan(board, domain.Ledger{})
	item := plan.Items[0]
	if item.TerminalDisposition != string(SourceStatusClosedInsufficient) {
		t.Fatalf("disposition not conserved: %q", item.TerminalDisposition)
	}
	if item.Acceptance == domain.AcceptanceAccepted {
		t.Fatal("closed_insufficient record was promoted to accepted")
	}
}

// TestImportReportsAmbiguity proves that a duplicate source identity is
// reported as an explicit ambiguity instead of being silently resolved.
func TestImportReportsAmbiguity(t *testing.T) {
	board := SourceBoard{
		Source: "conversation-log",
		Items: []SourceItem{
			{SourceID: "W-01", EpicID: "WS-ALPHA", Title: "One", Status: SourceStatusUnfinished},
			{SourceID: "W-01", EpicID: "WS-ALPHA", Title: "Two", Status: SourceStatusUnfinished},
		},
	}
	plan := BuildPlan(board, domain.Ledger{})
	if len(plan.Ambiguities) == 0 {
		t.Fatal("expected an ambiguity report")
	}
	found := false
	for _, a := range plan.Ambiguities {
		if a.Kind == "duplicate_source_id" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected duplicate_source_id ambiguity, got %+v", plan.Ambiguities)
	}
}

// TestImportReportsUnresolvedReference proves that a dangling source
// dependency is reported as an unresolved reference.
func TestImportReportsUnresolvedReference(t *testing.T) {
	board := SourceBoard{
		Source: "conversation-log",
		Items: []SourceItem{{
			SourceID:  "W-01",
			EpicID:    "WS-ALPHA",
			Title:     "One",
			Status:    SourceStatusUnfinished,
			DependsOn: []string{"W-99"},
		}},
	}
	plan := BuildPlan(board, domain.Ledger{})
	if len(plan.Unresolved) != 1 {
		t.Fatalf("expected one unresolved reference, got %+v", plan.Unresolved)
	}
	if plan.Unresolved[0].Ref != "W-99" {
		t.Fatalf("unexpected unresolved reference: %+v", plan.Unresolved[0])
	}
}

// TestImportIsIdempotent witnesses adversarial row 14: repeated migration
// yields no unintended canonical delta.
func TestImportIsIdempotent(t *testing.T) {
	dir := t.TempDir()
	st := store.New(filepath.Join(dir, "ledger"))
	if err := st.Init(); err != nil {
		t.Fatalf("init: %v", err)
	}
	board := iecrSourceBoard()

	before, err := st.Load()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	plan := BuildPlan(board, before)
	if _, err := Apply(st, plan, "import-1"); err != nil {
		t.Fatalf("apply: %v", err)
	}
	afterFirst, err := st.Load()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	firstDelta := ComputeDelta(before, afterFirst)
	if firstDelta.Empty() {
		t.Fatal("expected the first import to change the canonical state")
	}

	// Second import: everything already exists.
	secondPlan := BuildPlan(board, afterFirst)
	if len(secondPlan.Items) != 0 {
		t.Fatalf("expected no new items on replay, got %d", len(secondPlan.Items))
	}
	if len(secondPlan.Skipped) != len(board.Items) {
		t.Fatalf("expected %d skipped records, got %d", len(board.Items), len(secondPlan.Skipped))
	}
	if _, err := Apply(st, secondPlan, "import-2"); err != nil {
		t.Fatalf("apply replay: %v", err)
	}
	afterSecond, err := st.Load()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	secondDelta := ComputeDelta(afterFirst, afterSecond)
	if !secondDelta.Empty() {
		t.Fatalf("replayed import produced a canonical delta: %s", secondDelta.Describe())
	}
}

// TestConflictingHistoricalStatusesRemainUnresolved witnesses adversarial
// row 12: conflicting historical statuses remain unresolved rather than
// being silently promoted.
//
// The FWL01 amendment corrects this row: a conflicting source identity
// group is QUARANTINED as a complete group. Input order must never select
// a winner, so no record of the group is imported at all.
func TestConflictingHistoricalStatusesRemainUnresolved(t *testing.T) {
	board := SourceBoard{
		Source: "conversation-log",
		Items: []SourceItem{
			{SourceID: "W-01", EpicID: "WS-ALPHA", Title: "One", Status: SourceStatusClosed},
			{SourceID: "W-01", EpicID: "WS-ALPHA", Title: "One", Status: SourceStatusUnfinished},
		},
	}
	plan := BuildPlan(board, domain.Ledger{})
	if len(plan.Ambiguities) == 0 {
		t.Fatal("expected the conflicting statuses to be reported")
	}
	if len(plan.Quarantined) == 0 {
		t.Fatal("expected the conflicting identity group to be quarantined")
	}
	if len(plan.Items) != 0 {
		t.Fatalf("a conflicting identity group acquired authority: %d records imported", len(plan.Items))
	}

	// The order of the conflicting records must not matter: neither
	// ordering may select a winner.
	reversed := SourceBoard{
		Source: "conversation-log",
		Items: []SourceItem{
			{SourceID: "W-01", EpicID: "WS-ALPHA", Title: "One", Status: SourceStatusUnfinished},
			{SourceID: "W-01", EpicID: "WS-ALPHA", Title: "One", Status: SourceStatusClosed},
		},
	}
	reversedPlan := BuildPlan(reversed, domain.Ledger{})
	if len(reversedPlan.Items) != 0 {
		t.Fatalf("input order selected a winner: %d records imported", len(reversedPlan.Items))
	}
}

// TestImportPreservesWorkItemCount proves the import conserves the number
// of engineering obligations: nothing is lost.
func TestImportPreservesWorkItemCount(t *testing.T) {
	board := iecrSourceBoard()
	plan := BuildPlan(board, domain.Ledger{})
	if len(plan.Items) != len(board.Items) {
		t.Fatalf("import lost obligations: %d of %d", len(plan.Items), len(board.Items))
	}
	if len(plan.Epics) != ExpectedIECRInventory.Workstreams {
		t.Fatalf("expected %d epics, got %d", ExpectedIECRInventory.Workstreams, len(plan.Epics))
	}
	totalExits := 0
	for _, item := range plan.Items {
		totalExits += len(item.ExitCriteria)
	}
	if totalExits != ExpectedIECRInventory.ExitCriteria {
		t.Fatalf("expected %d exit criteria, got %d",
			ExpectedIECRInventory.ExitCriteria, totalExits)
	}
}
