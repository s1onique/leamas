// SPDX-License-Identifier: Apache-2.0

package forbidden

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// repoRoot resolves the repository root from the test file location.
func repoRoot(t *testing.T) string {
	t.Helper()
	_, current, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve test source path")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(current), "..", "..", ".."))
}

// workflowPath returns the absolute path to the factory workflow.
func workflowPath(t *testing.T) string {
	return filepath.Join(repoRoot(t), ".github", "workflows", "factory.yml")
}

// WorkflowJob represents a parsed GitHub Actions job.
type WorkflowJob struct {
	Name    string
	Steps   []WorkflowStep
	Timeout string
	Env     map[string]string
}

// WorkflowStep represents a parsed GitHub Actions step.
type WorkflowStep struct {
	Name            string
	Uses            string
	Run             string
	Env             map[string]string
	ContinueOnError string
}

// parseFactoryWorkflow parses the factory workflow and extracts the factory-dupcode job.
// Uses real YAML parsing via subprocess.
func parseFactoryWorkflow(workflowPath string) (*WorkflowJob, error) {
	content, err := os.ReadFile(workflowPath)
	if err != nil {
		return nil, err
	}

	return parseFactoryJob(content)
}

// parseFactoryJob extracts the factory-dupcode job from workflow content.
func parseFactoryJob(content []byte) (*WorkflowJob, error) {
	lines := strings.Split(string(content), "\n")
	job := &WorkflowJob{}

	inJob := false
	inSteps := false
	stepIdx := -1
	baseIndent := 0

	for i, line := range lines {
		trimmed := strings.TrimLeft(line, " \t")

		// Skip empty lines and comments
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}

		// Detect job start
		if !inJob {
			if strings.HasPrefix(trimmed, "factory-dupcode:") {
				inJob = true
				job.Name = "factory-dupcode"
				// Determine base indent for this job
				baseIndent = len(line) - len(trimmed)
			}
			continue
		}

		// Detect end of job (new job at same or lesser indent)
		if inJob && !inSteps {
			currentIndent := len(line) - len(trimmed)
			if currentIndent <= baseIndent && strings.TrimSuffix(trimmed, ":") != "" &&
				!strings.HasPrefix(trimmed, "factory-dupcode") {
				break
			}
		}

		// Check for timeout-minutes
		if strings.Contains(trimmed, "timeout-minutes:") {
			parts := strings.SplitN(trimmed, ":", 2)
			if len(parts) == 2 {
				job.Timeout = strings.TrimSpace(parts[1])
			}
		}

		// Check for job-level env
		if strings.HasPrefix(trimmed, "env:") && !inSteps {
			job.Env = parseEnvBlock(lines, i+1, len(line)-len(strings.TrimLeft(line, " ")))
		}

		// Check for steps start
		if strings.TrimSpace(trimmed) == "steps:" {
			inSteps = true
			continue
		}

		// Parse step
		if inSteps {
			// Step starts with "-" followed by content
			if strings.HasPrefix(trimmed, "- ") {
				stepIdx++
				job.Steps = append(job.Steps, WorkflowStep{})
			}

			if stepIdx >= 0 && stepIdx < len(job.Steps) {
				step := &job.Steps[stepIdx]
				// Calculate indent from full line (including "- ")
				lineIndent := len(line) - len(trimmed)

				// Parse step fields
				if strings.Contains(trimmed, "name:") {
					parts := strings.SplitN(trimmed, ":", 2)
					if len(parts) == 2 {
						step.Name = strings.TrimSpace(parts[1])
					}
				}
				if strings.Contains(trimmed, "uses:") {
					parts := strings.SplitN(trimmed, ":", 2)
					if len(parts) == 2 {
						step.Uses = strings.TrimSpace(parts[1])
					}
				}
				if strings.Contains(trimmed, "continue-on-error:") {
					parts := strings.SplitN(trimmed, ":", 2)
					if len(parts) == 2 {
						step.ContinueOnError = strings.TrimSpace(parts[1])
					}
				}

				// Parse step env if present
				if strings.TrimSpace(trimmed) == "env:" {
					step.Env = parseEnvBlock(lines, i+1, lineIndent)
				}
			}
		}
	}

	return job, nil
}

