package closure

import (
	"reflect"
	"testing"
)

// plan_contract_descriptor_identity_adversarial_test.go contains
// adversarial tests for the descriptor-identity implementation.
// These tests prove the observer cannot affect suppression, that
// identical and conflicting duplicates are distinct fixtures,
// and that malformed examples fail closed.

// identicalDuplicateRules returns applicability rules where both
// rules carry the same Presence value (Required + Required).
func identicalDuplicateRules() []fieldApplicabilityRule {
	return []fieldApplicabilityRule{
		{Sibling: "mode", Value: CheckModeRun, Presence: PresenceRequired},
		{Sibling: "mode", Value: CheckModeRun, Presence: PresenceRequired},
	}
}

// conflictingDuplicateRules returns applicability rules where the
// two rules carry different Presence values (Required + Forbidden).
func conflictingDuplicateRules() []fieldApplicabilityRule {
	return []fieldApplicabilityRule{
		{Sibling: "mode", Value: CheckModeRun, Presence: PresenceRequired},
		{Sibling: "mode", Value: CheckModeRun, Presence: PresenceForbidden},
	}
}

// buildIdenticalDuplicateDescriptor returns a contract with identical duplicate rules.
func buildIdenticalDuplicateDescriptor(t *testing.T) planContractV1Descriptor {
	t.Helper()
	return planContractV1Descriptor{
		ContractVersion: ContractVersionV1,
		Root: planObjectDescriptor{
			Path: "",
			Fields: map[string]planFieldDescriptor{
				"argv": {
					JSONName:           "argv",
					Kind:               kindString,
					ApplicabilityRules: identicalDuplicateRules(),
				},
			},
		},
	}
}

// buildConflictingDuplicateDescriptor returns a contract with conflicting duplicate rules.
func buildConflictingDuplicateDescriptor(t *testing.T) planContractV1Descriptor {
	t.Helper()
	return planContractV1Descriptor{
		ContractVersion: ContractVersionV1,
		Root: planObjectDescriptor{
			Path: "",
			Fields: map[string]planFieldDescriptor{
				"argv": {
					JSONName:           "argv",
					Kind:               kindString,
					ApplicabilityRules: conflictingDuplicateRules(),
				},
			},
		},
	}
}

// TestDescriptorObserverIdentityMutation proves the production ordering
// and defensive-copy safeguards by running the same duplicate-laden
// descriptor through both a noop observer and an identity-mutating
// observer.
func TestDescriptorObserverIdentityMutation(t *testing.T) {
	contract := buildCrossSubtreeContract(t, true)
	root := runModeDocument()

	noop := validateModeDependentApplicabilityWithObserver(root, contract, noopDescriptorValidationObserver{})

	mutObs := &identityMutatingDescriptorObserver{}
	withMut := validateModeDependentApplicabilityWithObserver(root, contract, mutObs)
	if !mutObs.mutated {
		t.Fatalf("mutating observer was not invoked")
	}

	if !reflect.DeepEqual(noop, withMut) {
		t.Fatalf("observer mutation changed applicability result:\nnoop=%+v\nmut=%+v", noop, withMut)
	}

	noopHasRequired := false
	withMutHasRequired := false
	for _, d := range noop {
		if d.Code == PlanCodeRequiredPropertyMissing && d.InstancePath == "/checks/0/argv" {
			noopHasRequired = true
		}
	}
	for _, d := range withMut {
		if d.Code == PlanCodeRequiredPropertyMissing && d.InstancePath == "/checks/0/argv" {
			withMutHasRequired = true
		}
	}
	if noopHasRequired != withMutHasRequired {
		t.Fatalf("duplicate suppression differs: noop=%v mut=%v", noopHasRequired, withMutHasRequired)
	}
}

// TestDescriptorDuplicateKinds proves identical and conflicting duplicates are distinct.
func TestDescriptorDuplicateKinds(t *testing.T) {
	t.Run("identical", func(t *testing.T) {
		contract := buildIdenticalDuplicateDescriptor(t)
		example, diags := descriptorExampleWithContract(contract)
		if len(example) != 0 || len(diags) != 1 || diags[0].Code != PlanCodeDuplicateApplicabilityRule {
			t.Fatalf("unexpected result: example=%v diags=%v", example, diags)
		}
	})
	t.Run("conflicting", func(t *testing.T) {
		contract := buildConflictingDuplicateDescriptor(t)
		example, diags := descriptorExampleWithContract(contract)
		if len(example) != 0 || len(diags) != 1 || diags[0].Code != PlanCodeDuplicateApplicabilityRule {
			t.Fatalf("unexpected result: example=%v diags=%v", example, diags)
		}
	})
}

// TestDescriptorExampleCrossSubtreeFailClosed proves malformed examples fail closed.
func TestDescriptorExampleCrossSubtreeFailClosed(t *testing.T) {
	t.Run("other-argv", func(t *testing.T) {
		contract := buildCrossSubtreeContract(t, false)
		example, diags := descriptorExampleWithContract(contract)
		if len(example) != 0 {
			t.Fatalf("example must be empty; got %v", example)
		}
		found := false
		for _, d := range diags {
			if d.Code == PlanCodeDuplicateApplicabilityRule && d.InstancePath == "/other/argv" {
				found = true
			}
		}
		if !found {
			t.Fatalf("expected duplicate at /other/argv; got %v", diags)
		}
	})
	t.Run("checks-argv", func(t *testing.T) {
		contract := buildCrossSubtreeContract(t, true)
		example, diags := descriptorExampleWithContract(contract)
		if len(example) != 0 {
			t.Fatalf("example must be empty; got %v", example)
		}
		found := false
		for _, d := range diags {
			if d.Code == PlanCodeDuplicateApplicabilityRule && d.InstancePath == "/checks/argv" {
				found = true
			}
		}
		if !found {
			t.Fatalf("expected duplicate at /checks/argv; got %v", diags)
		}
	})
}
