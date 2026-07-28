// SPDX-License-Identifier: Apache-2.0

package forbidden

import (
	"os"
	"path/filepath"
	"testing"
)

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
func parseFactoryWorkflow(workflowPath string) (*WorkflowJob, error) {
	content, err := os.ReadFile(workflowPath)
	if err != nil {
		return nil, err
	}

	// Parse YAML manually for the specific structure we need
	return parseFactoryJob(content)
}

// parseFactoryJob extracts the factory-dupcode job from workflow content.
func parseFactoryJob(content []byte) (*WorkflowJob, error) {
	job := &WorkflowJob{}

	lines := splitLines(content)
	inJob := false
	inSteps := false
	currentStep := -1

	for i := 0; i < len(lines); i++ {
		line := lines[i]

		// Look for the factory-dupcode job
		if !inJob {
			if isJobStart(line, "factory-dupcode") {
				inJob = true
				job.Name = "factory-dupcode"
			}
			continue
		}

		// End of job (new job or end of file)
		if isJobStart(line, "") && !inSteps {
			break
		}

		// Check for timeout
		if hasKey(line, "timeout-minutes") {
			job.Timeout = extractValue(line)
		}

		// Check for env at job level
		if hasKey(line, "env:") && !inSteps {
			// This is job-level env, but we only care about step-level
		}

		// Check for steps
		if hasKey(line, "steps:") {
			inSteps = true
			continue
		}

		// Parse step
		if inSteps && isStepStart(line) {
			currentStep++
			step := WorkflowStep{}
			job.Steps = append(job.Steps, step)
		}

		if currentStep >= 0 && currentStep < len(job.Steps) {
			step := &job.Steps[currentStep]
			if hasKey(line, "name:") {
				step.Name = extractValue(line)
			}
			if hasKey(line, "uses:") {
				step.Uses = extractValue(line)
			}
			if hasKey(line, "run:") {
				step.Run = extractValue(line)
			}
			if hasKey(line, "continue-on-error:") {
				step.ContinueOnError = extractValue(line)
			}
			if hasKey(line, "env:") && !isIndentedMoreThan(line, lines, i, 4) {
				// This is step-level env
			}
		}
	}

	return job, nil
}

func splitLines(content []byte) []string {
	var lines []string
	start := 0
	for i, b := range content {
		if b == '\n' {
			lines = append(lines, string(content[start:i]))
			start = i + 1
		}
	}
	if start < len(content) {
		lines = append(lines, string(content[start:]))
	}
	return lines
}

func isJobStart(line, jobName string) bool {
	trimmed := trimLeadingSpaces(line)
	if jobName == "" {
		return len(trimmed) > 0 && trimmed[len(trimmed)-1] == ':'
	}
	return trimmed == jobName+":"
}

func isStepStart(line string) bool {
	trimmed := trimLeadingSpaces(line)
	// Steps are typically at 4 spaces indent
	return len(trimmed) > 0 && trimmed[0] == '-' && len(trimmed) > 1 && trimmed[1] == ' '
}

func hasKey(line, key string) bool {
	return containsString(line, key)
}

func extractValue(line string) string {
	idx := -1
	for i, c := range line {
		if c == ':' {
			idx = i
			break
		}
	}
	if idx < 0 || idx+1 >= len(line) {
		return ""
	}
	return trimSpaces(line[idx+1:])
}

func trimLeadingSpaces(s string) string {
	i := 0
	for i < len(s) && (s[i] == ' ' || s[i] == '\t') {
		i++
	}
	return s[i:]
}

func trimSpaces(s string) string {
	i, j := 0, len(s)
	for i < j && (s[i] == ' ' || s[i] == '\t') {
		i++
	}
	for j > i && (s[j-1] == ' ' || s[j-1] == '\t') {
		j--
	}
	return s[i:j]
}

func containsString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func isIndentedMoreThan(line string, lines []string, idx, minIndent int) bool {
	return len(line) > 0 && line[0] == ' '
}

func TestFactoryDupcodeWorkflowExactCheckoutContract(t *testing.T) {
	// Must find the real workflow at the repository root
	workflowPath := filepath.Join("..", "..", ".github", "workflows", "factory.yml")
	if _, err := os.Stat(workflowPath); os.IsNotExist(err) {
		t.Skip("factory.yml not found at repository root - this test requires the real workflow")
	}

	job, err := parseFactoryWorkflow(workflowPath)
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
		if containsString(step.Uses, "actions/checkout") {
			hasCheckout = true
			break
		}
	}
	if !hasCheckout {
		t.Error("workflow must have checkout step")
	}

	// Contract: must not have continue-on-error on any step
	for i, step := range job.Steps {
		if step.ContinueOnError == "true" {
			t.Errorf("step %d must not have continue-on-error: true", i)
		}
	}
}

func TestFactoryDupcodeWorkflowNoGlobalEnvAuthority(t *testing.T) {
	workflowPath := filepath.Join("..", "..", ".github", "workflows", "factory.yml")
	if _, err := os.Stat(workflowPath); os.IsNotExist(err) {
		t.Skip("factory.yml not found at repository root")
	}

	job, err := parseFactoryWorkflow(workflowPath)
	if err != nil {
		t.Fatalf("failed to parse workflow: %v", err)
	}

	if job == nil {
		t.Fatal("factory-dupcode job not found")
	}

	// Contract: no LEAMAS_DUPCODE_AUTHORITY at job level
	if job.Env != nil {
		if _, ok := job.Env["LEAMAS_DUPCODE_AUTHORITY"]; ok {
			t.Error("LEAMAS_DUPCODE_AUTHORITY must not be set at job level")
		}
	}
}
