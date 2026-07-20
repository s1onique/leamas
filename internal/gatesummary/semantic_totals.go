package gatesummary

import (
	"fmt"
	"math/big"
)

// validateTestTotals checks that test count arithmetic is valid for v2 checks.
// The decoder already validates partial presence (GS_PARTIAL_TEST_TOTALS).
// This validator checks the arithmetic invariant: total = pass + fail + skip + unavailable.
func validateTestTotals(checks []Check) []Diagnostic {
	var diags []Diagnostic
	for i, c := range checks {
		if c.Totals == nil {
			continue
		}
		// Derive expected total from component counts
		expected := sumTestCounts(c.Totals)
		observed := c.Totals.Total

		// Compare using big.Int arithmetic
		observedBi, ok := observed.BigInt()
		if !ok {
			continue
		}
		if expected.Cmp(observedBi) != 0 {
			diags = append(diags, Diagnostic{
				Code:     CodeTestTotalMismatch,
				Path:     fmt.Sprintf("/checks/%d", i),
				Expected: expected.String(),
				Observed: observed.String(),
				Message:  "test total mismatch",
			})
		}
	}
	return diags
}

// sumTestCounts computes pass + fail + skip + unavailable using big.Int arithmetic.
func sumTestCounts(t *TestTotals) *big.Int {
	sum := new(big.Int)
	if bi, ok := t.Pass.BigInt(); ok {
		sum.Add(sum, bi)
	}
	if bi, ok := t.Fail.BigInt(); ok {
		sum.Add(sum, bi)
	}
	if bi, ok := t.Skip.BigInt(); ok {
		sum.Add(sum, bi)
	}
	if bi, ok := t.Unavailable.BigInt(); ok {
		sum.Add(sum, bi)
	}
	return sum
}

// checkTotalsArithmetic validates totals arithmetic from wire values.
// Returns a diagnostic or nil.
func checkTotalsArithmetic(index int, total, pass, fail, skip, unavailable *WireInteger) Diagnostic {
	if total == nil || pass == nil || fail == nil || skip == nil || unavailable == nil {
		return Diagnostic{} // decoder owns partial presence
	}

	// Compute derived sum
	sum := new(big.Int)
	addToSum(sum, pass)
	addToSum(sum, fail)
	addToSum(sum, skip)
	addToSum(sum, unavailable)

	// Compare
	totalBi, ok := total.BigInt()
	if !ok {
		return Diagnostic{}
	}

	if sum.Cmp(totalBi) != 0 {
		return Diagnostic{
			Code:     CodeTestTotalMismatch,
			Path:     fmt.Sprintf("/checks/%d", index),
			Expected: sum.String(),
			Observed: total.String(),
			Message:  "test total mismatch",
		}
	}
	return Diagnostic{}
}

// addToSum adds a WireInteger to a big.Int.
func addToSum(sum *big.Int, w *WireInteger) {
	if w == nil {
		return
	}
	bi, ok := w.BigInt()
	if !ok {
		return
	}
	sum.Add(sum, bi)
}
