// SPDX-License-Identifier: Apache-2.0

// binary_gate_failure_matrix_rows_test.go owns the
// per-row fakes and runner-configurations the strict
// 12-row matrix uses. Splitting the row definitions from
// the matrix schema (binary_gate_failure_matrix_test.go)
// keeps the matrix schema file under the LLM-friendly
// 400-line threshold while allowing each row to use a
// focused fake without bloating the main matrix file.
//
// The row data itself lives in binary_gate_failure_matrix_test.go;
// this file only owns the package-private fakes and helpers
// the rows reference.

package closure

import (
	"context"
	"errors"

	"github.com/s1onique/leamas/internal/factory/closure/evidence"
)

// r6BCleanupOnlyFakeGit is the focused fake git client
// row 12 uses. Unlike the broader subjectMatrixGitClient,
// this fake fails ONLY the worktree remove call so the
// cleanup authority is exercised WITHOUT disturbing the
// AFTER inventory observation. All other commands are
// delegated to the supplied git client (typically
// RealGit).
//
// The fake exists because the broader subjectMatrixGitClient
// is shared with the R6-A adversarial matrix; its
// cleanupFail flag cascades into the AFTER inventory
// observation, which would mask row 12's owned R6-B
// subject-cleanup authority with a subject_observation
// diagnostic.
type r6BCleanupOnlyFakeGit struct {
	delegate gitClient
}

func (m *r6BCleanupOnlyFakeGit) Run(ctx context.Context, directory string, args ...string) gitCommandResult {
	if len(args) >= 4 && args[0] == "worktree" && args[1] == "remove" {
		// Deterministic failure for `git worktree remove --force <path>`.
		// The 4-element form is what newSubjectWorktreeLifecycle uses.
		return gitCommandResult{
			ExitCode: 1,
			Stderr:   []byte("fatal: r6b-cleanup-only fake: deterministic cleanup failure"),
		}
	}
	if m.delegate == nil {
		return gitCommandResult{Err: errors.New("r6b-cleanup-only fake: no delegate")}
	}
	return m.delegate.Run(ctx, directory, args...)
}

func (m *r6BCleanupOnlyFakeGit) RunWithStdin(ctx context.Context, directory, stdin string, args ...string) gitCommandResult {
	if m.delegate == nil {
		return gitCommandResult{}
	}
	return m.delegate.RunWithStdin(ctx, directory, stdin, args...)
}

func (m *r6BCleanupOnlyFakeGit) RunWithEnv(ctx context.Context, directory string, env []string, args ...string) gitCommandResult {
	if m.delegate == nil {
		return gitCommandResult{}
	}
	return m.delegate.RunWithEnv(ctx, directory, env, args...)
}

func (m *r6BCleanupOnlyFakeGit) RunWithStdinAndEnv(ctx context.Context, directory, stdin string, env []string, args ...string) gitCommandResult {
	if m.delegate == nil {
		return gitCommandResult{}
	}
	return m.delegate.RunWithStdinAndEnv(ctx, directory, stdin, env, args...)
}

// r6BRealSubjectCleanupFailureGitClient returns a
// gitClient that fails ONLY `git worktree remove --force`
// and delegates every other command to the production
// RealGit. The seam is package-private; no production
// API is exposed.
func r6BRealSubjectCleanupFailureGitClient() gitClient {
	return &r6BCleanupOnlyFakeGit{delegate: RealGit{}}
}

// silence unused-import warnings if the file is ever
// pruned of helpers; keeps gofmt consistent.
var _ = evidence.GateCapture{}
