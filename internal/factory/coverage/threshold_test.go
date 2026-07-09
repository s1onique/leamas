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

// TestCheckThreshold_ModulePassesAtFloor verifies module passes when at floor.
func TestCheckThreshold_ModulePassesAtFloor(t *testing.T) {
	report := &Report{
		TotalPercent: 64.0,
		Modules: []ModuleSummary{
			{Module: "cmd/leamas", Percent: 50.0},
			{Module: "internal/factory", Percent: 67.0},
		},
	}
	threshold := &Threshold{
		MinTotalPercent: 64.0,
		MinModulePercents: map[string]float64{
			"cmd/leamas":       50.0,
			"internal/factory": 67.0,
		},
	}
	err := CheckThreshold(report, threshold)
	if err != nil {
		t.Errorf("CheckThreshold() should pass when module is at floor, got error: %v", err)
	}
}

// TestCheckThreshold_ModuleFailsBelowFloor verifies module fails when below floor.
func TestCheckThreshold_ModuleFailsBelowFloor(t *testing.T) {
	report := &Report{
		TotalPercent: 64.0,
		Modules: []ModuleSummary{
			{Module: "cmd/leamas", Percent: 49.9},
		},
	}
	threshold := &Threshold{
		MinTotalPercent: 64.0,
		MinModulePercents: map[string]float64{
			"cmd/leamas": 50.0,
		},
	}
	err := CheckThreshold(report, threshold)
	if err == nil {
		t.Error("CheckThreshold() should fail when module is below floor, got nil error")
	}
	covErr, ok := err.(*Error)
	if !ok {
		t.Fatalf("expected *Error, got %T", err)
	}
	if covErr.Kind != "module_threshold_fail" {
		t.Errorf("Kind = %q, want %q", covErr.Kind, "module_threshold_fail")
	}
}

// TestCheckThreshold_ModuleMissingFailsClosed verifies missing module fails closed.
func TestCheckThreshold_ModuleMissingFailsClosed(t *testing.T) {
	report := &Report{
		TotalPercent: 64.0,
		Modules: []ModuleSummary{
			{Module: "internal/factory", Percent: 70.0},
			// internal/web is missing
		},
	}
	threshold := &Threshold{
		MinTotalPercent: 64.0,
		MinModulePercents: map[string]float64{
			"internal/web": 70.0,
		},
	}
	err := CheckThreshold(report, threshold)
	if err == nil {
		t.Error("CheckThreshold() should fail when enforced module is missing, got nil error")
	}
	covErr, ok := err.(*Error)
	if !ok {
		t.Fatalf("expected *Error, got %T", err)
	}
	if covErr.Kind != "module_threshold_fail" {
		t.Errorf("Kind = %q, want %q", covErr.Kind, "module_threshold_fail")
	}
}

// TestCheckThreshold_ModuleFailuresUseDeterministicOrder verifies deterministic order.
func TestCheckThreshold_ModuleFailuresUseDeterministicOrder(t *testing.T) {
	report := &Report{
		TotalPercent: 64.0,
		Modules: []ModuleSummary{
			{Module: "cmd/leamas", Percent: 10.0},
			{Module: "internal/web", Percent: 10.0},
			{Module: "internal/hulk", Percent: 10.0},
			{Module: "internal/factory", Percent: 10.0},
			{Module: "internal/witness", Percent: 10.0},
		},
	}
	threshold := &Threshold{
		MinTotalPercent: 64.0,
		MinModulePercents: map[string]float64{
			"cmd/leamas":       50.0,
			"internal/factory": 67.0,
			"internal/hulk":    90.0,
			"internal/web":     70.0,
			"internal/witness": 80.0,
		},
	}
	err := CheckThreshold(report, threshold)
	if err == nil {
		t.Error("CheckThreshold() should fail when modules are below floor")
	}
	covErr, ok := err.(*Error)
	if !ok {
		t.Fatalf("expected *Error, got %T", err)
	}
	// cmd/leamas should fail first (first in deterministic order)
	if covErr.Message != "module cmd/leamas coverage 10.0% is below minimum 50.0%" {
		t.Errorf("unexpected first failure: %s", covErr.Message)
	}
}

// TestCheckThreshold_TotalFailureStillUsesThresholdFail verifies total failure uses threshold_fail.
func TestCheckThreshold_TotalFailureStillUsesThresholdFail(t *testing.T) {
	report := &Report{TotalPercent: 63.9}
	threshold := &Threshold{
		MinTotalPercent: 64.0,
		MinModulePercents: map[string]float64{
			"cmd/leamas": 50.0,
		},
	}
	err := CheckThreshold(report, threshold)
	if err == nil {
		t.Error("CheckThreshold() should fail for total below threshold")
	}
	covErr, ok := err.(*Error)
	if !ok {
		t.Fatalf("expected *Error, got %T", err)
	}
	// Total failure should use threshold_fail (not module_threshold_fail)
	if covErr.Kind != "threshold_fail" {
		t.Errorf("Kind = %q, want %q for total failure", covErr.Kind, "threshold_fail")
	}
}

// TestDefaultModuleThresholds verifies default module thresholds.
func TestDefaultModuleThresholds(t *testing.T) {
	defaults := DefaultModuleThresholds()

	expected := map[string]float64{
		"cmd/leamas":       50.0,
		"internal/factory": 67.0,
		"internal/hulk":    90.0,
		"internal/web":     70.0,
		"internal/witness": 80.0,
	}

	for module, expectedFloor := range expected {
		floor, exists := defaults[module]
		if !exists {
			t.Errorf("expected module %q in defaults, not found", module)
			continue
		}
		if floor != expectedFloor {
			t.Errorf("module %q floor = %v, want %v", module, floor, expectedFloor)
		}
	}

	if len(defaults) != len(expected) {
		t.Errorf("expected %d modules in defaults, got %d", len(expected), len(defaults))
	}
}

// TestDefaultThreshold verifies default threshold has both total and module values.
func TestDefaultThreshold(t *testing.T) {
	threshold := DefaultThreshold()

	if threshold.MinTotalPercent != 64.0 {
		t.Errorf("MinTotalPercent = %v, want 64.0", threshold.MinTotalPercent)
	}

	if len(threshold.MinModulePercents) == 0 {
		t.Error("MinModulePercents should not be empty")
	}

	// Verify defaults match expected
	expected := DefaultModuleThresholds()
	for module, expectedFloor := range expected {
		floor, exists := threshold.MinModulePercents[module]
		if !exists {
			t.Errorf("expected module %q in MinModulePercents", module)
			continue
		}
		if floor != expectedFloor {
			t.Errorf("module %q floor = %v, want %v", module, floor, expectedFloor)
		}
	}
}
