// SPDX-License-Identifier: Apache-2.0

package main

import (
	"bytes"
	"context"
	"sync/atomic"
	"testing"

	"github.com/s1onique/leamas/internal/factory/gate"
)

// sentinelEmptyResult is used for routing tests - no findings means success path.
var sentinelEmptyResult = gate.DispatchResult{}

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

	args := []string{}
	var stdout, stderr bytes.Buffer
	exitCode := handleDupcode(args, dispatchers, &stdout, &stderr)

	if atomic.LoadInt64(&verifyCalls) != 1 {
		t.Errorf("verifyCalls = %d, want 1", atomic.LoadInt64(&verifyCalls))
	}
	if atomic.LoadInt64(&updateCalls) != 0 {
		t.Errorf("updateCalls = %d, want 0", atomic.LoadInt64(&updateCalls))
	}
	if exitCode != ExitSuccess {
		t.Errorf("exitCode = %d, want %d (ExitSuccess)", exitCode, ExitSuccess)
	}
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

	args := []string{"--update-baseline"}
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

// TestDupcodeMalformedUnknownOption proves parser rejects unknown options without invoking dispatchers.
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

	args := []string{"--unknown-option"}
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
	if stderr.Len() == 0 {
		t.Errorf("expected stderr to contain error output")
	}
}

// TestDupcodeMalformedUnexpectedArgument proves parser rejects unexpected positional args.
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

	args := []string{"unexpected-argument"}
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
}
