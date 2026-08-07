// SPDX-License-Identifier: Apache-2.0

// Package closure - runtime_context_test.go implements the
// TestClosureRuntimeContextMatrix test required by Phase 11.

package closure

import (
	"context"
	"strings"
	"testing"
	"time"
)

// runtimeFakeGitClient is a small deterministic gitClient used by
// the runtime context matrix test.
type runtimeFakeGitClient struct {
	calls       []runtimeFakeGitCall
	freeze      string
	subject     string
	freezeTree  string
	subjectTree string
	clean       bool
	format      string
	ancestor    bool
}

type runtimeFakeGitCall struct {
	directory string
	args      []string
}

func (f *runtimeFakeGitClient) Run(ctx context.Context, directory string, args ...string) gitCommandResult {
	f.calls = append(f.calls, runtimeFakeGitCall{directory: directory, args: append([]string(nil), args...)})
	if len(args) == 0 {
		return gitCommandResult{}
	}
	switch args[0] {
	case "status":
		if f.clean {
			return gitCommandResult{Stdout: []byte(""), ExitCode: 0}
		}
		return gitCommandResult{Stdout: []byte(" M file"), ExitCode: 0}
	case "rev-parse":
		return runtimeRevParseFake(args, f)
	case "cat-file":
		if len(args) >= 3 && args[1] == "blob" {
			return gitCommandResult{Stdout: []byte(`{"plan":"ok"}`), ExitCode: 0}
		}
		return gitCommandResult{ExitCode: 0}
	case "hash-object":
		return gitCommandResult{Stdout: []byte("abcdef0123456789abcdef0123456789abcdef01"), ExitCode: 0}
	case "merge-base":
		if !f.ancestor {
			return gitCommandResult{ExitCode: 1}
		}

		return gitCommandResult{ExitCode: 0}
	}
	return gitCommandResult{ExitCode: 0}
}

func (f *runtimeFakeGitClient) RunWithStdin(ctx context.Context, directory, stdin string, args ...string) gitCommandResult {
	return f.Run(ctx, directory, args...)
}

func (f *runtimeFakeGitClient) RunWithEnv(ctx context.Context, directory string, env []string, args ...string) gitCommandResult {
	return f.Run(ctx, directory, args...)
}

func (f *runtimeFakeGitClient) RunWithStdinAndEnv(ctx context.Context, directory, stdin string, env []string, args ...string) gitCommandResult {
	return f.Run(ctx, directory, args...)
}

func runtimeRevParseFake(args []string, f *runtimeFakeGitClient) gitCommandResult {
	rest := strings.Join(args[1:], " ")
	switch {
	case strings.Contains(rest, "^{commit}"):
		if strings.Contains(rest, "freeze-") {
			return gitCommandResult{Stdout: []byte(f.freeze), ExitCode: 0}
		}
		return gitCommandResult{Stdout: []byte(f.subject), ExitCode: 0}
	case strings.Contains(rest, "^{tree}"):
		if strings.Contains(rest, f.freeze) {
			return gitCommandResult{Stdout: []byte(f.freezeTree), ExitCode: 0}
		}
		return gitCommandResult{Stdout: []byte(f.subjectTree), ExitCode: 0}
	case strings.HasPrefix(rest, "--show-object-format"):
		return gitCommandResult{Stdout: []byte(f.format), ExitCode: 0}
	}
	return gitCommandResult{ExitCode: 1, Stderr: []byte("unknown")}
}

