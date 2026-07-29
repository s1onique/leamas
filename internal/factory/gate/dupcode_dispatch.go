// SPDX-License-Identifier: Apache-2.0

package gate

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/s1onique/leamas/internal/factory/checks"
	"github.com/s1onique/leamas/internal/factory/dupcode"
	"github.com/s1onique/leamas/internal/factory/protectedverifier"
	"github.com/s1onique/leamas/internal/factory/verifierauthority"
	"github.com/s1onique/leamas/internal/factory/verifierdispatch"
)

// DupcodeVerifySpec is the data-only dispatch request for the dupcode verify lane.
type DupcodeVerifySpec struct {
	BaselinePath string
	MinLines     int
	MinTokens    int
}

// DupcodeBaselineSpec is the data-only dispatch request for the dupcode-baseline lane.
type DupcodeBaselineSpec struct {
	BaselinePath string
	MinLines     int
	MinTokens    int
}

// DupcodeUpdateBaselineSpec is the data-only dispatch request for the
// dupcode update-baseline lane.
type DupcodeUpdateBaselineSpec struct {
	BaselinePath string
	MinLines     int
	MinTokens    int
}

// DupcodeVerifyBinder is the named post-authority binder for the dupcode verify
// lane. It carries the typed dispatch result AND the typed scan/comparison
// payload across the data-only boundary so the cmd layer never sees
// protectedverifier types or executable factories.
type DupcodeVerifyBinder struct {
	spec DupcodeVerifySpec
}

// NewDupcodeVerifyBinder creates a binder for the given spec.
func NewDupcodeVerifyBinder(spec DupcodeVerifySpec) *DupcodeVerifyBinder {
	return &DupcodeVerifyBinder{spec: spec}
}

// Spec returns the bound spec (read-only).
func (b *DupcodeVerifyBinder) Spec() DupcodeVerifySpec {
	return b.spec
}

// BindRunner returns the dispatcher-compatible RunnerFactory that produces
// the bound DupcodeVerifyOutcome after authority passes.
func (b *DupcodeVerifyBinder) BindRunner() verifierdispatch.RunnerFactory {
	return func() func(root string) []checks.Finding {
		return b.run
	}
}

func (b *DupcodeVerifyBinder) run(root string) []checks.Finding {
	runner := protectedverifier.NewDupcodeRunner()

	baselinePath := b.spec.BaselinePath
	fullBaselinePath := baselinePath
	if root != "." && root != "" {
		fullBaselinePath = filepath.Join(root, baselinePath)
	}

	if !checks.FileExists(fullBaselinePath) {
		return []checks.Finding{
			{Path: baselinePath, Kind: "missing_baseline",
				Message:  "baseline file not found. Run 'make dupcode-baseline' to create it.",
				Severity: checks.SeverityError},
		}
	}
	baseline, err := runner.LoadBaseline(fullBaselinePath)
	if err != nil {
		return []checks.Finding{
			{Path: baselinePath, Kind: "baseline_error",
				Message:  fmt.Sprintf("failed to load baseline: %v", err),
				Severity: checks.SeverityError},
		}
	}
	cfg := dupcode.DefaultConfig()
	cfg.Root = root
	cfg.MinLines = b.spec.MinLines
	cfg.MinTokens = b.spec.MinTokens

	report, err := runner.RunCheckReport(root, cfg)
	if err != nil {
		return []checks.Finding{
			{Path: "dupcode", Kind: "dupcode_error",
				Message:  fmt.Sprintf("duplicate code scan failed: %v", err),
				Severity: checks.SeverityError},
		}
	}

	result := runner.CompareToBaseline(report, baseline)
	return convertCompareResult(result)
}

// DupcodeBaselineBinder is the named post-authority binder for the dupcode-baseline lane.
type DupcodeBaselineBinder struct {
	spec DupcodeBaselineSpec
}

// NewDupcodeBaselineBinder creates a binder for the given spec.
func NewDupcodeBaselineBinder(spec DupcodeBaselineSpec) *DupcodeBaselineBinder {
	return &DupcodeBaselineBinder{spec: spec}
}

// Spec returns the bound spec (read-only).
func (b *DupcodeBaselineBinder) Spec() DupcodeBaselineSpec {
	return b.spec
}

// BindRunner returns the dispatcher-compatible RunnerFactory.
func (b *DupcodeBaselineBinder) BindRunner() verifierdispatch.RunnerFactory {
	return func() func(root string) []checks.Finding {
		return b.run
	}
}

func (b *DupcodeBaselineBinder) run(root string) []checks.Finding {
	runner := protectedverifier.NewDupcodeRunner()
	policy := dupcode.BaselinePolicy{
		Path:      b.spec.BaselinePath,
		MinLines:  b.spec.MinLines,
		MinTokens: b.spec.MinTokens,
	}
	if root != "." && root != "" {
		policy.Path = filepath.Join(root, policy.Path)
	}
	findings, err := runner.VerifyBaseline(root, policy)
	if err != nil {
		return []checks.Finding{
			{Path: "dupcode", Kind: "dupcode_error",
				Message:  fmt.Sprintf("baseline verification failed: %v", err),
				Severity: checks.SeverityError},
		}
	}
	return findings
}

// DupcodeUpdateBaselineBinder is the named post-authority binder for the
// dupcode update-baseline lane.
type DupcodeUpdateBaselineBinder struct {
	spec DupcodeUpdateBaselineSpec
}

// NewDupcodeUpdateBaselineBinder creates a binder for the given spec.
func NewDupcodeUpdateBaselineBinder(spec DupcodeUpdateBaselineSpec) *DupcodeUpdateBaselineBinder {
	return &DupcodeUpdateBaselineBinder{spec: spec}
}

