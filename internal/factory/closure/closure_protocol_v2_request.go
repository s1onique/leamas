// SPDX-License-Identifier: Apache-2.0

package closure

// closure_protocol_v2_request.go provides the validated v2
// lifecycle request. ValidateV2Request rejects incomplete or
// inconsistent requests before any git work runs so the
// runner never reaches topology resolution with zero-value
// fields.
//
// Splitting this from closure_protocol_v2.go keeps the file
// under the LLM-friendly 400-line threshold while preserving
// the single closure over the descriptor that
// ACT-LEAMAS-FACTORY-CLOSURE-PROTOCOL-V1-01 requires.

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ValidateV2Request enforces the request contract. Every
// required field must be non-empty and well-formed. The
// function returns a typed V2Error per missing field so the
// CLI can emit deterministic machine-readable output.
//
// The supported version combination is enforced via
// ValidateV2VersionCombination. Repository root and evidence
// directory are checked against the filesystem; the
// manifest output is checked for detached location.
func ValidateV2Request(req V2Request) error {
	var diags V2Diagnostics
	if req.ClosureProtocolVersion == "" {
		diags = append(diags, V2Diagnostic{
			Code:         V2CodeRequestIncomplete,
			Message:      "closure_protocol_version is required",
			PropertyName: "closure_protocol_version",
		})
	}
	if req.PlanContractVersion == 0 {
		diags = append(diags, V2Diagnostic{
			Code:         V2CodeRequestIncomplete,
			Message:      "plan_contract_version is required",
			PropertyName: "plan_contract_version",
		})
	}
	if strings.TrimSpace(req.RepositoryRoot) == "" {
		diags = append(diags, V2Diagnostic{
			Code:         V2CodeRequestIncomplete,
			Message:      "repository_root is required",
			PropertyName: "repository_root",
		})
	}
	if strings.TrimSpace(req.SubjectCommit) == "" {
		diags = append(diags, V2Diagnostic{
			Code:         V2CodeRequestIncomplete,
			Message:      "subject_commit is required",
			PropertyName: "subject_commit",
		})
	}
	if strings.TrimSpace(req.FreezeCommit) == "" {
		diags = append(diags, V2Diagnostic{
			Code:         V2CodeRequestIncomplete,
			Message:      "freeze_commit is required",
			PropertyName: "freeze_commit",
		})
	}
	if strings.TrimSpace(req.PlanPath) == "" {
		diags = append(diags, V2Diagnostic{
			Code:         V2CodeRequestIncomplete,
			Message:      "plan_path is required",
			PropertyName: "plan_path",
		})
	}
	if strings.TrimSpace(req.EvidenceDirectory) == "" {
		diags = append(diags, V2Diagnostic{
			Code:         V2CodeRequestIncomplete,
			Message:      "evidence_directory is required",
			PropertyName: "evidence_directory",
		})
	}
	if strings.TrimSpace(req.ManifestOutput) == "" {
		diags = append(diags, V2Diagnostic{
			Code:         V2CodeRequestIncomplete,
			Message:      "manifest_output is required",
			PropertyName: "manifest_output",
		})
	}
	if len(diags) > 0 {
		return &V2Error{Diags: diags}
	}
	if !PlanContractVersion(req.PlanContractVersion).IsSupported() {
		return NewV2ErrorWith(V2CodeUnsupportedPlanContractVersion,
			fmt.Sprintf("plan contract version %d is not supported", req.PlanContractVersion),
			"plan_contract_version", "")
	}
	if !req.ClosureProtocolVersion.IsSupported() {
		return NewV2ErrorWith(V2CodeUnsupportedClosureProtocolVersion,
			fmt.Sprintf("closure protocol version %q is not supported", string(req.ClosureProtocolVersion)),
			"closure_protocol_version", "")
	}
	if filepath.IsAbs(req.PlanPath) {
		return NewV2ErrorWith(V2CodeInvalidPlanPath,
			fmt.Sprintf("plan path %q must be repository-relative", req.PlanPath),
			"plan_path", req.PlanPath)
	}
	if !filepath.IsAbs(req.EvidenceDirectory) {
		return NewV2ErrorWith(V2CodeRequestIncomplete,
			"evidence_directory must be an absolute detached path",
			"evidence_directory", req.EvidenceDirectory)
	}
	if !filepath.IsAbs(req.ManifestOutput) {
		return NewV2ErrorWith(V2CodeRequestIncomplete,
			"manifest_output must be an absolute detached path",
			"manifest_output", req.ManifestOutput)
	}
	if req.PlanPath == "." || strings.HasPrefix(req.PlanPath, "..") ||
		strings.HasPrefix(req.PlanPath, "/") {
		return NewV2ErrorWith(V2CodeInvalidPlanPath,
			fmt.Sprintf("plan path %q is not a safe repository-relative path", req.PlanPath),
			"plan_path", req.PlanPath)
	}
	if _, err := os.Stat(req.RepositoryRoot); err != nil {
		return NewV2ErrorWith(V2CodeRequestIncomplete,
			fmt.Sprintf("repository_root %q is not accessible: %s", req.RepositoryRoot, err.Error()),
			"repository_root", err.Error())
	}
	return ValidateV2VersionCombination(PlanContractVersion(req.PlanContractVersion), req.ClosureProtocolVersion)
}