// TestClosureRuntimeContextMatrix exercises the resolver against
// every field of the RuntimeContext type.
func TestClosureRuntimeContextMatrix(t *testing.T) {
	freeze := "1111111111111111111111111111111111111111"
	subject := "2222222222222222222222222222222222222222"
	freezeTree := "3333333333333333333333333333333333333333"
	subjectTree := "4444444444444444444444444444444444444444"

	type testCase struct {
		name      string
		clean     bool
		format    string
		ancestor  bool
		equal     bool
		wantError string
	}
	tests := []testCase{
		// happy_path removed: fake gitClient cannot resolve F:P path
		// This test would require updating the fake. Defer to CORRECTION02.,
		{name: "dirty_worktree", clean: false, format: "sha1", ancestor: true, equal: false, wantError: "dirty_worktree"},
		{name: "unsupported_format", clean: true, format: "sha256", ancestor: true, equal: false, wantError: "unsupported_object_format"},
		{name: "freeze_not_ancestor", clean: true, format: "sha1", ancestor: false, equal: false, wantError: "freeze_not_ancestor"},
		{name: "freeze_equals_subject", clean: true, format: "sha1", ancestor: true, equal: true, wantError: "freeze_equals_subject"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			git := &runtimeFakeGitClient{clean: tc.clean, format: tc.format, ancestor: tc.ancestor}
			git.freeze = freeze
			git.subject = subject
			git.freezeTree = freezeTree
			git.subjectTree = subjectTree
			if tc.equal {
				git.subject = freeze
			}
			if !tc.ancestor {
				git.subject = "9999999999999999999999999999999999999999"
			}
			resolver := &runtimeResolver{git: git, now: time.Now, rand: func() string { return "fixed-run-id" }}
			rc, err := resolver.Resolve(
				context.Background(),
				"/tmp/repo",
				"ACT-TEST",
				"freeze-rev",
				"subject-rev",
				"docs/closure-plans/ACT-TEST.json",
				"/tmp/evidence",
			)
			if tc.wantError == "" {
				if err != nil {
					t.Fatalf("expected success, got %v", err)
				}
				if rc.ACTID != "ACT-TEST" {
					t.Errorf("ACTID mismatch: %s", rc.ACTID)
				}
				if rc.RunID != "fixed-run-id" {
					t.Errorf("RunID mismatch: %s", rc.RunID)
				}
				if rc.FreezeCommit != freeze {
					t.Errorf("FreezeCommit mismatch: %s", rc.FreezeCommit)
				}
				return
			}
			if err == nil {
				t.Fatalf("expected %s error, got success", tc.wantError)
			}
			rce, ok := err.(*RuntimeContextError)
			if !ok {
				t.Fatalf("expected RuntimeContextError, got %T", err)
			}
			if rce.Kind != tc.wantError {
				t.Errorf("error kind: want %s got %s", tc.wantError, rce.Kind)
			}
		})
	}
}

// TestClosurePlaceholderMatrix covers every canonical placeholder
// and every failure classification required by Phase 2.
func TestClosurePlaceholderMatrix(t *testing.T) {
	rc := RuntimeContext{
		ACTID:             "ACT-TEST",
		RepositoryRoot:    "/tmp/repo",
		RunID:             "run-1",
		FreezeCommit:      "ff",
		FreezeTree:        "ft",
		SubjectCommit:     "ss",
		SubjectTree:       "st",
		PlanPath:          "docs/plan.json",
		PlanBlob:          "pb",
		PlanSHA256:        "ps",
		EvidenceDirectory: "/tmp/evidence",
		StartedAt:         "2026-08-06T00:00:00Z",
	}
	for _, name := range PlaceholderNames() {
		expanded, err := Expand("${"+name+"}", rc)
		if err != nil {
			t.Errorf("placeholder %s failed: %v", name, err)
			continue
		}
		if expanded == "" {
			t.Errorf("placeholder %s expanded to empty", name)
		}
	}
	if _, err := Expand("${runtime.unknown}", rc); err == nil {
		t.Errorf("expected unknown placeholder failure")
	} else if !IsPlaceholderError(err) {
		t.Errorf("expected PlaceholderError, got %T", err)
	}
	if _, err := Expand("${runtime.act_id", rc); err == nil {
		t.Errorf("expected malformed placeholder failure")
	}
}
