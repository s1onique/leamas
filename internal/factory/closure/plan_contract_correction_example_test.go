package closure

import (
	"encoding/json"
	"testing"
)

// plan_contract_correction_example_test.go contains the Phase 8
// descriptor-generated example test for
// ACT-LEAMAS-FACTORY-CLOSE-PLAN-CONTRACT-AUTHORITY01-CORRECTION01-
// STRUCTURAL-PARITY-CLOSURE01. Splitting it keeps every file under
// the LLM-friendly 400-line threshold.

// --- Phase 8: descriptor-generated example ---

func TestContractDescriptorExamplePassesAllStages(t *testing.T) {
	example := DescriptorExample()
	// Marshal and re-decode to confirm shape.
	bytes, err := json.Marshal(example)
	if err != nil {
		t.Fatalf("marshal descriptor example: %v", err)
	}
	structural := ValidatePlanStructural(bytes)
	if !structural.Valid {
		t.Fatalf("descriptor example failed structural validation: %v", structural.Errors)
	}
	plan, err := DecodePlan(bytes)
	if err != nil {
		t.Fatalf("descriptor example failed typed decoding: %v", err)
	}
	if err := ValidatePlan(plan); err != nil {
		t.Fatalf("descriptor example failed semantic validation: %v", err)
	}
}
