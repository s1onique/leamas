// SPDX-License-Identifier: Apache-2.0

package closure

// v2_verifier_manifest.go implements Phase 1-7 of
// ACT-LEAMAS-FACTORY-CLOSURE-PROTOCOL-V2-VERIFIER-MANIFEST-RESULTS01.
//
// The verifier parses the exact C:M bytes, rejects anything
// that disagrees with the independently computed Git
// authority, classifies the binary identity, parses the frozen
// plan at F:P and enforces a strict check/result bijection
// against the committed manifest's CheckResults, and finally
// enforces a successful-run aggregate (no failed check, no
// cleanup failure, no timeout / cancellation / overflow
// classified as success, exclude semantics preserved).
//
// Every step is fail-closed: each violation emits a typed
// diagnostic and returns the partial verdict. A successful
// parse + identity + bijection + integrity sequence yields
// exactly one canonical V2ManifestIdentityFacts value with an
// empty Diagnostics slice.

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

// parsedManifest is the strict typed view of the committed
// manifest bytes at C:M. The struct is private: callers see
// only the validated V2ManifestIdentityFacts verdict.
//
// Fields use json.Number wherever the production wire type
// might be ambiguous so the verifier never silently coerces
// integers or floats.
type parsedManifest struct {
	ClosureProtocolVersion json.Number `json:"closure_protocol_version"`
	PlanContractVersion    json.Number `json:"plan_contract_version"`

	SubjectCommit string `json:"subject_commit"`
	SubjectTree   string `json:"subject_tree"`

	FreezeCommit string `json:"freeze_commit"`
	FreezeTree   string `json:"freeze_tree"`

	PlanPath   string `json:"plan_path"`
	PlanBlob   string `json:"plan_blob"`
	PlanSHA256 string `json:"plan_sha256"`

	ExecutionTree string `json:"execution_tree"`

	CheckResults []parsedCheckResult `json:"check_results"`

	LeamasBinaryIdentity parsedBinaryIdentity `json:"leamas_binary_identity"`

	CallerHead    string `json:"caller_head,omitempty"`
	ClosureCommit string `json:"closure_commit,omitempty"`
}

// parsedCheckResult is the strict typed view of one manifest
// check result entry.
type parsedCheckResult struct {
	ID                      string           `json:"id"`
	Mode                    string           `json:"mode"`
	Outcome                 string           `json:"outcome"`
	ExitCode                *json.Number     `json:"exit_code,omitempty"`
	DurationMS              json.Number      `json:"duration_ms"`
	ExecutionClassification string           `json:"execution_classification"`
	CleanupStatus           string           `json:"cleanup_status"`
	Evidence                []parsedEvidence `json:"evidence,omitempty"`
	Detail                  string           `json:"detail,omitempty"`
}

// parsedEvidence is the strict typed view of one detached
// evidence reference.
type parsedEvidence struct {
	LogicalName  string      `json:"logical_name"`
	MediaType    string      `json:"media_type"`
	SHA256       string      `json:"sha256"`
	ByteCount    json.Number `json:"byte_count"`
	Availability string      `json:"availability"`
}

// parsedBinaryIdentity is the strict typed view of the
// committed binary identity.
type parsedBinaryIdentity struct {
	Path          string `json:"path"`
	SHA256        string `json:"sha256"`
	VCSRevision   string `json:"vcs_revision"`
	VCSModified   bool   `json:"vcs_modified"`
	LeamasVersion string `json:"leamas_version"`
}

// V2ManifestIdentityFacts is the structured verdict for the
// manifest-identity + bijection + integrity checks. The
// caller receives this only when the manifest is well-formed;
// otherwise the function returns early with a non-empty
// Diagnostics slice.
//
// The verdict is deterministic: every field is derived from
// the bound authority plus the parsed manifest bytes; the
// order of Diagnostics is fixed; identical inputs always
// produce identical facts.
type V2ManifestIdentityFacts struct {
	ManifestIdentityValid bool
	BinaryIdentityValid   bool
	BijectionValid        bool
	SuccessValid          bool

	PlanChecks    []PlanCheck
	MappedResults []V2CheckResult

	Diagnostics V2VerifierDiagnostics
}

