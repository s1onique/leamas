package gatesummary

import (
	"path/filepath"
	"strings"
	"testing"
)

// corpusCase is one row of the corpus matrix.
type corpusCase struct {
	fixture       string
	expectedCode  string
	expectSuccess bool
}

func TestCorpusValid(t *testing.T) {
	cases := []corpusCase{
		{"valid/v1-minimal.json", "", true},
		{"valid/v1-full.json", "", true},
		{"valid/v2-minimal.json", "", true},
		{"valid/v2-full.json", "", true},
		{"valid/v2-root-scope.json", "", true},
		{"valid/v2-clinemm-microc3.json", "", true},
		{"valid/v2-leamas-self-hosted.json", "", true},
	}
	for _, tc := range cases {
		t.Run(tc.fixture, func(t *testing.T) {
			data := readFixture(t, filepath.Join("testdata", tc.fixture))
			res := Decode(strings.NewReader(string(data)))
			if tc.expectSuccess && !res.Success() {
				t.Fatalf("expected success, got diags=%v err=%v", res.Diagnostics, res.Err)
			}
		})
	}
}

// TestCorpusStructuralInvalid exercises fixtures rejected at the
// structural decoder pipeline. Semantic-only-invalid
// fixtures decode successfully and are validated by
// TestCorpusSemanticInvalid.
func TestCorpusStructuralInvalid(t *testing.T) {
	cases := []corpusCase{
		{"invalid/v1-unknown-field.json", CodeUnknownField, false},
		{"invalid/v2-missing-schema-version.json", CodeVersionMissing, false},
		{"invalid/v2-schema-version-string.json", CodeInvalidVersionType, false},
		{"invalid/v2-schema-version-decimal.json", CodeInvalidVersionType, false},
		{"invalid/v2-schema-version-zero.json", CodeUnsupportedVersion, false},
		{"invalid/v2-schema-version-negative.json", CodeUnsupportedVersion, false},
		{"invalid/v2-unsupported-version-3.json", CodeUnsupportedVersion, false},
		{"invalid/v2-empty-generated-at.json", CodeInvalidTimestamp, false},
		{"invalid/v2-invalid-timestamp.json", CodeInvalidTimestamp, false},
		{"invalid/v2-missing-execution-head-oid.json", CodeRequiredFieldMissing, false},
		{"invalid/v2-null-execution-head-oid.json", CodeInvalidOID, false},
		{"invalid/v2-uppercase-oid.json", CodeInvalidOID, false},
		{"invalid/v2-partial-test-totals.json", CodePartialTestTotals, false},
		{"invalid/v2-negative-duration.json", CodeInvalidDuration, false},
		{"invalid/v2-invalid-hash.json", CodeInvalidOutputHash, false},
		{"invalid/v2-trailing-second-value.json", CodeTrailingJSON, false},
		{"invalid/v2-unknown-field.json", CodeUnknownField, false},
		{"invalid/v2-bad-status-enum.json", CodeInvalidStatus, false},
		{"invalid/v2-lower-lifecycle.json", CodeInvalidStatus, false},
		{"invalid/v2-truncated.json", CodeMalformedJSON, false},
	}
	for _, tc := range cases {
		t.Run(tc.fixture, func(t *testing.T) {
			data := readFixture(t, filepath.Join("testdata", tc.fixture))
			res := Decode(strings.NewReader(string(data)))
			if res.Success() {
				t.Fatalf("expected reject for %s", tc.fixture)
			}
			var found bool
			for _, d := range res.Diagnostics {
				if d.Code == tc.expectedCode {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("expected code %s in diagnostics, got %+v",
					tc.expectedCode, res.Diagnostics)
			}
		})
	}
}

// TestCorpusSemanticInvalid documents that the decoder accepts
// (returns a typed wire document) for fixtures whose only failures
// are post-schema semantic invariants. NORMALIZATION01 owns those
// rejections.
func TestCorpusSemanticInvalid(t *testing.T) {
	cases := []string{
		"invalid/v2-pass-nonzero-exit.json",
		"invalid/v2-fail-exit-zero.json",
		"invalid/v2-skip-nonnull-exit.json",
		"invalid/v2-unavailable-nonnull-exit.json",
		"invalid/v2-test-total-mismatch.json",
		"invalid/v2-duplicate-check-name.json",
		"invalid/v2-overall-mismatch.json",
		"invalid/v2-scope-closed-dirty-after.json",
	}
	for _, fixture := range cases {
		t.Run(fixture, func(t *testing.T) {
			data := readFixture(t, filepath.Join("testdata", fixture))
			res := Decode(strings.NewReader(string(data)))
			if !res.Success() {
				t.Fatalf("semantic-only invalid %s must decode for handoff: %v",
					fixture, res.Diagnostics)
			}
		})
	}
}

func TestCorpusDuplicateKeys(t *testing.T) {
	cases := []string{
		"duplicate-keys/v2-duplicate-schema-version.json",
		"duplicate-keys/v2-duplicate-top-level-field.json",
		"duplicate-keys/v2-duplicate-nested-field.json",
	}
	for _, fixture := range cases {
		t.Run(fixture, func(t *testing.T) {
			data := readFixture(t, filepath.Join("testdata", fixture))
			res := Decode(strings.NewReader(string(data)))
			if res.Success() {
				t.Fatalf("expected reject for %s", fixture)
			}
			var found bool
			for _, d := range res.Diagnostics {
				if d.Code == CodeDuplicateKey {
					found = true
				}
			}
			if !found {
				t.Errorf("expected %s, got %+v", CodeDuplicateKey, res.Diagnostics)
			}
		})
	}
}

func TestCorpusLimitShape(t *testing.T) {
	cases := []string{
		"limits/v2-document-size-shape.json",
		"limits/v2-checks-boundary-shape.json",
		"limits/v2-checks-over-boundary-shape.json",
	}
	for _, fixture := range cases {
		t.Run(fixture, func(t *testing.T) {
			data := readFixture(t, filepath.Join("testdata", fixture))
			res := Decode(strings.NewReader(string(data)))
			if !res.Success() {
				t.Fatalf("limit-shape %s should accept, got diags=%v err=%v",
					fixture, res.Diagnostics, res.Err)
			}
		})
	}
}

func TestCorpusCounts(t *testing.T) {
	// The corpus must contain exactly 41 committed JSON fixtures.
	matches, err := filepath.Glob("testdata/valid/*.json")
	if err != nil {
		t.Fatal(err)
	}
	if got := len(matches); got != 7 {
		t.Errorf("valid fixtures: got %d, want 7", got)
	}
	matches, _ = filepath.Glob("testdata/invalid/*.json")
	if got := len(matches); got != 28 {
		t.Errorf("invalid fixtures: got %d, want 28", got)
	}
	matches, _ = filepath.Glob("testdata/duplicate-keys/*.json")
	if got := len(matches); got != 3 {
		t.Errorf("duplicate-key fixtures: got %d, want 3", got)
	}
	matches, _ = filepath.Glob("testdata/limits/*.json")
	if got := len(matches); got != 3 {
		t.Errorf("limit fixtures: got %d, want 3", got)
	}
}
