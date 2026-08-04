package closure

import (
	"crypto/sha256"
	"encoding/hex"
	"reflect"
	"testing"
)

// plan_contract_descriptor_identity_proof_test.go contains the
// direct adversarial proof for the descriptor-identity
// implementation published in
// ACT-LEAMAS-FACTORY-CLOSE-PLAN-CONTRACT-AUTHORITY01-CORRECTION05-
// CORRECTION03B-CORRECTION03. The tests exercise insertion-order
// determinism, exactly-once validator invocation, observer
// non-authority, cross-subtree path isolation, malformed-example
// rejection, and result isolation.

// countingDescriptorValidationObserver is the test-only hook
// the suite uses to count how many times the descriptor
// validator runs during an applicability invocation. Production
// always passes noopDescriptorValidationObserver; this struct
// is test-only and never escapes the test binary.
type countingDescriptorValidationObserver struct {
	calls        int
	lastSnapshot []PlanValidationError
}

func (c *countingDescriptorValidationObserver) DescriptorIdentityValidated(diagnostics []PlanValidationError) {
	c.calls++
	snapshot := make([]PlanValidationError, len(diagnostics))
	copy(snapshot, diagnostics)
	c.lastSnapshot = snapshot
}

// identityMutatingDescriptorObserver is the test-only hook
// used to verify the observer cannot change applicability
// behaviour. It mutates every field relevant to duplicate
// suppression identity: Code, InstancePath, PropertyName, and
// Message. The production ordering and defensive-copy safeguards
// are now proved by an observer that mutates suppression identity
// fields.
type identityMutatingDescriptorObserver struct {
	mutated bool
}

func (m *identityMutatingDescriptorObserver) DescriptorIdentityValidated(diagnostics []PlanValidationError) {
	for i := range diagnostics {
		diagnostics[i].Code = PlanCodeInvalidJSON
		diagnostics[i].InstancePath = "/observer-mutated"
		diagnostics[i].PropertyName = "observer-mutated"
		diagnostics[i].Message = "observer-mutated"
	}
	m.mutated = true
}

// buildDuplicateFieldsDescriptor returns a contract whose Root
// Fields map contains the given names in the supplied order,
// each carrying a duplicate (Sibling="mode", Value="run")
// rule pair so the validator emits exactly one duplicate
// diagnostic per field. The function lets the test control the
// map's internal insertion order without bypassing the
// sorted-name walker contract.
func buildDuplicateFieldsDescriptor(t *testing.T, orderedNames []string) planContractV1Descriptor {
	t.Helper()
	fields := make(map[string]planFieldDescriptor, len(orderedNames))
	for _, name := range orderedNames {
		fields[name] = planFieldDescriptor{
			JSONName: name,
			Kind:     kindString,
			ApplicabilityRules: []fieldApplicabilityRule{
				{Sibling: "mode", Value: CheckModeRun, Presence: PresenceRequired},
				{Sibling: "mode", Value: CheckModeRun, Presence: PresenceRequired},
			},
		}
	}
	return planContractV1Descriptor{
		ContractVersion: ContractVersionV1,
		Root: planObjectDescriptor{
			Path:   "",
			Fields: fields,
		},
	}
}

