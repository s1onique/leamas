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

// TestDupcodeHelpZeroDispatch proves --help exits successfully without calling dispatchers.
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
	if !strings.Contains(stderr.String(), "Usage: leamas factory verify dupcode") {
		t.Errorf("stderr = %q, want to contain 'Usage: leamas factory verify dupcode'", stderr.String())
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

// TestDupcodeMalformedJSONUnknownOption proves JSON output for unknown options is valid.
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

	var result map[string]interface{}
	dec := json.NewDecoder(&stderr)
	if err := dec.Decode(&result); err != nil {
		t.Fatalf("failed to decode stderr as JSON: %v, stderr=%q", err, stderr.String())
	}
	if _, ok := result["error"]; !ok {
		t.Errorf("result missing 'error' field")
	}
}

// TestDupcodeMalformedJSONUnexpectedArgument proves JSON output for unexpected args is valid.
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

	var result map[string]interface{}
	dec := json.NewDecoder(&stderr)
	if err := dec.Decode(&result); err != nil {
		t.Fatalf("failed to decode stderr as JSON: %v, stderr=%q", err, stderr.String())
	}
	if _, ok := result["error"]; !ok {
		t.Errorf("result missing 'error' field")
	}
}

// TestDupcodeProductionARGVContract proves the production wrapper passes correct args.
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

	args := os.Args[4:]
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
