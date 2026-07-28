// SPDX-License-Identifier: Apache-2.0

package dupcodeauthority

import (
	"github.com/s1onique/leamas/internal/factory/checks"
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
func Guard(ctx DupcodeExecutionContext, real GuardedVerifier) GuardedVerifier {
	return func(root string) []checks.Finding {
		if err := ValidateDupcodeExecutionAuthority(ctx); err != nil {
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
func GuardedRun(root string, real GuardedVerifier) []checks.Finding {
	return Guard(DetectContext(root), real)(root)
}

// DetectDupcodeExecutionContext is an exported alias for DetectContext
// so the gate package can call it without importing internal functions.
func DetectDupcodeExecutionContext(root string) DupcodeExecutionContext {
	return DetectContext(root)
}