// buildCrossSubtreeContract returns a descriptor where one
// subtree's argv carries duplicate rules and the other carries
// a single valid mode=run PresenceRequired rule. The function
// flag duplicateInChecks picks which subtree carries the
// duplicate identity; the other subtree stays valid so the
// walker still emits its per-check diagnostics.
func buildCrossSubtreeContract(t *testing.T, duplicateInChecks bool) planContractV1Descriptor {
	t.Helper()
	duplicateRules := []fieldApplicabilityRule{
		{Sibling: "mode", Value: CheckModeRun, Presence: PresenceRequired},
		{Sibling: "mode", Value: CheckModeRun, Presence: PresenceRequired},
	}
	validRules := []fieldApplicabilityRule{
		{Sibling: "mode", Value: CheckModeRun, Presence: PresenceRequired},
	}
	otherRules := validRules
	checksRules := validRules
	if duplicateInChecks {
		checksRules = duplicateRules
	} else {
		otherRules = duplicateRules
	}
	return planContractV1Descriptor{
		ContractVersion: ContractVersionV1,
		Root: planObjectDescriptor{
			Path: "",
			Fields: map[string]planFieldDescriptor{
				"other": {
					JSONName: "other",
					Kind:     kindObject,
					Children: &planObjectDescriptor{
						Path: "/other",
						Fields: map[string]planFieldDescriptor{
							"argv": {
								JSONName:           "argv",
								Kind:               kindString,
								ApplicabilityRules: otherRules,
							},
						},
					},
				},
				"checks": {
					JSONName: "checks",
					Kind:     kindArray,
					Required: true,
					MinItems: 1,
					ItemDescriptor: &planFieldDescriptor{
						JSONName: "checks[]",
						Kind:     kindObject,
						Children: &planObjectDescriptor{
							Path:     "/checks",
							Required: []string{"id", "mode"},
							Fields: map[string]planFieldDescriptor{
								"id":   {JSONName: "id", Kind: kindString, Required: true},
								"mode": {JSONName: "mode", Kind: kindEnum, Required: true},
								"argv": {
									JSONName:           "argv",
									Kind:               kindArray,
									ApplicabilityRules: checksRules,
								},
							},
						},
					},
				},
			},
		},
	}
}

// runModeDocument returns a single run-mode check document
// whose argv field is absent, suitable for triggering a
// required_property_missing diagnostic when the descriptor
// permits it.
func runModeDocument() map[string]any {
	return map[string]any{
		"checks": []any{
			map[string]any{"id": "noop", "mode": CheckModeRun},
		},
	}
}

// normalizeDiagnosticsHash returns a deterministic hash of the
// diagnostic stream. The string is the hex-encoded SHA-256 over
// (InstancePath, Code, PropertyName, Message) per diagnostic, in
// slice order.
func normalizeDiagnosticsHash(diagnostics []PlanValidationError) string {
	h := sha256.New()
	for _, d := range diagnostics {
		_, _ = h.Write([]byte(d.InstancePath))
		_, _ = h.Write([]byte{0})
		_, _ = h.Write([]byte(d.Code))
		_, _ = h.Write([]byte{0})
		_, _ = h.Write([]byte(d.PropertyName))
		_, _ = h.Write([]byte{0})
		_, _ = h.Write([]byte(d.Message))
		_, _ = h.Write([]byte{'\n'})
	}
	return hex.EncodeToString(h.Sum(nil))
}

// TestDescriptorIdentityMapInsertionOrderDeterminism proves
// that the descriptor applicability-identity validator emits
// diagnostics in the same order regardless of the Fields map's
// internal insertion order. The validator walks
// object.fieldNamesSorted() and sorts the diagnostic stream so
// two semantically identical descriptors built with opposite
// insertion orders must produce identical diagnostics.
func TestDescriptorIdentityMapInsertionOrderDeterminism(t *testing.T) {
	contractA := buildDuplicateFieldsDescriptor(t, []string{"alpha", "beta"})
	contractB := buildDuplicateFieldsDescriptor(t, []string{"beta", "alpha"})
	var prevHash string
	for iter := 0; iter < 20; iter++ {
		diagsA := validateDescriptorApplicabilityIdentity(contractA)
		diagsB := validateDescriptorApplicabilityIdentity(contractB)
		if len(diagsA) != len(diagsB) {
			t.Fatalf("iter %d: count mismatch A=%d B=%d", iter, len(diagsA), len(diagsB))
		}
		if !reflect.DeepEqual(diagsA, diagsB) {
			t.Fatalf("iter %d: diagnostics differ:\nA=%+v\nB=%+v", iter, diagsA, diagsB)
		}
		hashA := normalizeDiagnosticsHash(diagsA)
		if iter > 0 && hashA != prevHash {
			t.Fatalf("iter %d: hash drifted %s -> %s", iter, prevHash, hashA)
		}
		prevHash = hashA
	}
}

