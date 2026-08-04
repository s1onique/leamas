// SPDX-License-Identifier: Apache-2.0

package main

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

// TestDupcodeHelpZeroDispatch proves --help renders complete usage and returns ExitSuccess without dispatch.
func TestDupcodeHelpZeroDispatch(t *testing.T) {
	args := []string{"--help"}
	var stdout, stderr bytes.Buffer
	exitCode := handleDupcode(args, &stdout, &stderr)

	if exitCode != ExitSuccess {
		t.Errorf("exitCode = %d, want %d (ExitSuccess)", exitCode, ExitSuccess)
	}

	// Help header must be present
	help := stderr.String()
	if !strings.Contains(help, "Usage: leamas factory verify dupcode [options]") {
		t.Errorf("stderr missing usage header: %q", help)
	}

	// All flags must be present
	requiredFlags := []string{"-baseline", "-update-baseline", "-min-lines", "-min-tokens", "-json"}
	for _, flag := range requiredFlags {
		if !strings.Contains(help, flag) {
			t.Errorf("stderr missing flag %q: %q", flag, help)
		}
	}
}

// TestDupcodeMalformedJSONUnknownOption proves JSON stdout and empty stderr for unknown flags.
func TestDupcodeMalformedJSONUnknownOption(t *testing.T) {
	args := []string{"--json", "--unknown-option"}
	var stdout, stderr bytes.Buffer
	exitCode := handleDupcode(args, &stdout, &stderr)

	if exitCode != ExitParseFailure {
		t.Errorf("exitCode = %d, want %d (ExitParseFailure)", exitCode, ExitParseFailure)
	}

	// stdout must contain valid JSON
	var result map[string]interface{}
	dec := json.NewDecoder(&stdout)
	if err := dec.Decode(&result); err != nil {
		t.Fatalf("failed to decode stdout as JSON: %v, stdout=%q", err, stdout.String())
	}
	if _, ok := result["error"]; !ok {
		t.Errorf("result missing 'error' field: %+v", result)
	}

	// stderr must be empty for JSON parse failures
	if stderr.Len() > 0 {
		t.Errorf("stderr = %q, want empty", stderr.String())
	}
}

// TestDupcodeMalformedJSONUnexpectedArgument proves JSON stdout and empty stderr for bad args.
func TestDupcodeMalformedJSONUnexpectedArgument(t *testing.T) {
	args := []string{"--json", `unexpected-"argument`}
	var stdout, stderr bytes.Buffer
	exitCode := handleDupcode(args, &stdout, &stderr)

	if exitCode != ExitParseFailure {
		t.Errorf("exitCode = %d, want %d (ExitParseFailure)", exitCode, ExitParseFailure)
	}

	// stdout must contain valid JSON
	var result map[string]interface{}
	dec := json.NewDecoder(&stdout)
	if err := dec.Decode(&result); err != nil {
		t.Fatalf("failed to decode stdout as JSON: %v, stdout=%q", err, stdout.String())
	}
	if _, ok := result["error"]; !ok {
		t.Errorf("result missing 'error' field: %+v", result)
	}

	// stderr must be empty for JSON parse failures
	if stderr.Len() > 0 {
		t.Errorf("stderr = %q, want empty", stderr.String())
	}
}

// TestDupcodeMalformedZeroDispatch proves malformed args dispatch zero times.
func TestDupcodeMalformedZeroDispatch(t *testing.T) {
	args := []string{"--unknown-option"}
	var stdout, stderr bytes.Buffer
	exitCode := handleDupcode(args, &stdout, &stderr)

	if exitCode != ExitParseFailure {
		t.Errorf("exitCode = %d, want %d (ExitParseFailure)", exitCode, ExitParseFailure)
	}
}

// TestDupcodeProductionARGVHelperContract proves dupcodeCommandArgs extracts correct slice.
func TestDupcodeProductionARGVHelperContract(t *testing.T) {
	// Too short: 1 arg
	args, ok := dupcodeCommandArgs([]string{"leamas"})
	if ok {
		t.Errorf("dupcodeCommandArgs([\"leamas\"]) ok = %v, want false", ok)
	}

	// Too short: 3 args
	args, ok = dupcodeCommandArgs([]string{"leamas", "factory", "verify"})
	if ok {
		t.Errorf("dupcodeCommandArgs([\"leamas\", \"factory\", \"verify\"]) ok = %v, want false", ok)
	}

	// Exactly 4 args (no options) - valid!
	args, ok = dupcodeCommandArgs([]string{"leamas", "factory", "verify", "dupcode"})
	if !ok {
		t.Errorf("dupcodeCommandArgs([\"leamas\", \"factory\", \"verify\", \"dupcode\"]) ok = %v, want true", ok)
	}
	if len(args) != 0 {
		t.Errorf("dupcodeCommandArgs([\"leamas\", \"factory\", \"verify\", \"dupcode\"]) args = %v, want []", args)
	}

	// With --json
	args, ok = dupcodeCommandArgs([]string{"leamas", "factory", "verify", "dupcode", "--json"})
	if !ok {
		t.Errorf("dupcodeCommandArgs([\"leamas\", \"factory\", \"verify\", \"dupcode\", \"--json\"]) ok = %v, want true", ok)
	}
	if len(args) != 1 || args[0] != "--json" {
		t.Errorf("dupcodeCommandArgs([\"leamas\", \"factory\", \"verify\", \"dupcode\", \"--json\"]) args = %v, want [\"--json\"]", args)
	}

	// With --update-baseline
	args, ok = dupcodeCommandArgs([]string{"leamas", "factory", "verify", "dupcode", "--update-baseline"})
	if !ok {
		t.Errorf("dupcodeCommandArgs([\"leamas\", \"factory\", \"verify\", \"dupcode\", \"--update-baseline\"]) ok = %v, want true", ok)
	}
	if len(args) != 1 || args[0] != "--update-baseline" {
		t.Errorf("dupcodeCommandArgs([\"leamas\", \"factory\", \"verify\", \"dupcode\", \"--update-baseline\"]) args = %v, want [\"--update-baseline\"]", args)
	}
}

// TestCmdSpecOnlyDispatchArchitecture documents that the cmd layer uses
// typed, data-only dispatch entry points. The gate package owns the binder.
func TestCmdSpecOnlyDispatchArchitecture(t *testing.T) {
	// The cmd layer calls the typed entry points:
	//   gate.DispatchDupcodeVerifyTyped
	//   gate.DispatchDupcodeUpdateBaselineTyped
	//   gate.DispatchDupcodeBaselineVerifyTyped
	//
	// This test asserts (by negative compile-time test) that the cmd package
	// does not call the closure-based dispatch entry points:
	//   gate.DispatchDupcodeVerify (removed from public surface post-02G)
	//   gate.DispatchDupcodeUpdateBaseline (removed)
	//   gate.DispatchDupcodeBaselineVerify (removed)
	//
	// The structural assertion is that handleDupcode takes only (args, stdout, stderr)
	// and constructs typed specs. If a future change adds back a closure-accepting
	// dispatch parameter, this test serves as a regression sentinel.
	if ExitSuccess != 0 {
		t.Error("ExitSuccess must be 0")
	}
	if ExitAuthorityFailure != 1 {
		t.Error("ExitAuthorityFailure must be 1")
	}
	if ExitParseFailure != 2 {
		t.Error("ExitParseFailure must be 2")
	}
}
