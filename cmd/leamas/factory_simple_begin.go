// SPDX-License-Identifier: Apache-2.0

// factory_simple_begin.go implements `leamas factory begin
// <ACT-ID>`. The command is a thin CLI wrapper around
// closure.BeginAct. It MUST NOT reimplement closure logic;
// its only job is:
//
//   1. Parse + validate the ACT ID (single positional arg).
//   2. Resolve the repository root.
//   3. Build closure.SimpleCloseDeps with the production
//      RealGit client.
//   4. Emit a machine-readable envelope.
//
// The CLI is fail-closed: it never invents a freeze commit
// and never hides a post-authority sync failure from the
// operator. On a post-commit sync failure the freeze_commit
// is still surfaced so the operator can verify F is real.

package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/s1onique/leamas/internal/factory/closure"
)

// simpleBeginResult is the canonical machine-readable
// envelope for `leamas factory begin`. The CLI is the ONLY
// place that adds this presentation layer; the closure
// package exposes the raw FrozenPlan.
type simpleBeginResult struct {
	ActID         string `json:"act_id"`
	FreezeCommit  string `json:"freeze_commit"`
	PlanPath      string `json:"plan_path"`
	State         string `json:"state"`
	AuthoritySync string `json:"authority_committed,omitempty"`
}

func handleFactoryBegin() {
	os.Exit(runFactoryBegin(os.Args[3:], os.Stdout, os.Stderr))
}

func runFactoryBegin(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("factory begin", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	jsonOutput := fs.Bool("json", false, "emit machine-readable JSON")
	if err := fs.Parse(args); err != nil {
		fmt.Fprintf(stderr, "factory begin: %v\n", err)
		return 2
	}
	rest := fs.Args()
	if len(rest) != 1 {
		fmt.Fprintln(stderr, "factory begin: usage: leamas factory begin <ACT-ID>")
		return 2
	}
	actID := strings.TrimSpace(rest[0])
	if actID == "" {
		fmt.Fprintln(stderr, "factory begin: ACT-ID is empty")
		return 2
	}

	repoRoot, err := resolveCLIRepoRoot()
	if err != nil {
		fmt.Fprintf(stderr, "factory begin: %v\n", err)
		return 1
	}
	deps := closure.SimpleCloseDeps{
		Git:            closure.NewRealGit(),
		RepositoryRoot: repoRoot,
		Remote:         "origin",
	}
	frozen, err := closure.BeginAct(context.Background(), deps, actID)
	if err != nil {
		// BeginAct returns FrozenPlan{FreezeCommit: fOID} on
		// post-authority sync failure so the operator can see
		// the authoritative F. Honor that contract here.
		result := simpleBeginResult{
			ActID:        actID,
			FreezeCommit: frozen.FreezeCommit,
			PlanPath:     frozen.PlanPath,
			State:        "failed",
		}
		if frozen.FreezeCommit != "" && strings.Contains(err.Error(), "post_commit_sync_failed") {
			result.AuthoritySync = "true"
			result.State = "post_commit_sync_failed"
		}
		if *jsonOutput {
			_ = json.NewEncoder(stdout).Encode(result)
		} else {
			fmt.Fprintf(stderr, "factory begin: %v\n", err)
		}
		return 1
	}
	result := simpleBeginResult{
		ActID:        actID,
		FreezeCommit: frozen.FreezeCommit,
		PlanPath:     frozen.PlanPath,
		State:        "frozen",
	}
	if *jsonOutput {
		_ = json.NewEncoder(stdout).Encode(result)
	} else {
		fmt.Fprintf(stdout, "act_id=%s freeze_commit=%s plan_path=%s state=%s\n",
			result.ActID, result.FreezeCommit, result.PlanPath, result.State)
	}
	return 0
}

// resolveCLIRepoRoot finds the canonical repository root via
// `git rev-parse --show-toplevel`. The CLI runs against the
// operator's current working directory. This is the only
// piece of CLI plumbing that touches Git directly; the rest
// of the work is delegated to the closure package.
func resolveCLIRepoRoot() (string, error) {
	out, err := exec.Command("git", "rev-parse", "--show-toplevel").Output()
	if err != nil {
		return "", fmt.Errorf("locate repository root: %w", err)
	}
	root := strings.TrimSpace(string(out))
	if root == "" {
		return "", errors.New("repository root is empty")
	}
	return root, nil
}
