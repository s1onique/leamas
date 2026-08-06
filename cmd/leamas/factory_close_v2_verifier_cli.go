// SPDX-License-Identifier: Apache-2.0

package main

// factory_close_v2_verifier_cli.go implements the public
// `leamas factory close verify-v2-authority` command for
// ACT-LEAMAS-FACTORY-CLOSURE-PROTOCOL-V2-VERIFIER-CLI-TAG-STATE01,
// with the read-only output authority, duplicate-flag
// rejection, atomic publication, and exact observer exit
// classification required by
// ACT-LEAMAS-FACTORY-CLOSURE-PROTOCOL-V2-VERIFIER-MAC-HANDOFF01-CORRECTION01.
//
// The CLI exposes the v2 closure verifier end-to-end:
//
//   --repository / --protocol-version / --plan-contract-version
//   --subject / --freeze / --closure
//   --plan-path / --manifest-path
//   --working-manifest-assertion (optional)
//   --expected-tag (optional)
//   --output (text or JSON target path, MUST live outside the
//             target repository and any of its linked worktrees)
//   --json (single JSON document on stdout)
//   --help (the dedicated help contract from ACT 4)
//
// The CLI never infers C from HEAD, M from convention, or P
// from the working tree. The CLI rejects --output paths that
// resolve inside the target repository BEFORE any Git
// observation, rejects duplicate flag occurrences BEFORE
// any Git observation, and publishes the verdict-derived
// artifact via an atomic temp+rename write so a partial
// write can never leave a half-formed file behind.
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
// Exit codes (Phase 3 of correction01):
//
//   0 = valid verifier result
//   2 = CLI usage failure
//   3 = authoritative verification rejection
//   4 = observer/infrastructure failure

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	"github.com/s1onique/leamas/internal/factory/closure"
)

// v2VerifierCLI exit codes pin a stable surface for
// downstream tooling.
//
//	0 -> verifier reported valid=true
//	2 -> usage error (unknown flag, missing required flag, duplicate flag)
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
// unknown flags, missing-required combinations, repeated
// flags, and --output paths that are not safely detached
// from the target repository before any Git observation
// so the CLI never observes a half-built request.
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
	FailureClass string                        `json:"failure_class,omitempty"`
	Verification closure.V2ClosureVerification `json:"verification"`
}

// v2VerifierUsage is the stable help text printed by the
// --help flag and on `factory close verify-v2-authority`
// without arguments.
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

--output (when set) MUST resolve outside the target repository
and every linked worktree. The CLI rejects inside-the-repo
--output paths BEFORE any Git observation, so a rejected
invocation never touches the object database.

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
                         when absent the summary is written to stdout.
                         The path MUST live outside the target
                         repository and every linked worktree.
  --json                 emit exactly one JSON document on stdout
  --capture-caller-state capture pre/post caller state (test surface)
  --help                 print this help and exit 0

Exit codes:
  0  verifier reported valid=true
  2  usage error (unknown flag, missing flag, duplicate flag, bad output)
  3  verifier failure (topology / manifest / tag / state mismatch)
  4  observer failure (git authority unavailable)
