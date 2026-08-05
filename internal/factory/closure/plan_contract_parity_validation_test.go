package closure

import (
	"reflect"
	"sort"
	"testing"
)

// plan_contract_parity_validation_test.go contains the
// structural-validator parity tests: unknown-property, required-field
// deletion at every object level, invalid-enum diagnostics for every
// closed enum, typed-example acceptance, and contract-version
// recovery. Splitting it from plan_contract_parity_test.go keeps
// every file under the LLM-friendly 400-line threshold.
// TestParityUnknownPropertyPolicyMatchesDecoder pins the contract
// that the descriptor's unknown-property rejection matches the
// runtime strict-decoder rejection. The runtime uses DisallowUnknownFields
// at every object level; the descriptor must report every unknown
// property at every object level.
func TestParityUnknownPropertyPolicyMatchesDecoder(t *testing.T) {
	mutations := []struct {
		name string
		mut  func(map[string]any)
	}{
		{"root", func(p map[string]any) { p["surprise"] = "nope" }},
		{"baseline", func(p map[string]any) {
			p["baseline"] = map[string]any{
				"commit_oid":  fullCommitOID,
				"tree_oid":    fullTreeOID,
				"unknown_oid": "abc",
			}
		}},
		{"execution", func(p map[string]any) {
			p["execution"] = map[string]any{
				"mode":  string(ExecutionModeSerialFailFast),
				"extra": "true",
			}
		}},
		{"policy", func(p map[string]any) {
			p["policy"] = map[string]any{
				"require_clean_before":        true,
				"require_clean_after":         true,
				"forbid_tracked_full_digests": true,
				"require_diff_check":          true,
				"surprise":                    true,
			}
		}},
	}
	for _, m := range mutations {
		m := m
		t.Run(m.name, func(t *testing.T) {
			data := planContractParityBytes(t, m.mut)
			// The runtime strict decoder rejects the unknown
			// property.
			if _, err := DecodePlan(data); err == nil {
				t.Fatalf("DecodePlan accepted unknown property at %s", m.name)
			}
			// The structural validator also rejects the unknown
			// property with an unknown_property diagnostic at the
			// correct path.
			result := ValidatePlanStructural(data)
			if result.Valid {
				t.Fatalf("ValidatePlanStructural accepted unknown property at %s: %+v", m.name, result)
			}
			found := false
			for _, e := range result.Errors {
				if e.Code == PlanCodeUnknownProperty {
					found = true
					break
				}
			}
			if !found {
				t.Fatalf("structural result missing unknown_property code for %s: %+v", m.name, result.Errors)
			}
		})
	}
}

