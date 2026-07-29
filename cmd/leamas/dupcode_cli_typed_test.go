// SPDX-License-Identifier: Apache-2.0

package main

import (
	"bytes"
	"context"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/s1onique/leamas/internal/factory/gate"
)

// countingDupcodeDispatchers records the number of typed dispatcher calls
// along with the most-recent spec captured for each lane. It returns a
// fully-controlled DupcodeVerifyOutcome / DupcodeUpdateBaselineOutcome so
// the rendering layer's failure-channel handling can be observed directly.
type countingDupcodeDispatchers struct {
	verifyCalls       atomic.Int64
	updateCalls       atomic.Int64
	lastVerifySpec    atomic.Value // gate.DupcodeVerifySpec
	lastUpdateSpec    atomic.Value // gate.DupcodeUpdateBaselineSpec
	lastVerifyRoot    atomic.Value // string
	lastUpdateRoot    atomic.Value // string
	verifyResult      gate.DupcodeVerifyOutcome
	updateResult      gate.DupcodeUpdateBaselineOutcome
	verifyResultIsSet atomic.Bool
	updateResultIsSet atomic.Bool
}

func (c *countingDupcodeDispatchers) setVerify(spec gate.DupcodeVerifySpec) {
	c.lastVerifySpec.Store(spec)
}

func (c *countingDupcodeDispatchers) setUpdate(spec gate.DupcodeUpdateBaselineSpec) {
	c.lastUpdateSpec.Store(spec)
}

func (c *countingDupcodeDispatchers) verify(ctx context.Context, root string, spec gate.DupcodeVerifySpec) gate.DupcodeVerifyOutcome {
	c.verifyCalls.Add(1)
	c.lastVerifySpec.Store(spec)
	c.lastVerifyRoot.Store(root)
	if c.verifyResultIsSet.Load() {
		return c.verifyResult
	}
	return gate.DupcodeVerifyOutcome{}
}

func (c *countingDupcodeDispatchers) updateBaseline(ctx context.Context, root string, spec gate.DupcodeUpdateBaselineSpec) gate.DupcodeUpdateBaselineOutcome {
	c.updateCalls.Add(1)
	c.lastUpdateSpec.Store(spec)
	c.lastUpdateRoot.Store(root)
	if c.updateResultIsSet.Load() {
		return c.updateResult
	}
	return gate.DupcodeUpdateBaselineOutcome{}
}

func (c *countingDupcodeDispatchers) dispatchers() dupcodeTypedDispatchers {
	return dupcodeTypedDispatchers{
		verify:         c.verify,
		updateBaseline: c.updateBaseline,
	}
}

func newCountingDispatchers() *countingDupcodeDispatchers {
	return &countingDupcodeDispatchers{}
}

// TestDupcodeTypedRoutingNoOptions proves the no-options default verify
// dispatch: exactly one verify call, zero update calls, root ".",
// and the captured spec uses the default thresholds.
func TestDupcodeTypedRoutingNoOptions(t *testing.T) {
	d := newCountingDispatchers()
	var stdout, stderr bytes.Buffer

	exitCode := handleDupcodeWith(nil, &stdout, &stderr, d.dispatchers())

	if exitCode != ExitSuccess {
		t.Errorf("exitCode = %d, want %d (ExitSuccess)", exitCode, ExitSuccess)
	}
	if got := d.verifyCalls.Load(); got != 1 {
		t.Errorf("verifyCalls = %d, want 1", got)
	}
	if got := d.updateCalls.Load(); got != 0 {
		t.Errorf("updateCalls = %d, want 0", got)
	}
	if got, _ := d.lastVerifyRoot.Load().(string); got != "." {
		t.Errorf("lastVerifyRoot = %q, want %q", got, ".")
	}
	spec, ok := d.lastVerifySpec.Load().(gate.DupcodeVerifySpec)
	if !ok {
		t.Fatalf("lastVerifySpec missing")
	}
	if spec.BaselinePath != ".factory/dupcode-baseline.json" {
		t.Errorf("BaselinePath = %q, want %q", spec.BaselinePath, ".factory/dupcode-baseline.json")
	}
	if spec.MinLines != 40 {
		t.Errorf("MinLines = %d, want 40", spec.MinLines)
	}
	if spec.MinTokens != 400 {
		t.Errorf("MinTokens = %d, want 400", spec.MinTokens)
	}
}

// TestDupcodeTypedRoutingCustomOptions proves the parser threads custom
// flag values into the captured verify spec unchanged.
func TestDupcodeTypedRoutingCustomOptions(t *testing.T) {
	d := newCountingDispatchers()
	var stdout, stderr bytes.Buffer

	args := []string{
		"-baseline", "custom.json",
		"-min-lines", "52",
		"-min-tokens", "630",
		"-json",
	}
	exitCode := handleDupcodeWith(args, &stdout, &stderr, d.dispatchers())

	if exitCode != ExitSuccess {
		t.Errorf("exitCode = %d, want %d (ExitSuccess)", exitCode, ExitSuccess)
	}
	if got := d.verifyCalls.Load(); got != 1 {
		t.Errorf("verifyCalls = %d, want 1", got)
	}
	if got := d.updateCalls.Load(); got != 0 {
		t.Errorf("updateCalls = %d, want 0", got)
	}
	spec, ok := d.lastVerifySpec.Load().(gate.DupcodeVerifySpec)
	if !ok {
		t.Fatalf("lastVerifySpec missing")
	}
	if spec.BaselinePath != "custom.json" {
		t.Errorf("BaselinePath = %q, want %q", spec.BaselinePath, "custom.json")
	}
	if spec.MinLines != 52 {
		t.Errorf("MinLines = %d, want 52", spec.MinLines)
	}
	if spec.MinTokens != 630 {
		t.Errorf("MinTokens = %d, want 630", spec.MinTokens)
	}
}

