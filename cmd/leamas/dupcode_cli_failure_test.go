// SPDX-License-Identifier: Apache-2.0

package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/s1onique/leamas/internal/factory/checks"
	"github.com/s1onique/leamas/internal/factory/gate"
	"github.com/s1onique/leamas/internal/factory/verifierdispatch"
)

// TestDupcodeFailureChannelFindingHuman proves that a Dispatch.Findings
// outcome propagates as a stderr message in human mode and never reaches
// the typed-payload rendering path.
func TestDupcodeFailureChannelFindingHuman(t *testing.T) {
	d := newCountingDispatchers()
	d.verifyResult = gate.DupcodeVerifyOutcome{
		Dispatch: verifierdispatch.Result{
			Findings: []checks.Finding{{
				Kind:    "sentinel-authority-kind",
				Message: "sentinel authority denial",
			}},
		},
	}
	d.verifyResultIsSet.Store(true)

	var stdout, stderr bytes.Buffer
	exitCode := handleDupcodeWith(nil, &stdout, &stderr, d.dispatchers())

	if exitCode != ExitAuthorityFailure {
		t.Errorf("exitCode = %d, want %d (ExitAuthorityFailure)", exitCode, ExitAuthorityFailure)
	}
	if got := d.verifyCalls.Load(); got != 1 {
		t.Errorf("verifyCalls = %d, want 1", got)
	}
	if !strings.Contains(stderr.String(), "sentinel authority denial") {
		t.Errorf("stderr = %q, want to contain 'sentinel authority denial'", stderr.String())
	}
	if strings.Contains(stdout.String(), "No duplicate code violations found") {
		t.Errorf("stdout must not render success payload: %q", stdout.String())
	}
}

// TestDupcodeFailureChannelFindingJSON proves that a Dispatch.Findings
// outcome in JSON mode is encoded as {"error":..., "kind":...} with empty
// stderr.
func TestDupcodeFailureChannelFindingJSON(t *testing.T) {
	d := newCountingDispatchers()
	d.verifyResult = gate.DupcodeVerifyOutcome{
		Dispatch: verifierdispatch.Result{
			Findings: []checks.Finding{{
				Kind:    "sentinel-authority-kind",
				Message: "sentinel authority denial",
			}},
		},
	}
	d.verifyResultIsSet.Store(true)

	var stdout, stderr bytes.Buffer
	exitCode := handleDupcodeWith([]string{"--json"}, &stdout, &stderr, d.dispatchers())

	if exitCode != ExitAuthorityFailure {
		t.Errorf("exitCode = %d, want %d (ExitAuthorityFailure)", exitCode, ExitAuthorityFailure)
	}

	var decoded map[string]interface{}
	if err := json.Unmarshal(stdout.Bytes(), &decoded); err != nil {
		t.Fatalf("stdout JSON decode: %v; stdout=%q", err, stdout.String())
	}
	if got, want := decoded["error"], "sentinel authority denial"; got != want {
		t.Errorf("error = %v, want %v", got, want)
	}
	if got, want := decoded["kind"], "sentinel-authority-kind"; got != want {
		t.Errorf("kind = %v, want %v", got, want)
	}
	if stderr.Len() > 0 {
		t.Errorf("stderr = %q, want empty for JSON failure channel", stderr.String())
	}
}

// TestDupcodeFailureChannelErrorHuman proves that a Dispatch.Error outcome
// propagates as a stderr message in human mode.
func TestDupcodeFailureChannelErrorHuman(t *testing.T) {
	d := newCountingDispatchers()
	d.verifyResult = gate.DupcodeVerifyOutcome{
		Dispatch: verifierdispatch.Result{
			Error: errors.New("sentinel dispatcher error"),
		},
	}
	d.verifyResultIsSet.Store(true)

	var stdout, stderr bytes.Buffer
	exitCode := handleDupcodeWith(nil, &stdout, &stderr, d.dispatchers())

	if exitCode != ExitAuthorityFailure {
		t.Errorf("exitCode = %d, want %d (ExitAuthorityFailure)", exitCode, ExitAuthorityFailure)
	}
	if !strings.Contains(stderr.String(), "sentinel dispatcher error") {
		t.Errorf("stderr = %q, want to contain 'sentinel dispatcher error'", stderr.String())
	}
}

