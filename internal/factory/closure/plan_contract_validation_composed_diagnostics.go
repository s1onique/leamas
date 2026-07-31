package closure

import "strings"

// plan_contract_validation_composed_diagnostics.go centralises
// the diagnostic-mapping helpers the composed pipeline uses.
// Splitting them out keeps plan_contract_validation_composed.go
// under the LLM-friendly 400-line threshold while every helper
// remains reviewable in one place.

// typedDecodeDiagnostic wraps a typed-decode error as a
// decode-stage diagnostic. The typed-decode failure lives in the
// decode_errors stage array of the composed result; precise paths
// are deferred to a later correction.
func typedDecodeDiagnostic(err error) PlanValidationError {
	return PlanValidationError{
		InstancePath: "",
		SchemaPath:   "",
		Code:         PlanCodeInvalidType,
		Keyword:      KeywordType,
		Message:      "typed decode: " + err.Error(),
	}
}

// semanticDiagnostic maps a semantic Go error to a structured
// diagnostic. The path is /act_id for ActID errors, /baseline for
// baseline errors, /execution/mode for execution mode errors, and
// the root for everything else; the field/codename is preserved in
// the message for human readers.
func semanticDiagnostic(err error) PlanValidationError {
	msg := err.Error()
	path := semanticPathFromError(msg)
	return PlanValidationError{
		InstancePath: path,
		SchemaPath:   path,
		Code:         PlanCodeSemanticConstraintFailed,
		Keyword:      KeywordType,
		Message:      msg,
	}
}

// semanticPathFromError extracts the most precise instance path
// documented in the error message. The mapping is conservative;
// unknown forms fall back to the root.
func semanticPathFromError(msg string) string {
	switch {
	case strings.HasPrefix(msg, "invalid act_id"):
		return "/act_id"
	case strings.HasPrefix(msg, "baseline.commit_oid"), strings.HasPrefix(msg, "baseline.tree_oid"):
		return "/" + strings.SplitN(msg, " ", 2)[0]
	case strings.HasPrefix(msg, "checks[") && strings.Contains(msg, "duplicate check id"):
		// duplicate check id "<id>": caller extracts <id> from the message.
		return extractDuplicateCheckIDPath(msg)
	case strings.HasPrefix(msg, "unsupported closure plan contract_version"):
		return "/contract_version"
	}
	return ""
}

func extractDuplicateCheckIDPath(msg string) string {
	const marker = "duplicate check id \""
	i := strings.Index(msg, marker)
	if i < 0 {
		return ""
	}
	rest := msg[i+len(marker):]
	j := strings.Index(rest, "\"")
	if j < 0 {
		return ""
	}
	return "/checks/" + rest[:j]
}
