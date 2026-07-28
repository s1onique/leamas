// SPDX-License-Identifier: Apache-2.0

package forbidden

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func repoRootAuth(t *testing.T) string {
	t.Helper()
	_, current, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve test source path")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(current), "..", "..", ".."))
}

func workflowPathAuth(t *testing.T) string {
	return filepath.Join(repoRootAuth(t), ".github", "workflows", "factory.yml")
}

// TestFactoryDupcodeWorkflowAuthorityStepScoped verifies authority is step-scoped.
func TestFactoryDupcodeWorkflowAuthorityStepScoped(t *testing.T) {
	path := workflowPathAuth(t)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Fatalf("factory.yml not found - failing closed: %v", err)
	}
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read workflow: %v", err)
	}
	job, err := parseFactoryJob(content)
	if err != nil {
		t.Fatalf("failed to parse workflow: %v", err)
	}
	if job == nil {
		t.Fatal("factory-dupcode job not found")
	}

	var authoritySteps []int
	for i, step := range job.Steps {
		if step.Env != nil {
			if _, ok := step.Env["LEAMAS_DUPCODE_AUTHORITY"]; ok {
				authoritySteps = append(authoritySteps, i)
			}
		}
	}

	if len(authoritySteps) == 0 {
		t.Error("workflow must have at least one step with LEAMAS_DUPCODE_AUTHORITY")
	}
	if len(authoritySteps) > 1 {
		t.Errorf("workflow must have exactly one authority-bearing step, found %d", len(authoritySteps))
	}

	if len(authoritySteps) == 1 {
		step := &job.Steps[authoritySteps[0]]
		if step.Name != "Run gate-dupcode" {
			t.Errorf("authority step should be named 'Run gate-dupcode', got %q", step.Name)
		}
		if val, ok := step.Env["LEAMAS_DUPCODE_AUTHORITY"]; !ok || val != "github-actions" {
			t.Errorf("authority value should be 'github-actions', got %q", val)
		}
		if step.Run == "" || !strings.Contains(step.Run, "make gate-dupcode") {
			t.Error("authority step must run 'make gate-dupcode'")
		}
	}
}
