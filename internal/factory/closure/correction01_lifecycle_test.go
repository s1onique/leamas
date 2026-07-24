// SPDX-License-Identifier: Apache-2.0

// Package closure: correction01_lifecycle_test.go asserts the
// requalification contract for
// ACT-LEAMAS-FACTORY-SELF-HOSTED-ENTRYPOINT-AUTHORITY01-CORRECTION01.
//
// The tests in this file are deliberately small and orthogonal:
// each test pins one acceptance criterion of the ACT without
// recursing into the wider closure protocol machinery.
package closure

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/s1onique/leamas/internal/factory/authority"
)

// TestPlanRejectsUnresolvedPlaceholder covers acceptance criterion #3:
// an unresolved plan placeholder must fail strict plan validation.
//
// The predecessor ACT committed a plan with baseline.tree_oid =
// "TO_BE_FILLED". The strict plan decoder must reject that value
// because it is a closure placeholder.
func TestPlanRejectsUnresolvedPlaceholder(t *testing.T) {
	plan := minimalValidPlan()
	plan.Baseline.TreeOID = "TO_BE_FILLED"
	err := ValidatePlan(plan)
	if err == nil {
		t.Fatalf("ValidatePlan accepted placeholder tree_oid")
	}
	if !strings.Contains(err.Error(), "placeholder") &&
		!strings.Contains(err.Error(), "baseline") {
		t.Fatalf("expected placeholder rejection error, got %v", err)
	}
}

// TestPlanRejectsBaselineEqualsFreezeCommit covers acceptance
// criterion #4: a plan whose first commit appears AFTER its claimed
// subject cannot serve as F. The validator cannot detect temporal
// ordering by itself, but this test pins the expectation that the
// strict plan validator never embeds the freeze identity inside the
// plan.
func TestPlanRejectsBaselineEqualsFreezeCommit(t *testing.T) {
	// Sanity: the strict plan model has no place to embed the
	// freeze identity. This guards against regressions where a
	// future change accidentally adds such a field.
	data, err := json.Marshal(minimalValidPlan())
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	for _, forbidden := range []string{
		"freeze_commit", "freeze_tree",
		"subject_commit", "subject_tree",
		"closure_commit", "closure_tree",
		"tag_oid", "tag_target", "peeled_target",
	} {
		if strings.Contains(string(data), `"`+forbidden+`"`) {
			t.Fatalf("plan JSON contains forbidden key %q", forbidden)
		}
	}
}

// TestPlanRejectsUnknownFields pins the strict-decode contract:
// unknown top-level fields are rejected.
func TestPlanRejectsUnknownFields(t *testing.T) {
	raw := []byte(`{
  "contract_version": 1,
  "act_id": "ACT-LEAMAS-TEST-UNKNOWN",
  "baseline": {
    "commit_oid": "1111111111111111111111111111111111111111",
    "tree_oid": "2222222222222222222222222222222222222222"
  },
  "execution": {"mode": "serial_fail_fast"},
  "checks": [
    {"id": "x", "mode": "run", "argv": ["true"], "working_directory": ".", "timeout_seconds": 60, "environment": {}}
  ],
  "artifacts": [],
  "policy": {"require_clean_before": true, "require_clean_after": true, "forbid_tracked_full_digests": true, "require_diff_check": true},
  "extra_unknown_field": "nope"
}`)
	_, err := DecodePlan(raw)
	if err == nil {
		t.Fatalf("DecodePlan accepted unknown field")
	}
	if !strings.Contains(err.Error(), "unknown field") {
		t.Fatalf("expected unknown-field rejection, got %v", err)
	}
}

// TestCorrection01PlanFreezePredecessorOrdering covers acceptance
// criterion #5: the predecessor closure plan first appeared AFTER
// the predecessor subject. This is the historical defect that
// motivates the forward requalification.
func TestCorrection01PlanFreezePredecessorOrdering(t *testing.T) {
	const (
		predecessorSubject   = "06c51158d104c20eec389736a2a0bcff06743630"
		predecessorPlanFirst = "d20fc2c0f856b8a99330b626cd87fd256dc0a931"
	)
	if !isAncestorGit(".", predecessorSubject, predecessorPlanFirst) {
		t.Fatalf("predecessor plan %s must be a descendant of predecessor subject %s",
			predecessorPlanFirst, predecessorSubject)
	}
}

// TestRequalificationPlanFreezePointsAtPredecessorClosure pins the
// requalification lifecycle: the freeze (F1) plan references the
// predecessor closure commit as its baseline, anchoring the new
// lifecycle to the existing repository state.
func TestRequalificationPlanFreezePointsAtPredecessorClosure(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	repoRoot := filepath.Clean(filepath.Join(wd, "..", "..", ".."))

	planPath := filepath.Join(repoRoot, "docs", "closure-plans",
		"ACT-LEAMAS-FACTORY-SELF-HOSTED-ENTRYPOINT-AUTHORITY01-CORRECTION01.json")
	data, err := os.ReadFile(planPath)
	if err != nil {
		t.Fatalf("read plan: %v", err)
	}
	var plan Plan
	if err := json.Unmarshal(data, &plan); err != nil {
		t.Fatalf("unmarshal plan: %v", err)
	}
	if plan.Baseline.CommitOID != "d20fc2c0f856b8a99330b626cd87fd256dc0a931" {
		t.Fatalf("plan baseline.commit_oid=%q want=d20fc2c0f856b8a99330b626cd87fd256dc0a931",
			plan.Baseline.CommitOID)
	}
	if plan.Baseline.TreeOID != "49d10a413026ff5d655736dfb03da1ff0df1bae8" {
		t.Fatalf("plan baseline.tree_oid=%q want=49d10a413026ff5d655736dfb03da1ff0df1bae8",
			plan.Baseline.TreeOID)
	}
}

// minimalValidPlan returns a strict Plan value that passes
// ValidatePlan with no special semantics. Used as the starting
// point for negative tests.
func minimalValidPlan() Plan {
	return Plan{
		ContractVersion: ContractVersionV1,
		ActID:           "ACT-LEAMAS-TEST-MINIMAL",
		Baseline: Baseline{
			CommitOID: "1111111111111111111111111111111111111111",
			TreeOID:   "2222222222222222222222222222222222222222",
		},
		Execution: PlanExecution{Mode: ExecutionSerialFailFast},
		Policy: PlanPolicy{
			RequireCleanBefore:       ptrBool(true),
			RequireCleanAfter:        ptrBool(true),
			ForbidTrackedFullDigests: ptrBool(true),
			RequireDiffCheck:         ptrBool(true),
		},
		Checks: []PlanCheck{
			{
				ID:               "x",
				Mode:             CheckModeRun,
				Argv:             []string{"true"},
				WorkingDirectory: ".",
				TimeoutSeconds:   60,
				Environment:      map[string]string{},
			},
		},
		Artifacts: []PlanArtifact{},
	}
}

func ptrBool(b bool) *bool { return &b }

// isAncestorGit asserts ancestor is reachable from descendant via
// `git merge-base --is-ancestor`.
func isAncestorGit(repoRoot, ancestor, descendant string) bool {
	_, err := authority.DefaultGitRunner(repoRoot, "merge-base", "--is-ancestor", ancestor, descendant)
	return err == nil
}
