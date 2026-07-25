// SPDX-License-Identifier: Apache-2.0

package closure

import (
	"bytes"
	"encoding/json"
	"fmt"
)

const closureProtocolVersionV2 = 2

// v2CanonicalArtifacts are the stable bytes committed in C. Runtime-instance
// details remain detached and will be bound by a later evidence transaction.
type v2CanonicalArtifacts struct {
	ManifestBytes  []byte
	ManifestSHA256 string
	ReportBytes    []byte
	ReportSHA256   string
	Verdict        string
}

type v2CanonicalArtifactInput struct {
	ActID           string
	PlanPath        string
	PlanSHA256      string
	PlanBlobOID     string
	FreezeCommit    string
	FreezeTree      string
	SubjectCommit   string
	SubjectTree     string
	Branch          string // detached-only input; deliberately not serialized
	Runner          RunnerIdentity
	RunnerAuthority *RunnerAuthority
	Checks          []CheckResult
	PatchHygiene    PatchHygiene
	ClosurePolicy   ClosurePolicyResult
}

type v2CanonicalManifest struct {
	ContractVersion        int                 `json:"contract_version"`
	ClosureProtocolVersion int                 `json:"closure_protocol_version"`
	ActID                  string              `json:"act_id"`
	Plan                   ManifestPlanRef     `json:"plan"`
	PlanFreeze             ManifestPlanFreeze  `json:"plan_freeze"`
	Subject                ManifestSubject     `json:"subject"`
	RunnerAuthority        v2RunnerAuthority   `json:"runner_authority"`
	Checks                 []v2CanonicalCheck  `json:"checks"`
	PatchHygiene           PatchHygiene        `json:"patch_hygiene"`
	ClosurePolicy          ClosurePolicyResult `json:"closure_policy"`
	Verdict                string              `json:"verdict"`
}

type v2RunnerAuthority struct {
	Mode     RunnerAuthorityMode `json:"mode"`
	Tool     *ToolAuthority      `json:"tool,omitempty"`
	Revision string              `json:"revision,omitempty"`
}

type v2CanonicalCheck struct {
	CheckID          string   `json:"check_id"`
	SubjectTreeOID   string   `json:"subject_tree_oid"`
	Argv             []string `json:"argv"`
	WorkingDirectory string   `json:"working_directory"`
	ExitCode         *int     `json:"exit_code,omitempty"`
	Status           string   `json:"status"`
	CleanupStatus    string   `json:"cleanup_status"`
	OutputComplete   bool     `json:"output_complete"`
}

func generateV2CanonicalArtifacts(input v2CanonicalArtifactInput) (v2CanonicalArtifacts, error) {
	checks := make([]v2CanonicalCheck, len(input.Checks))
	verdict := VerdictPass
	for i, result := range input.Checks {
		checks[i] = stableV2Check(result)
		if !canonicalCheckPassed(checks[i]) {
			verdict = VerdictFail
		}
	}
	if input.Runner.VCSModified || input.PatchHygiene.Status != CheckStatusPass ||
		input.ClosurePolicy.TrackedFullDigestStatus != CheckStatusPass {
		verdict = VerdictFail
	}

	// Determine runner authority mode
	runnerMode := RunnerAuthoritySubjectExact
	var runnerTool *ToolAuthority
	if input.RunnerAuthority != nil {
		runnerMode = input.RunnerAuthority.Mode
		runnerTool = input.RunnerAuthority.Tool
	}

	manifest := v2CanonicalManifest{
		ContractVersion: ContractVersionV1, ClosureProtocolVersion: closureProtocolVersionV2,
		ActID: input.ActID,
		Plan:  ManifestPlanRef{Path: input.PlanPath, SHA256: input.PlanSHA256},
		PlanFreeze: ManifestPlanFreeze{FreezeCommit: input.FreezeCommit, FreezeTree: input.FreezeTree,
			PlanPath: input.PlanPath, PlanBlobOID: input.PlanBlobOID, PlanSHA256: input.PlanSHA256,
			SubjectCommit: input.SubjectCommit},
		Subject:         ManifestSubject{CommitOID: input.SubjectCommit, TreeOID: input.SubjectTree},
		RunnerAuthority: v2RunnerAuthority{Mode: runnerMode, Tool: runnerTool}, Checks: checks,
		PatchHygiene: input.PatchHygiene, ClosurePolicy: input.ClosurePolicy, Verdict: verdict,
	}
	manifestBytes, err := marshalCanonicalJSON(manifest)
	if err != nil {
		return v2CanonicalArtifacts{}, fmt.Errorf("marshal canonical manifest: %w", err)
	}
	reportBytes := renderV2CanonicalReport(manifest)
	return v2CanonicalArtifacts{
		ManifestBytes: manifestBytes, ManifestSHA256: SHA256Hex(manifestBytes),
		ReportBytes: reportBytes, ReportSHA256: SHA256Hex(reportBytes), Verdict: verdict,
	}, nil
}

func stableV2Check(result CheckResult) v2CanonicalCheck {
	return v2CanonicalCheck{
		CheckID: result.CheckID, SubjectTreeOID: result.SubjectTreeOID,
		Argv: result.Argv, WorkingDirectory: result.WorkingDirectory,
		ExitCode: result.ExitCode, Status: result.Status, CleanupStatus: result.CleanupStatus,
		OutputComplete: !result.OutputTruncated && !result.OutputIncomplete,
	}
}

func canonicalCheckPassed(check v2CanonicalCheck) bool {
	return check.Status == CheckStatusPass && check.ExitCode != nil && *check.ExitCode == 0 &&
		check.CleanupStatus == CleanupPass && check.OutputComplete
}

func marshalCanonicalJSON(value any) ([]byte, error) {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(data, '\n'), nil
}

func renderV2CanonicalReport(manifest v2CanonicalManifest) []byte {
	var report bytes.Buffer
	fmt.Fprintf(&report, "# Close Report: %s\n\n", manifest.ActID)
	fmt.Fprintf(&report, "- Contract version: %d\n", manifest.ContractVersion)
	fmt.Fprintf(&report, "- Closure protocol version: %d\n", manifest.ClosureProtocolVersion)
	fmt.Fprintf(&report, "- Verdict: %s\n", manifest.Verdict)
	fmt.Fprintf(&report, "- Freeze commit: %s\n", manifest.PlanFreeze.FreezeCommit)
	fmt.Fprintf(&report, "- Subject commit: %s\n", manifest.Subject.CommitOID)
	fmt.Fprintf(&report, "- Runner authority: %s\n\n## Checks\n", manifest.RunnerAuthority.Mode)
	for _, check := range manifest.Checks {
		fmt.Fprintf(&report, "- %s: %s (exit=%s, cleanup=%s, output_complete=%t)\n",
			check.CheckID, check.Status, canonicalExitCode(check.ExitCode),
			check.CleanupStatus, check.OutputComplete)
	}
	return report.Bytes()
}

func canonicalExitCode(exitCode *int) string {
	if exitCode == nil {
		return "none"
	}
	return fmt.Sprintf("%d", *exitCode)
}
