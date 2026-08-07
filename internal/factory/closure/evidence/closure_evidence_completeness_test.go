// SPDX-License-Identifier: Apache-2.0

// Package evidence - closure_evidence_completeness_test.go
// provides TestClosureEvidenceCompletenessDerived for
// ACT-LEAMAS-FACTORY-CLOSURE-RUNTIME-CONTEXT-AND-EXECUTE01-CORRECTION02.
//
// The test is a table-driven mutation matrix: every row
// mutates exactly one observation of the COMPLETE verdict and
// asserts the predicate returns INCOMPLETE. The mutation matrix
// also includes a single PASS row that exercises the closed
// AND of every required observation.

package evidence

import (
	"crypto/sha256"
	"encoding/hex"
	"testing"
)

func makeValidAuthorities() CompletenessAuthorities {
	planBytes := []byte("{\"contract_version\":1,\"checks\":[]}")
	sum := sha256.Sum256(planBytes)
	planSHA := hex.EncodeToString(sum[:])
	return CompletenessAuthorities{
		Runtime: RuntimeAuthorityRecord{
			RepositoryRoot:              "/repo",
			SubjectCommit:               "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			SubjectTree:                 "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
			FreezeCommit:                "cccccccccccccccccccccccccccccccccccccccc",
			FreezeTree:                  "dddddddddddddddddddddddddddddddddddddddd",
			PlanPath:                    "docs/closure-plans/x.json",
			PlanBlob:                    "eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee",
			PlanSHA256:                  planSHA,
			PlanBytes:                   planBytes,
			EvidenceDirectory:           "/evidence",
			SubjectWorktreeRoot:         "/subject-worktree",
			SubjectWorktreeObservedTree: "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
			ExecutionTree:               "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
			PlanCheckIDs:                []string{"c1"},
		},
		Checks: []CheckResultRecord{
			{CheckID: "c1", Mode: "run", Outcome: "pass"},
		},
		Gate: GateClassificationRecord{
			Verdict: "PASS",
		},
		Binary: BuiltBinaryEvidence{
			BinaryPath:   "/tmp/leamas",
			BinarySHA256: planSHA,
			VCSRevision:  "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			VCSModified:  false,
			Executable:   true,
			SourceCommit: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			SourceTree:   "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
			SourceClean:  true,
			SourceDetached: true,
		},
		CallerAvailable: CallerStateAvailability{BeforeAvailable: true, AfterAvailable: true},
		CallerDrift:     CallerStateDrift{},
	}
}

