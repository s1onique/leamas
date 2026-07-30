package closure

import (
	"encoding/json"
	"strings"
	"testing"
)

// plan_contract_correction_argv_test.go contains the Phase 4 argv
// item-type tests for
// ACT-LEAMAS-FACTORY-CLOSE-PLAN-CONTRACT-AUTHORITY01-CORRECTION01-
// STRUCTURAL-PARITY-CLOSURE01. Splitting them keeps every file under
// the LLM-friendly 400-line threshold.

// --- Phase 4: argv item descriptors ---

func TestContractArgvStringItemsAccepted(t *testing.T) {
	cases := [][]string{
		{"true"},
		{"sh", "-c", "exit 0"},
		{"echo", "hello world"},
	}
	for _, argv := range cases {
		argv := argv
		t.Run(strings.Join(argv, "_"), func(t *testing.T) {
			plan := canonicalValidPlan()
			argvJSON, _ := json.Marshal(argv)
			data := []byte(strings.Replace(plan, `"argv": ["true"]`, `"argv": `+string(argvJSON), 1))
			result := ValidatePlanStructural(data)
			if !result.Valid {
				t.Fatalf("argv with strings must pass: %v", result.Errors)
			}
		})
	}
}

func TestContractArgvNonStringItemsRejected(t *testing.T) {
	cases := []struct {
		name  string
		value string
		path  string
	}{
		{"integer", `[42]`, "/checks/0/argv/0"},
		{"boolean", `[true]`, "/checks/0/argv/0"},
		{"object", `[{}]`, "/checks/0/argv/0"},
		{"null", `[null]`, "/checks/0/argv/0"},
		{"mixed", `["true", 42]`, "/checks/0/argv/1"},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			plan := canonicalValidPlan()
			data := []byte(strings.Replace(plan, `"argv": ["true"]`, `"argv": `+tc.value, 1))
			result := ValidatePlanStructural(data)
			if result.Valid {
				t.Fatalf("argv with %s item must be rejected", tc.name)
			}
			found := false
			for _, e := range result.Errors {
				if e.Code == PlanCodeInvalidType && e.InstancePath == tc.path {
					found = true
				}
			}
			if !found {
				t.Fatalf("expected invalid_type at %s: %v", tc.path, result.Errors)
			}
		})
	}
}

func TestContractArgvEmptyRejected(t *testing.T) {
	plan := canonicalValidPlan()
	data := []byte(strings.Replace(plan, `"argv": ["true"]`, `"argv": []`, 1))
	result := ValidatePlanStructural(data)
	if result.Valid {
		t.Fatalf("empty argv must be rejected")
	}
}
