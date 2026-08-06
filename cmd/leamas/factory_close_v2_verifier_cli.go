// SPDX-License-Identifier: Apache-2.0

package main

// factory_close_v2_verifier_cli.go implements the public
// `leamas factory close verify-v2-authority` command for
// ACT-LEAMAS-FACTORY-CLOSURE-PROTOCOL-V2-VERIFIER-CLI-TAG-STATE01.
//
// The CLI exposes the v2 closure verifier end-to-end:
//
//   --repository / --protocol-version / --plan-contract-version
//   --subject / --freeze / --closure
//   --plan-path / --manifest-path
//   --working-manifest-assertion (optional)
//   --expected-tag (optional)
//   --output (text or JSON target path)
//   --json (single JSON document on stdout)
//   --help (the dedicated help contract from ACT 4)
//
// The CLI never infers C from HEAD, M from convention, or P
// from the working tree. The verifier itself is read-only;
// the CLI captures caller state before/after when the
// --capture-caller-state test surface flag is supplied.
//
// Text output:
//
//   exit 0  -> "OK subject=… freeze=… closure=… manifest_sha256=…"
//   exit 1  -> typed failure rendering with diagnostics
//
// JSON output:
//
//   stdout  = single deterministic JSON envelope
//   stderr  = empty for success, diagnostics for failure
//
// Exit codes:
//
//   0 = success / invalid_pass (verifier reported valid=true)
//   2 = usage error (missing flag, unknown flag)
//   3 = verifier failure (topology, manifest, tag mismatch)
//   4 = observer failure (git authority broken)

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"sort"

	"github.com/s1onique/leamas/internal/factory/closure"
)

// v2VerifierCLI exit codes pin a stable surface for
// downstream tooling.
//
//	0 -> verifier reported valid=true
//	2 -> usage error (unknown flag, missing required flag)
//	3 -> verifier failure (topology / manifest / tag / state)
//	4 -> observer failure (git authority unavailable)
const (
	v2VerifierExitSuccess        = 0
	v2VerifierExitUsage          = 2
	v2VerifierExitVerifier       = 3
	v2VerifierExitObserverBroken = 4
)

// v2VerifierParsedInput captures the parsed CLI arguments
// before they reach the orchestrator. The parser rejects
// unknown flags and missing-required combinations before
// any Git observation so the CLI never observes a half-built
// request.
type v2VerifierParsedInput struct {
	Request             closure.V2ClosureVerifyRequest
	ExpectedTag         string
	WorkingManifestPath string
	OutputPath          string
	JSONOutput          bool
	CaptureCallerState  bool
}

// v2VerifierJSONEnvelope is the deterministic JSON document
// rendered on stdout for --json. The struct always includes
// the underlying verification result plus a stable
// diagnostics slice; success and failure share the same
// envelope so downstream JSON parsers need a single schema.
type v2VerifierJSONEnvelope struct {
	OK           bool                          `json:"ok"`
	OutputPath   string                        `json:"output_path,omitempty"`
	Verification closure.V2ClosureVerification `json:"verification"`
}

