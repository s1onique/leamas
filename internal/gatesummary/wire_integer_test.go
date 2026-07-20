package gatesummary

import (
	"encoding/json"
	"testing"
)

// wireIntegerForTest creates a WireInteger with arbitrary internal state.
// This is package-private and only for testing malformed values.
func wireIntegerForTest(raw string) WireInteger {
	return WireInteger{raw: json.Number(raw)}
}

// TestMalformedWireIntegerProjection tests that malformed WireInteger values
// fail normalization through the projection layer.
func TestMalformedWireIntegerProjection(t *testing.T) {
	// Test that a non-empty but malformed WireInteger fails projection.
	// We construct a WireInteger directly to bypass JSON validation.
	
	t.Run("malformed v1 duration_ms", func(t *testing.T) {
		wi := wireIntegerForTest("invalid")
		wire := V1Summary{
			SchemaVersion:  1,
			GeneratedAt:    "2026-07-19T08:43:26Z",
			OverallStatus:  "pass",
			Checks: []V1Check{
				{
					Name:        "test",
					Status:      "pass",
					DurationMs:  &wi,
				},
			},
		}
		_, err := projectV1(wire)
		if err == nil {
			t.Error("projectV1: expected error for malformed duration_ms")
		}
	})

	t.Run("malformed v2 duration_ms", func(t *testing.T) {
		wire := V2Summary{
			SchemaVersion: 2,
			GeneratedAt:    "2026-07-19T08:43:26Z",
			ScopeID:        "TEST",
			ScopeStatus:    "OPEN",
			OverallStatus:  "pass",
			Checks: []V2Check{
				{
					Name:   "test",
					Status: "pass",
					Extras: V2Extras{
						DurationMs: wireIntegerForTest("invalid"),
					},
				},
			},
		}
		_, err := projectV2(wire)
		if err == nil {
			t.Error("projectV2: expected error for malformed duration_ms")
		}
	})

	t.Run("malformed v2 exit_code", func(t *testing.T) {
		ec := wireIntegerForTest("invalid")
		wire := V2Summary{
			SchemaVersion: 2,
			GeneratedAt:    "2026-07-19T08:43:26Z",
			ScopeID:        "TEST",
			ScopeStatus:    "OPEN",
			OverallStatus:  "pass",
			Checks: []V2Check{
				{
					Name:   "test",
					Status: "pass",
					Extras: V2Extras{
						ExitCode: &ec,
					},
				},
			},
		}
		_, err := projectV2(wire)
		if err == nil {
			t.Error("projectV2: expected error for malformed exit_code")
		}
	})

	t.Run("malformed v2 test_total", func(t *testing.T) {
		total := wireIntegerForTest("invalid")
		wire := V2Summary{
			SchemaVersion: 2,
			GeneratedAt:    "2026-07-19T08:43:26Z",
			ScopeID:        "TEST",
			ScopeStatus:    "OPEN",
			OverallStatus:  "pass",
			Checks: []V2Check{
				{
					Name:  "test",
					Status: "pass",
					Total: &total,
				},
			},
		}
		_, err := projectV2(wire)
		if err == nil {
			t.Error("projectV2: expected error for malformed test_total")
		}
	})
}