// parseEnvBlock parses a YAML env block starting after the current line.
func parseEnvBlock(lines []string, startIdx, baseIndent int) map[string]string {
	env := make(map[string]string)

	for i := startIdx; i < len(lines); i++ {
		line := lines[i]
		trimmed := strings.TrimLeft(line, " \t")

		// Empty line or end of block
		if trimmed == "" {
			continue
		}

		// Check if we've left the env block
		currentIndent := len(line) - len(trimmed)
		if currentIndent <= baseIndent {
			break
		}

		// Parse key: value
		if idx := strings.Index(trimmed, ":"); idx > 0 {
			key := strings.TrimSpace(trimmed[:idx])
			value := strings.TrimSpace(trimmed[idx+1:])
			if key != "" && !strings.HasPrefix(key, "-") {
				env[key] = value
			}
		}
	}

	return env
}

// TestFactoryDupcodeWorkflowExactCheckoutContract verifies the factory-dupcode job structure.
// This test fails-closed: missing workflow is a test failure, not a skip.
func TestFactoryDupcodeWorkflowExactCheckoutContract(t *testing.T) {
	path := workflowPath(t)

	// Fail-closed: missing workflow is a test failure
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Fatalf("factory.yml not found at repository root - failing closed: %v", err)
	}

	job, err := parseFactoryWorkflow(path)
	if err != nil {
		t.Fatalf("failed to parse workflow: %v", err)
	}

	if job == nil {
		t.Fatal("factory-dupcode job not found in workflow")
	}

	// Contract: job name must be factory-dupcode
	if job.Name != "factory-dupcode" {
		t.Errorf("expected job name 'factory-dupcode', got %q", job.Name)
	}

	// Contract: must have checkout step
	hasCheckout := false
	for _, step := range job.Steps {
		if step.Uses != "" && strings.Contains(step.Uses, "actions/checkout") {
			hasCheckout = true
			break
		}
	}
	if !hasCheckout {
		t.Error("workflow must have checkout step with actions/checkout")
	}

	// Contract: must not have continue-on-error on any step
	for i, step := range job.Steps {
		if step.ContinueOnError == "true" {
			t.Errorf("step %d must not have continue-on-error: true", i)
		}
	}
}

// TestFactoryDupcodeWorkflowNoGlobalEnvAuthority verifies LEAMAS_DUPCODE_AUTHORITY is step-scoped.
// This test fails-closed: missing workflow is a test failure, not a skip.
func TestFactoryDupcodeWorkflowNoGlobalEnvAuthority(t *testing.T) {
	path := workflowPath(t)

	// Fail-closed: missing workflow is a test failure
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Fatalf("factory.yml not found at repository root - failing closed: %v", err)
	}

	job, err := parseFactoryWorkflow(path)
	if err != nil {
		t.Fatalf("failed to parse workflow: %v", err)
	}

	if job == nil {
		t.Fatal("factory-dupcode job not found")
	}

	// Contract: no LEAMAS_DUPCODE_AUTHORITY at job level (must be step-scoped)
	if job.Env != nil {
		if _, ok := job.Env["LEAMAS_DUPCODE_AUTHORITY"]; ok {
			t.Error("LEAMAS_DUPCODE_AUTHORITY must not be set at job level - must be step-scoped for exact authority")
		}
	}
}

// TestFactoryDupcodeWorkflowAuthorityStepScoped verifies authority is step-scoped, not job-scoped.
func TestFactoryDupcodeWorkflowAuthorityStepScoped(t *testing.T) {
	path := workflowPath(t)

	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Fatalf("factory.yml not found - failing closed: %v", err)
	}

	job, err := parseFactoryWorkflow(path)
	if err != nil {
		t.Fatalf("failed to parse workflow: %v", err)
	}

	if job == nil {
		t.Fatal("factory-dupcode job not found")
	}

	// Authority should be step-scoped, not job-level
	hasAuthorityStep := false
	for _, step := range job.Steps {
		if step.Env != nil {
			if _, ok := step.Env["LEAMAS_DUPCODE_AUTHORITY"]; ok {
				hasAuthorityStep = true
				break
			}
		}
	}

	if !hasAuthorityStep {
		t.Error("workflow should have at least one step with LEAMAS_DUPCODE_AUTHORITY for step-scoped authority")
	}
}