// V2ManifestIdentityVerifier verifies the committed manifest
// against the independently computed Git authority plus the
// frozen plan at F:P.
type V2ManifestIdentityVerifier interface {
	VerifyManifestIdentity(
		rawManifest []byte,
		frozenPlan V2FrozenPlanAuthority,
		committedManifest V2CommittedManifestAuthority,
		topology V2ClosureTopology,
	) (V2ManifestIdentityFacts, error)
}

type v2ManifestIdentityVerifier struct{}

func NewV2ManifestIdentityVerifier() V2ManifestIdentityVerifier {
	return v2ManifestIdentityVerifier{}
}

// VerifyManifestIdentity runs the four-phase verification
// pipeline:
//
//  1. Strict parse
//  2. Identity binding
//  3. Binary identity
//  4. Frozen-plan inventory + bijection + success
func (v2ManifestIdentityVerifier) VerifyManifestIdentity(
	rawManifest []byte,
	frozenPlan V2FrozenPlanAuthority,
	committedManifest V2CommittedManifestAuthority,
	topology V2ClosureTopology,
) (V2ManifestIdentityFacts, error) {
	if len(rawManifest) == 0 {
		return V2ManifestIdentityFacts{Diagnostics: V2VerifierDiagnostics{NewV2VerifierDiagnostic(
			V2VerifierClosureManifestInvalidJSON,
			"committed manifest bytes are empty",
		)}}, nil
	}
	if len(frozenPlan.Diagnostics) > 0 {
		return V2ManifestIdentityFacts{Diagnostics: copyVerifierDiagnostics(frozenPlan.Diagnostics)}, nil
	}
	if len(committedManifest.Diagnostics) > 0 {
		return V2ManifestIdentityFacts{Diagnostics: copyVerifierDiagnostics(committedManifest.Diagnostics)}, nil
	}
	if topology.SubjectCommit == "" ||
		topology.SubjectTree == "" ||
		topology.FreezeCommit == "" ||
		topology.FreezeTree == "" {
		return V2ManifestIdentityFacts{}, NewV2VerifierError(NewV2VerifierDiagnostic(
			V2VerifierClosureManifestContractInvalid,
			"topology anchor is required for manifest identity verification",
		))
	}

	parsed, diags := parseV2StrictManifest(rawManifest)
	if len(diags) > 0 {
		return V2ManifestIdentityFacts{Diagnostics: diags}, nil
	}

	identityDiags := verifyV2ManifestIdentityWithAnchor(parsed, topology, frozenPlan)
	if len(identityDiags) > 0 {
		return V2ManifestIdentityFacts{Diagnostics: identityDiags}, nil
	}

	binaryDiags := verifyV2ManifestBinaryIdentity(parsed.LeamasBinaryIdentity)
	binaryValid := len(binaryDiags) == 0

	planChecks, bijectDiags := verifyV2FrozenPlanInventory(frozenPlan.RawBytes)
	if len(bijectDiags) > 0 {
		combined := append(V2VerifierDiagnostics{}, binaryDiags...)
		combined = append(combined, bijectDiags...)
		return V2ManifestIdentityFacts{
			BinaryIdentityValid: binaryValid,
			Diagnostics:         combined,
		}, nil
	}
	bijectionValid, mapped, bijectDiags := verifyV2ManifestCheckBijection(planChecks, parsed.CheckResults)
	successValid, successDiags := verifyV2ManifestRunSuccessIntegrity(planChecks, mapped)
	combined := append(V2VerifierDiagnostics{}, binaryDiags...)
	combined = append(combined, bijectDiags...)
	combined = append(combined, successDiags...)
	facts := V2ManifestIdentityFacts{
		ManifestIdentityValid: true,
		BinaryIdentityValid:   binaryValid,
		BijectionValid:        bijectionValid,
		SuccessValid:          successValid,
		PlanChecks:            planChecks,
		MappedResults:         mapped,
	}
	if len(combined) > 0 {
		facts.Diagnostics = combined
	}
	return facts, nil
}