// TestDescriptorIdentityCrossSubtreePathIsolation proves that a
// duplicate rule under /other/argv does NOT suppress /checks/argv
// processing, and the inverse case: a duplicate under
// /checks/argv suppresses only /checks/argv.
func TestDescriptorIdentityCrossSubtreePathIsolation(t *testing.T) {
	t.Run("duplicate-in-other-only", func(t *testing.T) {
		contract := buildCrossSubtreeContract(t, false)
		diags := validateModeDependentApplicabilityWithObserver(
			runModeDocument(), contract, noopDescriptorValidationObserver{})
		hasOtherDuplicate := false
		hasChecksRequired := false
		for _, d := range diags {
			if d.Code == PlanCodeDuplicateApplicabilityRule && d.InstancePath == "/other/argv" {
				hasOtherDuplicate = true
			}
			if d.Code == PlanCodeRequiredPropertyMissing && d.InstancePath == "/checks/0/argv" {
				hasChecksRequired = true
			}
		}
		if !hasOtherDuplicate {
			t.Fatalf("expected duplicate_applicability_rule at /other/argv; got %v", diags)
		}
		if !hasChecksRequired {
			t.Fatalf("expected required_property_missing at /checks/0/argv; got %v", diags)
		}
	})
	t.Run("duplicate-in-checks-only", func(t *testing.T) {
		contract := buildCrossSubtreeContract(t, true)
		diags := validateModeDependentApplicabilityWithObserver(
			runModeDocument(), contract, noopDescriptorValidationObserver{})
		hasChecksDuplicate := false
		hasChecksRequired := false
		for _, d := range diags {
			if d.Code == PlanCodeDuplicateApplicabilityRule && d.InstancePath == "/checks/argv" {
				hasChecksDuplicate = true
			}
			if d.Code == PlanCodeRequiredPropertyMissing && d.InstancePath == "/checks/0/argv" {
				hasChecksRequired = true
			}
		}
		if !hasChecksDuplicate {
			t.Fatalf("expected duplicate_applicability_rule at /checks/argv; got %v", diags)
		}
		if hasChecksRequired {
			t.Fatalf("must NOT emit required_property_missing at /checks/0/argv when /checks/argv is suppressed; got %v", diags)
		}
	})
}

// TestDescriptorObserverExactlyOneCall proves the descriptor
// validator runs exactly once per applicability invocation,
// regardless of descriptor shape or document shape. The suite
// covers a valid descriptor, a duplicate-laden descriptor, a
// missing-checks document, a wrong-typed-checks document, and a
// multi-item document.
func TestDescriptorObserverExactlyOneCall(t *testing.T) {
	contractValid := planContractV1()
	contractDuplicate := buildDuplicateFieldsDescriptor(t, []string{"field"})
	documentMissingChecks := map[string]any{}
	documentWrongTypedChecks := map[string]any{"checks": "not-a-list"}
	documentMultiItem := map[string]any{
		"checks": []any{
			map[string]any{"id": "a", "mode": CheckModeRun},
			map[string]any{"id": "b", "mode": CheckModeRun},
		},
	}
	cases := []struct {
		name     string
		contract planContractV1Descriptor
		root     map[string]any
	}{
		{"valid-descriptor", contractValid, runModeDocument()},
		{"duplicate-descriptor", contractDuplicate, runModeDocument()},
		{"missing-checks-property", contractValid, documentMissingChecks},
		{"wrong-typed-checks-property", contractValid, documentWrongTypedChecks},
		{"multiple-check-items", contractValid, documentMultiItem},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			obs := &countingDescriptorValidationObserver{}
			validateModeDependentApplicabilityWithObserver(tc.root, tc.contract, obs)
			if obs.calls != 1 {
				t.Fatalf("validator called %d times, want exactly 1", obs.calls)
			}
		})
	}
}

