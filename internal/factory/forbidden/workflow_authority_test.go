// SPDX-License-Identifier: Apache-2.0

package forbidden

import (
	"os"
	"testing"
)

// TestFactoryDupcodeWorkflowAuthorityScoped verifies authority is step-scoped in canonical workflow.
func TestFactoryDupcodeWorkflowAuthorityScoped(t *testing.T) {
	path := workflowPath(t)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Fatalf("factory.yml not found - failing closed: %v", err)
	}
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read workflow: %v", err)
	}

	violations := validateFactoryDupcodeWorkflow(content, true)
	// Check for authority step violations
	for _, v := range violations {
		if v.Type == ViolationWrongAuthorityStep || v.Type == ViolationMissingGateDupcode {
			t.Errorf("authority violation: %s - %s", v.Type, v.Message)
		}
	}
}
