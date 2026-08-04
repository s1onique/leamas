package closure

import "fmt"

// plan_semantic_validation_checks.go contains typed error constructors
// for checks validation.

// errInvalidCheckID returns a typed error for invalid check ID.
func errInvalidCheckID(index int, id string) *PlanSemanticError {
	path := jsonPointerCheckID(index, "id")
	return newSemanticError(
		path,
		PlanCodeSemanticConstraintFailed,
		KeywordType,
		fmt.Sprintf("checks[%d].id is invalid", index),
		fmt.Errorf("checks[%d].id is invalid", index),
	)
}

// errDuplicateCheckID returns a typed error pointing to the duplicate occurrence.
func errDuplicateCheckID(duplicateIndex int, id string) *PlanSemanticError {
	path := jsonPointerCheckID(duplicateIndex, "id")
	return newSemanticError(
		path,
		PlanCodeSemanticConstraintFailed,
		KeywordType,
		fmt.Sprintf("duplicate check id %q", id),
		fmt.Errorf("duplicate check id %q", id),
	)
}

// errUnknownCheckMode returns a typed error for unknown check mode.
func errUnknownCheckMode(index int, mode string) *PlanSemanticError {
	path := jsonPointerCheckID(index, "mode")
	return newSemanticError(
		path,
		PlanCodeSemanticConstraintFailed,
		KeywordEnum,
		fmt.Sprintf("checks[%d] has unknown mode %q", index, mode),
		fmt.Errorf("checks[%d] has unknown mode %q", index, mode),
	)
}

// errInvalidCheckArgvCount returns a typed error for invalid argv count.
func errInvalidCheckArgvCount(index int) *PlanSemanticError {
	path := jsonPointerCheckID(index, "argv")
	return newSemanticError(
		path,
		PlanCodeSemanticConstraintFailed,
		KeywordMinItems,
		fmt.Sprintf("checks[%d].argv count must be between 1 and %d", index, MaxArgvElements),
		fmt.Errorf("checks[%d].argv count must be between 1 and %d", index, MaxArgvElements),
	)
}

// errInvalidCheckArgvElement returns a typed error for invalid argv element.
func errInvalidCheckArgvElement(checkIndex, argIndex int) *PlanSemanticError {
	path := jsonPointerArgvElement(checkIndex, argIndex)
	return newSemanticError(
		path,
		PlanCodeSemanticConstraintFailed,
		KeywordType,
		fmt.Sprintf("checks[%d].argv[%d] is invalid or contains a placeholder", checkIndex, argIndex),
		fmt.Errorf("checks[%d].argv[%d] is invalid or contains a placeholder", checkIndex, argIndex),
	)
}

// errInvalidCheckWorkingDirectory returns a typed error for invalid working directory.
func errInvalidCheckWorkingDirectory(index int) *PlanSemanticError {
	path := jsonPointerCheckID(index, "working_directory")
	cause := fmt.Errorf("checks[%d].working_directory: must be a non-empty repository-relative path", index)
	return newSemanticError(
		path,
		PlanCodeSemanticConstraintFailed,
		KeywordType,
		fmt.Sprintf("checks[%d].working_directory: must be a non-empty repository-relative path", index),
		cause,
	)
}

// errInvalidCheckTimeout returns a typed error for invalid timeout.
func errInvalidCheckTimeout(index int) *PlanSemanticError {
	path := jsonPointerCheckID(index, "timeout_seconds")
	return newSemanticError(
		path,
		PlanCodeSemanticConstraintFailed,
		KeywordType,
		fmt.Sprintf("checks[%d].timeout_seconds must be between 1 and %d", index, MaxCheckTimeoutSeconds),
		fmt.Errorf("checks[%d].timeout_seconds must be between 1 and %d", index, MaxCheckTimeoutSeconds),
	)
}

// errInvalidCheckEnvironment returns a typed error for invalid environment.
func errInvalidCheckEnvironment(index int) *PlanSemanticError {
	path := jsonPointerCheckID(index, "environment")
	return newSemanticError(
		path,
		PlanCodeSemanticConstraintFailed,
		KeywordType,
		fmt.Sprintf("checks[%d].environment must be an object with at most %d entries", index, MaxEnvironmentEntries),
		fmt.Errorf("checks[%d].environment must be an object with at most %d entries", index, MaxEnvironmentEntries),
	)
}

// errInvalidCheckEnvironmentKey returns a typed error for invalid environment key.
func errInvalidCheckEnvironmentKey(checkIndex int, key string) *PlanSemanticError {
	path := jsonPointerCheckID(checkIndex, "environment") + "/" + jsonPointerToken(key)
	return newSemanticError(
		path,
		PlanCodeSemanticConstraintFailed,
		KeywordType,
		fmt.Sprintf("checks[%d].environment contains invalid entry %q", checkIndex, key),
		fmt.Errorf("checks[%d].environment contains invalid entry %q", checkIndex, key),
	)
}

// errInvalidCheckReason returns a typed error for invalid/missing exclusion reason.
func errInvalidCheckReason(index int) *PlanSemanticError {
	path := jsonPointerCheckID(index, "reason")
	return newSemanticError(
		path,
		PlanCodeSemanticConstraintFailed,
		KeywordType,
		fmt.Sprintf("checks[%d].reason is required and must be compact final prose", index),
		fmt.Errorf("checks[%d].reason is required and must be compact final prose", index),
	)
}

// errExclusionWithExecutionFields returns a typed error for exclusion with execution fields.
func errExclusionWithExecutionFields(index int) *PlanSemanticError {
	path := jsonPointerCheckID(index, "argv")
	return newSemanticError(
		path,
		PlanCodeSemanticConstraintFailed,
		KeywordType,
		fmt.Sprintf("checks[%d] exclusion contains execution fields", index),
		fmt.Errorf("checks[%d] exclusion contains execution fields", index),
	)
}

// errRunnableCheckWithReason returns a typed error for runnable check with reason.
func errRunnableCheckWithReason(index int) *PlanSemanticError {
	path := jsonPointerCheckID(index, "reason")
	return newSemanticError(
		path,
		PlanCodeSemanticConstraintFailed,
		KeywordType,
		fmt.Sprintf("checks[%d] runnable check contains exclusion reason", index),
		fmt.Errorf("checks[%d] runnable check contains exclusion reason", index),
	)
}
