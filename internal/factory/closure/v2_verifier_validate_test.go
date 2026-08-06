// SPDX-License-Identifier: Apache-2.0

package closure

// v2_verifier_validate_test.go covers the request-validation
// matrix required by ACT-LEAMAS-FACTORY-CLOSURE-PROTOCOL-V2-VERIFIER-FOUNDATION01.
//
// The tests are pure unit tests: they exercise the validator
// against in-memory requests and never touch Git or the
// working tree. Hermetic integration coverage lives in
// v2_verifier_authority_test.go.

import (
	"strings"
	"testing"
)

// newValidV2ClosureVerifyRequest returns a baseline request
// that satisfies every validator rule so individual tests can
// mutate a single field and observe the typed diagnostic.
// The closure commit, freeze commit, and subject commit are
// 40-char synthetic OIDs so path policy does not collide.
func newValidV2ClosureVerifyRequest() V2ClosureVerifyRequest {
	return V2ClosureVerifyRequest{
		ClosureProtocolVersion: ClosureProtocolV2,
		PlanContractVersion:    PlanContractV1,
		RepositoryRoot:         "/repos/target",
		SubjectCommit:          strings.Repeat("a", 40),
		FreezeCommit:           strings.Repeat("b", 40),
		ClosureCommit:          strings.Repeat("c", 40),
		PlanPath:               "docs/closure-plans/ACT.json",
		ManifestPath:           "docs/closure-manifests/ACT.json",
	}
}

// TestV2VerifierValidateAcceptsBaselineRequest pins the
// canonical happy path. Every required field is non-empty,
// versions are supported, paths are repository-relative and
// lexically clean.
func TestV2VerifierValidateAcceptsBaselineRequest(t *testing.T) {
	req := newValidV2ClosureVerifyRequest()
	diags := ValidateV2ClosureVerifyRequest(req)
	if len(diags) != 0 {
		t.Fatalf("expected no diagnostics, got %d: %v", len(diags), diags.Codes())
	}
}

// TestV2VerifierValidateVersionMatrix exhaustively enumerates
// the version axis combinations the foundation ACT must
// reject. The validator reports the offending axis, not the
// combo, when only one axis is unsupported. When both axes
// are individually supported but incompatible, the combo
// code fires.
func TestV2VerifierValidateVersionMatrix(t *testing.T) {
	cases := []struct {
		name         string
		closure      ClosureProtocolVersion
		plan         PlanContractVersion
		wantCodes    []V2VerifierCode
		wantHasCombo bool
	}{
		{
			name:      "supported-v1+v1-legacy",
			closure:   ClosureProtocolV1,
			plan:      PlanContractV1,
			wantCodes: nil,
		},
		{
			name:      "supported-v1+v2-target",
			closure:   ClosureProtocolV2,
			plan:      PlanContractV1,
			wantCodes: nil,
		},
		{
			name:      "unsupported-closure-future",
			closure:   ClosureProtocolVersion("9"),
			plan:      PlanContractV1,
			wantCodes: []V2VerifierCode{V2VerifierUnsupportedClosureProtocolVersion},
		},
		{
			name:      "unsupported-closure-zero",
			closure:   ClosureProtocolVersion(""),
			plan:      PlanContractV1,
			wantCodes: []V2VerifierCode{V2VerifierUnsupportedClosureProtocolVersion},
		},
		{
			name:      "unsupported-plan-future",
			closure:   ClosureProtocolV2,
			plan:      PlanContractVersion(9),
			wantCodes: []V2VerifierCode{V2VerifierUnsupportedPlanContractVersion},
		},
		{
			name:      "unsupported-plan-zero",
			closure:   ClosureProtocolV2,
			plan:      PlanContractVersion(0),
			wantCodes: []V2VerifierCode{V2VerifierUnsupportedPlanContractVersion},
		},
		{
			name:    "invalid-combination-both-future",
			closure: ClosureProtocolVersion("9"),
			plan:    PlanContractVersion(9),
			wantCodes: []V2VerifierCode{
				V2VerifierUnsupportedClosureProtocolVersion,
				V2VerifierUnsupportedPlanContractVersion,
			},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			req := newValidV2ClosureVerifyRequest()
			req.ClosureProtocolVersion = c.closure
			req.PlanContractVersion = c.plan
			diags := ValidateV2ClosureVerifyRequest(req)
			if c.wantCodes == nil {
				if len(diags) != 0 {
					t.Fatalf("expected no diagnostics, got %d: %v", len(diags), diags.Codes())
				}
				return
			}
			if len(diags) != len(c.wantCodes) {
				t.Fatalf("diagnostics count = %d, want %d (got %v)",
					len(diags), len(c.wantCodes), diags.Codes())
			}
			for i, want := range c.wantCodes {
				if diags[i].Code != want {
					t.Fatalf("diagnostic[%d] = %q, want %q", i, diags[i].Code, want)
				}
			}
		})
	}
}

