// SPDX-License-Identifier: Apache-2.0

package closure

import (
	"encoding/json"
	"fmt"
	"path/filepath"
)

type v2RuntimeEvidence struct {
	ContractVersion          int                 `json:"contract_version"`
	ActID                    string              `json:"act_id"`
	PublicationBranch        string              `json:"publication_branch"`
	Runner                   RunnerIdentity      `json:"runner"`
	Checks                   []CheckResult       `json:"checks"`
	CheckEvidence            []EvidenceRecord    `json:"check_evidence"`
	PatchHygiene             PatchHygiene        `json:"patch_hygiene"`
	ClosurePolicy            ClosurePolicyResult `json:"closure_policy"`
	PatchDiagnosticsSHA256   string              `json:"patch_diagnostics_sha256"`
	ClosureDiagnosticsSHA256 string              `json:"closure_diagnostics_sha256"`
}

var v2ReservedEvidenceNames = map[string]struct{}{
	"runtime.json":        {},
	"patch-policy.log":    {},
	"closure-policy.log":  {},
	v2EvidenceIndexName:   {},
	"patch_hygiene.json":  {},
	"closure_policy.json": {},
}

func writeV2RuntimeEvidence(input v2FinalizeInput) error {
	for _, record := range input.CheckEvidence {
		if _, reserved := v2ReservedEvidenceNames[record.LogicalName]; reserved {
			return fmt.Errorf("check evidence uses reserved top-level name %q", record.LogicalName)
		}
	}
	if input.Branch == "" {
		return fmt.Errorf("finalize input is missing publication branch")
	}
	runtime := v2RuntimeEvidence{
		ContractVersion: 1, ActID: input.Plan.ActID,
		PublicationBranch: input.Branch, Runner: input.Runner,
		Checks: input.Checks, CheckEvidence: input.CheckEvidence,
		PatchHygiene:             input.Patch.Value,
		ClosurePolicy:            input.Closure.Value,
		PatchDiagnosticsSHA256:   SHA256Hex(input.Patch.Diagnostics),
		ClosureDiagnosticsSHA256: SHA256Hex(input.Closure.Diagnostics),
	}
	data, err := json.MarshalIndent(runtime, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal runtime evidence: %w", err)
	}
	data = append(data, '\n')
	if err := writeExclusiveRegular(filepath.Join(input.EvidenceDirectory, "runtime.json"), data, 0o600); err != nil {
		return fmt.Errorf("write runtime evidence: %w", err)
	}
	if err := writeExclusiveRegular(filepath.Join(input.EvidenceDirectory, "patch-policy.log"), input.Patch.Diagnostics, 0o600); err != nil {
		return fmt.Errorf("write patch diagnostics: %w", err)
	}
	if err := writeExclusiveRegular(filepath.Join(input.EvidenceDirectory, "closure-policy.log"), input.Closure.Diagnostics, 0o600); err != nil {
		return fmt.Errorf("write closure diagnostics: %w", err)
	}
	return nil
}
