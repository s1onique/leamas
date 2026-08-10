// SPDX-License-Identifier: Apache-2.0
// factory_close_execute.go wires the
// `leamas factory close execute` CLI command required by
// ACT-LEAMAS-FACTORY-CLOSURE-RUNTIME-CONTEXT-AND-EXECUTE01-CORRECTION02.
//
// The command is the public, immutable-subject execution
// pipeline. It dispatches into the production v2 runner
// (closure.RunClosureProtocolV2WithBinary) and reports exit
// codes per the published contract:
//
//	0 PASS
//	2 request / topology / policy rejection
//	3 authoritative verification failure
//	4 observer / execution / publication unavailable
//
// The flag set is fixed and minimal:
//
//	--repository         <repo>             caller repo root
//	--act-id             <ACT>              ACT identifier
//	--freeze             <F>                frozen plan commit
//	--subject            <S>                immutable subject commit
//	--plan-path          <P>                path of plan in repo
//	--evidence-directory <outside>          detached evidence dir
//	--json                                  emit a single JSON doc
//
// Strict JSON mode emits exactly one JSON document on stdout.
// Non-JSON mode emits a short text summary on stdout and a
// diagnostic summary on stderr.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/s1onique/leamas/internal/factory/closure"
)

// inventoryToCanonicalWorktrees adapts the B3 inventory
// type to the orchestrator's CanonicalWorktree slice. The
// inventory already enforces the absolute / clean / dedup
// invariants; the adapter is a pure copy.
func inventoryToCanonicalWorktrees(inv closure.RepositoryWorktreeInventory) []closure.CanonicalWorktree {
	roots := inv.RootsView()
	out := make([]closure.CanonicalWorktree, 0, len(roots))
	for _, r := range roots {
		out = append(out, closure.CanonicalWorktree{Path: r})
	}
	return out
}

// executeFlags captures the parsed `factory close execute`
// flags. Every field maps 1:1 to a published --flag.
type executeFlags struct {
	Repository        string
	ACTID             string
	Freeze            string
	Subject           string
	PlanPath          string
	EvidenceDirectory string
	JSONOutput        bool
}

// parseExecuteFlags parses the `factory close execute` flags.
// The function is total: any missing flag produces a typed
// error so the CLI can surface the exact missing field.
func parseExecuteFlags(args []string) (executeFlags, error) {
	var f executeFlags
	for i := 0; i < len(args); i++ {
		arg := args[i]
		consume := func() (string, error) {
			if i+1 >= len(args) {
				return "", fmt.Errorf("flag %s requires a value", arg)
			}
			i++
			return args[i], nil
		}
		switch arg {
		case "--repository":
			v, err := consume()
			if err != nil {
				return f, err
			}
			f.Repository = v
		case "--act-id":
			v, err := consume()
			if err != nil {
				return f, err
			}
			f.ACTID = v
		case "--freeze":
			v, err := consume()
			if err != nil {
				return f, err
			}
			f.Freeze = v
		case "--subject":
			v, err := consume()
			if err != nil {
				return f, err
			}
			f.Subject = v
		case "--plan-path":
			v, err := consume()
			if err != nil {
				return f, err
			}
			f.PlanPath = v
		case "--evidence-directory":
			v, err := consume()
			if err != nil {
				return f, err
			}
			f.EvidenceDirectory = v
		case "--json":
			f.JSONOutput = true
		default:
			return f, fmt.Errorf("unknown flag: %s", arg)
		}
	}
	// Required-field check in deterministic order so the CLI
	// surfaces the first missing field consistently.
	required := []struct {
		name, value string
	}{
		{"--repository", f.Repository},
		{"--act-id", f.ACTID},
		{"--freeze", f.Freeze},
		{"--subject", f.Subject},
		{"--plan-path", f.PlanPath},
		{"--evidence-directory", f.EvidenceDirectory},
	}
	for _, r := range required {
		if strings.TrimSpace(r.value) == "" {
			return f, fmt.Errorf("%s is required", r.name)
		}
	}
	if !filepath.IsAbs(f.Repository) {
		return f, fmt.Errorf("--repository must be an absolute path")
	}
	if !filepath.IsAbs(f.EvidenceDirectory) {
		return f, fmt.Errorf("--evidence-directory must be an absolute path")
	}
	if f.Repository == f.EvidenceDirectory {
		return f, fmt.Errorf("--evidence-directory must differ from --repository")
	}
	return f, nil
}

