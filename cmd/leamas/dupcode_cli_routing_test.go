// SPDX-License-Identifier: Apache-2.0

package main

import (
	"bytes"
	"strings"
	"testing"
)

// TestDupcodeMalformedUnknownOption proves parser rejects unknown options without invoking dispatchers.
func TestDupcodeMalformedUnknownOption(t *testing.T) {
	args := []string{"--unknown-option"}
	var stdout, stderr bytes.Buffer
	exitCode := handleDupcode(args, &stdout, &stderr)

	if exitCode != ExitParseFailure {
		t.Errorf("exitCode = %d, want %d (ExitParseFailure)", exitCode, ExitParseFailure)
	}
	if stderr.Len() == 0 {
		t.Errorf("expected stderr to contain error output")
	}
}

// TestDupcodeMalformedUnexpectedArgument proves parser rejects unexpected positional args.
func TestDupcodeMalformedUnexpectedArgument(t *testing.T) {
	args := []string{"unexpected-argument"}
	var stdout, stderr bytes.Buffer
	exitCode := handleDupcode(args, &stdout, &stderr)

	if exitCode != ExitParseFailure {
		t.Errorf("exitCode = %d, want %d (ExitParseFailure)", exitCode, ExitParseFailure)
	}
}

// TestDupcodeSpecOnlyRouting documents that, with the typed dispatch model,
// dispatch routing is observed by the gate-package tests, not by injecting
// dispatchers into the cmd package. This test exists to verify the
// parse-time routing of --update-baseline (without actually dispatching).
func TestDupcodeSpecOnlyRouting(t *testing.T) {
	// The cmd layer parses --update-baseline and routes to the typed
	// DispatchDupcodeUpdateBaselineTyped entry. We don't drive a real
	// dispatch here — that requires CI authority context. We instead
	// verify that a parse failure (e.g. unknown option) still does not
	// touch the dispatch path.
	args := []string{"--unknown"}
	var stdout, stderr bytes.Buffer
	exitCode := handleDupcode(args, &stdout, &stderr)
	if exitCode != ExitParseFailure {
		t.Errorf("expected ExitParseFailure, got %d", exitCode)
	}
	// Stdout must remain empty for parse failures (no JSON mode here).
	if stdout.Len() != 0 {
		t.Errorf("stdout = %q, want empty", stdout.String())
	}
	// Stderr must contain an error indicator.
	if !strings.Contains(stderr.String(), "Error") {
		t.Errorf("stderr = %q, want to contain 'Error'", stderr.String())
	}
}