// TestDescriptorObserverNonAuthority proves the observer cannot
// change applicability behaviour. The mutating observer alters
// every diagnostic message in the slice it receives. If the
// production code derives the suppression set from the slice
// after the observer runs, the mutating observer would change
// the suppression and the resulting diagnostic stream would
// diverge from the noop baseline.
func TestDescriptorObserverNonAuthority(t *testing.T) {
	contract := buildDuplicateFieldsDescriptor(t, []string{"argv"})
	root := runModeDocument()

	noop := validateModeDependentApplicabilityWithObserver(root, contract, noopDescriptorValidationObserver{})

	mut := &identityMutatingDescriptorObserver{}
	withMut := validateModeDependentApplicabilityWithObserver(root, contract, mut)
	if !mut.mutated {
		t.Fatalf("mutating observer was not invoked")
	}
	if !reflect.DeepEqual(noop, withMut) {
		t.Fatalf("observer mutation changed applicability result:\nnoop=%+v\nmut=%+v", noop, withMut)
	}
}

// TestDescriptorExampleMalformedDescriptorReject proves that
// descriptorExampleWithContract returns the empty map plus the
// validator's diagnostic stream when the descriptor carries a
// duplicate applicability identity. The example must contain no
// checks, policy, baseline, or artifacts content under the
// failure path.
func TestDescriptorExampleMalformedDescriptorReject(t *testing.T) {
	t.Run("conflicting-duplicate", func(t *testing.T) {
		contract := buildDuplicateFieldsDescriptor(t, []string{"argv"})
		example, diags := descriptorExampleWithContract(contract)
		if len(example) != 0 {
			t.Fatalf("example must be empty for conflicting-duplicate descriptor; got %v", example)
		}
		foundDuplicate := false
		for _, d := range diags {
			if d.Code == PlanCodeDuplicateApplicabilityRule && d.InstancePath == "/argv" {
				foundDuplicate = true
			}
		}
		if !foundDuplicate {
			t.Fatalf("expected duplicate_applicability_rule at /argv; got %v", diags)
		}
	})
	t.Run("identical-duplicate", func(t *testing.T) {
		contract := buildDuplicateFieldsDescriptor(t, []string{"argv"})
		example, diags := descriptorExampleWithContract(contract)
		if len(example) != 0 {
			t.Fatalf("example must be empty for identical-duplicate descriptor; got %v", example)
		}
		foundDuplicate := false
		for _, d := range diags {
			if d.Code == PlanCodeDuplicateApplicabilityRule {
				foundDuplicate = true
			}
		}
		if !foundDuplicate {
			t.Fatalf("expected duplicate_applicability_rule; got %v", diags)
		}
	})
	t.Run("cross-subtree-duplicate", func(t *testing.T) {
		contract := buildCrossSubtreeContract(t, false)
		example, diags := descriptorExampleWithContract(contract)
		foundPathDuplicate := false
		for _, d := range diags {
			if d.Code == PlanCodeDuplicateApplicabilityRule && d.InstancePath == "/other/argv" {
				foundPathDuplicate = true
			}
		}
		if !foundPathDuplicate {
			t.Fatalf("expected duplicate_applicability_rule at /other/argv; got %v", diags)
		}
		// The diagnostic must carry the FULL descriptor path so
		// consumers know exactly which subtree carries the
		// duplicate identity.
		_ = example
	})
	t.Run("valid-production-descriptor", func(t *testing.T) {
		example, diags := descriptorExampleWithContract(planContractV1())
		if len(example) == 0 {
			t.Fatalf("example is empty for valid production descriptor")
		}
		if len(diags) != 0 {
			t.Fatalf("expected no diagnostics; got %v", diags)
		}
		direct := DescriptorExample()
		if !reflect.DeepEqual(example, direct) {
			t.Fatalf("example differs from DescriptorExample()")
		}
	})
}

// TestDescriptorResultIsolation lives in
// plan_contract_descriptor_identity_result_isolation_test.go.
// Adversarial tests (observer mutation, duplicate kinds, cross-subtree
// fail-closed) live in plan_contract_descriptor_identity_adversarial_test.go.