// parseV2StrictManifest parses the exact committed manifest
// bytes with a closed contract.
func parseV2StrictManifest(raw []byte) (parsedManifest, V2VerifierDiagnostics) {
	var diags V2VerifierDiagnostics

	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 {
		return parsedManifest{}, V2VerifierDiagnostics{NewV2VerifierDiagnostic(
			V2VerifierClosureManifestInvalidJSON,
			"committed manifest bytes are whitespace-only",
		)}
	}
	if trimmed[0] != '{' {
		return parsedManifest{}, V2VerifierDiagnostics{NewV2VerifierDiagnostic(
			V2VerifierClosureManifestInvalidJSON,
			"committed manifest must be a JSON object",
		)}
	}

	dec := json.NewDecoder(bytes.NewReader(trimmed))
	dec.UseNumber()
	var rawObj map[string]json.RawMessage
	if err := dec.Decode(&rawObj); err != nil {
		return parsedManifest{}, V2VerifierDiagnostics{NewV2VerifierDiagnostic(
			V2VerifierClosureManifestInvalidJSON,
			fmt.Sprintf("invalid committed manifest JSON: %s", err.Error()),
		)}
	}
	if dec.More() {
		return parsedManifest{}, V2VerifierDiagnostics{NewV2VerifierDiagnostic(
			V2VerifierClosureManifestInvalidJSON,
			"trailing non-whitespace tokens after top-level manifest object",
		)}
	}

	buf, err := json.Marshal(rawObj)
	if err != nil {
		return parsedManifest{}, V2VerifierDiagnostics{NewV2VerifierDiagnostic(
			V2VerifierClosureManifestInvalidJSON,
			fmt.Sprintf("remarshal manifest object: %s", err.Error()),
		)}
	}
	strict := json.NewDecoder(bytes.NewReader(buf))
	strict.UseNumber()
	strict.DisallowUnknownFields()
	var parsed parsedManifest
	if err := strict.Decode(&parsed); err != nil {
		code := V2VerifierClosureManifestInvalidJSON
		msg := err.Error()
		lower := strings.ToLower(msg)
		if strings.Contains(lower, "unknown field") ||
			strings.Contains(lower, "cannot unmarshal") ||
			strings.Contains(lower, "invalid") {
			code = V2VerifierClosureManifestContractInvalid
		}
		return parsedManifest{}, V2VerifierDiagnostics{NewV2VerifierDiagnostic(code, msg)}
	}

	for key := range rawObj {
		if !v2ManifestTopLevelAllowed(key) {
			diags = append(diags, NewV2VerifierDiagnostic(
				V2VerifierClosureManifestContractInvalid,
				fmt.Sprintf("unknown top-level manifest field %q", key),
			).withProperty(key))
		}
	}

	if parsed.ClosureProtocolVersion.String() != "2" {
		diags = append(diags, NewV2VerifierDiagnostic(
			V2VerifierManifestProtocolVersionMismatch,
			fmt.Sprintf("manifest.closure_protocol_version must be 2, got %q", parsed.ClosureProtocolVersion.String()),
		).withProperty("closure_protocol_version").
			withExpected("2").
			withObserved(parsed.ClosureProtocolVersion.String()))
	}

	if parsed.PlanContractVersion.String() != "1" {
		diags = append(diags, NewV2VerifierDiagnostic(
			V2VerifierManifestPlanContractVersionMismatch,
			fmt.Sprintf("manifest.plan_contract_version must be 1, got %q", parsed.PlanContractVersion.String()),
		).withProperty("plan_contract_version").
			withExpected("1").
			withObserved(parsed.PlanContractVersion.String()))
	}

	if len(diags) > 0 {
		return parsedManifest{}, diags
	}
	return parsed, nil
}

// v2ManifestTopLevelAllowed enumerates the closed set of
// top-level manifest fields. Fields outside this set are
// rejected as manifest_contract_invalid.
func v2ManifestTopLevelAllowed(key string) bool {
	switch key {
	case "closure_protocol_version",
		"plan_contract_version",
		"subject_commit",
		"subject_tree",
		"freeze_commit",
		"freeze_tree",
		"plan_path",
		"plan_blob",
		"plan_sha256",
		"execution_tree",
		"check_results",
		"leamas_binary_identity",
		"caller_head",
		"closure_commit":
		return true
	}
	return false
}

