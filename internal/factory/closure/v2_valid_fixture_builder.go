// SPDX-License-Identifier: Apache-2.0

package closure

// v2_valid_fixture_builder.go provides the authoritative fixture
// builder for contract-valid Plan Contract v1 documents used by
// the v2 runner hermetic tests and the invalid-fixture matrix.
//
// ACT-LEAMAS-FACTORY-CLOSURE-PROTOCOL-V2-RUNNER-VALID-PLAN-AUTHORITY01
// requires every accepted runner fixture to bind:
//   - contract_version = 1
//   - baseline.commit_oid = subject commit
//   - baseline.tree_oid = subject tree (NEVER a commit OID)
//   - valid execution mode (serial_fail_fast)
//   - at least one valid check with unique ID
//   - run-mode checks must include working_directory and
//     timeout_seconds
//   - explicit policy fields
//
// The builder is the single source of truth so producers and
// tests cannot drift from the contract.

import (
	"encoding/json"
	"fmt"
)

// v2FixtureCheck is the canonical in-memory shape for a check
// used by BuildV2ValidPlanFixture. It mirrors the wire-shape
// field names exactly so the marshalled bytes round-trip.
type v2FixtureCheck struct {
	ID               string            `json:"id"`
	Mode             string            `json:"mode"`
	Argv             []string          `json:"argv"`
	WorkingDirectory string            `json:"working_directory"`
	TimeoutSeconds   int               `json:"timeout_seconds"`
	Environment      map[string]string `json:"environment"`
	Reason           string            `json:"reason,omitempty"`
}

// v2FixturePolicy is the canonical policy shape. All four
// boolean fields must be set; otherwise the v1 contract emits
// required_property_missing.
type v2FixturePolicy struct {
	RequireCleanBefore       bool `json:"require_clean_before"`
	RequireCleanAfter        bool `json:"require_clean_after"`
	ForbidTrackedFullDigests bool `json:"forbid_tracked_full_digests"`
	RequireDiffCheck         bool `json:"require_diff_check"`
}

// v2FixtureBaseline binds commit_oid and tree_oid. The
// builder refuses to accept a commit OID as tree_oid.
type v2FixtureBaseline struct {
	CommitOID string `json:"commit_oid"`
	TreeOID   string `json:"tree_oid"`
}

// BuildV2ValidPlanFixture marshals a contract-valid Plan
// Contract v1 document for the supplied subject commit + tree.
//
// The returned bytes:
//   - use contract_version = 1
//   - bind baseline.commit_oid and baseline.tree_oid from the
//     supplied arguments (which must be distinct)
//   - include one run-mode check that exits 0 ("true")
//   - include all four policy boolean fields explicitly
//
// The caller is responsible for committing these bytes under
// the freeze commit's tree so the runner can load them from
// F:P.
func BuildV2ValidPlanFixture(actID, subjectCommit, subjectTree string) ([]byte, error) {
	return BuildV2ValidPlanFixtureWithCheck(actID, subjectCommit, subjectTree,
		v2FixtureCheck{
			ID:               "noop",
			Mode:             "run",
			Argv:             []string{"true"},
			WorkingDirectory: ".",
			TimeoutSeconds:   60,
			Environment:      map[string]string{},
		})
}

// BuildV2ValidPlanFixtureWithCheck is the lower-level builder
// used by tests that need to assert a specific check shape.
func BuildV2ValidPlanFixtureWithCheck(actID, subjectCommit, subjectTree string, check v2FixtureCheck) ([]byte, error) {
	if subjectCommit == "" {
		return nil, fmt.Errorf("subject commit required")
	}
	if subjectTree == "" {
		return nil, fmt.Errorf("subject tree required")
	}
	if subjectCommit == subjectTree {
		// Tree OIDs and commit OIDs are distinct object
		// kinds; rejecting here prevents the recurring
		// "use commit as tree" mistake.
		return nil, fmt.Errorf("commit_oid must differ from tree_oid")
	}
	if check.ID == "" {
		return nil, fmt.Errorf("check id required")
	}
	if check.Mode == "" {
		return nil, fmt.Errorf("check mode required")
	}
	if check.Mode == "run" {
		if check.WorkingDirectory == "" {
			return nil, fmt.Errorf("run check working_directory required")
		}
		if check.TimeoutSeconds < 1 || check.TimeoutSeconds > 600 {
			return nil, fmt.Errorf("run check timeout_seconds must be in [1,600], got %d", check.TimeoutSeconds)
		}
	}
	doc := map[string]any{
		"contract_version": 1,
		"act_id":           actID,
		"baseline": v2FixtureBaseline{
			CommitOID: subjectCommit,
			TreeOID:   subjectTree,
		},
		"execution": map[string]any{
			"mode": "serial_fail_fast",
		},
		"checks":    []v2FixtureCheck{check},
		"artifacts": []map[string]any{},
		"policy": v2FixturePolicy{
			RequireCleanBefore:       true,
			RequireCleanAfter:        true,
			ForbidTrackedFullDigests: true,
			RequireDiffCheck:         true,
		},
	}
	return json.MarshalIndent(doc, "", "  ")
}
