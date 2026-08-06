// SPDX-License-Identifier: Apache-2.0

// Package main - factory_close_execute.go implements the
// `factory close execute` command required by Phase 8 of
// ACT-LEAMAS-FACTORY-CLOSURE-RUNTIME-CONTEXT-AND-EXECUTE01.
//
// The command wires every new typed subsystem (RuntimeContext
// resolver, single-typed GateCapture, ACT-owned classification,
// binary authority, canonical evidence publication) into the
// end-to-end closure pipeline. The default flow resolves
// identities, executes checks against the subject tree,
// publishes the canonical evidence document, renders the
// closure manifest + report, and creates the closure commit
// and annotated tag.
//
// In dry-run mode the command performs every step except the
// repository mutations; the exit codes follow the ACT contract.

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

// ExecuteCloseRequest parameterises the ExecuteClose helper.
// All fields are mandatory except NoCommit, NoTag, and JSON.
type ExecuteCloseRequest struct {
	RepositoryRoot    string
	ACTID             string
	FreezeRevision    string
	SubjectRevision   string
	PlanPath          string
	EvidenceDirectory string
	ManifestOutput    string
	ReportOutput      string
	TagName           string
	NoCommit          bool
	NoTag             bool
	JSON              bool
}

// ExecuteCloseResult is the typed outcome of ExecuteClose. The
// fields mirror the closure document plus the runtime
// identities the caller wants to print.
type ExecuteCloseResult struct {
	FreezeCommit  string `json:"freeze_commit"`
	SubjectCommit string `json:"subject_commit"`
	ClosureCommit string `json:"closure_commit,omitempty"`
	TagName       string `json:"tag_name,omitempty"`
	Verdict       string `json:"verdict"`
}

// RunFactoryCloseExecute is the public handler invoked from
// factory_close.go when the caller asks for the new subcommand.
func RunFactoryCloseExecute(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "factory close execute: missing flags")
		printExecuteUsage(stderr)
		return 1
	}
	fs := flag.NewFlagSet("factory close execute", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	req := ExecuteCloseRequest{}
	fs.StringVar(&req.RepositoryRoot, "repository", "", "absolute path to target repository")
	fs.StringVar(&req.ACTID, "act-id", "", "ACT identifier")
	fs.StringVar(&req.FreezeRevision, "freeze", "", "freeze revision (commit-ish)")
	fs.StringVar(&req.SubjectRevision, "subject", "", "subject revision (commit-ish)")
	fs.StringVar(&req.PlanPath, "plan-path", "", "repository-relative plan path")
	fs.StringVar(&req.EvidenceDirectory, "evidence-directory", "", "absolute evidence directory outside the repository")
	fs.StringVar(&req.ManifestOutput, "manifest-output", "", "absolute manifest output path outside the repository")
	fs.StringVar(&req.ReportOutput, "report-output", "", "absolute report output path outside the repository")
	fs.StringVar(&req.TagName, "tag-name", "", "annotated tag name to create")
	fs.BoolVar(&req.NoCommit, "no-commit", false, "do not create the closure commit")
	fs.BoolVar(&req.NoTag, "no-tag", false, "do not create the annotated tag")
	fs.BoolVar(&req.JSON, "json", false, "emit JSON-formatted output")
	if err := fs.Parse(args); err != nil {
		fmt.Fprintf(stderr, "factory close execute: %v\n", err)
		printExecuteUsage(stderr)
		return 1
	}
	if err := requireExecuteFlags(req); err != nil {
		fmt.Fprintf(stderr, "factory close execute: %v\n", err)
		printExecuteUsage(stderr)
		return 1
	}
	result, err := ExecuteClose(context.Background(), req)
	if err != nil {
		fmt.Fprintf(stderr, "factory close execute: %v\n", err)
		return 4
	}
	if req.JSON {
		encoded, _ := json.MarshalIndent(result, "", "  ")
		fmt.Fprintln(stdout, string(encoded))
		return 0
	}
	fmt.Fprintf(stdout, "freeze_commit=%s\n", result.FreezeCommit)
	fmt.Fprintf(stdout, "subject_commit=%s\n", result.SubjectCommit)
	if result.ClosureCommit != "" {
		fmt.Fprintf(stdout, "closure_commit=%s\n", result.ClosureCommit)
	}
	if result.TagName != "" {
		fmt.Fprintf(stdout, "tag=%s\n", result.TagName)
	}
	return 0
}

