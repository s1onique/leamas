package closure

import "fmt"

// plan_semantic_validation_artifacts.go contains typed error constructors
// for artifacts validation.

// errInvalidArtifactID returns a typed error for invalid artifact ID.
func errInvalidArtifactID(index int, id string) *PlanSemanticError {
	path := jsonPointerArtifactID(index, "id")
	return newSemanticError(
		path,
		PlanCodeSemanticConstraintFailed,
		KeywordType,
		fmt.Sprintf("artifacts[%d].id is invalid", index),
		fmt.Errorf("artifacts[%d].id is invalid", index),
	)
}

// errDuplicateArtifactID returns a typed error pointing to the duplicate occurrence.
func errDuplicateArtifactID(duplicateIndex int, id string) *PlanSemanticError {
	path := jsonPointerArtifactID(duplicateIndex, "id")
	return newSemanticError(
		path,
		PlanCodeSemanticConstraintFailed,
		KeywordType,
		fmt.Sprintf("duplicate artifact id %q", id),
		fmt.Errorf("duplicate artifact id %q", id),
	)
}

// errInvalidArtifactPath returns a typed error for invalid artifact path.
func errInvalidArtifactPath(index int) *PlanSemanticError {
	path := jsonPointerArtifactID(index, "path")
	cause := fmt.Errorf("artifacts[%d].path: must be a non-empty repository-relative path", index)
	return newSemanticError(
		path,
		PlanCodeSemanticConstraintFailed,
		KeywordType,
		fmt.Sprintf("artifacts[%d].path: must be a non-empty repository-relative path", index),
		cause,
	)
}

// errMissingArtifactRequired returns a typed error for missing required field.
func errMissingArtifactRequired(index int) *PlanSemanticError {
	path := jsonPointerArtifactID(index, "required")
	return newSemanticError(
		path,
		PlanCodeSemanticConstraintFailed,
		KeywordRequired,
		fmt.Sprintf("artifacts[%d].required is missing", index),
		fmt.Errorf("artifacts[%d].required is missing", index),
	)
}

// errInvalidArtifactMaxBytes returns a typed error for invalid max_bytes.
func errInvalidArtifactMaxBytes(index int) *PlanSemanticError {
	path := jsonPointerArtifactID(index, "max_bytes")
	return newSemanticError(
		path,
		PlanCodeSemanticConstraintFailed,
		KeywordType,
		fmt.Sprintf("artifacts[%d].max_bytes must be positive", index),
		fmt.Errorf("artifacts[%d].max_bytes must be positive", index),
	)
}

// errInvalidArtifactMediaType returns a typed error for invalid media_type.
func errInvalidArtifactMediaType(index int) *PlanSemanticError {
	path := jsonPointerArtifactID(index, "media_type")
	return newSemanticError(
		path,
		PlanCodeSemanticConstraintFailed,
		KeywordType,
		fmt.Sprintf("artifacts[%d].media_type is invalid", index),
		fmt.Errorf("artifacts[%d].media_type is invalid", index),
	)
}

// errInvalidArtifactRole returns a typed error for invalid artifact role.
func errInvalidArtifactRole(index int, role string) *PlanSemanticError {
	path := jsonPointerArtifactID(index, "role")
	return newSemanticError(
		path,
		PlanCodeSemanticConstraintFailed,
		KeywordEnum,
		fmt.Sprintf("artifacts[%d].role %q is invalid", index, role),
		fmt.Errorf("artifacts[%d].role %q is invalid", index, role),
	)
}
