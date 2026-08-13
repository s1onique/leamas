// SPDX-License-Identifier: Apache-2.0

// factory_simple_close.go implements the simplified close
// invocation: `leamas factory close --act <ACT-ID> --subject
// <S> --lane fast [--publish]`. The command is a thin CLI
// wrapper around closure.SimpleClose. It MUST NOT reimplement
// closure logic; its only job is:
//
//   1. Parse + validate the four flags.
//   2. Resolve the repository root.
//   3. Build closure.SimpleCloseDeps with the production
//      RealGit client.
//   4. Emit the typed SimpleCloseResult envelope.
//
// The CLI is detected at the top of `leamas factory close`
// dispatch: if the first non-flag token looks like a known
// subcommand (`plan`, `run`, ...), the legacy dispatcher runs;
// otherwise the simplified path handles the invocation. This
// preserves backward compatibility for the existing close
// subcommands while giving operators the canonical UX.

package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"strings"

	"github.com/s1onique/leamas/internal/factory/closure"
)

// simpleCloseCommandLSIT is the list of legacy factory-close
// subcommands. When the first positional argument matches one
// of these, dispatch is delegated to the legacy handler.
// Otherwise, the simplified close path runs.
var simpleCloseCommandLSIT = map[string]bool{
	"plan":                true,
	"run":                 true,
	"run-v2-authority":    true,
	"verify-v2-authority": true,
	"verify":              true,
	"render":              true,
	"tag":                 true,
	"status":              true,
	"chain":               true,
	"attest":              true,
	"execute":             true,
}

// runFactorySimpleClose is the simplified close path. It is
// invoked when the first positional arg is NOT a legacy
// subcommand. All four canonical flags are required except
// --publish (which is optional).
func runFactorySimpleClose(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("factory close", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	var actID, subject, lane string
	var publish, jsonOutput bool
	fs.StringVar(&actID, "act", "", "ACT identifier (required)")
	fs.StringVar(&subject, "subject", "", "committed subject OID (required)")
	fs.StringVar(&lane, "lane", "", "verification lane (required; only fast is supported)")
	fs.BoolVar(&publish, "publish", false, "push to remote after closure passes")
	fs.BoolVar(&jsonOutput, "json", false, "emit machine-readable JSON")
	if err := fs.Parse(args); err != nil {
		fmt.Fprintf(stderr, "factory close: %v\n", err)
		return 2
	}

	repoRoot, err := resolveCLIRepoRoot()
	if err != nil {
		fmt.Fprintf(stderr, "factory close: %v\n", err)
		return 1
	}
	deps := closure.SimpleCloseDeps{
		Git:            closure.NewRealGit(),
		RepositoryRoot: repoRoot,
		Remote:         "origin",
	}
	req := closure.SimpleCloseRequest{
		ActID:   strings.TrimSpace(actID),
		Subject: strings.TrimSpace(subject),
		Lane:    strings.TrimSpace(lane),
		Publish: publish,
	}
	result, err := closure.SimpleClose(context.Background(), req, deps)
	if err != nil {
		if jsonOutput {
			_ = json.NewEncoder(stdout).Encode(result)
		} else {
			fmt.Fprintf(stderr, "factory close: %v\n", err)
		}
		return 1
	}
	if jsonOutput {
		_ = json.NewEncoder(stdout).Encode(result)
	} else {
		fmt.Fprintf(stdout,
			"act_id=%s freeze_commit=%s subject_commit=%s subject_tree=%s "+
				"closure_commit=%s closure_tree=%s verdict=%s state=%s "+
				"rerun_required=%t published=%t publication_head=%s reason_code=%s\n",
			result.ActID, result.FreezeCommit, result.SubjectCommit,
			result.SubjectTree, result.ClosureCommit, result.ClosureTree,
			result.Verdict, result.State, result.RerunRequired,
			result.Published, result.PublicationHead, result.ReasonCode)
	}
	if result.State != "fixed_point" || result.Verdict != "PASS" {
		return 1
	}
	return 0
}

// isLegacyCloseSubcommand reports whether the first
// positional argument of `factory close` is one of the
// legacy subcommands. Used by the dispatcher to choose
// between legacy and simplified paths.
func isLegacyCloseSubcommand(arg string) bool {
	if strings.HasPrefix(arg, "-") {
		return false
	}
	return simpleCloseCommandLSIT[arg]
}

// guardUnusedImport satisfies the compiler if errors is
// unreferenced (it IS referenced by resolveCLIRepoRoot in
// factory_simple_begin.go but kept here for symmetry).
var _ = errors.New
