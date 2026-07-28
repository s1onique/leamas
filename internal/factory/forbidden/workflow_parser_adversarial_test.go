// SPDX-License-Identifier: Apache-2.0

package forbidden

import (
	"testing"
)

// TestContractRejectsWorkflowLevelAuthority verifies workflow-level authority is rejected.
func TestContractRejectsWorkflowLevelAuthority(t *testing.T) {
	yaml := `name: Factory CI
on: [push]
env:
  LEAMAS_DUPCODE_AUTHORITY: github-actions
jobs:
  factory-dupcode:
    name: Factory Dupcode
    timeout-minutes: 30
    steps:
      - name: Checkout
        uses: actions/checkout@v4
`
	violations := validateFactoryDupcodeWorkflow([]byte(yaml), false)
	findViolation(t, violations, ViolationWorkflowAuthority, "workflow-level LEAMAS_DUPCODE_AUTHORITY")
}

// TestContractRejectsJobLevelAuthority verifies job-level authority is rejected.
func TestContractRejectsJobLevelAuthority(t *testing.T) {
	yaml := `name: Factory CI
on: [push]
jobs:
  factory-dupcode:
    name: Factory Dupcode
    timeout-minutes: 30
    env:
      LEAMAS_DUPCODE_AUTHORITY: github-actions
    steps:
      - name: Checkout
        uses: actions/checkout@v4
`
	violations := validateFactoryDupcodeWorkflow([]byte(yaml), false)
	findViolation(t, violations, ViolationJobAuthority, "job-level LEAMAS_DUPCODE_AUTHORITY")
}

// TestContractRejectsJobContinueOnError verifies job-level continue-on-error is rejected.
func TestContractRejectsJobContinueOnError(t *testing.T) {
	yaml := `name: Factory CI
on: [push]
jobs:
  factory-dupcode:
    name: Factory Dupcode
    timeout-minutes: 30
    continue-on-error: true
    steps:
      - name: Checkout
        uses: actions/checkout@v4
`
	violations := validateFactoryDupcodeWorkflow([]byte(yaml), false)
	findViolation(t, violations, ViolationJobContinueOnError, "job-level continue-on-error: true")
}

// TestContractRejectsWrongAuthorityStep verifies authority on wrong step is rejected.
func TestContractRejectsWrongAuthorityStep(t *testing.T) {
	yaml := `name: Factory CI
on: [push]
jobs:
  factory-dupcode:
    name: Factory Dupcode
    timeout-minutes: 30
    steps:
      - name: Checkout
        uses: actions/checkout@v4
        env:
          LEAMAS_DUPCODE_AUTHORITY: github-actions
`
	violations := validateFactoryDupcodeWorkflow([]byte(yaml), true)
	findViolation(t, violations, ViolationWrongAuthorityStep, "authority on wrong step")
}

// TestContractRejectsMissingGateDupcode verifies missing make gate-dupcode is rejected.
func TestContractRejectsMissingGateDupcode(t *testing.T) {
	yaml := `name: Factory CI
on: [push]
jobs:
  factory-dupcode:
    name: Factory Dupcode
    timeout-minutes: 30
    steps:
      - name: Checkout
        uses: actions/checkout@v4
      - name: Run gate-dupcode
        env:
          LEAMAS_DUPCODE_AUTHORITY: github-actions
        run: make build
`
	violations := validateFactoryDupcodeWorkflow([]byte(yaml), false)
	findViolation(t, violations, ViolationMissingGateDupcode, "missing make gate-dupcode")
}

// TestContractRejectsSiblingLeakage verifies sibling job steps don't leak into factory-dupcode.
func TestContractRejectsSiblingLeakage(t *testing.T) {
	yaml := `name: Factory CI
on: [push]
jobs:
  factory-dupcode:
    name: Factory Dupcode
    timeout-minutes: 30
    steps:
      - name: Checkout
        uses: actions/checkout@v4
      - name: Dupcode CI preflight
        run: test "$GITHUB_ACTIONS" = "true"
  factory-long:
    name: Factory Long
    timeout-minutes: 120
    steps:
      - name: Long test step
        run: echo "this must not be in factory-dupcode"
`
	job, _ := parseFactoryJob([]byte(yaml))
	if len(job.Steps) != 2 {
		t.Errorf("expected 2 steps, got %d", len(job.Steps))
	}
	for _, step := range job.Steps {
		if step.Name == "Long test step" {
			t.Error("parser must not absorb factory-long steps")
		}
	}
}

// TestContractRejectsWrongDisplayName verifies wrong display name is rejected.
func TestContractRejectsWrongDisplayName(t *testing.T) {
	yaml := `name: Factory CI
on: [push]
jobs:
  factory-dupcode:
    name: Wrong Name
    timeout-minutes: 30
    steps:
      - name: Checkout
        uses: actions/checkout@v4
`
	violations := validateFactoryDupcodeWorkflow([]byte(yaml), true)
	findViolation(t, violations, ViolationWrongJobName, "wrong display name")
}

// TestContractRejectsWrongTimeout verifies wrong timeout is rejected.
func TestContractRejectsWrongTimeout(t *testing.T) {
	yaml := `name: Factory CI
on: [push]
jobs:
  factory-dupcode:
    name: Factory Dupcode
    timeout-minutes: 60
    steps:
      - name: Checkout
        uses: actions/checkout@v4
`
	violations := validateFactoryDupcodeWorkflow([]byte(yaml), true)
	findViolation(t, violations, ViolationWrongTimeout, "wrong timeout")
}