// v2VerifierUsage is the stable help text printed by the
// --help flag and on `factory close verify-v2-authority`
// without arguments. The text is intentionally long because
// ACT 4's Phase 3 help-contract requirement specifies that
// the help MUST explain:
//
//	S = execution subject
//	F = frozen-plan authority
//	C = closure commit
//	P loaded from F
//	M loaded from C
//	HEAD is not authority
//	C need not appear inside M
const v2VerifierUsage = `Usage: leamas factory close verify-v2-authority [flags]

Verify the immutable authority of a Closure Protocol v2 closure
transaction. The verifier is read-only and never infers C from HEAD,
M from convention, or P from the working tree.

Authority OIDs required:
  --subject  S  the execution subject commit (resolved via F^{commit})
  --freeze   F  the frozen-plan authority commit (parent of C)
  --closure  C  the externally supplied closure commit (NOT HEAD)
  --plan-path     P loaded from F:F:P, never from HEAD or working tree
  --manifest-path M loaded from C:C:M, never from HEAD or working tree

Verifier never requires:
  - manifest.closure_commit to equal C  (self-reference doctrine)
  - C to appear anywhere in M
  - HEAD to point at C
  - the working tree to be clean

Options:
  --repository           path to a Git repository root (required)
  --protocol-version 2   closure protocol version (only 2 is accepted)
  --plan-contract-version 1
                         plan contract version (only 1 is accepted)
  --subject              S commit OID (required, never inferred)
  --freeze               F commit OID (required)
  --closure              C commit OID (required, never inferred from HEAD)
  --plan-path            P repository-relative path (required)
  --manifest-path        M repository-relative path (required)
  --working-manifest-assertion <file>
                         optional path to bytes whose SHA-256 must
                         match C:M. The assertion is non-authoritative;
                         a mismatch NEVER replaces the C:M binding.
  --expected-tag <name>  optional annotated-tag name. When set the
                         tag must exist, be annotated (not lightweight),
                         and dereference to C exactly.
  --output <path>        optional path for the structured text summary;
                         when absent the summary is written to stdout
  --json                 emit exactly one JSON document on stdout
  --capture-caller-state capture pre/post caller state (test surface)
  --help                 print this help and exit 0

Exit codes:
  0  verifier reported valid=true
  2  usage error (unknown flag, missing flag)
  3  verifier failure (topology / manifest / tag / state mismatch)
  4  observer failure (git authority unavailable)

Examples:
  leamas factory close verify-v2-authority \
    --repository /path/to/repo \
    --subject 56fd526e1923f2546fa0aeb53a0dc6e7501e5061 \
    --freeze  01822bf5c8b99e5a4b89a6761a713ec3603754b0 \
    --closure <C> \
    --plan-path docs/closure-plans/ACT-…json \
    --manifest-path docs/closure-manifests/ACT-…json \
    --expected-tag act/your-tag-name \
    --json
`

// parseV2VerifierFlags parses the command arguments. The
// function returns a fully populated request or a typed
// usage error. The parser rejects repeated flags, unknown
// flags, and missing-required flags before any orchestrator
// state is touched.
func parseV2VerifierFlags(name string, stderr io.Writer, args []string) (v2VerifierParsedInput, error) {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(stderr)

	var in v2VerifierParsedInput
	fs.StringVar(&in.Request.RepositoryRoot, "repository", "", "Git repository root (absolute path)")
	fs.StringVar((*string)(&in.Request.ClosureProtocolVersion), "protocol-version", string(closure.ClosureProtocolV2),
		"closure protocol version (only 2 is accepted)")
	fs.IntVar((*int)(&in.Request.PlanContractVersion), "plan-contract-version", int(closure.PlanContractV1),
		"plan contract version (only 1 is accepted)")
	fs.StringVar(&in.Request.SubjectCommit, "subject", "", "subject S commit OID")
	fs.StringVar(&in.Request.FreezeCommit, "freeze", "", "freeze F commit OID")
	fs.StringVar(&in.Request.ClosureCommit, "closure", "", "closure C commit OID")
	fs.StringVar(&in.Request.PlanPath, "plan-path", "", "repository-relative plan path P")
	fs.StringVar(&in.Request.ManifestPath, "manifest-path", "", "repository-relative manifest path M")
	fs.StringVar(&in.WorkingManifestPath, "working-manifest-assertion", "",
		"optional path whose bytes must match C:M (non-authoritative)")
	fs.StringVar(&in.ExpectedTag, "expected-tag", "",
		"optional annotated-tag name that must target C")
	fs.StringVar(&in.OutputPath, "output", "",
		"optional path for the structured text summary")
	fs.BoolVar(&in.JSONOutput, "json", false, "emit exactly one JSON document on stdout")
	fs.BoolVar(&in.CaptureCallerState, "capture-caller-state", false,
		"capture pre/post caller state (test surface)")
	fs.Bool("help", false, "print help and exit")

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			// Surfaces as success exit 0 below.
			return v2VerifierParsedInput{}, flag.ErrHelp
		}
		return v2VerifierParsedInput{}, fmt.Errorf("%s: %w", name, err)
	}
	if fs.NArg() != 0 {
		return v2VerifierParsedInput{}, fmt.Errorf("%s: unexpected arguments: %v", name, fs.Args())
	}

	helpFlag := fs.Lookup("help")
	if helpFlag != nil && helpFlag.Value.String() == "true" {
		return v2VerifierParsedInput{}, flag.ErrHelp
	}

	if string(in.Request.ClosureProtocolVersion) != string(closure.ClosureProtocolV2) {
		return v2VerifierParsedInput{}, fmt.Errorf("--protocol-version must be %q, got %q",
			closure.ClosureProtocolV2, in.Request.ClosureProtocolVersion)
	}
	if int(in.Request.PlanContractVersion) != int(closure.PlanContractV1) {
		return v2VerifierParsedInput{}, fmt.Errorf("--plan-contract-version must be %d, got %d",
			int(closure.PlanContractV1), int(in.Request.PlanContractVersion))
	}

	required := []struct {
		name, value string
	}{
		{"--repository", in.Request.RepositoryRoot},
		{"--subject", in.Request.SubjectCommit},
		{"--freeze", in.Request.FreezeCommit},
		{"--closure", in.Request.ClosureCommit},
		{"--plan-path", in.Request.PlanPath},
		{"--manifest-path", in.Request.ManifestPath},
	}
	for _, field := range required {
		if field.value == "" {
			return v2VerifierParsedInput{}, fmt.Errorf("%s is required", field.name)
		}
	}

	if in.WorkingManifestPath != "" {
		bytes, err := os.ReadFile(in.WorkingManifestPath)
		if err != nil {
			return v2VerifierParsedInput{}, fmt.Errorf("--working-manifest-assertion: %w", err)
		}
		in.Request.OptionalManifestAssertion = bytes
	}
	in.Request.ExpectedTagName = in.ExpectedTag
	return in, nil
}

