package coverage

import "testing"

// TestCheckThreshold_PassesAtCurrentRatchet verifies the current ratchet (64%) passes at exact threshold.
func TestCheckThreshold_PassesAtCurrentRatchet(t *testing.T) {
	report := &Report{TotalPercent: 64.0}
	threshold := &Threshold{MinTotalPercent: 64.0}
	err := CheckThreshold(report, threshold)
	if err != nil {
		t.Errorf("CheckThreshold() should pass at exactly 64.0, got error: %v", err)
	}
}

// TestCheckThreshold_FailsBelowCurrentRatchet verifies the current ratchet (64%) fails below threshold.
func TestCheckThreshold_FailsBelowCurrentRatchet(t *testing.T) {
	report := &Report{TotalPercent: 63.9}
	threshold := &Threshold{MinTotalPercent: 64.0}
	err := CheckThreshold(report, threshold)
	if err == nil {
		t.Error("CheckThreshold() should fail at 63.9 for 64.0 threshold, got nil error")
	}
	// Verify error message format
	covErr, ok := err.(*Error)
	if !ok {
		t.Fatalf("expected *Error, got %T", err)
	}
	if covErr.Kind != "threshold_fail" {
		t.Errorf("Kind = %q, want %q", covErr.Kind, "threshold_fail")
	}
}
