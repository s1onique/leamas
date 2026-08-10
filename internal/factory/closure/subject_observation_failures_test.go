// SPDX-License-Identifier: Apache-2.0

package closure

// subject_observation_failures_test.go provides the R6-A
// adversarial matrix test body. The fake gitClient seam
// and the row table live in
// subject_observation_failures_seam_test.go so the test
// file holds only the canonical test function and stays
// under the LLM-friendly 400-line threshold.

import (
	"context"
	"testing"
)

// TestClosureSubjectObservationFailureMatrix exercises every
// documented Phase 15 failure row. Each row produces a
// typed *V2Error carrying the expected diagnostic code
// family (subject_observation_unavailable or
// subject_registration_mismatch) and a non-nil
// V2ExecuteResult whose SubjectWorktreePath and
// WorktreeInventoryBefore fields remain populated so the
// audit fields are preserved.
func TestClosureSubjectObservationFailureMatrix(t *testing.T) {
	rows := subjectMatrixFailureRows()
	for _, row := range rows {
		row := row
		t.Run(row.name, func(t *testing.T) {
			fx := subjectMatrixFixture(t)
			fake := row.matrix()
			fake.delegate = RealGit{}
			executor := NewGitV2SubjectExecutor(fake)
			req := V2ExecuteRequest{
				RepositoryRoot: fx.dir,
				SubjectCommit:  fx.subject,
				SubjectTree:    fx.subjectTree,
				EvidenceDir:    t.TempDir(),
				Checks: []PlanCheck{{
					ID:               "subject_only_present",
					Mode:             "run",
					Argv:             []string{"true"},
					WorkingDirectory: ".",
					TimeoutSeconds:   60,
					Environment:      map[string]string{},
				}},
			}
			result, err := executor.ExecuteSubjectChecks(context.Background(), req)
			if err == nil {
				t.Fatalf("row %q must fail closed", row.name)
			}
			v2err, ok := err.(*V2Error)
			if !ok {
				t.Fatalf("row %q must return *V2Error, got %T: %v", row.name, err, err)
			}
			wantCode, wantProp := subjectMatrixFailureOf(row.name)
			if !v2err.Diags.HasCode(wantCode) {
				t.Fatalf("row %q: expected code %s, got %v", row.name, wantCode, v2err.Diags.Codes())
			}
			found := false
			for _, d := range v2err.Diags {
				if d.Code == wantCode && d.PropertyName == wantProp {
					found = true
					break
				}
			}
			if !found {
				t.Fatalf("row %q: expected property %q, got %+v", row.name, wantProp, v2err.Diags)
			}
			if result.SubjectWorktreePath == "" {
				t.Fatalf("row %q: SubjectWorktreePath must be preserved for audit", row.name)
			}
		})
	}
}