// runFactoryCloseVerifyV2Authority is the CLI entry point
// for `factory close verify-v2-authority`. The function
// returns the process exit code so the dispatcher in
// factory_close.go can route the value directly to
// os.Exit; the function never writes to process CWD and
// never mutates the target repository.
func runFactoryCloseVerifyV2Authority(args []string, stdout, stderr io.Writer) int {
	const command = "factory close verify-v2-authority"

	in, err := parseV2VerifierFlags(command, stderr, args)
	if err != nil {
		if errors.Is(err, flag.ErrHelp) {
			fmt.Fprint(stdout, v2VerifierUsage)
			return v2VerifierExitSuccess
		}
		fmt.Fprintf(stderr, "%s: %v\n", command, err)
		return v2VerifierExitUsage
	}

	authority, err := closure.NewV2ClosureGitAuthority(in.Request.RepositoryRoot)
	if err != nil {
		return writeV2VerifierFailure(command, stdout, stderr, in.JSONOutput, in.OutputPath, closure.V2RunResult{}, err)
	}

	orchestrator := closure.NewV2VerifierOrchestrator()
	runReq := closure.V2RunRequest{
		Request:            in.Request,
		CaptureCallerState: in.CaptureCallerState,
	}
	runResult := orchestrator.Run(context.Background(), authority, runReq)

	if in.JSONOutput {
		return writeV2VerifierJSON(command, stdout, stderr, in.OutputPath, runResult)
	}
	return writeV2VerifierText(command, stdout, stderr, in.OutputPath, runResult)
}

// writeV2VerifierText renders the verifier outcome in the
// stable text contract: success prints one summary line per
// the ACT 4 contract; failure prints the typed diagnostics
// in deterministic order. Either way the same single error
// or single summary reaches the destination stream.
func writeV2VerifierText(command string, stdout, stderr io.Writer, outputPath string, run closure.V2RunResult) int {
	if !run.Verification.Valid {
		diags := run.Verification.Diagnostics
		header := fmt.Sprintf("%s: verifier rejected closure authority", command)
		if len(diags) == 0 {
			fmt.Fprintf(stderr, "%s: no verdict diagnostics, valid=false\n", command)
		} else {
			fmt.Fprintf(stderr, "%s\n", header)
			for _, d := range sortedV2DiagsForText(diags) {
				fmt.Fprintf(stderr, "  %s [%s]: %s\n", d.Code, d.PropertyName, d.Message)
			}
		}
		if outputPath != "" {
			// On failure the verifier NEVER writes the
			// optional --output file: a clean summary
			// is the only success signal there.
			fmt.Fprintf(stderr, "%s: --output %s suppressed on failure\n", command, outputPath)
		}
		return v2VerifierExitVerifier
	}

	summary := fmt.Sprintf("%s subject=%s freeze=%s closure=%s manifest_sha256=%s plan_sha256=%s valid=true",
		command,
		run.Verification.SubjectCommit,
		run.Verification.FreezeCommit,
		run.Verification.ClosureCommit,
		run.Verification.ManifestSHA256,
		run.Verification.PlanSHA256,
	)
	if outputPath != "" {
		if err := os.WriteFile(outputPath, []byte(summary+"\n"), 0o644); err != nil {
			fmt.Fprintf(stderr, "%s: --output %s write failed: %v\n", command, outputPath, err)
			return v2VerifierExitVerifier
		}
	} else {
		fmt.Fprintln(stdout, summary)
	}
	return v2VerifierExitSuccess
}

