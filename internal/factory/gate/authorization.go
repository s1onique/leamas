// SPDX-License-Identifier: Apache-2.0

package gate

import (
	"context"
	"fmt"
	"os"

	"github.com/s1onique/leamas/internal/factory/verifierauthority"
	"github.com/s1onique/leamas/internal/factory/verifierdispatch"
)

// authorizeFactorize performs batch authorization for factorize.
// This is called BEFORE any expensive operations like shared context creation.
func authorizeFactorize(ctx context.Context, root string) (*verifierdispatch.AuthorizedProfile, error) {
	allVerifiers := AllVerifiers()

	dispatcher, err := verifierdispatch.NewDispatcher(allVerifiers)
	if err != nil {
		return nil, fmt.Errorf("verifier dispatch: %w", err)
	}

	observer := &verifierdispatch.DefaultContextObserver{}

	// Build authorization requests for all verifiers
	requests := make([]verifierdispatch.ProfileRequest, len(allVerifiers))
	for i, v := range allVerifiers {
		requests[i] = verifierdispatch.ProfileRequest{
			VerifierID: v.Name,
			Operation:  verifierauthority.OperationVerify,
		}
	}

	profile, err := dispatcher.AuthorizeProfile(ctx, root, requests, observer)
	if err != nil {
		return nil, fmt.Errorf("authorization: %w", err)
	}

	return profile, nil
}

// printAuthorizationDenials prints denial findings to stderr.
func printAuthorizationDenials(denials []verifierdispatch.ProfileDenial) {
	fmt.Fprintf(os.Stderr, "\n--- FACTORIZE AUTHORIZATION DENIED ---\n")
	for _, denial := range denials {
		for _, f := range denial.Findings {
			fmt.Fprintf(os.Stderr, "  %s: %s: %s\n", f.Path, f.Kind, f.Message)
		}
	}
	fmt.Fprintf(os.Stderr, "\n*** FACTORIZE FAILED: Authorization denied ***\n")
}
