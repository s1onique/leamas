package closure

import "fmt"

// plan_semantic_validation_root.go contains typed error constructors
// for root-level semantic validations: contract version, act_id, baseline.

// errUnsupportedContractVersion returns a typed error for unsupported contract version.
func errUnsupportedContractVersion(version int) *PlanSemanticError {
	return newSemanticError(
		"/contract_version",
		PlanCodeSemanticConstraintFailed,
		KeywordConst,
		fmt.Sprintf("unsupported closure plan contract_version %d", version),
		fmt.Errorf("unsupported closure plan contract_version %d", version),
	)
}

// errInvalidActID returns a typed error for invalid act_id.
func errInvalidActID(actID string) *PlanSemanticError {
	return newSemanticError(
		"/act_id",
		PlanCodeSemanticConstraintFailed,
		KeywordType,
		fmt.Sprintf("invalid act_id %q", actID),
		fmt.Errorf("invalid act_id %q", actID),
	)
}

// errInvalidBaselineCommitOID returns a typed error for invalid commit OID.
func errInvalidBaselineCommitOID(commitOID string) *PlanSemanticError {
	cause := fmt.Errorf("baseline.commit_oid: must be a full lowercase Git OID")
	return newSemanticError(
		"/baseline/commit_oid",
		PlanCodeSemanticConstraintFailed,
		KeywordType,
		"baseline.commit_oid must be a full lowercase Git OID",
		cause,
	)
}

// errInvalidBaselineTreeOID returns a typed error for invalid tree OID.
func errInvalidBaselineTreeOID(treeOID string) *PlanSemanticError {
	cause := fmt.Errorf("baseline.tree_oid: must be a full lowercase Git OID")
	return newSemanticError(
		"/baseline/tree_oid",
		PlanCodeSemanticConstraintFailed,
		KeywordType,
		"baseline.tree_oid must be a full lowercase Git OID",
		cause,
	)
}

// errBaselineCommitOIDPlaceholder returns a typed error for commit OID with placeholder.
func errBaselineCommitOIDPlaceholder() *PlanSemanticError {
	return newSemanticError(
		"/baseline/commit_oid",
		PlanCodeSemanticConstraintFailed,
		KeywordType,
		"baseline.commit_oid contains a closure placeholder",
		fmt.Errorf("baseline.commit_oid contains a closure placeholder"),
	)
}

// errBaselineTreeOIDPlaceholder returns a typed error for tree OID with placeholder.
func errBaselineTreeOIDPlaceholder() *PlanSemanticError {
	return newSemanticError(
		"/baseline/tree_oid",
		PlanCodeSemanticConstraintFailed,
		KeywordType,
		"baseline.tree_oid contains a closure placeholder",
		fmt.Errorf("baseline.tree_oid contains a closure placeholder"),
	)
}

// errInvalidChecksCount returns a typed error for invalid checks count.
func errInvalidChecksCount(count int) *PlanSemanticError {
	return newSemanticError(
		"/checks",
		PlanCodeSemanticConstraintFailed,
		KeywordMinItems,
		fmt.Sprintf("checks count must be between 1 and %d", MaxChecks),
		fmt.Errorf("checks count must be between 1 and %d", MaxChecks),
	)
}

// errInvalidArtifactsCount returns a typed error for invalid artifacts count.
func errInvalidArtifactsCount(count int) *PlanSemanticError {
	return newSemanticError(
		"/artifacts",
		PlanCodeSemanticConstraintFailed,
		KeywordMinItems,
		fmt.Sprintf("artifacts count exceeds %d", MaxArtifacts),
		fmt.Errorf("artifacts count exceeds %d", MaxArtifacts),
	)
}

// validateBaselineCommitOID validates the baseline commit OID using exact field identity.
// No unknown field can fall through to a root fallback.
func validateBaselineCommitOID(commitOID string) error {
	if containsClosurePlaceholder(commitOID) {
		return errBaselineCommitOIDPlaceholder()
	}
	if !oidPattern.MatchString(commitOID) {
		return errInvalidBaselineCommitOID(commitOID)
	}
	return nil
}

// validateBaselineTreeOID validates the baseline tree OID using exact field identity.
// No unknown field can fall through to a root fallback.
func validateBaselineTreeOID(treeOID string) error {
	if containsClosurePlaceholder(treeOID) {
		return errBaselineTreeOIDPlaceholder()
	}
	if !oidPattern.MatchString(treeOID) {
		return errInvalidBaselineTreeOID(treeOID)
	}
	return nil
}
