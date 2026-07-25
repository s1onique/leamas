// SPDX-License-Identifier: Apache-2.0

package closure

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestEvaluateRequiredPatchHygieneV2FailClosed(t *testing.T) {
	plan := requiredPolicyPlan()
	// Freeze commit F (must NOT equal plan.baseline — the explicit F..S
	// policy range decision depends on this distinction; see
	// PolicyRangeDecision in run_v2_policy.go).
	freeze := strings.Repeat("f", 40)
	tests := []struct {
		name       string
		result     gitCommandResult
		wantPassed bool
		wantErr    string
	}{
		{name: "positive", result: gitCommandResult{}, wantPassed: true},
		{name: "negative verdict", result: gitCommandResult{ExitCode: 2, Err: errors.New("exit status 2"), Stdout: []byte("file.go:1: trailing whitespace\n")}, wantErr: "required patch hygiene failed"},
		{name: "process failure", result: gitCommandResult{ExitCode: 128, Err: errors.New("exit status 128"), Stderr: []byte("fatal: bad object\n")}, wantErr: "required patch hygiene failed"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			git := &scriptedPolicyGit{results: []gitCommandResult{tt.result}}
			outcome, err := evaluateRequiredPatchHygieneV2(context.Background(), git, "/repo", plan, freeze, "subject")
			assertPolicyEvaluation(t, outcome.Passed, outcome.Diagnostics, err, tt.wantPassed, tt.wantErr)
		})
	}
}

func TestEvaluateRequiredClosurePolicyV2FailClosed(t *testing.T) {
	plan := requiredPolicyPlan()
	tests := []struct {
		name       string
		results    []gitCommandResult
		wantPassed bool
		wantErr    string
	}{
		{name: "positive", results: []gitCommandResult{{}}, wantPassed: true},
		{name: "negative verdict", results: []gitCommandResult{{Stdout: []byte("digest.txt\x00")}, {Stdout: append([]byte(nil), fullDigestMarker...)}}, wantErr: "required closure policy failed"},
		{name: "process failure", results: []gitCommandResult{{ExitCode: 128, Err: errors.New("exit status 128"), Stderr: []byte("fatal: bad range\n")}}, wantErr: "required closure policy failed"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			git := &scriptedPolicyGit{results: tt.results}
			outcome, err := evaluateRequiredClosurePolicyV2(context.Background(), git, "/repo", plan, "subject")
			assertPolicyEvaluation(t, outcome.Passed, outcome.Diagnostics, err, tt.wantPassed, tt.wantErr)
		})
	}
}

func requiredPolicyPlan() Plan {
	yes := true
	return Plan{
		Baseline: Baseline{CommitOID: strings.Repeat("a", 40)},
		Policy: PlanPolicy{
			RequireDiffCheck:         &yes,
			ForbidTrackedFullDigests: &yes,
		},
	}
}

func assertPolicyEvaluation(t *testing.T, passed bool, diagnostics []byte, err error, wantPassed bool, wantErr string) {
	t.Helper()
	if passed != wantPassed {
		t.Fatalf("Passed = %v, want %v", passed, wantPassed)
	}
	if wantErr == "" {
		if err != nil {
			t.Fatal(err)
		}
		return
	}
	if err == nil || !strings.Contains(err.Error(), wantErr) {
		t.Fatalf("error = %v, want substring %q", err, wantErr)
	}
	if len(diagnostics) == 0 {
		t.Fatal("failed evaluation must retain diagnostics")
	}
}

type scriptedPolicyGit struct {
	RealGit
	results []gitCommandResult
	calls   int
}

func (g *scriptedPolicyGit) Run(context.Context, string, ...string) gitCommandResult {
	if g.calls >= len(g.results) {
		return gitCommandResult{Err: errors.New("unexpected Git call"), ExitCode: -1}
	}
	result := g.results[g.calls]
	g.calls++
	return result
}