`

// v2VerifierScalarFlagNames is the closed set of scalar
// flags the duplicate-detection pass walks over. Each name
// is the canonical CLI flag without the leading "--".
// The parser rejects repeated occurrences of any name in
// the list (including repeats with the same value).
var v2VerifierScalarFlagNames = []string{
	"repository",
	"protocol-version",
	"plan-contract-version",
	"subject",
	"freeze",
	"closure",
	"plan-path",
	"manifest-path",
	"working-manifest-assertion",
	"expected-tag",
	"output",
}

// detectDuplicateV2VerifierFlags scans args for repeated
// occurrences of any scalar flag and returns a typed
// *V2VerifierError with code V2VerifierDuplicateCLIFlag
// naming the first duplicate. The check runs BEFORE
// flag.Parse so the duplicate surfaces before any Git
// observation.
func detectDuplicateV2VerifierFlags(args []string) error {
	seen := make(map[string]int, len(args))
	for _, a := range args {
		name, isFlag := stripV2VerifierFlagName(a)
		if !isFlag {
			continue
		}
		if !isTrackedV2VerifierFlag(name) {
			continue
		}
		seen[name]++
		if seen[name] > 1 {
			return closure.NewV2VerifierError(closure.V2VerifierDiagnostic{
				Code:         closure.V2VerifierDuplicateCLIFlag,
				Message:      fmt.Sprintf("duplicate flag: --%s", name),
				PropertyName: "flag",
			})
		}
	}
	return nil
}

// stripV2VerifierFlagName returns the canonical flag
// name and true when arg is a --flag or --flag=value
// token. The function tolerates "-flag" as a synonym for
// "--flag" to match Go's flag package conventions; a
// positional or non-flag argument returns ("", false).
func stripV2VerifierFlagName(arg string) (string, bool) {
	if arg == "" {
		return "", false
	}
	if !strings.HasPrefix(arg, "-") {
		return "", false
	}
	trimmed := strings.TrimLeft(arg, "-")
	if trimmed == "" {
		return "", false
	}
	if eq := strings.IndexByte(trimmed, '='); eq >= 0 {
		trimmed = trimmed[:eq]
	}
	return trimmed, true
}

// isTrackedV2VerifierFlag reports whether name is in
// the closed set of scalar flags the duplicate detector
// tracks. The detector is intentionally narrow: a stray
// unknown flag will be caught by flag.Parse below.
func isTrackedV2VerifierFlag(name string) bool {
	for _, n := range v2VerifierScalarFlagNames {
		if n == name {
			return true
		}
	}
	return false
}

// parseV2VerifierFlags parses the command arguments. The
// function returns a fully populated request or a typed
// usage error.
func parseV2VerifierFlags(name string, stderr io.Writer, args []string) (v2VerifierParsedInput, error) {
	if err := detectDuplicateV2VerifierFlags(args); err != nil {
		return v2VerifierParsedInput{}, err
	}

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
		"optional path for the structured text summary; MUST live outside --repository")
	fs.BoolVar(&in.JSONOutput, "json", false, "emit exactly one JSON document on stdout")
	fs.BoolVar(&in.CaptureCallerState, "capture-caller-state", false,
		"capture pre/post caller state (test surface)")
	fs.Bool("help", false, "print help and exit")

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
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

	if err := closure.ValidateDetachedVerifierOutputPath(in.Request.RepositoryRoot, in.OutputPath); err != nil {
		return v2VerifierParsedInput{}, err
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
// for `factory close verify-v2-authority`.
func runFactoryCloseVerifyV2Authority(args []string, stdout, stderr io.Writer) int {
	const command = "factory close verify-v2-authority"

	in, err := parseV2VerifierFlags(command, stderr, args)
	if err != nil {
		if errors.Is(err, flag.ErrHelp) {
			fmt.Fprint(stdout, v2VerifierUsage)
			return v2VerifierExitSuccess
		}
		if isV2VerifierOutputPathError(err) {
			fmt.Fprintf(stderr, "%s: %v\n", command, err)
			return v2VerifierExitObserverBroken
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

// isV2VerifierOutputPathError reports whether err
// originates from the output-path resolver.
func isV2VerifierOutputPathError(err error) bool {
	var vErr *closure.V2VerifierError
	if !errors.As(err, &vErr) {
		return false
	}
	for _, d := range vErr.Diags {
		if d.Code == closure.V2VerifierOutputPathNotDetached {
			return true
		}
	}
	return false
}

// writeV2VerifierText renders the verifier outcome in the
// stable text contract. On success the summary is published
// via the atomic writer. On failure the optional --output
// file is NEVER written.
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
		if err := closure.WriteFileAtomic(outputPath, []byte(summary+"\n"), 0o644); err != nil {
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
// envelope is stable regardless of success or failure.
func writeV2VerifierJSON(command string, stdout, stderr io.Writer, outputPath string, run closure.V2RunResult) int {
	envelope := v2VerifierJSONEnvelope{
		OK:           run.Verification.Valid,
		OutputPath:   outputPath,
		Verification: run.Verification,
	}
	if !run.Verification.Valid {
		envelope.FailureClass = classifyV2VerifierFailure(run.Verification.Diagnostics)
	}
	bytes, err := json.MarshalIndent(envelope, "", "  ")
	if err != nil {
		fmt.Fprintf(stderr, "%s: marshal JSON envelope: %v\n", command, err)
		return v2VerifierExitObserverBroken
	}
	stdoutBytes := append(bytes, '\n')
	if outputPath != "" {
		if err := closure.WriteFileAtomic(outputPath, stdoutBytes, 0o644); err != nil {
			fmt.Fprintf(stderr, "%s: --output %s write failed: %v\n", command, outputPath, err)
			return v2VerifierExitObserverBroken
		}
		stdout.Write(stdoutBytes)
	} else {
		stdout.Write(stdoutBytes)
	}

	if !run.Verification.Valid {
		return v2VerifierExitVerifier
	}
	return v2VerifierExitSuccess
}

// classifyV2VerifierFailure returns the canonical failure
// class token ("observer" or "verifier") for the supplied
// diagnostics. The function never returns an empty string.
func classifyV2VerifierFailure(diags closure.V2VerifierDiagnostics) string {
	for _, d := range diags {
		switch d.Code {
		case closure.V2VerifierObjectFormatUnavailable,
			closure.V2VerifierUnsupportedObjectFormat,
			closure.V2VerifierStateCaptureHeadFailed,
			closure.V2VerifierStateCaptureStatusFailed,
			closure.V2VerifierStateCaptureWorktreeFailed,
			closure.V2VerifierStateCaptureRefsFailed,
			closure.V2VerifierTopologyObservationFailed,
			closure.V2VerifierFrozenPlanReadFailed,
			closure.V2VerifierClosureManifestReadFailed,
			closure.V2VerifierRepositoryUnavailable,
			closure.V2VerifierClosureTagUnreadable,
			closure.V2VerifierOutputPublicationFailed,
			closure.V2VerifierObserverClass:
			return "observer"
		}
	}
	return "verifier"
}

// writeV2VerifierFailure handles the orchestrator-failure
// path. The function never emits an empty {ok:false}
// envelope: even when no diagnostics are available the
// failure_class is set to "observer".
func writeV2VerifierFailure(
	command string,
	stdout, stderr io.Writer,
	jsonOutput bool,
	outputPath string,
	_ closure.V2RunResult,
	err error,
) int {
	diags := closure.V2VerifierDiagnostics{}
	var vErr *closure.V2VerifierError
	if errors.As(err, &vErr) && vErr != nil {
		diags = vErr.Diags
	}
	verifier := closure.V2ClosureVerification{
		Valid:       false,
		Diagnostics: diags,
	}
	envelope := v2VerifierJSONEnvelope{
		OK:           false,
		OutputPath:   outputPath,
		FailureClass: classifyV2VerifierFailure(diags),
		Verification: verifier,
	}
	if envelope.FailureClass == "" {
		envelope.FailureClass = "observer"
	}
	if jsonOutput {
		bytes, mErr := json.MarshalIndent(envelope, "", "  ")
		if mErr != nil {
			fmt.Fprintf(stderr, "%s: marshal JSON envelope: %v\n", command, mErr)
			return v2VerifierExitObserverBroken
		}
		stdout.Write(append(bytes, '\n'))
		if envelope.FailureClass == "observer" {
			return v2VerifierExitObserverBroken
		}
		return v2VerifierExitVerifier
	}
	for _, d := range sortedV2DiagsForText(diags) {
		fmt.Fprintf(stderr, "%s: %s [%s]: %s\n", command, d.Code, d.PropertyName, d.Message)
	}
	if len(diags) == 0 {
		fmt.Fprintf(stderr, "%s: %v\n", command, err)
	}
	if envelope.FailureClass == "observer" {
		return v2VerifierExitObserverBroken
	}
	return v2VerifierExitVerifier
}

// sortedV2DiagsForText returns the diagnostics in a
// deterministic order for text rendering: by Code, then by
// PropertyName, then by Message.
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
