package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ledgerCLI runs a ledger command against the given root and captures
// stdout and stderr.
//
// The root flag is appended after the subcommand, matching the CLI
// convention where the subcommand owns its flag set. For a two-level
// subcommand such as "evidence add" the root flag is appended after the
// second level.
func ledgerCLI(t *testing.T, root string, args ...string) (int, string, string) {
	t.Helper()
	if len(args) == 0 {
		t.Fatal("ledgerCLI requires a subcommand")
	}
	depth := 1
	if args[0] == "evidence" {
		depth = 2
	}
	full := make([]string, 0, len(args)+2)
	full = append(full, args[:depth]...)
	full = append(full, "--root", root)
	full = append(full, args[depth:]...)
	var stdout, stderr bytes.Buffer

	oldStdout, oldStderr := os.Stdout, os.Stderr
	rOut, wOut, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	rErr, wErr, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	os.Stdout, os.Stderr = wOut, wErr

	code := runLedger(full)

	_ = wOut.Close()
	_ = wErr.Close()
	os.Stdout, os.Stderr = oldStdout, oldStderr
	_, _ = stdout.ReadFrom(rOut)
	_, _ = stderr.ReadFrom(rErr)

	return code, stdout.String(), stderr.String()
}

// newLedgerRoot creates an initialized ledger in a temporary directory.
//
// The working directory is switched to the repository root that contains
// the ledger, exactly as an operator would run the command from the
// repository root. The evidence artifacts are resolved against it.
func newLedgerRoot(t *testing.T) string {
	t.Helper()
	repoRoot := t.TempDir()
	root := filepath.Join(repoRoot, "ledger")
	code, _, stderr := ledgerCLI(t, root, "init")
	if code != 0 {
		t.Fatalf("init failed: %d %s", code, stderr)
	}
	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(repoRoot); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(oldWd) })
	return root
}

// TestLedgerCLIInitCreatesCanonicalRoot proves the CLI creates the
// canonical storage layout.
func TestLedgerCLIInitCreatesCanonicalRoot(t *testing.T) {
	root := newLedgerRoot(t)
	for _, rel := range []string{
		"meta.json", "epics", "items", "acts", "evidence", "decisions", "knowledge",
	} {
		if _, err := os.Stat(filepath.Join(root, rel)); err != nil {
			t.Fatalf("expected %s to exist: %v", rel, err)
		}
	}
}

// TestLedgerCLIReadCommandsNeedNoCredentials proves adversarial row 16:
// the default read commands do not require provider credentials or network
// access.
func TestLedgerCLIReadCommandsNeedNoCredentials(t *testing.T) {
	root := newLedgerRoot(t)
	code, _, stderr := ledgerCLI(t, root, "create", "--epic", "EPIC-A",
		"--item", "ITEM-1", "--title", "First")
	if code != 0 {
		t.Fatalf("create failed: %d %s", code, stderr)
	}

	// Every read command must succeed with a hostile environment that has
	// no provider credentials and no network configuration.
	for _, env := range []string{"OPENAI_API_KEY", "ANTHROPIC_API_KEY", "AWS_ACCESS_KEY_ID"} {
		t.Setenv(env, "")
		os.Unsetenv(env)
	}

	for _, args := range [][]string{
		{"list"},
		{"list", "--json"},
		{"show", "EPIC-A/ITEM-1"},
		{"next"},
		{"prime"},
		{"stats"},
		{"validate"},
		{"render", "--view", "board"},
	} {
		code, _, stderr := ledgerCLI(t, root, args...)
		if code != 0 {
			t.Fatalf("read command %v failed: %d %s", args, code, stderr)
		}
	}
}

// TestLedgerCLIJSONOutputIsStable proves the machine-readable output is
// parseable and stable.
func TestLedgerCLIJSONOutputIsStable(t *testing.T) {
	root := newLedgerRoot(t)
	code, _, stderr := ledgerCLI(t, root, "create", "--epic", "EPIC-A",
		"--item", "ITEM-1", "--title", "First")
	if code != 0 {
		t.Fatalf("create failed: %d %s", code, stderr)
	}

	_, first, _ := ledgerCLI(t, root, "list", "--json")
	_, second, _ := ledgerCLI(t, root, "list", "--json")
	if first != second {
		t.Fatal("JSON output is not stable")
	}
	var payload map[string]any
	if err := json.Unmarshal([]byte(first), &payload); err != nil {
		t.Fatalf("JSON output is not parseable: %v\n%s", err, first)
	}
	if payload["ok"] != true {
		t.Fatalf("expected ok=true, got %v", payload["ok"])
	}
}

// TestLedgerCLIRejectsDuplicateIdentity proves the CLI surfaces the
// duplicate-identity rejection.
func TestLedgerCLIRejectsDuplicateIdentity(t *testing.T) {
	root := newLedgerRoot(t)
	code, _, stderr := ledgerCLI(t, root, "create", "--epic", "EPIC-A",
		"--item", "ITEM-1", "--title", "First")
	if code != 0 {
		t.Fatalf("create failed: %d %s", code, stderr)
	}
	code, _, stderr = ledgerCLI(t, root, "create", "--epic", "EPIC-A",
		"--item", "ITEM-1", "--title", "Duplicate")
	if code == 0 {
		t.Fatal("expected duplicate identity rejection")
	}
	if !strings.Contains(stderr, "already exists") {
		t.Fatalf("unexpected error message: %q", stderr)
	}
}

// TestLedgerCLIRejectsStaleRevision proves the CLI surfaces the stale
// revision rejection.
func TestLedgerCLIRejectsStaleRevision(t *testing.T) {
	root := newLedgerRoot(t)
	code, _, stderr := ledgerCLI(t, root, "create", "--epic", "EPIC-A",
		"--item", "ITEM-1", "--title", "First")
	if code != 0 {
		t.Fatalf("create failed: %d %s", code, stderr)
	}
	code, _, stderr = ledgerCLI(t, root, "update", "--id", "EPIC-A/ITEM-1",
		"--title", "Second", "--expect-revision", "99")
	if code == 0 {
		t.Fatal("expected stale revision rejection")
	}
	if !strings.Contains(stderr, "stale revision") {
		t.Fatalf("unexpected error message: %q", stderr)
	}
}
