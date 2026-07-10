//go:build unix || darwin || linux

package execution

import (
	"testing"
)

// testHelperBinary is the path to the test helper binary.
const testHelperBinary = "internal/execution/testdata/testhelper/main"

// expectedRolesForMode defines which roles are expected for each test mode.
var expectedRolesForMode = map[string][]string{
	"sleep":                     {"parent"},
	"ignore-sigterm":            {"parent", "child"},
	"spawn-child":               {"parent", "child"},
	"spawn-grandchild":          {"parent", "child", "grandchild"},
	"hold-stdout-open":          {"parent", "child"},
	"output-forever":            {"parent"},
	"output-forever-child":      {"parent", "child"},
	"output-forever-grandchild": {"parent", "child", "grandchild"},
	"exit-nonzero":              {"parent"},
	"exit-nonzero-child":        {"parent", "child"},
}

// PIDRecord represents a recorded process in the manifest.
type PIDRecord struct {
	Role  string `json:"role"`  // "parent", "child", "grandchild"
	Mode  string `json:"mode"`  // The mode that created this record
	PID   int    `json:"pid"`   // Process ID
	PPID  int    `json:"ppid"`  // Parent process ID
	PGID  int    `json:"pgid"`  // Process group ID
	Start int64  `json:"start"` // Unix timestamp when recorded
}

// processVerifier provides deterministic process verification.
type processVerifier struct {
	manifestFile string
	records      []PIDRecord
	t            *testing.T
}
