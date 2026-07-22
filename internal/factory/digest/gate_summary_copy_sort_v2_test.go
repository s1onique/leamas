// SPDX-License-Identifier: Apache-2.0

package digest

import (
	"strings"
	"testing"

	"github.com/s1onique/leamas/internal/gatesummary"
)

// normalizeValidV2Document normalizes a V2 JSON document and returns the summary.
// Fails the test if normalization fails.
func normalizeValidV2Document(t *testing.T, json string) gatesummary.Summary {
	t.Helper()
	decoded := gatesummary.Decode(strings.NewReader(json))
	if !decoded.Success() {
		t.Fatalf("decode failed: %v", decoded.Diagnostics)
	}
	norm := gatesummary.Normalize(decoded.Document)
	if !norm.Success() {
		t.Fatalf("normalize failed: %v", norm.Diagnostics)
	}
	return norm.Summary
}

// stringPtr returns a pointer to a string.
func stringPtr(s string) *string {
	return &s
}

// TestCopyAndSortChecksV2Duration exercises the digest's copyAndSortChecksV2
// with checks that have identical earlier keys but different duration values.
func TestCopyAndSortChecksV2Duration(t *testing.T) {
	t.Parallel()
	// Create two checks with different names so normalization succeeds
	// then equalize all earlier keys in test-owned copies.
	doc := `{
		"schema_version": 2,
		"generated_at": "2024-01-01T00:00:00Z",
		"scope_id": "TEST",
		"scope_status": "CLOSED",
		"scope_disposition": "done",
		"parent_act": "",
		"parent_status": "CLOSED",
		"parent_disposition": "root",
		"overall_status": "pass",
		"overall_disposition": "done",
		"execution_head_oid": "0123456789abcdef0123456789abcdef01234567",
		"execution_tree_oid": "0123456789abcdef0123456789abcdef01234567",
		"subject_tree_oid": "0123456789abcdef0123456789abcdef01234567",
		"worktree_clean_before": true,
		"worktree_clean_after": true,
		"checks": [
			{
				"name": "check_alpha",
				"scope": "ROOT",
				"status": "pass",
				"evidence": "alpha.sh",
				"detail": "ok",
				"extras": {
					"argv": ["alpha.sh"],
					"exit_code": 0,
					"duration_ms": 200,
					"stdout_sha256": "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
					"stderr_sha256": "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
				}
			},
			{
				"name": "check_beta",
				"scope": "ROOT",
				"status": "pass",
				"evidence": "beta.sh",
				"detail": "ok",
				"extras": {
					"argv": ["beta.sh"],
					"exit_code": 0,
					"duration_ms": 100,
					"stdout_sha256": "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
					"stderr_sha256": "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
				}
			}
		]
	}`

	summary := normalizeValidV2Document(t, doc)

	// Equalize all earlier sort keys so duration decides order.
	// V2 canonical key order: name, scope, status, duration-present, duration-value, ...
	// We need name, scope, status to be equal.
	left := summary.Checks[0]
	right := summary.Checks[1]

	// Make names equal
	left.Name = "same"
	right.Name = "same"

	// Make scopes equal (both are already "ROOT" via stringPtr("ROOT"))
	scope := "ROOT"
	left.Scope = &scope
	right.Scope = &scope

	// Make statuses equal
	left.Status = gatesummary.GatePass
	right.Status = gatesummary.GatePass

	// Make evidence equal
	left.Evidence = stringPtr("same")
	right.Evidence = stringPtr("same")

	// Sort: left has duration=200, right has duration=100
	sorted := copyAndSortChecksV2([]gatesummary.Check{left, right})

	// After canonical sort, duration=100 should come first
	if got := sorted[0].DurationMs.String(); got != "100" {
		t.Fatalf("first duration = %s, want 100", got)
	}
	if got := sorted[1].DurationMs.String(); got != "200" {
		t.Fatalf("second duration = %s, want 200", got)
	}
}

// TestCopyAndSortChecksV2ExitCode exercises the digest's copyAndSortChecksV2
// with checks that have identical earlier keys but different exit code values.
func TestCopyAndSortChecksV2ExitCode(t *testing.T) {
	t.Parallel()
	// Create two checks with different names so normalization succeeds
	// then equalize all earlier keys in test-owned copies.
	doc := `{
		"schema_version": 2,
		"generated_at": "2024-01-01T00:00:00Z",
		"scope_id": "TEST",
		"scope_status": "CLOSED",
		"scope_disposition": "done",
		"parent_act": "",
		"parent_status": "CLOSED",
		"parent_disposition": "root",
		"overall_status": "fail",
		"overall_disposition": "done",
		"execution_head_oid": "0123456789abcdef0123456789abcdef01234567",
		"execution_tree_oid": "0123456789abcdef0123456789abcdef01234567",
		"subject_tree_oid": "0123456789abcdef0123456789abcdef01234567",
		"worktree_clean_before": true,
		"worktree_clean_after": true,
		"checks": [
			{
				"name": "check_alpha",
				"scope": "ROOT",
				"status": "fail",
				"evidence": "alpha.sh",
				"detail": "failed",
				"extras": {
					"argv": ["alpha.sh"],
					"exit_code": 2,
					"duration_ms": 100,
					"stdout_sha256": "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
					"stderr_sha256": "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
				}
			},
			{
				"name": "check_beta",
				"scope": "ROOT",
				"status": "fail",
				"evidence": "beta.sh",
				"detail": "failed",
				"extras": {
					"argv": ["beta.sh"],
					"exit_code": 1,
					"duration_ms": 100,
					"stdout_sha256": "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
					"stderr_sha256": "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
				}
			}
		]
	}`

	summary := normalizeValidV2Document(t, doc)

	// Equalize all earlier sort keys so exit code decides order.
	// V2 canonical key order: name, scope, status, duration-present, duration-value, exit-code-present, exit-code-value, ...
	left := summary.Checks[0]
	right := summary.Checks[1]

	// Make names equal
	left.Name = "same"
	right.Name = "same"

	// Make scopes equal
	scope := "ROOT"
	left.Scope = &scope
	right.Scope = &scope

	// Make statuses equal
	left.Status = gatesummary.GateFail
	right.Status = gatesummary.GateFail

	// Make evidence equal
	left.Evidence = stringPtr("same")
	right.Evidence = stringPtr("same")

	// Make durations equal
	left.DurationMs = right.DurationMs

	// Sort: left has exit_code=2, right has exit_code=1
	sorted := copyAndSortChecksV2([]gatesummary.Check{left, right})

	// After canonical sort, exit_code=1 should come first
	if got := sorted[0].Execution.ExitCode.String(); got != "1" {
		t.Fatalf("first exit_code = %s, want 1", got)
	}
	if got := sorted[1].Execution.ExitCode.String(); got != "2" {
		t.Fatalf("second exit_code = %s, want 2", got)
	}
}
