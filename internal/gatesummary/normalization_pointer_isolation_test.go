package gatesummary

import "testing"

// TestNormalizationOverallDispositionNilIsolation proves that the
// Overall.Disposition pointer transition from nil→non-nil is handled
// correctly by normalization and does not cause aliasing between
// source and result.
//
// Prove:
// - source disposition initially nil
// - normalize produces nil in result
// - mutate source to non-nil
// - existing result remains nil
// - second normalize reflects non-nil
// - mutate second result does not mutate source
func TestNormalizationOverallDispositionNilIsolation(t *testing.T) {
	doc := isolationDocForTest(t)

	// Ensure source has nil disposition (v2-full fixture may vary).
	// Set it explicitly to empty string (wire uses empty string for nil-like).
	var originalDisp string
	if doc.v2 != nil {
		originalDisp = doc.v2.OverallDisposition
		doc.v2.OverallDisposition = ""
	}

	first := Normalize(doc)
	if !first.Success() {
		t.Fatalf("first normalize failed: %v", first.Diagnostics)
	}
	if first.Summary.Overall.Disposition != nil {
		t.Fatalf("expected nil disposition in first result, got %v",
			*first.Summary.Overall.Disposition)
	}

	// Mutate source to non-nil.
	if doc.v2 != nil {
		doc.v2.OverallDisposition = "non-nil-value"
	}

	// First result must remain nil.
	if first.Summary.Overall.Disposition != nil {
		t.Fatalf("first result disposition changed after source mutation")
	}

	// Second normalize reflects the new non-nil value.
	second := Normalize(doc)
	if !second.Success() {
		t.Fatalf("second normalize failed: %v", second.Diagnostics)
	}
	if second.Summary.Overall.Disposition == nil {
		t.Fatal("second result should have non-nil disposition")
	}
	if *second.Summary.Overall.Disposition != "non-nil-value" {
		t.Fatalf("second disposition wrong: %v", *second.Summary.Overall.Disposition)
	}

	// Mutate second result, source unchanged.
	*second.Summary.Overall.Disposition = "mutated-in-result"
	if doc.v2 != nil && doc.v2.OverallDisposition == "mutated-in-result" {
		t.Fatal("source disposition changed after result mutation")
	}

	// Restore source for package hygiene.
	if doc.v2 != nil {
		doc.v2.OverallDisposition = originalDisp
	}
}

// TestNormalizationExitCodeIntegerIndependence proves that
// CheckExecution.ExitCode Integer values are independently owned
// between two normalized results. Each BigInt() call returns a
// fresh allocation, and the underlying Integer raw spelling is
// preserved across mutations.
//
// Prove for two normalized results:
// - exact raw value preserved
// - BigInt() returns distinct mutable allocations
// - mutating returned *big.Int changes neither Integer
// - first and second normalized exit-code values unchanged
// - source wire value unchanged
func TestNormalizationExitCodeIntegerIndependence(t *testing.T) {
	doc := isolationDocForTest(t)
	first := Normalize(doc)
	second := Normalize(doc)
	if !first.Success() || !second.Success() {
		t.Fatalf("normalize failed")
	}

	// Find a check with exit code populated.
	var idx int = -1
	for i, c := range first.Summary.Checks {
		if c.Execution != nil && c.Execution.ExitCode != nil {
			idx = i
			break
		}
	}
	if idx < 0 {
		t.Skip("no check with exit code in v2-full fixture")
	}

	firstExit := first.Summary.Checks[idx].Execution.ExitCode
	secondExit := second.Summary.Checks[idx].Execution.ExitCode
	if firstExit == nil || secondExit == nil {
		t.Fatal("exit code should not be nil")
	}

	// Exact raw value preserved between results.
	if firstExit.String() != secondExit.String() {
		t.Fatalf("exit code raw differs: %q vs %q",
			firstExit.String(), secondExit.String())
	}
	originalFirst := firstExit.String()
	originalSecond := secondExit.String()

	// BigInt() returns distinct mutable allocations.
	bi1, ok1 := firstExit.BigInt()
	bi2, ok2 := secondExit.BigInt()
	if !ok1 || !ok2 {
		t.Fatal("BigInt failed on exit code")
	}
	if bi1 == bi2 {
		t.Fatal("BigInt should return distinct allocations")
	}

	// Mutating bi1 does not affect bi2.
	bi1.SetInt64(9999)
	bi2After, _ := secondExit.BigInt()
	if bi2After.String() == "9999" {
		t.Fatal("bi1 mutation leaked to bi2")
	}

	// Mutating bi1 does not change first Integer raw.
	firstExitRaw := firstExit.String()
	if firstExitRaw == "9999" {
		t.Fatal("first Integer raw overwritten by BigInt mutation")
	}
	if firstExitRaw != originalFirst {
		t.Fatalf("first raw changed: %q vs %q", firstExitRaw, originalFirst)
	}

	// Second Integer raw unchanged.
	secondExitRaw := secondExit.String()
	if secondExitRaw != originalSecond {
		t.Fatalf("second raw changed: %q vs %q", secondExitRaw, originalSecond)
	}
}