// v2FieldMismatch is the canonical manifest-identity
// diagnostic shape. The helper keeps the call sites
// single-line so the multi-line chain pattern is not needed.
func v2FieldMismatch(code V2VerifierCode, field, expected, observed string) V2VerifierDiagnostic {
	return NewV2VerifierDiagnostic(code,
		fmt.Sprintf("%s=%q does not match authority %q", field, observed, expected),
	).withProperty(field).
		withExpected(expected).
		withObserved(observed)
}

// verifyV2ManifestIdentityWithAnchor independently compares
// each identity field in the parsed manifest against the
// externally supplied topology anchor plus the frozen-plan
// authority.
func verifyV2ManifestIdentityWithAnchor(
	parsed parsedManifest,
	anchor V2ClosureTopology,
	frozenPlan V2FrozenPlanAuthority,
) V2VerifierDiagnostics {
	var diags V2VerifierDiagnostics

	if anchor.SubjectCommit != "" && parsed.SubjectCommit != anchor.SubjectCommit {
		diags = append(diags, v2FieldMismatch(
			V2VerifierManifestSubjectMismatch,
			"subject_commit", anchor.SubjectCommit, parsed.SubjectCommit,
		).withObjectOID(anchor.SubjectCommit))
	}
	if anchor.SubjectTree != "" && parsed.SubjectTree != anchor.SubjectTree {
		diags = append(diags, v2FieldMismatch(
			V2VerifierManifestSubjectTreeMismatch,
			"subject_tree", anchor.SubjectTree, parsed.SubjectTree,
		).withObjectOID(anchor.SubjectTree))
	}
	if anchor.FreezeCommit != "" && parsed.FreezeCommit != anchor.FreezeCommit {
		diags = append(diags, v2FieldMismatch(
			V2VerifierManifestFreezeMismatch,
			"freeze_commit", anchor.FreezeCommit, parsed.FreezeCommit,
		).withObjectOID(anchor.FreezeCommit))
	}
	if anchor.FreezeTree != "" && parsed.FreezeTree != anchor.FreezeTree {
		diags = append(diags, v2FieldMismatch(
			V2VerifierManifestFreezeTreeMismatch,
			"freeze_tree", anchor.FreezeTree, parsed.FreezeTree,
		).withObjectOID(anchor.FreezeTree))
	}
	if anchor.SubjectTree != "" && parsed.ExecutionTree != anchor.SubjectTree {
		diags = append(diags, v2FieldMismatch(
			V2VerifierManifestExecutionTreeMismatch,
			"execution_tree", anchor.SubjectTree, parsed.ExecutionTree,
		).withObjectOID(anchor.SubjectTree))
	}

	if parsed.PlanPath != frozenPlan.Path {
		diags = append(diags, v2FieldMismatch(
			V2VerifierManifestPlanPathMismatch,
			"plan_path", frozenPlan.Path, parsed.PlanPath,
		))
	}
	if parsed.PlanBlob != frozenPlan.BlobOID {
		diags = append(diags, v2FieldMismatch(
			V2VerifierManifestPlanBlobMismatch,
			"plan_blob", frozenPlan.BlobOID, parsed.PlanBlob,
		).withObjectOID(frozenPlan.BlobOID))
	}
	if parsed.PlanSHA256 != frozenPlan.BlobSHA256 {
		diags = append(diags, v2FieldMismatch(
			V2VerifierManifestPlanSHA256Mismatch,
			"plan_sha256", frozenPlan.BlobSHA256, parsed.PlanSHA256,
		).withObjectOID(frozenPlan.BlobOID))
	}
	return diags
}

