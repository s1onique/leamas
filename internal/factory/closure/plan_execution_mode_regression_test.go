package closure

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

// TestClosurePlanExecutionModeContractIsRepresentable is the named
// regression test whose intent is to prove that a value required by
// runtime validation can be represented through the strict JSON
// decoder and accepted by the committed JSON Schema.
//
// This test exists because pre-ACT, the canonical execution-mode path
// (`execution.mode`) was technically accepted by the strict decoder,
// but no JSON Schema documented the contract, no semantic presence
// classification distinguished missing from empty, and no shared
// fixture table pinned runtime/schema parity. The test is therefore
// intentionally broad: it covers every presence category that the
// directive ACT demanded, asserts that schema and runtime agree on
// every fixture, and pins the canonical spelling for one valid mode.
func TestClosurePlanExecutionModeContractIsRepresentable(t *testing.T) {
	canonical := newExecutionModeFixture("canonical").
		With("execution", map[string]any{"mode": string(ExecutionModeSerialFailFast)}).
		Bytes()
	plan, err := DecodePlan(canonical)
	if err != nil {
		t.Fatalf("DecodePlan(canonical) error = %v", err)
	}
	if plan.Execution.Mode == nil || *plan.Execution.Mode != ExecutionModeSerialFailFast {
		t.Fatalf("DecodePlan(canonical).Execution.Mode = %v, want %q", plan.Execution.Mode, ExecutionModeSerialFailFast)
	}
	if err := ValidatePlanJSON(canonical); err != nil {
		t.Fatalf("ValidatePlanJSON(canonical) error = %v", err)
	}
}

// TestClosurePlanExecutionModeSchemaRuntimeParity pins the contract
// that every fixture is either accepted by both schema and runtime or
// rejected by both, with the same error category. The fixture table
// is the single source of truth shared by the unit tests, the schema
// parity test, and the CLI subprocess tests.
func TestClosurePlanExecutionModeSchemaRuntimeParity(t *testing.T) {
	cases := executionModeParityFixtures()
	for _, tc := range cases {
		tc := tc
		t.Run(tc.Name, func(t *testing.T) {
			data := tc.Bytes(t)
			runtimeErr := DecodePlanErr(data)
			schemaErr := ValidatePlanJSON(data)
			if tc.WantRuntimeAccept && runtimeErr != nil {
				t.Fatalf("runtime rejected accepted fixture %q: %v", tc.Name, runtimeErr)
			}
			if !tc.WantRuntimeAccept && runtimeErr == nil {
				t.Fatalf("runtime accepted rejected fixture %q", tc.Name)
			}
			if tc.WantSchemaAccept && schemaErr != nil {
				t.Fatalf("schema rejected accepted fixture %q: %v", tc.Name, schemaErr)
			}
			if !tc.WantSchemaAccept && schemaErr == nil {
				t.Fatalf("schema accepted rejected fixture %q", tc.Name)
			}
			if !tc.WantRuntimeAccept && tc.WantDiagnosticContains != "" {
				if !strings.Contains(strings.ToLower(runtimeErr.Error()), strings.ToLower(tc.WantDiagnosticContains)) {
					t.Fatalf("runtime diagnostic %q missing substring %q", runtimeErr, tc.WantDiagnosticContains)
				}
			}
		})
	}
}

// TestClosurePlanExecutionModeRuntimeRejectsEveryAlias makes the
// compatibility policy explicit: every alias observed in the
// pre-change executable — policy.mode, policy.execution,
// policy.execution_mode, and a top-level mode — is rejected by the
// strict decoder. The test pins the directive's "no accidental
// aliases" requirement.
func TestClosurePlanExecutionModeRuntimeRejectsEveryAlias(t *testing.T) {
	aliases := []struct {
		name string
		mut  func(map[string]any)
	}{
		{"policy.mode", func(p map[string]any) {
			p["policy"] = map[string]any{"mode": string(ExecutionModeSerialFailFast), "require_clean_before": true, "require_clean_after": true, "forbid_tracked_full_digests": true, "require_diff_check": true}
		}},
		{"policy.execution", func(p map[string]any) {
			p["policy"] = map[string]any{"execution": string(ExecutionModeSerialFailFast), "require_clean_before": true, "require_clean_after": true, "forbid_tracked_full_digests": true, "require_diff_check": true}
		}},
		{"policy.execution_mode", func(p map[string]any) {
			p["policy"] = map[string]any{"execution_mode": string(ExecutionModeSerialFailFast), "require_clean_before": true, "require_clean_after": true, "forbid_tracked_full_digests": true, "require_diff_check": true}
		}},
		{"top-level.mode", func(p map[string]any) { p["mode"] = string(ExecutionModeSerialFailFast) }},
	}
	for _, alias := range aliases {
		alias := alias
		t.Run(alias.name, func(t *testing.T) {
			raw := newExecutionModeFixture(alias.name).Bytes()
			var p map[string]any
			if err := json.Unmarshal(raw, &p); err != nil {
				t.Fatalf("unmarshal: %v", err)
			}
			alias.mut(p)
			mut, err := json.Marshal(p)
			if err != nil {
				t.Fatalf("marshal: %v", err)
			}
			err = DecodePlanErr(mut)
			if err == nil {
				t.Fatalf("alias %q accepted by runtime", alias.name)
			}
		})
	}
}

// TestValidatePlanReportsMissingExecutionModeDistinctly pins the
// presence-category contract. The strict decoder must distinguish
// between "execution object absent", "execution present but mode
// absent", and "execution present with mode explicitly empty string".
func TestValidatePlanReportsMissingExecutionModeDistinctly(t *testing.T) {
	cases := []struct {
		name string
		mut  func(map[string]any)
		want ExecutionModePresence
	}{
		{
			name: "execution-omitted",
			mut:  func(p map[string]any) { delete(p, "execution") },
			want: ExecutionModeMissing,
		},
		{
			name: "execution-mode-omitted",
			mut:  func(p map[string]any) { p["execution"] = map[string]any{} },
			want: ExecutionModeMissing,
		},
		{
			name: "execution-mode-empty-string",
			mut:  func(p map[string]any) { p["execution"] = map[string]any{"mode": ""} },
			want: ExecutionModePresentEmpty,
		},
		{
			name: "execution-mode-whitespace",
			mut:  func(p map[string]any) { p["execution"] = map[string]any{"mode": "   "} },
			want: ExecutionModePresentWhitespace,
		},
		{
			name: "execution-mode-unknown",
			mut:  func(p map[string]any) { p["execution"] = map[string]any{"mode": "parallel"} },
			want: ExecutionModePresentUnknown,
		},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			raw := newExecutionModeFixture(tc.name).Bytes()
			var p map[string]any
			if err := json.Unmarshal(raw, &p); err != nil {
				t.Fatalf("unmarshal: %v", err)
			}
			tc.mut(p)
			mut, err := json.Marshal(p)
			if err != nil {
				t.Fatalf("marshal: %v", err)
			}
			err = DecodePlanErr(mut)
			if err == nil {
				t.Fatalf("accepted bad fixture %q", tc.name)
			}
			var typed *ExecutionModeError
			if !errors.As(err, &typed) {
				t.Fatalf("error %v is not *ExecutionModeError", err)
			}
			if typed.Presence != tc.want {
				t.Fatalf("Presence = %v, want %v", typed.Presence, tc.want)
			}
		})
	}
}

// DecodePlanErr is a tiny helper so tests can read like a sentence:
// `err := DecodePlanErr(data)`. It is a stable test surface.
func DecodePlanErr(data []byte) error {
	_, err := DecodePlan(data)
	return err
}
