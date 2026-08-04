// SPDX-License-Identifier: Apache-2.0

package forbidden

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func repoRoot(t *testing.T) string {
	t.Helper()
	_, current, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve test source path")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(current), "..", "..", ".."))
}

func workflowPath(t *testing.T) string {
	return filepath.Join(repoRoot(t), ".github", "workflows", "factory.yml")
}

// TestFactoryDupcodeWorkflowContract validates the canonical workflow.
func TestFactoryDupcodeWorkflowContract(t *testing.T) {
	path := workflowPath(t)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Fatalf("factory.yml not found - failing closed: %v", err)
	}
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read workflow: %v", err)
	}

	violations := validateFactoryDupcodeWorkflow(content, true)
	if len(violations) > 0 {
		for _, v := range violations {
			t.Errorf("violation: %s - %s", v.Type, v.Message)
		}
	}
}
