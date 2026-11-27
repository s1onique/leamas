package boardmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// This file qualifies the board extraction adapter against a synthetic
// source that reproduces the ACTUAL topology of the historical board.
//
// The fixture is synthetic: it reproduces the SHAPE of the real board
// (ACT report sections, a current-frontier lane table and per-epic detail
// files) without carrying the real history.

// writeSource writes a synthetic source repository and returns its root.
func writeSource(t *testing.T, board string, epics map[string]string) string {
	t.Helper()
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, EpicsDir), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, BoardPath), []byte(board), 0o644); err != nil {
		t.Fatalf("write board: %v", err)
	}
	for name, content := range epics {
		path := filepath.Join(root, EpicsDir, name)
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatalf("write epic: %v", err)
		}
	}
	return root
}

// demoBoard is the synthetic historical board.
//
// It reproduces the shape of the real board: an ACT report section with a
// metadata block, and a current-frontier lane table.
const demoBoard = "# ACT-CLINEMM-DEMO-01 — PASS — 2026-01-01\n\n" +
	"```text\nACT_ID: ACT-CLINEMM-DEMO-01\nTYPE: demo\n" +
	"VERDICT: PASS\nEPIC: EPIC-DEMO\nWORK_ITEM: DEMO-01 — the demo work item\n" +
	"ENTRY_HEAD: abc1234\nBRANCH: main\n```\n\n" +
	"## Current frontier\n\n" +
	"| Lane | Pri | State | NEXT | Detail |\n" +
	"|---|---|---|---|---|\n" +
	"| Demo lane | P0 | OPEN | `ACT-CLINEMM-DEMO-02` | [`demo.md`](./epics/demo.md) |\n" +
	"| Closed lane | P1 | `CLOSED` | (none) | [`demo.md`](./epics/demo.md) |\n"

// demoEpic is the synthetic epic detail file.
const demoEpic = "# EPIC-DEMO\n\n## Current status\n\n- Status: OPEN\n" +
	"- Priority: **HIGH**\n\n## Open work\n\n" +
	"- **`DEMO-OPEN-01`** (row 1). Status: **OPEN / HIGH**. The open item.\n\n" +
	"## Deferred work\n\n### DEMO-DEFERRED-01\n\nDeferred.\n"

// TestExtractDoesNotTurnACTReportsIntoWorkItems witnesses the W2
// contract.
//
// The contract: a board section is an ACT REPORT, never a work item. The
// board is a navigation index, not an archive of obligations.
func TestExtractDoesNotTurnACTReportsIntoWorkItems(t *testing.T) {
	board := "# ACT-CLINEMM-DEMO-01 — PASS — 2026-01-01\n\n" +
		"```text\nACT_ID: ACT-CLINEMM-DEMO-01\nVERDICT: PASS\nEPIC: EPIC-DEMO\n" +
		"WORK_ITEM: DEMO-01 — the demo work item\nENTRY_HEAD: abc1234\n```\n"
	root := writeSource(t, board, map[string]string{
		"demo.md": "# EPIC-DEMO\n\n## Current status\n\n- Status: OPEN\n",
	})
	ex, err := Extract(root, "demo-repo", "head-1")
	if err != nil {
		t.Fatalf("extract: %v", err)
	}
	if ex.Inventory.ACTs != 1 {
		t.Fatalf("expected 1 ACT execution, got %d", ex.Inventory.ACTs)
	}
	if ex.Inventory.WorkItems != 0 {
		t.Fatalf("the ACT report became a work item: %d records", ex.Inventory.WorkItems)
	}
}

// TestExtractConservesTheProvenanceOfEverySection witnesses the W2
// contract.
//
// The contract: every extracted section carries the source path, the line
// range and the content digest.
func TestExtractConservesTheProvenanceOfEverySection(t *testing.T) {
	board := "# ACT-CLINEMM-DEMO-01 — PASS — 2026-01-01\n\n" +
		"```text\nACT_ID: ACT-CLINEMM-DEMO-01\nVERDICT: PASS\n```\n"
	root := writeSource(t, board, map[string]string{
		"demo.md": "# EPIC-DEMO\n\n## Current status\n\n- Status: OPEN\n",
	})
	ex, err := Extract(root, "demo-repo", "head-1")
	if err != nil {
		t.Fatalf("extract: %v", err)
	}
	if len(ex.Sections) == 0 {
		t.Fatal("no section provenance was recorded")
	}
	for _, sec := range ex.Sections {
		if sec.Path == "" || sec.Locator == "" || sec.Digest == "" {
			t.Fatalf("the section provenance is incomplete: %+v", sec)
		}
		if sec.StartLine < 1 || sec.EndLine < sec.StartLine {
			t.Fatalf("the section line range is invalid: %+v", sec)
		}
		if len(sec.Digest) != 64 {
			t.Fatalf("the section digest is not a SHA-256: %q", sec.Digest)
		}
	}
}

