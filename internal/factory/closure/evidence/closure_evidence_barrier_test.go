// SPDX-License-Identifier: Apache-2.0

// Package evidence - closure_evidence_barrier_test.go provides
// the publication barrier matrix tests required by Phase 7 and
// the cannot-be-forced matrix required by Phase 4.
//
// The publication barrier (PrepareClosureEvidenceForPublication)
// is the only authority that may emit final evidence bytes.
// Every negative matrix row proves:
//   - the barrier returns a zero PublicationCandidate
//   - the barrier returns a typed error
//   - no marshaled JSON bytes leak through any return path
//
// The cannot-be-forced matrix proves that:
//
//	DECLARED_COMPLETE_IGNORED_AS_AUTHORITY=true
//	DERIVED_COMPLETE_ONLY=true
//
// even when a caller supplies a JSON document with
// `"completeness": "COMPLETE"`, the barrier re-derives and
// rejects if the candidate is incomplete.
package evidence

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"testing"
)

// TestClosureEvidencePublicationBarrier is the umbrella for
// the barrier negative matrix. Every row mutates a fully
// valid candidate into an invalid one and asserts that the
// barrier refuses to emit a PublicationCandidate.
func TestClosureEvidencePublicationBarrier(t *testing.T) {
	t.Parallel()

	// Helper: assert the barrier rejects the candidate.
	assertBarrierRejects := func(t *testing.T, mutate func(*ClosureEvidence), label string) {
		t.Helper()
		candidate := validCandidate()
		mutate(&candidate)
		got, err := PrepareClosureEvidenceForPublication(candidate)
		if err == nil {
			t.Fatalf("barrier must reject %s, got %+v", label, got)
		}
		if got.Bytes != nil {
			t.Fatalf("barrier must not return bytes for %s, got %d bytes", label, len(got.Bytes))
		}
		if got.SHA256 != "" {
			t.Fatalf("barrier must not return SHA256 for %s, got %q", label, got.SHA256)
		}
		if !errors.Is(err, ErrIncompleteEvidence) {
			// The barrier may also return a validation error
			// when the candidate is structurally invalid.
			// Both are acceptable; the contract is that no
			// PublicationCandidate is returned.
			if !strings.Contains(err.Error(), "incomplete") && !strings.Contains(err.Error(), "validation") {
				t.Fatalf("unexpected error for %s: %v", label, err)
			}
		}
	}

	// --- caller BEFORE unavailable ---
	t.Run("caller_before_unavailable", func(t *testing.T) {
		t.Parallel()
		assertBarrierRejects(t, func(c *ClosureEvidence) { c.CallerBefore.Available = false }, "caller_before unavailable")
	})

	// --- caller AFTER unavailable ---
	t.Run("caller_after_unavailable", func(t *testing.T) {
		t.Parallel()
		assertBarrierRejects(t, func(c *ClosureEvidence) { c.CallerAfter.Available = false }, "caller_after unavailable")
	})

	// --- drifts ---
	t.Run("head_drift", func(t *testing.T) {
		t.Parallel()
		assertBarrierRejects(t, func(c *ClosureEvidence) {
			c.CallerAfter.Head = "1111111111111111111111111111111111111111"
		}, "head drift")
	})
	t.Run("tree_drift", func(t *testing.T) {
		t.Parallel()
		assertBarrierRejects(t, func(c *ClosureEvidence) {
			c.CallerAfter.Tree = "2222222222222222222222222222222222222222"
		}, "tree drift")
	})
	t.Run("status_drift", func(t *testing.T) {
		t.Parallel()
		assertBarrierRejects(t, func(c *ClosureEvidence) {
			c.CallerAfter.StatusHash = "different"
		}, "status drift")
	})
	t.Run("refs_drift", func(t *testing.T) {
		t.Parallel()
		assertBarrierRejects(t, func(c *ClosureEvidence) {
			c.CallerAfter.RefsHash = "different"
		}, "refs drift")
	})
	t.Run("worktree_inventory_drift", func(t *testing.T) {
		t.Parallel()
		assertBarrierRejects(t, func(c *ClosureEvidence) {
			c.CallerAfter.WorktreeInventoryHash = "different"
		}, "worktree inventory drift")
	})

	// --- plan/result mismatch ---
	t.Run("plan_result_mismatch", func(t *testing.T) {
		t.Parallel()
		assertBarrierRejects(t, func(c *ClosureEvidence) {
			c.Results = append(c.Results, CheckResult{CheckID: "c2", Mode: "run", Outcome: "pass"})
		}, "plan/result cardinality mismatch")
	})

	// --- run check failure ---
	t.Run("run_check_failure", func(t *testing.T) {
		t.Parallel()
		assertBarrierRejects(t, func(c *ClosureEvidence) {
			c.Results[0].Outcome = "fail"
		}, "run check failure")
	})

	// --- exclude violation ---
	t.Run("exclude_violation", func(t *testing.T) {
		t.Parallel()
		assertBarrierRejects(t, func(c *ClosureEvidence) {
			c.Plan.ExpectedChecks = []PlanCheckSpec{{ID: "ex1", Mode: "exclude"}}
			c.Results = []CheckResult{{CheckID: "ex1", Mode: "exclude", Outcome: "pass", ExitCode: 0}}
		}, "exclude reported successful")
	})

	// --- execution timeout / cancellation / truncation ---
	t.Run("execution_timeout", func(t *testing.T) {
		t.Parallel()
		assertBarrierRejects(t, func(c *ClosureEvidence) { c.Results[0].TimedOut = true }, "execution timeout")
	})
	t.Run("execution_cancellation", func(t *testing.T) {
		t.Parallel()
		assertBarrierRejects(t, func(c *ClosureEvidence) { c.Results[0].Canceled = true }, "execution cancellation")
	})
	t.Run("execution_stdout_truncation", func(t *testing.T) {
		t.Parallel()
		assertBarrierRejects(t, func(c *ClosureEvidence) { c.Results[0].StdoutTruncated = true }, "execution stdout truncation")
	})
	t.Run("execution_stderr_truncation", func(t *testing.T) {
		t.Parallel()
		assertBarrierRejects(t, func(c *ClosureEvidence) { c.Results[0].StderrTruncated = true }, "execution stderr truncation")
	})
	t.Run("execution_cleanup_failure", func(t *testing.T) {
		t.Parallel()
		assertBarrierRejects(t, func(c *ClosureEvidence) {
			c.Cleanup.SubjectCleanupError = "boom"
		}, "execution cleanup failure")
	})

	// --- gate failure modes ---
	t.Run("gate_FAIL", func(t *testing.T) {
		t.Parallel()
		assertBarrierRejects(t, func(c *ClosureEvidence) { c.Gate.Classification = "FAIL" }, "gate FAIL")
	})
	t.Run("gate_UNAVAILABLE", func(t *testing.T) {
		t.Parallel()
		assertBarrierRejects(t, func(c *ClosureEvidence) { c.Gate.Classification = "UNAVAILABLE" }, "gate UNAVAILABLE")
	})
	t.Run("gate_calls_0", func(t *testing.T) {
		t.Parallel()
		assertBarrierRejects(t, func(c *ClosureEvidence) { c.Gate.InvocationCount = 0 }, "gate calls 0")
	})
	t.Run("gate_calls_2", func(t *testing.T) {
		t.Parallel()
		assertBarrierRejects(t, func(c *ClosureEvidence) { c.Gate.InvocationCount = 2 }, "gate calls 2")
	})
	t.Run("gate_timeout", func(t *testing.T) {
		t.Parallel()
		assertBarrierRejects(t, func(c *ClosureEvidence) { c.Gate.TimedOut = true }, "gate timeout")
	})
	t.Run("gate_output_truncation", func(t *testing.T) {
		t.Parallel()
		assertBarrierRejects(t, func(c *ClosureEvidence) { c.Gate.StdoutTruncated = true }, "gate output truncation")
	})

	// --- binary failure modes ---
	t.Run("binary_invalid", func(t *testing.T) {
		t.Parallel()
		assertBarrierRejects(t, func(c *ClosureEvidence) { c.Binary.BinaryPath = "" }, "binary invalid")
	})
	t.Run("binary_commit_mismatch", func(t *testing.T) {
		t.Parallel()
		assertBarrierRejects(t, func(c *ClosureEvidence) {
			c.Binary.BinaryCommit = "ffffffffffffffffffffffffffffffffffffffff"
		}, "binary commit mismatch")
	})
	t.Run("binary_modified", func(t *testing.T) {
		t.Parallel()
		assertBarrierRejects(t, func(c *ClosureEvidence) { c.Binary.BinaryModified = true }, "binary modified")
	})
	t.Run("binary_cleanup_failure", func(t *testing.T) {
		t.Parallel()
		assertBarrierRejects(t, func(c *ClosureEvidence) {
			c.Cleanup.BinaryCleanupError = "boom"
		}, "binary cleanup failure")
	})

	// --- happy path: COMPLETE candidate produces a PublicationCandidate ---
	t.Run("happy_path_produces_publication_candidate", func(t *testing.T) {
		t.Parallel()
		candidate := validCandidate()
		got, err := PrepareClosureEvidenceForPublication(candidate)
		if err != nil {
			t.Fatalf("barrier rejected valid candidate: %v", err)
		}
		if len(got.Bytes) == 0 {
			t.Fatalf("barrier must produce non-empty bytes")
		}
		if len(got.SHA256) != 64 {
			t.Fatalf("barrier SHA256 must be 64-char hex, got %q", got.SHA256)
		}
		if ComputeEvidenceSHA256(got.Bytes) != got.SHA256 {
			t.Fatalf("candidate SHA256 must match SHA256(bytes)")
		}
	})
}

