package gatesummary

import (
	"fmt"
	"math/big"
)

// validateExitCodes checks the frozen status/exit-code relationship for v2 checks.
// Returns diagnostics for each violation found. Collects all violations in one pass.
func validateExitCodes(checks []Check) []Diagnostic {
	var diags []Diagnostic
	for i, c := range checks {
		if c.Execution == nil || c.Execution.ExitCode == nil {
			// No execution or no exit code - check if that's valid
			switch c.Status {
			case GateSkip, GateUnavailable:
				// null exit code is valid for skip/unavailable
				continue
			case GatePass:
				// null exit code is invalid for pass
				diags = append(diags, Diagnostic{
					Code:     CodePassExitCodeMismatch,
					Path:     fmt.Sprintf("/checks/%d/extras/exit_code", i),
					Expected: "0",
					Observed: "null",
					Message:  "pass requires exit_code 0",
				})
			case GateFail:
				// null exit code is valid for fail (spawn/setup/timeout/infrastructure failure)
				continue
			}
			continue
		}

		// Has exit code - check the relationship
		ec := c.Execution.ExitCode
		sign := ec.Sign()

		switch c.Status {
		case GatePass:
			if sign != 0 {
				diags = append(diags, Diagnostic{
					Code:     CodePassExitCodeMismatch,
					Path:     fmt.Sprintf("/checks/%d/extras/exit_code", i),
					Expected: "0",
					Observed: ec.String(),
					Message:  "pass requires exit_code 0",
				})
			}
		case GateFail:
			if sign == 0 {
				diags = append(diags, Diagnostic{
					Code:     CodeFailExitCodeMismatch,
					Path:     fmt.Sprintf("/checks/%d/extras/exit_code", i),
					Expected: "non-zero integer or null",
					Observed: ec.String(),
					Message:  "fail requires non-zero exit_code or null",
				})
			}
		case GateSkip:
			diags = append(diags, Diagnostic{
				Code:     CodeSkipExitCodeMismatch,
				Path:     fmt.Sprintf("/checks/%d/extras/exit_code", i),
				Expected: "null",
				Observed: ec.String(),
				Message:  "skip requires exit_code null",
			})
		case GateUnavailable:
			diags = append(diags, Diagnostic{
				Code:     CodeUnavailExitCodeMismatch,
				Path:     fmt.Sprintf("/checks/%d/extras/exit_code", i),
				Expected: "null",
				Observed: ec.String(),
				Message:  "unavailable requires exit_code null",
			})
		}
	}
	return diags
}

// checkExitCodeForWire validates exit code for a single wire check.
// Returns a diagnostic or nil.
func checkExitCodeForWire(index int, status string, exitCode *WireInteger) Diagnostic {
	if exitCode == nil {
		// null exit code
		switch status {
		case "pass":
			return Diagnostic{
				Code:     CodePassExitCodeMismatch,
				Path:     fmt.Sprintf("/checks/%d/extras/exit_code", index),
				Expected: "0",
				Observed: "null",
				Message:  "pass requires exit_code 0",
			}
		case "fail", "skip", "unavailable":
			return Diagnostic{} // valid
		}
		return Diagnostic{}
	}

	// Has exit code - check the relationship
	bi, ok := exitCode.BigInt()
	if !ok {
		return Diagnostic{} // decoder should have caught this
	}

	switch status {
	case "pass":
		if bi.Sign() != 0 {
			return Diagnostic{
				Code:     CodePassExitCodeMismatch,
				Path:     fmt.Sprintf("/checks/%d/extras/exit_code", index),
				Expected: "0",
				Observed: exitCode.String(),
				Message:  "pass requires exit_code 0",
			}
		}
	case "fail":
		if bi.Sign() == 0 {
			return Diagnostic{
				Code:     CodeFailExitCodeMismatch,
				Path:     fmt.Sprintf("/checks/%d/extras/exit_code", index),
				Expected: "non-zero integer or null",
				Observed: exitCode.String(),
				Message:  "fail requires non-zero exit_code or null",
			}
		}
	case "skip":
		return Diagnostic{
			Code:     CodeSkipExitCodeMismatch,
			Path:     fmt.Sprintf("/checks/%d/extras/exit_code", index),
			Expected: "null",
			Observed: exitCode.String(),
			Message:  "skip requires exit_code null",
		}
	case "unavailable":
		return Diagnostic{
			Code:     CodeUnavailExitCodeMismatch,
			Path:     fmt.Sprintf("/checks/%d/extras/exit_code", index),
			Expected: "null",
			Observed: exitCode.String(),
			Message:  "unavailable requires exit_code null",
		}
	}
	return Diagnostic{}
}

// isNonZero returns true if the integer is not zero using big.Int arithmetic.
func isNonZero(w *WireInteger) bool {
	if w == nil {
		return false
	}
	bi, ok := w.BigInt()
	if !ok {
		return false
	}
	return bi.Sign() != 0
}

// bigIntSign returns the sign of a WireInteger using arbitrary precision.
func bigIntSign(w *WireInteger) int {
	if w == nil {
		return 0
	}
	bi, ok := w.BigInt()
	if !ok {
		return 0
	}
	return bi.Sign()
}

// newBigIntFromWire creates a new big.Int from a WireInteger.
func newBigIntFromWire(w *WireInteger) *big.Int {
	if w == nil {
		return new(big.Int)
	}
	bi, ok := w.BigInt()
	if !ok {
		return new(big.Int)
	}
	return bi
}
