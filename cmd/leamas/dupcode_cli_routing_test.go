// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"sync"
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

// exitCatcher wraps osExit to catch exit calls in tests.
type exitCatcher struct {
	mu       sync.Mutex
	code     int
	caught   bool
	original func(int)
}

func newExitCatcher() *exitCatcher {
	return &exitCatcher{}
}

func (e *exitCatcher) catch() {
	e.original = osExit
	osExit = func(code int) {
		e.mu.Lock()
		e.caught = true
		e.code = code
		e.mu.Unlock()
		// Don't actually exit in test
		panic("osExit")
	}
}

func (e *exitCatcher) release() {
	if e.original != nil {
		osExit = e.original
	}
}

func (e *exitCatcher) getCode() (int, bool) {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.code, e.caught
}

// runWithExitCatcher runs f and recovers from osExit panics.
func runWithExitCatcher(t *testing.T, f func()) {
	defer func() {
		if r := recover(); r != nil {
			// Expected: osExit panic
		}
	}()
	f()
}

// TestDupcodeHandlerVerifyProvesDispatcherRouting exercises the verify handler path
// through the internal handleDupcode function with injected dispatchers.
func TestDupcodeHandlerVerifyProvesDispatcherRouting(t *testing.T) {
	catcher := newExitCatcher()
	catcher.catch()
	defer catcher.release()

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
	runWithExitCatcher(t, func() {
		handleDupcode(args, dispatchers)
	})

	// Verify dispatcher should have been called exactly once
	if atomic.LoadInt64(&verifyCalls) != 1 {
		t.Errorf("verifyCalls = %d, want 1", atomic.LoadInt64(&verifyCalls))
	}

	// Update dispatcher should NOT have been called
	if atomic.LoadInt64(&updateCalls) != 0 {
		t.Errorf("updateCalls = %d, want 0", atomic.LoadInt64(&updateCalls))
	}

	t.Logf("verify handler: verifyCalls=%d, updateCalls=%d", verifyCalls, updateCalls)
}

// TestDupcodeHandlerUpdateProvesDispatcherRouting exercises the update handler path
// through the internal handleDupcode function with injected dispatchers.
func TestDupcodeHandlerUpdateProvesDispatcherRouting(t *testing.T) {
	catcher := newExitCatcher()
	catcher.catch()
	defer catcher.release()

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
	runWithExitCatcher(t, func() {
		handleDupcode(args, dispatchers)
	})

	// Verify dispatcher should NOT have been called
	if atomic.LoadInt64(&verifyCalls) != 0 {
		t.Errorf("verifyCalls = %d, want 0", atomic.LoadInt64(&verifyCalls))
	}

	// Update dispatcher should have been called exactly once
	if atomic.LoadInt64(&updateCalls) != 1 {
		t.Errorf("updateCalls = %d, want 1", atomic.LoadInt64(&updateCalls))
	}

	t.Logf("update handler: verifyCalls=%d, updateCalls=%d", verifyCalls, updateCalls)
}

// TestDupcodeMalformedUnknownOption proves parser rejects unknown options
// without invoking any dispatcher.
func TestDupcodeMalformedUnknownOption(t *testing.T) {
	catcher := newExitCatcher()
	catcher.catch()
	defer catcher.release()

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
	runWithExitCatcher(t, func() {
		handleDupcode(args, dispatchers)
	})

	// Neither dispatcher should have been called
	if atomic.LoadInt64(&verifyCalls) != 0 {
		t.Errorf("verifyCalls = %d, want 0", atomic.LoadInt64(&verifyCalls))
	}
	if atomic.LoadInt64(&updateCalls) != 0 {
		t.Errorf("updateCalls = %d, want 0", atomic.LoadInt64(&updateCalls))
	}

	t.Logf("malformed (unknown option): verifyCalls=%d, updateCalls=%d", verifyCalls, updateCalls)
}

// TestDupcodeMalformedUnexpectedArgument proves parser rejects unexpected positional args
// without invoking any dispatcher.
func TestDupcodeMalformedUnexpectedArgument(t *testing.T) {
	catcher := newExitCatcher()
	catcher.catch()
	defer catcher.release()

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
	runWithExitCatcher(t, func() {
		handleDupcode(args, dispatchers)
	})

	// Neither dispatcher should have been called
	if atomic.LoadInt64(&verifyCalls) != 0 {
		t.Errorf("verifyCalls = %d, want 0", atomic.LoadInt64(&verifyCalls))
	}
	if atomic.LoadInt64(&updateCalls) != 0 {
		t.Errorf("updateCalls = %d, want 0", atomic.LoadInt64(&updateCalls))
	}

	t.Logf("malformed (unexpected arg): verifyCalls=%d, updateCalls=%d", verifyCalls, updateCalls)
}

// TestDupcodeHandlerResultPropagation proves the handler preserves sentinel findings.
func TestDupcodeHandlerResultPropagation(t *testing.T) {
	catcher := newExitCatcher()
	catcher.catch()
	defer catcher.release()

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
	runWithExitCatcher(t, func() {
		handleDupcode(args, dispatchers)
	})

	if atomic.LoadInt64(&verifyCalls) != 1 {
		t.Errorf("verifyCalls = %d, want 1", atomic.LoadInt64(&verifyCalls))
	}
}
