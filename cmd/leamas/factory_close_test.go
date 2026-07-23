package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseFactoryCommandAcceptsClose(t *testing.T) {
	got, err := parseFactoryCommand([]string{"close"})
	if err != nil || got != "close" {
		t.Fatalf("parseFactoryCommand() = %q, %v", got, err)
	}
}

func TestFactoryClosePlanValidate(t *testing.T) {
	path := filepath.Join(t.TempDir(), "plan.json")
	if err := os.WriteFile(path, []byte(testClosurePlanJSON()), 0o644); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	exitCode := runFactoryClose([]string{"plan", "validate", "--file", path}, &stdout, &stderr)
	if exitCode != 0 || stdout.String() != "VALID\n" || stderr.Len() != 0 {
		t.Fatalf("exit=%d stdout=%q stderr=%q", exitCode, stdout.String(), stderr.String())
	}
}

func TestFactoryCloseRejectsUnknownSubcommand(t *testing.T) {
	var stdout, stderr bytes.Buffer
	exitCode := runFactoryClose([]string{"mystery"}, &stdout, &stderr)
	if exitCode == 0 || !strings.Contains(stderr.String(), "unknown") || stdout.Len() != 0 {
		t.Fatalf("exit=%d stdout=%q stderr=%q", exitCode, stdout.String(), stderr.String())
	}
}

func testClosurePlanJSON() string {
	return `{
  "contract_version": 1,
  "act_id": "ACT-LEAMAS-CLI-TEST01",
  "baseline": {
    "commit_oid": "1111111111111111111111111111111111111111",
    "tree_oid": "2222222222222222222222222222222222222222"
  },
  "execution": {"mode": "serial_fail_fast"},
  "checks": [
    {
      "id": "focused",
      "mode": "run",
      "argv": ["go", "test", "./..."],
      "working_directory": ".",
      "timeout_seconds": 60,
      "environment": {}
    }
  ],
  "artifacts": [],
  "policy": {
    "require_clean_before": true,
    "require_clean_after": true,
    "forbid_tracked_full_digests": true,
    "require_diff_check": true
  }
}`
}
