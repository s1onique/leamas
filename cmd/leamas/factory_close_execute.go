// SPDX-License-Identifier: Apache-2.0

// Package main - factory_close_execute.go implements the dry-run
// variant of `factory close execute` required by CORRECTION01-R1-R1.
// The exact exit taxonomy is frozen: request -> 2, verification ->
// 3, observer -> 4, PASS -> 0.

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

// ExecuteFailureClass is the typed exit taxonomy.
type ExecuteFailureClass string

const (
	ExecuteRequest      ExecuteFailureClass = "request"
	ExecuteVerification ExecuteFailureClass = "verification"
	ExecuteObserver     ExecuteFailureClass = "observer"
)

// Exit codes follow the dry-run contract.
const (
	exitPass    = 0
	exitRequest = 2
	exitVerify  = 3
	exitObserve = 4
)

// ExecuteDryRunRequest parameterises the dry-run entry point.
type ExecuteDryRunRequest struct {
	RepositoryRoot    string
	ACTID             string
	FreezeRevision    string
	SubjectRevision   string
	PlanPath          string
	EvidenceDirectory string
	GatePolicy        GatePolicy
	JSON              bool
}

// GatePolicy supplies the authority the classifier needs.
type GatePolicy struct {
	BaselineFindings []evidence.GateFinding
	ACTOwnedPaths    []string
}

// RunFactoryCloseExecute is the public dry-run entry point.
func RunFactoryCloseExecute(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "factory close execute: missing flags")
		printExecuteUsage(stderr)
		return exitRequest
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
		return exitRequest
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
			return exitRequest
		}
	}
	result, class, err := ExecuteCloseDryRun(context.Background(), req)
	if err != nil {
		fmt.Fprintln(stderr, "factory close execute: ", err)
		return exitCodeForClass(class)
	}
	if req.JSON {
		encoded, _ := json.MarshalIndent(result, "", "  ")
		fmt.Fprintln(stdout, string(encoded))
		return exitCodeForClass(class)
	}
	fmt.Fprintf(stdout, "freeze_commit=%s\n", result.FreezeCommit)
	fmt.Fprintf(stdout, "subject_commit=%s\n", result.SubjectCommit)
	fmt.Fprintf(stdout, "verdict=%s\n", class)
	return exitCodeForClass(class)
}

// ExecuteCloseDryRun performs the dry-run pipeline and returns
// the typed result plus the failure class.
func ExecuteCloseDryRun(ctx context.Context, req ExecuteDryRunRequest) (ExecuteCloseResult, ExecuteFailureClass, error) {
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
		return ExecuteCloseResult{}, classifyRuntimeError(err), err
	}
	// An absent policy is a fail-closed observer failure; the
	// dry-run never treats nil baseline as an authoritative
	// empty set.
	if len(req.GatePolicy.BaselineFindings) == 0 && len(req.GatePolicy.ACTOwnedPaths) == 0 {
		return ExecuteCloseResult{}, ExecuteObserver, errors.New("gate policy authority absent")
	}
	collector := evidence.NewGateCollector(nil)
	capture, captureErr := collector.Capture(ctx, evidence.GateCaptureRequest{
		RepositoryRoot: rc.RepositoryRoot,
		SubjectRoot:    rc.RepositoryRoot,
		EvidenceDir:    filepath.Join(rc.EvidenceDirectory, "gate"),
		RunID:          rc.RunID,
	})
	if captureErr != nil {
		return ExecuteCloseResult{}, ExecuteObserver, captureErr
	}
	verdict := evidence.ClassifyACTOwnedGate(evidence.ClassificationInputs{
		ObservedStatus:   capture.ExecGateObservedStatus,
		ObservedFindings: capture.PreExistingFindings,
		BaselineFindings: req.GatePolicy.BaselineFindings,
		ACTOwnedPaths:    req.GatePolicy.ACTOwnedPaths,
		LaneMissing:      capture.ExecGateObservedStatus == "UNKNOWN",
		LaneTimedOut:     capture.TimedOut,
		LaneTruncated:    capture.StdoutTruncated || capture.StderrTruncated,
	})
	class := classFromVerdict(string(verdict))
	result := ExecuteCloseResult{
		FreezeCommit:  rc.FreezeCommit,
		SubjectCommit: rc.SubjectCommit,
	}
	return result, class, nil
}

// ExecuteCloseResult is the typed result.
type ExecuteCloseResult struct {
	FreezeCommit  string `json:"freeze_commit"`
	SubjectCommit string `json:"subject_commit"`
}

// exitCodeForClass projects a class onto the exact exit code.
func exitCodeForClass(class ExecuteFailureClass) int {
	switch class {
	case "":
		return exitPass
	case ExecuteRequest:
		return exitRequest
	case ExecuteVerification:
		return exitVerify
	case ExecuteObserver:
		return exitObserve
	}
	return exitObserve
}

// classFromVerdict projects the classifier verdict onto a class.
func classFromVerdict(verdict string) ExecuteFailureClass {
	switch verdict {
	case "PASS":
		return ExecuteFailureClass("")
	case "FAIL":
		return ExecuteVerification
	default:
		return ExecuteObserver
	}
}

// classifyRuntimeError uses the standard errors.As to map a
// RuntimeContextError into a failure class.
func classifyRuntimeError(err error) ExecuteFailureClass {
	if err == nil {
		return ExecuteFailureClass("")
	}
	var rce *closure.RuntimeContextError
	if errors.As(err, &rce) {
		switch rce.Kind {
		case "dirty_worktree", "freeze_equals_subject", "freeze_not_ancestor",
			"unsupported_object_format", "plan_path_invalid", "empty_field":
			return ExecuteRequest
		default:
			return ExecuteObserver
		}
	}
	return ExecuteObserver
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
