// SPDX-License-Identifier: Apache-2.0

package protectedverifier

import (
	"slices"

	"github.com/s1onique/leamas/internal/factory/dupcode"
)

// cloneDupcodeConfig copies every current mutable configuration field.
func cloneDupcodeConfig(cfg dupcode.Config) dupcode.Config {
	out := cfg
	out.ExcludeDirs = slices.Clone(cfg.ExcludeDirs)
	out.ExcludeFileSuffixes = slices.Clone(cfg.ExcludeFileSuffixes)
	return out
}

// cloneDupcodeInput copies an input without retaining caller-owned slices.
func cloneDupcodeInput(input DupcodeInput) DupcodeInput {
	out := input
	out.Config = cloneDupcodeConfig(input.Config)
	return out
}

// cloneDupcodeFinding copies a finding and its occurrence slice.
func cloneDupcodeFinding(f dupcode.Finding) dupcode.Finding {
	out := f
	out.Occurrences = slices.Clone(f.Occurrences)
	return out
}

// cloneDupcodeAnalysis recursively isolates every current slice field.
func cloneDupcodeAnalysis(a *DupcodeAnalysis) *DupcodeAnalysis {
	if a == nil {
		return nil
	}
	out := &DupcodeAnalysis{
		Config:      cloneDupcodeConfig(a.Config),
		Occurrences: slices.Clone(a.Occurrences),
		Findings:    make([]dupcode.Finding, len(a.Findings)),
	}
	for i := range a.Findings {
		out.Findings[i] = cloneDupcodeFinding(a.Findings[i])
	}
	return out
}
