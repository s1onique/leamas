// SPDX-License-Identifier: Apache-2.0

package main

// factory_close_v2_verifier_cli_test.go covers the CLI
// surface of `factory close verify-v2-authority` for
// ACT-LEAMAS-FACTORY-CLOSURE-PROTOCOL-V2-VERIFIER-CLI-TAG-STATE01.
//
// The CLI matrix covers:
//
//   - usage errors + exit 2 for missing/unknown flags
//   - help text + exit 0 on --help
//   - JSON envelope renders a single document when --json
//   - exit codes 0/2/3/4 are stable
//   - non-repository path routes to a typed failure (3 or 4)

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"io"
	"strings"
	"testing"

	"github.com/s1onique/leamas/internal/factory/closure"
)

// TestV2VerifierCLIUsageErrors exercises the usage-error
// surface required by Phase 2 of the ACT 4 contract.
func TestV2VerifierCLIUsageErrors(t *testing.T) {
	cases := []struct {
		name string
		args []string
		want int
	}{
		{
			name: "no_args",
			args: nil,
			want: v2VerifierExitUsage,
		},
		{
			name: "missing_repository",
			args: []string{
				"--subject", "1111111111111111111111111111111111111111",
				"--freeze", "2222222222222222222222222222222222222222",
				"--closure", "3333333333333333333333333333333333333333",
				"--plan-path", "plan/plan.json",
				"--manifest-path", "manifest/manifest.json",
			},
			want: v2VerifierExitUsage,
		},
		{
			name: "missing_closure",
			args: []string{
				"--repository", "/tmp/nope",
				"--subject", "1111111111111111111111111111111111111111",
				"--freeze", "2222222222222222222222222222222222222222",
				"--plan-path", "plan/plan.json",
				"--manifest-path", "manifest/manifest.json",
			},
			want: v2VerifierExitUsage,
		},
		{
			name: "unsupported_protocol",
			args: []string{
				"--protocol-version", "9",
				"--repository", "/tmp",
			},
			want: v2VerifierExitUsage,
		},
		{
			name: "unsupported_plan_contract",
			args: []string{
				"--plan-contract-version", "9",
				"--repository", "/tmp",
			},
			want: v2VerifierExitUsage,
		},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			stdout := &bytes.Buffer{}
			stderr := &bytes.Buffer{}
			got := runFactoryCloseVerifyV2Authority(tc.args, stdout, stderr)
			if got != tc.want {
				t.Fatalf("exit code = %d, want %d (stderr=%s stdout=%s)",
					got, tc.want, stderr.String(), stdout.String())
			}
		})
	}
}

// TestV2VerifierCLIHelpExitsZero asserts the help-text
// contract required by Phase 3 of the ACT 4 specification.
func TestV2VerifierCLIHelpExitsZero(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	got := runFactoryCloseVerifyV2Authority([]string{"--help"}, stdout, stderr)
	if got != v2VerifierExitSuccess {
		t.Fatalf("--help exit = %d, want %d (stderr=%s stdout=%s)",
			got, v2VerifierExitSuccess, stderr.String(), stdout.String())
	}
	combined := stdout.String() + stderr.String()
	for _, want := range []string{
		"Usage:", "--subject", "--freeze", "--closure",
		"--plan-path", "--manifest-path",
		"HEAD", "self-reference",
	} {
		if !strings.Contains(combined, want) {
			t.Fatalf("help text must mention %q, got: %s", want, combined)
		}
	}
}

// TestV2VerifierCLIHelperSet exercises parseV2VerifierFlags
// directly so CLI parser changes are flagged.
func TestV2VerifierCLIHelperSet(t *testing.T) {
	in, err := parseV2VerifierFlags("factory close verify-v2-authority", io.Discard, []string{
		"--repository", "/tmp",
		"--subject", "1111111111111111111111111111111111111111",
		"--freeze", "2222222222222222222222222222222222222222",
		"--closure", "3333333333333333333333333333333333333333",
		"--plan-path", "plan/plan.json",
		"--manifest-path", "manifest/manifest.json",
		"--expected-tag", "v2-test",
		"--json",
	})
	if err != nil {
		t.Fatalf("parseV2VerifierFlags: %v", err)
	}
	if !in.JSONOutput {
		t.Fatalf("JSONOutput flag not parsed")
	}
	if in.ExpectedTag != "v2-test" {
		t.Fatalf("ExpectedTag = %s, want v2-test", in.ExpectedTag)
	}
}

