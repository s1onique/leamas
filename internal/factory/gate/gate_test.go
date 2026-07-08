package gate

import (
	"testing"
)

func TestAllVerifiers(t *testing.T) {
	verifiers := AllVerifiers()
	if len(verifiers) == 0 {
		t.Error("AllVerifiers should return verifiers")
	}

	// Check all have names
	for _, v := range verifiers {
		if v.Name == "" {
			t.Error("verifier should have a name")
		}
		if v.Run == nil {
			t.Error("verifier should have a Run function")
		}
	}
}

func TestRunFactorize(t *testing.T) {
	// Test that it runs without panicking
	result := RunFactorize(".")
	// We don't know the state of the repo, so just check it runs
	if result < 0 {
		t.Error("RunFactorize should return 0 or 1")
	}
}
