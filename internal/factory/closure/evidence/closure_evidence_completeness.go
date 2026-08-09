// SPDX-License-Identifier: Apache-2.0

// Package evidence - closure_evidence_completeness.go owns the
// single canonical predicate DeriveClosureEvidenceCompleteness
// and the shared hex helpers.
//
// COMPLETE requires a closed AND of every required observation.
// No predicate may be skipped because a string is empty:
// "empty" and "unavailable" are deliberately distinct states
// and the predicate rejects the empty presence of an authority
// field as INCOMPLETE.
//
// The 47 predicates are encoded in declaration order. Each
// predicate is a separate function so the mutation matrix in
// TestClosureEvidenceCompletenessCanonical can exercise them
// independently. The matrix MUST grow with every added
// predicate; the test asserts row count via the tracked
// predicate set in completenessPredicates.
//
// The per-predicate functions are split across multiple
// files to keep each file under the LLM-friendly 400-line
// threshold while preserving the single closure over the
// descriptor that ACT-LEAMAS-FACTORY-CLOSURE-PROTOCOL-V1-01
// requires.
package evidence

// DeriveClosureEvidenceCompleteness is the canonical predicate.
// It returns EvidenceComplete only when every required
// predicate is true, and EvidenceIncomplete otherwise. Callers
// cannot force COMPLETE: the verdict is always derived from
// the candidate's authorities.
func DeriveClosureEvidenceCompleteness(candidate ClosureEvidence) EvidenceCompleteness {
	for _, fn := range completenessPredicatesInOrder {
		if !fn(candidate) {
			return EvidenceIncomplete
		}
	}
	return EvidenceComplete
}

// completenessPredicatesInOrder is the authoritative ordered
// list of the 47 predicates. The slice form drives the
// canonical predicate; the map form (completenessPredicates)
// drives the mutation matrix test. Both MUST stay in sync.
//
// B2-R1 changes vs the prior 43-predicate matrix:
//   - added runtimeExpectedChecksDerivedFromPlanBytes (binds
//     Plan.ExpectedChecks to the production-decoded Plan
//     Contract)
//   - added gateSubjectExecutionRootMatchesTree (companion
//     gate binding to SubjectTree)
//   - added binaryPathNonEmpty (replaces the BinaryPath slot
//     of the removed binaryAuthorityValid composite)
//   - added callerBeforeSnapshotComplete and
//     callerAfterSnapshotComplete (Available implies all
//     observable fields are non-empty)
//   - removed binaryAuthorityValid (composite of atomic
//     predicates; one mutation could fail multiple predicates
//     at once)
var completenessPredicatesInOrder = []func(ClosureEvidence) bool{
	// runtime predicates (1..8)
	runtimeIdentitiesStructurallyValid,
	runtimeFreezeDifferentFromSubject,
	runtimeFAncestorOfSVerified,
	runtimeExecutionTreeEqualsSubjectTree,
	runtimePlanBlobValid,
	runtimePlanSHA256Valid,
	runtimePlanBytesParseSuccessfully,
	runtimeExpectedChecksDerivedFromPlanBytes,

	// plan/result predicates (9..13)
	planResultCardinalityEqual,
	planResultIDsBijective,
	planResultOrderMatchesPlan,
	planResultModeMatchesPlanMode,
	planNoUnknownCheckMode,

	// results predicates (14..20)
	resultsEveryRunCheckSuccessful,
	resultsEveryExcludeCheckExcluded,
	resultsNoTimeout,
	resultsNoCancellation,
	resultsNoStdoutTruncation,
	resultsNoStderrTruncation,
	resultsNoExecutionCleanupError,

	// gate predicates (21..27)
	gateClassificationEqualsPASS,
	gateInvocationCountEqualsOne,
	gateSubjectRootEqualsSExecutionRoot,
	gateSubjectExecutionRootMatchesTree,
	gateNotTimedOut,
	gateNoOutputTruncation,
	gateErrorAbsent,

	// binary predicates (28..38)
	binaryPathNonEmpty,
	binaryCommitEqualsSubjectCommit,
	binaryNotModified,
	binarySourceCommitEqualsSubjectCommit,
	binarySourceTreeEqualsSubjectTree,
	binarySourceClean,
	binarySourceDetached,
	binaryOutputOutsideAllWorktrees,
	binaryExecutable,
	binarySHA256Valid,
	binaryCleanupErrorAbsent,

	// caller predicates (39..47)
	callerBeforeAvailable,
	callerAfterAvailable,
	callerBeforeSnapshotComplete,
	callerAfterSnapshotComplete,
	callerHEADUnchanged,
	callerTreeUnchanged,
	callerStatusUnchanged,
	callerRefsUnchanged,
	callerWorktreeInventoryUnchanged,
}