// TestClosureEvidenceCompletenessCannotBeForced is the umbrella
// for the cannot-be-forced matrix. The test proves that no
// caller-controlled mutation can force the canonical predicate
// to return EvidenceComplete.
//
//	DECLARED_COMPLETE_IGNORED_AS_AUTHORITY=true
//	DERIVED_COMPLETE_ONLY=true
func TestClosureEvidenceCompletenessCannotBeForced(t *testing.T) {
	t.Parallel()

	t.Run("serialized_completeness_in_JSON_is_ignored", func(t *testing.T) {
		t.Parallel()
		// A caller writes a JSON document that claims
		// `completeness: "COMPLETE"`. The strict decoder
		// (UnmarshalClosureEvidence) MUST reject the
		// unknown field, and the barrier MUST re-derive
		// and reject the incomplete candidate.
		candidate := validCandidate()
		// Mutate the candidate so it is NOT complete.
		candidate.Results[0].Outcome = "fail"
		bytes, err := json.Marshal(candidate)
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
		// Inject a parallel completeness field. The
		// canonical struct has no Completeness field and
		// the strict decoder rejects unknown keys.
		injected := []byte(`{"completeness":"COMPLETE",`)
		injected = append(injected, bytes[1:]...) // strip the leading '{'
		if _, err := UnmarshalClosureEvidence(injected); err == nil {
			t.Fatalf("strict decoder must reject injected completeness field")
		}
		// The strict decoder must also reject unknown
		// authority-looking fields such as
		// "classification" at the document root.
		for _, field := range []string{"classification", "verdict", "pass"} {
			injected := []byte(`{"` + field + `":"COMPLETE",`)
			injected = append(injected, bytes[1:]...)
			if _, err := UnmarshalClosureEvidence(injected); err == nil {
				t.Fatalf("strict decoder must reject unknown field %q", field)
			}
		}
		// For comparison: the canonical struct decoded
		// with the strict decoder from the unmodified
		// marshaled bytes must succeed and the resulting
		// candidate must be derivable (still incomplete).
		decoded, err := UnmarshalClosureEvidence(bytes)
		if err != nil {
			t.Fatalf("strict decoder must accept unmodified bytes: %v", err)
		}
		if got := DeriveClosureEvidenceCompleteness(decoded); got == EvidenceComplete {
			t.Fatalf("incomplete candidate must NOT pass derived, got %q", got)
		}
		// The barrier must reject the decoded candidate.
		got, err := PrepareClosureEvidenceForPublication(decoded)
		if err == nil {
			t.Fatalf("barrier must reject decoded incomplete candidate, got %+v", got)
		}
	})

	t.Run("manual_struct_field_mutation_cannot_force_complete", func(t *testing.T) {
		t.Parallel()
		// The canonical struct has no Completeness field;
		// there is no field to mutate. The test confirms
		// an incomplete candidate cannot be promoted via
		// the candidate -> barrier path.
		candidate := validCandidate()
		candidate.Results[0].Outcome = "fail"
		if got := DeriveClosureEvidenceCompleteness(candidate); got == EvidenceComplete {
			t.Fatalf("incomplete candidate must NOT pass derived, got %q", got)
		}
		_, err := PrepareClosureEvidenceForPublication(candidate)
		if err == nil {
			t.Fatalf("incomplete candidate must be rejected by the barrier")
		}
	})

	t.Run("candidate_copied_after_derivation_still_validated", func(t *testing.T) {
		t.Parallel()
		// Derive the predicate once on a fresh candidate, then
		// build a fresh candidate for the mutated copy. The
		// barrier must re-derive and reject the mutated copy
		// but accept the original. Using a fresh copy is
		// necessary because Go slice copies share the backing
		// array.
		original := validCandidate()
		_ = DeriveClosureEvidenceCompleteness(original)
		copy := validCandidate()
		copy.Results[0].Outcome = "fail"
		got, err := PrepareClosureEvidenceForPublication(copy)
		if err == nil {
			t.Fatalf("barrier must reject mutated copy, got %+v", got)
		}
		_, err = PrepareClosureEvidenceForPublication(original)
		if err != nil {
			t.Fatalf("barrier must accept original valid candidate, got %v", err)
		}
	})

	t.Run("invalid_candidate_passed_directly_to_validation", func(t *testing.T) {
		t.Parallel()
		// An invalid candidate passed directly to the barrier
		// must be rejected on the validation AND derivation path.
		candidate := BuildEmptyEvidence()
		// Mark nothing valid; barrier must reject.
		if err := ValidateClosureEvidence(candidate); err == nil {
			t.Fatalf("empty candidate must fail validation")
		}
		if got := DeriveClosureEvidenceCompleteness(candidate); got == EvidenceComplete {
			t.Fatalf("empty candidate must be INCOMPLETE, got %q", got)
		}
		_, err := PrepareClosureEvidenceForPublication(candidate)
		if err == nil {
			t.Fatalf("barrier must reject empty candidate")
		}
		if !errors.Is(err, ErrIncompleteEvidence) && !strings.Contains(err.Error(), "validation") {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("DECLARED_COMPLETE_IGNORED_AS_AUTHORITY", func(t *testing.T) {
		t.Parallel()
		// Document the invariant: the canonical struct has
		// no Completeness field. Any caller that supplies
		// one in the JSON is dropped on decode.
		candidate := validCandidate()
		bytes, err := json.Marshal(candidate)
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
		if bytesContains(bytes, []byte("completeness")) {
			t.Fatalf("canonical JSON must not contain a completeness field, got %s", bytes)
		}
	})

	t.Run("DERIVED_COMPLETE_ONLY", func(t *testing.T) {
		t.Parallel()
		// A valid candidate must derive as COMPLETE.
		candidate := validCandidate()
		if got := DeriveClosureEvidenceCompleteness(candidate); got != EvidenceComplete {
			t.Fatalf("valid candidate must derive COMPLETE, got %q", got)
		}
	})
}

// bytesContains is a tiny helper that returns true if haystack
// contains needle.
func bytesContains(haystack, needle []byte) bool {
	return bytes.Contains(haystack, needle)
}

// Ensure io is referenced for the strict-JSON rule test
// imported by closure_evidence_serialization_test.go.
var _ = io.EOF
