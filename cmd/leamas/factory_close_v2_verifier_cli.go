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
// rendered on stdout for --json and on disk through
// --output. The struct always includes the underlying
// verification result plus a stable diagnostics slice;
// success and failure share the same envelope so downstream
// JSON parsers need a single schema.
//
// The publication state (published, published but directory
// fsync failed, not published) is NOT included in the JSON
// envelope: the same bytes cannot both DESCRIBE and PRODUCE
// the publication atomically. The CLI instead surfaces the
// publication state through:
//   - the exit code (0 = published, 4 = published_but_*_failed
//     or not_published)
//   - a single stderr line in the format:
//     publication_state=<state>
//     prefixed with `leamas: ` so consumers can grep it.
//
// Result: stdout JSON and --output JSON are always byte-
// identical (apart from the documented trailing-newline rule).
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

	// Inventory worktrees and prepare the publication authority
	// BEFORE any verifier Git/object observation. The CLI never
	// publishes on failure paths, so failure here short-circuits
	// the orchestrator without committing to a verifier result.
	var prepared *closure.VerifierOutputAuthority
	if in.OutputPath != "" {
		inventory, invErr := closure.InventoryRepositoryWorktrees(
			context.Background(),
			in.Request.RepositoryRoot,
			nil,
		)
		if invErr != nil {
			fmt.Fprintf(stderr, "%s: %v\n", command, invErr)
			return v2VerifierExitObserverBroken
		}
		auth, prepErr := closure.PrepareVerifierOutput(
			in.Request.RepositoryRoot,
			in.OutputPath,
			canonicalWorktreesFrom(inventory),
		)
		if prepErr != nil {
			fmt.Fprintf(stderr, "%s: %v\n", command, prepErr)
			return v2VerifierExitObserverBroken
		}
		prepared = auth
		defer prepared.Close()
	}

	authority, err := closure.NewV2ClosureGitAuthority(in.Request.RepositoryRoot)
	if err != nil {
		return writeV2VerifierFailure(command, stdout, stderr, in.JSONOutput, prepared, in.OutputPath, closure.V2RunResult{}, err)
	}

	orchestrator := closure.NewV2VerifierOrchestrator()
	runReq := closure.V2RunRequest{
		Request:            in.Request,
		CaptureCallerState: in.CaptureCallerState,
	}
	runResult := orchestrator.Run(context.Background(), authority, runReq)

	if in.JSONOutput {
		return writeV2VerifierJSON(command, stdout, stderr, prepared, in.OutputPath, runResult)
	}
	return writeV2VerifierText(command, stdout, stderr, prepared, in.OutputPath, runResult)
}

// writeV2VerifierText renders the verifier outcome in the
// stable text contract. On success the summary is published
// via the prepared authority (or, when no --output was
// supplied, written to stdout). On failure the optional
// --output file is NEVER written.
//
// The CLI's bytes-on-disk policy is: a single trailing newline
// is appended to whichever path publishes the bytes (file or
// stdout). When --output is set, the file path is bytes-identical
// to what would have been emitted to stdout (summary + "\n").
func writeV2VerifierText(command string, stdout, stderr io.Writer, prepared *closure.VerifierOutputAuthority, outputPath string, run closure.V2RunResult) int {
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
	bytes := []byte(summary + "\n")
	if prepared != nil {
		res := prepared.Publish(bytes)
		if res.State == closure.PublicationNotPublished {
			fmt.Fprintf(stderr, "%s: --output %s write failed: %v\n", command, outputPath, res.Err)
			return v2VerifierExitObserverBroken
		}
		stdout.Write(bytes)
		// Stable CLI contract: published_but_directory_sync_failed
		// is an observer-class outcome and maps to exit 4. The
		// text-mode path does not have a JSON envelope so the
		// surface communicates durability state via stderr alone.
		if res.State == closure.PublicationPublishedButDirectorySyncFailed ||
			res.State == closure.PublicationPublishedButPostPublishObservationFailed {
			fmt.Fprintf(stderr, "%s: --output %s publication observation failed: %v\n",
				command, outputPath, res.Err)
			return v2VerifierExitObserverBroken
		}
		return v2VerifierExitSuccess
	}
	stdout.Write(bytes)
	return v2VerifierExitSuccess
}

// writeV2VerifierJSON renders the verifier outcome as a
// single deterministic JSON document. The output envelope
// is stable regardless of success or failure. The CLI
// publishes the JSON EXACTLY ONCE through the prepared
// authority (or writes it to stdout if --output is unset)
// so the published file and the stdout document are
// byte-identical.
//
// Documented newline rule: the JSON envelope always ends
// with a single trailing '\n'. The CLI does NOT emit a
// trailing newline beyond that single character.
//
// Publication durability is signaled via:
//   - exit code 0 (published) vs 4 (published_but_*_failed)
//   - a single stderr line in the format:
//     <command>: publication_state=<state>
//
// The JSON envelope itself intentionally does NOT include
// the publication state: the same bytes cannot both DESCRIBE
// and PRODUCE the publication atomically. Consumers parse the
// exit code for the durable verdict; the stderr line is a
// stable, parseable human-readable trace.
func writeV2VerifierJSON(command string, stdout, stderr io.Writer, prepared *closure.VerifierOutputAuthority, outputPath string, run closure.V2RunResult) int {
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
	if prepared != nil {
		res := prepared.Publish(stdoutBytes)
		if res.State == closure.PublicationNotPublished {
			fmt.Fprintf(stderr, "%s: --output %s write failed: %v\n", command, outputPath, res.Err)
			// Do NOT emit the JSON envelope to stdout: the
			// publication was supposed to install it on disk and
			// failed. Emit a typed failure instead.
			fmt.Fprintf(stderr, "%s: publication_state=not_published\n", command)
			return v2VerifierExitObserverBroken
		}
		stdout.Write(stdoutBytes)
		if res.State == closure.PublicationPublishedButDirectorySyncFailed ||
			res.State == closure.PublicationPublishedButPostPublishObservationFailed {
			fmt.Fprintf(stderr, "%s: --output %s publication observation failed: %v\n",
				command, outputPath, res.Err)
			fmt.Fprintf(stderr, "%s: publication_state=%s\n", command, res.State)
			return v2VerifierExitObserverBroken
		}
		fmt.Fprintf(stderr, "%s: publication_state=published\n", command)
		if !run.Verification.Valid {
			return v2VerifierExitVerifier
		}
		return v2VerifierExitSuccess
	}
	stdout.Write(stdoutBytes)
	if !run.Verification.Valid {
		return v2VerifierExitVerifier
	}
	return v2VerifierExitSuccess
}
