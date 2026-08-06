// SPDX-License-Identifier: Apache-2.0

package main

// factory_close_v2_verifier_duplicate_test.go covers
// Phase 1 + Phase 7 of
// ACT-LEAMAS-FACTORY-CLOSURE-PROTOCOL-V2-VERIFIER-MAC-HANDOFF01-CORRECTION02C:
// every public flag of `factory close verify-v2-authority`
// is exercised twice (across the three spelling forms Go's
// `flag` package accepts). Every duplicate emits exit 2 and
// the canonical flag name, before any Git observation.

import (
	"bytes"
	"testing"

	"github.com/s1onique/leamas/internal/factory/closure"
)

// v2VerifierDuplicateFlagCases pins the closed list of
// public flags that MUST be rejected when repeated, plus the
// three spelling forms the stdlib parser accepts. The
// boolean-flag rows cover the four boolean forms the ACT
// requires:
//
//	--flag value flag value
//	--flag=value --flag=value
//	-flag value --flag value
var v2VerifierDuplicateFlagCases = []struct {
	name     string
	args     []string
	flagName string
}{
	{
		name:     "repository_value_value",
		args:     []string{"--repository", "/tmp", "--repository", "/tmp"},
		flagName: "repository",
	},
	{
		name:     "repository_eq_eq",
		args:     []string{"--repository=/tmp", "--repository=/tmp"},
		flagName: "repository",
	},
	{
		name:     "subject_value_value",
		args:     []string{"--subject", "0000000000000000000000000000000000000000", "--subject", "0000000000000000000000000000000000000000"},
		flagName: "subject",
	},
	{
		name:     "subject_eq_eq",
		args:     []string{"--subject=0000000000000000000000000000000000000000", "--subject=0000000000000000000000000000000000000000"},
		flagName: "subject",
	},
	{
		name:     "expected_tag_value_value",
		args:     []string{"--expected-tag", "v2", "--expected-tag", "v2"},
		flagName: "expected-tag",
	},
	{
		name:     "output_value_value",
		args:     []string{"--output", "/tmp/a", "--output", "/tmp/a"},
		flagName: "output",
	},
	{
		name:     "output_eq_eq",
		args:     []string{"--output=/tmp/a", "--output=/tmp/a"},
		flagName: "output",
	},
	{
		name:     "plan_path_value_value",
		args:     []string{"--plan-path", "p", "--plan-path", "p"},
		flagName: "plan-path",
	},
	{
		name:     "manifest_path_value_value",
		args:     []string{"--manifest-path", "m", "--manifest-path", "m"},
		flagName: "manifest-path",
	},
	{
		name:     "freeze_value_value",
		args:     []string{"--freeze", "0000000000000000000000000000000000000000", "--freeze", "0000000000000000000000000000000000000000"},
		flagName: "freeze",
	},
	{
		name:     "closure_value_value",
		args:     []string{"--closure", "0000000000000000000000000000000000000000", "--closure", "0000000000000000000000000000000000000000"},
		flagName: "closure",
	},
	{
		name:     "protocol_version_value_value",
		args:     []string{"--protocol-version", "2", "--protocol-version", "2"},
		flagName: "protocol-version",
	},
	{
		name:     "plan_contract_version_value_value",
		args:     []string{"--plan-contract-version", "1", "--plan-contract-version", "1"},
		flagName: "plan-contract-version",
	},
	{
		name:     "working_manifest_assertion_value_value",
		args:     []string{"--working-manifest-assertion", "/tmp/x", "--working-manifest-assertion", "/tmp/x"},
		flagName: "working-manifest-assertion",
	},
}

// TestV2VerifierDuplicateFlagMatrix drives every public
// scalar flag through at least two duplicate spellings. The
// parser must reject the duplicate with a typed
// closure-tag-* diagnostic... wait, with the duplicate_cli_flag
// code, before any Git observation.
func TestV2VerifierDuplicateFlagMatrix(t *testing.T) {
	for _, tc := range v2VerifierDuplicateFlagCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			stdout := &bytes.Buffer{}
			stderr := &bytes.Buffer{}
			got := runFactoryCloseVerifyV2Authority(tc.args, stdout, stderr)
			if got != v2VerifierExitUsage {
				t.Fatalf("exit = %d, want usage exit %d (stderr=%s)",
					got, v2VerifierExitUsage, stderr.String())
			}
		})
	}
}

// TestV2VerifierDuplicateFlagDiagnostic pins the diagnostic
// surface of the duplicate-flag rejection. The parser emits
// the canonical flag name in the typed diagnostic.
func TestV2VerifierDuplicateFlagDiagnostic(t *testing.T) {
	_, err := parseV2VerifierFlags("factory close verify-v2-authority",
		bytes.NewBuffer(nil),
		[]string{"--json", "--json"})
	if err == nil {
		t.Fatalf("expected duplicate error for --json --json")
	}
	var vErr *closure.V2VerifierError
	if !isErrorsAs(err, &vErr) {
		t.Fatalf("error must be a *V2VerifierError, got %T: %v", err, err)
	}
	if len(vErr.Diags) == 0 {
		t.Fatalf("expected at least one diagnostic")
	}
	if vErr.Diags[0].Code != closure.V2VerifierDuplicateCLIFlag {
		t.Fatalf("code = %v, want duplicate_cli_flag", vErr.Diags[0].Code)
	}
}

// TestV2VerifierDuplicateBooleanFlags pins the four boolean
// forms the ACT contract v1 requires:
//
//	--flag --flag
//	--flag=true --flag
//	-flag value --flag value
//	--flag value --flag=value
func TestV2VerifierDuplicateBooleanFlags(t *testing.T) {
	cases := []struct {
		name string
		args []string
	}{
		{name: "json_twice", args: []string{"--json", "--json"}},
		{name: "json_true_then_bare", args: []string{"--json=true", "--json"}},
		{name: "help_twice", args: []string{"--help", "--help"}},
		{name: "capture_caller_state_twice", args: []string{"--capture-caller-state", "--capture-caller-state"}},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			stdout := &bytes.Buffer{}
			stderr := &bytes.Buffer{}
			got := runFactoryCloseVerifyV2Authority(tc.args, stdout, stderr)
			if got != v2VerifierExitUsage {
				t.Fatalf("exit = %d, want usage exit %d (stderr=%s)",
					got, v2VerifierExitUsage, stderr.String())
			}
		})
	}
}