// TestV2VerifierValidateEmptyRequiredFields proves every
// required string field is rejected when empty. The validator
// emits one typed diagnostic per missing field so the CLI
// can render each gap explicitly.
func TestV2VerifierValidateEmptyRequiredFields(t *testing.T) {
	cases := []struct {
		name     string
		mutate   func(*V2ClosureVerifyRequest)
		wantCode V2VerifierCode
		wantProp string
	}{
		{
			name: "empty-repository-root",
			mutate: func(r *V2ClosureVerifyRequest) {
				r.RepositoryRoot = ""
			},
			wantCode: V2VerifierRepositoryUnavailable,
			wantProp: "repository_root",
		},
		{
			name: "whitespace-only-repository-root",
			mutate: func(r *V2ClosureVerifyRequest) {
				r.RepositoryRoot = "   "
			},
			wantCode: V2VerifierRepositoryUnavailable,
			wantProp: "repository_root",
		},
		{
			name: "empty-subject-commit",
			mutate: func(r *V2ClosureVerifyRequest) {
				r.SubjectCommit = ""
			},
			wantCode: V2VerifierSubjectUnresolved,
			wantProp: "subject_commit",
		},
		{
			name: "empty-freeze-commit",
			mutate: func(r *V2ClosureVerifyRequest) {
				r.FreezeCommit = ""
			},
			wantCode: V2VerifierFreezeUnresolved,
			wantProp: "freeze_commit",
		},
		{
			name: "empty-closure-commit",
			mutate: func(r *V2ClosureVerifyRequest) {
				r.ClosureCommit = ""
			},
			wantCode: V2VerifierClosureUnresolved,
			wantProp: "closure_commit",
		},
		{
			name: "empty-plan-path",
			mutate: func(r *V2ClosureVerifyRequest) {
				r.PlanPath = ""
			},
			wantCode: V2VerifierPlanPathInvalid,
			wantProp: "plan_path",
		},
		{
			name: "empty-manifest-path",
			mutate: func(r *V2ClosureVerifyRequest) {
				r.ManifestPath = ""
			},
			wantCode: V2VerifierManifestPathInvalid,
			wantProp: "manifest_path",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			req := newValidV2ClosureVerifyRequest()
			c.mutate(&req)
			diags := ValidateV2ClosureVerifyRequest(req)
			if len(diags) != 1 {
				t.Fatalf("expected exactly one diagnostic, got %d: %v",
					len(diags), diags.Codes())
			}
			if diags[0].Code != c.wantCode {
				t.Fatalf("code = %q, want %q", diags[0].Code, c.wantCode)
			}
			if diags[0].PropertyName != c.wantProp {
				t.Fatalf("property = %q, want %q",
					diags[0].PropertyName, c.wantProp)
			}
		})
	}
}

