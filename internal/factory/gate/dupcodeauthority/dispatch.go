// SPDX-License-Identifier: Apache-2.0

package dupcodeauthority

import (
	"context"

	"github.com/s1onique/leamas/internal/factory/checks"
	"github.com/s1onique/leamas/internal/factory/verifierauthority"
)

// GuardedVerifier is the canonical contract every dupcode verifier
// must satisfy before invoking the real analyzer. The closure
// returned by the central guard MUST be called before any other
// initialization in the underlying verifier.
type GuardedVerifier func(root string) []checks.Finding

// Guard wraps a real verifier with the central authority check.
// The real verifier is invoked only when the authority validator
// approves the execution context. Otherwise the wrapper returns a
// single canonical CI-only-denial finding without invoking the real
// verifier, recording the denial kind, severity, and message so other
// tools can grep the gate output.
func Guard(ctx DupcodeExecutionContext, operation verifierauthority.VerifierOperation, real GuardedVerifier) GuardedVerifier {
	return func(root string) []checks.Finding {
		if err := ValidateDupcodeExecutionAuthority(ctx, operation); err != nil {
			return []checks.Finding{
				{
					Path:     "dupcode",
					Kind:     "dupcode_ci_only_authority_denied",
					Message:  err.Error(),
					Severity: checks.SeverityError,
				},
			}
		}
		return real(root)
	}
}

// GuardedRun is a convenience wrapper that resolves the production
// context from the supplied repository root and applies the guard. It
// is the preferred entry point for callers that cannot supply a
// pre-built context (e.g., the CLI handler that handles
// `leamas factory gate --lane=dupcode`).
func GuardedRun(ctx DupcodeExecutionContext, operation verifierauthority.VerifierOperation, real GuardedVerifier, root string) []checks.Finding {
	return Guard(ctx, operation, real)(root)
}

// DetectDupcodeExecutionContextFromRoot reads the production environment
// and repository root to assemble a DupcodeExecutionContext.
func DetectDupcodeExecutionContextFromRoot(ctx context.Context, root string) DupcodeExecutionContext {
	ec := verifierauthority.DetectExecutionContext(ctx, root)
	return DetectContext(ec)
}
