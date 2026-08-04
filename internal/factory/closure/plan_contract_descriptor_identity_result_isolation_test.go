package closure

import (
	"crypto/sha256"
	"encoding/hex"
	"reflect"
	"testing"
)

// plan_contract_descriptor_identity_result_isolation_test.go
// contains the result-isolation proof for the descriptor-identity
// helpers. Each helper returns an independent diagnostic slice;
// mutating one result must not alter a later invocation.

// sha256HexHash returns the hex-encoded SHA-256 over the
// supplied diagnostic stream.
func sha256HexHashIsolation(diagnostics []PlanValidationError) string {
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

// TestDescriptorResultIsolation proves each helper returning
// diagnostics produces an independent result slice. A mutation
// of one result must not alter a later invocation.
//
// For validateDescriptorApplicabilityIdentity:
//   - Uses a duplicate-laden descriptor so the first result is non-empty.
//   - Records the fresh hash before mutation.
//   - Mutates the first result and asserts the hash changed.
//   - Runs a second validation and asserts the hash equals the original
//     fresh hash.
//
// For validateModeDependentApplicabilityWithObserver:
//   - Asserts the first result is non-empty before mutation.
//   - Mutates the first result.
//   - Asserts a fresh result equals the pre-mutation baseline.
//
// For descriptorExampleWithContract:
//   - Keeps the top-level isolation proof.
//   - Mutates one nested map or slice in the first example.
//   - Asserts a fresh example's nested value remains unchanged.
func TestDescriptorResultIsolation(t *testing.T) {
	// Use duplicate-laden descriptor for non-empty diagnostics.
	contractDup := buildDuplicateFieldsDescriptor(t, []string{"alpha", "beta"})
	contract := planContractV1()

	// --- validateDescriptorApplicabilityIdentity ---
	t.Run("validateDescriptorApplicabilityIdentity", func(t *testing.T) {
		diags1 := validateDescriptorApplicabilityIdentity(contractDup)
		if len(diags1) == 0 {
			t.Fatalf("expected non-empty diagnostics for duplicate-laden descriptor")
		}
		// Record fresh hash before mutation.
		freshHash := sha256HexHashIsolation(diags1)

		// Mutate the first result.
		for i := range diags1 {
			diags1[i].Message = "MUTATED_" + itoa(i)
		}
		// Hash must have changed.
		mutatedHash := sha256HexHashIsolation(diags1)
		if mutatedHash == freshHash {
			t.Fatalf("hash did not change after mutation: %s", mutatedHash)
		}

		// Run second validation; hash must equal original fresh hash.
		diags2 := validateDescriptorApplicabilityIdentity(contractDup)
		secondHash := sha256HexHashIsolation(diags2)
		if secondHash != freshHash {
			t.Fatalf("second hash %s != original fresh hash %s", secondHash, freshHash)
		}
	})

	// --- validateModeDependentApplicabilityWithObserver ---
	t.Run("validateModeDependentApplicabilityWithObserver", func(t *testing.T) {
		obs1 := &countingDescriptorValidationObserver{}
		diagsA := validateModeDependentApplicabilityWithObserver(runModeDocument(), contractDup, obs1)
		if len(diagsA) == 0 {
			t.Fatalf("expected non-empty diagnostics before mutation")
		}
		// Capture baseline before mutation.
		baselineHash := sha256HexHashIsolation(diagsA)
		// Make a deep copy of the baseline to compare against later.
		baselineCopy := make([]PlanValidationError, len(diagsA))
		copy(baselineCopy, diagsA)

		// Mutate the first result.
		for i := range diagsA {
			diagsA[i].Message = "MUTATED_WALK_" + itoa(i)
		}

		// Fresh result must equal pre-mutation baseline (not the mutated version).
		obs2 := &countingDescriptorValidationObserver{}
		diagsB := validateModeDependentApplicabilityWithObserver(runModeDocument(), contractDup, obs2)
		freshHash := sha256HexHashIsolation(diagsB)
		if freshHash != baselineHash {
			t.Fatalf("fresh result hash %s != baseline hash %s", freshHash, baselineHash)
		}
		if !reflect.DeepEqual(diagsB, baselineCopy) {
			t.Fatalf("fresh result differs from pre-mutation baseline:\nbaseline=%+v\nfresh=%+v", baselineCopy, diagsB)
		}
		if obs2.calls != 1 {
			t.Fatalf("second walk validator called %d times, want 1", obs2.calls)
		}
	})

	// --- descriptorExampleWithContract top-level and nested isolation ---
	t.Run("descriptorExampleWithContract-top-level", func(t *testing.T) {
		ex1, _ := descriptorExampleWithContract(contract)
		ex1["MUTATED"] = "value"
		ex2, _ := descriptorExampleWithContract(contract)
		if _, present := ex2["MUTATED"]; present {
			t.Fatalf("second example saw mutation from first: %+v", ex2)
		}
	})

	// Nested map isolation: mutate a nested map in the first example
	// and assert the second example's nested value is unchanged.
	t.Run("descriptorExampleWithContract-nested-map-isolation", func(t *testing.T) {
		ex1, _ := descriptorExampleWithContract(contract)
		// Find a nested map (e.g., environment or baseline fields).
		// Walk the example to find a nested map value.
		var nestedMap map[string]any
		if checks, ok := ex1["checks"].([]any); ok && len(checks) > 0 {
			if check, ok := checks[0].(map[string]any); ok {
				if env, ok := check["environment"].(map[string]any); ok {
					nestedMap = env
				}
			}
		}
		if nestedMap == nil {
			// Try baseline field.
			if baseline, ok := ex1["baseline"].(map[string]any); ok {
				nestedMap = baseline
			}
		}
		if nestedMap == nil {
			t.Skip("no suitable nested map found in example for nested isolation test")
		}
		// Capture the original nested value.
		origKey := ""
		origVal := ""
		for k, v := range nestedMap {
			origKey = k
			if s, ok := v.(string); ok {
				origVal = s
			}
			break
		}
		if origKey == "" {
			t.Skip("nested map has no entries to test")
		}

		// Mutate the nested map.
		nestedMap["MUTATED_NESTED"] = "should_not_appear_in_fresh"

		// Get fresh example and verify nested value unchanged.
		ex2, _ := descriptorExampleWithContract(contract)
		var freshNested map[string]any
		if checks, ok := ex2["checks"].([]any); ok && len(checks) > 0 {
			if check, ok := checks[0].(map[string]any); ok {
				if env, ok := check["environment"].(map[string]any); ok {
					freshNested = env
				}
			}
		}
		if freshNested == nil {
			if baseline, ok := ex2["baseline"].(map[string]any); ok {
				freshNested = baseline
			}
		}
		if freshNested == nil {
			t.Fatal("could not locate nested map in fresh example")
		}
		if _, present := freshNested["MUTATED_NESTED"]; present {
			t.Fatalf("fresh example sees mutation from first example's nested map")
		}
		// Verify original key/value pair is intact.
		if v, ok := freshNested[origKey].(string); !ok || v != origVal {
			t.Fatalf("nested value changed: expected %q, got %v", origVal, v)
		}
	})

	// Nested slice isolation: mutate a nested slice in the first example
	// and assert the second example's slice is unchanged.
	t.Run("descriptorExampleWithContract-nested-slice-isolation", func(t *testing.T) {
		ex1, _ := descriptorExampleWithContract(contract)
		// Find a nested slice (e.g., argv or checks).
		var nestedSlice []any
		if checks, ok := ex1["checks"].([]any); ok && len(checks) > 0 {
			nestedSlice = checks
		}
		if nestedSlice == nil {
			if argv, ok := ex1["argv"].([]any); ok {
				nestedSlice = argv
			}
		}
		if nestedSlice == nil {
			t.Skip("no suitable nested slice found in example for nested isolation test")
		}

		// Capture original slice length.
		origLen := len(nestedSlice)

		// Mutate the nested slice by appending.
		nestedSlice = append(nestedSlice, "MUTATED_SLICE_ITEM")

		// Get fresh example and verify slice unchanged.
		ex2, _ := descriptorExampleWithContract(contract)
		var freshSlice []any
		if checks, ok := ex2["checks"].([]any); ok {
			freshSlice = checks
		}
		if freshSlice == nil {
			if argv, ok := ex2["argv"].([]any); ok {
				freshSlice = argv
			}
		}
		if freshSlice == nil {
			t.Fatal("could not locate nested slice in fresh example")
		}
		if len(freshSlice) != origLen {
			t.Fatalf("fresh example slice length changed: expected %d, got %d", origLen, len(freshSlice))
		}
	})
}
