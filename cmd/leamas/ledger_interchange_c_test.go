package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// This file qualifies the FWL02-C interchange behaviour at the CLI seam.
//
// The fixture is the bounded SYNTHETIC multi-epic interchange package. It
// proves the CLI behaviour of the interchange contract; it does NOT prove a
// migration of any actual source.

// TestLedgerCLIInterchangeConservesTheMultiEpicPackage proves the CLI import
// of the bounded multi-epic package conserves its content.
func TestLedgerCLIInterchangeConservesTheMultiEpicPackage(t *testing.T) {
	root := newLedgerRoot(t)
	pkg := writeMultiEpicInterchangePackage(t)
	code, out, stderr := ledgerCLI(t, root, "import-interchange", "--source", pkg, "--apply", "--json")
	if code != 0 {
		t.Fatalf("apply failed: %d %s", code, stderr)
	}
	var doc struct {
		Receipt struct {
			Conserved bool `json:"conserved"`
			Epics     int  `json:"epics"`
			Nodes     int  `json:"nodes"`
		} `json:"receipt"`
		Stats struct {
			Blockers               int `json:"blockers"`
			HistoricalAssociations int `json:"historical_associations"`
			ReconciliationRecords  int `json:"reconciliation_records"`
		} `json:"stats"`
	}
	if err := json.Unmarshal([]byte(out), &doc); err != nil {
		t.Fatalf("decode the apply output: %v", err)
	}
	if !doc.Receipt.Conserved {
		t.Fatalf("the applied import did not conserve its content: %s", out)
	}
	if doc.Receipt.Epics != 2 {
		t.Errorf("epics = %d, want 2 (the completed scope stays discoverable)", doc.Receipt.Epics)
	}
	if doc.Stats.Blockers != 1 {
		t.Errorf("explicit blockers = %d, want 1", doc.Stats.Blockers)
	}
	if doc.Stats.ReconciliationRecords != 2 {
		t.Errorf("reconciliation records = %d, want 2", doc.Stats.ReconciliationRecords)
	}
}

// TestLedgerCLIInterchangeExactReplayIsByteIdentical proves the exact replay
// of an interchange import produces a BYTE-IDENTICAL persisted state.
//
// The contract: the replay does not duplicate a record, does not rewrite a
// record and does not change the persisted bytes.
func TestLedgerCLIInterchangeExactReplayIsByteIdentical(t *testing.T) {
	root := newLedgerRoot(t)
	pkg := writeMultiEpicInterchangePackage(t)
	if code, _, stderr := ledgerCLI(t, root, "import-interchange", "--source", pkg, "--apply"); code != 0 {
		t.Fatalf("apply failed: %d %s", code, stderr)
	}
	before := readLedgerBytes(t, root)
	if code, _, stderr := ledgerCLI(t, root, "import-interchange", "--source", pkg, "--apply"); code != 0 {
		t.Fatalf("replay failed: %d %s", code, stderr)
	}
	after := readLedgerBytes(t, root)
	if len(before) != len(after) {
		t.Fatalf("the replay changed the persisted file set: %d != %d", len(before), len(after))
	}
	for name, content := range before {
		if after[name] != content {
			t.Errorf("the replay changed the persisted bytes of %s", name)
		}
	}
}

// TestLedgerCLIInterchangeChangedReconciliationPreservesThePriorRecord
// proves the changed reconciliation replay at the CLI seam.
//
// The contract: a reconciliation record whose REASON changed in a second
// package does not overwrite the prior durable migration state. The prior
// record is conserved, the change is reported as an explicit reconciliation
// and the import still reports a conserved state.
func TestLedgerCLIInterchangeChangedReconciliationPreservesThePriorRecord(t *testing.T) {
	root := newLedgerRoot(t)
	pkg := writeMultiEpicInterchangePackage(t)
	if code, _, stderr := ledgerCLI(t, root, "import-interchange", "--source", pkg, "--apply"); code != 0 {
		t.Fatalf("the first apply failed: %d %s", code, stderr)
	}
	before := readMigrationDetails(t, root)
	if len(before) == 0 {
		t.Fatal("the first apply produced no durable migration record")
	}

	// The operator changes the reason of one reconciliation.
	path := filepath.Join(pkg, "reconciliation.json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	changed := strings.Replace(string(data),
		`"reason": "Determine whether these are one obligation or two."`,
		`"reason": "The operator changed the reason of this reconciliation."`, 1)
	if err := os.WriteFile(path, []byte(changed), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	code, out, stderr := ledgerCLI(t, root, "import-interchange", "--source", pkg, "--apply", "--json")
	if code != 0 {
		t.Fatalf("the changed replay failed: %d %s", code, stderr)
	}
	var doc struct {
		Receipt struct {
			Conserved bool `json:"conserved"`
		} `json:"receipt"`
		Reconciliations []struct {
			ScopedID string `json:"scoped_id"`
			Outcome  string `json:"outcome"`
		} `json:"reconciliations"`
	}
	if err := json.Unmarshal([]byte(out), &doc); err != nil {
		t.Fatalf("decode the replay output: %v", err)
	}
	if !doc.Receipt.Conserved {
		t.Fatalf("the changed replay is reported as not conserved: %s", out)
	}
	reported := false
	for _, rec := range doc.Reconciliations {
		if rec.Outcome == "source_changed" {
			reported = true
		}
	}
	if !reported {
		t.Fatalf("the changed reconciliation record was not reported: %s", out)
	}
	after := readMigrationDetails(t, root)
	for id, detail := range before {
		if after[id] != detail {
			t.Fatalf("the prior durable migration state %q was overwritten: %q != %q",
				id, after[id], detail)
		}
	}
}

// readMigrationDetails returns the detail of every durable migration record
// of a ledger root, keyed by its identity.
func readMigrationDetails(t *testing.T, root string) map[string]string {
	t.Helper()
	out := map[string]string{}
	dir := filepath.Join(root, "migrations")
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read the migration directory: %v", err)
	}
	for _, entry := range entries {
		data, err := os.ReadFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			t.Fatalf("read %s: %v", entry.Name(), err)
		}
		var record struct {
			MigrationID string `json:"migration_id"`
			Detail      string `json:"detail"`
		}
		if err := json.Unmarshal(data, &record); err != nil {
			t.Fatalf("decode %s: %v", entry.Name(), err)
		}
		out[record.MigrationID] = record.Detail
	}
	return out
}

// TestLedgerCLIInterchangeRejectsAConflictingReconciliationIdentity proves
// the CLI rejects a conflicting reconciliation identity BEFORE publication.
func TestLedgerCLIInterchangeRejectsAConflictingReconciliationIdentity(t *testing.T) {
	root := newLedgerRoot(t)
	pkg := writeMultiEpicInterchangePackage(t)
	path := filepath.Join(pkg, "reconciliation.json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	conflicting := strings.Replace(string(data),
		`"identities": ["EPIC-C/Y", "EPIC-D/Y"]`,
		`"identities": ["EPIC-A/X", "EPIC-B/X"]`, 1)
	if err := os.WriteFile(path, []byte(conflicting), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	code, _, stderr := ledgerCLI(t, root, "import-interchange", "--source", pkg, "--apply")
	if code == 0 {
		t.Fatal("the conflicting reconciliation identity was accepted")
	}
	if !strings.Contains(stderr, "conflicting reconciliation identity") {
		t.Errorf("stderr = %q, want it to name the conflict", stderr)
	}
}