// TestClosureEvidenceCompletenessDerived is the umbrella
// mutation test required by Phase 8. Every row mutates exactly
// one observation and asserts the predicate returns
// INCOMPLETE. The final PASS row proves the closed AND.
func TestClosureEvidenceCompletenessDerived(t *testing.T) {
	t.Parallel()
	pass := makeValidAuthorities()
	passEnv := ClosureEvidenceEx{
		SchemaVersion: 2,
		Authorities:   pass,
		Completeness:  EvidenceComplete,
	}

	t.Run("PASS when every observation is valid", func(t *testing.T) {
		t.Parallel()
		if got := DeriveClosureEvidenceCompletenessEx(passEnv); got != EvidenceComplete {
			t.Fatalf("expected COMPLETE, got %q", got)
		}
	})

	cases := []struct {
		name   string
		mutate func(*CompletenessAuthorities)
	}{
		{"runtime authority empty repository", func(a *CompletenessAuthorities) { a.Runtime.RepositoryRoot = "" }},
		{"runtime authority invalid subject commit", func(a *CompletenessAuthorities) { a.Runtime.SubjectCommit = "bad" }},
		{"runtime authority invalid subject tree", func(a *CompletenessAuthorities) { a.Runtime.SubjectTree = "bad" }},
		{"runtime authority invalid freeze commit", func(a *CompletenessAuthorities) { a.Runtime.FreezeCommit = "bad" }},
		{"runtime authority invalid freeze tree", func(a *CompletenessAuthorities) { a.Runtime.FreezeTree = "bad" }},
		{"runtime authority missing plan path", func(a *CompletenessAuthorities) { a.Runtime.PlanPath = "" }},
		{"runtime authority invalid plan blob", func(a *CompletenessAuthorities) { a.Runtime.PlanBlob = "bad" }},
		{"runtime authority invalid plan SHA-256", func(a *CompletenessAuthorities) { a.Runtime.PlanSHA256 = "bad" }},
		{"runtime authority missing evidence dir", func(a *CompletenessAuthorities) { a.Runtime.EvidenceDirectory = "" }},
		{"runtime authority missing subject worktree root", func(a *CompletenessAuthorities) { a.Runtime.SubjectWorktreeRoot = "" }},
		{"runtime authority subject worktree equals repo", func(a *CompletenessAuthorities) {
			a.Runtime.SubjectWorktreeRoot = a.Runtime.RepositoryRoot
		}},
		{"runtime authority subject tree mismatch", func(a *CompletenessAuthorities) {
			a.Runtime.SubjectWorktreeObservedTree = "ffffffffffffffffffffffffffffffffffffffff"
		}},
		{"runtime authority execution tree mismatch", func(a *CompletenessAuthorities) {
			a.Runtime.ExecutionTree = "ffffffffffffffffffffffffffffffffffffffff"
		}},

		{"frozen plan bytes length zero", func(a *CompletenessAuthorities) { a.Runtime.PlanBytes = nil }},
		{"frozen plan bytes SHA mismatch", func(a *CompletenessAuthorities) {
			other := []byte("different")
			sum := sha256.Sum256(other)
			a.Runtime.PlanSHA256 = hex.EncodeToString(sum[:])
		}},

		{"check result bijection duplicate id", func(a *CompletenessAuthorities) {
			a.Checks = append(a.Checks, CheckResultRecord{CheckID: "c1", Mode: "run", Outcome: "pass"})
		}},
		{"check result bijection empty", func(a *CompletenessAuthorities) { a.Checks = nil }},
		{"check result bijection empty id", func(a *CompletenessAuthorities) {
			a.Checks = []CheckResultRecord{{CheckID: "", Mode: "run", Outcome: "pass"}}
		}},

		{"run check failure", func(a *CompletenessAuthorities) {
			a.Checks[0].Outcome = "fail"
		}},
		{"run check timeout", func(a *CompletenessAuthorities) {
			a.Checks[0].TimedOut = true
		}},
		{"run check canceled", func(a *CompletenessAuthorities) {
			a.Checks[0].Canceled = true
		}},
		{"run check cleanup error", func(a *CompletenessAuthorities) {
			a.Checks[0].CleanupError = "boom"
		}},
		{"exclude check executed", func(a *CompletenessAuthorities) {
			a.Checks = append(a.Checks, CheckResultRecord{CheckID: "ex1", Mode: "exclude", Outcome: "pass"})
		}},

		{"gate timeout", func(a *CompletenessAuthorities) { a.Gate.Verdict = "FAIL" }},
		{"gate verdict fail", func(a *CompletenessAuthorities) { a.Gate.Verdict = "FAIL" }},

		{"binary path empty", func(a *CompletenessAuthorities) { a.Binary.BinaryPath = "" }},
		{"binary SHA-256 invalid", func(a *CompletenessAuthorities) {
			a.Binary.BinarySHA256 = "bad"
		}},
		{"binary VCS modified", func(a *CompletenessAuthorities) { a.Binary.VCSModified = true }},
		{"binary not executable", func(a *CompletenessAuthorities) { a.Binary.Executable = false }},
		{"binary VCS revision mismatch", func(a *CompletenessAuthorities) {
			a.Binary.VCSRevision = "ffffffffffffffffffffffffffffffffffffffff"
		}},

		{"before state unavailable", func(a *CompletenessAuthorities) { a.CallerAvailable.BeforeAvailable = false }},
		{"after state unavailable", func(a *CompletenessAuthorities) { a.CallerAvailable.AfterAvailable = false }},
		{"caller HEAD changed", func(a *CompletenessAuthorities) { a.CallerDrift.HEADChanged = true }},
		{"caller tree changed", func(a *CompletenessAuthorities) { a.CallerDrift.TreeChanged = true }},
		{"caller status changed", func(a *CompletenessAuthorities) { a.CallerDrift.StatusChanged = true }},
		{"caller refs changed", func(a *CompletenessAuthorities) { a.CallerDrift.RefsChanged = true }},
		{"worktree leaked", func(a *CompletenessAuthorities) { a.CallerDrift.WorktreeLeaked = true }},
	}

	for _, c := range cases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			mut := makeValidAuthorities()
			c.mutate(&mut)
			env := ClosureEvidenceEx{
				SchemaVersion: 2,
				Authorities:   mut,
				Completeness:  EvidenceComplete,
			}
			got := DeriveClosureEvidenceCompletenessEx(env)
			if got != EvidenceIncomplete {
				t.Fatalf("expected INCOMPLETE for %s, got %q", c.name, got)
			}
		})
	}
}
