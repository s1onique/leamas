// SPDX-License-Identifier: Apache-2.0

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/s1onique/leamas/internal/factory/checks"
	"github.com/s1onique/leamas/internal/factory/gate"
	"github.com/s1onique/leamas/internal/factory/verifierdispatch"
)

// baselineWriterTestDispatcher records the number of typed dispatcher
// calls and returns a fully-controlled DupcodeBaselineOutcome. It is
// the test seam for the standalone dupcode-baseline renderer.
type baselineWriterTestDispatcher struct {
	calls atomic.Int64
	out   gate.DupcodeBaselineOutcome
}

func (c *baselineWriterTestDispatcher) dispatch(ctx context.Context, root string, spec gate.DupcodeBaselineSpec) gate.DupcodeBaselineOutcome {
	c.calls.Add(1)
	return c.out
}

// TestDupcodeBaselineHumanSuccessWriterIsolated proves a human-mode
// success writes only to the supplied stdout, with no leakage to process
// stdout and an exit code of 0.
func TestDupcodeBaselineHumanSuccessWriterIsolated(t *testing.T) {
	d := &baselineWriterTestDispatcher{}
	var stdout, stderr bytes.Buffer
	exit := runFactoryVerifyDupcodeBaseline(
		[]string{}, &stdout, &stderr, d.dispatch,
	)
	if exit != 0 {
		t.Errorf("exit = %d, want 0", exit)
	}
	if got := d.calls.Load(); got != 1 {
		t.Errorf("dispatch calls = %d, want 1", got)
	}
	if !strings.Contains(stdout.String(), "dupcode baseline: OK") {
		t.Errorf("stdout = %q, want to contain 'dupcode baseline: OK'", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Errorf("stderr = %q, want empty on success", stderr.String())
	}
}

// TestDupcodeBaselineHumanFailureWriterIsolated proves a human-mode
// failure renders the FAILED line plus the exact findings on the
// supplied stdout, and returns exit 1. The supplied findings carry a
// normal verification-failure kind (not a dispatcher authority-denial
// kind) so the rendering path treats them as a normal failure.
func TestDupcodeBaselineHumanFailureWriterIsolated(t *testing.T) {
	d := &baselineWriterTestDispatcher{out: gate.DupcodeBaselineOutcome{
		Dispatch: verifierdispatch.Result{
			Findings: []checks.Finding{{
				Path:    "dupcode-baseline",
				Kind:    "baseline_threshold_mismatch",
				Message: "min_lines mismatch: got 10, want 40",
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
	if !strings.Contains(stdout.String(), "FAILED") {
		t.Errorf("stdout = %q, want to contain 'FAILED'", stdout.String())
	}
	if !strings.Contains(stdout.String(), "baseline_threshold_mismatch") {
		t.Errorf("stdout = %q, want to contain 'baseline_threshold_mismatch'", stdout.String())
	}
	if !strings.Contains(stdout.String(), "min_lines mismatch: got 10, want 40") {
		t.Errorf("stdout = %q, want to contain 'min_lines mismatch: got 10, want 40'", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Errorf("stderr = %q, want empty on human failure rendering", stderr.String())
	}
}

// TestDupcodeBaselineHelpRendersAllFlags proves --help emits every
// declared flag to supplied stderr while keeping stdout empty.
func TestDupcodeBaselineHelpRendersAllFlags(t *testing.T) {
	d := &baselineWriterTestDispatcher{}
	var stdout, stderr bytes.Buffer
	exit := runFactoryVerifyDupcodeBaseline(
		[]string{"--help"}, &stdout, &stderr, d.dispatch,
	)
	if exit != 0 {
		t.Errorf("exit = %d, want 0", exit)
	}
	if got := d.calls.Load(); got != 0 {
		t.Errorf("dispatch calls = %d, want 0 on help", got)
	}
	if stdout.Len() != 0 {
		t.Errorf("stdout = %q, want empty on help", stdout.String())
	}
	wantFlags := []string{"-baseline", "-json", "-min-lines", "-min-tokens"}
	for _, want := range wantFlags {
		if !strings.Contains(stderr.String(), want) {
			t.Errorf("stderr = %q, want to contain %q", stderr.String(), want)
		}
	}
	if !strings.Contains(stderr.String(), "Usage:") {
		t.Errorf("stderr = %q, want to contain 'Usage:'", stderr.String())
	}
}

// baselineFailingWriter is an io.Writer that always returns an error. It is
// used to drive JSON write-failure paths deterministically.
type baselineFailingWriter struct {
	calls int
}

var baselineWriterErr = errors.New("always-fail-writer")

func (f *baselineFailingWriter) Write(p []byte) (int, error) {
	f.calls++
	return 0, baselineWriterErr
}

// TestDupcodeBaselineJSONWriteFailureExitsTwo proves a JSON encoding
// failure exits 2 with a diagnostic on supplied stderr and emits no
// success classification.
func TestDupcodeBaselineJSONWriteFailureExitsTwo(t *testing.T) {
	d := &baselineWriterTestDispatcher{}
	w := &baselineFailingWriter{}
	stderr := &bytes.Buffer{}
	exit := runFactoryVerifyDupcodeBaseline(
		[]string{"--json"}, w, stderr, d.dispatch,
	)
	if exit != 2 {
		t.Errorf("exit = %d, want 2 on JSON write failure", exit)
	}
	if w.calls == 0 {
		t.Error("failing writer was never called")
	}
	if !strings.Contains(stderr.String(), "json write failure") {
		t.Errorf("stderr = %q, want to contain 'json write failure'", stderr.String())
	}
}

// TestDupcodeBaselineJSONFailureChannelWriteFailureExitsTwo proves that
// a JSON failure-channel rendering write failure also exits 2.
func TestDupcodeBaselineJSONFailureChannelWriteFailureExitsTwo(t *testing.T) {
	d := &baselineWriterTestDispatcher{out: gate.DupcodeBaselineOutcome{
		Dispatch: verifierdispatch.Result{
			Error: errors.New("sentinel dispatcher error"),
		},
	}}
	w := &baselineFailingWriter{}
	stderr := &bytes.Buffer{}
	exit := runFactoryVerifyDupcodeBaseline(
		[]string{"--json"}, w, stderr, d.dispatch,
	)
	if exit != 2 {
		t.Errorf("exit = %d, want 2 on failure-channel JSON write failure", exit)
	}
	if w.calls == 0 {
		t.Error("failing writer was never called")
	}
}

// TestDupcodeBaselineParseFailureJSONWriteFailureExitsTwo proves that a
// parse failure in JSON mode with a failing writer also returns 2.
func TestDupcodeBaselineParseFailureJSONWriteFailureExitsTwo(t *testing.T) {
	d := &baselineWriterTestDispatcher{}
	w := &baselineFailingWriter{}
	stderr := &bytes.Buffer{}
	exit := runFactoryVerifyDupcodeBaseline(
		[]string{"--unknown-option", "--json"}, w, stderr, d.dispatch,
	)
	if exit != 2 {
		t.Errorf("exit = %d, want 2 on parse-failure JSON write failure", exit)
	}
}

// TestDupcodeBaselineJSONSuccessRoundtrip proves a successful JSON
// payload can be unmarshalled into a typed result. This guards against
// regressions where the field set drifts.
func TestDupcodeBaselineJSONSuccessRoundtrip(t *testing.T) {
	d := &baselineWriterTestDispatcher{}
	var stdout, stderr bytes.Buffer
	exit := runFactoryVerifyDupcodeBaseline(
		[]string{"--json"}, &stdout, &stderr, d.dispatch,
	)
	if exit != 0 {
		t.Fatalf("exit = %d, want 0", exit)
	}
	var decoded map[string]interface{}
	if err := json.Unmarshal(stdout.Bytes(), &decoded); err != nil {
		t.Fatalf("stdout JSON decode: %v; stdout=%q", err, stdout.String())
	}
	if got, want := decoded["status"], "ok"; got != want {
		t.Errorf("status = %v, want %v", got, want)
	}
	if got, want := decoded["baseline"], ".factory/dupcode-baseline.json"; got != want {
		t.Errorf("baseline = %v, want %v", got, want)
	}
}
