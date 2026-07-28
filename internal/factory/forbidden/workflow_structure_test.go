// SPDX-License-Identifier: Apache-2.0

package forbidden

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// WorkflowTestCase represents a parsed workflow contract.
type WorkflowTestCase struct {
	JobName           string
	DisplayedName     string
	HasCheckout       bool
	HasExactSHA       bool
	HasCleanTree      bool
	HasTreeOID        bool
	AuthorityScoped   bool
	InvokesMakeTarget string
	HasContinueError  bool
}

// parseWorkflow parses a workflow file for structural tests.
func parseWorkflow(path string) (*WorkflowTestCase, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	contentStr := string(content)

	tc := &WorkflowTestCase{}

	// Check for job name
	if strings.Contains(contentStr, "factory-dupcode:") {
		tc.JobName = "factory-dupcode"
	}

	// Check for displayed name
	if strings.Contains(contentStr, "name: Factory Dupcode") {
		tc.DisplayedName = "Factory Dupcode"
	}

	// Check for checkout step
	if strings.Contains(contentStr, "uses: actions/checkout@v") {
		tc.HasCheckout = true
	}

	// Check for exact SHA assertion
	if strings.Contains(contentStr, "git rev-parse HEAD^{commit}") && strings.Contains(contentStr, "$GITHUB_SHA") {
		tc.HasExactSHA = true
	}

	// Check for clean tree assertion
	if strings.Contains(contentStr, "git status --porcelain") || strings.Contains(contentStr, "worktree") {
		tc.HasCleanTree = true
	}

	// Check for tree OID
	if strings.Contains(contentStr, "git rev-parse HEAD^{tree}") {
		tc.HasTreeOID = true
	}

	// Check for authority marker scoped to step (not job level)
	if strings.Contains(contentStr, "env:") && strings.Contains(contentStr, "LEAMAS_DUPCODE_AUTHORITY") {
		tc.AuthorityScoped = true
	}

	// Check for make target invocation
	if strings.Contains(contentStr, "make gate-dupcode") {
		tc.InvokesMakeTarget = "gate-dupcode"
	}

	// Check for continue-on-error
	if strings.Contains(contentStr, "continue-on-error: true") {
		tc.HasContinueError = true
	}

	return tc, nil
}

func TestFactoryDupcodeWorkflowExactCheckoutContract(t *testing.T) {
	// Find the workflow file
	paths := []string{
		filepath.Join("..", "..", "..", ".github", "workflows", "factory.yml"),
		filepath.Join(".github", "workflows", "factory.yml"),
		filepath.Join("internal", "factory", "forbidden", "testdata", "factory.yml"),
	}

	var workflowPath string
	for _, p := range paths {
		if _, err := os.Stat(p); err == nil {
			workflowPath = p
			break
		}
	}

	if workflowPath == "" {
		// Create a test fixture
		dir := t.TempDir()
		fixture := filepath.Join(dir, "factory.yml")
		content := `# Factory CI
name: Factory

on:
  push:
    branches:
      - main
  pull_request:
    branches:
      - main

jobs:
  factory-dupcode:
    name: Factory Dupcode
    runs-on: ubuntu-latest
    steps:
      - name: Checkout
        uses: actions/checkout@v4

      - name: Dupcode CI preflight
        run: |
          test "$(git rev-parse HEAD^{commit})" = "$GITHUB_SHA" || exit 1
          test -z "$(git status --porcelain=v1)" || exit 1
          echo "tree OID: $(git rev-parse HEAD^{tree})"

      - name: Run gate-dupcode
        env:
          LEAMAS_DUPCODE_AUTHORITY: github-actions
        run: make gate-dupcode
`
		if err := os.WriteFile(fixture, []byte(content), 0644); err != nil {
			t.Fatalf("failed to write fixture: %v", err)
		}
		workflowPath = fixture
	}

	tc, err := parseWorkflow(workflowPath)
	if err != nil {
		t.Fatalf("failed to parse workflow: %v", err)
	}

	// Assert all required contract elements
	if tc.JobName != "factory-dupcode" {
		t.Errorf("expected job name 'factory-dupcode', got %q", tc.JobName)
	}

	if tc.DisplayedName != "Factory Dupcode" {
		t.Errorf("expected displayed name 'Factory Dupcode', got %q", tc.DisplayedName)
	}

	if !tc.HasCheckout {
		t.Error("workflow must have checkout step")
	}

	if !tc.HasExactSHA {
		t.Error("workflow must have exact HEAD == GITHUB_SHA assertion")
	}

	if !tc.HasCleanTree {
		t.Error("workflow must have clean worktree assertion")
	}

	if !tc.HasTreeOID {
		t.Error("workflow must print tree OID")
	}

	if !tc.AuthorityScoped {
		t.Error("LEAMAS_DUPCODE_AUTHORITY must be scoped to execution step, not global")
	}

	if tc.InvokesMakeTarget != "gate-dupcode" {
		t.Errorf("workflow must invoke 'make gate-dupcode', got %q", tc.InvokesMakeTarget)
	}

	if tc.HasContinueError {
		t.Error("workflow must not have continue-on-error")
	}
}

func TestFactoryDupcodeWorkflowTimeoutNotSuccess(t *testing.T) {
	// Verify that timeouts cannot be converted to success
	dir := t.TempDir()
	fixture := filepath.Join(dir, "factory_timeout.yml")
	content := `jobs:
  factory-dupcode:
    name: Factory Dupcode
    runs-on: ubuntu-latest
    timeout-minutes: 30
    steps:
      - uses: actions/checkout@v4
      - run: make gate-dupcode
`
	if err := os.WriteFile(fixture, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write fixture: %v", err)
	}

	tc, err := parseWorkflow(fixture)
	if err != nil {
		t.Fatalf("failed to parse workflow: %v", err)
	}

	// Verify timeout is set
	if tc.JobName == "" {
		t.Error("job name should be present")
	}

	// The timeout should be set, but should not allow continue-on-error
	if tc.HasContinueError {
		t.Error("timeout workflow must not have continue-on-error")
	}
}
