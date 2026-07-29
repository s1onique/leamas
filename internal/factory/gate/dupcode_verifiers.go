// SPDX-License-Identifier: Apache-2.0

package gate

import (
	"fmt"
	"path/filepath"

	"github.com/s1onique/leamas/internal/factory/checks"
	"github.com/s1onique/leamas/internal/factory/protectedverifier"
)

// dupCodeVerifier is the named post-authority bound runner for the dupcode
// lane when used through the AllVerifiers registry (direct commands). It is
// registered as the Run function of the "dupcode" entry in AllVerifiers.
//
// This function is callable as a verifier by the dispatcher ONLY after the
// dispatcher admits authority for the request — the registry entry exists
// before authority, but the Run function is invoked exclusively from the
// dispatcher post-authority path.
func dupCodeVerifier(root string) []checks.Finding {
	runner := protectedverifier.NewDupcodeRunner()
	cfg := protectedverifier.DefaultConfig()

	baselinePath := ".factory/dupcode-baseline.json"
	fullBaselinePath := baselinePath
	if root != "." && root != "" {
		fullBaselinePath = filepath.Join(root, baselinePath)
	}

	if !checks.FileExists(fullBaselinePath) {
		return []checks.Finding{
			{Path: baselinePath, Kind: "missing_baseline", Message: "baseline file not found. Run 'make dupcode-baseline' to create it.", Severity: checks.SeverityError},
		}
	}

	baseline, err := runner.LoadBaseline(fullBaselinePath)
	if err != nil {
		return []checks.Finding{
			{Path: baselinePath, Kind: "baseline_error", Message: fmt.Sprintf("failed to load baseline: %v", err), Severity: checks.SeverityError},
		}
	}

	cfg.Root = root
	cfg.MinLines = baseline.Thresholds.MinLines
	cfg.MinTokens = baseline.Thresholds.MinTokens

	report, err := runner.RunCheckReport(root, cfg)
	if err != nil {
		return []checks.Finding{
			{Path: "dupcode", Kind: "dupcode_error", Message: fmt.Sprintf("duplicate code scan failed: %v", err), Severity: checks.SeverityError},
		}
	}

	result := runner.CompareToBaseline(report, baseline)
	return convertDupcodeCompareResult(result)
}

// dupcodeBaselineVerifier is the named post-authority bound runner for the
// dupcode-baseline lane when used through the AllVerifiers registry (direct
// commands). It is registered as the Run function of the "dupcode-baseline"
// entry in AllVerifiers.
func dupcodeBaselineVerifier(root string) []checks.Finding {
	runner := protectedverifier.NewDupcodeRunner()
	policy := protectedverifier.BaselinePolicy{
		Path: ".factory/dupcode-baseline.json",
	}
	if root != "." && root != "" {
		policy.Path = filepath.Join(root, policy.Path)
	}

	findings, err := runner.VerifyBaseline(root, policy)
	if err != nil {
		return []checks.Finding{
			{Path: "dupcode", Kind: "dupcode_error", Message: fmt.Sprintf("baseline verification failed: %v", err), Severity: checks.SeverityError},
		}
	}
	return findings
}

// convertDupcodeCompareResult converts dupcode comparison results to gate findings.
func convertDupcodeCompareResult(result protectedverifier.CompareResult) []checks.Finding {
	var findings []checks.Finding
	if !result.HasChanges {
		return findings
	}

	for _, f := range result.NewFindings {
		findings = append(findings, checks.Finding{
			Path:     "dupcode",
			Kind:     "new_duplicate",
			Message:  fmt.Sprintf("new duplicate block (tokens: %d)", f.TokenCount),
			Severity: checks.SeverityError,
		})
	}

	for range result.WorsenedFindings {
		findings = append(findings, checks.Finding{
			Path:     "dupcode",
			Kind:     "worsened_duplicate",
			Message:  "worsened duplicate block",
			Severity: checks.SeverityError,
		})
	}

	return findings
}

// dupCodeUpdateBaselineVerifier is the named post-authority bound runner
// for the dupcode-update-baseline lane when used through the AllVerifiers
// registry (direct commands). The typed dispatch path uses
// dispatchDupcodeUpdateBaselineTypedWith and binder.BindRunner instead of
// invoking this Run directly. This stub returns nil findings to satisfy the
// registry signature; the real work is performed by the binder inside the
// typed dispatcher.
func dupCodeUpdateBaselineVerifier(root string) []checks.Finding {
	_ = root
	return nil
}
