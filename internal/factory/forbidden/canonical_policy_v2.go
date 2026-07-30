// SPDX-License-Identifier: Apache-2.0

package forbidden

import (
	"fmt"

	"github.com/s1onique/leamas/internal/factory/checks"
	"github.com/s1onique/leamas/internal/factory/forbidden/reporoot"
)

// DupcodeBypassPolicy binds immutable repository identity for one analysis.
type DupcodeBypassPolicy struct {
	repoRoot   string
	modulePath string
	resolver   *reporoot.RootResolver
}

func NewDupcodeBypassPolicy(repoRoot, modulePath string) (*DupcodeBypassPolicy, error) {
	resolver := reporoot.New()
	canonicalRoot, err := resolver.Resolve(repoRoot)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve canonical root: %w", err)
	}
	return &DupcodeBypassPolicy{
		repoRoot:   canonicalRoot,
		modulePath: modulePath,
		resolver:   resolver,
	}, nil
}

// CanonicalCheckDupcodeBypass performs one self-contained globally typed
// policy analysis. No mutable analysis state survives this call.
func CanonicalCheckDupcodeBypass(root, modulePath string) []checks.Finding {
	return runCanonicalAnalysis(root, modulePath, productionCanonicalConfig()).Findings
}