// verifyV2ManifestBinaryIdentity classifies the committed
// manifest's binary identity as a structural assertion.
func verifyV2ManifestBinaryIdentity(id parsedBinaryIdentity) V2VerifierDiagnostics {
	var diags V2VerifierDiagnostics
	if strings.TrimSpace(id.Path) == "" {
		diag := NewV2VerifierDiagnostic(
			V2VerifierManifestBinaryIdentityInvalid,
			"binary path is empty",
		)
		diag.PropertyName = "leamas_binary_identity.path"
		diags = append(diags, diag)
	}
	if !sha256Pattern.MatchString(id.SHA256) {
		diag := NewV2VerifierDiagnostic(
			V2VerifierManifestBinaryIdentityInvalid,
			"binary SHA-256 must be 64 lowercase hexadecimal characters",
		)
		diag.PropertyName = "leamas_binary_identity.sha256"
		diag.Observed = id.SHA256
		diags = append(diags, diag)
	}
	if !sha1Pattern.MatchString(id.VCSRevision) {
		diag := NewV2VerifierDiagnostic(
			V2VerifierManifestBinaryIdentityInvalid,
			"binary VCS revision must be a full 40-character lowercase Git OID",
		)
		diag.PropertyName = "leamas_binary_identity.vcs_revision"
		diag.Observed = id.VCSRevision
		diags = append(diags, diag)
	}
	if id.VCSModified {
		diag := NewV2VerifierDiagnostic(
			V2VerifierManifestBinaryIdentityInvalid,
			"binary VCS modified must be false for committed manifest assertions",
		)
		diag.PropertyName = "leamas_binary_identity.vcs_modified"
		diag.Observed = "true"
		diags = append(diags, diag)
	}
	if strings.TrimSpace(id.LeamasVersion) == "" {
		diag := NewV2VerifierDiagnostic(
			V2VerifierManifestBinaryIdentityInvalid,
			"binary Leamas version must be nonempty",
		)
		diag.PropertyName = "leamas_binary_identity.leamas_version"
		diags = append(diags, diag)
	}
	return diags
}

// verifyV2FrozenPlanInventory parses the frozen plan bytes
// via the production plan contract.
func verifyV2FrozenPlanInventory(rawPlan []byte) ([]PlanCheck, V2VerifierDiagnostics) {
	if len(rawPlan) == 0 {
		return nil, V2VerifierDiagnostics{NewV2VerifierDiagnostic(
			V2VerifierFrozenPlanInvalid,
			"frozen plan bytes are empty",
		)}
	}
	plan, err := DecodePlan(rawPlan)
	if err != nil {
		diag := NewV2VerifierDiagnostic(
			V2VerifierFrozenPlanInvalid,
			fmt.Sprintf("frozen plan did not parse via the production contract: %s", err.Error()),
		)
		diag.ObjectPath = "plan_bytes"
		diag.Detail = err.Error()
		return nil, V2VerifierDiagnostics{diag}
	}
	canonical := append([]PlanCheck(nil), plan.Checks...)
	sort.SliceStable(canonical, func(i, j int) bool {
		return canonical[i].ID < canonical[j].ID
	})
	return canonical, nil
}

