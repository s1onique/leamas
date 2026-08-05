package closure

import "strings"

// buildTimeoutPlanBody emits a single-check plan with the
// supplied timeout_seconds literal. It centralises the
// fixture string so every matrix row shares the same
// shape, identical baseline, and identical execution.
func buildTimeoutPlanBody(timeout string, present bool) string {
	timeoutLine := ""
	if present {
		timeoutLine = `"timeout_seconds": ` + timeout + `,`
	}
	return `{
		"contract_version": 1,
		"act_id": "ACT-LEAMAS-FULL-MATRIX",
		"baseline": {
			"commit_oid": "1111111111111111111111111111111111111111",
			"tree_oid": "2222222222222222222222222222222222222222"
		},
		"execution": {"mode": "serial_fail_fast"},
		"checks": [{
			"id": "x",
			"mode": "run",
			"argv": ["true"],
			"working_directory": ".",
			` + timeoutLine + `
			"environment": {}
		}],
		"artifacts": [],
		"policy": {
			"require_clean_before": true,
			"require_clean_after": true,
			"forbid_tracked_full_digests": true,
			"require_diff_check": true
		}
	}`
}

// buildTimeoutPlan is the simplified alias used by the
// huge-integer tests. It assumes present=true.
func buildTimeoutPlan(timeout string) string {
	return buildTimeoutPlanBody(timeout, true)
}

// findTimeoutDiagnostic returns the first diagnostic whose
// instance_path targets timeout_seconds, or nil when no
// such diagnostic is present.
func findTimeoutDiagnostic(diags []PlanValidationError) *PlanValidationError {
	for i := range diags {
		if strings.HasSuffix(diags[i].InstancePath, "/timeout_seconds") ||
			diags[i].PropertyName == "timeout_seconds" {
			return &diags[i]
		}
	}
	return nil
}
