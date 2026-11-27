package main

import (
	"os"
	"path/filepath"
	"testing"
)

// This file holds the bounded SYNTHETIC multi-epic interchange fixture of the
// FWL02-C CLI qualification.
//
// The fixture exercises the complete interchange contract: an empty
// historical epic, multiple reconciliation records with the same kind and
// resolution, explicit blockers, a known-empty dependency coverage, a
// required dependency outcome, historical ACT associations, evidence locators
// and reopen conditions.
//
// The fixture is SYNTHETIC. It proves the CLI behaviour of the interchange
// contract; it does NOT prove a migration of any actual source.

// writeMultiEpicInterchangePackage writes the bounded multi-epic interchange
// package.
func writeMultiEpicInterchangePackage(t *testing.T) string {
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
  "original_status": "OPEN",
  "lifecycle": "OPEN",
  "priority": "P1",
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
      "criteria_coverage": "COMPLETE",
      "dependencies": [],
      "dependency_coverage": "COMPLETE",
      "blockers": ["the source states that the upstream decision is pending"],
      "current_act": "ACT-CURRENT-01",
      "related_acts": ["ACT-EXAMPLE-01"],
      "evidence_locators": [".factory/evidence/act-example-01.md"],
      "reopen_conditions": ["the upstream decision is taken"],
      "source_refs": []
    }
  ],
  "source_refs": []
}`,
		"epics/EPIC-COMPLETED.json": `{
  "schema_version": 1,
  "epic_id": "EPIC-COMPLETED",
  "title": "The completed scope",
  "original_status": "TERMINAL",
  "lifecycle": "TERMINAL",
  "priority": "P2",
  "exit_criteria": ["the scope is finished"],
  "items": [],
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
    },
    {
      "schema_version": 1,
      "kind": "POSSIBLE_CROSS_LINK",
      "identities": ["EPIC-C/Y", "EPIC-D/Y"],
      "resolution": "UNRESOLVED",
      "reason": "A second, independent cross-link ambiguity.",
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

// readLedgerBytes returns the byte content of every JSON file of a ledger
// root, keyed by its relative path.
func readLedgerBytes(t *testing.T, root string) map[string]string {
	t.Helper()
	out := map[string]string{}
	for _, rel := range listJSONFiles(t, root) {
		data, err := os.ReadFile(filepath.Join(root, rel))
		if err != nil {
			t.Fatalf("read %s: %v", rel, err)
		}
		out[rel] = string(data)
	}
	return out
}