// writeV2VerifierJSON renders the verifier outcome as a
// single deterministic JSON document on stdout. The output
// envelope is stable regardless of success or failure so
// downstream parsers can rely on a single schema.
func writeV2VerifierJSON(command string, stdout, stderr io.Writer, outputPath string, run closure.V2RunResult) int {
	envelope := v2VerifierJSONEnvelope{
		OK:           run.Verification.Valid,
		Verification: run.Verification,
	}
	bytes, err := json.MarshalIndent(envelope, "", "  ")
	if err != nil {
		fmt.Fprintf(stderr, "%s: marshal JSON envelope: %v\n", command, err)
		return v2VerifierExitObserverBroken
	}
	if outputPath != "" {
		if err := os.WriteFile(outputPath, append(bytes, '\n'), 0o644); err != nil {
			fmt.Fprintf(stderr, "%s: --output %s write failed: %v\n", command, outputPath, err)
			return v2VerifierExitObserverBroken
		}
		// When --output is set the JSON also goes to
		// stdout so the CWD-detached witness bundle
		// can still capture it.
		fmt.Fprintln(stdout, string(bytes))
	} else {
		fmt.Fprintln(stdout, string(bytes))
	}

	if !run.Verification.Valid {
		return v2VerifierExitVerifier
	}
	return v2VerifierExitSuccess
}

// writeV2VerifierFailure handles the orchestrator-failure
// path: a "git authority unavailable" error is treated as
// an observer failure (exit 4); any other failure path is
// routed through the standard failure writers.
func writeV2VerifierFailure(
	command string,
	stdout, stderr io.Writer,
	jsonOutput bool,
	outputPath string,
	_ closure.V2RunResult,
	err error,
) int {
	if vErr, ok := err.(*closure.V2VerifierError); ok && vErr != nil {
		for _, d := range vErr.Diags {
			if d.Code == closure.V2VerifierRepositoryUnavailable {
				if jsonOutput {
					envelope := v2VerifierJSONEnvelope{OK: false, Verification: closure.V2ClosureVerification{
						Diagnostics: closure.V2VerifierDiagnostics{d},
					}}
					bytes, _ := json.MarshalIndent(envelope, "", "  ")
					fmt.Fprintln(stdout, string(bytes))
					return v2VerifierExitObserverBroken
				}
				fmt.Fprintf(stderr, "%s: %s: %s\n", command, d.Code, d.Message)
				return v2VerifierExitObserverBroken
			}
		}
	}
	if jsonOutput {
		envelope := v2VerifierJSONEnvelope{OK: false}
		bytes, _ := json.MarshalIndent(envelope, "", "  ")
		fmt.Fprintln(stdout, string(bytes))
		return v2VerifierExitVerifier
	}
	fmt.Fprintf(stderr, "%s: %v\n", command, err)
	return v2VerifierExitVerifier
}

// sortedV2DiagsForText returns the diagnostics in a
// deterministic order for text rendering: by Code, then by
// PropertyName, then by Message. The function ensures the
// text failure rendering is reproducible across runs.
func sortedV2DiagsForText(in closure.V2VerifierDiagnostics) closure.V2VerifierDiagnostics {
	out := make(closure.V2VerifierDiagnostics, len(in))
	copy(out, in)
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Code != out[j].Code {
			return string(out[i].Code) < string(out[j].Code)
		}
		if out[i].PropertyName != out[j].PropertyName {
			return out[i].PropertyName < out[j].PropertyName
		}
		return out[i].Message < out[j].Message
	})
	return out
}
