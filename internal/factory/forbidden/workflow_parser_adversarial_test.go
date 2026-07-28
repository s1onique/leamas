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
	var found bool
	for _, v := range violations {
		if v.Type == ViolationWorkflowAuthority {
			found = true
			break
		}
	}
	if !found {
		t.Error("contract should reject workflow-level LEAMAS_DUPCODE_AUTHORITY")
	}
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
	var found bool
	for _, v := range violations {
		if v.Type == ViolationJobAuthority {
			found = true
			break
		}
	}
	if !found {
		t.Error("contract should reject job-level LEAMAS_DUPCODE_AUTHORITY")
	}
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
	var found bool
	for _, v := range violations {
		if v.Type == ViolationJobContinueOnError {
			found = true
			break
		}
	}
	if !found {
		t.Error("contract should reject job-level continue-on-error: true")
	}
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
	var found bool
	for _, v := range violations {
		if v.Type == ViolationWrongAuthorityStep {
			found = true
			break
		}
	}
	if !found {
		t.Error("contract should reject authority on wrong step in canonical mode")
	}
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
	var found bool
	for _, v := range violations {
		if v.Type == ViolationMissingGateDupcode {
			found = true
			break
		}
	}
	if !found {
		t.Error("contract should reject missing make gate-dupcode command")
	}
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
        run: |
          test "$GITHUB_ACTIONS" = "true"
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

// TestContractRejectsWrongDisplayName verifies wrong display name is rejected in canonical mode.
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
	var found bool
	for _, v := range violations {
		if v.Type == ViolationWrongJobName {
			found = true
			break
		}
	}
	if !found {
		t.Error("contract should reject wrong display name in canonical mode")
	}
}

// TestContractRejectsWrongTimeout verifies wrong timeout is rejected in canonical mode.
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
	var found bool
	for _, v := range violations {
		if v.Type == ViolationWrongTimeout {
			found = true
			break
		}
	}
	if !found {
		t.Error("contract should reject wrong timeout in canonical mode")
	}
}

// TestContractRejectsMissingCheckout verifies missing checkout is rejected in canonical mode.
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
	var found bool
	for _, v := range violations {
		if v.Type == ViolationMissingCheckout {
			found = true
			break
		}
	}
	if !found {
		t.Error("contract should reject missing checkout in canonical mode")
	}
}

// TestContractRejectsMissingPreflight verifies missing preflight is rejected in canonical mode.
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
	var found bool
	for _, v := range violations {
		if v.Type == ViolationMissingPreflight {
			found = true
			break
		}
	}
	if !found {
		t.Error("contract should reject missing preflight in canonical mode")
	}
}

// TestContractRejectsMissingSHAAssert verifies commented SHA assertion is rejected.
func TestContractRejectsMissingSHAAssert(t *testing.T) {
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
	var found bool
	for _, v := range violations {
		if v.Type == ViolationMissingSHAAssert {
			found = true
			break
		}
	}
	if !found {
		t.Error("contract should reject commented SHA assertion in canonical mode")
	}
}

// TestContractRejectsMissingCleanTree verifies commented clean tree assertion is rejected.
func TestContractRejectsMissingCleanTree(t *testing.T) {
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
          # test -z "$(git status --porcelain=v1)"
`
	violations := validateFactoryDupcodeWorkflow([]byte(yaml), true)
	var found bool
	for _, v := range violations {
		if v.Type == ViolationMissingCleanTree {
			found = true
			break
		}
	}
	if !found {
		t.Error("contract should reject commented clean tree assertion in canonical mode")
	}
}
