package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestLedgerCLIArchiveRestoreRoundTrip proves the CLI archive and restore
// commands work end to end.
func TestLedgerCLIArchiveRestoreRoundTrip(t *testing.T) {
	root := newLedgerRoot(t)
	code, _, stderr := ledgerCLI(t, root, "create", "--epic", "EPIC-A",
		"--item", "ITEM-1", "--title", "First")
	if code != 0 {
		t.Fatalf("create failed: %d %s", code, stderr)
	}
	if code, _, stderr = ledgerCLI(t, root, "archive", "--id", "EPIC-A/ITEM-1"); code != 0 {
		t.Fatalf("archive failed: %d %s", code, stderr)
	}
	// The archived record is hidden from the live list.
	_, out, _ := ledgerCLI(t, root, "list")
	if strings.Contains(out, "EPIC-A/ITEM-1") {
		t.Fatalf("archived record leaked into the live list:\n%s", out)
	}
	// It remains discoverable.
	_, out, _ = ledgerCLI(t, root, "list", "--archived")
	if !strings.Contains(out, "EPIC-A/ITEM-1") {
		t.Fatalf("archived record is not discoverable:\n%s", out)
	}
	if code, _, stderr = ledgerCLI(t, root, "restore", "--id", "EPIC-A/ITEM-1"); code != 0 {
		t.Fatalf("restore failed: %d %s", code, stderr)
	}
	_, out, _ = ledgerCLI(t, root, "list")
	if !strings.Contains(out, "EPIC-A/ITEM-1") {
		t.Fatalf("restored record is not live:\n%s", out)
	}
}

// TestLedgerCLIPrimeProducesBootstrap proves the bootstrap command renders
// the agent bootstrap context.
func TestLedgerCLIPrimeProducesBootstrap(t *testing.T) {
	root := newLedgerRoot(t)
	code, _, stderr := ledgerCLI(t, root, "create", "--epic", "EPIC-A",
		"--item", "ITEM-1", "--title", "First")
	if code != 0 {
		t.Fatalf("create failed: %d %s", code, stderr)
	}
	code, out, stderr := ledgerCLI(t, root, "prime")
	if code != 0 {
		t.Fatalf("prime failed: %d %s", code, stderr)
	}
	if !strings.Contains(out, "EPIC-A/ITEM-1") {
		t.Fatalf("bootstrap does not carry the work item:\n%s", out)
	}
}

