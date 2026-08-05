// SPDX-License-Identifier: Apache-2.0

package main

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
//	    [--working-plan-assertion <file>]
//
// ACT-LEAMAS-FACTORY-CLOSURE-PROTOCOL-V2-RUNNER-AUTHORITY01-
// CORRECTION01 enforces:
//
//   - --protocol-version must be 2; protocol 1 is rejected
//     with unsupported_closure_protocol_version before any
//     topology resolution.
//   - detached evidence and manifest locations are
//     canonicalised and checked against the repository worktree
//     and the Git common directory; symlinks are resolved.
//   - the produced manifest carries the binary identity of
//     the exact leamas binary invoked by the CLI.
//   - on failure, no manifest is written and a stable JSON
//     diagnostic document is emitted on stdout when --json is
//     set; human-readable output goes to stderr only.

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/s1onique/leamas/internal/factory/closure"
)

func runFactoryCloseRunV2Authority(args []string, stdout, stderr io.Writer) int {
	fs := newCloseFlagSet("factory close run-v2-authority", stderr)
	var req closure.V2Request
	var protocolVersionRaw string
	var workingPlanAssertion string
	var jsonOutput bool
	fs.StringVar(&protocolVersionRaw, "protocol-version", "2",
		"closure protocol version (must be 2)")
	fs.IntVar(&req.PlanContractVersion, "plan-contract-version", 1,
		"plan contract version")
	fs.StringVar(&req.RepositoryRoot, "repository", "", "repository root")
	fs.StringVar(&req.SubjectCommit, "subject", "", "subject commit")
	fs.StringVar(&req.FreezeCommit, "freeze", "", "freeze commit")
	fs.StringVar(&req.PlanPath, "plan-path", "", "plan path relative to repository")
	fs.StringVar(&req.EvidenceDirectory, "evidence-directory", "",
		"absolute detached evidence directory")
	fs.StringVar(&req.ManifestOutput, "manifest-output", "",
		"absolute detached manifest output")
	fs.StringVar(&workingPlanAssertion, "working-plan-assertion", "",
		"optional absolute path whose bytes must match the frozen plan")
	fs.BoolVar(&jsonOutput, "json", false, "emit structured JSON output")
	if err := parseCloseFlags(fs, args); err != nil {
		return reportCloseFlagError(stderr, "factory close run-v2-authority", err,
			"all required flags must be supplied")
	}
	if protocolVersionRaw == "" {
		protocolVersionRaw = string(closure.ClosureProtocolV2)
	}
	req.ClosureProtocolVersion = closure.ClosureProtocolVersion(protocolVersionRaw)
	req.OptionalWorkingPlanAssertion = workingPlanAssertion
	// Capture the exact binary identity of the running leamas
	// process so the manifest records the file that actually
	// produced it.
	binaryIdentity, binaryErr := captureRunningBinaryIdentity()
	if binaryErr != nil {
		return reportCloseError(stderr, "factory close run-v2-authority", binaryErr)
	}
	manifest, runErr := closure.RunClosureProtocolV2WithBinary(
		context.Background(), req, binaryIdentity,
	)
	if runErr != nil {
		v2err, ok := runErr.(*closure.V2Error)
		if ok {
			if jsonOutput {
				data, _ := json.MarshalIndent(struct {
					Code     string                `json:"code"`
					Diags    closure.V2Diagnostics `json:"diagnostics"`
					Manifest string                `json:"manifest_path,omitempty"`
				}{
					Code:     "v2_failure",
					Diags:    v2err.Diags,
					Manifest: req.ManifestOutput,
				}, "", "  ")
				fmt.Fprintln(stdout, string(data))
			} else {
				for _, d := range v2err.Diags {
					fmt.Fprintf(stderr, "%s: %s (%s)\n", d.Code, d.Message, d.PropertyName)
				}
			}
		}
		return reportCloseError(stderr, "factory close run-v2-authority", runErr)
	}
	if jsonOutput {
		data, _ := json.MarshalIndent(manifest, "", "  ")
		fmt.Fprintln(stdout, string(data))
	} else {
		fmt.Fprintln(stdout, "OK")
		fmt.Fprintf(stderr,
			"Closure Protocol v2: S=%s F=%s plan_blob=%s plan_sha256=%s execution_tree=%s binary=%s\n",
			trunc8(manifest.SubjectCommit), trunc8(manifest.FreezeCommit),
			trunc8(manifest.PlanBlob), trunc8(manifest.PlanSHA256),
			trunc8(manifest.ExecutionTree), trunc8(manifest.LeamasBinaryIdentity.Path))
	}
	return closeSuccessCode()
}

// captureRunningBinaryIdentity reads the absolute path of the
// current leamas binary and computes its SHA-256. It returns
// the exact file the OS is executing, which is the identity the
// manifest records.
func captureRunningBinaryIdentity() (closure.V2BinaryIdentity, error) {
	exe, err := os.Executable()
	if err != nil {
		return closure.V2BinaryIdentity{}, fmt.Errorf("locate leamas binary: %w", err)
	}
	resolved, err := filepath.EvalSymlinks(exe)
	if err != nil {
		return closure.V2BinaryIdentity{}, fmt.Errorf("resolve leamas binary symlinks: %w", err)
	}
	data, err := os.ReadFile(resolved)
	if err != nil {
		return closure.V2BinaryIdentity{}, fmt.Errorf("read leamas binary: %w", err)
	}
	sum := sha256.Sum256(data)
	return closure.V2BinaryIdentity{
		Path:          resolved,
		SHA256:        hex.EncodeToString(sum[:]),
		VCSRevision:   closure.RunningLeamasVCSRevision(),
		VCSModified:   closure.RunningLeamasVCSModified(),
		LeamasVersion: closure.RunningLeamasVersion(),
	}, nil
}

// ensure imports used even if the surrounding wiring shifts.
var _ = os.Stdout
