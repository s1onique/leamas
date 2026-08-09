// SPDX-License-Identifier: Apache-2.0

// Package evidence - closure_evidence_completeness_test.go
// provides TestClosureEvidenceCompletenessCanonical for
// ACT-LEAMAS-FACTORY-CLOSURE-RUNTIME-CONTEXT-AND-EXECUTE01-
// CORRECTION02-B2.
//
// The test is the single mutation matrix for the canonical
// completeness predicate. It builds exactly one valid candidate,
// asserts DeriveClosureEvidenceCompleteness returns EvidenceComplete,
// then mutates every predicate independently and asserts the
// predicate returns EvidenceIncomplete. Mutations are never
// combined.
//
// The test also asserts the matrix row count matches the
// declared predicate count so adding a predicate without
// adding a mutation row fails the test.
package evidence

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"testing"
)

// validCandidate returns a fully valid ClosureEvidence candidate.
// Every field is populated so DeriveClosureEvidenceCompleteness
// returns EvidenceComplete.
func validCandidate() ClosureEvidence {
	planBytes := []byte("{\"contract_version\":1,\"checks\":[{\"id\":\"c1\",\"mode\":\"run\"}]}")
	sum := sha256.Sum256(planBytes)
	planSHA := hex.EncodeToString(sum[:])
	subjectCommit := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	subjectTree := "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	freezeCommit := "cccccccccccccccccccccccccccccccccccccccc"
	freezeTree := "dddddddddddddddddddddddddddddddddddddddd"
	executionTree := subjectTree
	return ClosureEvidence{
		SchemaVersion: ClosureEvidenceSchemaVersion,
		Protocol:      ClosureProtocolVersion,
		Runtime: RuntimeAuthority{
			RepositoryRoot:       "/repo",
			FreezeCommit:         freezeCommit,
			FreezeTree:           freezeTree,
			SubjectCommit:        subjectCommit,
			SubjectTree:          subjectTree,
			ExecutionTree:        executionTree,
			PlanPath:             "docs/closure-plans/x.json",
			PlanBlob:             "eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee",
			PlanSHA256:           planSHA,
			PlanBytes:            planBytes,
			FAncestorOfSVerified: true,
		},
		Plan: PlanAuthority{
			ExpectedChecks: []PlanCheckSpec{
				{ID: "c1", Mode: "run"},
			},
		},
		Results: []CheckResult{
			{
				CheckID:  "c1",
				Mode:     "run",
				Outcome:  "pass",
				ExitCode: 0,
			},
		},
		Gate: GateAuthority{
			ObservedStatus:       "OK",
			Classification:       "PASS",
			InvocationCount:      1,
			RepositoryRoot:       "/repo",
			SubjectRoot:          subjectTree,
			SubjectExecutionRoot: subjectTree,
		},
		Binary: BinaryAuthority{
			BinaryPath:                "/tmp/leamas",
			BinarySHA256:              planSHA,
			BinaryCommit:              subjectCommit,
			BinaryModified:            false,
			SourceCommit:              subjectCommit,
			SourceTree:                subjectTree,
			SourceClean:               true,
			SourceDetached:            true,
			OutputOutsideAllWorktrees: true,
			Executable:                true,
		},
		CallerBefore: CallerStateSnapshot{
			Available:             true,
			Head:                  subjectCommit,
			Tree:                  subjectTree,
			StatusHash:            "status-hash",
			RefsHash:              "refs-hash",
			WorktreeInventoryHash: "wt-hash",
		},
		CallerAfter: CallerStateSnapshot{
			Available:             true,
			Head:                  subjectCommit,
			Tree:                  subjectTree,
			StatusHash:            "status-hash",
			RefsHash:              "refs-hash",
			WorktreeInventoryHash: "wt-hash",
		},
		Cleanup: CleanupAuthority{},
	}
}

