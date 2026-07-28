// Package gate provides the quality gate command that runs all Factory verifiers.
package gate

import (
	"context"

	"github.com/s1onique/leamas/internal/factory/gate/dupcodeauthority"
	"github.com/s1onique/leamas/internal/factory/verifierauthority"
)

// DetectDupcodeExecutionContext detects the execution context for dupcode authority.
// This is the canonical context detection used by the CLI.
func DetectDupcodeExecutionContext(ctx context.Context, root string) (dupcodeauthority.DupcodeExecutionContext, error) {
	ec := verifierauthority.DetectExecutionContext(ctx, root)
	return dupcodeauthority.DetectContext(ec), nil
}

// ValidateDupcodeExecutionAuthority validates the dupcode execution authority.
// This is the single point of authority enforcement for all direct CLI dupcode access.
func ValidateDupcodeExecutionAuthority(ctx dupcodeauthority.DupcodeExecutionContext, operation verifierauthority.VerifierOperation) error {
	return dupcodeauthority.ValidateDupcodeExecutionAuthority(ctx, operation)
}
