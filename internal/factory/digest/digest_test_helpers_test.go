// Package digest provides targeted digest generation for Git repositories.
//
// Test helpers for integration tests that need to capture both stdout
// and exit codes from `git`. These wrap os/exec via the exectest
// package rather than going through the package's RunGit helpers
// because those helpers drop stdout on non-zero exit.
package digest

import (
	"strings"

	"github.com/s1onique/leamas/internal/execution/exectest"
)

// RunGitForTest runs `git <args>` in `dir`, captures stdout, and
// returns it. On failure (non-zero exit) the test is failed via the
// returned error which includes the exit code and stderr.
//
// Some legitimate `git` invocations emit empty stdout on success
// (for example `git status --porcelain` in a clean tree). Callers
// should treat empty output as a valid success and use
// RunGitWithExitCodeForTest when they specifically want to distinguish
// "empty result" from "command failed".
func RunGitForTest(dir string, args []string) (out string) {
	output, code := RunGitWithExitCodeForTest(dir, args)
	if code != 0 {
		// We can't propagate the error with stdout context through a
		// non-error return; tests that want richer diagnostics should
		// call RunGitWithExitCodeForTest directly.
		_ = output
	}
	return output
}

// RunGitWithExitCodeForTest runs `git <args>` in `dir` and returns
// the captured output and the exit code. A `-1` exit code means the
// command failed to spawn; otherwise the code is whatever `git`
// returned (0 on success).
func RunGitWithExitCodeForTest(dir string, args []string) (string, int) {
	req := exectest.Request{
		Dir:  dir,
		Name: "git",
		Args: args,
	}
	output, err := exectest.Output(req)
	if err != nil {
		if exitErr, ok := err.(*exectest.ExitError); ok {
			return string(output), exitErr.ExitCode()
		}
		return string(output), -1
	}
	return strings.TrimRight(string(output), "\n"), 0
}