// TestClosureEvidenceCompletenessCanonical is the umbrella
// mutation matrix required by Phase 3. The test proves:
//   - exactly one valid candidate yields EvidenceComplete
//   - every predicate mutation yields EvidenceIncomplete
//   - the mutation matrix covers every predicate in completenessPredicates
func TestClosureEvidenceCompletenessCanonical(t *testing.T) {
	t.Parallel()

	pass := validCandidate()
	t.Run("PASS when every observation is valid", func(t *testing.T) {
		t.Parallel()
		if got := DeriveClosureEvidenceCompleteness(pass); got != EvidenceComplete {
			t.Fatalf("expected COMPLETE, got %q", got)
		}
	})

	// Mutation matrix: every entry mutates exactly one predicate
	// and asserts the result is INCOMPLETE. Mutations are never
	// combined.
	type mutation struct {
		name   string
		mutate func(*ClosureEvidence)
	}
	mutations := []mutation{
		// runtime predicates (1..7)
		{"runtime_identities_structurally_valid: empty repo root", func(c *ClosureEvidence) { c.Runtime.RepositoryRoot = "" }},
		{"runtime_identities_structurally_valid: invalid freeze OID", func(c *ClosureEvidence) { c.Runtime.FreezeCommit = "bad" }},
		{"runtime_identities_structurally_valid: invalid freeze tree", func(c *ClosureEvidence) { c.Runtime.FreezeTree = "bad" }},
		{"runtime_identities_structurally_valid: invalid subject OID", func(c *ClosureEvidence) { c.Runtime.SubjectCommit = "bad" }},
		{"runtime_identities_structurally_valid: invalid subject tree", func(c *ClosureEvidence) { c.Runtime.SubjectTree = "bad" }},
		{"runtime_identities_structurally_valid: invalid execution tree", func(c *ClosureEvidence) { c.Runtime.ExecutionTree = "bad" }},
		{"runtime_identities_structurally_valid: empty plan path", func(c *ClosureEvidence) { c.Runtime.PlanPath = "" }},
		{"runtime_identities_structurally_valid: invalid plan blob", func(c *ClosureEvidence) { c.Runtime.PlanBlob = "bad" }},
		{"runtime_identities_structurally_valid: invalid plan SHA", func(c *ClosureEvidence) { c.Runtime.PlanSHA256 = "bad" }},
		{"runtime_freeze_different_from_subject: F==S", func(c *ClosureEvidence) {
			c.Runtime.FreezeCommit = c.Runtime.SubjectCommit
		}},
		{"runtime_f_ancestor_of_s_verified: false", func(c *ClosureEvidence) {
			c.Runtime.FAncestorOfSVerified = false
		}},
		{"runtime_execution_tree_equals_subject: mismatch", func(c *ClosureEvidence) {
			c.Runtime.ExecutionTree = "ffffffffffffffffffffffffffffffffffffffff"
		}},
		{"runtime_plan_blob_valid: bad blob", func(c *ClosureEvidence) { c.Runtime.PlanBlob = "bad" }},
		{"runtime_plan_sha256_valid: bad SHA", func(c *ClosureEvidence) { c.Runtime.PlanSHA256 = "bad" }},
		{"runtime_plan_bytes_parse_successfully: nil bytes", func(c *ClosureEvidence) { c.Runtime.PlanBytes = nil }},
		{"runtime_plan_bytes_parse_successfully: SHA mismatch", func(c *ClosureEvidence) {
			other := []byte("different")
			sum := sha256.Sum256(other)
			c.Runtime.PlanSHA256 = hex.EncodeToString(sum[:])
		}},
		{"runtime_plan_bytes_parse_successfully: arbitrary bytes matching SHA", func(c *ClosureEvidence) {
			// Garbage bytes whose SHA-256 matches the
			// recorded PlanSHA256. The previous B2
			// predicate accepted this; the B2-R1
			// production decoder rejects it.
			other := []byte("not a plan contract\n")
			sum := sha256.Sum256(other)
			c.Runtime.PlanBytes = other
			c.Runtime.PlanSHA256 = hex.EncodeToString(sum[:])
		}},
		{"runtime_expected_checks_derived_from_plan_bytes: drop the check", func(c *ClosureEvidence) {
			// Replace the candidate's expected checks with
			// an empty list. The predicate that binds
			// ExpectedChecks to the production-decoded
			// Plan must reject the candidate.
			c.Plan.ExpectedChecks = c.Plan.ExpectedChecks[:0]
		}},
		{"runtime_expected_checks_derived_from_plan_bytes: substitute plan", func(c *ClosureEvidence) {
			// Replace the bytes with a different but
			// still-parseable plan whose declared check
			// differs from the candidate's expected list.
			other := []byte(`{"contract_version":1,"checks":[{"id":"ghost","mode":"run"}]}`)
			sum := sha256.Sum256(other)
			c.Runtime.PlanBytes = other
			c.Runtime.PlanSHA256 = hex.EncodeToString(sum[:])
		}},

		// plan/result predicates (8..12)
		{"plan_result_cardinality_equal: missing expected checks", func(c *ClosureEvidence) { c.Plan.ExpectedChecks = nil }},
		{"plan_result_cardinality_equal: extra result", func(c *ClosureEvidence) {
			c.Results = append(c.Results, CheckResult{CheckID: "c2", Mode: "run", Outcome: "pass"})
		}},
		{"plan_result_ids_bijective: duplicate expected", func(c *ClosureEvidence) {
			c.Plan.ExpectedChecks = append(c.Plan.ExpectedChecks, PlanCheckSpec{ID: "c1", Mode: "run"})
		}},
		{"plan_result_ids_bijective: duplicate result", func(c *ClosureEvidence) {
			c.Results = append(c.Results, CheckResult{CheckID: "c1", Mode: "run", Outcome: "pass"})
		}},
		{"plan_result_ids_bijective: unknown result ID", func(c *ClosureEvidence) {
			c.Results[0].CheckID = "ghost"
		}},
		{"plan_result_order_matches_plan: order mismatch", func(c *ClosureEvidence) {
			c.Plan.ExpectedChecks = []PlanCheckSpec{
				{ID: "c2", Mode: "run"},
				{ID: "c1", Mode: "run"},
			}
		}},
		{"plan_result_mode_matches_plan: mode mismatch", func(c *ClosureEvidence) {
			c.Results[0].Mode = "exclude"
		}},
		{"plan_no_unknown_check_mode: unknown result mode", func(c *ClosureEvidence) {
			c.Results[0].Mode = "unknown"
		}},
		{"plan_no_unknown_check_mode: unknown plan mode", func(c *ClosureEvidence) {
			c.Plan.ExpectedChecks[0].Mode = "unknown"
		}},
		{"plan_result_ids_bijective: empty results", func(c *ClosureEvidence) { c.Results = nil }},

		// results predicates (13..19)
		{"results_every_run_check_successful: outcome fail", func(c *ClosureEvidence) { c.Results[0].Outcome = "fail" }},
		{"results_every_run_check_successful: run timeout", func(c *ClosureEvidence) { c.Results[0].TimedOut = true }},
		{"results_every_run_check_successful: run canceled", func(c *ClosureEvidence) { c.Results[0].Canceled = true }},
		{"results_every_run_check_successful: run cleanup error", func(c *ClosureEvidence) { c.Results[0].CleanupError = "boom" }},
		{"results_every_exclude_check_excluded: missing exclude", func(c *ClosureEvidence) {
			c.Plan.ExpectedChecks = []PlanCheckSpec{
				{ID: "c1", Mode: "run"},
				{ID: "ex1", Mode: "exclude"},
			}
		}},
		{"results_every_exclude_check_excluded: excluded reported successful", func(c *ClosureEvidence) {
			c.Plan.ExpectedChecks = []PlanCheckSpec{{ID: "ex1", Mode: "exclude"}}
			c.Results = []CheckResult{{CheckID: "ex1", Mode: "exclude", Outcome: "pass", ExitCode: 0}}
		}},
		{"results_every_exclude_check_excluded: run reported excluded", func(c *ClosureEvidence) {
			c.Results[0].Outcome = "excluded"
		}},
		{"results_every_exclude_check_excluded: exit nonzero", func(c *ClosureEvidence) {
			c.Plan.ExpectedChecks = []PlanCheckSpec{{ID: "ex1", Mode: "exclude"}}
			c.Results = []CheckResult{{CheckID: "ex1", Mode: "exclude", Outcome: "excluded", ExitCode: 1}}
		}},
		{"results_no_timeout: timed out", func(c *ClosureEvidence) { c.Results[0].TimedOut = true }},
		{"results_no_cancellation: canceled", func(c *ClosureEvidence) { c.Results[0].Canceled = true }},
		{"results_no_stdout_truncation: stdout truncated", func(c *ClosureEvidence) { c.Results[0].StdoutTruncated = true }},
		{"results_no_stderr_truncation: stderr truncated", func(c *ClosureEvidence) { c.Results[0].StderrTruncated = true }},
		{"results_no_execution_cleanup_error: cleanup error", func(c *ClosureEvidence) {
			c.Cleanup.SubjectCleanupError = "boom"
		}},

		// gate predicates (20..25)
		{"gate_classification_equals_pass: FAIL", func(c *ClosureEvidence) { c.Gate.Classification = "FAIL" }},
		{"gate_classification_equals_pass: UNAVAILABLE", func(c *ClosureEvidence) { c.Gate.Classification = "UNAVAILABLE" }},
		{"gate_invocation_count_equals_one: count 0", func(c *ClosureEvidence) { c.Gate.InvocationCount = 0 }},
		{"gate_invocation_count_equals_one: count 2", func(c *ClosureEvidence) { c.Gate.InvocationCount = 2 }},
		{"gate_subject_root_equals_s_exec_root: empty subject root", func(c *ClosureEvidence) { c.Gate.SubjectRoot = "" }},
		{"gate_subject_root_equals_s_exec_root: mismatched root", func(c *ClosureEvidence) {
			// SubjectRoot must equal the runtime
			// SubjectTree. The previous B2
			// implementation only checked non-empty.
			c.Gate.SubjectRoot = "ffffffffffffffffffffffffffffffffffffffff"
		}},
		{"gate_subject_execution_root_matches_tree: empty", func(c *ClosureEvidence) { c.Gate.SubjectExecutionRoot = "" }},
		{"gate_subject_execution_root_matches_tree: mismatch", func(c *ClosureEvidence) {
			c.Gate.SubjectExecutionRoot = "ffffffffffffffffffffffffffffffffffffffff"
		}},
		{"gate_not_timed_out: timed out", func(c *ClosureEvidence) { c.Gate.TimedOut = true }},
		{"gate_no_output_truncation: stdout truncated", func(c *ClosureEvidence) { c.Gate.StdoutTruncated = true }},
		{"gate_no_output_truncation: stderr truncated", func(c *ClosureEvidence) { c.Gate.StderrTruncated = true }},
		{"gate_error_absent: error present", func(c *ClosureEvidence) { c.Gate.Error = "boom" }},

		// binary predicates (28..38)
		{"binary_path_non_empty: empty path", func(c *ClosureEvidence) { c.Binary.BinaryPath = "" }},
		{"binary_sha256_valid: invalid", func(c *ClosureEvidence) { c.Binary.BinarySHA256 = "bad" }},
		{"binary_commit_equals_subject_commit: mismatch", func(c *ClosureEvidence) {
			c.Binary.BinaryCommit = "ffffffffffffffffffffffffffffffffffffffff"
		}},
		{"binary_not_modified: modified=true", func(c *ClosureEvidence) { c.Binary.BinaryModified = true }},
		{"binary_source_commit_equals_subject_commit: mismatch", func(c *ClosureEvidence) {
			c.Binary.SourceCommit = "ffffffffffffffffffffffffffffffffffffffff"
		}},
		{"binary_source_tree_equals_subject_tree: mismatch", func(c *ClosureEvidence) {
			c.Binary.SourceTree = "ffffffffffffffffffffffffffffffffffffffff"
		}},
		{"binary_source_clean: false", func(c *ClosureEvidence) { c.Binary.SourceClean = false }},
		{"binary_source_detached: false", func(c *ClosureEvidence) { c.Binary.SourceDetached = false }},
		{"binary_output_outside_all_worktrees: false", func(c *ClosureEvidence) { c.Binary.OutputOutsideAllWorktrees = false }},
		{"binary_executable: false", func(c *ClosureEvidence) { c.Binary.Executable = false }},
		{"binary_cleanup_error_absent: error", func(c *ClosureEvidence) {
			c.Cleanup.BinaryCleanupError = "boom"
		}},

		// caller predicates (39..47)
		{"caller_before_available: unavailable", func(c *ClosureEvidence) { c.CallerBefore.Available = false }},
		{"caller_after_available: unavailable", func(c *ClosureEvidence) { c.CallerAfter.Available = false }},
		{"caller_before_snapshot_complete: empty head", func(c *ClosureEvidence) {
			// Available=true but the BEFORE observer
			// failed to record any of the required fields.
			// The previous B2 implementation accepted this.
			c.CallerBefore.Head = ""
		}},
		{"caller_before_snapshot_complete: empty tree", func(c *ClosureEvidence) {
			c.CallerBefore.Tree = ""
		}},
		{"caller_before_snapshot_complete: empty status", func(c *ClosureEvidence) {
			c.CallerBefore.StatusHash = ""
		}},
		{"caller_before_snapshot_complete: empty refs", func(c *ClosureEvidence) {
			c.CallerBefore.RefsHash = ""
		}},
		{"caller_before_snapshot_complete: empty inventory", func(c *ClosureEvidence) {
			c.CallerBefore.WorktreeInventoryHash = ""
		}},
		{"caller_after_snapshot_complete: empty head", func(c *ClosureEvidence) {
			c.CallerAfter.Head = ""
		}},
		{"caller_after_snapshot_complete: empty tree", func(c *ClosureEvidence) {
			c.CallerAfter.Tree = ""
		}},
		{"caller_after_snapshot_complete: empty status", func(c *ClosureEvidence) {
			c.CallerAfter.StatusHash = ""
		}},
		{"caller_after_snapshot_complete: empty refs", func(c *ClosureEvidence) {
			c.CallerAfter.RefsHash = ""
		}},
		{"caller_after_snapshot_complete: empty inventory", func(c *ClosureEvidence) {
			c.CallerAfter.WorktreeInventoryHash = ""
		}},
		{"caller_head_unchanged: changed", func(c *ClosureEvidence) {
			c.CallerAfter.Head = "1111111111111111111111111111111111111111"
		}},
		{"caller_tree_unchanged: changed", func(c *ClosureEvidence) {
			c.CallerAfter.Tree = "2222222222222222222222222222222222222222"
		}},
		{"caller_status_unchanged: changed", func(c *ClosureEvidence) { c.CallerAfter.StatusHash = "different" }},
		{"caller_refs_unchanged: changed", func(c *ClosureEvidence) { c.CallerAfter.RefsHash = "different" }},
		{"caller_worktree_inventory_unchanged: changed", func(c *ClosureEvidence) { c.CallerAfter.WorktreeInventoryHash = "different" }},
		{"caller_refs_unchanged: empty-but-available then changed", func(c *ClosureEvidence) {
			// Simulate the empty-but-available state: caller
			// recorded an empty refs hash but the runner saw
			// a different one. The drift is rejected.
			c.CallerBefore.RefsHash = ""
			c.CallerAfter.RefsHash = "different"
		}},
	}

	for _, m := range mutations {
		m := m
		t.Run(m.name, func(t *testing.T) {
			t.Parallel()
			mut := validCandidate()
			m.mutate(&mut)
			got := DeriveClosureEvidenceCompleteness(mut)
			if got != EvidenceIncomplete {
				t.Fatalf("expected INCOMPLETE for %s, got %q", m.name, got)
			}
		})
	}

	// Row-count guard: every entry in completenessPredicates MUST
	// have at least one mutation row above. The guard catches a
	// developer who adds a new predicate but forgets to add a
	// mutation row, which would otherwise leave the predicate
	// untested.
	t.Run("mutation matrix covers every predicate", func(t *testing.T) {
		t.Parallel()
		covered := make(map[string]bool)
		for _, m := range mutations {
			// Trim trailing ":<description>" so the prefix matches
			// the predicate map key.
			name := m.name
			if idx := strings.Index(name, ":"); idx > 0 {
				name = name[:idx]
			}
			covered[name] = true
		}
		if len(covered) != len(completenessPredicates) {
			missing := []string{}
			extra := []string{}
			for k := range completenessPredicates {
				if !covered[k] {
					missing = append(missing, k)
				}
			}
			for k := range covered {
				if _, ok := completenessPredicates[k]; !ok {
					extra = append(extra, k)
				}
			}
			t.Fatalf("mutation matrix mismatch: covered=%d predicates=%d missing=%v extra=%v",
				len(covered), len(completenessPredicates), missing, extra)
		}
	})

	// Predicate count guard: the matrix must have the expected
	// number of predicates. The constant is the only allowed
	// source of truth.
	if got := len(completenessPredicates); got != completenessPredicateCount {
		t.Fatalf("predicate count drift: declared=%d actual=%d", completenessPredicateCount, got)
	}
}