// executeEnvelope is the single JSON document the strict
// mode emits. The struct is intentionally flat so callers
// never have to walk nested objects.
type executeEnvelope struct {
	OK                bool                  `json:"ok"`
	ACTID             string                `json:"act_id"`
	Repository        string                `json:"repository"`
	Freeze            string                `json:"freeze"`
	Subject           string                `json:"subject"`
	PlanPath          string                `json:"plan_path"`
	EvidenceDirectory string                `json:"evidence_directory"`
	Manifest          *closure.V2Manifest   `json:"manifest,omitempty"`
	Diagnostics       closure.V2Diagnostics `json:"diagnostics,omitempty"`
	ErrorCode         string                `json:"error_code,omitempty"`
}

// RunFactoryCloseExecute is the public entry point for
// `leamas factory close execute`. The function is total:
// every error path returns a typed exit code and a JSON or
// text envelope.
func RunFactoryCloseExecute(args []string, stdout, stderr io.Writer) int {
	flags, err := parseExecuteFlags(args)
	if err != nil {
		return executeUsageError(stderr, err)
	}
	// The runner refuses to run unless the caller repository
	// is a real path. Reject empty paths explicitly so we do
	// not accidentally execute in the runner's working dir.
	if _, statErr := os.Stat(flags.Repository); statErr != nil {
		return executeEmit(flags, stdout, stderr, executeResult{
			exitCode: 2,
			errCode:  "repository_unavailable",
			diag: closure.V2Diagnostics{{
				Code:         closure.V2CodeGitOperationFailed,
				Message:      fmt.Sprintf("repository root unavailable: %s", statErr.Error()),
				PropertyName: "repository",
			}},
		})
	}
	identity, err := captureRunningBinaryIdentity()
	if err != nil {
		return executeEmit(flags, stdout, stderr, executeResult{
			exitCode: 4,
			errCode:  "binary_identity_invalid",
			diag: closure.V2Diagnostics{{
				Code:         closure.V2CodeBinaryIdentityInvalid,
				Message:      fmt.Sprintf("capture running binary identity: %s", err.Error()),
				PropertyName: "leamas_binary_identity",
			}},
		})
	}
	if err := closure.ValidateV2BinaryIdentity(identity); err != nil {
		return executeEmit(flags, stdout, stderr, executeResult{
			exitCode: 4,
			errCode:  "binary_identity_invalid",
			diag: closure.V2Diagnostics{{
				Code:         closure.V2CodeBinaryIdentityInvalid,
				Message:      err.Error(),
				PropertyName: "leamas_binary_identity",
			}},
		})
	}
	// Inventory every linked worktree via the existing
	// authority. The B3 publisher MUST refuse to write inside
	// any of these paths. We do not reconstruct the inventory
	// from one root.
	inventory, err := closure.InventoryRepositoryWorktrees(context.Background(), flags.Repository, nil)
	if err != nil {
		return executeEmit(flags, stdout, stderr, executeResult{
			exitCode: 4,
			errCode:  "worktree_inventory_unavailable",
			diag: closure.V2Diagnostics{{
				Code:         closure.V2CodeWorktreeInventoryUnavailable,
				Message:      fmt.Sprintf("inventory linked worktrees: %s", err.Error()),
				PropertyName: "worktree_inventory",
			}},
		})
	}
	// Derive the canonical evidence file under the
	// --evidence-directory. The directory is the user-facing
	// option; the B3 publisher requires a .json filename.
	evidenceFile := filepath.Join(flags.EvidenceDirectory, "closure-evidence.json")
	req := closure.V2Request{
		ClosureProtocolVersion: closure.ClosureProtocolV2,
		PlanContractVersion:    1,
		RepositoryRoot:         flags.Repository,
		SubjectCommit:          flags.Subject,
		FreezeCommit:           flags.Freeze,
		PlanPath:               flags.PlanPath,
		EvidenceDirectory:      flags.EvidenceDirectory,
		ManifestOutput:         filepath.Join(flags.EvidenceDirectory, "manifest.json"),
	}
	// B3-R3: route through the B2+B3 publication
	// orchestrator. The orchestrator derives the B2 inputs
	// from the runner's authoritative V2ExecutionObservation
	// (captured by the non-publishing runner entry point);
	// a caller cannot smuggle a fabricated CandidateInputs
	// bundle past the B2 barrier.
	orchestrator := &closure.EvidencePublicationOrchestrator{
		Runner:              closure.RunClosureProtocolV2Execute,
		RepositoryRoot:      flags.Repository,
		EvidenceDestination: evidenceFile,
		Worktrees:           inventoryToCanonicalWorktrees(inventory),
	}
	manifest, _, runErr := orchestrator.PublishEvidence(
		context.Background(), req, identity,
	)
	if runErr != nil {
		// Map inner V2Error to typed exit code via the
		// diagnostic code. Any error that lacks a typed
		// V2Diagnostic collapses to exit 3 (authoritative
		// verification failure) because the inner runner
		// failed before reaching the publication barrier.
		return executeEmit(flags, stdout, stderr, classifyExecuteError(runErr))
	}
	return executeEmit(flags, stdout, stderr, executeResult{
		ok:       true,
		manifest: &manifest,
	})
}


