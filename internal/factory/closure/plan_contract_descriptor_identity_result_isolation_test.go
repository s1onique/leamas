package closure

import "testing"

// plan_contract_descriptor_identity_result_isolation_test.go
// contains the result-isolation proof for the descriptor-identity
// helpers. Each helper returns an independent diagnostic slice;
// mutating one result must not alter a later invocation.

// TestDescriptorResultIsolation proves each helper returning
// diagnostics produces an independent result slice. A mutation
// of one result must not alter a later invocation.
func TestDescriptorResultIsolation(t *testing.T) {
	contract := planContractV1()

	diags1 := validateDescriptorApplicabilityIdentity(contract)
	for i := range diags1 {
		diags1[i].Message = "MUTATED_" + itoa(i)
	}

	diags2 := validateDescriptorApplicabilityIdentity(contract)
	for _, d := range diags2 {
		if d.Message == "MUTATED_0" || d.Message == "MUTATED_1" {
			t.Fatalf("second validator invocation saw mutation from first: %+v", d)
		}
	}

	obs1 := &countingDescriptorValidationObserver{}
	diagsA := validateModeDependentApplicabilityWithObserver(runModeDocument(), contract, obs1)
	for i := range diagsA {
		diagsA[i].Message = "MUTATED_WALK_" + itoa(i)
	}
	obs2 := &countingDescriptorValidationObserver{}
	diagsB := validateModeDependentApplicabilityWithObserver(runModeDocument(), contract, obs2)
	for _, d := range diagsB {
		if d.Message == "MUTATED_WALK_0" || d.Message == "MUTATED_WALK_1" {
			t.Fatalf("walk result saw mutation from previous walk: %+v", d)
		}
	}
	if obs2.calls != 1 {
		t.Fatalf("second walk validator called %d times, want 1", obs2.calls)
	}

	ex1, _ := descriptorExampleWithContract(contract)
	ex1["MUTATED"] = "value"
	ex2, _ := descriptorExampleWithContract(contract)
	if _, present := ex2["MUTATED"]; present {
		t.Fatalf("second example saw mutation from first: %+v", ex2)
	}
}
