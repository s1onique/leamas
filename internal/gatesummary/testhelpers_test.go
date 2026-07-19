package gatesummary

import (
	"errors"
	"os"
	"testing"

	jsonschema "github.com/santhosh-tekuri/jsonschema/v6"
)

// mustValidationError asserts that err is a *jsonschema.ValidationError
// and returns the typed value.
func mustValidationError(t *testing.T, err error) *jsonschema.ValidationError {
	t.Helper()
	var ve *jsonschema.ValidationError
	if !errors.As(err, &ve) {
		t.Fatalf("expected *jsonschema.ValidationError, got %T: %v", err, err)
	}
	return ve
}

// readFixture loads a fixture path relative to the package directory.
func readFixture(t testing.TB, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fixture %s: %v", path, err)
	}
	return data
}
