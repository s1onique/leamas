// SPDX-License-Identifier: Apache-2.0

// factory_close_v2_runner.go wires the production Closure
// Protocol v2 runner from internal/factory/closure into a
// public CLI command:
//
//	leamas factory close run-v2-authority \
//	    --protocol-version 2 \
//	    --plan-contract-version 1 \
//	    --repository <repo> \
//	    --subject <S> \
//	    --freeze <F> \
//	    --plan-path <P> \
//	    --evidence-directory <dir> \
//	    --manifest-output <file> \
//	    [--working-plan-assertion <file>] \
//	    [--json]
//
// ACT-LEAMAS-FACTORY-CLOSURE-PROTOCOL-V2-RUNNER-MANIFEST-CLI-AUTHORITY01
// enforces:
//
//   - mandatory binary identity: the runner refuses to publish
//     a manifest when the running binary's identity cannot be
//     resolved or fails runtime validation (path absolute,
//     executable, SHA-256 matches file contents, VCS revision
//     is a 40-character lowercase Git OID, version is nonempty).
//   - detached evidence and manifest locations are canonicalised
//     and checked against the repository worktree and the Git
//     common directory; symlinks are resolved.
//   - on failure, no manifest is written and a stable JSON
//     envelope (or text summary) is emitted. The envelope
//     shape, exit codes, and stdout/stderr roles are pinned
//     so downstream tooling never has to parse message text.

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"

	"github.com/s1onique/leamas/internal/factory/closure"
)

// v2RunnerCLIResult is the wire shape of the CLI envelope
// under --json. The struct is intentionally flat: every field
// is either a typed machine identifier or a string. The
// Manifest field is omitted on failure to keep the failure
// envelope small.
type v2RunnerCLIResult struct {
	OK          bool                  `json:"ok"`
	Verdict     string                `json:"verdict,omitempty"`
	Manifest    *closure.V2Manifest   `json:"manifest,omitempty"`
	Diagnostics closure.V2Diagnostics `json:"diagnostics,omitempty"`
}

// runFactoryCloseRunV2Authority is the production CLI entry
// point for `leamas factory close run-v2-authority`. The
// function is total: every error path returns a non-zero exit
// code and a typed diagnostic envelope.
func runFactoryCloseRunV2Authority(args []string, stdout, stderr io.Writer) int {
	parsed, err := parseV2AuthorityFlags(args)
	if err != nil {
		writeV2AuthorityTextError(stderr, "factory close run-v2-authority", err)
		return v2ExitUsage
	}
	identity, err := captureRunningBinaryIdentity()
	if err != nil {
		writeV2AuthorityFailure(stdout, stderr, parsed.JSONOutput, v2CLIIdentityFailure{
			Code:         "binary_identity_invalid",
			Message:      fmt.Sprintf("capture running binary identity: %v", err),
			PropertyName: "leamas_binary_identity",
		})
		return v2ExitManifest
	}
	if err := closure.ValidateV2BinaryIdentity(identity); err != nil {
		writeV2AuthorityFailure(stdout, stderr, parsed.JSONOutput, v2CLIIdentityFailure{
			Code:         "binary_identity_invalid",
			Message:      err.Error(),
			PropertyName: "leamas_binary_identity",
		})
		return v2ExitManifest
	}
	manifest, runErr := closure.RunClosureProtocolV2WithBinary(
		context.Background(), parsed.Request, identity,
	)
	if runErr != nil {
		writeV2AuthorityFailure(stdout, stderr, parsed.JSONOutput, v2CLIIdentityFailure{
			Code:         "v2_failure",
			Message:      runErr.Error(),
			PropertyName: "v2_runner",
			Diagnostics:  diagnosticsFromError(runErr),
		})
		return v2ExitFailure(runErr)
	}
	if parsed.JSONOutput {
		data, _ := json.MarshalIndent(v2RunnerCLIResult{
			OK:       true,
			Verdict:  closureVerdictFromManifest(manifest),
			Manifest: &manifest,
		}, "", "  ")
		fmt.Fprintln(stdout, string(data))
	} else {
		fmt.Fprintln(stdout, "OK")
		fmt.Fprintf(stderr,
			"Closure Protocol v2: S=%s F=%s plan_blob=%s plan_sha256=%s execution_tree=%s binary=%s\n",
			trunc8(manifest.SubjectCommit), trunc8(manifest.FreezeCommit),
			trunc8(manifest.PlanBlob), trunc8(manifest.PlanSHA256),
			trunc8(manifest.ExecutionTree), trunc8(manifest.LeamasBinaryIdentity.Path))
	}
	return v2ExitSuccess
}

func diagnosticsFromError(err error) closure.V2Diagnostics {
	if v2err, ok := err.(*closure.V2Error); ok {
		return v2err.Diags
	}
	return closure.V2Diagnostics{{
		Code:         "v2_failure",
		Message:      err.Error(),
		PropertyName: "v2_runner",
	}}
}

func closureVerdictFromManifest(manifest closure.V2Manifest) string {
	for _, result := range manifest.CheckResults {
		if result.Outcome == closure.CheckStatusFail {
			return closure.VerdictFail
		}
	}
	return closure.VerdictPass
}