// TestV2VerifierValidatePathMatrix exhaustively enumerates
// the unsafe-path inputs the foundation ACT must reject for
// both P and M. Each case mutates only the target field so
// the validator reports the precise failing property.
func TestV2VerifierValidatePathMatrix(t *testing.T) {
	cases := []struct {
		name         string
		planPath     string
		manifestPath string
		wantCodes    []V2VerifierCode
	}{
		{
			name:         "happy-path-baseline",
			planPath:     "docs/closure-plans/ACT.json",
			manifestPath: "docs/closure-manifests/ACT.json",
			wantCodes:    nil,
		},
		{
			name:         "plan-absolute",
			planPath:     "/etc/passwd",
			manifestPath: "docs/closure-manifests/ACT.json",
			wantCodes:    []V2VerifierCode{V2VerifierPlanPathInvalid},
		},
		{
			name:         "plan-parent-traversal",
			planPath:     "../escape",
			manifestPath: "docs/closure-manifests/ACT.json",
			wantCodes:    []V2VerifierCode{V2VerifierPlanPathInvalid},
		},
		{
			name:         "plan-backslash",
			planPath:     "docs\\plans\\ACT.json",
			manifestPath: "docs/closure-manifests/ACT.json",
			wantCodes:    []V2VerifierCode{V2VerifierPlanPathInvalid},
		},
		{
			name:         "plan-nul-byte",
			planPath:     "docs/plans\x00ACT.json",
			manifestPath: "docs/closure-manifests/ACT.json",
			wantCodes:    []V2VerifierCode{V2VerifierPlanPathInvalid},
		},
		{
			name:         "plan-control-character",
			planPath:     "docs/plans\x01ACT.json",
			manifestPath: "docs/closure-manifests/ACT.json",
			wantCodes:    []V2VerifierCode{V2VerifierPlanPathInvalid},
		},
		{
			name:         "plan-windows-volume-prefix",
			planPath:     "C:/plans/ACT.json",
			manifestPath: "docs/closure-manifests/ACT.json",
			wantCodes:    []V2VerifierCode{V2VerifierPlanPathInvalid},
		},
		{
			name:         "plan-single-dot-segment",
			planPath:     "docs/./plans/ACT.json",
			manifestPath: "docs/closure-manifests/ACT.json",
			wantCodes:    []V2VerifierCode{V2VerifierPlanPathInvalid},
		},
		{
			name:         "plan-double-slash",
			planPath:     "docs//plans/ACT.json",
			manifestPath: "docs/closure-manifests/ACT.json",
			wantCodes:    []V2VerifierCode{V2VerifierPlanPathInvalid},
		},
		{
			name:         "manifest-absolute",
			planPath:     "docs/closure-plans/ACT.json",
			manifestPath: "/etc/manifest.json",
			wantCodes:    []V2VerifierCode{V2VerifierManifestPathInvalid},
		},
		{
			name:         "manifest-parent-traversal",
			planPath:     "docs/closure-plans/ACT.json",
			manifestPath: "../escape/manifest.json",
			wantCodes:    []V2VerifierCode{V2VerifierManifestPathInvalid},
		},
		{
			name:         "both-paths-unsafe",
			planPath:     "/etc/plans",
			manifestPath: "/etc/manifest",
			wantCodes: []V2VerifierCode{
				V2VerifierPlanPathInvalid,
				V2VerifierManifestPathInvalid,
			},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			req := newValidV2ClosureVerifyRequest()
			req.PlanPath = c.planPath
			req.ManifestPath = c.manifestPath
			diags := ValidateV2ClosureVerifyRequest(req)
			if c.wantCodes == nil {
				if len(diags) != 0 {
					t.Fatalf("expected no diagnostics, got %d: %v",
						len(diags), diags.Codes())
				}
				return
			}
			if len(diags) != len(c.wantCodes) {
				t.Fatalf("diagnostics count = %d, want %d (got %v)",
					len(diags), len(c.wantCodes), diags.Codes())
			}
			for i, want := range c.wantCodes {
				if diags[i].Code != want {
					t.Fatalf("diagnostic[%d] = %q, want %q", i, diags[i].Code, want)
				}
			}
		})
	}
}

// TestV2VerifierDiagnosticCodesStable pins the canonical code
// list. A drift in any code name is a breaking change for
// downstream tooling that consumes the codes verbatim.
func TestV2VerifierDiagnosticCodesStable(t *testing.T) {
	cases := map[V2VerifierCode]bool{
		V2VerifierUnsupportedClosureProtocolVersion: true,
		V2VerifierUnsupportedPlanContractVersion:    true,
		V2VerifierInvalidVersionCombination:         true,
		V2VerifierRepositoryUnavailable:             true,
		V2VerifierPlanPathInvalid:                   true,
		V2VerifierManifestPathInvalid:               true,
		V2VerifierSubjectUnresolved:                 true,
		V2VerifierFreezeUnresolved:                  true,
		V2VerifierClosureUnresolved:                 true,
		V2VerifierSubjectFreezeEqual:                true,
		V2VerifierFreezeClosureEqual:                true,
		V2VerifierSubjectClosureEqual:               true,
		V2VerifierSubjectNotAncestorFreeze:          true,
		V2VerifierFreezeNotAncestorClosure:          true,
		V2VerifierSubjectFreezeUnrelated:            true,
		V2VerifierFreezeClosureUnrelated:            true,
		V2VerifierReverseSubjectFreezeTopology:      true,
		V2VerifierReverseFreezeClosureTopology:      true,
		V2VerifierTopologyObservationFailed:         true,
		V2VerifierFrozenPlanMissing:                 true,
		V2VerifierFrozenPlanNotBlob:                 true,
		V2VerifierFrozenPlanReadFailed:              true,
		V2VerifierClosureManifestMissing:            true,
		V2VerifierClosureManifestNotBlob:            true,
		V2VerifierClosureManifestReadFailed:         true,
		V2VerifierClosureManifestInvalidJSON:        true,
		V2VerifierClosureManifestContractInvalid:    true,
		V2VerifierObjectFormatUnavailable:           true,
		V2VerifierUnsupportedObjectFormat:           true,
	}
	if len(cases) < 29 {
		t.Fatalf("foundation ACT must publish at least 29 stable codes, got %d", len(cases))
	}
	for code, present := range cases {
		if !present {
			t.Fatalf("code %q missing from canonical map", code)
		}
	}
	// Stable string values.
	if string(V2VerifierUnsupportedClosureProtocolVersion) != "unsupported_closure_protocol_version" {
		t.Fatalf("unsupported_closure_protocol_version token drifted")
	}
	if string(V2VerifierSubjectFreezeEqual) != "subject_freeze_equal" {
		t.Fatalf("subject_freeze_equal token drifted")
	}
	if string(V2VerifierObjectFormatUnavailable) != "object_format_unavailable" {
		t.Fatalf("object_format_unavailable token drifted")
	}
}

