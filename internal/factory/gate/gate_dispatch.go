// Package gate provides the quality gate command that runs all Factory verifiers.
package gate

import (
	"context"
	"fmt"

	"github.com/s1onique/leamas/internal/factory/verifierauthority"
	"github.com/s1onique/leamas/internal/factory/verifierdispatch"
)

// DispatchResult is the result returned by dispatch functions.
type DispatchResult = verifierdispatch.Result

// RunnerFactory is a function that creates a runner function.
type RunnerFactory = verifierdispatch.RunnerFactory

// DispatchFunc is the function signature for dispatch functions.
type DispatchFunc func(context.Context, string, RunnerFactory) DispatchResult

// DispatcherForVerifier creates a dispatcher configured for a specific verifier.
// This is used by production entry points that need to route through the central
// authority check.
func DispatcherForVerifier(verifierName string) (*verifierdispatch.Dispatcher, bool) {
	verifiers := AllVerifiers()
	for _, v := range verifiers {
		if v.Name == verifierName {
			dispatcher, err := verifierdispatch.NewDispatcher(verifiers)
			if err != nil {
				return nil, false
			}
			return dispatcher, true
		}
	}
	return nil, false
}

// DispatchDupcodeVerify dispatches a dupcode verify request through the central authority.
// This is the canonical entry point for `factory verify dupcode`.
// The RunnerFactory is invoked only after authority validation passes.
func DispatchDupcodeVerify(ctx context.Context, root string, factory verifierdispatch.RunnerFactory) verifierdispatch.Result {
	return DispatchDupcodeVerifyWithObserver(ctx, root, factory, &verifierdispatch.DefaultContextObserver{})
}

// DispatchDupcodeVerifyWithObserver is the exported variant that accepts an injectable observer.
// This enables testing without real environment variables.
func DispatchDupcodeVerifyWithObserver(ctx context.Context, root string, factory verifierdispatch.RunnerFactory, observer verifierdispatch.ContextObserver) verifierdispatch.Result {
	dispatcher, ok := DispatcherForVerifier("dupcode")
	if !ok {
		return verifierdispatch.Result{
			Error: fmt.Errorf("dupcode verifier not found in registry"),
		}
	}

	request := verifierdispatch.Request{
		VerifierID: "dupcode",
		Operation:  verifierauthority.OperationVerify,
		Root:       root,
	}

	return dispatcher.Dispatch(ctx, request, observer, factory)
}

// DispatchDupcodeBaselineVerify dispatches a dupcode-baseline verify request.
// This is the canonical entry point for `factory verify dupcode-baseline`.
// The RunnerFactory is invoked only after authority validation passes.
func DispatchDupcodeBaselineVerify(ctx context.Context, root string, factory verifierdispatch.RunnerFactory) verifierdispatch.Result {
	dispatcher, ok := DispatcherForVerifier("dupcode-baseline")
	if !ok {
		return verifierdispatch.Result{
			Error: fmt.Errorf("dupcode-baseline verifier not found in registry"),
		}
	}

	request := verifierdispatch.Request{
		VerifierID: "dupcode-baseline",
		Operation:  verifierauthority.OperationVerify,
		Root:       root,
	}

	// Use Dispatch with observer - factory is invoked only after authority validation
	observer := &verifierdispatch.DefaultContextObserver{}
	return dispatcher.Dispatch(ctx, request, observer, factory)
}

// DispatchDupcodeUpdateBaseline dispatches a dupcode update-baseline request.
// This is the canonical entry point for `factory verify dupcode --update-baseline`.
// The RunnerFactory is invoked only after authority validation passes.
func DispatchDupcodeUpdateBaseline(ctx context.Context, root string, factory verifierdispatch.RunnerFactory) verifierdispatch.Result {
	return DispatchDupcodeUpdateBaselineWithObserver(ctx, root, factory, &verifierdispatch.DefaultContextObserver{})
}

// DispatchDupcodeUpdateBaselineWithObserver is the exported variant that accepts an injectable observer.
// This enables testing without real environment variables.
func DispatchDupcodeUpdateBaselineWithObserver(ctx context.Context, root string, factory verifierdispatch.RunnerFactory, observer verifierdispatch.ContextObserver) verifierdispatch.Result {
	dispatcher, ok := DispatcherForVerifier("dupcode")
	if !ok {
		return verifierdispatch.Result{
			Error: fmt.Errorf("dupcode verifier not found in registry"),
		}
	}

	request := verifierdispatch.Request{
		VerifierID: "dupcode",
		Operation:  verifierauthority.OperationUpdateBaseline,
		Root:       root,
	}

	return dispatcher.Dispatch(ctx, request, observer, factory)
}
