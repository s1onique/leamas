// SPDX-License-Identifier: Apache-2.0

package main

// factory_close_v2_verifier_cli_writers.go isolates the JSON
// envelope and the failure-classification helpers used by
// writeV2VerifierJSON/writeV2VerifierText and the failure
// writer from the command dispatcher in
// factory_close_v2_verifier_cli.go. Splitting along the
// writer boundary keeps each file under the LLM-friendliness
// 400-line threshold.

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/s1onique/leamas/internal/factory/closure"
)

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
//
// The prepared authority parameter accepts the publication
// handle acquired by runFactoryCloseVerifyV2Authority before
// the orchestrator ran; on the failure path the CLI MUST NOT
// publish to --output, so the authority is recorded in
// stderr for diagnostic clarity but never written. The
// caller remains responsible for closing the authority.
func writeV2VerifierFailure(
	command string,
	stdout, stderr io.Writer,
	jsonOutput bool,
	prepared *closure.VerifierOutputAuthority,
	outputPath string,
	_ closure.V2RunResult,
	err error,
) int {
	diags := closure.V2VerifierDiagnostics{}
	var vErr *closure.V2VerifierError
	if isErrorsAs(err, &vErr) && vErr != nil {
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
	if prepared != nil && outputPath != "" {
		fmt.Fprintf(stderr, "%s: --output %s suppressed on orchestrator failure\n", command, outputPath)
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