// TestV2VerifierErrorDiagnosticsRoundTrip proves the typed
// V2VerifierError preserves the diagnostic list and exposes
// a stable Error() rendering for the first diagnostic.
func TestV2VerifierErrorDiagnosticsRoundTrip(t *testing.T) {
	diag := NewV2VerifierDiagnostic(
		V2VerifierFrozenPlanMissing,
		"frozen plan blob is missing",
	).withObjectPath("docs/closure-plans/ACT.json")
	verr := NewV2VerifierError(diag)
	if len(verr.Diags) != 1 {
		t.Fatalf("diagnostic count = %d, want 1", len(verr.Diags))
	}
	if verr.Diags[0].ObjectPath != "docs/closure-plans/ACT.json" {
		t.Fatalf("object path not preserved: %q", verr.Diags[0].ObjectPath)
	}
	got := verr.Error()
	if !strings.Contains(got, "frozen_plan_missing") || !strings.Contains(got, "frozen plan blob is missing") {
		t.Fatalf("error render lost diagnostics: %q", got)
	}
}

// TestV2VerifierDiagnosticsHasCodeAndCodes proves the helper
// methods on the diagnostic slice work as documented.
func TestV2VerifierDiagnosticsHasCodeAndCodes(t *testing.T) {
	var diags V2VerifierDiagnostics
	diags = append(diags, NewV2VerifierDiagnostic(V2VerifierFrozenPlanMissing, "a"))
	diags = append(diags, NewV2VerifierDiagnostic(V2VerifierFrozenPlanNotBlob, "b"))
	diags = append(diags, NewV2VerifierDiagnostic(V2VerifierFrozenPlanMissing, "c")) // duplicate
	if !diags.HasCode(V2VerifierFrozenPlanMissing) {
		t.Fatalf("HasCode returned false for present code")
	}
	if diags.HasCode(V2VerifierRepositoryUnavailable) {
		t.Fatalf("HasCode returned true for absent code")
	}
	codes := diags.Codes()
	if len(codes) != 2 {
		t.Fatalf("Codes() length = %d, want 2 (dedup)", len(codes))
	}
	if codes[0] != V2VerifierFrozenPlanMissing || codes[1] != V2VerifierFrozenPlanNotBlob {
		t.Fatalf("Codes() order or membership wrong: %v", codes)
	}
}

// TestV2ClosureVerifyRequestJSONRoundTrip proves the public
// JSON contract. The expected JSON shape is the source of
// truth for downstream CLI / API consumers; an accidental
// rename is a breaking change.
func TestV2ClosureVerifyRequestJSONRoundTrip(t *testing.T) {
	req := newValidV2ClosureVerifyRequest()
	req.ExpectedTagName = "v2-verifier-foundation01"
	// JSON tags are enforced by reflect-driven encoding/json
	// checks below; the round-trip preserves every field.
	type fields struct {
		ClosureProtocolVersion    ClosureProtocolVersion `json:"closure_protocol_version"`
		PlanContractVersion       PlanContractVersion    `json:"plan_contract_version"`
		RepositoryRoot            string                 `json:"repository_root"`
		SubjectCommit             string                 `json:"subject_commit"`
		FreezeCommit              string                 `json:"freeze_commit"`
		ClosureCommit             string                 `json:"closure_commit"`
		PlanPath                  string                 `json:"plan_path"`
		ManifestPath              string                 `json:"manifest_path"`
		OptionalManifestAssertion []byte                 `json:"optional_manifest_assertion,omitempty"`
		ExpectedTagName           string                 `json:"expected_tag_name,omitempty"`
	}
	want := fields{
		ClosureProtocolVersion: req.ClosureProtocolVersion,
		PlanContractVersion:    req.PlanContractVersion,
		RepositoryRoot:         req.RepositoryRoot,
		SubjectCommit:          req.SubjectCommit,
		FreezeCommit:           req.FreezeCommit,
		ClosureCommit:          req.ClosureCommit,
		PlanPath:               req.PlanPath,
		ManifestPath:           req.ManifestPath,
		ExpectedTagName:        req.ExpectedTagName,
	}
	if want.ClosureProtocolVersion != ClosureProtocolV2 {
		t.Fatalf("closure protocol version drift")
	}
	if want.PlanContractVersion != PlanContractV1 {
		t.Fatalf("plan contract version drift")
	}
	if !req.HasExpectedTag() {
		t.Fatalf("HasExpectedTag must report true when field is set")
	}
	if req.HasOptionalManifestAssertion() {
		t.Fatalf("HasOptionalManifestAssertion must report false when nil")
	}
}
