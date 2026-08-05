package closure

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

// plan_contract_run_execution_fields_schema_parity_test.go owns
// the schema/descriptor/example parity tests for the run-mode
// `working_directory` and `timeout_seconds` fields. Splitting it
// from the main parity suite keeps every file under the
// LLM-friendly 400-line threshold while each test remains
// reviewable in one screen.

func TestRunExecutionFieldsSchemaParity(t *testing.T) {
	schema, err := JSONSchema()
	if err != nil {
		t.Fatalf("JSONSchema() error = %v", err)
	}
	checkItems, ok := schema["properties"].(map[string]any)["checks"].(map[string]any)["items"].(map[string]any)
	if !ok {
		t.Fatalf("schema does not contain /checks items descriptor")
	}
	props, ok := checkItems["properties"].(map[string]any)
	if !ok {
		t.Fatalf("/checks items descriptor has no properties map")
	}

	// working_directory: type=string, minLength=1, pattern=
	// ^[^/]+(/[^/]+)*$, applicability run=required, exclude=forbidden.
	wd, ok := props["working_directory"].(map[string]any)
	if !ok {
		t.Fatalf("/checks[].working_directory missing from schema")
	}
	if wd["type"] != "string" {
		t.Fatalf("working_directory type = %v, want string", wd["type"])
	}
	if v, ok := wd["minLength"].(int); !ok || v != 1 {
		t.Fatalf("working_directory minLength = %v (%T), want int(1)", wd["minLength"], wd["minLength"])
	}
	if wd["pattern"] != `^[^/]+(/[^/]+)*$` {
		t.Fatalf("working_directory pattern = %q, want %q", wd["pattern"], `^[^/]+(/[^/]+)*$`)
	}
	if !applicabilityHas(wd, "mode", CheckModeRun, "required") {
		t.Fatalf("working_directory applicability for mode=run must be required; got %+v", wd["x-applicability"])
	}
	if !applicabilityHas(wd, "mode", CheckModeExclude, "forbidden") {
		t.Fatalf("working_directory applicability for mode=exclude must be forbidden; got %+v", wd["x-applicability"])
	}

	// timeout_seconds: type=integer, minimum=1, maximum=600,
	// applicability run=required, exclude=forbidden.
	ts, ok := props["timeout_seconds"].(map[string]any)
	if !ok {
		t.Fatalf("/checks[].timeout_seconds missing from schema")
	}
	if ts["type"] != "integer" {
		t.Fatalf("timeout_seconds type = %v, want integer", ts["type"])
	}
	if v, ok := ts["minimum"].(int64); !ok || v != 1 {
		t.Fatalf("timeout_seconds minimum = %v (%T), want int(1)", ts["minimum"], ts["minimum"])
	}
	if v, ok := ts["maximum"].(int64); !ok || v != 600 {
		t.Fatalf("timeout_seconds maximum = %v (%T), want int(600)", ts["maximum"], ts["maximum"])
	}
	if !applicabilityHas(ts, "mode", CheckModeRun, "required") {
		t.Fatalf("timeout_seconds applicability for mode=run must be required; got %+v", ts["x-applicability"])
	}
	if !applicabilityHas(ts, "mode", CheckModeExclude, "forbidden") {
		t.Fatalf("timeout_seconds applicability for mode=exclude must be forbidden; got %+v", ts["x-applicability"])
	}
}

// applicabilityHas reports whether the supplied x-applicability
// list contains a (sibling, value, presence) triple.
func applicabilityHas(field map[string]any, sibling, value, presence string) bool {
	rules, ok := field["x-applicability"].([]map[string]any)
	if !ok {
		return false
	}
	for _, r := range rules {
		rs, _ := r["sibling"].(string)
		rv, _ := r["value"].(string)
		rp, _ := r["presence"].(string)
		if rs == sibling && rv == value && rp == presence {
			return true
		}
	}
	return false
}

// TestRunExecutionFieldsDescriptorMapParity proves the legacy
// PlanSchema() map exposes the value-level constraint fields so
// older consumers (e.g. CLI tooling) can detect them without
// using JSONSchema().
func TestRunExecutionFieldsDescriptorMapParity(t *testing.T) {
	schema := PlanSchema()
	root := schema["root"].(map[string]any)
	checks := root["fields"].(map[string]any)["checks"].(map[string]any)["item_descriptor"].(map[string]any)["children"].(map[string]any)["fields"].(map[string]any)
	for _, fieldName := range []string{"working_directory", "timeout_seconds"} {
		field := checks[fieldName].(map[string]any)
		if fieldName == "working_directory" {
			if v, ok := field["min_length"].(int); !ok || v != 1 {
				t.Fatalf("working_directory descriptor map: min_length = %v (%T), want int(1)", field["min_length"], field["min_length"])
			}
		}
		if fieldName == "timeout_seconds" {
			if v, ok := field["minimum"].(int64); !ok || v != 1 {
				t.Fatalf("timeout_seconds descriptor map: minimum = %v (%T), want int64(1)", field["minimum"], field["minimum"])
			}
			if v, ok := field["maximum"].(int64); !ok || v != 600 {
				t.Fatalf("timeout_seconds descriptor map: maximum = %v (%T), want int64(600)", field["maximum"], field["maximum"])
			}
		}
	}
}

