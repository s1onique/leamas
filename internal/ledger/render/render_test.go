package render

import (
	"strings"
	"testing"

	"github.com/s1onique/leamas/internal/ledger/domain"
)

// sampleLedger returns a representative ledger with several scopes.
func sampleLedger() domain.Ledger {
	return domain.Ledger{
		SchemaVersion: domain.LedgerSchemaVersion,
		Epics: []domain.Epic{
			{SchemaVersion: 1, EpicID: "EPIC-A", Title: "Alpha", Priority: domain.PriorityP1, Revision: 1},
			{SchemaVersion: 1, EpicID: "EPIC-B", Title: "Bravo", Priority: domain.PriorityP0, Revision: 1},
		},
		Items: []domain.WorkItem{
			{
				SchemaVersion: 1, EpicID: "EPIC-A", ItemID: "ITEM-2",
				Title: "Second", Kind: domain.ItemKindTask,
				Lifecycle: domain.LifecycleOpen, Acceptance: domain.AcceptancePending,
				Priority: domain.PriorityP1, Quality: domain.EvidenceQualityNone,
				Revision:           1,
				AcceptanceCriteria: []string{"criterion one"},
				Dependencies: []domain.Dependency{
					{Kind: domain.DependencyBlocks, Ref: "EPIC-A/ITEM-1"},
				},
			},
			{
				SchemaVersion: 1, EpicID: "EPIC-A", ItemID: "ITEM-1",
				Title: "First", Kind: domain.ItemKindGate,
				Lifecycle: domain.LifecycleInProgress, Acceptance: domain.AcceptancePending,
				Priority: domain.PriorityP0, Quality: domain.EvidenceQualityReferenced,
				Owner: "agent-a", Revision: 2,
			},
			{
				SchemaVersion: 1, EpicID: "EPIC-B", ItemID: "ITEM-9",
				Title: "Archived", Kind: domain.ItemKindTask,
				Lifecycle: domain.LifecycleArchived, Acceptance: domain.AcceptancePending,
				Quality: domain.EvidenceQualityNone, Revision: 3, Archived: true,
			},
			{
				SchemaVersion: 1, EpicID: "EPIC-B", ItemID: "ITEM-8",
				Title: "Bravo work", Kind: domain.ItemKindTask,
				Lifecycle: domain.LifecycleOpen, Acceptance: domain.AcceptancePending,
				Priority: domain.PriorityP2, Quality: domain.EvidenceQualityNone,
				Revision: 1,
			},
		},
		Acts: []domain.ActExecution{
			{
				SchemaVersion: 1, ActID: "ACT-B", ItemID: "EPIC-A/ITEM-1",
				Verdict: domain.ActVerdictClosed, TerminalDisposition: "closed",
				RetainedObligation: "pending", Revision: 1,
			},
			{
				SchemaVersion: 1, ActID: "ACT-A", ItemID: "EPIC-B/ITEM-9",
				Verdict: domain.ActVerdictClosedInsuf, TerminalDisposition: "closed_insufficient",
				Revision: 1,
			},
		},
	}
}

// TestBoardRenderIsByteIdenticalForIdenticalInput witnesses adversarial
// row 13: identical records generate identical board output.
func TestBoardRenderIsByteIdenticalForIdenticalInput(t *testing.T) {
	a := sampleLedger()
	b := sampleLedger()

	first := Board(a)
	second := Board(b)
	if first != second {
		t.Fatal("identical input produced a different board rendering")
	}
	if Board(a) != Board(b) {
		t.Fatal("board render is not deterministic")
	}
	if len(first) == 0 {
		t.Fatal("board rendering is empty")
	}
}

// TestRenderIsStableAcrossRepeatedCalls proves the render functions are
// pure functions of the canonical input.
func TestRenderIsStableAcrossRepeatedCalls(t *testing.T) {
	l := sampleLedger()
	for i := 0; i < 5; i++ {
		if Board(l) != Board(l) {
			t.Fatal("board render is not stable")
		}
		if ReadyView(l, domain.ReadyOptions{}) != ReadyView(l, domain.ReadyOptions{}) {
			t.Fatal("ready view is not stable")
		}
		if BlockersView(l) != BlockersView(l) {
			t.Fatal("blockers view is not stable")
		}
		if HistoryView(l) != HistoryView(l) {
			t.Fatal("history view is not stable")
		}
		if StatsView(l) != StatsView(l) {
			t.Fatal("stats view is not stable")
		}
	}
}