// verifyV2ManifestCheckBijection validates that the
// manifest's CheckResults form a bijection against the
// frozen-plan check inventory.
func verifyV2ManifestCheckBijection(
	planChecks []PlanCheck,
	parsedResults []parsedCheckResult,
) (bool, []V2CheckResult, V2VerifierDiagnostics) {
	if len(planChecks) == 0 {
		return false, nil, V2VerifierDiagnostics{NewV2VerifierDiagnostic(
			V2VerifierManifestCheckResultsInvalid,
			"frozen plan contains no checks",
		)}
	}
	if len(parsedResults) != len(planChecks) {
		return false, nil, V2VerifierDiagnostics{NewV2VerifierDiagnostic(
			V2VerifierManifestCheckResultBijectionFailed,
			fmt.Sprintf("manifest check_results length %d does not match plan checks length %d",
				len(parsedResults), len(planChecks)),
		)}
	}

	planByID := make(map[string]PlanCheck, len(planChecks))
	for _, p := range planChecks {
		planByID[p.ID] = p
	}

	seen := make(map[string]bool, len(parsedResults))
	mapped := make([]V2CheckResult, 0, len(planChecks))
	for _, r := range parsedResults {
		// duplicate
		if seen[r.ID] {
			diag := NewV2VerifierDiagnostic(
				V2VerifierManifestCheckResultBijectionFailed,
				fmt.Sprintf("duplicate manifest check_results entry %q", r.ID),
			)
			diag.ObjectOID = r.ID
			return false, nil, V2VerifierDiagnostics{diag}
		}
		seen[r.ID] = true

		plan, ok := planByID[r.ID]
		// unknown
		if !ok {
			diag := NewV2VerifierDiagnostic(
				V2VerifierManifestUnknownCheckID,
				fmt.Sprintf("manifest check_results entry %q is not in the frozen plan", r.ID),
			)
			diag.ObjectOID = r.ID
			return false, nil, V2VerifierDiagnostics{diag}
		}

		// mode mutation
		if r.Mode != plan.Mode {
			diag := NewV2VerifierDiagnostic(
				V2VerifierManifestCheckResultBijectionFailed,
				fmt.Sprintf("manifest check_results[%q].mode=%q does not match plan mode %q",
					r.ID, r.Mode, plan.Mode),
			)
			diag.ObjectOID = r.ID
			diag.Expected = plan.Mode
			diag.Observed = r.Mode
			return false, nil, V2VerifierDiagnostics{diag}
		}

		if r.Mode != CheckModeRun && r.Mode != CheckModeExclude {
			diag := NewV2VerifierDiagnostic(
				V2VerifierManifestCheckResultBijectionFailed,
				fmt.Sprintf("manifest check_results[%q].mode=%q is not in {run,exclude}", r.ID, r.Mode),
			)
			diag.ObjectOID = r.ID
			return false, nil, V2VerifierDiagnostics{diag}
		}
		if r.Outcome == "" {
			diag := NewV2VerifierDiagnostic(
				V2VerifierManifestCheckResultsInvalid,
				fmt.Sprintf("manifest check_results[%q].outcome is empty", r.ID),
			)
			diag.ObjectOID = r.ID
			return false, nil, V2VerifierDiagnostics{diag}
		}
		if _, ok := jsonNumberToInteger(r.DurationMS); !ok {
			diag := NewV2VerifierDiagnostic(
				V2VerifierManifestCheckResultsInvalid,
				fmt.Sprintf("manifest check_results[%q].duration_ms is not an integer", r.ID),
			)
			diag.ObjectOID = r.ID
			return false, nil, V2VerifierDiagnostics{diag}
		}
		durationInt, _ := jsonNumberToInteger(r.DurationMS)
		if int64(durationInt) < 0 {
			diag := NewV2VerifierDiagnostic(
				V2VerifierManifestCheckResultsInvalid,
				fmt.Sprintf("manifest check_results[%q].duration_ms is negative", r.ID),
			)
			diag.ObjectOID = r.ID
			return false, nil, V2VerifierDiagnostics{diag}
		}
		if r.ExecutionClassification == "" {
			diag := NewV2VerifierDiagnostic(
				V2VerifierManifestCheckResultsInvalid,
				fmt.Sprintf("manifest check_results[%q].execution_classification is empty", r.ID),
			)
			diag.ObjectOID = r.ID
			return false, nil, V2VerifierDiagnostics{diag}
		}
		if r.CleanupStatus == "" {
			diag := NewV2VerifierDiagnostic(
				V2VerifierManifestCheckResultsInvalid,
				fmt.Sprintf("manifest check_results[%q].cleanup_status is empty", r.ID),
			)
			diag.ObjectOID = r.ID
			return false, nil, V2VerifierDiagnostics{diag}
		}

		// Exclude-mode structural invariants.
		if r.Mode == CheckModeExclude {
			if r.Outcome != v2CheckOutcomeExcluded {
				diag := NewV2VerifierDiagnostic(
					V2VerifierManifestCheckResultsInvalid,
					fmt.Sprintf("exclude check_results[%q].outcome=%q must be %q",
						r.ID, r.Outcome, v2CheckOutcomeExcluded),
				)
				diag.ObjectOID = r.ID
				return false, nil, V2VerifierDiagnostics{diag}
			}
			if r.ExitCode != nil {
				diag := NewV2VerifierDiagnostic(
					V2VerifierManifestCheckResultsInvalid,
					fmt.Sprintf("exclude check_results[%q] must not carry an exit_code", r.ID),
				)
				diag.ObjectOID = r.ID
				return false, nil, V2VerifierDiagnostics{diag}
			}
			excludeDuration, _ := jsonNumberToInteger(r.DurationMS)
			if int64(excludeDuration) != 0 {
				diag := NewV2VerifierDiagnostic(
					V2VerifierManifestCheckResultsInvalid,
					fmt.Sprintf("exclude check_results[%q].duration_ms must be 0", r.ID),
				)
				diag.ObjectOID = r.ID
				return false, nil, V2VerifierDiagnostics{diag}
			}
			if r.ExecutionClassification != v2ExecutionExcludedByPlan {
				diag := NewV2VerifierDiagnostic(
					V2VerifierManifestCheckResultsInvalid,
					fmt.Sprintf("exclude check_results[%q].execution_classification=%q must be %q",
						r.ID, r.ExecutionClassification, v2ExecutionExcludedByPlan),
				)
				diag.ObjectOID = r.ID
				return false, nil, V2VerifierDiagnostics{diag}
			}
			if r.CleanupStatus != CleanupNotRequired {
				diag := NewV2VerifierDiagnostic(
					V2VerifierManifestCheckResultsInvalid,
					fmt.Sprintf("exclude check_results[%q].cleanup_status=%q must be %q",
						r.ID, r.CleanupStatus, CleanupNotRequired),
				)
				diag.ObjectOID = r.ID
				return false, nil, V2VerifierDiagnostics{diag}
			}
		}

		durationEntry, _ := jsonNumberToInteger(r.DurationMS)
		entry := V2CheckResult{
			ID:                      r.ID,
			Mode:                    r.Mode,
			Outcome:                 r.Outcome,
			DurationMS:              int64(durationEntry),
			ExecutionClassification: r.ExecutionClassification,
			CleanupStatus:           r.CleanupStatus,
			Detail:                  r.Detail,
		}
		if r.ExitCode != nil {
			exitInt, ok := jsonNumberToInteger(*r.ExitCode)
			if !ok {
				diag := NewV2VerifierDiagnostic(
					V2VerifierManifestCheckResultsInvalid,
					fmt.Sprintf("manifest check_results[%q].exit_code is not an integer", r.ID),
				)
				diag.ObjectOID = r.ID
				return false, nil, V2VerifierDiagnostics{diag}
			}
			v := exitInt
			entry.ExitCode = &v
		}
		if len(r.Evidence) > 0 {
			refs := make([]V2EvidenceReference, 0, len(r.Evidence))
			for _, e := range r.Evidence {
				byteCount, ok := jsonNumberToInteger(e.ByteCount)
				if !ok {
					diag := NewV2VerifierDiagnostic(
						V2VerifierManifestCheckResultsInvalid,
						fmt.Sprintf("manifest check_results[%q].evidence[%q].byte_count is not an integer",
							r.ID, e.LogicalName),
					)
					diag.ObjectOID = r.ID
					return false, nil, V2VerifierDiagnostics{diag}
				}
				refs = append(refs, V2EvidenceReference{
					LogicalName:  e.LogicalName,
					MediaType:    e.MediaType,
					SHA256:       e.SHA256,
					ByteCount:    int64(byteCount),
					Availability: e.Availability,
				})
			}
			entry.Evidence = refs
		}
		mapped = append(mapped, entry)
	}

	planOrder := make(map[string]int, len(planChecks))
	for i, p := range planChecks {
		planOrder[p.ID] = i
	}
	sort.SliceStable(mapped, func(i, j int) bool {
		return planOrder[mapped[i].ID] < planOrder[mapped[j].ID]
	})

	return true, mapped, nil
}

