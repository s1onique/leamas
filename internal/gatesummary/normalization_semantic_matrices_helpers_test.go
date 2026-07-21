package gatesummary

import (
	"reflect"
)

// compareProjections normalizes both expected and actual slices
// so that nil and empty are treated equivalently. Production code
// emits nil for zero diagnostics; tests should not have to track
// the difference.
func compareProjections(got, want []diagnosticProjection) bool {
	if len(got) == 0 && len(want) == 0 {
		return true
	}
	return reflect.DeepEqual(got, want)
}

// exitCodeMatrixBody wraps a single check in a complete valid v2
// document body with overall_status chosen by the caller.
func exitCodeMatrixBody(checkJSON, overallStatus string) string {
	return `{
		"schema_version": 2,
		"generated_at": "2026-07-20T12:00:00Z",
		"scope_id": "ACT-EXIT-CODE-MATRIX",
		"scope_status": "OPEN",
		"scope_disposition": "exit code matrix",
		"parent_act": "",
		"parent_status": "CLOSED",
		"parent_disposition": "root",
		"overall_status": "` + overallStatus + `",
		"overall_disposition": "exit code matrix",
		"execution_head_oid": "0123456789abcdef0123456789abcdef01234567",
		"execution_tree_oid": "0123456789abcdef0123456789abcdef01234567",
		"subject_tree_oid": "0123456789abcdef0123456789abcdef01234567",
		"worktree_clean_before": true,
		"worktree_clean_after": true,
		"checks": [` + checkJSON + `]
	}`
}

// checkJSONForMatrix builds a single v2 check fragment with the
// given status and exit_code wire spelling. This is a low-level
// helper; valid builders above enforce closed-world rules.
func checkJSONForMatrix(name, status, exitCodeWire string) string {
	return `{
		"name": "` + name + `",
		"scope": "ROOT",
		"status": "` + status + `",
		"evidence": "e",
		"detail": "d",
		"extras": {
			"argv": [],
			"exit_code": ` + exitCodeWire + `,
			"duration_ms": 0,
			"stdout_sha256": "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
			"stderr_sha256": "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
		}
	}`
}

// largePositiveRaw is greater than math.MaxInt64 (= 9223372036854775807).
const largePositiveRaw = "99999999999999999999"

// largeNegativeRaw is below math.MinInt64 (= -9223372036854775808).
const largeNegativeRaw = "-99999999999999999999"