// executeEmit renders the result envelope and returns the
// correct exit code. JSON mode emits exactly one document on
// stdout; non-JSON mode emits a short summary on stdout and a
// diagnostic summary on stderr.
func executeEmit(flags executeFlags, stdout, stderr io.Writer, r executeResult) int {
	env := executeEnvelope{
		OK:                r.ok,
		ACTID:             flags.ACTID,
		Repository:        flags.Repository,
		Freeze:            flags.Freeze,
		Subject:           flags.Subject,
		PlanPath:          flags.PlanPath,
		EvidenceDirectory: flags.EvidenceDirectory,
		Manifest:          r.manifest,
		Diagnostics:       r.diag,
		ErrorCode:         r.errCode,
	}
	if flags.JSONOutput {
		data, err := json.MarshalIndent(env, "", "  ")
		if err != nil {
			fmt.Fprintf(stderr, "factory close execute: marshal envelope: %s\n", err.Error())
			return 4
		}
		fmt.Fprintln(stdout, string(data))
	} else {
		if r.ok && r.manifest != nil {
			fmt.Fprintln(stdout, "OK")
			fmt.Fprintf(stderr,
				"Closure Protocol v2 execute: act=%s S=%s F=%s execution_tree=%s\n",
				flags.ACTID,
				trunc8(r.manifest.SubjectCommit),
				trunc8(r.manifest.FreezeCommit),
				trunc8(r.manifest.ExecutionTree),
			)
		} else {
			fmt.Fprintf(stdout, "ERROR %s\n", r.errCode)
			for _, d := range r.diag {
				fmt.Fprintf(stderr, "  %s: %s\n", d.Code, d.Message)
			}
		}
	}
	if r.ok {
		return 0
	}
	return r.exitCode
}

// executeUsageError prints a usage hint and returns exit 2.
func executeUsageError(stderr io.Writer, err error) int {
	fmt.Fprintf(stderr, "factory close execute: %s\n", err.Error())
	return 2
}
