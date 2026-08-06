// SPDX-License-Identifier: Apache-2.0

package closure

// v2_verifier_manifest_determinism_test.go covers Phase 8 of
// ACT-LEAMAS-FACTORY-CLOSURE-PROTOCOL-V2-VERIFIER-MANIFEST-RESULTS01:
// the deterministic V2ClosureVerification result model.
//
// The test verifies that:
//
//   - identical inputs produce byte-identical V2ClosureVerification
//     JSON output across two independent constructions
//   - the Valid flag is computed from the canonical validity
//     predicate (topology valid AND manifest valid AND
//     result-set valid AND diagnostics empty AND required
//     identity fields present)

import (
	"encoding/json"
	"strings"
	"testing"
)

// TestV2VerifierVerificationDeterminism proves the
// V2ClosureVerification result is deterministic: identical
// build inputs always produce byte-identical JSON output.
func TestV2VerifierVerificationDeterminism(t *testing.T) {
	cases := []struct {
		name  string
		build V2VerificationBuild
	}{
		{
			name: "happy_path",
			build: V2VerificationBuild{
				ClosureProtocolVersion: ClosureProtocolV2,
				PlanContractVersion:    PlanContractV1,
				RepositoryRoot:         "/repo",
				SubjectCommit:          strings.Repeat("a", 40),
				SubjectTree:            strings.Repeat("b", 40),
				FreezeCommit:           strings.Repeat("c", 40),
				FreezeTree:             strings.Repeat("d", 40),
				ClosureCommit:          strings.Repeat("e", 40),
				ClosureTree:            strings.Repeat("f", 40),
				PlanPath:               "docs/closure-plans/PLAN.json",
				PlanBlob:               strings.Repeat("1", 40),
				PlanSHA256:             strings.Repeat("2", 64),
				ManifestPath:           "docs/closure-manifests/MANIFEST.json",
				ManifestBlob:           strings.Repeat("3", 40),
				ManifestSHA256:         strings.Repeat("4", 64),
				TopologyValid:          true,
				ManifestValid:          true,
				ResultSetValid:         true,
			},
		},
		{
			name: "with_diagnostics",
			build: V2VerificationBuild{
				ClosureProtocolVersion: ClosureProtocolV2,
				PlanContractVersion:    PlanContractV1,
				RepositoryRoot:         "/repo",
				SubjectCommit:          strings.Repeat("a", 40),
				SubjectTree:            strings.Repeat("b", 40),
				FreezeCommit:           strings.Repeat("c", 40),
				FreezeTree:             strings.Repeat("d", 40),
				ClosureCommit:          strings.Repeat("e", 40),
				ClosureTree:            strings.Repeat("f", 40),
				PlanPath:               "docs/closure-plans/PLAN.json",
				PlanBlob:               strings.Repeat("1", 40),
				PlanSHA256:             strings.Repeat("2", 64),
				ManifestPath:           "docs/closure-manifests/MANIFEST.json",
				ManifestBlob:           strings.Repeat("3", 40),
				ManifestSHA256:         strings.Repeat("4", 64),
				TopologyValid:          true,
				ManifestValid:          false,
				ResultSetValid:         true,
				Diagnostics: V2VerifierDiagnostics{
					NewV2VerifierDiagnostic(
						V2VerifierManifestSubjectMismatch,
						"subject_commit does not match authority",
					),
					NewV2VerifierDiagnostic(
						V2VerifierManifestPlanBlobMismatch,
						"plan_blob does not match authority",
					),
				},
			},
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			first := NewV2ClosureVerification(tc.build)
			second := NewV2ClosureVerification(tc.build)

			firstJSON, err := json.Marshal(first)
			if err != nil {
				t.Fatalf("marshal first: %v", err)
			}
			secondJSON, err := json.Marshal(second)
			if err != nil {
				t.Fatalf("marshal second: %v", err)
			}
			if string(firstJSON) != string(secondJSON) {
				t.Fatalf("verification result is not deterministic\nfirst:  %s\nsecond: %s",
					string(firstJSON), string(secondJSON))
			}

			expectedValid := tc.build.TopologyValid &&
				tc.build.ManifestValid &&
				tc.build.ResultSetValid &&
				len(tc.build.Diagnostics) == 0 &&
				tc.build.SubjectCommit != "" &&
				tc.build.SubjectTree != "" &&
				tc.build.FreezeCommit != "" &&
				tc.build.FreezeTree != "" &&
				tc.build.ClosureCommit != "" &&
				tc.build.ClosureTree != "" &&
				tc.build.PlanPath != "" &&
				tc.build.PlanBlob != "" &&
				tc.build.PlanSHA256 != "" &&
				tc.build.ManifestPath != "" &&
				tc.build.ManifestBlob != "" &&
				tc.build.ManifestSHA256 != ""
			if first.Valid != expectedValid {
				t.Fatalf("Valid=%v, want %v (manifest_valid=%v, result_set_valid=%v, diags=%d)",
					first.Valid, expectedValid, tc.build.ManifestValid,
					tc.build.ResultSetValid, len(tc.build.Diagnostics))
			}
		})
	}
}