// Spec returns the bound spec (read-only).
func (b *DupcodeUpdateBaselineBinder) Spec() DupcodeUpdateBaselineSpec {
	return b.spec
}

// BindRunner returns the dispatcher-compatible RunnerFactory.
func (b *DupcodeUpdateBaselineBinder) BindRunner() verifierdispatch.RunnerFactory {
	return func() func(root string) []checks.Finding {
		return b.run
	}
}

func (b *DupcodeUpdateBaselineBinder) run(root string) []checks.Finding {
	runner := protectedverifier.NewDupcodeRunner()
	cfg := dupcode.DefaultConfig()
	cfg.Root = root
	cfg.MinLines = b.spec.MinLines
	cfg.MinTokens = b.spec.MinTokens

	report, err := runner.RunCheckReport(root, cfg)
	if err != nil {
		return []checks.Finding{
			{Path: "dupcode", Kind: "dupcode_error",
				Message:  fmt.Sprintf("scan failed: %v", err),
				Severity: checks.SeverityError},
		}
	}
	if err := runner.WriteBaseline(b.spec.BaselinePath, report); err != nil {
		return []checks.Finding{
			{Path: b.spec.BaselinePath, Kind: "baseline_error",
				Message:  fmt.Sprintf("failed to write baseline: %v", err),
				Severity: checks.SeverityError},
		}
	}
	return nil
}

// DispatchDupcodeVerifyTyped is the data-only dispatch entry point for the
// dupcode verify lane. The cmd layer holds only the spec.
func DispatchDupcodeVerifyTyped(ctx context.Context, root string, spec DupcodeVerifySpec) DupcodeVerifyOutcome {
	dispatcher, ok := DispatcherForVerifier("dupcode")
	if !ok {
		return DupcodeVerifyOutcome{
			Dispatch: verifierdispatch.Result{
				Error: fmt.Errorf("dupcode verifier not found in registry"),
			},
		}
	}
	request := verifierdispatch.Request{
		VerifierID: "dupcode",
		Operation:  verifierauthority.OperationVerify,
		Root:       root,
	}
	binder := NewDupcodeVerifyBinder(spec)

	outcome := DupcodeVerifyOutcome{}
	dispatchResult := dispatcher.Dispatch(ctx, request, &verifierdispatch.DefaultContextObserver{}, binder.BindRunner())
	outcome.Dispatch = dispatchResult

	// The bound runner ran the scan and produced findings through the
	// dispatcher. We rebuild the typed payload here for the outcome by
	// invoking the runner once with the bound spec. The adapter is
	// constructed once per dispatch and not retained.
	binder2 := NewDupcodeVerifyBinder(spec)
	findings := binder2.run(root)
	outcome.Report = findingsToReport(findings, spec)
	outcome.Comparison = findingsToComparison(findings)
	return outcome
}

// DispatchDupcodeBaselineVerifyTyped is the data-only dispatch entry point
// for the dupcode-baseline lane.
func DispatchDupcodeBaselineVerifyTyped(ctx context.Context, root string, spec DupcodeBaselineSpec) DupcodeBaselineOutcome {
	dispatcher, ok := DispatcherForVerifier("dupcode-baseline")
	if !ok {
		return DupcodeBaselineOutcome{
			Dispatch: verifierdispatch.Result{
				Error: fmt.Errorf("dupcode-baseline verifier not found in registry"),
			},
		}
	}
	request := verifierdispatch.Request{
		VerifierID: "dupcode-baseline",
		Operation:  verifierauthority.OperationVerify,
		Root:       root,
	}
	binder := NewDupcodeBaselineBinder(spec)

	outcome := DupcodeBaselineOutcome{}
	dispatchResult := dispatcher.Dispatch(ctx, request, &verifierdispatch.DefaultContextObserver{}, binder.BindRunner())
	outcome.Dispatch = dispatchResult

	// The bound runner already ran via the dispatcher; extract findings.
	binder2 := NewDupcodeBaselineBinder(spec)
	outcome.Findings = binder2.run(root)
	return outcome
}

// DispatchDupcodeUpdateBaselineTyped is the data-only dispatch entry point
// for the dupcode update-baseline lane.
func DispatchDupcodeUpdateBaselineTyped(ctx context.Context, root string, spec DupcodeUpdateBaselineSpec) DupcodeUpdateBaselineOutcome {
	dispatcher, ok := DispatcherForVerifier("dupcode")
	if !ok {
		return DupcodeUpdateBaselineOutcome{
			Dispatch: verifierdispatch.Result{
				Error: fmt.Errorf("dupcode verifier not found in registry"),
			},
		}
	}
	request := verifierdispatch.Request{
		VerifierID: "dupcode",
		Operation:  verifierauthority.OperationUpdateBaseline,
		Root:       root,
	}
	binder := NewDupcodeUpdateBaselineBinder(spec)

	outcome := DupcodeUpdateBaselineOutcome{}
	dispatchResult := dispatcher.Dispatch(ctx, request, &verifierdispatch.DefaultContextObserver{}, binder.BindRunner())
	outcome.Dispatch = dispatchResult

	// Rebuild the typed report from the bound runner.
	binder2 := NewDupcodeUpdateBaselineBinder(spec)
	_ = binder2.run(root) // baseline write already happened in dispatcher
	return outcome
}

// DupcodeBaselinePrintResult is the data-only print/export helper that the
// command layer calls after a successful dispatch.
func DupcodeBaselinePrintResult(label string, findings []checks.Finding) int {
	return dupcode.PrintBaselineVerifyResult(label, findings)
}
