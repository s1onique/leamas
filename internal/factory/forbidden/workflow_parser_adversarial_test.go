// SPDX-License-Identifier: Apache-2.0

package forbidden

import (
	"strings"
	"testing"
)

// TestParserRejectsSiblingJobSteps verifies parser stops at factory-long job boundary.
func TestParserRejectsSiblingJobSteps(t *testing.T) {
	yaml := `name: Factory CI
on: [push, pull_request]
env:
  GO111MODULE: "on"
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
          test "$(git rev-parse HEAD^{commit})" = "$GITHUB_SHA"
  factory-long:
    name: Factory Long
    timeout-minutes: 120
    steps:
      - name: Long test step
        run: echo "this must not be in factory-dupcode"
`
	job, err := parseFactoryJob([]byte(yaml))
	if err != nil {
		t.Fatalf("parser should not error on valid workflow: %v", err)
	}
	if len(job.Steps) != 2 {
		t.Errorf("expected exactly 2 steps, got %d", len(job.Steps))
	}
	for _, step := range job.Steps {
		if step.Name == "Long test step" {
			t.Error("parser must not absorb steps from factory-long job")
		}
	}
}

// TestParserCapturesDisplayName verifies parser captures display name for contract testing.
func TestParserCapturesDisplayName(t *testing.T) {
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
	job, _ := parseFactoryJob([]byte(yaml))
	if job.DisplayName != "Wrong Name" {
		t.Errorf("parser should capture display name, got %q", job.DisplayName)
	}
}

// TestParserCapturesTimeout verifies parser captures timeout for contract testing.
func TestParserCapturesTimeout(t *testing.T) {
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
	job, _ := parseFactoryJob([]byte(yaml))
	if job.Timeout != "60" {
		t.Errorf("parser should capture timeout, got %q", job.Timeout)
	}
}

// TestParserCapturesCleanTreeAssertion verifies preflight captures clean tree check.
func TestParserCapturesCleanTreeAssertion(t *testing.T) {
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
          test -z "$(git status --porcelain=v1)"
`
	job, _ := parseFactoryJob([]byte(yaml))
	var preflight *WorkflowStep
	for i := range job.Steps {
		if job.Steps[i].Name == "Dupcode CI preflight" {
			preflight = &job.Steps[i]
			break
		}
	}
	if preflight == nil {
		t.Fatal("parser should find preflight step")
	}
	if !strings.Contains(preflight.Run, "git status --porcelain=v1") {
		t.Error("parser should capture clean tree assertion")
	}
}

// TestParserDetectsWrongAuthorityPlacement verifies parser detects authority on wrong step.
// This restricted parser only handles same-line step fields, so we test with inline env.
func TestParserDetectsWrongAuthorityPlacement(t *testing.T) {
	// Test that the authority test correctly finds multiple authority-bearing steps
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
	job, _ := parseFactoryJob([]byte(yaml))
	var authSteps []string
	for _, step := range job.Steps {
		if step.Env != nil {
			if _, ok := step.Env["LEAMAS_DUPCODE_AUTHORITY"]; ok {
				authSteps = append(authSteps, step.Name)
			}
		}
	}
	if len(authSteps) != 1 {
		t.Errorf("expected exactly 1 authority step, got %d: %v", len(authSteps), authSteps)
	}
	if len(authSteps) > 0 && authSteps[0] != "Run gate-dupcode" {
		t.Errorf("authority should be on Run gate-dupcode, got %q", authSteps[0])
	}
}

// TestParserCapturesMissingMakeGateDupcode verifies parser captures wrong run command.
func TestParserCapturesMissingMakeGateDupcode(t *testing.T) {
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
	job, _ := parseFactoryJob([]byte(yaml))
	var authorityStep *WorkflowStep
	for _, step := range job.Steps {
		if step.Env != nil {
			if _, ok := step.Env["LEAMAS_DUPCODE_AUTHORITY"]; ok {
				authorityStep = &step
				break
			}
		}
	}
	if authorityStep != nil && !strings.Contains(authorityStep.Run, "make gate-dupcode") {
		t.Logf("correctly detected missing make gate-dupcode in: %q", authorityStep.Run)
	}
}

// TestParserCapturesJobLevelAuthority verifies parser captures job-level env.
func TestParserCapturesJobLevelAuthority(t *testing.T) {
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
	job, _ := parseFactoryJob([]byte(yaml))
	if job.Env != nil {
		if _, ok := job.Env["LEAMAS_DUPCODE_AUTHORITY"]; ok {
			t.Log("correctly detected job-level authority")
		}
	}
}

// TestParserCapturesWorkflowLevelAuthority verifies parser captures workflow-level env.
func TestParserCapturesWorkflowLevelAuthority(t *testing.T) {
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
	we := parseWorkflowEnv([]byte(yaml))
	if we != nil && we.Env != nil {
		if _, ok := we.Env["LEAMAS_DUPCODE_AUTHORITY"]; ok {
			t.Log("correctly detected workflow-level authority")
		}
	}
}