// TestRunExecutionFieldsCanonicalExample passes the descriptor's
// generated example through the composed validator. The example
// is the single source the future schema/example CLI command
// emits; it must satisfy the public contract.
func TestRunExecutionFieldsCanonicalExample(t *testing.T) {
	example := DescriptorExample()
	raw, err := json.Marshal(example)
	if err != nil {
		t.Fatalf("marshal canonical example: %v", err)
	}
	composed := ValidatePlanComposed(raw)
	if !composed.Valid {
		t.Fatalf("canonical example must be valid: structural=%+v semantic=%+v",
			composed.Structural.Errors, composed.SemanticErrors)
	}
	// Example must include working_directory and timeout_seconds
	// for the run-mode check so the doctrine is visible to a
	// source-free consumer.
	body, ok := example["checks"].([]any)
	if !ok || len(body) == 0 {
		t.Fatalf("canonical example missing /checks array")
	}
	check := body[0].(map[string]any)
	if check["working_directory"] == nil {
		t.Fatalf("canonical example check missing working_directory")
	}
	if check["timeout_seconds"] == nil {
		t.Fatalf("canonical example check missing timeout_seconds")
	}
	if check["working_directory"] != "." {
		t.Fatalf("canonical example working_directory = %v, want \".\"", check["working_directory"])
	}
	if v, ok := check["timeout_seconds"].(int); !ok || v != 60 {
		t.Fatalf("canonical example timeout_seconds = %v, want 60", check["timeout_seconds"])
	}
}

// TestRunExecutionFieldsMacShapedPlanRegression validates the
// exact shape the ClineMM environment emitted: a run-mode check
// whose argv is "sh -c exit 0" and whose environment is {}, with
// no working_directory or timeout_seconds. The selected doctrine
// is REQUIRED, so this plan must be rejected structurally with
// the exact missing-property diagnostic at the correct path.
func TestRunExecutionFieldsMacShapedPlanRegression(t *testing.T) {
	planJSON := []byte(`{
  "contract_version": 1,
  "act_id": "ACT-LEAMAS-MAC-SHAPED",
  "baseline": {
    "commit_oid": "1111111111111111111111111111111111111111",
    "tree_oid": "2222222222222222222222222222222222222222"
  },
  "execution": {"mode": "serial_fail_fast"},
  "checks": [
    {"id": "bounded_smoke", "mode": "run", "argv": ["sh", "-c", "exit 0"], "environment": {}}
  ],
  "artifacts": [],
  "policy": {
    "require_clean_before": true,
    "require_clean_after": true,
    "forbid_tracked_full_digests": true,
    "require_diff_check": true
  }
}`)
	result := ValidatePlanStructural(planJSON)
	if result.Valid {
		t.Fatalf("Mac-shaped plan must be rejected; got valid")
	}
	wdDiag := findDiagnosticAt(result.Errors, "/checks/0/working_directory")
	if wdDiag == nil {
		t.Fatalf("expected working_directory required_property_missing diagnostic; got %+v", result.Errors)
	}
	if wdDiag.Code != PlanCodeRequiredPropertyMissing || wdDiag.Keyword != KeywordRequired {
		t.Fatalf("Mac-shaped working_directory diagnostic: code=%q keyword=%q; want required_property_missing/required",
			wdDiag.Code, wdDiag.Keyword)
	}
	tsDiag := findDiagnosticAt(result.Errors, "/checks/0/timeout_seconds")
	if tsDiag == nil {
		t.Fatalf("expected timeout_seconds required_property_missing diagnostic; got %+v", result.Errors)
	}
	if tsDiag.Code != PlanCodeRequiredPropertyMissing || tsDiag.Keyword != KeywordRequired {
		t.Fatalf("Mac-shaped timeout_seconds diagnostic: code=%q keyword=%q; want required_property_missing/required",
			tsDiag.Code, tsDiag.Keyword)
	}
	// The diagnostic message MUST identify the field without
	// requiring message parsing.
	if !strings.Contains(wdDiag.Message, "working_directory") {
		t.Fatalf("working_directory message missing field name: %q", wdDiag.Message)
	}
	if !strings.Contains(tsDiag.Message, "timeout_seconds") {
		t.Fatalf("timeout_seconds message missing field name: %q", tsDiag.Message)
	}
}

// TestRunExecutionFieldsDiagnosticEightKeyShape proves every
// diagnostic remains compatible with the fixed eight-key public
// shape: instance_path, schema_path, code, keyword, message,
// rejected_value, accepted_values, property_name. The shape is
// what downstream tooling pins against.
func TestRunExecutionFieldsDiagnosticEightKeyShape(t *testing.T) {
	data := applyRunExecutionParityMutations(t, func(check map[string]any) {
		delete(check, "working_directory")
	})
	result := ValidatePlanStructural(data)
	if result.Valid {
		t.Fatalf("plan with omitted working_directory must be rejected")
	}
	diag := findDiagnosticAt(result.Errors, "/checks/0/working_directory")
	if diag == nil {
		t.Fatalf("expected working_directory diagnostic")
	}
	if reflect.TypeOf(*diag).NumField() != 8 {
		t.Fatalf("PlanValidationError must keep exactly 8 fields, got %d", reflect.TypeOf(*diag).NumField())
	}
}
