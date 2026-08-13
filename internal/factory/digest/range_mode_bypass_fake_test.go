// SPDX-License-Identifier: Apache-2.0

// Package digest provides targeted digest generation for Git repositories.
//
// range_mode_bypass_fake_test.go contains the fakeGitRunner implementation
// for testing range mode bypass scenarios.
package digest

import (
	"fmt"
	"strings"
)

// fakeGitRunner records all invocations and can be configured to fail specific commands.
type fakeGitRunner struct {
	Commands     [][]string
	FailPatterns []string
	UseRealGit  bool
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
