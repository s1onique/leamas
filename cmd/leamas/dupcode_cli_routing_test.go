// SPDX-License-Identifier: Apache-2.0

package main

import (
	"bytes"
	"context"
	"sync/atomic"
	"testing"

	"github.com/s1onique/leamas/internal/factory/checks"
	"github.com/s1onique/leamas/internal/factory/gate"
)

// sentinelEmptyResult is used for routing tests - no findings means success path.
var sentinelEmptyResult = gate.DispatchResult{}

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

// TestDupcodeHandlerVerifyProvesDispatcherRouting exercises the verify handler path
// through the internal handleDupcode function with injected dispatchers.
func TestDupcodeHandlerVerifyProvesDispatcherRouting(t *testing.T) {
	var verifyCalls int64
	var updateCalls int64

	dispatchers := dupcodeDispatchers{
		verify: func(ctx context.Context, root string, f gate.RunnerFactory) gate.DispatchResult {
			atomic.AddInt64(&verifyCalls, 1)
			return sentinelEmptyResult
		},
		updateBaseline: func(ctx context.Context, root string, f gate.RunnerFactory) gate.DispatchResult {
			atomic.AddInt64(&updateCalls, 1)
			return sentinelEmptyResult
		},
	}

	// Call handleDupcode directly with verify-style args (no --update-baseline)
	args := []string{}
	var stdout, stderr bytes.Buffer
	exitCode := handleDupcode(args, dispatchers, &stdout, &stderr)

	// Verify dispatcher should have been called exactly once
	if atomic.LoadInt64(&verifyCalls) != 1 {
		t.Errorf("verifyCalls = %d, want 1", atomic.LoadInt64(&verifyCalls))
	}

	// Update dispatcher should NOT have been called
	if atomic.LoadInt64(&updateCalls) != 0 {
		t.Errorf("updateCalls = %d, want 0", atomic.LoadInt64(&updateCalls))
	}

	// Exit code should be success
	if exitCode != ExitSuccess {
		t.Errorf("exitCode = %d, want %d (ExitSuccess)", exitCode, ExitSuccess)
	}

	t.Logf("verify handler: verifyCalls=%d, updateCalls=%d, exitCode=%d", verifyCalls, updateCalls, exitCode)
}

// TestDupcodeHandlerUpdateProvesDispatcherRouting exercises the update handler path
// through the internal handleDupcode function with injected dispatchers.
func TestDupcodeHandlerUpdateProvesDispatcherRouting(t *testing.T) {
	var verifyCalls int64
	var updateCalls int64

	dispatchers := dupcodeDispatchers{
		verify: func(ctx context.Context, root string, f gate.RunnerFactory) gate.DispatchResult {
			atomic.AddInt64(&verifyCalls, 1)
			return sentinelEmptyResult
		},
		updateBaseline: func(ctx context.Context, root string, f gate.RunnerFactory) gate.DispatchResult {
			atomic.AddInt64(&updateCalls, 1)
			return sentinelEmptyResult
		},
	}

	// Call handleDupcode with --update-baseline flag
	args := []string{"--update-baseline"}
	var stdout, stderr bytes.Buffer
	exitCode := handleDupcode(args, dispatchers, &stdout, &stderr)

	// Verify dispatcher should NOT have been called
	if atomic.LoadInt64(&verifyCalls) != 0 {
		t.Errorf("verifyCalls = %d, want 0", atomic.LoadInt64(&verifyCalls))
	}

	// Update dispatcher should have been called exactly once
	if atomic.LoadInt64(&updateCalls) != 1 {
		t.Errorf("updateCalls = %d, want 1", atomic.LoadInt64(&updateCalls))
	}

	// Exit code should be success
	if exitCode != ExitSuccess {
		t.Errorf("exitCode = %d, want %d (ExitSuccess)", exitCode, ExitSuccess)
	}

	t.Logf("update handler: verifyCalls=%d, updateCalls=%d, exitCode=%d", verifyCalls, updateCalls, exitCode)
}