// completenessPredicateNamesInOrder is the ordered list of
// predicate names parallel to completenessPredicatesInOrder.
// The mutation matrix asserts that every entry has at least
// one row. B2-R1 rearranged the slots:
//   - added runtimeExpectedChecksDerivedFromPlanBytes
//   - added gateSubjectExecutionRootMatchesTree
//   - replaced binaryAuthorityValid with binaryPathNonEmpty
//   - added callerBeforeSnapshotComplete and
//     callerAfterSnapshotComplete
var completenessPredicateNamesInOrder = []string{
	"runtime_identities_structurally_valid",
	"runtime_freeze_different_from_subject",
	"runtime_f_ancestor_of_s_verified",
	"runtime_execution_tree_equals_subject",
	"runtime_plan_blob_valid",
	"runtime_plan_sha256_valid",
	"runtime_plan_bytes_parse_successfully",
	"runtime_expected_checks_derived_from_plan_bytes",
	"plan_result_cardinality_equal",
	"plan_result_ids_bijective",
	"plan_result_order_matches_plan",
	"plan_result_mode_matches_plan",
	"plan_no_unknown_check_mode",
	"results_every_run_check_successful",
	"results_every_exclude_check_excluded",
	"results_no_timeout",
	"results_no_cancellation",
	"results_no_stdout_truncation",
	"results_no_stderr_truncation",
	"results_no_execution_cleanup_error",
	"gate_classification_equals_pass",
	"gate_invocation_count_equals_one",
	"gate_subject_root_equals_s_exec_root",
	"gate_subject_execution_root_matches_tree",
	"gate_not_timed_out",
	"gate_no_output_truncation",
	"gate_error_absent",
	"binary_path_non_empty",
	"binary_commit_equals_subject_commit",
	"binary_not_modified",
	"binary_source_commit_equals_subject_commit",
	"binary_source_tree_equals_subject_tree",
	"binary_source_clean",
	"binary_source_detached",
	"binary_output_outside_all_worktrees",
	"binary_executable",
	"binary_sha256_valid",
	"binary_cleanup_error_absent",
	"caller_before_available",
	"caller_after_available",
	"caller_before_snapshot_complete",
	"caller_after_snapshot_complete",
	"caller_head_unchanged",
	"caller_tree_unchanged",
	"caller_status_unchanged",
	"caller_refs_unchanged",
	"caller_worktree_inventory_unchanged",
}

// completenessPredicates is the closed set of named predicates.
// Every entry MUST have a matching branch in
// DeriveClosureEvidenceCompleteness and a matching mutation
// row in TestClosureEvidenceCompletenessCanonical. The map
// length is asserted by the test so a missing mutation row
// fails.
var completenessPredicates = func() map[string]func(ClosureEvidence) bool {
	out := make(map[string]func(ClosureEvidence) bool, len(completenessPredicatesInOrder))
	for i, fn := range completenessPredicatesInOrder {
		out[completenessPredicateNamesInOrder[i]] = fn
	}
	return out
}()

// completenessPredicateCount is the row count the mutation
// matrix must agree with. Keep in sync with completenessPredicatesInOrder.
// B2-R1 increased the count from 43 to 47: removed
// binaryAuthorityValid composite (-1), added five new
// atomic predicates (+5).
const completenessPredicateCount = 47

// ----------------------------------------------------------------------------
// Small shared helpers
// ----------------------------------------------------------------------------

// isValidOID reports whether s is a 40-char lower- or upper-case
// hex string. SHA-1 OIDs are exactly 40 hex chars.
func isValidOID(s string) bool {
	if len(s) != 40 {
		return false
	}
	for _, ch := range s {
		if !((ch >= '0' && ch <= '9') || (ch >= 'a' && ch <= 'f') || (ch >= 'A' && ch <= 'F')) {
			return false
		}
	}
	return true
}

// isHexSHA256 reports whether s is a 64-char lowercase hex
// digest.
func isHexSHA256(s string) bool {
	if len(s) != 64 {
		return false
	}
	for _, ch := range s {
		if !((ch >= '0' && ch <= '9') || (ch >= 'a' && ch <= 'f')) {
			return false
		}
	}
	return true
}