// TestBoardSortOrderIsStable proves the board sorts deterministically.
func TestBoardSortOrderIsStable(t *testing.T) {
	l := sampleLedger()
	board := domain.BuildBoard(l)
	if len(board.Epics) != 2 {
		t.Fatalf("expected two epics, got %d", len(board.Epics))
	}
	if board.Epics[0].EpicID != "EPIC-A" || board.Epics[1].EpicID != "EPIC-B" {
		t.Fatalf("epics are not sorted: %+v", board.Epics)
	}
	items := board.Epics[0].Items
	if len(items) != 2 {
		t.Fatalf("expected two items, got %d", len(items))
	}
	// P0 outranks P1, so ITEM-1 comes first.
	if items[0].ScopedID != "EPIC-A/ITEM-1" {
		t.Fatalf("items are not sorted by priority: %+v", items)
	}
}

// TestArchivedRecordsAreExcludedFromTheOperationalBoard proves the
// operational board is a projection of the live records.
func TestArchivedRecordsAreExcludedFromTheOperationalBoard(t *testing.T) {
	l := sampleLedger()
	board := domain.BuildBoard(l)
	for _, epic := range board.Epics {
		for _, item := range epic.Items {
			if item.ScopedID == "EPIC-B/ITEM-9" {
				t.Fatal("archived record leaked into the operational board")
			}
		}
	}
}

// TestReadyViewRespectsAcceptanceRequirements proves that ready-work
// selection respects acceptance requirements.
func TestReadyViewRespectsAcceptanceRequirements(t *testing.T) {
	l := sampleLedger()
	ready := domain.SelectReady(l, domain.ReadyOptions{})
	for _, item := range ready {
		if item.ScopedID == "EPIC-A/ITEM-2" {
			t.Fatal("item with unsatisfied acceptance criteria was selected")
		}
	}
	found := false
	for _, item := range ready {
		if item.ScopedID == "EPIC-A/ITEM-1" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected EPIC-A/ITEM-1 to be ready, got %+v", ready)
	}
}

// TestReadyViewRespectsEpicScope proves that ready-work selection respects
// the authorization boundary of the epic scope.
func TestReadyViewRespectsEpicScope(t *testing.T) {
	l := sampleLedger()
	ready := domain.SelectReady(l, domain.ReadyOptions{Epic: "EPIC-B"})
	for _, item := range ready {
		if item.EpicID != "EPIC-B" {
			t.Fatalf("selection escaped the epic scope: %+v", item)
		}
	}
}

// TestBootstrapReportsBudgetOverflow proves that the bootstrap context
// never silently truncates a required obligation.
func TestBootstrapReportsBudgetOverflow(t *testing.T) {
	l := sampleLedger()
	text := Bootstrap(l, BootstrapOptions{MaxTokens: 1})
	if !strings.Contains(text, "token budget") {
		t.Fatalf("expected the budget overflow to be reported, got:\n%s", text)
	}
}

// TestBootstrapKeepsObligations proves the bootstrap renders the criteria.
func TestBootstrapKeepsObligations(t *testing.T) {
	l := sampleLedger()
	text := Bootstrap(l, BootstrapOptions{})
	if !strings.Contains(text, "criterion one") {
		t.Fatalf("expected the acceptance criterion in the bootstrap, got:\n%s", text)
	}
	if strings.Contains(text, "Archived") {
		t.Fatalf("archived record leaked into the bootstrap, got:\n%s", text)
	}
}

// TestEstimateTokensIsConservative proves the token estimate never
// under-reports the real size.
func TestEstimateTokensIsConservative(t *testing.T) {
	text := strings.Repeat("x", 400)
	if EstimateTokens(text) < 100 {
		t.Fatalf("token estimate under-reports: %d", EstimateTokens(text))
	}
	if EstimateTokens("") != 0 {
		t.Fatal("empty text must estimate zero tokens")
	}
}