// TestExtractIsDeterministic witnesses the W2 contract.
//
// The contract: the same source produces byte-identical extraction.
func TestExtractIsDeterministic(t *testing.T) {
	root := writeSource(t, demoBoard, map[string]string{"demo.md": demoEpic})
	first, err := Extract(root, "demo-repo", "head-1")
	if err != nil {
		t.Fatalf("extract: %v", err)
	}
	second, err := Extract(root, "demo-repo", "head-1")
	if err != nil {
		t.Fatalf("extract: %v", err)
	}
	if len(first.Board.Items) != len(second.Board.Items) {
		t.Fatalf("the extraction is not deterministic: %d vs %d items",
			len(first.Board.Items), len(second.Board.Items))
	}
	for i := range first.Board.Items {
		if first.Board.Items[i].SourceID != second.Board.Items[i].SourceID {
			t.Fatalf("the extraction is not deterministic at %d", i)
		}
	}
	if first.Inventory != second.Inventory {
		t.Fatal("the observed inventory is not deterministic")
	}
}

// The contract: the current frontier of the historical board is the
// authoritative current state. Every lane becomes a current obligation.
func TestExtractConservesTheFrontierLanes(t *testing.T) {
	root := writeSource(t, demoBoard, map[string]string{"demo.md": demoEpic})

	ex, err := Extract(root, "demo-repo", "head-1")
	if err != nil {
		t.Fatalf("extract: %v", err)
	}
	if ex.Inventory.Epics != 1 {
		t.Fatalf("expected 1 epic scope, got %d", ex.Inventory.Epics)
	}
	if ex.Inventory.ACTs != 1 {
		t.Fatalf("expected 1 ACT execution, got %d", ex.Inventory.ACTs)
	}
	// The two frontier lanes plus the two epic work items.
	if ex.Inventory.WorkItems != 4 {
		t.Fatalf("expected 4 work items, got %d", ex.Inventory.WorkItems)
	}
	lanes := 0
	for _, item := range ex.Board.Items {
		if item.Workstream != "Demo lane" {
			continue
		}
		lanes++
		if item.EpicID != "EPIC-DEMO" {
			t.Fatalf("the frontier lane has the wrong epic scope: %q", item.EpicID)
		}
		if item.Title != "ACT-CLINEMM-DEMO-02" {
			t.Fatalf("the frontier lane title is not the frontier ACT: %q", item.Title)
		}
	}
	if lanes != 1 {
		t.Fatalf("expected 1 frontier lane, got %d", lanes)
	}
}

// TestExtractConservesTheWorkstreamIdentity witnesses the W2 contract.
//
// The contract: the workstream identity of the epic file is conserved
// explicitly. It is distinct from the epic scope.
func TestExtractConservesTheWorkstreamIdentity(t *testing.T) {
	board := "# ACT-CLINEMM-DEMO-01 — PASS — 2026-01-01\n\n" +
		"```text\nACT_ID: ACT-CLINEMM-DEMO-01\nVERDICT: PASS\n```\n"
	root := writeSource(t, board, map[string]string{
		"demo.md": "# EPIC-DEMO\n\n## Current status\n\n- Status: OPEN\n\n" +
			"## Open work\n\n- **`DEMO-OPEN-01`**. Status: **OPEN**. The item.\n",
	})
	ex, err := Extract(root, "demo-repo", "head-1")
	if err != nil {
		t.Fatalf("extract: %v", err)
	}
	found := false
	for _, item := range ex.Board.Items {
		if item.SourceID != "DEMO-OPEN-01" {
			continue
		}
		found = true
		if item.Workstream != "demo" {
			t.Fatalf("the workstream identity was not conserved: %q", item.Workstream)
		}
		if item.EpicID != "EPIC-DEMO" {
			t.Fatalf("the epic scope was not conserved: %q", item.EpicID)
		}
	}
	if !found {
		t.Fatal("the open work item was not extracted")
	}
}

// TestExtractReportsTheUnavailableHistoricalInformation witnesses the W2
// contract.
//
// The contract: the historical information the source does NOT contain is
// reported explicitly.
func TestExtractReportsTheUnavailableHistoricalInformation(t *testing.T) {
	board := "# ACT-CLINEMM-DEMO-01 — PASS — 2026-01-01\n\n" +
		"```text\nACT_ID: ACT-CLINEMM-DEMO-01\nVERDICT: PASS\n```\n"
	root := writeSource(t, board, map[string]string{
		"demo.md": "# EPIC-DEMO\n\n## Current status\n\n- Status: OPEN\n",
	})
	ex, err := Extract(root, "demo-repo", "head-1")
	if err != nil {
		t.Fatalf("extract: %v", err)
	}
	if len(ex.Unavailable) == 0 {
		t.Fatal("the unavailable historical information was not reported")
	}
	joined := strings.Join(ex.Unavailable, "\n")
	if !strings.Contains(joined, "dependency graph") {
		t.Fatalf("the missing dependency graph was not reported: %v", ex.Unavailable)
	}
}

// TestExtractRefusesAMissingBoard witnesses the W2 contract.
//
// The contract: a source topology the adapter cannot read is refused.
func TestExtractRefusesAMissingBoard(t *testing.T) {
	root := t.TempDir()
	if _, err := Extract(root, "demo-repo", "head-1"); err == nil {
		t.Fatal("the adapter accepted a source without a board")
	}
}
