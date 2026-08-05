// SPDX-License-Identifier: Apache-2.0

package main

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/s1onique/leamas/internal/factory/closure"
)

// v2Authority exit codes pin a stable surface for downstream tooling.
// 0 — success and a v2 manifest was published.
// 2 — usage error (unknown flag, missing flag, invalid combination).
// 3 — plan validation failure (frozen plan bytes are invalid).
// 4 — binary identity failure (running binary cannot be bound).
// 5 — manifest failure (construct, render, or write the v2 manifest).
// 6 — execution failure (clean run with a non-PASS verdict).
// 1 — any other failure detected by the runner.
const (
	v2ExitSuccess  = 0
	v2ExitUsage    = 2
	v2ExitPlan     = 3
	v2ExitIdentity = 4
	v2ExitManifest = 5
	v2ExitVerdict  = 6
	v2ExitOther    = 1
)

// v2AuthorityParsed is the parsed argument bundle for
// `factory close run-v2-authority`. The parser rejects unknown
// flags, repeated flags, and missing required fields before any
// runner state is touched.
type v2AuthorityParsed struct {
	Request    closure.V2Request
	JSONOutput bool
}

type v2CLIIdentityFailure struct {
	Code         string                `json:"code"`
	Message      string                `json:"message"`
	PropertyName string                `json:"property_name,omitempty"`
	Diagnostics  closure.V2Diagnostics `json:"diagnostics,omitempty"`
}

// parseV2AuthorityFlags parses the command arguments and
// returns a fully populated request or a typed error.
func parseV2AuthorityFlags(args []string) (v2AuthorityParsed, error) {
	fs := newCloseFlagSet("factory close run-v2-authority", io.Discard)
	var protocolRaw string
	var planContract int
	var working string
	var jsonOutput bool
	var out v2AuthorityParsed
	fs.StringVar(&protocolRaw, "protocol-version", string(closure.ClosureProtocolV2),
		"closure protocol version (only 2 is accepted)")
	fs.IntVar(&planContract, "plan-contract-version", int(closure.PlanContractV1),
		"plan contract version (only 1 is accepted)")
	fs.StringVar(&out.Request.RepositoryRoot, "repository", "", "repository root")
	fs.StringVar(&out.Request.SubjectCommit, "subject", "", "subject commit OID")
	fs.StringVar(&out.Request.FreezeCommit, "freeze", "", "freeze commit OID")
	fs.StringVar(&out.Request.PlanPath, "plan-path", "", "repository-relative plan path")
	fs.StringVar(&out.Request.EvidenceDirectory, "evidence-directory", "", "absolute detached evidence directory")
	fs.StringVar(&out.Request.ManifestOutput, "manifest-output", "", "absolute detached manifest output")
	fs.StringVar(&working, "working-plan-assertion", "", "optional absolute path whose bytes must match the frozen plan")
	fs.BoolVar(&jsonOutput, "json", false, "emit a single valid JSON document on stdout")
	if err := parseCloseFlags(fs, args); err != nil {
		return v2AuthorityParsed{}, err
	}
	if protocolRaw != string(closure.ClosureProtocolV2) {
		return v2AuthorityParsed{}, fmt.Errorf("--protocol-version must be %q, got %q", closure.ClosureProtocolV2, protocolRaw)
	}
	if planContract != int(closure.PlanContractV1) {
		return v2AuthorityParsed{}, fmt.Errorf("--plan-contract-version must be 1, got %d", planContract)
	}
	required := []struct {
		name, value string
	}{
		{"--repository", out.Request.RepositoryRoot},
		{"--subject", out.Request.SubjectCommit},
		{"--freeze", out.Request.FreezeCommit},
		{"--plan-path", out.Request.PlanPath},
		{"--evidence-directory", out.Request.EvidenceDirectory},
		{"--manifest-output", out.Request.ManifestOutput},
	}
	for _, field := range required {
		if field.value == "" {
			return v2AuthorityParsed{}, fmt.Errorf("%s is required", field.name)
		}
	}
	out.Request.ClosureProtocolVersion = closure.ClosureProtocolV2
	out.Request.PlanContractVersion = int(closure.PlanContractV1)
	out.Request.OptionalWorkingPlanAssertion = working
	out.JSONOutput = jsonOutput
	return out, nil
}

func writeV2AuthorityFailure(stdout, stderr io.Writer, jsonOutput bool, failure v2CLIIdentityFailure) {
	if jsonOutput {
		data, _ := json.MarshalIndent(struct {
			OK          bool                  `json:"ok"`
			Code        string                `json:"code"`
			Message     string                `json:"message"`
			Property    string                `json:"property_name,omitempty"`
			Diagnostics closure.V2Diagnostics `json:"diagnostics,omitempty"`
		}{
			OK: false, Code: failure.Code, Message: failure.Message,
			Property: failure.PropertyName, Diagnostics: failure.Diagnostics,
		}, "", "  ")
		fmt.Fprintln(stdout, string(data))
		return
	}
	if failure.PropertyName != "" {
		fmt.Fprintf(stderr, "%s (%s): %s\n", failure.Code, failure.PropertyName, failure.Message)
	} else {
		fmt.Fprintf(stderr, "%s: %s\n", failure.Code, failure.Message)
	}
	for _, diag := range failure.Diagnostics {
		fmt.Fprintf(stderr, "  %s [%s]: %s\n", diag.Code, diag.PropertyName, diag.Message)
	}
}

func writeV2AuthorityTextError(stderr io.Writer, command string, err error) {
	fmt.Fprintf(stderr, "%s: %v\n", command, err)
}

func v2ExitFailure(err error) int {
	if v2err, ok := err.(*closure.V2Error); ok {
		for _, diag := range v2err.Diags {
			switch diag.Code {
			case closure.V2CodeFrozenPlanInvalid, closure.V2CodeFrozenPlanNotBlob, closure.V2CodeFrozenPlanPathMissing:
				return v2ExitPlan
			case closure.V2CodeBinaryIdentityInvalid, closure.V2CodeManifestIdentityInvalid, closure.V2CodeCheckResultMappingInvalid:
				return v2ExitIdentity
			case closure.V2CodeManifestWriteFailed:
				return v2ExitManifest
			}
		}
		for _, diag := range v2err.Diags {
			if diag.Code == closure.V2CodeExecutionFailed {
				return v2ExitVerdict
			}
		}
	}
	return v2ExitOther
}