// verifyV2ManifestRunSuccessIntegrity enforces the aggregate
// success invariant.
func verifyV2ManifestRunSuccessIntegrity(
	planChecks []PlanCheck,
	mapped []V2CheckResult,
) (bool, V2VerifierDiagnostics) {
	var diags V2VerifierDiagnostics

	planMode := make(map[string]string, len(planChecks))
	for _, p := range planChecks {
		planMode[p.ID] = p.Mode
	}

	for _, r := range mapped {
		mode := planMode[r.ID]
		switch mode {
		case CheckModeRun:
			if r.Outcome == CheckStatusFail {
				diag := NewV2VerifierDiagnostic(
					V2VerifierManifestUnsuccessfulRun,
					fmt.Sprintf("run-mode check %q has outcome %q", r.ID, CheckStatusFail),
				)
				diag.ObjectOID = r.ID
				diags = append(diags, diag)
				continue
			}
			if r.Outcome != CheckStatusPass {
				diag := NewV2VerifierDiagnostic(
					V2VerifierManifestCheckResultsInvalid,
					fmt.Sprintf("run-mode check %q has non-success outcome %q", r.ID, r.Outcome),
				)
				diag.ObjectOID = r.ID
				diags = append(diags, diag)
				continue
			}
			if r.CleanupStatus == CleanupFailed {
				diag := NewV2VerifierDiagnostic(
					V2VerifierManifestUnsuccessfulRun,
					fmt.Sprintf("run-mode check %q has cleanup_status=%q", r.ID, CleanupFailed),
				)
				diag.ObjectOID = r.ID
				diags = append(diags, diag)
				continue
			}
			if r.CleanupStatus != CleanupPass {
				diag := NewV2VerifierDiagnostic(
					V2VerifierManifestCheckResultsInvalid,
					fmt.Sprintf("run-mode check %q has unexpected cleanup_status=%q", r.ID, r.CleanupStatus),
				)
				diag.ObjectOID = r.ID
				diags = append(diags, diag)
				continue
			}
			switch r.ExecutionClassification {
			case v2ExecutionTimeout, v2ExecutionCancelled, v2ExecutionOutputOverflow,
				v2ExecutionOutputTruncated, v2ExecutionOutputIncomplete:
				diag := NewV2VerifierDiagnostic(
					V2VerifierManifestUnsuccessfulRun,
					fmt.Sprintf("run-mode check %q has execution_classification=%q classified as success",
						r.ID, r.ExecutionClassification),
				)
				diag.ObjectOID = r.ID
				diags = append(diags, diag)
			}
			if r.ExitCode == nil {
				diag := NewV2VerifierDiagnostic(
					V2VerifierManifestCheckResultsInvalid,
					fmt.Sprintf("run-mode check %q has no exit_code", r.ID),
				)
				diag.ObjectOID = r.ID
				diags = append(diags, diag)
				continue
			}
			if *r.ExitCode != 0 {
				diag := NewV2VerifierDiagnostic(
					V2VerifierManifestUnsuccessfulRun,
					fmt.Sprintf("run-mode check %q has non-zero exit_code %d", r.ID, *r.ExitCode),
				)
				diag.ObjectOID = r.ID
				diags = append(diags, diag)
			}
		case CheckModeExclude:
			if r.Outcome != v2CheckOutcomeExcluded {
				diag := NewV2VerifierDiagnostic(
					V2VerifierManifestCheckResultsInvalid,
					fmt.Sprintf("exclude-mode check %q has non-excluded outcome %q", r.ID, r.Outcome),
				)
				diag.ObjectOID = r.ID
				diags = append(diags, diag)
			}
		default:
			diag := NewV2VerifierDiagnostic(
				V2VerifierManifestCheckResultsInvalid,
				fmt.Sprintf("check %q has unknown plan mode %q", r.ID, mode),
			)
			diag.ObjectOID = r.ID
			diags = append(diags, diag)
		}
	}

	return len(diags) == 0, diags
}

// v2ExecutionTimeout / v2ExecutionCancelled /
// v2ExecutionOutputOverflow mirror the v2 runner's
// classification vocabulary. The truncated / incomplete
// classification constants live in
// closure_protocol_v2_results.go and are reused from this
// package.
const (
	v2ExecutionTimeout        = "timeout"
	v2ExecutionCancelled      = "cancelled"
	v2ExecutionOutputOverflow = "output_overflow"
)

// copyVerifierDiagnostics returns a defensive copy of the
// supplied diagnostics slice.
func copyVerifierDiagnostics(in V2VerifierDiagnostics) V2VerifierDiagnostics {
	out := make(V2VerifierDiagnostics, len(in))
	copy(out, in)
	return out
}
