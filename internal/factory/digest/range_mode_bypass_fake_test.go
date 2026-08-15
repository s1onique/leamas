// SPDX-License-Identifier: Apache-2.0

// Package digest provides targeted digest generation for Git repositories.
//
// range_mode_bypass_fake_test.go contains the fakeGitRunner implementation
// for testing range mode bypass scenarios.
package digest

import (
	"fmt"
	"os/exec"
	"strings"
)

// fakeGitRunner records all invocations and can be
// configured to fail specific commands.
type fakeGitRunner struct {
	Commands     [][]string
	FailPatterns []string
	UseRealGit   bool
	// CatFileOutput: when non-empty, RunWithStdin
	// for cat-file --batch-check returns this
	// string instead of empty output. Lets tests
	// simulate short, complete, malformed, or
	// multi-record batch responses without
	// touching real git. F24 (CORRECTION06): the
	// short-output integration test now drives
	// rangeBlobOIDsBatch end-to-end through this
	// field instead of merely exercising the
	// parser + zero-value property.
	CatFileOutput string
}

// Run implements GitRunner, recording invocations and optionally failing.
func (f *fakeGitRunner) Run(repoRoot string, args []string) (string, error) {
	f.Commands = append(f.Commands, args)
	for _, pattern := range f.FailPatterns {
		if matchesPattern(args, pattern) {
			return "", fmt.Errorf("fakegit: simulated failure for %v", args)
		}
	}
	if f.UseRealGit {
		output, exitCode := RunGitWithExitCode(repoRoot, args)
		if exitCode != 0 {
			return output, fmt.Errorf("git %s failed with exit %d", strings.Join(args, " "), exitCode)
		}
		return output, nil
	}
	return "", nil
}

// RunWithStdin implements the F15 (CORRECTION02) batched
// primitive. The fake records the call and delegates to
// RunGitWithExitCodeForTest when running through real git.
// Tests that care about the batch output use CatFileOutput
// (F24 CORRECTION06) or fail it via FailPatterns.
func (f *fakeGitRunner) RunWithStdin(repoRoot string, args []string,
	input string) (string, error) {
	f.Commands = append(f.Commands, args)
	for _, pattern := range f.FailPatterns {
		if matchesPattern(args, pattern) {
			return "", fmt.Errorf("fakegit: simulated failure for %v", args)
		}
	}
	// F24 (CORRECTION06): when CatFileOutput is set
	// and the call targets cat-file --batch-check,
	// return that exact stdout. Lets tests drive
	// short / malformed / multi-record responses
	// without invoking real git.
	if f.CatFileOutput != "" && len(args) >= 2 &&
		args[0] == "cat-file" && args[1] == "--batch-check" {
		return f.CatFileOutput, nil
	}
	if f.UseRealGit {
		// Pipe `input` into the git process. We use
		// exec.Command directly here; the rest of the
		// fake exposes the same surface.
		cmd := exec.Command("git", args...)
		cmd.Dir = repoRoot
		cmd.Stdin = strings.NewReader(input)
		output, err := cmd.Output()
		if err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok {
				return string(output), fmt.Errorf("git %s failed with exit %d", strings.Join(args, " "), exitErr.ExitCode())
			}
			return string(output), err
		}
		return string(output), nil
	}
	return "", nil
}

// matchesPattern checks if args match a pattern with required and forbidden tokens.
func matchesPattern(args []string, pattern string) bool {
	required := []string{}
	forbidden := []string{}
	for _, tok := range strings.Split(strings.TrimSpace(pattern), " ") {
		if tok == "" {
			continue
		}
		if strings.HasPrefix(tok, "!") {
			forbidden = append(forbidden, tok[1:])
		} else {
			required = append(required, tok)
		}
	}
	for _, req := range required {
		found := false
		for _, arg := range args {
			if arg == req {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	for _, forb := range forbidden {
		for _, arg := range args {
			if arg == forb {
				return false
			}
		}
	}
	return true
}

func sliceContainsAll(slice, items []string) bool {
	for _, item := range items {
		found := false
		for _, s := range slice {
			if s == item {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

func sliceContainsItem(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
