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
//	    --manifest-output <file> \
//	    --evidence-directory <dir>

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/s1onique/leamas/internal/factory/closure"
)

func runFactoryCloseRunV2Authority(args []string, stdout, stderr io.Writer) int {
	fs := newCloseFlagSet("factory close run-v2-authority", stderr)
	var req closure.V2Request
	fs.StringVar((*string)(&req.ClosureProtocolVersion), "protocol-version", "2",
		"closure protocol version (1 or 2)")
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
	var jsonOutput bool
	fs.BoolVar(&jsonOutput, "json", false, "emit structured JSON diagnostics")
	if err := parseCloseFlags(fs, args); err != nil {
		return reportCloseFlagError(stderr, "factory close run-v2-authority", err,
			"all required flags must be supplied")
	}
	if req.ClosureProtocolVersion == "" {
		req.ClosureProtocolVersion = closure.ClosureProtocolV2
	}
	if req.PlanContractVersion == 0 {
		req.PlanContractVersion = int(closure.PlanContractV1)
	}
	manifest, err := closure.RunClosureProtocolV2(context.Background(), req)
	if err != nil {
		v2err, ok := err.(*closure.V2Error)
		if ok {
			if jsonOutput {
				data, _ := json.MarshalIndent(struct {
					Code  string                `json:"code"`
					Diags closure.V2Diagnostics `json:"diagnostics"`
				}{Code: "v2_failure", Diags: v2err.Diags}, "", "  ")
				fmt.Fprintln(stdout, string(data))
			} else {
				for _, d := range v2err.Diags {
					fmt.Fprintf(stderr, "%s: %s (%s)\n", d.Code, d.Message, d.PropertyName)
				}
			}
		}
		return reportCloseError(stderr, "factory close run-v2-authority", err)
	}
	if jsonOutput {
		data, _ := json.MarshalIndent(manifest, "", "  ")
		fmt.Fprintln(stdout, string(data))
	} else {
		fmt.Fprintln(stdout, "OK")
		fmt.Fprintf(stderr,
			"Closure Protocol v2: S=%s F=%s plan_blob=%s plan_sha256=%s execution_tree=%s\n",
			trunc8(manifest.SubjectCommit), trunc8(manifest.FreezeCommit),
			trunc8(manifest.PlanBlob), trunc8(manifest.PlanSHA256),
			trunc8(manifest.ExecutionTree))
	}
	return closeSuccessCode()
}

// ensure imports used even if the surrounding wiring shifts.
var _ = os.Stdout