// TestLedgerCLIValidateRejectsBrokenState proves the validate command
// reports a broken candidate state.
func TestLedgerCLIValidateRejectsBrokenState(t *testing.T) {
	root := newLedgerRoot(t)
	code, _, stderr := ledgerCLI(t, root, "create", "--epic", "EPIC-A",
		"--item", "ITEM-1", "--title", "First")
	if code != 0 {
		t.Fatalf("create failed: %d %s", code, stderr)
	}
	// Corrupt the canonical state with an unknown-field record.
	broken := filepath.Join(root, "items", "EPIC-A__ITEM-1.json")
	brokenBody := `{"schema_version":1,"epic_id":"EPIC-A","item_id":"ITEM-1",` +
		`"title":"x","kind":"task","lifecycle":"open","acceptance":"pending",` +
		`"revision":1,"unknown_field":true}` + "\n"
	if err := os.WriteFile(broken, []byte(brokenBody), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	code, out, _ := ledgerCLI(t, root, "validate")
	if code == 0 {
		t.Fatalf("expected validation failure, got:\n%s", out)
	}
}

// TestLedgerCLIImportDryRunWritesNothing proves the import command is a
// dry run by default.
func TestLedgerCLIImportDryRunWritesNothing(t *testing.T) {
	root := newLedgerRoot(t)
	source := filepath.Join(t.TempDir(), "board.json")
	board := `{"source":"test","items":[{"source_id":"W-01","epic_id":"WS-A","title":"One","status":"unfinished"}]}`
	if err := os.WriteFile(source, []byte(board), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	code, out, stderr := ledgerCLI(t, root, "import", "--source", source)
	if code != 0 {
		t.Fatalf("import failed: %d %s", code, stderr)
	}
	if !strings.Contains(out, "dry run") {
		t.Fatalf("expected the dry-run notice:\n%s", out)
	}
	entries, err := os.ReadDir(filepath.Join(root, "items"))
	if err != nil {
		t.Fatalf("read dir: %v", err)
	}
	if len(entries) != 0 {
		t.Fatalf("dry run wrote %d records", len(entries))
	}
}

// TestLedgerCLIProtectedEvidenceRemainsUnchanged witnesses adversarial row
// 18: existing protected evidence remains unchanged.
//
// The ledger never rewrites an evidence artifact. It only records a
// reference to it.
func TestLedgerCLIProtectedEvidenceRemainsUnchanged(t *testing.T) {
	root := newLedgerRoot(t)
	repoRoot := filepath.Dir(root)
	artifactDir := filepath.Join(repoRoot, "evidence-artifacts")
	if err := os.MkdirAll(artifactDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	artifact := filepath.Join(artifactDir, "protected-report.txt")
	original := []byte("protected evidence payload\n")
	if err := os.WriteFile(artifact, original, 0o644); err != nil {
		t.Fatalf("write artifact: %v", err)
	}
	relPath, err := filepath.Rel(repoRoot, artifact)
	if err != nil {
		t.Fatalf("rel: %v", err)
	}

	code, _, stderr := ledgerCLI(t, root, "create", "--epic", "EPIC-A",
		"--item", "ITEM-1", "--title", "First")
	if code != 0 {
		t.Fatalf("create failed: %d %s", code, stderr)
	}
	code, _, stderr = ledgerCLI(t, root, "evidence", "add", "--id", "EV-A",
		"--kind", "file", "--title", "Protected report",
		"--relative-path", relPath,
		"--sha256", "0000000000000000000000000000000000000000000000000000000000000000",
		"--item", "EPIC-A/ITEM-1", "--quality", "verified")
	if code != 0 {
		t.Fatalf("evidence add failed: %d %s", code, stderr)
	}

	after, err := os.ReadFile(artifact)
	if err != nil {
		t.Fatalf("read artifact: %v", err)
	}
	if string(after) != string(original) {
		t.Fatal("the ledger modified the protected evidence artifact")
	}
}

// TestLedgerCLIRejectsEvidencePathTraversal witnesses adversarial row 15
// at the CLI seam: an evidence path that escapes the repository is
// rejected.
func TestLedgerCLIRejectsEvidencePathTraversal(t *testing.T) {
	root := newLedgerRoot(t)
	code, _, stderr := ledgerCLI(t, root, "evidence", "add", "--id", "EV-A",
		"--kind", "file", "--title", "Escape",
		"--relative-path", "../../etc/passwd")
	if code == 0 {
		t.Fatal("expected path traversal rejection")
	}
	if !strings.Contains(stderr, "escapes repository root") {
		t.Fatalf("unexpected error message: %q", stderr)
	}
}

// TestLedgerCLITerminalActRetainsObligation witnesses adversarial row 6 at
// the CLI seam: closing an ACT does not close the engineering obligation.
func TestLedgerCLITerminalActRetainsObligation(t *testing.T) {
	root := newLedgerRoot(t)
	code, _, stderr := ledgerCLI(t, root, "create", "--epic", "EPIC-A",
		"--item", "ITEM-1", "--title", "First")
	if code != 0 {
		t.Fatalf("create failed: %d %s", code, stderr)
	}
	code, _, stderr = ledgerCLI(t, root, "close", "--act", "ACT-A",
		"--item", "EPIC-A/ITEM-1", "--verdict", "closed")
	if code != 0 {
		t.Fatalf("close failed: %d %s", code, stderr)
	}
	_, out, _ := ledgerCLI(t, root, "show", "--json", "EPIC-A/ITEM-1")
	if !strings.Contains(out, `"acceptance": "pending"`) {
		t.Fatalf("terminal ACT closed the engineering obligation:\n%s", out)
	}
}

// TestLedgerCLILinkRejectsDanglingDependency witnesses adversarial row 2
// at the CLI seam.
func TestLedgerCLILinkRejectsDanglingDependency(t *testing.T) {
	root := newLedgerRoot(t)
	code, _, stderr := ledgerCLI(t, root, "create", "--epic", "EPIC-A",
		"--item", "ITEM-1", "--title", "First")
	if code != 0 {
		t.Fatalf("create failed: %d %s", code, stderr)
	}
	code, _, stderr = ledgerCLI(t, root, "link", "--from", "EPIC-A/ITEM-1",
		"--to", "EPIC-A/ITEM-MISSING")
	if code == 0 {
		t.Fatal("expected dangling dependency rejection")
	}
}

// TestLedgerCLIStatsReportsInventory proves the statistics command reports
// the ledger inventory.
func TestLedgerCLIStatsReportsInventory(t *testing.T) {
	root := newLedgerRoot(t)
	code, _, stderr := ledgerCLI(t, root, "create", "--epic", "EPIC-A",
		"--item", "ITEM-1", "--title", "First")
	if code != 0 {
		t.Fatalf("create failed: %d %s", code, stderr)
	}
	code, out, stderr := ledgerCLI(t, root, "stats", "--json")
	if code != 0 {
		t.Fatalf("stats failed: %d %s", code, stderr)
	}
	var payload struct {
		Stats struct {
			Items int `json:"items"`
		} `json:"stats"`
	}
	if err := json.Unmarshal([]byte(out), &payload); err != nil {
		t.Fatalf("parse: %v", err)
	}
	if payload.Stats.Items != 1 {
		t.Fatalf("expected one work item, got %d", payload.Stats.Items)
	}
}

// TestLedgerCLIImportReportsUnresolvedReference witnesses adversarial row 2
// at the import seam: a dangling dependency is explicitly reported as an
// unresolved reference during import.
func TestLedgerCLIImportReportsUnresolvedReference(t *testing.T) {
	root := newLedgerRoot(t)
	source := filepath.Join(t.TempDir(), "board.json")
	board := `{"source":"test","items":[` +
		`{"source_id":"W-01","epic_id":"WS-A","title":"One","status":"unfinished","depends_on":["W-99"]}` +
		`]}`
	if err := os.WriteFile(source, []byte(board), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	code, out, stderr := ledgerCLI(t, root, "import", "--source", source)
	if code != 0 {
		t.Fatalf("import failed: %d %s", code, stderr)
	}
	if !strings.Contains(out, "unresolved") {
		t.Fatalf("expected an unresolved reference report:\n%s", out)
	}
	if !strings.Contains(out, "W-99") {
		t.Fatalf("expected the unresolved reference to be named:\n%s", out)
	}
}

// TestLedgerCLIImportReportsAmbiguity witnesses adversarial row 12 at the
// import seam: conflicting historical statuses remain unresolved.
func TestLedgerCLIImportReportsAmbiguity(t *testing.T) {
	root := newLedgerRoot(t)
	source := filepath.Join(t.TempDir(), "board.json")
	board := `{"source":"test","items":[` +
		`{"source_id":"W-01","epic_id":"WS-A","title":"One","status":"closed"},` +
		`{"source_id":"W-01","epic_id":"WS-A","title":"One","status":"unfinished"}` +
		`]}`
	if err := os.WriteFile(source, []byte(board), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	code, out, stderr := ledgerCLI(t, root, "import", "--source", source)
	if code != 0 {
		t.Fatalf("import failed: %d %s", code, stderr)
	}
	if !strings.Contains(out, "ambiguity") {
		t.Fatalf("expected an ambiguity report:\n%s", out)
	}
}

// TestLedgerCLIRecoverReportsCleanState proves the recover command is
// harmless when no transaction was interrupted.
func TestLedgerCLIRecoverReportsCleanState(t *testing.T) {
	root := newLedgerRoot(t)
	code, out, stderr := ledgerCLI(t, root, "recover")
	if code != 0 {
		t.Fatalf("recover failed: %d %s", code, stderr)
	}
	if !strings.Contains(out, "no interrupted transaction") {
		t.Fatalf("unexpected recover output: %q", out)
	}
}
