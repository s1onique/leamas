// SPDX-License-Identifier: Apache-2.0

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/s1onique/leamas/internal/factory/checks"
	"github.com/s1onique/leamas/internal/factory/gate"
)

// sentinelErrorResult is used to test error handling with findings.
var sentinelErrorResult = gate.DispatchResult{
	Findings: []checks.Finding{
		{
			Path:     "dupcode",
			Kind:     "test-sentinel",
			Message:  "sentinel from test dispatcher",
			Severity: checks.SeverityWarn,
		},
	},
}

// TestDupcodeHelpZeroDispatch proves --help renders complete usage and returns ExitSuccess without dispatch.
func TestDupcodeHelpZeroDispatch(t *testing.T) {
	var verifyCalls int64
	var updateCalls int64

	dispatchers := dupcodeDispatchers{
		verify: func(ctx context.Context, root string, f gate.RunnerFactory) gate.DispatchResult {
			atomic.AddInt64(&verifyCalls, 1)
			return gate.DispatchResult{}
		},
		updateBaseline: func(ctx context.Context, root string, f gate.RunnerFactory) gate.DispatchResult {
			atomic.AddInt64(&updateCalls, 1)
			return gate.DispatchResult{}
		},
	}

	args := []string{"--help"}
	var stdout, stderr bytes.Buffer
	exitCode := handleDupcode(args, dispatchers, &stdout, &stderr)

	if atomic.LoadInt64(&verifyCalls) != 0 {
		t.Errorf("verifyCalls = %d, want 0", atomic.LoadInt64(&verifyCalls))
	}
	if atomic.LoadInt64(&updateCalls) != 0 {
		t.Errorf("updateCalls = %d, want 0", atomic.LoadInt64(&updateCalls))
	}
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

// TestDupcodeHumanSentinelPropagation proves sentinel findings propagate in human mode.
func TestDupcodeHumanSentinelPropagation(t *testing.T) {
	var verifyCalls int64

	dispatchers := dupcodeDispatchers{
		verify: func(ctx context.Context, root string, f gate.RunnerFactory) gate.DispatchResult {
			atomic.AddInt64(&verifyCalls, 1)
			return sentinelErrorResult
		},
		updateBaseline: func(ctx context.Context, root string, f gate.RunnerFactory) gate.DispatchResult {
			return gate.DispatchResult{}
		},
	}

	args := []string{}
	var stdout, stderr bytes.Buffer
	exitCode := handleDupcode(args, dispatchers, &stdout, &stderr)

	if atomic.LoadInt64(&verifyCalls) != 1 {
		t.Errorf("verifyCalls = %d, want 1", atomic.LoadInt64(&verifyCalls))
	}
	if exitCode != ExitAuthorityFailure {
		t.Errorf("exitCode = %d, want %d (ExitAuthorityFailure)", exitCode, ExitAuthorityFailure)
	}
	if !strings.Contains(stderr.String(), "sentinel from test dispatcher") {
		t.Errorf("stderr = %q, want to contain 'sentinel from test dispatcher'", stderr.String())
	}
}

// TestDupcodeJSONSentinelPropagation proves sentinel findings propagate in JSON mode.
func TestDupcodeJSONSentinelPropagation(t *testing.T) {
	var verifyCalls int64

	dispatchers := dupcodeDispatchers{
		verify: func(ctx context.Context, root string, f gate.RunnerFactory) gate.DispatchResult {
			atomic.AddInt64(&verifyCalls, 1)
			return sentinelErrorResult
		},
		updateBaseline: func(ctx context.Context, root string, f gate.RunnerFactory) gate.DispatchResult {
			return gate.DispatchResult{}
		},
	}

	args := []string{"--json"}
	var stdout, stderr bytes.Buffer
	exitCode := handleDupcode(args, dispatchers, &stdout, &stderr)

	if atomic.LoadInt64(&verifyCalls) != 1 {
		t.Errorf("verifyCalls = %d, want 1", atomic.LoadInt64(&verifyCalls))
	}
	if exitCode != ExitAuthorityFailure {
		t.Errorf("exitCode = %d, want %d (ExitAuthorityFailure)", exitCode, ExitAuthorityFailure)
	}

	var result map[string]interface{}
	dec := json.NewDecoder(&stdout)
	if err := dec.Decode(&result); err != nil {
		t.Fatalf("failed to decode stdout as JSON: %v", err)
	}
	if err, ok := result["error"].(string); !ok || !strings.Contains(err, "sentinel from test dispatcher") {
		t.Errorf("result[error] = %v, want to contain 'sentinel from test dispatcher'", result["error"])
	}
	if kind, ok := result["kind"].(string); !ok || kind != "test-sentinel" {
		t.Errorf("result[kind] = %v, want 'test-sentinel'", result["kind"])
	}
}

// TestDupcodeMalformedJSONUnknownOption proves JSON stdout and empty stderr for unknown flags.
func TestDupcodeMalformedJSONUnknownOption(t *testing.T) {
	var verifyCalls int64
	var updateCalls int64

	dispatchers := dupcodeDispatchers{
		verify: func(ctx context.Context, root string, f gate.RunnerFactory) gate.DispatchResult {
			atomic.AddInt64(&verifyCalls, 1)
			return gate.DispatchResult{}
		},
		updateBaseline: func(ctx context.Context, root string, f gate.RunnerFactory) gate.DispatchResult {
			atomic.AddInt64(&updateCalls, 1)
			return gate.DispatchResult{}
		},
	}

	args := []string{"--json", "--unknown-option"}
	var stdout, stderr bytes.Buffer
	exitCode := handleDupcode(args, dispatchers, &stdout, &stderr)

	if atomic.LoadInt64(&verifyCalls) != 0 {
		t.Errorf("verifyCalls = %d, want 0", atomic.LoadInt64(&verifyCalls))
	}
	if atomic.LoadInt64(&updateCalls) != 0 {
		t.Errorf("updateCalls = %d, want 0", atomic.LoadInt64(&updateCalls))
	}
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
	var verifyCalls int64
	var updateCalls int64

	dispatchers := dupcodeDispatchers{
		verify: func(ctx context.Context, root string, f gate.RunnerFactory) gate.DispatchResult {
			atomic.AddInt64(&verifyCalls, 1)
			return gate.DispatchResult{}
		},
		updateBaseline: func(ctx context.Context, root string, f gate.RunnerFactory) gate.DispatchResult {
			atomic.AddInt64(&updateCalls, 1)
			return gate.DispatchResult{}
		},
	}

	args := []string{"--json", `unexpected-"argument`}
	var stdout, stderr bytes.Buffer
	exitCode := handleDupcode(args, dispatchers, &stdout, &stderr)

	if atomic.LoadInt64(&verifyCalls) != 0 {
		t.Errorf("verifyCalls = %d, want 0", atomic.LoadInt64(&verifyCalls))
	}
	if atomic.LoadInt64(&updateCalls) != 0 {
		t.Errorf("updateCalls = %d, want 0", atomic.LoadInt64(&updateCalls))
	}
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

// TestDupcodeProductionARGVHelperContract proves dupcodeCommandArgs extracts correct slice.
func TestDupcodeProductionARGVHelperContract(t *testing.T) {
	// Normal case: leamas factory verify dupcode --update-baseline
	argv := []string{"leamas", "factory", "verify", "dupcode", "--update-baseline"}
	result := dupcodeCommandArgs(argv)
	if len(result) != 1 || result[0] != "--update-baseline" {
		t.Errorf("dupcodeCommandArgs(%v) = %v, want [\"--update-baseline\"]", argv, result)
	}

	// No args case
	result = dupcodeCommandArgs([]string{"leamas"})
	if result != nil {
		t.Errorf("dupcodeCommandArgs([\"leamas\"]) = %v, want nil", result)
	}

	// Exactly 4 args (insufficient)
	result = dupcodeCommandArgs([]string{"leamas", "factory", "verify", "dupcode"})
	if result != nil {
		t.Errorf("dupcodeCommandArgs([\"leamas\", \"factory\", \"verify\", \"dupcode\"]) = %v, want nil", result)
	}
}

// TestDupcodeProductionARGVContract proves production wrapper uses dupcodeCommandArgs.
func TestDupcodeProductionARGVContract(t *testing.T) {
	oldArgs := os.Args
	t.Cleanup(func() { os.Args = oldArgs })

	os.Args = []string{"leamas", "factory", "verify", "dupcode", "--update-baseline"}

	var verifyCalls int64
	var updateCalls int64

	dispatchers := dupcodeDispatchers{
		verify: func(ctx context.Context, root string, f gate.RunnerFactory) gate.DispatchResult {
			atomic.AddInt64(&verifyCalls, 1)
			return gate.DispatchResult{}
		},
		updateBaseline: func(ctx context.Context, root string, f gate.RunnerFactory) gate.DispatchResult {
			atomic.AddInt64(&updateCalls, 1)
			return gate.DispatchResult{}
		},
	}

	// Simulate what handleFactoryVerifyDupcode does
	args := dupcodeCommandArgs(os.Args)
	if args == nil {
		t.Fatal("dupcodeCommandArgs returned nil for valid argv")
	}

	var stdout, stderr bytes.Buffer
	exitCode := handleDupcode(args, dispatchers, &stdout, &stderr)

	if atomic.LoadInt64(&verifyCalls) != 0 {
		t.Errorf("verifyCalls = %d, want 0", atomic.LoadInt64(&verifyCalls))
	}
	if atomic.LoadInt64(&updateCalls) != 1 {
		t.Errorf("updateCalls = %d, want 1", atomic.LoadInt64(&updateCalls))
	}
	if exitCode != ExitSuccess {
		t.Errorf("exitCode = %d, want %d (ExitSuccess)", exitCode, ExitSuccess)
	}
}

// TestDupcodeShortARGVFailClosed proves short argv is handled safely.
func TestDupcodeShortARGVFailClosed(t *testing.T) {
	// dupcodeCommandArgs should return nil for insufficient args
	result := dupcodeCommandArgs([]string{"leamas"})
	if result != nil {
		t.Errorf("dupcodeCommandArgs(short) = %v, want nil", result)
	}
}
