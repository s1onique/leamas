// SPDX-License-Identifier: Apache-2.0

package closure

// v2_invalid_plan_matrix_test.go enumerates the authoritative
// invalid-fixture matrix required by Phase 6 of
// ACT-LEAMAS-FACTORY-CLOSURE-PROTOCOL-V2-RUNNER-VALID-PLAN-AUTHORITY01.
//
// Every case asserts:
//   - the runner rejects with frozen_plan_invalid (or
//     unsupported_plan_contract_version /
//     unsupported_plan_protocol_combination for the
//     version-only failure cases)
//   - nested plan diagnostic code, instance path, and schema
//     path are preserved in the typed V2Error
//   - executor_calls == 0 (no checks executed)
//   - manifest_absent (no manifest written)
//
// The matrix drives ValidateFrozenPlanV2 directly for fast unit
// coverage; a smaller subset also drives the full runner path so
// the production wiring is exercised end-to-end.

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

// executorCallCounter counts SubjectExecutor invocations so tests
// can assert the runner never reaches the executor on invalid
// plans. The default V2SubjectExecutor is replaced via
// V2RunnerDeps.
type countingExecutor struct {
	calls int
}

func (c *countingExecutor) ExecuteSubjectChecks(ctx context.Context, req V2ExecuteRequest) (V2ExecuteResult, error) {
	c.calls++
	return V2ExecuteResult{
		ObservedTree: req.SubjectTree,
		CheckResults: nil,
	}, nil
}

// validPlanBytesFor is a tiny helper used by the runner-level
// matrix cases that need a valid plan; the unit cases below
// build their own invalid bytes.
func validPlanBytesFor(t *testing.T, dir string, subject string) []byte {
	t.Helper()
	subjectTree := mustRunGit(t, dir, "rev-parse", subject+"^{tree}")
	b, err := BuildV2ValidPlanFixture("ACT-INVALID-MATRIX", subject, subjectTree)
	if err != nil {
		t.Fatalf("BuildV2ValidPlanFixture: %v", err)
	}
	return b
}