// TestContractRejectsMissingCheckout verifies missing checkout is rejected.
func TestContractRejectsMissingCheckout(t *testing.T) {
	yaml := `name: Factory CI
on: [push]
jobs:
  factory-dupcode:
    name: Factory Dupcode
    timeout-minutes: 30
    steps:
      - name: Run gate-dupcode
        env:
          LEAMAS_DUPCODE_AUTHORITY: github-actions
        run: make gate-dupcode
`
	violations := validateFactoryDupcodeWorkflow([]byte(yaml), true)
	findViolation(t, violations, ViolationMissingCheckout, "missing checkout")
}

// TestContractRejectsMissingPreflight verifies missing preflight is rejected.
func TestContractRejectsMissingPreflight(t *testing.T) {
	yaml := `name: Factory CI
on: [push]
jobs:
  factory-dupcode:
    name: Factory Dupcode
    timeout-minutes: 30
    steps:
      - name: Checkout
        uses: actions/checkout@v4
      - name: Run gate-dupcode
        env:
          LEAMAS_DUPCODE_AUTHORITY: github-actions
        run: make gate-dupcode
`
	violations := validateFactoryDupcodeWorkflow([]byte(yaml), true)
	findViolation(t, violations, ViolationMissingPreflight, "missing preflight")
}

// TestContractRejectsPreflightViolation verifies commented assertions are rejected.
func TestContractRejectsPreflightViolation(t *testing.T) {
	yaml := `name: Factory CI
on: [push]
jobs:
  factory-dupcode:
    name: Factory Dupcode
    timeout-minutes: 30
    steps:
      - name: Checkout
        uses: actions/checkout@v4
      - name: Dupcode CI preflight
        run: |
          test "$GITHUB_ACTIONS" = "true"
          # test "$(git rev-parse HEAD^{commit})" = "$GITHUB_SHA"
`
	violations := validateFactoryDupcodeWorkflow([]byte(yaml), true)
	findViolation(t, violations, ViolationMissingSHAAssert, "commented SHA assertion")
}

// TestContractRejectsMultipleAuthoritySteps verifies exactly one authority step is required.
func TestContractRejectsMultipleAuthoritySteps(t *testing.T) {
	yaml := `name: Factory CI
on: [push]
jobs:
  factory-dupcode:
    name: Factory Dupcode
    timeout-minutes: 30
    steps:
      - name: Checkout
        uses: actions/checkout@v4
      - name: Dupcode CI preflight
        env:
          LEAMAS_DUPCODE_AUTHORITY: github-actions
        run: test "$GITHUB_ACTIONS" = "true"
      - name: Run gate-dupcode
        env:
          LEAMAS_DUPCODE_AUTHORITY: github-actions
        run: make gate-dupcode
`
	violations := validateFactoryDupcodeWorkflow([]byte(yaml), true)
	findViolation(t, violations, ViolationMultipleAuthority, "multiple authority steps")
}

// TestContractRejectsInvalidAuthorityValue verifies authority value must be 'github-actions'.
func TestContractRejectsInvalidAuthorityValue(t *testing.T) {
	yaml := `name: Factory CI
on: [push]
jobs:
  factory-dupcode:
    name: Factory Dupcode
    timeout-minutes: 30
    steps:
      - name: Checkout
        uses: actions/checkout@v4
      - name: Run gate-dupcode
        env:
          LEAMAS_DUPCODE_AUTHORITY: local
        run: make gate-dupcode
`
	violations := validateFactoryDupcodeWorkflow([]byte(yaml), true)
	findViolation(t, violations, ViolationAuthorityValue, "invalid authority value 'local'")
}

// TestContractRejectsStepContinueOnError verifies step-level continue-on-error is rejected.
func TestContractRejectsStepContinueOnError(t *testing.T) {
	yaml := `name: Factory CI
on: [push]
jobs:
  factory-dupcode:
    name: Factory Dupcode
    timeout-minutes: 30
    steps:
      - name: Checkout
        uses: actions/checkout@v4
      - name: Run gate-dupcode
        continue-on-error: true
        env:
          LEAMAS_DUPCODE_AUTHORITY: github-actions
        run: make gate-dupcode
`
	violations := validateFactoryDupcodeWorkflow([]byte(yaml), false)
	findViolation(t, violations, ViolationStepContinueOnError, "step-level continue-on-error")
}

// TestContractRejectsInvalidCheckoutAction verifies checkout must use actions/checkout@.
func TestContractRejectsInvalidCheckoutAction(t *testing.T) {
	yaml := `name: Factory CI
on: [push]
jobs:
  factory-dupcode:
    name: Factory Dupcode
    timeout-minutes: 30
    steps:
      - name: Checkout
        uses: malicious/action@v1
      - name: Run gate-dupcode
        env:
          LEAMAS_DUPCODE_AUTHORITY: github-actions
        run: make gate-dupcode
`
	violations := validateFactoryDupcodeWorkflow([]byte(yaml), true)
	findViolation(t, violations, ViolationCheckoutAction, "non-canonical checkout action")
}

// findViolation checks that violations contain a specific type.
func findViolation(t *testing.T, violations []WorkflowViolation, typ, desc string) {
	t.Helper()
	for _, v := range violations {
		if v.Type == typ {
			return
		}
	}
	t.Errorf("contract should reject %s", desc)
}
