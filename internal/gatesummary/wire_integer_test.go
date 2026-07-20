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
	t.Run("malformed v1 duration_ms", func(t *testing.T) {
		wi := wireIntegerForTest("invalid")
		wire := V1Summary{
			SchemaVersion: 1,
			GeneratedAt:   "2026-07-19T08:43:26Z",
			OverallStatus: "pass",
			Checks: []V1Check{
				{
					Name:       "test",
					Status:     "pass",
					DurationMs: &wi,
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
			GeneratedAt:   "2026-07-19T08:43:26Z",
			ScopeID:       "TEST",
			ScopeStatus:   "OPEN",
			OverallStatus: "pass",
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
			GeneratedAt:   "2026-07-19T08:43:26Z",
			ScopeID:       "TEST",
			ScopeStatus:   "OPEN",
			OverallStatus: "pass",
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
			GeneratedAt:   "2026-07-19T08:43:26Z",
			ScopeID:       "TEST",
			ScopeStatus:   "OPEN",
			OverallStatus: "pass",
			Checks: []V2Check{
				{
					Name:   "test",
					Status: "pass",
					Total:  &total,
				},
			},
		}
		_, err := projectV2(wire)
		if err == nil {
			t.Error("projectV2: expected error for malformed test_total")
		}
	})
}

// TestMalformedWireIntegerPublicBoundary tests the complete public Normalize
// contract: malformed wire integers produce zero Summary, Err != nil, and
// Success() == false.
func TestMalformedWireIntegerPublicBoundary(t *testing.T) {
	t.Run("malformed v1 duration_ms via Normalize", func(t *testing.T) {
		wi := wireIntegerForTest("invalid")
		doc := Document{
			v1: &V1Summary{
				SchemaVersion: 1,
				GeneratedAt:   "2026-07-19T08:43:26Z",
				OverallStatus: "pass",
				Checks: []V1Check{
					{
						Name:       "test",
						Status:     "pass",
						DurationMs: &wi,
					},
				},
			},
		}
		result := Normalize(doc)
		if result.Success() {
			t.Error("Normalize: expected failure for malformed duration_ms")
		}
		if result.Err == nil {
			t.Error("Normalize: expected non-nil error")
		}
	})

	t.Run("malformed v2 duration_ms via Normalize", func(t *testing.T) {
		doc := Document{
			v2: &V2Summary{
				SchemaVersion: 2,
				GeneratedAt:   "2026-07-19T08:43:26Z",
				ScopeID:       "TEST",
				ScopeStatus:   "OPEN",
				OverallStatus: "pass",
				Checks: []V2Check{
					{
						Name:   "test",
						Status: "pass",
						Extras: V2Extras{
							DurationMs: wireIntegerForTest("invalid"),
						},
					},
				},
			},
		}
		result := Normalize(doc)
		if result.Success() {
			t.Error("Normalize: expected failure for malformed duration_ms")
		}
		if result.Err == nil {
			t.Error("Normalize: expected non-nil error")
		}
	})

	t.Run("malformed v2 exit_code via Normalize", func(t *testing.T) {
		ec := wireIntegerForTest("invalid")
		doc := Document{
			v2: &V2Summary{
				SchemaVersion: 2,
				GeneratedAt:   "2026-07-19T08:43:26Z",
				ScopeID:       "TEST",
				ScopeStatus:   "OPEN",
				OverallStatus: "pass",
				Checks: []V2Check{
					{
						Name:   "test",
						Status: "pass",
						Extras: V2Extras{
							ExitCode: &ec,
						},
					},
				},
			},
		}
		result := Normalize(doc)
		if result.Success() {
			t.Error("Normalize: expected failure for malformed exit_code")
		}
		if result.Err == nil {
			t.Error("Normalize: expected non-nil error")
		}
	})

	t.Run("malformed v2 test_total via Normalize", func(t *testing.T) {
		total := wireIntegerForTest("invalid")
		doc := Document{
			v2: &V2Summary{
				SchemaVersion: 2,
				GeneratedAt:   "2026-07-19T08:43:26Z",
				ScopeID:       "TEST",
				ScopeStatus:   "OPEN",
				OverallStatus: "pass",
				Checks: []V2Check{
					{
						Name:   "test",
						Status: "pass",
						Total:  &total,
					},
				},
			},
		}
		result := Normalize(doc)
		if result.Success() {
			t.Error("Normalize: expected failure for malformed test_total")
		}
		if result.Err == nil {
			t.Error("Normalize: expected non-nil error")
		}
	})
}
