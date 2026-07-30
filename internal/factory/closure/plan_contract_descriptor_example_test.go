package closure

import (
	"encoding/json"
	"testing"
)

// plan_contract_descriptor_example_test.go proves the canonical
// DescriptorExample() walks every descriptor field correctly:
// the generated example contains non-empty argv, omits reason,
// and exposes a string-valued environment, and it satisfies
// every validator stage (structural, typed, semantic, composed).

// TestDescriptorExampleRegressionShapes asserts the documented
// run-mode example shape: argv is non-empty, reason is absent,
// and environment carries a string-valued entry.
func TestDescriptorExampleRegressionShapes(t *testing.T) {
	example := DescriptorExample()
	checks, ok := example["checks"].([]any)
	if !ok || len(checks) == 0 {
		t.Fatalf("DescriptorExample checks missing or empty: %+v", example)
	}
	item, ok := checks[0].(map[string]any)
	if !ok {
		t.Fatalf("DescriptorExample checks[0] not an object: %+v", checks[0])
	}
	if item["mode"] != CheckModeRun {
		t.Fatalf("DescriptorExample mode = %v, want %q", item["mode"], CheckModeRun)
	}
	if _, present := item["reason"]; present {
		t.Fatalf("DescriptorExample must omit reason in run-mode example, got %v", item["reason"])
	}
	argv, ok := item["argv"].([]any)
	if !ok || len(argv) == 0 {
		t.Fatalf("DescriptorExample argv must be non-empty: %v", item["argv"])
	}
	for index, entry := range argv {
		if _, ok := entry.(string); !ok {
			t.Fatalf("DescriptorExample argv[%d] = %v, want string", index, entry)
		}
	}
	env, ok := item["environment"].(map[string]any)
	if !ok || len(env) == 0 {
		t.Fatalf("DescriptorExample environment must be a non-empty object: %v", item["environment"])
	}
	for name, value := range env {
		str, ok := value.(string)
		if !ok {
			t.Fatalf("DescriptorExample environment[%q] = %v (%T), want string", name, value, value)
		}
		if str == "" {
			t.Fatalf("DescriptorExample environment[%q] must be a non-empty string", name)
		}
	}
}

// TestDescriptorExampleRegressionPassesAllStages asserts the
// generated example satisfies structural, typed, semantic, and
// composed validation. The composed result must be Valid.
func TestDescriptorExampleRegressionPassesAllStages(t *testing.T) {
	example := DescriptorExample()
	raw, err := json.Marshal(example)
	if err != nil {
		t.Fatalf("marshal example: %v", err)
	}
	structural := ValidatePlanStructural(raw)
	if !structural.Valid {
		t.Fatalf("DescriptorExample must pass structural validation; errors=%v", structural.Errors)
	}
	composed := ValidatePlanComposed(raw)
	if !composed.Valid {
		t.Fatalf("DescriptorExample must pass composed validation; structural=%v semantic=%v",
			composed.Structural.Errors, composed.SemanticErrors)
	}
	if !composed.Decoded {
		t.Fatalf("DescriptorExample must decode successfully")
	}
	if !composed.SemanticValid {
		t.Fatalf("DescriptorExample must pass semantic validation; errors=%v", composed.SemanticErrors)
	}
}
