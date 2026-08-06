// SPDX-License-Identifier: Apache-2.0

// Package main - factory_close_execute.go implements the dry-run
// variant of `factory close execute` required by CORRECTION01-R1.
// Full lifecycle mutation remains deferred to CORRECTION02.

package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/s1onique/leamas/internal/factory/closure"
	"github.com/s1onique/leamas/internal/factory/closure/evidence"
)

// ExecuteDryRunRequest parameterises the dry-run entry point.
type ExecuteDryRunRequest struct {
	RepositoryRoot    string
	ACTID             string
	FreezeRevision    string
	SubjectRevision   string
	PlanPath          string
	EvidenceDirectory string
	JSON              bool
}

// RunFactoryCloseExecute is the public dry-run entry point.
func RunFactoryCloseExecute(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "factory close execute: missing flags")
		printExecuteUsage(stderr)
		return exitRequestError
	}
	fs := flag.NewFlagSet("factory close execute", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	req := ExecuteDryRunRequest{}
	fs.StringVar(&req.RepositoryRoot, "repository", "", "absolute path to target repository")
	fs.StringVar(&req.ACTID, "act-id", "", "ACT identifier")
	fs.StringVar(&req.FreezeRevision, "freeze", "", "freeze revision (commit-ish)")
	fs.StringVar(&req.SubjectRevision, "subject", "", "subject revision (commit-ish)")
	fs.StringVar(&req.PlanPath, "plan-path", "", "repository-relative plan path")
	fs.StringVar(&req.EvidenceDirectory, "evidence-directory", "", "absolute evidence directory outside the repository")
	fs.BoolVar(&req.JSON, "json", false, "emit JSON-formatted output")
	if err := fs.Parse(args); err != nil {
		fmt.Fprintln(stderr, "factory close execute: ", err)
		return exitRequestError
	}
	for _, field := range []struct {
		name, value string
	}{
		{"--repository", req.RepositoryRoot},
		{"--act-id", req.ACTID},
		{"--freeze", req.FreezeRevision},
		{"--subject", req.SubjectRevision},
		{"--plan-path", req.PlanPath},
		{"--evidence-directory", req.EvidenceDirectory},
	} {
		if strings.TrimSpace(field.value) == "" {
			fmt.Fprintf(stderr, "factory close execute: %s is required\n", field.name)
			printExecuteUsage(stderr)
			return exitRequestError
		}
	}
	result, verdict, err := ExecuteCloseDryRun(context.Background(), req)
	if err != nil {
		fmt.Fprintln(stderr, "factory close execute: ", err)
		return exitCodeForVerdict(verdict)
	}
	if req.JSON {
		encoded, _ := json.MarshalIndent(result, "", "  ")
		fmt.Fprintln(stdout, string(encoded))
		return exitCodeForVerdict(verdict)
	}
	fmt.Fprintf(stdout, "freeze_commit=%s\n", result.FreezeCommit)
	fmt.Fprintf(stdout, "subject_commit=%s\n", result.SubjectCommit)
	fmt.Fprintf(stdout, "verdict=%s\n", verdict)
	return exitCodeForVerdict(verdict)
}

// Exit codes follow the dry-run contract.
const (
	exitPass         = 0
	exitRequestError = 2
	exitFail         = 3
	exitUnavailable  = 4
)

// ExecuteCloseDryRun performs the dry-run pipeline and returns
// the typed result plus the derived verdict.
func ExecuteCloseDryRun(ctx context.Context, req ExecuteDryRunRequest) (ExecuteCloseResult, string, error) {
	resolver := closure.NewRuntimeContextResolver()
	rc, err := resolver.Resolve(
		ctx,
		req.RepositoryRoot,
		req.ACTID,
		req.FreezeRevision,
		req.SubjectRevision,
		req.PlanPath,
		req.EvidenceDirectory,
	)
	if err != nil {
		return ExecuteCloseResult{}, verdictForRuntimeError(err), err
	}
	collector := evidence.NewGateCollector(nil)
	capture, err := collector.Capture(ctx, evidence.GateCaptureRequest{
		RepositoryRoot: rc.RepositoryRoot,
		SubjectRoot:    rc.RepositoryRoot,
		EvidenceDir:    filepath.Join(rc.EvidenceDirectory, "gate"),
		RunID:          rc.RunID,
	})
	if err != nil {
		return ExecuteCloseResult{}, "UNAVAILABLE", err
	}
	// The dry-run verdict MUST come from the classifier, not
	// from the raw gate exit code.
	verdict := string(evidence.ClassifyACTOwnedGate(evidence.ClassificationInputs{
		ObservedStatus:   capture.ExecGateObservedStatus,
		ObservedFindings: capture.PreExistingFindings,
		BaselineFindings: nil,
		ACTOwnedPaths:    nil,
		LaneMissing:      capture.ExecGateObservedStatus == "UNKNOWN",
		LaneTimedOut:     capture.TimedOut,
		LaneTruncated:    capture.StdoutTruncated || capture.StderrTruncated,
	}))
	result := ExecuteCloseResult{
		FreezeCommit:  rc.FreezeCommit,
		SubjectCommit: rc.SubjectCommit,
	}
	return result, verdict, nil
}

// ExecuteCloseResult is the typed result.
type ExecuteCloseResult struct {
	FreezeCommit  string `json:"freeze_commit"`
	SubjectCommit string `json:"subject_commit"`
}

// exitCodeForVerdict is the deterministic mapping from verdict
// to exit code.
func exitCodeForVerdict(verdict string) int {
	switch verdict {
	case "PASS":
		return exitPass
	case "FAIL":
		return exitFail
	default:
		return exitUnavailable
	}
}

// verdictForRuntimeError uses the standard errors.As to map a
// RuntimeContextError into the dry-run verdict. The function
// never panics: if no typed error matches, the verdict is
// UNAVAILABLE.
func verdictForRuntimeError(err error) string {
	if err == nil {
		return "PASS"
	}
	var rce *closure.RuntimeContextError
	if errors.As(err, &rce) {
		switch rce.Kind {
		case "dirty_worktree", "freeze_equals_subject", "freeze_not_ancestor",
			"unsupported_object_format", "plan_path_invalid", "empty_field":
			return "FAIL"
		default:
			return "UNAVAILABLE"
		}
	}
	return "UNAVAILABLE"
}

func printExecuteUsage(w io.Writer) {
	fmt.Fprintf(w, `usage: leamas factory close execute --repository <path> --act-id <id> --freeze <rev> --subject <rev> --plan-path <path> --evidence-directory <outside-repo>

Mandatory flags:
  --repository           absolute path to target repository
  --act-id               ACT identifier (e.g. ACT-LEAMAS-...-01)
  --freeze               freeze revision (commit-ish)
  --subject              subject revision (commit-ish)
  --plan-path            repository-relative plan path
  --evidence-directory   absolute path outside the target repository

Optional flags:
  --json                 emit JSON-formatted output

Note: this command runs in dry-run mode. Full lifecycle mutation
is deferred to a follow-up ACT.
`)
}