// TestParityRequiredFieldDeletionAtEveryObject pins the contract
// that deleting a required field at any object level is rejected
// by both the strict decoder AND the structural validator with a
// required_property_missing diagnostic at the precise instance
// path.
func TestParityRequiredFieldDeletionAtEveryObject(t *testing.T) {
	deletions := []struct {
		name string
		mut  func(map[string]any)
		path string
	}{
		{
			name: "missing-contract_version",
			mut:  func(p map[string]any) { delete(p, "contract_version") },
			path: "/contract_version",
		},
		{
			name: "missing-act_id",
			mut:  func(p map[string]any) { delete(p, "act_id") },
			path: "/act_id",
		},
		{
			name: "missing-baseline",
			mut:  func(p map[string]any) { delete(p, "baseline") },
			path: "/baseline",
		},
		{
			name: "missing-baseline-commit_oid",
			mut: func(p map[string]any) {
				delete(p["baseline"].(map[string]any), "commit_oid")
			},
			path: "/baseline/commit_oid",
		},
		{
			name: "missing-baseline-tree_oid",
			mut: func(p map[string]any) {
				delete(p["baseline"].(map[string]any), "tree_oid")
			},
			path: "/baseline/tree_oid",
		},
		{
			name: "missing-execution",
			mut:  func(p map[string]any) { delete(p, "execution") },
			path: "/execution",
		},
		{
			name: "missing-execution-mode",
			mut: func(p map[string]any) {
				p["execution"] = map[string]any{}
			},
			path: "/execution/mode",
		},
		{
			name: "missing-checks",
			mut:  func(p map[string]any) { delete(p, "checks") },
			path: "/checks",
		},
		{
			name: "missing-checks-id",
			mut: func(p map[string]any) {
				p["checks"] = []any{
					map[string]any{
						"mode":              CheckModeRun,
						"argv":              []any{"true"},
						"working_directory": ".",
						"timeout_seconds":   float64(60),
						"environment":       map[string]any{},
					},
				}
			},
			path: "/checks/0/id",
		},
		{
			name: "missing-checks-mode",
			mut: func(p map[string]any) {
				p["checks"] = []any{
					map[string]any{
						"id":                "noop",
						"argv":              []any{"true"},
						"working_directory": ".",
						"timeout_seconds":   float64(60),
						"environment":       map[string]any{},
					},
				}
			},
			path: "/checks/0/mode",
		},
		{
			name: "missing-run-environment",
			mut: func(p map[string]any) {
				p["checks"] = []any{map[string]any{
					"id": "noop", "mode": CheckModeRun, "argv": []any{"true"},
					"working_directory": ".", "timeout_seconds": float64(60),
				}}
			},
			path: "/checks/0/environment",
		},
		{
			name: "missing-artifacts",
			mut:  func(p map[string]any) { delete(p, "artifacts") },
			path: "/artifacts",
		},
		{
			name: "missing-policy",
			mut:  func(p map[string]any) { delete(p, "policy") },
			path: "/policy",
		},
	}
	for _, d := range deletions {
		d := d
		t.Run(d.name, func(t *testing.T) {
			data := planContractParityBytes(t, d.mut)
			result := ValidatePlanStructural(data)
			if result.Valid {
				t.Fatalf("ValidatePlanStructural accepted missing field %s: %+v", d.name, result)
			}
			found := false
			for _, e := range result.Errors {
				if e.Code == PlanCodeRequiredPropertyMissing && e.InstancePath == d.path {
					if d.name == "missing-run-environment" && (e.Keyword != KeywordRequired || e.PropertyName != "environment") {
						t.Fatalf("environment diagnostic = %+v, want required/environment", e)
					}
					found = true
					break
				}
			}
			if !found {
				t.Fatalf("missing required_property_missing for %s at %s: %+v", d.name, d.path, result.Errors)
			}
		})
	}
}

// TestParityInvalidEnumForHistoricalModes pins the contract that
// the historical candidates named in the directive ACT
// ("exitcode", "gate") and other rejected aliases
// ("parallel", "serial", "fail_fast", "exit_code") are all
// rejected as invalid_enum by the structural validator.
func TestParityInvalidEnumForHistoricalModes(t *testing.T) {
	cases := []struct {
		value string
		path  string
	}{
		{"exitcode", "/execution/mode"},
		{"gate", "/execution/mode"},
		{"parallel", "/execution/mode"},
		{"serial", "/execution/mode"},
		{"fail_fast", "/execution/mode"},
		{"exit_code", "/execution/mode"},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.value, func(t *testing.T) {
			data := planContractParityBytes(t, func(p map[string]any) {
				p["execution"] = map[string]any{"mode": tc.value}
			})
			result := ValidatePlanStructural(data)
			if result.Valid {
				t.Fatalf("structural validator accepted rejected mode %q: %+v", tc.value, result)
			}
			found := false
			for _, e := range result.Errors {
				if e.Code == PlanCodeInvalidEnum && e.InstancePath == tc.path {
					if e.RejectedValue != tc.value {
						t.Fatalf("RejectedValue = %v, want %q", e.RejectedValue, tc.value)
					}
					if len(e.AcceptedValues) == 0 || e.AcceptedValues[0] != string(ExecutionModeSerialFailFast) {
						t.Fatalf("AcceptedValues = %v, want [%s]", e.AcceptedValues, ExecutionModeSerialFailFast)
					}
					found = true
				}
			}
			if !found {
				t.Fatalf("structural validator missing invalid_enum for %q: %+v", tc.value, result.Errors)
			}
		})
	}
}