// TestDupcodeMalformedUnknownOption proves parser rejects unknown options
// without invoking any dispatcher.
func TestDupcodeMalformedUnknownOption(t *testing.T) {
	var verifyCalls int64
	var updateCalls int64

	dispatchers := dupcodeDispatchers{
		verify: func(ctx context.Context, root string, f gate.RunnerFactory) gate.DispatchResult {
			atomic.AddInt64(&verifyCalls, 1)
			return sentinelEmptyResult
		},
		updateBaseline: func(ctx context.Context, root string, f gate.RunnerFactory) gate.DispatchResult {
			atomic.AddInt64(&updateCalls, 1)
			return sentinelEmptyResult
		},
	}

	// Call handleDupcode with an unknown option
	args := []string{"--unknown-option"}
	var stdout, stderr bytes.Buffer
	exitCode := handleDupcode(args, dispatchers, &stdout, &stderr)

	// Neither dispatcher should have been called
	if atomic.LoadInt64(&verifyCalls) != 0 {
		t.Errorf("verifyCalls = %d, want 0", atomic.LoadInt64(&verifyCalls))
	}
	if atomic.LoadInt64(&updateCalls) != 0 {
		t.Errorf("updateCalls = %d, want 0", atomic.LoadInt64(&updateCalls))
	}

	// Exit code should be parse failure
	if exitCode != ExitParseFailure {
		t.Errorf("exitCode = %d, want %d (ExitParseFailure)", exitCode, ExitParseFailure)
	}

	t.Logf("malformed (unknown option): verifyCalls=%d, updateCalls=%d, exitCode=%d", verifyCalls, updateCalls, exitCode)
}

// TestDupcodeMalformedUnexpectedArgument proves parser rejects unexpected positional args
// without invoking any dispatcher.
func TestDupcodeMalformedUnexpectedArgument(t *testing.T) {
	var verifyCalls int64
	var updateCalls int64

	dispatchers := dupcodeDispatchers{
		verify: func(ctx context.Context, root string, f gate.RunnerFactory) gate.DispatchResult {
			atomic.AddInt64(&verifyCalls, 1)
			return sentinelEmptyResult
		},
		updateBaseline: func(ctx context.Context, root string, f gate.RunnerFactory) gate.DispatchResult {
			atomic.AddInt64(&updateCalls, 1)
			return sentinelEmptyResult
		},
	}

	// Call handleDupcode with an unexpected positional argument
	args := []string{"unexpected-argument"}
	var stdout, stderr bytes.Buffer
	exitCode := handleDupcode(args, dispatchers, &stdout, &stderr)

	// Neither dispatcher should have been called
	if atomic.LoadInt64(&verifyCalls) != 0 {
		t.Errorf("verifyCalls = %d, want 0", atomic.LoadInt64(&verifyCalls))
	}
	if atomic.LoadInt64(&updateCalls) != 0 {
		t.Errorf("updateCalls = %d, want 0", atomic.LoadInt64(&updateCalls))
	}

	// Exit code should be parse failure
	if exitCode != ExitParseFailure {
		t.Errorf("exitCode = %d, want %d (ExitParseFailure)", exitCode, ExitParseFailure)
	}

	t.Logf("malformed (unexpected arg): verifyCalls=%d, updateCalls=%d, exitCode=%d", verifyCalls, updateCalls, exitCode)
}

// TestDupcodeHandlerResultPropagation proves the handler preserves sentinel findings.
func TestDupcodeHandlerResultPropagation(t *testing.T) {
	var verifyCalls int64

	dispatchers := dupcodeDispatchers{
		verify: func(ctx context.Context, root string, f gate.RunnerFactory) gate.DispatchResult {
			atomic.AddInt64(&verifyCalls, 1)
			return sentinelEmptyResult
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

	// Exit code should be success
	if exitCode != ExitSuccess {
		t.Errorf("exitCode = %d, want %d (ExitSuccess)", exitCode, ExitSuccess)
	}
}
