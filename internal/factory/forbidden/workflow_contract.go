// SPDX-License-Identifier: Apache-2.0

package forbidden

import (
	"strings"
)

// WorkflowViolation represents a contract violation.
type WorkflowViolation struct {
	Type    string // violation type identifier
	Message string // human-readable message
}

// WorkflowViolationType constants.
const (
	ViolationWorkflowAuthority  = "workflow_authority_forbidden"
	ViolationJobAuthority       = "job_authority_forbidden"
	ViolationWrongJobName       = "wrong_display_name"
	ViolationWrongTimeout       = "wrong_timeout"
	ViolationJobContinueOnError = "job_continue_on_error"
	ViolationWrongAuthorityStep = "authority_step_mismatch"
	ViolationMissingGateDupcode = "gate_dupcode_command_missing"
	ViolationMissingCheckout    = "checkout_step_missing"
	ViolationMissingPreflight   = "preflight_step_missing"
	ViolationMissingSHAAssert   = "exact_sha_assertion_missing"
	ViolationMissingCleanTree   = "clean_tree_assertion_missing"
	ViolationSiblingsLeaked     = "sibling_jobs_leaked"
)

// validateFactoryDupcodeWorkflow validates the factory-dupcode job contract.
// Returns zero violations for a valid workflow, or a list of violations.
func validateFactoryDupcodeWorkflow(content []byte, requireCanonical bool) []WorkflowViolation {
	var violations []WorkflowViolation

	// Check workflow-level env
	we := parseWorkflowEnv(content)
	if we != nil && we.Env != nil {
		if _, ok := we.Env["LEAMAS_DUPCODE_AUTHORITY"]; ok {
			violations = append(violations, WorkflowViolation{
				Type:    ViolationWorkflowAuthority,
				Message: "LEAMAS_DUPCODE_AUTHORITY must not be at workflow level",
			})
		}
	}

	// Parse job
	job, err := parseFactoryJob(content)
	if err != nil {
		violations = append(violations, WorkflowViolation{
			Type:    "parse_error",
			Message: "failed to parse workflow: " + err.Error(),
		})
		return violations
	}

	// Check job ID
	if job.ID != "factory-dupcode" {
		violations = append(violations, WorkflowViolation{
			Type:    "wrong_job_id",
			Message: "expected job ID 'factory-dupcode', got " + job.ID,
		})
	}

	// Check display name
	if requireCanonical && job.DisplayName != "Factory Dupcode" {
		violations = append(violations, WorkflowViolation{
			Type:    ViolationWrongJobName,
			Message: "expected display name 'Factory Dupcode', got " + job.DisplayName,
		})
	}

	// Check timeout
	if requireCanonical && job.Timeout != "30" {
		violations = append(violations, WorkflowViolation{
			Type:    ViolationWrongTimeout,
			Message: "expected timeout '30', got " + job.Timeout,
		})
	}

	// Check job-level continue-on-error
	if job.ContinueOnError == "true" {
		violations = append(violations, WorkflowViolation{
			Type:    ViolationJobContinueOnError,
			Message: "factory-dupcode job must not use continue-on-error: true",
		})
	}

	// Check job-level authority
	if job.Env != nil {
		if _, ok := job.Env["LEAMAS_DUPCODE_AUTHORITY"]; ok {
			violations = append(violations, WorkflowViolation{
				Type:    ViolationJobAuthority,
				Message: "LEAMAS_DUPCODE_AUTHORITY must not be at job level",
			})
		}
	}

	// Check for authority-bearing step
	var authorityStep *WorkflowStep
	var hasCheckout, hasPreflight bool
	for _, step := range job.Steps {
		if step.Name == "Checkout" && step.Uses != "" {
			hasCheckout = true
		}
		if step.Name == "Dupcode CI preflight" {
			hasPreflight = true
		}
		if step.Env != nil {
			if _, ok := step.Env["LEAMAS_DUPCODE_AUTHORITY"]; ok {
				authorityStep = &step
			}
		}
	}

	// Check authority step placement (always required, not just in canonical mode)
	if authorityStep == nil {
		violations = append(violations, WorkflowViolation{
			Type:    ViolationWrongAuthorityStep,
			Message: "no authority-bearing step found",
		})
	} else if requireCanonical && authorityStep.Name != "Run gate-dupcode" {
		violations = append(violations, WorkflowViolation{
			Type:    ViolationWrongAuthorityStep,
			Message: "authority should be on 'Run gate-dupcode', got " + authorityStep.Name,
		})
	}

	// Check authority step command
	if authorityStep != nil && !strings.Contains(authorityStep.Run, "make gate-dupcode") {
		violations = append(violations, WorkflowViolation{
			Type:    ViolationMissingGateDupcode,
			Message: "authority step must run 'make gate-dupcode', got: " + authorityStep.Run,
		})
	}

	// Check checkout step
	if requireCanonical && !hasCheckout {
		violations = append(violations, WorkflowViolation{
			Type:    ViolationMissingCheckout,
			Message: "workflow must have checkout step",
		})
	}

	// Check preflight step
	if requireCanonical && !hasPreflight {
		violations = append(violations, WorkflowViolation{
			Type:    ViolationMissingPreflight,
			Message: "workflow must have 'Dupcode CI preflight' step",
		})
	}

	// Check preflight assertions
	if hasPreflight {
		for _, step := range job.Steps {
			if step.Name == "Dupcode CI preflight" {
				// Normalize run block to executable lines
				lines := strings.Split(step.Run, "\n")
				var executableLines []string
				for _, l := range lines {
					l = strings.TrimSpace(l)
					if l == "" || strings.HasPrefix(l, "#") {
						continue
					}
					executableLines = append(executableLines, l)
				}

				runBody := strings.Join(executableLines, "\n")

				if requireCanonical {
					// Check for exact SHA assertion
					if !strings.Contains(runBody, `test "$(git rev-parse HEAD^{commit})" = "$GITHUB_SHA"`) {
						violations = append(violations, WorkflowViolation{
							Type:    ViolationMissingSHAAssert,
							Message: "preflight must contain exact SHA assertion",
						})
					}

					// Check for clean tree assertion
					if !strings.Contains(runBody, `test -z "$(git status --porcelain=v1)"`) {
						violations = append(violations, WorkflowViolation{
							Type:    ViolationMissingCleanTree,
							Message: "preflight must contain clean tree assertion",
						})
					}
				}
				break
			}
		}
	}

	return violations
}