// TestDupcodeTypedRoutingUpdate verifies the --update-baseline flag routes
// to the update dispatcher with the exact spec carried through.
func TestDupcodeTypedRoutingUpdate(t *testing.T) {
	d := newCountingDispatchers()
	var stdout, stderr bytes.Buffer

	args := []string{
		"--update-baseline",
		"--baseline", "custom.json",
		"--min-lines", "53",
		"--min-tokens", "631",
	}
	exitCode := handleDupcodeWith(args, &stdout, &stderr, d.dispatchers())

	if exitCode != ExitSuccess {
		t.Errorf("exitCode = %d, want %d (ExitSuccess)", exitCode, ExitSuccess)
	}
	if got := d.verifyCalls.Load(); got != 0 {
		t.Errorf("verifyCalls = %d, want 0", got)
	}
	if got := d.updateCalls.Load(); got != 1 {
		t.Errorf("updateCalls = %d, want 1", got)
	}
	spec, ok := d.lastUpdateSpec.Load().(gate.DupcodeUpdateBaselineSpec)
	if !ok {
		t.Fatalf("lastUpdateSpec missing")
	}
	if spec.BaselinePath != "custom.json" {
		t.Errorf("BaselinePath = %q, want %q", spec.BaselinePath, "custom.json")
	}
	if spec.MinLines != 53 {
		t.Errorf("MinLines = %d, want 53", spec.MinLines)
	}
	if spec.MinTokens != 631 {
		t.Errorf("MinTokens = %d, want 631", spec.MinTokens)
	}
	if got, _ := d.lastUpdateRoot.Load().(string); got != "." {
		t.Errorf("lastUpdateRoot = %q, want %q", got, ".")
	}
}

// TestDupcodeTypedRoutingHelp proves --help renders usage to stderr, exits 0
// and dispatches zero times.
func TestDupcodeTypedRoutingHelp(t *testing.T) {
	d := newCountingDispatchers()
	var stdout, stderr bytes.Buffer

	exitCode := handleDupcodeWith([]string{"--help"}, &stdout, &stderr, d.dispatchers())

	if exitCode != ExitSuccess {
		t.Errorf("exitCode = %d, want %d (ExitSuccess)", exitCode, ExitSuccess)
	}
	if got := d.verifyCalls.Load(); got != 0 {
		t.Errorf("verifyCalls = %d, want 0", got)
	}
	if got := d.updateCalls.Load(); got != 0 {
		t.Errorf("updateCalls = %d, want 0", got)
	}
	if !strings.Contains(stderr.String(), "Usage:") {
		t.Errorf("stderr = %q, want to contain 'Usage:'", stderr.String())
	}
	if stdout.Len() != 0 {
		t.Errorf("stdout = %q, want empty", stdout.String())
	}
}

// TestDupcodeTypedRoutingUnknownFlag proves an unknown flag exits with
// ExitParseFailure and never invokes a dispatcher.
func TestDupcodeTypedRoutingUnknownFlag(t *testing.T) {
	d := newCountingDispatchers()
	var stdout, stderr bytes.Buffer

	exitCode := handleDupcodeWith([]string{"--unknown-option"}, &stdout, &stderr, d.dispatchers())

	if exitCode != ExitParseFailure {
		t.Errorf("exitCode = %d, want %d (ExitParseFailure)", exitCode, ExitParseFailure)
	}
	if got := d.verifyCalls.Load(); got != 0 {
		t.Errorf("verifyCalls = %d, want 0", got)
	}
	if got := d.updateCalls.Load(); got != 0 {
		t.Errorf("updateCalls = %d, want 0", got)
	}
}

// TestDupcodeTypedRoutingPositionalArgument proves unexpected positional
// arguments exit with ExitParseFailure and never invoke a dispatcher.
func TestDupcodeTypedRoutingPositionalArgument(t *testing.T) {
	d := newCountingDispatchers()
	var stdout, stderr bytes.Buffer

	exitCode := handleDupcodeWith([]string{"unexpected-argument"}, &stdout, &stderr, d.dispatchers())

	if exitCode != ExitParseFailure {
		t.Errorf("exitCode = %d, want %d (ExitParseFailure)", exitCode, ExitParseFailure)
	}
	if got := d.verifyCalls.Load(); got != 0 {
		t.Errorf("verifyCalls = %d, want 0", got)
	}
	if got := d.updateCalls.Load(); got != 0 {
		t.Errorf("updateCalls = %d, want 0", got)
	}
}