// TestV2VerifierCLIRepositoryMissingRoutesToObserverError
// verifies the exit-code surface for a non-repository
// path. The verifier MUST distinguish observer failure
// (exit 4 = broken git authority) from verifier failure
// (exit 3 = topology / manifest mismatch).
func TestV2VerifierCLIRepositoryMissingRoutesToObserverError(t *testing.T) {
	dir := t.TempDir()
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	got := runFactoryCloseVerifyV2Authority([]string{
		"--repository", dir,
		"--subject", "1111111111111111111111111111111111111111",
		"--freeze", "2222222222222222222222222222222222222222",
		"--closure", "3333333333333333333333333333333333333333",
		"--plan-path", "plan/plan.json",
		"--manifest-path", "manifest/manifest.json",
		"--json",
	}, stdout, stderr)
	if got == v2VerifierExitSuccess {
		t.Fatalf("non-repository path must not produce exit 0, stdout=%s stderr=%s",
			stdout.String(), stderr.String())
	}
	if got != v2VerifierExitObserverBroken && got != v2VerifierExitVerifier {
		t.Fatalf("expected exit %d (observer broken) or %d (verifier), got %d (stdout=%s stderr=%s)",
			v2VerifierExitObserverBroken, v2VerifierExitVerifier, got,
			stdout.String(), stderr.String())
	}
	dec := json.NewDecoder(stdout)
	var doc map[string]any
	if err := dec.Decode(&doc); err != nil {
		t.Fatalf("decode JSON envelope: %v (stdout=%s)", err, stdout.String())
	}
	if _, err := dec.Token(); err == nil {
		t.Fatalf("JSON output must be exactly one document, got extra tokens")
	}
}

// TestV2VerifierExitCodesAreStable documents the exit-code
// matrix required by ACT 4 Phase 2.
func TestV2VerifierExitCodesAreStable(t *testing.T) {
	want := map[string]int{
		"v2VerifierExitSuccess":        v2VerifierExitSuccess,
		"v2VerifierExitUsage":          v2VerifierExitUsage,
		"v2VerifierExitVerifier":       v2VerifierExitVerifier,
		"v2VerifierExitObserverBroken": v2VerifierExitObserverBroken,
	}
	got := map[string]int{
		"v2VerifierExitSuccess":        0,
		"v2VerifierExitUsage":          2,
		"v2VerifierExitVerifier":       3,
		"v2VerifierExitObserverBroken": 4,
	}
	for key, wantVal := range want {
		if got[key] != wantVal {
			t.Fatalf("exit-code drift on %s: want %d, got %d", key, wantVal, got[key])
		}
	}
}

// TestV2VerifierCLIJSONEnvelopeStable ensures the JSON
// envelope format stays stable: success and failure share
// the same envelope shape with `ok` and `verification`.
func TestV2VerifierCLIJSONEnvelopeStable(t *testing.T) {
	envelope := v2VerifierJSONEnvelope{
		OK:           true,
		Verification: closure.V2ClosureVerification{},
	}
	data, err := json.MarshalIndent(envelope, "", "  ")
	if err != nil {
		t.Fatalf("MarshalIndent: %v", err)
	}
	if !bytes.Contains(data, []byte(`"ok"`)) {
		t.Fatalf("envelope must include ok field, got %s", data)
	}
	if !bytes.Contains(data, []byte(`"verification"`)) {
		t.Fatalf("envelope must include verification field, got %s", data)
	}
}

// TestV2VerifierCliHelpIsFlagErrHelp ensures parseV2VerifierFlags
// short-circuits on --help via flag.ErrHelp.
func TestV2VerifierCliHelpIsFlagErrHelp(t *testing.T) {
	_, err := parseV2VerifierFlags("factory close verify-v2-authority", io.Discard, []string{"--help"})
	if err == nil {
		t.Fatalf("--help must produce a sentinel error, got nil")
	}
	if !errors.Is(err, flag.ErrHelp) {
		t.Fatalf("--help error must satisfy errors.Is(err, flag.ErrHelp), got %v", err)
	}
}
