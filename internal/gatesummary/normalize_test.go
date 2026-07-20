package gatesummary

import (
	"testing"
)

// RED test: Demonstrate that Normalize does not exist yet (should fail to compile or pass vacuously).
// This test will be replaced by proper unit tests after implementation.
func TestNormalizeExists(t *testing.T) {
	// Basic smoke test that Normalize function exists
	doc := Document{}
	result := Normalize(doc)
	if result.Summary.Valid() {
		t.Error("zero document should not be valid")
	}
}

// TestNormalizationResultSuccess verifies the Success() method.
func TestNormalizationResultSuccess(t *testing.T) {
	tests := []struct {
		name     string
		result   NormalizationResult
		expected bool
	}{
		{
			name:     "zero result",
			result:   NormalizationResult{},
			expected: false,
		},
		{
			name: "valid summary no diagnostics",
			result: NormalizationResult{
				Summary: Summary{SchemaVersion: Version1},
			},
			expected: true,
		},
		{
			name: "summary with diagnostics",
			result: NormalizationResult{
				Summary:     Summary{SchemaVersion: Version1},
				Diagnostics: []Diagnostic{{Code: "GS_TEST"}},
			},
			expected: false,
		},
		{
			name: "error present",
			result: NormalizationResult{
				Err: errTest,
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.result.Success()
			if got != tt.expected {
				t.Errorf("Success() = %v, want %v", got, tt.expected)
			}
		})
	}
}

var errTest = &testError{msg: "test"}

type testError struct {
	msg string
}

func (e *testError) Error() string {
	return e.msg
}
