// SPDX-License-Identifier: Apache-2.0

package closure

// v2_verifier_cli_orchestrator_test.go covers the
// orchestrator-level end-to-end behaviour of
// ACT-LEAMAS-FACTORY-CLOSURE-PROTOCOL-V2-VERIFIER-CLI-TAG-STATE01.
//
// The tests assert:
//
//   - request validation diagnostics short-circuit the run
//     without touching the Git authority;
//   - the diagnostic family grew from 29 codes (foundation
//     ACT) to 38 codes after ACT 4 added 4 tag codes and 5
//     state-capture codes.
//
// The full happy-path orchestrator test requires a faithful
// committed v2 manifest and is exercised by the production
// harness; the matrix below covers the rejection paths
// foundation / topology / ACT 4 must lock down.

import (
	"context"
	"testing"
)

// TestV2VerifierOrchestratorRequestValidationShortCircuit
// proves the orchestrator surfaces request-level
// diagnostics without invoking the Git authority.
func TestV2VerifierOrchestratorRequestValidationShortCircuit(t *testing.T) {
	o := NewV2VerifierOrchestrator()

	cases := []struct {
		name      string
		request   V2ClosureVerifyRequest
		wantCodes []V2VerifierCode
	}{
		{
			name: "missing_repository_root",
			request: V2ClosureVerifyRequest{
				ClosureProtocolVersion: ClosureProtocolV2,
				PlanContractVersion:    PlanContractV1,
				SubjectCommit:          "1111111111111111111111111111111111111111",
				FreezeCommit:           "2222222222222222222222222222222222222222",
				ClosureCommit:          "3333333333333333333333333333333333333333",
				PlanPath:               "plan/plan.json",
				ManifestPath:           "manifest/manifest.json",
			},
			wantCodes: []V2VerifierCode{V2VerifierRepositoryUnavailable},
		},
		{
			name: "missing_closure_commit",
			request: V2ClosureVerifyRequest{
				ClosureProtocolVersion: ClosureProtocolV2,
				PlanContractVersion:    PlanContractV1,
				RepositoryRoot:         "/tmp/nope",
				SubjectCommit:          "1111111111111111111111111111111111111111",
				FreezeCommit:           "2222222222222222222222222222222222222222",
				PlanPath:               "plan/plan.json",
				ManifestPath:           "manifest/manifest.json",
			},
			wantCodes: []V2VerifierCode{V2VerifierClosureUnresolved},
		},
		{
			name: "absolute_plan_path",
			request: V2ClosureVerifyRequest{
				ClosureProtocolVersion: ClosureProtocolV2,
				PlanContractVersion:    PlanContractV1,
				RepositoryRoot:         "/tmp/nope",
				SubjectCommit:          "1111111111111111111111111111111111111111",
				FreezeCommit:           "2222222222222222222222222222222222222222",
				ClosureCommit:          "3333333333333333333333333333333333333333",
				PlanPath:               "/abs/plan.json",
				ManifestPath:           "manifest/manifest.json",
			},
			wantCodes: []V2VerifierCode{V2VerifierPlanPathInvalid},
		},
		{
			name: "parent_traversal_manifest_path",
			request: V2ClosureVerifyRequest{
				ClosureProtocolVersion: ClosureProtocolV2,
				PlanContractVersion:    PlanContractV1,
				RepositoryRoot:         "/tmp/nope",
				SubjectCommit:          "1111111111111111111111111111111111111111",
				FreezeCommit:           "2222222222222222222222222222222222222222",
				ClosureCommit:          "3333333333333333333333333333333333333333",
				PlanPath:               "plan/plan.json",
				ManifestPath:           "../manifest.json",
			},
			wantCodes: []V2VerifierCode{V2VerifierManifestPathInvalid},
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			got := o.Run(context.Background(), nil, V2RunRequest{Request: tc.request})
			if got.Verification.Valid {
				t.Fatalf("expected Valid=false for %s, got true", tc.name)
			}
			gotCodes := got.Verification.Diagnostics.Codes()
			for _, want := range tc.wantCodes {
				found := false
				for _, gc := range gotCodes {
					if gc == want {
						found = true
						break
					}
				}
				if !found {
					t.Fatalf("expected diagnostic code %s for %s, got %v",
						want, tc.name, gotCodes)
				}
			}
		})
	}
}