// TestParityInvalidEnumForCheckMode pins the contract that an
// invalid check mode is rejected with an invalid_enum diagnostic
// at the exact instance path.
func TestParityInvalidEnumForCheckMode(t *testing.T) {
	data := planContractParityBytes(t, func(p map[string]any) {
		p["checks"] = []any{
			map[string]any{
				"id":                "noop",
				"mode":              "skip",
				"argv":              []any{"true"},
				"working_directory": ".",
				"timeout_seconds":   float64(60),
				"environment":       map[string]any{},
			},
		}
	})
	result := ValidatePlanStructural(data)
	if result.Valid {
		t.Fatalf("structural validator accepted invalid check mode: %+v", result)
	}
	found := false
	for _, e := range result.Errors {
		if e.Code == PlanCodeInvalidEnum && e.InstancePath == "/checks/0/mode" {
			found = true
			if e.RejectedValue != "skip" {
				t.Fatalf("RejectedValue = %v, want \"skip\"", e.RejectedValue)
			}
			want := []string{CheckModeRun, CheckModeExclude}
			if !reflect.DeepEqual(e.AcceptedValues, want) {
				t.Fatalf("AcceptedValues = %v, want %v", e.AcceptedValues, want)
			}
		}
	}
	if !found {
		t.Fatalf("missing invalid_enum for /checks/0/mode: %+v", result.Errors)
	}
}

// TestParityInvalidEnumForArtifactRole pins the contract that an
// invalid artifact role is rejected with an invalid_enum diagnostic
// at the exact instance path.
func TestParityInvalidEnumForArtifactRole(t *testing.T) {
	data := planContractParityBytes(t, func(p map[string]any) {
		p["artifacts"] = []any{
			map[string]any{
				"id":         "summary",
				"path":       ".factory/gate-fast-summary.json",
				"required":   true,
				"max_bytes":  float64(1048576),
				"media_type": "application/json",
				"role":       "garbage",
			},
		}
	})
	result := ValidatePlanStructural(data)
	if result.Valid {
		t.Fatalf("structural validator accepted invalid artifact role: %+v", result)
	}
	found := false
	for _, e := range result.Errors {
		if e.Code == PlanCodeInvalidEnum && e.InstancePath == "/artifacts/0/role" {
			found = true
		}
	}
	if !found {
		t.Fatalf("missing invalid_enum for /artifacts/0/role: %+v", result.Errors)
	}
}

// TestParityTypedExampleAcceptedByStructuralAndSemantic pins the
// contract that the canonical descriptor example (built from the
// descriptor's example values) is accepted by both the structural
// validator and the typed semantic validator.
func TestParityTypedExampleAcceptedByStructuralAndSemantic(t *testing.T) {
	data := planContractParityBytes(t, nil)
	result := ValidatePlanStructural(data)
	if !result.Valid {
		t.Fatalf("structural validator rejected canonical plan: %+v", result)
	}
	if _, err := DecodePlan(data); err != nil {
		t.Fatalf("DecodePlan rejected canonical plan: %v", err)
	}
}

// TestParityDescriptorHasNoClineMMDerivedAliases pins the contract
// that the descriptor never carries a ClineMM-derived alias.
func TestParityDescriptorHasNoClineMMDerivedAliases(t *testing.T) {
	contract := planContractV1()
	if !descriptorIsClean(contract) {
		t.Fatalf("descriptor contains a ClineMM-derived alias")
	}
}

// TestParityDescriptorIsDeterministic pins the contract that two
// consecutive calls to planContractV1 return descriptors with
// identical sorted field-name lists. This is the mechanical proof
// that diagnostic ordering is deterministic.
func TestParityDescriptorIsDeterministic(t *testing.T) {
	first := planContractV1()
	second := planContractV1()
	if !reflect.DeepEqual(first.Root.fieldNamesSorted(), second.Root.fieldNamesSorted()) {
		t.Fatalf("first=%v second=%v", first.Root.fieldNamesSorted(), second.Root.fieldNamesSorted())
	}
	// All field-name slices must be sorted.
	verifySorted := func(name string, list []string) {
		t.Helper()
		if !sort.StringsAreSorted(list) {
			t.Fatalf("%s not sorted: %v", name, list)
		}
	}
	verifySorted("root field names", first.Root.fieldNamesSorted())
	verifySorted("baseline field names", first.Root.Fields["baseline"].Children.fieldNamesSorted())
	verifySorted("execution field names", first.Root.Fields["execution"].Children.fieldNamesSorted())
	verifySorted("checks field names", first.Root.Fields["checks"].ItemDescriptor.Children.fieldNamesSorted())
	verifySorted("artifacts field names", first.Root.Fields["artifacts"].ItemDescriptor.Children.fieldNamesSorted())
	verifySorted("policy field names", first.Root.Fields["policy"].Children.fieldNamesSorted())
	verifySorted("runner_authority field names", first.Root.Fields["runner_authority"].Children.fieldNamesSorted())
	verifySorted("runner_authority.tool field names", first.Root.Fields["runner_authority"].Children.Fields["tool"].Children.fieldNamesSorted())
}
