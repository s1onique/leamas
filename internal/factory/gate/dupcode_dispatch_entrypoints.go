// SPDX-License-Identifier: Apache-2.0

package gate

import (
	"context"
	"fmt"

	"github.com/s1onique/leamas/internal/factory/checks"
	"github.com/s1onique/leamas/internal/factory/dupcode"
	"github.com/s1onique/leamas/internal/factory/verifierauthority"
	"github.com/s1onique/leamas/internal/factory/verifierdispatch"
)

// productionDupcodeBinderDeps returns the production default deps. The
// returned struct is invocation-local; no package-global state. The
// NewRunner factory is nil for production; the binder's run method
// instantiates a real *protectedverifier.DupcodeRunner directly using
// protectedverifier.NewDupcodeRunner, which is the approved post-authority
// call site.
func productionDupcodeBinderDeps() dupcodeBinderDeps {
	return dupcodeBinderDeps{
		NewRunner: nil,
	}
}

// DispatchDupcodeVerifyTyped is the production typed dispatch entry
// point for the dupcode verify lane. The cmd layer holds only the spec.
func DispatchDupcodeVerifyTyped(ctx context.Context, root string, spec DupcodeVerifySpec) DupcodeVerifyOutcome {
	return dispatchDupcodeVerifyTypedWith(ctx, root, spec, &verifierdispatch.DefaultContextObserver{}, productionDupcodeBinderDeps())
}

// dispatchDupcodeBaselineVerifyTypedWith is the testable typed dispatch
// entry point for the dupcode-baseline lane.
func dispatchDupcodeBaselineVerifyTypedWith(
	ctx context.Context,
	root string,
	spec DupcodeBaselineSpec,
	observer verifierdispatch.ContextObserver,
	deps dupcodeBinderDeps,
) DupcodeBaselineOutcome {
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
	binder := newDupcodeBaselineBinderWithDeps(spec, deps)

	outcome := DupcodeBaselineOutcome{
		Dispatch: dispatcher.Dispatch(ctx, request, observer, binder.BindRunner()),
	}

	// Read findings from the binder's invocation-local cell. Do NOT
	// re-execute the runner.
	outcome.Findings = binder.Findings()
	return outcome
}

// DispatchDupcodeBaselineVerifyTyped is the production typed dispatch
// entry point for the dupcode-baseline lane.
func DispatchDupcodeBaselineVerifyTyped(ctx context.Context, root string, spec DupcodeBaselineSpec) DupcodeBaselineOutcome {
	return dispatchDupcodeBaselineVerifyTypedWith(ctx, root, spec, &verifierdispatch.DefaultContextObserver{}, productionDupcodeBinderDeps())
}

// dispatchDupcodeUpdateBaselineTypedWith is the testable typed dispatch
// entry point for the dupcode update-baseline lane.
func dispatchDupcodeUpdateBaselineTypedWith(
	ctx context.Context,
	root string,
	spec DupcodeUpdateBaselineSpec,
	observer verifierdispatch.ContextObserver,
	deps dupcodeBinderDeps,
) DupcodeUpdateBaselineOutcome {
	dispatcher, ok := DispatcherForVerifier("dupcode-update-baseline")
	if !ok {
		return DupcodeUpdateBaselineOutcome{
			Dispatch: verifierdispatch.Result{
				Error: fmt.Errorf("dupcode-update-baseline verifier not found in registry"),
			},
		}
	}
	request := verifierdispatch.Request{
		VerifierID: "dupcode-update-baseline",
		Operation:  verifierauthority.OperationUpdateBaseline,
		Root:       root,
	}
	binder := newDupcodeUpdateBaselineBinderWithDeps(spec, deps)

	outcome := DupcodeUpdateBaselineOutcome{
		Dispatch: dispatcher.Dispatch(ctx, request, observer, binder.BindRunner()),
	}

	// Read the typed report from the binder's invocation-local cell.
	outcome.Report = reportToDTO(binder.Report(), DupcodeVerifySpec{
		MinLines:  spec.MinLines,
		MinTokens: spec.MinTokens,
	})
	return outcome
}

// DispatchDupcodeUpdateBaselineTyped is the production typed dispatch
// entry point for the dupcode update-baseline lane.
func DispatchDupcodeUpdateBaselineTyped(ctx context.Context, root string, spec DupcodeUpdateBaselineSpec) DupcodeUpdateBaselineOutcome {
	return dispatchDupcodeUpdateBaselineTypedWith(ctx, root, spec, &verifierdispatch.DefaultContextObserver{}, productionDupcodeBinderDeps())
}

// DupcodeBaselinePrintResult is the data-only print/export helper.
func DupcodeBaselinePrintResult(label string, findings []checks.Finding) int {
	return dupcode.PrintBaselineVerifyResult(label, findings)
}