// TestDupcodeFailureChannelErrorJSON proves that a Dispatch.Error outcome in
// JSON mode emits {"error":...} with empty stderr.
func TestDupcodeFailureChannelErrorJSON(t *testing.T) {
	d := newCountingDispatchers()
	d.verifyResult = gate.DupcodeVerifyOutcome{
		Dispatch: verifierdispatch.Result{
			Error: errors.New("sentinel dispatcher error"),
		},
	}
	d.verifyResultIsSet.Store(true)

	var stdout, stderr bytes.Buffer
	exitCode := handleDupcodeWith([]string{"--json"}, &stdout, &stderr, d.dispatchers())

	if exitCode != ExitAuthorityFailure {
		t.Errorf("exitCode = %d, want %d (ExitAuthorityFailure)", exitCode, ExitAuthorityFailure)
	}

	var decoded map[string]interface{}
	if err := json.Unmarshal(stdout.Bytes(), &decoded); err != nil {
		t.Fatalf("stdout JSON decode: %v; stdout=%q", err, stdout.String())
	}
	if got, ok := decoded["error"].(string); !ok || !strings.Contains(got, "sentinel dispatcher error") {
		t.Errorf("error = %v, want substring 'sentinel dispatcher error'", decoded["error"])
	}
	if stderr.Len() > 0 {
		t.Errorf("stderr = %q, want empty for JSON failure channel", stderr.String())
	}
}

// TestDupcodeFailureChannelUpdateFindingHuman proves the failure channel
// propagates from the update dispatcher in human mode.
func TestDupcodeFailureChannelUpdateFindingHuman(t *testing.T) {
	d := newCountingDispatchers()
	d.updateResult = gate.DupcodeUpdateBaselineOutcome{
		Dispatch: verifierdispatch.Result{
			Findings: []checks.Finding{{
				Kind:    "sentinel-authority-kind",
				Message: "sentinel update authority denial",
			}},
		},
	}
	d.updateResultIsSet.Store(true)

	var stdout, stderr bytes.Buffer
	exitCode := handleDupcodeWith([]string{"--update-baseline"}, &stdout, &stderr, d.dispatchers())

	if exitCode != ExitAuthorityFailure {
		t.Errorf("exitCode = %d, want %d (ExitAuthorityFailure)", exitCode, ExitAuthorityFailure)
	}
	if got := d.updateCalls.Load(); got != 1 {
		t.Errorf("updateCalls = %d, want 1", got)
	}
	if !strings.Contains(stderr.String(), "sentinel update authority denial") {
		t.Errorf("stderr = %q, want to contain 'sentinel update authority denial'", stderr.String())
	}
}

// TestDupcodeFailureChannelUpdateFindingJSON proves the failure channel
// propagates from the update dispatcher in JSON mode.
func TestDupcodeFailureChannelUpdateFindingJSON(t *testing.T) {
	d := newCountingDispatchers()
	d.updateResult = gate.DupcodeUpdateBaselineOutcome{
		Dispatch: verifierdispatch.Result{
			Findings: []checks.Finding{{
				Kind:    "sentinel-authority-kind",
				Message: "sentinel update authority denial",
			}},
		},
	}
	d.updateResultIsSet.Store(true)

	var stdout, stderr bytes.Buffer
	exitCode := handleDupcodeWith([]string{"--update-baseline", "--json"}, &stdout, &stderr, d.dispatchers())

	if exitCode != ExitAuthorityFailure {
		t.Errorf("exitCode = %d, want %d (ExitAuthorityFailure)", exitCode, ExitAuthorityFailure)
	}

	var decoded map[string]interface{}
	if err := json.Unmarshal(stdout.Bytes(), &decoded); err != nil {
		t.Fatalf("stdout JSON decode: %v; stdout=%q", err, stdout.String())
	}
	if got, want := decoded["error"], "sentinel update authority denial"; got != want {
		t.Errorf("error = %v, want %v", got, want)
	}
	if got, want := decoded["kind"], "sentinel-authority-kind"; got != want {
		t.Errorf("kind = %v, want %v", got, want)
	}
	if stderr.Len() > 0 {
		t.Errorf("stderr = %q, want empty for JSON failure channel", stderr.String())
	}
}

// TestDupcodeOutputCompatibilityNoChanges proves the human-mode "no
// violations" path renders exactly the canonical message.
func TestDupcodeOutputCompatibilityNoChanges(t *testing.T) {
	d := newCountingDispatchers()
	var stdout, stderr bytes.Buffer
	exitCode := handleDupcodeWith(nil, &stdout, &stderr, d.dispatchers())
	if exitCode != ExitSuccess {
		t.Errorf("exitCode = %d, want ExitSuccess", exitCode)
	}
	if !strings.Contains(stdout.String(), "No duplicate code violations found.") {
		t.Errorf("stdout = %q, want to contain canonical success message", stdout.String())
	}
}
