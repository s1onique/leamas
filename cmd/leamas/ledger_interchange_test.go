package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeInterchangePackage writes a minimal valid FWMI v1 interchange package.
func writeInterchangePackage(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	files := map[string]string{
		"source-manifest.json": `{
  "schema_version": 1,
  "repository": "clinemm",
  "source_head": "29774ba05851f9dd5a46a33a90d963ff3ef286d5",
  "files": []
}`,
		"epics/EPIC-ARCHITECTURE.json": `{
  "schema_version": 1,
  "epic_id": "EPIC-ARCHITECTURE",
  "title": "Architecture",
  "original_status": null,
  "lifecycle": "OPEN",
  "priority": null,
  "exit_criteria": ["the architecture is documented"],
  "items": [
    {
      "schema_version": 1,
      "item_id": "E8",
      "title": "Recover the meaning of E8",
      "description": "The E8 obligation must be recovered from the source board.",
      "workstream": null,
      "original_status": "OPEN",
      "lifecycle": "OPEN",
      "priority": null,
      "qualification": "UNKNOWN",
      "criteria": [],
      "criteria_coverage": "UNKNOWN",
      "dependencies": [],
      "dependency_coverage": "UNKNOWN",
      "blockers": [],
      "current_act": null,
      "related_acts": ["ACT-EXAMPLE-01"],
      "evidence_locators": [],
      "reopen_conditions": [],
      "source_refs": []
    }
  ],
  "source_refs": []
}`,
		"historical-index.json": `{
  "schema_version": 1,
  "records": [
    {
      "schema_version": 1,
      "act_id": "ACT-EXAMPLE-01",
      "original_verdict": "PASS_STRUCTURAL_ONLY",
      "epic_id": "EPIC-ARCHITECTURE",
      "work_item_ids": ["EPIC-ARCHITECTURE/E8"],
      "evidence_locators": [".factory/evidence/act-example-01.md"],
      "source_refs": [],
      "reconciliation": "UNAMBIGUOUS"
    }
  ]
}`,
		"reconciliation.json": `{
  "schema_version": 1,
  "records": [
    {
      "schema_version": 1,
      "kind": "POSSIBLE_CROSS_LINK",
      "identities": ["EPIC-A/X", "EPIC-B/X"],
      "resolution": "UNRESOLVED",
      "reason": "Determine whether these are one obligation or two.",
      "source_refs": []
    }
  ]
}`,
	}
	for name, body := range files {
		path := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("create %s: %v", name, err)
		}
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}
	return dir
}

// TestLedgerCLIInterchangeDryRunWritesNothing proves the interchange import
// is a dry run by default.
func TestLedgerCLIInterchangeDryRunWritesNothing(t *testing.T) {
	root := newLedgerRoot(t)
	pkg := writeInterchangePackage(t)
	before := listJSONFiles(t, root)
	code, out, stderr := ledgerCLI(t, root, "import-interchange", "--source", pkg)
	if code != 0 {
		t.Fatalf("dry run failed: %d %s", code, stderr)
	}
	if !strings.Contains(out, "dry run") {
		t.Errorf("output = %q, want the dry-run notice", out)
	}
	after := listJSONFiles(t, root)
	if len(after) != len(before) {
		t.Fatalf("the dry run wrote records: %v -> %v", before, after)
	}
}

// TestLedgerCLIInterchangeImportConservesAndReplays proves the applied
// interchange import conserves its semantic content and that an exact replay
// produces no canonical delta.
func TestLedgerCLIInterchangeImportConservesAndReplays(t *testing.T) {
	root := newLedgerRoot(t)
	pkg := writeInterchangePackage(t)
	code, out, stderr := ledgerCLI(t, root, "import-interchange", "--source", pkg, "--apply", "--json")
	if code != 0 {
		t.Fatalf("apply failed: %d %s", code, stderr)
	}
	var doc struct {
		Receipt struct {
			Conserved bool `json:"conserved"`
			Nodes     int  `json:"nodes"`
		} `json:"receipt"`
		Delta struct {
			AddedItems []string `json:"added_items"`
		} `json:"delta"`
	}
	if err := json.Unmarshal([]byte(out), &doc); err != nil {
		t.Fatalf("decode the apply output: %v", err)
	}
	if !doc.Receipt.Conserved {
		t.Fatalf("the applied import did not conserve its content: %s", out)
	}
	if doc.Receipt.Nodes != 1 {
		t.Errorf("nodes = %d, want 1", doc.Receipt.Nodes)
	}
	if len(doc.Delta.AddedItems) != 1 {
		t.Errorf("added items = %v, want one", doc.Delta.AddedItems)
	}

	// The exact replay produces no canonical delta.
	code, out, stderr = ledgerCLI(t, root, "import-interchange", "--source", pkg, "--apply", "--json")
	if code != 0 {
		t.Fatalf("replay failed: %d %s", code, stderr)
	}
	if err := json.Unmarshal([]byte(out), &doc); err != nil {
		t.Fatalf("decode the replay output: %v", err)
	}
	if len(doc.Delta.AddedItems) != 0 {
		t.Fatalf("the replay produced a canonical delta: %v", doc.Delta.AddedItems)
	}
}

// TestLedgerCLIInterchangeHistoryLookup proves the indexed historical ACT
// execution stays discoverable with its original disposition.
func TestLedgerCLIInterchangeHistoryLookup(t *testing.T) {
	root := newLedgerRoot(t)
	pkg := writeInterchangePackage(t)
	if code, _, stderr := ledgerCLI(t, root, "import-interchange", "--source", pkg, "--apply"); code != 0 {
		t.Fatalf("apply failed: %d %s", code, stderr)
	}
	code, out, stderr := ledgerCLI(t, root, "interchange-history", "--id", "ACT-EXAMPLE-01", "--json")
	if code != 0 {
		t.Fatalf("history lookup failed: %d %s", code, stderr)
	}
	var doc struct {
		Act struct {
			ActID           string `json:"act_id"`
			Archived        bool   `json:"archived"`
			Verdict         string `json:"verdict"`
			OriginalVerdict string `json:"original_verdict"`
		} `json:"act"`
	}
	if err := json.Unmarshal([]byte(out), &doc); err != nil {
		t.Fatalf("decode the lookup output: %v", err)
	}
	if doc.Act.ActID != "ACT-EXAMPLE-01" {
		t.Errorf("act id = %q", doc.Act.ActID)
	}
	if !doc.Act.Archived {
		t.Error("the indexed historical execution must stay archived")
	}
	if doc.Act.OriginalVerdict != "PASS_STRUCTURAL_ONLY" {
		t.Errorf("original verdict = %q", doc.Act.OriginalVerdict)
	}
	if doc.Act.Verdict == "closed" {
		t.Error("the historical verdict must not become a canonical terminal verdict")
	}
}
