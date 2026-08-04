// SPDX-License-Identifier: Apache-2.0

package forbidden

import (
	"fmt"
	"strings"
)

// WorkflowViolation represents a contract violation.
type WorkflowViolation struct {
	Type    string // violation type identifier
	Message string // human-readable message
}

// WorkflowViolationType constants.
const (
	ViolationWorkflowAuthority   = "workflow_authority_forbidden"
	ViolationJobAuthority        = "job_authority_forbidden"
	ViolationWrongJobName        = "wrong_display_name"
	ViolationWrongTimeout        = "wrong_timeout"
	ViolationJobContinueOnError  = "job_continue_on_error"
	ViolationStepContinueOnError = "step_continue_on_error"
	ViolationWrongAuthorityStep  = "authority_step_mismatch"
	ViolationMultipleAuthority   = "multiple_authority_steps"
	ViolationAuthorityValue      = "authority_value_invalid"
	ViolationMissingGateDupcode  = "gate_dupcode_command_missing"
	ViolationMissingCheckout     = "checkout_step_missing"
	ViolationCheckoutAction      = "checkout_action_invalid"
	ViolationMissingPreflight    = "preflight_step_missing"
	ViolationMissingSHAAssert    = "exact_sha_assertion_missing"
	ViolationMissingCleanTree    = "clean_tree_assertion_missing"
	ViolationSiblingsLeaked      = "sibling_jobs_leaked"
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

	// Collect authority-bearing steps and check step-level continue-on-error
	var authoritySteps []*WorkflowStep
	var hasCheckout, hasPreflight bool
	var checkoutUses string
	for i := range job.Steps {
		step := &job.Steps[i]

		// Check step-level continue-on-error
		if step.ContinueOnError == "true" {
			violations = append(violations, WorkflowViolation{
				Type:    ViolationStepContinueOnError,
				Message: "step '" + step.Name + "' must not use continue-on-error: true",
			})
		}

		if step.Name == "Checkout" && step.Uses != "" {
			hasCheckout = true
			checkoutUses = step.Uses
		}
		if step.Name == "Dupcode CI preflight" {
			hasPreflight = true
		}
		if step.Env != nil {
			if val, ok := step.Env["LEAMAS_DUPCODE_AUTHORITY"]; ok {
				authoritySteps = append(authoritySteps, step)
				// Check authority value in canonical mode
				if requireCanonical && val != "github-actions" {
					violations = append(violations, WorkflowViolation{
						Type:    ViolationAuthorityValue,
						Message: "authority value must be 'github-actions', got " + val,
					})
				}
			}
		}
	}

	// Check exactly one authority step
	if len(authoritySteps) == 0 {
		violations = append(violations, WorkflowViolation{
			Type:    ViolationWrongAuthorityStep,
			Message: "no authority-bearing step found",
		})
	} else if len(authoritySteps) > 1 {
		violations = append(violations, WorkflowViolation{
			Type:    ViolationMultipleAuthority,
			Message: "exactly one authority step required, found " + fmt.Sprintf("%d", len(authoritySteps)),
		})
	} else {
		// Exactly one authority step - check placement
		authorityStep := authoritySteps[0]
		if requireCanonical && authorityStep.Name != "Run gate-dupcode" {
			violations = append(violations, WorkflowViolation{
				Type:    ViolationWrongAuthorityStep,
				Message: "authority should be on 'Run gate-dupcode', got " + authorityStep.Name,
			})
		}

		// Check authority step command using exact line matching
		authLines := strings.Split(authorityStep.Run, "\n")
		var authExecutableLines []string
		for _, l := range authLines {
			l = strings.TrimSpace(l)
			if l == "" || strings.HasPrefix(l, "#") {
				continue
			}
			authExecutableLines = append(authExecutableLines, l)
		}
		authLineSet := make(map[string]bool)
		for _, l := range authExecutableLines {
			authLineSet[l] = true
		}
		if !authLineSet["make gate-dupcode"] {
			violations = append(violations, WorkflowViolation{
				Type:    ViolationMissingGateDupcode,
				Message: "authority step must execute exact line: make gate-dupcode",
			})
		}
	}

	// Check checkout step
	if requireCanonical {
		if !hasCheckout {
			violations = append(violations, WorkflowViolation{
				Type:    ViolationMissingCheckout,
				Message: "workflow must have checkout step",
			})
		} else if !strings.HasPrefix(checkoutUses, "actions/checkout@") {
			violations = append(violations, WorkflowViolation{
				Type:    ViolationCheckoutAction,
				Message: "checkout must use actions/checkout@..., got " + checkoutUses,
			})
		}
	}

	// Check preflight step
	if requireCanonical && !hasPreflight {
		violations = append(violations, WorkflowViolation{
			Type:    ViolationMissingPreflight,
			Message: "workflow must have 'Dupcode CI preflight' step",
		})
	}

	// Check preflight assertions with exact line matching
	if hasPreflight && requireCanonical {
		for _, step := range job.Steps {
			if step.Name == "Dupcode CI preflight" {
				// Normalize run block to executable lines (no comments, no blanks)
				lines := strings.Split(step.Run, "\n")
				var executableLines []string
				for _, l := range lines {
					l = strings.TrimSpace(l)
					if l == "" || strings.HasPrefix(l, "#") {
						continue
					}
					// Strip error handling suffix (|| { ... })
					if idx := strings.Index(l, "||"); idx > 0 {
						l = strings.TrimSpace(l[:idx])
					}
					executableLines = append(executableLines, l)
				}

				// Build a map for exact line matching
				lineSet := make(map[string]bool)
				for _, l := range executableLines {
					lineSet[l] = true
				}

				// Required executable command lines (exact match)
				requiredLines := []string{
					`test "$GITHUB_ACTIONS" = "true"`,
					`test "$CI" = "true"`,
					`test -n "$GITHUB_SHA"`,
					`test "$(git rev-parse HEAD^{commit})" = "$GITHUB_SHA"`,
					`test -z "$(git status --porcelain=v1)"`,
				}

				for _, req := range requiredLines {
					if !lineSet[req] {
						violations = append(violations, WorkflowViolation{
							Type:    ViolationMissingSHAAssert,
							Message: "preflight must contain executable line: " + req,
						})
					}
				}

				break
			}
		}
	}

	return violations
}
