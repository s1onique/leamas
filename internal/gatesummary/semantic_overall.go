package gatesummary

import "fmt"

// deriveOverallStatus computes the aggregate status from check statuses per the v2 spec:
// - fail dominates
// - unavailable dominates pass when no check failed
// - pass when any check passed
// - unavailable otherwise (empty or all skipped)
func deriveOverallStatus(checks []Check) GateStatus {
	hasFail := false
	hasUnavailable := false
	hasPass := false

	for _, c := range checks {
		switch c.Status {
		case GateFail:
			hasFail = true
		case GateUnavailable:
			hasUnavailable = true
		case GatePass:
			hasPass = true
		case GateSkip:
			// skip does not contribute
		}
	}

	if hasFail {
		return GateFail
	}
	if hasUnavailable {
		return GateUnavailable
	}
	if hasPass {
		return GatePass
	}
	return GateUnavailable
}

// validateOverallStatus checks that the producer's claimed overall status matches
// the derived value. Returns diagnostics if there's a mismatch.
func validateOverallStatus(checks []Check, recordedOverall GateStatus) []Diagnostic {
	derived := deriveOverallStatus(checks)
	if derived != recordedOverall {
		return []Diagnostic{{
			Code:     CodeOverallStatusMismatch,
			Path:     "/overall_status",
			Expected: string(derived),
			Observed: string(recordedOverall),
			Message:  fmt.Sprintf("overall_status mismatch: derived %s, recorded %s", derived, recordedOverall),
		}}
	}
	return nil
}

// deriveOverallFromWire computes the derived overall status from wire check statuses.
func deriveOverallFromWire(statuses []string) GateStatus {
	hasFail := false
	hasUnavailable := false
	hasPass := false

	for _, s := range statuses {
		switch s {
		case "fail":
			hasFail = true
		case "unavailable":
			hasUnavailable = true
		case "pass":
			hasPass = true
		case "skip":
			// skip does not contribute
		}
	}

	if hasFail {
		return GateFail
	}
	if hasUnavailable {
		return GateUnavailable
	}
	if hasPass {
		return GatePass
	}
	return GateUnavailable
}
