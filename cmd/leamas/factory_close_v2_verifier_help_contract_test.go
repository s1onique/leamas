// SPDX-License-Identifier: Apache-2.0

package main

// factory_close_v2_verifier_help_contract_test.go covers
// Phase 9 of
// ACT-LEAMAS-FACTORY-CLOSURE-PROTOCOL-V2-VERIFIER-MAC-HANDOFF01-CORRECTION02C:
// the help/implementation parity contract. The test
// verifies that every public flag has help coverage, that
// every exit-code meaning is documented, and that the
// metadata contract block is documented when --expected-tag
// is supplied.

import (
	"bytes"
	"strings"
	"testing"
)

// TestV2VerifierHelpContractEveryFlagDocumented pins the
// flag/help parity: every public flag of
// `factory close verify-v2-authority` appears in the
// canonical help text.
func TestV2VerifierHelpContractEveryFlagDocumented(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	got := runFactoryCloseVerifyV2Authority([]string{"--help"}, stdout, stderr)
	if got != v2VerifierExitSuccess {
		t.Fatalf("--help must exit 0, got %d (stderr=%s)", got, stderr.String())
	}
	combined := stdout.String() + stderr.String()
	required := []string{
		"--repository",
		"--protocol-version",
		"--plan-contract-version",
		"--subject",
		"--freeze",
		"--closure",
		"--plan-path",
		"--manifest-path",
		"--working-manifest-assertion",
		"--expected-tag",
		"--output",
		"--json",
		"--capture-caller-state",
		"--help",
	}
	for _, name := range required {
		if !strings.Contains(combined, name) {
			t.Fatalf("help text must mention %q, got: %s", name, combined)
		}
	}
}

// TestV2VerifierHelpContractEveryExitDocumented pins the
// exit-code/help parity: the help text documents the four
// canonical exit meanings.
func TestV2VerifierHelpContractEveryExitDocumented(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	got := runFactoryCloseVerifyV2Authority([]string{"--help"}, stdout, stderr)
	if got != v2VerifierExitSuccess {
		t.Fatalf("--help must exit 0, got %d (stderr=%s)", got, stderr.String())
	}
	combined := stdout.String()
	required := []string{
		"0",
		"2",
		"3",
		"4",
		"usage error",
		"verifier rejection",
		"observer failure",
	}
	for _, snippet := range required {
		if !strings.Contains(combined, snippet) {
			t.Fatalf("help text must mention %q, got: %s", snippet, combined)
		}
	}
}

// TestV2VerifierHelpContractSFCPMDocumented pins the
// external-input / no-inference contract. The help text
// must state that S/F/C/P/M are explicit, the expected tag
// is optional, and HEAD inference is forbidden.
func TestV2VerifierHelpContractSFCPMDocumented(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	_ = runFactoryCloseVerifyV2Authority([]string{"--help"}, stdout, stderr)
	combined := stdout.String()
	required := []string{
		"never infers C from HEAD",
		"never inferred from HEAD",
		"never inferred",
		"annotated (not lightweight)",
		"Leamas-Closure-Protocol-Version",
	}
	for _, snippet := range required {
		if !strings.Contains(combined, snippet) {
			t.Fatalf("help text must mention %q, got: %s", snippet, combined)
		}
	}
}

// TestV2VerifierHelpContractMetadataBlockDocumented pins
// the metadata trailer block: every required key is
// documented, and the key names match the constants.
func TestV2VerifierHelpContractMetadataBlockDocumented(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	_ = runFactoryCloseVerifyV2Authority([]string{"--help"}, stdout, stderr)
	combined := stdout.String()
	required := []string{
		"Leamas-Closure-Protocol-Version: 2",
		"Leamas-Plan-Contract-Version: 1",
		"Leamas-Subject-Commit:",
		"Leamas-Freeze-Commit:",
		"Leamas-Closure-Commit:",
		"Leamas-Plan-Path:",
		"Leamas-Manifest-Path:",
	}
	for _, snippet := range required {
		if !strings.Contains(combined, snippet) {
			t.Fatalf("help text must mention %q, got: %s", snippet, combined)
		}
	}
}

// TestV2VerifierHelpContractObjectFormatPinned pins the
// object-format policy in the help text:
//
//	unsupported object format -> verifier rejection (exit 3)
//	unavailable object-format observation -> observer (exit 4)
func TestV2VerifierHelpContractObjectFormatPinned(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	_ = runFactoryCloseVerifyV2Authority([]string{"--help"}, stdout, stderr)
	combined := stdout.String()
	if !strings.Contains(combined, "unsupported_object_format") &&
		!strings.Contains(combined, "unsupported object format") {
		t.Fatalf("help text must mention unsupported object format, got: %s", combined)
	}
	if !strings.Contains(combined, "object-format observation") &&
		!strings.Contains(combined, "object-format") {
		t.Fatalf("help text must mention object-format observation, got: %s", combined)
	}
}
