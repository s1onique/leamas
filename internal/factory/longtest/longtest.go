// Package longtest provides long-test tiering infrastructure for the Factory gate.
//
// Long tests are registered in .factory/long-tests-baseline.json and are
// skipped when running in fast mode (testing.Short() == true). This enables
// the development loop to skip expensive repository-scanning tests while CI
// still runs all registered tests with the full timeout.
//
// Policy: Every long test must have a unique baseline ID. Unregistered
// direct skips fail policy validation.
package longtest

import (
	"encoding/json"
	"errors"
	"os"
	"regexp"
	"strings"
	"testing"
	"time"
)

// TestSpec defines a registered long test entry.
type TestSpec struct {
	ID         string `json:"id"`
	Package    string `json:"package"`
	Test       string `json:"test"`
	FastPolicy string `json:"fast_policy"` // "skip-under-short" or "run-always"
	CITimeout  string `json:"ci_timeout"`
	CIGroup    string `json:"ci_group"`
	Reason     string `json:"reason"`
	Owner      string `json:"owner"`
}

// Baseline represents the long-tests-baseline.json file.
type Baseline struct {
	SchemaVersion int        `json:"schema_version"`
	Tests         []TestSpec `json:"tests"`
}

// ErrBaselineMissing is returned when the long-tests baseline file is not found.
var ErrBaselineMissing = errors.New("long-test baseline file is missing but is required")

// LoadBaseline loads the long-tests-baseline.json from the given root.
// Returns an error if the file does not exist (fail-closed policy).
func LoadBaseline(root string) (*Baseline, error) {
	path := root + "/.factory/long-tests-baseline.json"
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrBaselineMissing
		}
		return nil, err
	}
	var b Baseline
	if err := json.Unmarshal(data, &b); err != nil {
		return nil, err
	}
	return &b, nil
}

// RequireLongTest skips the test if running in fast mode (-short flag).
// It requires a non-empty baselineID that must match a registered entry.
func RequireLongTest(t *testing.T, baselineID string) {
	t.Helper()

	if baselineID == "" {
		t.Fatal("long-test baseline ID is required")
	}

	if testing.Short() {
		t.Skipf(
			"long test %s skipped in fast mode; run make test-long",
			baselineID,
		)
	}
}

// ValidFastPolicies contains the allowed fast_policy values.
var ValidFastPolicies = map[string]bool{
	"skip-under-short": true,
	"run-always":       true,
}

// testNameRegex matches valid Go test function names: TestXxx where Xxx starts with uppercase.
var testNameRegex = regexp.MustCompile(`^Test[A-Z][a-zA-Z0-9_]*$`)

// packageSelectorRegex matches valid ./... package selectors.
var packageSelectorRegex = regexp.MustCompile(`^\./(\.\./)*\.\.\.$|^(\./)?[a-zA-Z0-9_/]+(\.\./)*\.\.\.$`)

// parseTimeout parses a Go duration string and returns its value.
func parseTimeout(s string) (time.Duration, error) {
	return time.ParseDuration(s)
}

// ValidateBaseline strictly validates the long-tests-baseline.json.
// It enforces:
//   - schema_version == 1
//   - tests is non-empty
//   - fast_policy == skip-under-short
//   - ci_timeout is positive and parseable
//   - ci_group, reason, and owner are non-empty
//   - package is a valid ./... selector
//   - test has valid TestXxx syntax
//   - IDs are unique
//   - package/test pairs are unique
func ValidateBaseline(baseline *Baseline) error {
	if baseline == nil {
		return &ValidationError{Field: "baseline", Message: "baseline is nil"}
	}

	if baseline.SchemaVersion != 1 {
		return &ValidationError{Field: "schema_version", Message: "must be 1"}
	}

	if len(baseline.Tests) == 0 {
		return &ValidationError{Field: "tests", Message: "tests is empty"}
	}

	seenIDs := make(map[string]bool)
	seenPkgTest := make(map[string]bool)

	for _, tt := range baseline.Tests {
		if tt.ID == "" {
			return &ValidationError{Field: "id", Message: "missing ID"}
		}
		if seenIDs[tt.ID] {
			return &ValidationError{ID: tt.ID, Field: "id", Message: "duplicate ID"}
		}
		seenIDs[tt.ID] = true

		if tt.CIGroup == "" {
			return &ValidationError{ID: tt.ID, Field: "ci_group", Message: "missing ci_group"}
		}
		if tt.Reason == "" {
			return &ValidationError{ID: tt.ID, Field: "reason", Message: "missing reason"}
		}
		if tt.Owner == "" {
			return &ValidationError{ID: tt.ID, Field: "owner", Message: "missing owner"}
		}

		if tt.Package == "" {
			return &ValidationError{ID: tt.ID, Field: "package", Message: "missing package path"}
		}
		if !isValidPackageSelector(tt.Package) {
			return &ValidationError{ID: tt.ID, Field: "package", Message: "must be a valid ./... selector"}
		}

		if tt.Test == "" {
			return &ValidationError{ID: tt.ID, Field: "test", Message: "missing test name"}
		}
		if !testNameRegex.MatchString(tt.Test) {
			return &ValidationError{ID: tt.ID, Field: "test", Message: "must match TestXxx pattern (uppercase after Test)"}
		}

		pkgTestKey := tt.Package + "#" + tt.Test
		if seenPkgTest[pkgTestKey] {
			return &ValidationError{ID: tt.ID, Field: "package/test", Message: "duplicate package/test pair"}
		}
		seenPkgTest[pkgTestKey] = true

		// fast_policy must be skip-under-short (not run-always)
		if tt.FastPolicy != "skip-under-short" {
			return &ValidationError{ID: tt.ID, Field: "fast_policy", Message: "must be 'skip-under-short'"}
		}

		// ci_timeout must be positive and parseable
		if tt.CITimeout == "" {
			return &ValidationError{ID: tt.ID, Field: "ci_timeout", Message: "missing ci_timeout"}
		}
		timeout, err := parseTimeout(tt.CITimeout)
		if err != nil {
			return &ValidationError{ID: tt.ID, Field: "ci_timeout", Message: "unparseable duration"}
		}
		if timeout <= 0 {
			return &ValidationError{ID: tt.ID, Field: "ci_timeout", Message: "ci_timeout must be positive"}
		}
	}
	return nil
}

// isValidPackageSelector checks if a package path is valid for go test.
// Valid forms: "./path/to/pkg", "./path/to/pkg/...", "./..."
func isValidPackageSelector(pkg string) bool {
	if pkg == "" {
		return false
	}
	// Must start with ./
	if !strings.HasPrefix(pkg, "./") {
		return false
	}
	// Remove trailing / if present
	pkg = strings.TrimSuffix(pkg, "/")
	// Path segments must not contain spaces
	if strings.Contains(pkg, " ") {
		return false
	}
	return true
}

// ValidationError represents a baseline validation failure.
type ValidationError struct {
	ID      string
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	if e.ID != "" {
		return "test " + e.ID + ": " + e.Message
	}
	return e.Field + ": " + e.Message
}

// PolicyBaselineIDs returns the set of registered baseline IDs.
func PolicyBaselineIDs(baseline *Baseline) map[string]bool {
	ids := make(map[string]bool)
	if baseline != nil {
		for _, tt := range baseline.Tests {
			ids[tt.ID] = true
		}
	}
	return ids
}

// BaselineJSON serializes a Baseline to JSON bytes.
func BaselineJSON(baseline *Baseline) ([]byte, error) {
	return json.Marshal(baseline)
}
