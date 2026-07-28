// Package gate provides the quality gate command that runs all Factory verifiers.
package gate

import (
	"context"
	"fmt"

	"github.com/s1onique/leamas/internal/factory/verifierauthority"
	"github.com/s1onique/leamas/internal/factory/verifierdispatch"
)

// DispatcherForVerifier creates a dispatcher configured for a specific verifier.
// This is used by production entry points that need to route through the central
// authority check.
func DispatcherForVerifier(verifierName string) (*verifierdispatch.Dispatcher, bool) {
	verifiers := AllVerifiers()
	for _, v := range verifiers {
		if v.Name == verifierName {
			dispatcher := verifierdispatch.NewDispatcher(verifiers)
			return dispatcher, true
		}
	}
	return nil, false
}

// DispatchDupcodeVerify dispatches a dupcode verify request through the central authority.
// This is the canonical entry point for `factory verify dupcode`.
// The RunnerFactory is invoked only after authority validation passes.
func DispatchDupcodeVerify(ctx context.Context, root string, factory verifierdispatch.RunnerFactory) verifierdispatch.Result {
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

	// Use Dispatch with RunnerFactory - factory is invoked only after authority validation
	return dispatcher.Dispatch(ctx, request, factory)
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

	// Use Dispatch with RunnerFactory - factory is invoked only after authority validation
	return dispatcher.Dispatch(ctx, request, factory)
}

// DispatchDupcodeUpdateBaseline dispatches a dupcode update-baseline request.
// This is the canonical entry point for `factory verify dupcode --update-baseline`.
// The RunnerFactory is invoked only after authority validation passes.
func DispatchDupcodeUpdateBaseline(ctx context.Context, root string, factory verifierdispatch.RunnerFactory) verifierdispatch.Result {
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

	// Use Dispatch with RunnerFactory - factory is invoked only after authority validation
	return dispatcher.Dispatch(ctx, request, factory)
}
