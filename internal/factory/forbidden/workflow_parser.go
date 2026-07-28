// SPDX-License-Identifier: Apache-2.0

package forbidden

import (
	"strings"
)

// WorkflowJob represents a parsed GitHub Actions job.
type WorkflowJob struct {
	ID              string
	DisplayName     string
	Steps           []WorkflowStep
	Timeout         string
	Env             map[string]string
	ContinueOnError string
}

// WorkflowStep represents a parsed GitHub Actions step.
type WorkflowStep struct {
	Name            string
	Uses            string
	Run             string
	Env             map[string]string
	ContinueOnError string
}

// WorkflowEnv represents parsed workflow-level environment.
type WorkflowEnv struct {
	Env map[string]string
}

// parseWorkflowEnv extracts top-level env block from workflow content.
// Only considers env: at indent 0 (no leading spaces).
func parseWorkflowEnv(content []byte) *WorkflowEnv {
	lines := strings.Split(string(content), "\n")
	we := &WorkflowEnv{Env: make(map[string]string)}

	for i := 0; i < len(lines); i++ {
		line, trimmed := lines[i], strings.TrimLeft(lines[i], " \t")
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		// Track indentation: env: must be at indent 0
		indent := len(line) - len(trimmed)
		if strings.TrimSpace(trimmed) == "env:" && indent == 0 {
			we.Env = parseEnvBlock(lines, i+1, 0)
			break
		}
		// Stop at jobs: key (start of job definitions)
		if strings.TrimSpace(trimmed) == "jobs:" {
			break
		}
	}
	return we
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

		if strings.Contains(trimmed, "continue-on-error:") && !inSteps {
			if p := strings.SplitN(trimmed, ":", 2); len(p) == 2 {
				job.ContinueOnError = strings.TrimSpace(p[1])
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
			if strings.HasPrefix(trimmed, "- ") {
				stepContent := strings.TrimPrefix(trimmed, "- ")
				if stepContent != trimmed {
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
			}

			// Parse step fields
			if stepIdx >= 0 && stepIdx < len(job.Steps) {
				step := &job.Steps[stepIdx]
				stepContent := strings.TrimPrefix(trimmed, "- ")

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