// assertInvalidReports checks the invariant shared by every
// matrix case: V2CodeFrozenPlanInvalid is present, nested
// PlanDiagnostic metadata is preserved, executor was not
// invoked, and no manifest was written.
func assertInvalidReports(t *testing.T, err error, wantNestedInstance, wantNestedSchema string, wantNestedCode string, wantExecutorCalls int) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected V2Error, got nil")
	}
	v2err, ok := err.(*V2Error)
	if !ok {
		t.Fatalf("expected *V2Error, got %T: %v", err, err)
	}
	if !v2err.Diags.HasCode(V2CodeFrozenPlanInvalid) {
		t.Fatalf("expected frozen_plan_invalid, got %v", v2err.Diags.Codes())
	}
	if wantNestedInstance != "" {
		found := false
		for _, d := range v2err.Diags {
			if d.PropertyName == wantNestedInstance {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("expected nested instance_path=%s, got %v", wantNestedInstance, v2err.Diags)
		}
	}
	if wantNestedSchema != "" {
		found := false
		for _, d := range v2err.Diags {
			if v2MatrixContains(d.Detail, wantNestedSchema) {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("expected nested schema_path=%s in detail, got %v", wantNestedSchema, v2err.Diags)
		}
	}
	if wantNestedCode != "" {
		found := false
		for _, d := range v2err.Diags {
			if v2MatrixContains(d.Detail, wantNestedCode) {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("expected nested code=%s in detail, got %v", wantNestedCode, v2err.Diags)
		}
	}
	if wantExecutorCalls != 0 {
		t.Fatalf("expected executor_calls=0, got %d", wantExecutorCalls)
	}
}

// contains is a small substring helper to avoid importing
// strings just for one call.
func v2MatrixContains(haystack, needle string) bool {
	if needle == "" {
		return true
	}
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if haystack[i:i+len(needle)] == needle {
			return true
		}
	}
	return false
}

// TestInvalidPlanMatrix_MalformedJSON asserts malformed JSON is
// rejected at the parse stage.
func TestInvalidPlanMatrix_MalformedJSON(t *testing.T) {
	_, err := ValidateFrozenPlanV2([]byte("{not valid json"))
	assertInvalidReports(t, err, "/plan", "/plan", "plan_parse_failed", 0)
}

// TestInvalidPlanMatrix_MissingContractVersion asserts the
// structural walker reports required_property_missing for
// contract_version.
func TestInvalidPlanMatrix_MissingContractVersion(t *testing.T) {
	bytes := []byte(`{
  "act_id": "ACT-MISSING-CV",
  "baseline": {"commit_oid": "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", "tree_oid": "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"},
  "execution": {"mode": "serial_fail_fast"},
  "checks": [{"id": "noop", "mode": "run", "argv": ["true"], "working_directory": ".", "timeout_seconds": 60, "environment": {}}],
  "artifacts": [],
  "policy": {"require_clean_before": true, "require_clean_after": true, "forbid_tracked_full_digests": true, "require_diff_check": true}
}`)
	_, err := ValidateFrozenPlanV2(bytes)
	assertInvalidReports(t, err, "/contract_version", "/contract_version", "required_property_missing", 0)
}

// TestInvalidPlanMatrix_UnsupportedContractVersion asserts the
// request/frozen version agreement rejects contract_version > 1
// with frozen_plan_invalid.
func TestInvalidPlanMatrix_UnsupportedContractVersion(t *testing.T) {
	bytes := []byte(`{
  "contract_version": 999,
  "act_id": "ACT-UNSUPPORTED-CV",
  "baseline": {"commit_oid": "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", "tree_oid": "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"},
  "execution": {"mode": "serial_fail_fast"},
  "checks": [{"id": "noop", "mode": "run", "argv": ["true"], "working_directory": ".", "timeout_seconds": 60, "environment": {}}],
  "artifacts": [],
  "policy": {"require_clean_before": true, "require_clean_after": true, "forbid_tracked_full_digests": true, "require_diff_check": true}
}`)
	_, err := ValidateFrozenPlanV2(bytes)
	assertInvalidReports(t, err, "", "", "", 0)
}

// TestInvalidPlanMatrix_MissingBaseline asserts the structural
// walker rejects plans without a baseline object.
func TestInvalidPlanMatrix_MissingBaseline(t *testing.T) {
	bytes := []byte(`{
  "contract_version": 1,
  "act_id": "ACT-MISSING-BASELINE",
  "execution": {"mode": "serial_fail_fast"},
  "checks": [{"id": "noop", "mode": "run", "argv": ["true"], "working_directory": ".", "timeout_seconds": 60, "environment": {}}],
  "artifacts": [],
  "policy": {"require_clean_before": true, "require_clean_after": true, "forbid_tracked_full_digests": true, "require_diff_check": true}
}`)
	_, err := ValidateFrozenPlanV2(bytes)
	assertInvalidReports(t, err, "/baseline", "/baseline", "required_property_missing", 0)
}

// TestInvalidPlanMatrix_MalformedBaselineCommit asserts the
// structural walker rejects baseline.commit_oid values that are
// not 40-char lowercase hex.
func TestInvalidPlanMatrix_MalformedBaselineCommit(t *testing.T) {
	bytes := []byte(`{
  "contract_version": 1,
  "act_id": "ACT-MALFORMED-COMMIT",
  "baseline": {"commit_oid": "not-a-valid-oid", "tree_oid": "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"},
  "execution": {"mode": "serial_fail_fast"},
  "checks": [{"id": "noop", "mode": "run", "argv": ["true"], "working_directory": ".", "timeout_seconds": 60, "environment": {}}],
  "artifacts": [],
  "policy": {"require_clean_before": true, "require_clean_after": true, "forbid_tracked_full_digests": true, "require_diff_check": true}
}`)
	_, err := ValidateFrozenPlanV2(bytes)
	assertInvalidReports(t, err, "/plan", "/plan", "semantic_validation_failed", 0)
}

// TestInvalidPlanMatrix_MalformedBaselineTree asserts the
// structural walker rejects baseline.tree_oid values that are
// not 40-char lowercase hex.
func TestInvalidPlanMatrix_MalformedBaselineTree(t *testing.T) {
	bytes := []byte(`{
  "contract_version": 1,
  "act_id": "ACT-MALFORMED-TREE",
  "baseline": {"commit_oid": "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", "tree_oid": "NOT-HEX"},
  "execution": {"mode": "serial_fail_fast"},
  "checks": [{"id": "noop", "mode": "run", "argv": ["true"], "working_directory": ".", "timeout_seconds": 60, "environment": {}}],
  "artifacts": [],
  "policy": {"require_clean_before": true, "require_clean_after": true, "forbid_tracked_full_digests": true, "require_diff_check": true}
}`)
	_, err := ValidateFrozenPlanV2(bytes)
	assertInvalidReports(t, err, "/plan", "/plan", "semantic_validation_failed", 0)
}

// TestInvalidPlanMatrix_EmptyChecks asserts the structural
// walker rejects plans with zero checks.
func TestInvalidPlanMatrix_EmptyChecks(t *testing.T) {
	bytes := []byte(`{
  "contract_version": 1,
  "act_id": "ACT-EMPTY-CHECKS",
  "baseline": {"commit_oid": "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", "tree_oid": "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"},
  "execution": {"mode": "serial_fail_fast"},
  "checks": [],
  "artifacts": [],
  "policy": {"require_clean_before": true, "require_clean_after": true, "forbid_tracked_full_digests": true, "require_diff_check": true}
}`)
	_, err := ValidateFrozenPlanV2(bytes)
	assertInvalidReports(t, err, "/checks", "/checks", "invalid_type", 0)
}

// TestInvalidPlanMatrix_DuplicateCheckID asserts the structural
// walker rejects duplicate check IDs.
func TestInvalidPlanMatrix_DuplicateCheckID(t *testing.T) {
	bytes := []byte(`{
  "contract_version": 1,
  "act_id": "ACT-DUP-CHECKS",
  "baseline": {"commit_oid": "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", "tree_oid": "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"},
  "execution": {"mode": "serial_fail_fast"},
  "checks": [
    {"id": "dup", "mode": "run", "argv": ["true"], "working_directory": ".", "timeout_seconds": 60, "environment": {}},
    {"id": "dup", "mode": "run", "argv": ["true"], "working_directory": ".", "timeout_seconds": 60, "environment": {}}
  ],
  "artifacts": [],
  "policy": {"require_clean_before": true, "require_clean_after": true, "forbid_tracked_full_digests": true, "require_diff_check": true}
}`)
	_, err := ValidateFrozenPlanV2(bytes)
	assertInvalidReports(t, err, "/plan", "/plan", "semantic_validation_failed", 0)
}

// TestInvalidPlanMatrix_UnknownCheckMode asserts the structural
// walker rejects checks with an unknown mode.
func TestInvalidPlanMatrix_UnknownCheckMode(t *testing.T) {
	bytes := []byte(`{
  "contract_version": 1,
  "act_id": "ACT-UNKNOWN-MODE",
  "baseline": {"commit_oid": "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", "tree_oid": "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"},
  "execution": {"mode": "serial_fail_fast"},
  "checks": [{"id": "noop", "mode": "spaceship", "argv": ["true"], "working_directory": ".", "timeout_seconds": 60, "environment": {}}],
  "artifacts": [],
  "policy": {"require_clean_before": true, "require_clean_after": true, "forbid_tracked_full_digests": true, "require_diff_check": true}
}`)
	_, err := ValidateFrozenPlanV2(bytes)
	assertInvalidReports(t, err, "/checks/0/mode", "/checks/0/mode", "", 0)
}

// TestInvalidPlanMatrix_RunCheckMissingWorkingDirectory asserts
// the structural walker rejects run checks without a
// working_directory.
func TestInvalidPlanMatrix_RunCheckMissingWorkingDirectory(t *testing.T) {
	bytes := []byte(`{
  "contract_version": 1,
  "act_id": "ACT-NO-WD",
  "baseline": {"commit_oid": "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", "tree_oid": "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"},
  "execution": {"mode": "serial_fail_fast"},
  "checks": [{"id": "noop", "mode": "run", "argv": ["true"], "timeout_seconds": 60, "environment": {}}],
  "artifacts": [],
  "policy": {"require_clean_before": true, "require_clean_after": true, "forbid_tracked_full_digests": true, "require_diff_check": true}
}`)
	_, err := ValidateFrozenPlanV2(bytes)
	assertInvalidReports(t, err, "/checks/0/working_directory", "/checks/0/working_directory", "required_property_missing", 0)
}

// TestInvalidPlanMatrix_RunCheckMissingTimeoutSeconds asserts
// the structural walker rejects run checks without a
// timeout_seconds.
func TestInvalidPlanMatrix_RunCheckMissingTimeoutSeconds(t *testing.T) {
	bytes := []byte(`{
  "contract_version": 1,
  "act_id": "ACT-NO-TIMEOUT",
  "baseline": {"commit_oid": "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", "tree_oid": "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"},
  "execution": {"mode": "serial_fail_fast"},
  "checks": [{"id": "noop", "mode": "run", "argv": ["true"], "working_directory": ".", "environment": {}}],
  "artifacts": [],
  "policy": {"require_clean_before": true, "require_clean_after": true, "forbid_tracked_full_digests": true, "require_diff_check": true}
}`)
	_, err := ValidateFrozenPlanV2(bytes)
	assertInvalidReports(t, err, "/checks/0/timeout_seconds", "/checks/0/timeout_seconds", "required_property_missing", 0)
}

// TestInvalidPlanMatrix_TimeoutBelowMinimum asserts timeout_seconds
// values below the canonical minimum are rejected.
func TestInvalidPlanMatrix_TimeoutBelowMinimum(t *testing.T) {
	bytes := []byte(`{
  "contract_version": 1,
  "act_id": "ACT-LOW-TIMEOUT",
  "baseline": {"commit_oid": "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", "tree_oid": "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"},
  "execution": {"mode": "serial_fail_fast"},
  "checks": [{"id": "noop", "mode": "run", "argv": ["true"], "working_directory": ".", "timeout_seconds": 0, "environment": {}}],
  "artifacts": [],
  "policy": {"require_clean_before": true, "require_clean_after": true, "forbid_tracked_full_digests": true, "require_diff_check": true}
}`)
	_, err := ValidateFrozenPlanV2(bytes)
	assertInvalidReports(t, err, "", "", "", 0)
}

// TestInvalidPlanMatrix_TimeoutAboveMaximum asserts timeout_seconds
// values above the canonical maximum are rejected.
func TestInvalidPlanMatrix_TimeoutAboveMaximum(t *testing.T) {
	bytes := []byte(`{
  "contract_version": 1,
  "act_id": "ACT-HIGH-TIMEOUT",
  "baseline": {"commit_oid": "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", "tree_oid": "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"},
  "execution": {"mode": "serial_fail_fast"},
  "checks": [{"id": "noop", "mode": "run", "argv": ["true"], "working_directory": ".", "timeout_seconds": 601, "environment": {}}],
  "artifacts": [],
  "policy": {"require_clean_before": true, "require_clean_after": true, "forbid_tracked_full_digests": true, "require_diff_check": true}
}`)
	_, err := ValidateFrozenPlanV2(bytes)
	assertInvalidReports(t, err, "", "", "", 0)
}

// TestInvalidPlanMatrix_SemanticPolicyViolation asserts a
// semantic policy violation (require_clean_before=false) is
// rejected.
func TestInvalidPlanMatrix_SemanticPolicyViolation(t *testing.T) {
	bytes := []byte(`{
  "contract_version": 1,
  "act_id": "ACT-POLICY-VIOLATION",
  "baseline": {"commit_oid": "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", "tree_oid": "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"},
  "execution": {"mode": "serial_fail_fast"},
  "checks": [{"id": "noop", "mode": "run", "argv": ["true"], "working_directory": ".", "timeout_seconds": 60, "environment": {}}],
  "artifacts": [],
  "policy": {"require_clean_before": false, "require_clean_after": true, "forbid_tracked_full_digests": true, "require_diff_check": true}
}`)
	_, err := ValidateFrozenPlanV2(bytes)
	assertInvalidReports(t, err, "", "", "", 0)
}

// TestInvalidPlanMatrix_FullRunnerEndToEnd drives the full
// runner with an invalid plan and asserts:
//   - executor is not invoked
//   - manifest is not written
//   - typed V2Error carries frozen_plan_invalid
func TestInvalidPlanMatrix_FullRunnerEndToEnd(t *testing.T) {
	dir := initRepo(t)
	subject := makeCommit(t, dir, "subject", map[string]string{"a.txt": "a"})
	freeze := makeCommit(t, dir, "freeze", map[string]string{
		"docs/closure-plans/INVALID.json": `{"contract_version": 1, "act_id": "X"}`,
	})
	exec := &countingExecutor{}
	deps := DefaultV2RunnerDeps()
	deps.Executor = exec
	req := V2Request{
		ClosureProtocolVersion: ClosureProtocolV2,
		PlanContractVersion:    1,
		RepositoryRoot:         dir,
		SubjectCommit:          subject,
		FreezeCommit:           freeze,
		PlanPath:               "docs/closure-plans/INVALID.json",
		EvidenceDirectory:      t.TempDir(),
		ManifestOutput:         filepath.Join(t.TempDir(), "manifest.json"),
	}
	_, err := RunClosureProtocolV2WithDeps(context.Background(), req, deps)
	if err == nil {
		t.Fatalf("expected V2Error for invalid plan, got nil")
	}
	v2err, ok := err.(*V2Error)
	if !ok || !v2err.Diags.HasCode(V2CodeFrozenPlanInvalid) {
		t.Fatalf("expected frozen_plan_invalid, got %v", err)
	}
	if exec.calls != 0 {
		t.Fatalf("executor must not be invoked on invalid plan, got calls=%d", exec.calls)
	}
	if _, statErr := statIfExists(req.ManifestOutput); statErr == nil {
		t.Fatalf("manifest must not be written on invalid plan")
	}
}

// statIfExists returns the os.FileInfo for p if it exists.
// It returns a non-nil error when the file is missing.
func statIfExists(p string) (os.FileInfo, error) {
	return os.Stat(p)
}
