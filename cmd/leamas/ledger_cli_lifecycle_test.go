package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// acceptLedgerItem registers verified evidence for a work item and accepts
// it through the CLI.
//
// The acceptance needs admissible evidence: a verified artifact bound to
// the work item.
func acceptLedgerItem(t *testing.T, root, id string) {
	t.Helper()
	repoRoot := filepath.Dir(root)
	artifact := filepath.Join(repoRoot, "cli-acceptance.txt")
	payload := []byte("cli acceptance evidence\n")
	if err := os.WriteFile(artifact, payload, 0o644); err != nil {
		t.Fatalf("write artifact: %v", err)
	}
	relPath, err := filepath.Rel(repoRoot, artifact)
	if err != nil {
		t.Fatalf("rel: %v", err)
	}
	sum := sha256.Sum256(payload)
	digest := hex.EncodeToString(sum[:])
	rev := ledgerItemRevision(t, root, id)
	code, _, stderr := ledgerCLI(t, root, "evidence", "add", "--id", "EV-ACCEPT",
		"--kind", "live_qualification", "--title", "acceptance evidence",
		"--relative-path", relPath, "--sha256", digest,
		"--item", id, "--quality", "verified",
		"--class", "live_qualification",
		"--expect-revision", strconv.Itoa(rev))
	if code != 0 {
		t.Fatalf("evidence add failed: %d %s", code, stderr)
	}
	rev = ledgerItemRevision(t, root, id)
	code, _, stderr = ledgerCLI(t, root, "accept", "--id", id,
		"--accepted-by", "cli-test-acceptor",
		"--expect-revision", strconv.Itoa(rev))
	if code != 0 {
		t.Fatalf("accept failed: %d %s", code, stderr)
	}
}

// ledgerItemRevision reads the stored revision of a work item through the
// CLI.
//
// The externally initiated mutation contract requires a POSITIVE expected
// revision: the CLI caller must state the version of the record it
// observed.
func ledgerItemRevision(t *testing.T, root, id string) int {
	t.Helper()
	code, out, stderr := ledgerCLI(t, root, "show", "--json", id)
	if code != 0 {
		t.Fatalf("show %s failed: %d %s", id, code, stderr)
	}
	var doc struct {
		Item struct {
			Revision int `json:"revision"`
		} `json:"item"`
	}
	if err := json.Unmarshal([]byte(out), &doc); err != nil {
		t.Fatalf("decode the show output: %v", err)
	}
	if doc.Item.Revision <= 0 {
		t.Fatalf("the reported revision of %s is %d", id, doc.Item.Revision)
	}
	return doc.Item.Revision
}

// TestLedgerCLIArchiveRestoreRoundTrip proves the CLI archive and restore
// commands work end to end.
//
// The FWL01 amendment qualifies this round: an UNFINISHED work item is not
// archivable. The round trip is therefore performed on an accepted work
// item, which carries no outstanding obligation.
func TestLedgerCLIArchiveRestoreRoundTrip(t *testing.T) {
	root := newLedgerRoot(t)
	code, _, stderr := ledgerCLI(t, root, "create", "--epic", "EPIC-A",
		"--item", "ITEM-1", "--title", "First")
	if code != 0 {
		t.Fatalf("create failed: %d %s", code, stderr)
	}
	acceptLedgerItem(t, root, "EPIC-A/ITEM-1")
	archiveRev := ledgerItemRevision(t, root, "EPIC-A/ITEM-1")
	if code, _, stderr = ledgerCLI(t, root, "archive", "--id", "EPIC-A/ITEM-1",
		"--expect-revision", strconv.Itoa(archiveRev)); code != 0 {
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
	restoreRev := ledgerItemRevision(t, root, "EPIC-A/ITEM-1")
	if code, _, stderr = ledgerCLI(t, root, "restore", "--id", "EPIC-A/ITEM-1",
		"--expect-revision", strconv.Itoa(restoreRev)); code != 0 {
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
