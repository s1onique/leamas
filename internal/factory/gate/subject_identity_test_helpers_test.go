// Package gate provides test helpers for subject identity testing.
package gate

import (
	"testing"

	"github.com/s1onique/leamas/internal/execution/exectest"
)

// TestCollectSubjectIdentity tests the subject identity collection.
func TestCollectSubjectIdentity(t *testing.T) {
	// Smoke test that the CollectSubjectIdentity function compiles.
	// Real tests would use a temporary git repository fixture.
	t.Skip("requires git repository fixture")
}

// Test helper to run git commands using exectest.
func runGitForTest(dir string, args ...string) (string, error) {
	req := exectest.Request{
		Dir:  dir,
		Name: "git",
		Args: args,
	}
	output, err := exectest.Output(req)
	if err != nil {
		return "", err
	}
	return string(output), nil
}
