// SPDX-License-Identifier: Apache-2.0

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/s1onique/leamas/internal/factory/checks"
	"github.com/s1onique/leamas/internal/factory/gate"
	"github.com/s1onique/leamas/internal/factory/verifierdispatch"
)

// countingBaselineDispatcher records the number of typed dispatcher calls and
// returns a fully-controlled DupcodeBaselineOutcome. It is the test seam
// for the standalone dupcode-baseline renderer.
type countingBaselineDispatcher struct {
	calls atomic.Int64
	out   gate.DupcodeBaselineOutcome
}

func (c *countingBaselineDispatcher) dispatch(ctx context.Context, root string, spec gate.DupcodeBaselineSpec) gate.DupcodeBaselineOutcome {
	c.calls.Add(1)
	return c.out
}

// TestDupcodeBaselineErrorHuman proves a Dispatch.Error outcome surfaces
// as a stderr message in human mode and never reaches the typed-payload
// rendering path.
func TestDupcodeBaselineErrorHuman(t *testing.T) {
	d := &countingBaselineDispatcher{out: gate.DupcodeBaselineOutcome{
		Dispatch: verifierdispatch.Result{Error: errSentinel("sentinel dispatcher error")},
	}}
	var stdout, stderr bytes.Buffer
	exit := runFactoryVerifyDupcodeBaseline(
		[]string{}, &stdout, &stderr, d.dispatch,
	)
	if exit != 1 {
		t.Errorf("exit = %d, want 1", exit)
	}
	if got := d.calls.Load(); got != 1 {
		t.Errorf("dispatch calls = %d, want 1", got)
	}
	if !strings.Contains(stderr.String(), "sentinel dispatcher error") {
		t.Errorf("stderr = %q, want to contain 'sentinel dispatcher error'", stderr.String())
	}
	if stdout.Len() != 0 {
		t.Errorf("stdout = %q, want empty", stdout.String())
	}
}

// TestDupcodeBaselineErrorJSON proves a Dispatch.Error outcome in JSON mode
// is encoded as {"error":"..."} with empty stderr.
func TestDupcodeBaselineErrorJSON(t *testing.T) {
	d := &countingBaselineDispatcher{out: gate.DupcodeBaselineOutcome{
		Dispatch: verifierdispatch.Result{Error: errSentinel("sentinel dispatcher error")},
	}}
	var stdout, stderr bytes.Buffer
	exit := runFactoryVerifyDupcodeBaseline(
		[]string{"--json"}, &stdout, &stderr, d.dispatch,
	)
	if exit != 1 {
		t.Errorf("exit = %d, want 1", exit)
	}
	if got := d.calls.Load(); got != 1 {
		t.Errorf("dispatch calls = %d, want 1", got)
	}
	var decoded map[string]interface{}
	if err := json.Unmarshal(stdout.Bytes(), &decoded); err != nil {
		t.Fatalf("stdout JSON decode: %v; stdout=%q", err, stdout.String())
	}
	if got, _ := decoded["error"].(string); !strings.Contains(got, "sentinel dispatcher error") {
		t.Errorf("error = %v, want to contain 'sentinel dispatcher error'", decoded["error"])
	}
	if stderr.Len() > 0 {
		t.Errorf("stderr = %q, want empty", stderr.String())
	}
}

// TestDupcodeBaselineFindingHuman proves a Dispatch.Findings outcome
// carrying an authority-denial kind surfaces as a stderr message in
// human mode and never reaches success rendering.
func TestDupcodeBaselineFindingHuman(t *testing.T) {
	d := &countingBaselineDispatcher{out: gate.DupcodeBaselineOutcome{
		Dispatch: verifierdispatch.Result{
			Findings: []checks.Finding{{
				Path:    "dupcode-baseline",
				Kind:    "verifier_execution_authority_denied",
				Message: "sentinel baseline authority denial",
			}},
		},
	}}
	var stdout, stderr bytes.Buffer
	exit := runFactoryVerifyDupcodeBaseline(
		[]string{}, &stdout, &stderr, d.dispatch,
	)
	if exit != 1 {
		t.Errorf("exit = %d, want 1", exit)
	}
	if !strings.Contains(stderr.String(), "sentinel baseline authority denial") {
		t.Errorf("stderr = %q, want to contain 'sentinel baseline authority denial'", stderr.String())
	}
	if strings.Contains(stdout.String(), "ok") {
		t.Errorf("stdout = %q, must not contain success payload", stdout.String())
	}
}

