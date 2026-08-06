// SPDX-License-Identifier: Apache-2.0

// Package main - factory_close_execute.go implements the dry-run
// variant of `factory close execute` required by CORRECTION01.
// Full lifecycle mutation is rejected; only dry-run authority is
// exposed.

package main

import (
	"context"
	"encoding/json"
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
		return 2
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
		fmt.Fprintf(stderr, "factory close execute: %v\n", err)
		return 2
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
			return 2
		}
	}
	result, verdict, err := ExecuteCloseDryRun(context.Background(), req)
	if err != nil {
		fmt.Fprintf(stderr, "factory close execute: %v\n", err)
		return exitCodeForError(verdict)
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
	verdict := "PASS"
	if capture.ExitCode != 0 {
		verdict = "FAIL"
	}
	if capture.TimedOut || capture.StdoutTruncated || capture.StderrTruncated {
		verdict = "UNAVAILABLE"
	}
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

// exitCodeForVerdict is the deterministic mapping from the
// derived verdict to an exit code. A derived FAIL or UNAVAILABLE
// MUST never return 0.
func exitCodeForVerdict(verdict string) int {
	switch verdict {
	case "PASS":
		return 0
	case "FAIL":
		return 3
	default:
		return 4
	}
}

// exitCodeForError maps a Go error into the dry-run exit code.
// CLI/path/observation errors return 2; verification failures
// return 3; observer/publication failures return 4.
func exitCodeForError(verdict string) int {
	if verdict == "FAIL" {
		return 3
	}
	if verdict == "UNAVAILABLE" {
		return 4
	}
	return 2
}

// verdictForRuntimeError maps a runtime error into a verdict.
func verdictForRuntimeError(err error) string {
	if err == nil {
		return "PASS"
	}
	var rce *closure.RuntimeContextError
	if errAs(err, &rce) {
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

// errAs is the typed wrapper around errors.As used by the
// dry-run verdict mapper. It is intentionally unexported.
func errAs(err error, target interface{}) bool {
	type asTarget interface{}
	_, ok := target.(**closure.RuntimeContextError)
	if !ok {
		return false
	}
	if err == nil {
		return false
	}
	if e, ok := err.(*closure.RuntimeContextError); ok {
		_ = e
		return true
	}
	return false
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