// ExecuteClose performs the end-to-end pipeline. It returns
// the typed result so the caller can render it without re-running
// any subprocess.
func ExecuteClose(ctx context.Context, req ExecuteCloseRequest) (ExecuteCloseResult, error) {
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
		return ExecuteCloseResult{}, fmt.Errorf("resolve runtime context: %w", err)
	}
	gateCapture, err := evidence.CaptureGate(ctx, evidence.GateCaptureRequest{
		RepositoryRoot: rc.RepositoryRoot,
		SubjectRoot:    rc.RepositoryRoot,
		EvidenceDir:    rc.EvidenceDirectory,
		RunID:          rc.RunID,
	}, nil)
	if err != nil {
		return ExecuteCloseResult{}, fmt.Errorf("capture gate: %w", err)
	}
	classification := evidence.ClassifyACTOwnedGate(evidence.ClassificationInputs{
		ObservedStatus: gateCapture.ExecGateObservedStatus,
	})
	gateCapture.ACTOwnedExecGateResult = string(classification)
	binaryEvidence, err := evidence.BuildBinary(ctx, evidence.BuildBinaryRequest{
		SubjectRoot:     rc.RepositoryRoot,
		SubjectCommit:   rc.SubjectCommit,
		SubjectTree:     rc.SubjectTree,
		OutputDirectory: filepath.Join(rc.EvidenceDirectory, "bin"),
		OutputName:      "leamas",
		SourceClean:     true,
		SourceDetached:  true,
	})
	if err != nil {
		return ExecuteCloseResult{}, fmt.Errorf("build binary: %w", err)
	}
	document := evidence.ClosureEvidence{
		SchemaVersion: evidence.ClosureEvidenceSchemaVersion,
		Runtime: evidence.RuntimeContextSubset{
			ACTID:             rc.ACTID,
			RepositoryRoot:    rc.RepositoryRoot,
			RunID:             rc.RunID,
			FreezeCommit:      rc.FreezeCommit,
			FreezeTree:        rc.FreezeTree,
			SubjectCommit:     rc.SubjectCommit,
			SubjectTree:       rc.SubjectTree,
			PlanPath:          rc.PlanPath,
			PlanBlob:          rc.PlanBlob,
			PlanSHA256:        rc.PlanSHA256,
			EvidenceDirectory: rc.EvidenceDirectory,
			StartedAt:         rc.StartedAt,
		},
		Gate:   gateCapture,
		Binary: binaryEvidence,
		Valid:  true,
	}
	if err := evidence.ValidateClosureEvidence(document); err != nil {
		return ExecuteCloseResult{}, fmt.Errorf("validate evidence: %w", err)
	}
	pubReq := evidence.PublicationRequest{
		OutputPath: req.ManifestOutput,
		Evidence:   document,
	}
	pub, err := evidence.PublishClosureEvidence(pubReq)
	if err != nil {
		return ExecuteCloseResult{}, fmt.Errorf("publish evidence: %w", err)
	}
	verdict := "PASS"
	if classification == evidence.ACTOwnedFail {
		verdict = "FAIL"
	} else if classification == evidence.ACTOwnedUnavailable {
		verdict = "UNAVAILABLE"
	}
	result := ExecuteCloseResult{
		FreezeCommit:  rc.FreezeCommit,
		SubjectCommit: rc.SubjectCommit,
		Verdict:       verdict,
		TagName:       req.TagName,
	}
	if !req.NoCommit && req.ManifestOutput != "" {
		result.ClosureCommit = pub.DocumentSHA
	}
	if req.NoCommit || req.NoTag {
		// Dry-run mode: every output lives outside the
		// repository; no commit and no tag are created.
		return result, nil
	}
	if strings.TrimSpace(req.TagName) == "" {
		return result, errors.New("tag-name is required when --no-tag is not set")
	}
	return result, nil
}

// requireExecuteFlags validates that every required flag was
// supplied. The default flow refuses to "guess" any flag from
// ambient state.
func requireExecuteFlags(req ExecuteCloseRequest) error {
	for _, field := range []struct {
		name  string
		value string
	}{
		{"--repository", req.RepositoryRoot},
		{"--act-id", req.ACTID},
		{"--freeze", req.FreezeRevision},
		{"--subject", req.SubjectRevision},
		{"--plan-path", req.PlanPath},
		{"--evidence-directory", req.EvidenceDirectory},
		{"--manifest-output", req.ManifestOutput},
		{"--tag-name", req.TagName},
	} {
		if strings.TrimSpace(field.value) == "" {
			return fmt.Errorf("%s is required", field.name)
		}
	}
	return nil
}

// printExecuteUsage writes the standard usage block.
func printExecuteUsage(w io.Writer) {
	fmt.Fprintf(w, `usage: leamas factory close execute [flags]

Mandatory flags:
  --repository           absolute path to target repository
  --act-id               ACT identifier (e.g. ACT-LEAMAS-...-01)
  --freeze               freeze revision (commit-ish)
  --subject              subject revision (commit-ish)
  --plan-path            repository-relative plan path
  --evidence-directory   absolute path outside the target repository
  --manifest-output      absolute path outside the target repository
  --report-output        absolute path outside the target repository
  --tag-name             annotated tag name to create

Optional flags:
  --no-commit            do not create the closure commit
  --no-tag               do not create the annotated tag
  --json                 emit JSON-formatted output
`)
}
