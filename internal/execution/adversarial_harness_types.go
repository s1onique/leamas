//go:build unix || darwin || linux

package execution

import (
	"testing"
)

// expectedRolesForMode defines which roles are expected for each test mode.
//
// A role that requires a signal_ready manifest flag is listed in
// signalReadyForMode below. The harness waits for both the role and the
// signal_ready flag before publishing readiness to the test.
var expectedRolesForMode = map[string][]string{
	"sleep":                     {"parent"},
	"ignore-sigterm":            {"parent", "child"},
	"spawn-child":               {"parent", "child"},
	"spawn-grandchild":          {"parent", "child", "grandchild"},
	"sleep-grandchild":          {"parent", "child", "grandchild"},
	"hold-stdout-open":          {"parent", "child"},
	"output-forever":            {"parent"},
	"output-forever-child":      {"parent", "child"},
	"output-forever-grandchild": {"parent", "child", "grandchild"},
	"exit-nonzero":              {"parent"},
	"exit-nonzero-child":        {"parent", "child"},
}

// signalReadyForMode lists, for each mode, the roles that MUST show
// signal_ready=true before the test trigger can fire. Roles absent from a
// mode's list do not require the signal_ready flag.
var signalReadyForMode = map[string][]string{
	"ignore-sigterm":   {"child"},
	"sleep-grandchild": {"parent", "child", "grandchild"},
}

// PIDRecord represents a recorded process in the manifest.
//
// SignalReady is true ONLY when the recording process has already installed
// every required signal behavior. The test harness requires this flag for
// the relevant roles and never trusts a record whose flag is false.
type PIDRecord struct {
	Role        string `json:"role"`         // "parent", "child", "grandchild"
	Mode        string `json:"mode"`         // The mode that created this record
	PID         int    `json:"pid"`          // Process ID
	PPID        int    `json:"ppid"`         // Parent process ID
	PGID        int    `json:"pgid"`         // Process group ID
	Start       int64  `json:"start"`        // Unix timestamp when recorded
	SignalReady bool   `json:"signal_ready"` // True iff required signal handlers installed
}

// processVerifier provides deterministic process verification.
type processVerifier struct {
	manifestFile string
	readyDir     string
	records      []PIDRecord
	t            *testing.T
}
