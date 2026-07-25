// SPDX-License-Identifier: Apache-2.0

package closure

import (
	"context"
	"fmt"
)

// PolicyRangeDecision (Closure Protocol v2) is the explicit, named
// statement of which commit ranges the two orchestrator-mandated
// policies MUST scan. ProvenanceTopology (run_v2_authority.go)
// defines the four named roles B, P, F, S.
//
// The shared word "base" must NEVER select one of these by accident;
// this is the source of the historical B↔P naming confusion that
// motivated this fix. The decision is:
//
//	subject-only patch hygiene    : F..S    (freeze commit F,
//	                                        not plan.baseline B)
//	historical ACT scope policy   : plan.baseline..S  (B..S, the
//	                                        full historical range
//	                                        that any new tracked
//	                                        full digest would land
//	                                        in)
//
// F..S is the implementation-only range: the patch under review is
// the change introduced by THIS subject against its own freeze. B..S
// is the historical range: any tracked full digest introduced since
// plan.baseline must be rejected because it pollutes the long-term
// evidence chain.
const (
	policyRangePatchHygiene  = "F..S"
	policyRangeClosurePolicy = "plan.baseline..S"
)

type policyEvaluation[T any] struct {
	Value       T
	Diagnostics []byte
	Passed      bool
}

// evaluateRequiredPatchHygieneV2 runs the SUBJECT-ONLY patch
// hygiene policy on the F..S range. The freeze commit is passed
// explicitly so the implementation cannot accidentally fall back to
// plan.baseline when the two are distinct (they typically are; see
// ProvenanceTopology).
func evaluateRequiredPatchHygieneV2(
	ctx context.Context,
	git gitClient,
	root string,
	plan Plan,
	freezeCommit string,
	subject string,
) (policyEvaluation[PatchHygiene], error) {
	if plan.Policy.RequireDiffCheck == nil || !*plan.Policy.RequireDiffCheck {
		value := PatchHygiene{Status: CheckStatusPass}
		return policyEvaluation[PatchHygiene]{Value: value, Passed: true}, nil
	}
	value, diagnostics := evaluatePatchHygiene(ctx, git, root, freezeCommit, subject)
	outcome := policyEvaluation[PatchHygiene]{
		Value:       value,
		Diagnostics: diagnostics,
		Passed:      value.Status == CheckStatusPass,
	}
	if !outcome.Passed {
		return outcome, fmt.Errorf("required patch hygiene failed")
	}
	return outcome, nil
}

func recordFailedPolicyDiagnostics(evidenceDir, actID string, patch, policy []byte, cause error) error {
	if _, err := writeRunnerDiagnostics(evidenceDir, actID, patch, policy); err != nil {
		return fmt.Errorf("%w; write policy diagnostics: %v", cause, err)
	}
	return cause
}

// evaluateRequiredClosurePolicyV2 runs the HISTORICAL ACT SCOPE
// tracked-digest policy on the plan.baseline..S range. This is the
// only one of the two orchestrator policies that intentionally uses
// plan.baseline instead of the freeze commit, because a tracked full
// digest is by definition a long-term pollution that must be
// detected regardless of which freeze introduced it.
func evaluateRequiredClosurePolicyV2(
	ctx context.Context,
	git gitClient,
	root string,
	plan Plan,
	subject string,
) (policyEvaluation[ClosurePolicyResult], error) {
	if plan.Policy.ForbidTrackedFullDigests == nil || !*plan.Policy.ForbidTrackedFullDigests {
		value := ClosurePolicyResult{TrackedFullDigestStatus: CheckStatusPass}
		return policyEvaluation[ClosurePolicyResult]{Value: value, Passed: true}, nil
	}
	value, diagnostics := evaluateTrackedDigestPolicy(ctx, git, root, plan.Baseline.CommitOID, subject)
	outcome := policyEvaluation[ClosurePolicyResult]{
		Value:       value,
		Diagnostics: diagnostics,
		Passed:      value.TrackedFullDigestStatus == CheckStatusPass,
	}
	if !outcome.Passed {
		return outcome, fmt.Errorf("required closure policy failed")
	}
	return outcome, nil
}