// TestDupcodeBaselineFindingJSON proves a Dispatch.Findings outcome in
// JSON mode is encoded as {"error":..., "kind":...} with empty stderr
// when the finding is an authority-denial finding.
func TestDupcodeBaselineFindingJSON(t *testing.T) {
	d := &countingBaselineDispatcher{out: gate.DupcodeBaselineOutcome{
		Dispatch: verifierdispatch.Result{
			Findings: []checks.Finding{{
				Path:    "dupcode-baseline",
				Kind:    "verifier_execution_authority_denied",
				Message: "sentinel baseline authority denial",
			}},
		},
	}}
	var stdout, stderr bytes.Buffer
	exit := runFactoryVerifyDupcodeBaseline(
		[]string{"--json"}, &stdout, &stderr, d.dispatch,
	)
	if exit != 1 {
		t.Errorf("exit = %d, want 1", exit)
	}
	var decoded map[string]interface{}
	if err := json.Unmarshal(stdout.Bytes(), &decoded); err != nil {
		t.Fatalf("stdout JSON decode: %v; stdout=%q", err, stdout.String())
	}
	if got, want := decoded["error"], "sentinel baseline authority denial"; got != want {
		t.Errorf("error = %v, want %v", got, want)
	}
	if got, want := decoded["kind"], "verifier_execution_authority_denied"; got != want {
		t.Errorf("kind = %v, want %v", got, want)
	}
	if stderr.Len() > 0 {
		t.Errorf("stderr = %q, want empty", stderr.String())
	}
}

// TestDupcodeBaselineHelpZeroDispatch proves --help renders usage and
// returns ExitSuccess without invoking the dispatcher.
func TestDupcodeBaselineHelpZeroDispatch(t *testing.T) {
	d := &countingBaselineDispatcher{}
	var stdout, stderr bytes.Buffer
	exit := runFactoryVerifyDupcodeBaseline(
		[]string{"--help"}, &stdout, &stderr, d.dispatch,
	)
	if exit != 0 {
		t.Errorf("exit = %d, want 0", exit)
	}
	if got := d.calls.Load(); got != 0 {
		t.Errorf("dispatch calls = %d, want 0", got)
	}
	if !strings.Contains(stderr.String(), "Usage:") {
		t.Errorf("stderr = %q, want to contain 'Usage:'", stderr.String())
	}
}

// TestDupcodeBaselineMalformedUnknownFlag proves an unknown flag exits
// with ExitParseFailure and never invokes the dispatcher.
func TestDupcodeBaselineMalformedUnknownFlag(t *testing.T) {
	d := &countingBaselineDispatcher{}
	var stdout, stderr bytes.Buffer
	exit := runFactoryVerifyDupcodeBaseline(
		[]string{"--unknown-option"}, &stdout, &stderr, d.dispatch,
	)
	if exit != 2 {
		t.Errorf("exit = %d, want 2", exit)
	}
	if got := d.calls.Load(); got != 0 {
		t.Errorf("dispatch calls = %d, want 0", got)
	}
}

// TestDupcodeBaselineMalformedPositionalArgument proves an unexpected
// positional argument exits with ExitParseFailure and never invokes the
// dispatcher.
func TestDupcodeBaselineMalformedPositionalArgument(t *testing.T) {
	d := &countingBaselineDispatcher{}
	var stdout, stderr bytes.Buffer
	exit := runFactoryVerifyDupcodeBaseline(
		[]string{"unexpected-argument"}, &stdout, &stderr, d.dispatch,
	)
	if exit != 2 {
		t.Errorf("exit = %d, want 2", exit)
	}
	if got := d.calls.Load(); got != 0 {
		t.Errorf("dispatch calls = %d, want 0", got)
	}
}

// errSentinel is a minimal error type used by the failure-channel tests.
type errSentinel string

func (e errSentinel) Error() string { return string(e) }
