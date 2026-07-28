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
	ID          string
	DisplayName string
	Steps       []WorkflowStep
	Timeout     string
	Env         map[string]string
}

// WorkflowStep represents a parsed GitHub Actions step.
type WorkflowStep struct {
	Name            string
	Uses            string
	Run             string
	Env             map[string]string
	ContinueOnError string
}

// parseFactoryJob extracts the factory-dupcode job from workflow content.
func parseFactoryJob(content []byte) (*WorkflowJob, error) {
	lines := strings.Split(string(content), "\n")
	job := &WorkflowJob{}
	inJob, inSteps := false, false
	stepIdx, jobIndent, firstStepIndent := -1, 0, 0

	for i := 0; i < len(lines); i++ {
		line, trimmed := lines[i], strings.TrimLeft(lines[i], " \t")
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}

		if !inJob {
			if strings.HasPrefix(trimmed, "factory-dupcode:") {
				inJob, job.ID = true, "factory-dupcode"
				jobIndent = len(line) - len(trimmed)
			}
			continue
		}

		currentIndent := len(line) - len(trimmed)
		if currentIndent <= jobIndent && !strings.HasPrefix(trimmed, "- ") {
			if idx := strings.Index(trimmed, ":"); idx > 0 {
				if key := strings.TrimSpace(trimmed[:idx]); key != "" && !strings.HasPrefix(key, "-") {
					break
				}
			}
		}

		if strings.HasPrefix(trimmed, "name:") && !inSteps {
			if p := strings.SplitN(trimmed, ":", 2); len(p) == 2 {
				job.DisplayName = strings.TrimSpace(p[1])
			}
			continue
		}

		if strings.Contains(trimmed, "timeout-minutes:") && !inSteps {
			if p := strings.SplitN(trimmed, ":", 2); len(p) == 2 {
				job.Timeout = strings.TrimSpace(p[1])
			}
		}

		if strings.TrimSpace(trimmed) == "env:" && !inSteps {
			job.Env = parseEnvBlock(lines, i+1, currentIndent)
			continue
		}

		if strings.TrimSpace(trimmed) == "steps:" && !inSteps {
			inSteps = true
			continue
		}

		if inSteps {
			stepContent := strings.TrimPrefix(trimmed, "- ")
			if strings.HasPrefix(trimmed, "- ") && stepContent != trimmed {
				if stepIdx >= 0 && firstStepIndent > 0 {
					if len(line)-len(strings.TrimLeft(line, " \t")) <= jobIndent {
						break
					}
				}
				stepIdx++
				job.Steps = append(job.Steps, WorkflowStep{})
				if firstStepIndent == 0 {
					firstStepIndent = len(line) - len(strings.TrimLeft(line, " \t"))
				}
			}

			if stepIdx >= 0 && stepIdx < len(job.Steps) {
				step := &job.Steps[stepIdx]
				if strings.HasPrefix(stepContent, "name:") {
					if p := strings.SplitN(stepContent, ":", 2); len(p) == 2 {
						step.Name = strings.TrimSpace(p[1])
					}
				} else if strings.HasPrefix(stepContent, "uses:") {
					if p := strings.SplitN(stepContent, ":", 2); len(p) == 2 {
						step.Uses = strings.TrimSpace(p[1])
					}
				} else if strings.HasPrefix(stepContent, "continue-on-error:") {
					if p := strings.SplitN(stepContent, ":", 2); len(p) == 2 {
						step.ContinueOnError = strings.TrimSpace(p[1])
					}
				} else if stepContent == "env:" {
					step.Env = parseEnvBlock(lines, i+1, currentIndent)
				} else if strings.HasPrefix(stepContent, "run:") {
					if p := strings.SplitN(stepContent, ":", 2); len(p) == 2 {
						runBody := strings.TrimSpace(p[1])
						if strings.HasPrefix(runBody, "|") || strings.HasPrefix(runBody, ">") {
							runBody = strings.TrimSpace(strings.TrimPrefix(strings.TrimPrefix(runBody, "|"), ">"))
							for j := i + 1; j < len(lines); j++ {
								cLine, cTrim := lines[j], strings.TrimLeft(lines[j], " \t")
								if cTrim == "" {
									runBody += "\n"
									continue
								}
								if len(cLine)-len(cTrim) <= firstStepIndent {
									break
								}
								if runBody != "" && !strings.HasSuffix(runBody, "\n") {
									runBody += "\n"
								}
								runBody += cTrim
							}
						}
						step.Run = runBody
					}
				}
			}
		}
	}
	return job, nil
}

// parseEnvBlock parses a YAML env block.
func parseEnvBlock(lines []string, startIdx, baseIndent int) map[string]string {
	env := make(map[string]string)
	for i := startIdx; i < len(lines); i++ {
		line, trimmed := lines[i], strings.TrimLeft(lines[i], " \t")
		if trimmed == "" {
			continue
		}
		if len(line)-len(trimmed) <= baseIndent {
			break
		}
		if idx := strings.Index(trimmed, ":"); idx > 0 {
			if key := strings.TrimSpace(trimmed[:idx]); key != "" && !strings.HasPrefix(key, "-") {
				env[key] = strings.TrimSpace(trimmed[idx+1:])
			}
		}
	}
	return env
}

// TestFactoryDupcodeWorkflowExactCheckoutContract verifies the factory-dupcode job structure.
func TestFactoryDupcodeWorkflowExactCheckoutContract(t *testing.T) {
	path := workflowPath(t)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Fatalf("factory.yml not found at repository root - failing closed: %v", err)
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
		t.Fatal("factory-dupcode job not found in workflow")
	}
	if job.ID != "factory-dupcode" {
		t.Errorf("expected job ID 'factory-dupcode', got %q", job.ID)
	}
	if job.DisplayName == "" {
		t.Error("job display name is empty")
	}
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
	for i, step := range job.Steps {
		if step.ContinueOnError == "true" {
			t.Errorf("step %d must not have continue-on-error: true", i)
		}
	}
}

// TestFactoryDupcodeWorkflowNoGlobalEnvAuthority verifies LEAMAS_DUPCODE_AUTHORITY is step-scoped.
func TestFactoryDupcodeWorkflowNoGlobalEnvAuthority(t *testing.T) {
	path := workflowPath(t)
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
	if job.Env != nil {
		if _, ok := job.Env["LEAMAS_DUPCODE_AUTHORITY"]; ok {
			t.Error("LEAMAS_DUPCODE_AUTHORITY must not be at job level")
		}
	}
}