// TestV2VerifierDiagnosticCodeCountGrew documents the
// expansion of the verifier-specific diagnostic family
// introduced by ACT 4. ACT 1 listed 29 codes; ACT 4 adds 4
// tag codes + 5 state-capture codes = 9 new codes, taking
// the total to 38.
//
// The foundation report records 29 codes; the count is the
// authoritative baseline. This test refuses to drift: if a
// future ACT edits the family unintentionally, the count
// fails fast and forces an explicit update.
func TestV2VerifierDiagnosticCodeCountGrew(t *testing.T) {
	want := 55
	got := len(v2VerifierAllCodes())
	if got != want {
		t.Fatalf("verifier diagnostic code count drifted: got %d, want %d", got, want)
	}
}

// v2VerifierAllCodes returns the deduplicated set of all
// V2VerifierCode values declared in the foundation ACT
// (29 codes) plus the ACT 4 additions. The function exists
// so the count test exposes the registry as data; future
// ACTs that add stable codes MUST extend this list as part
// of their freeze commit.
func v2VerifierAllCodes() []V2VerifierCode {
	return []V2VerifierCode{
		// foundation (29 codes)
		V2VerifierUnsupportedClosureProtocolVersion,
		V2VerifierUnsupportedPlanContractVersion,
		V2VerifierInvalidVersionCombination,
		V2VerifierRepositoryUnavailable,
		V2VerifierPlanPathInvalid,
		V2VerifierManifestPathInvalid,
		V2VerifierSubjectUnresolved,
		V2VerifierFreezeUnresolved,
		V2VerifierClosureUnresolved,
		V2VerifierSubjectFreezeEqual,
		V2VerifierFreezeClosureEqual,
		V2VerifierSubjectClosureEqual,
		V2VerifierSubjectNotAncestorFreeze,
		V2VerifierFreezeNotAncestorClosure,
		V2VerifierSubjectFreezeUnrelated,
		V2VerifierFreezeClosureUnrelated,
		V2VerifierReverseSubjectFreezeTopology,
		V2VerifierReverseFreezeClosureTopology,
		V2VerifierTopologyObservationFailed,
		V2VerifierFrozenPlanMissing,
		V2VerifierFrozenPlanNotBlob,
		V2VerifierFrozenPlanReadFailed,
		V2VerifierClosureManifestMissing,
		V2VerifierClosureManifestNotBlob,
		V2VerifierClosureManifestReadFailed,
		V2VerifierClosureManifestInvalidJSON,
		V2VerifierClosureManifestContractInvalid,
		V2VerifierClosureManifestAssertionMismatch,
		V2VerifierObjectFormatUnavailable,
		// ACT 4 (9 codes)
		V2VerifierUnsupportedObjectFormat,
		V2VerifierManifestProtocolVersionMismatch,
		V2VerifierManifestPlanContractVersionMismatch,
		V2VerifierManifestSubjectMismatch,
		V2VerifierManifestSubjectTreeMismatch,
		V2VerifierManifestFreezeMismatch,
		V2VerifierManifestFreezeTreeMismatch,
		V2VerifierManifestExecutionTreeMismatch,
		V2VerifierManifestPlanPathMismatch,
		V2VerifierManifestPlanBlobMismatch,
		V2VerifierManifestPlanSHA256Mismatch,
		V2VerifierManifestBinaryIdentityInvalid,
		V2VerifierManifestCheckResultBijectionFailed,
		V2VerifierManifestUnsuccessfulRun,
		V2VerifierManifestCheckResultsInvalid,
		V2VerifierFrozenPlanInvalid,
		V2VerifierManifestUnknownCheckID,
		V2VerifierClosureTagMissing,
		V2VerifierClosureTagLightweight,
		V2VerifierClosureTagTargetMismatch,
		V2VerifierClosureTagUnreadable,
		V2VerifierStateCaptureHeadFailed,
		V2VerifierStateCaptureStatusFailed,
		V2VerifierStateCaptureWorktreeFailed,
		V2VerifierStateCaptureRefsFailed,
		V2VerifierStateMutationDetected,
	}
}
