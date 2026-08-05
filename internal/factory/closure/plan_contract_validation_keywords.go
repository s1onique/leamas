package closure

// plan_contract_validation_keywords.go centralises the
// PlanValidationKeyword and PlanValidationCode types and their
// closed constant sets. Splitting these declarations out of
// plan_contract_validation.go keeps the main validator file under
// the LLM-friendly 400-line threshold while preserving the single
// closure over the descriptor that
// ACT-LEAMAS-FACTORY-CLOSE-PLAN-CONTRACT-AUTHORITY01 requires.

// PlanValidationKeyword mirrors the JSON Schema keyword taxonomy
// and adds stable Leamas extensions for value-constraint
// classifications. JSON Schema's type/minLength/pattern/minimum/
// maximum keywords are kept as the public carrier when they
// accurately apply; path-policy violations that JSON Schema cannot
// express portably are surfaced under x-leamas-path-policy so a
// source-free consumer can distinguish them from generic
// invalid_type failures.
type PlanValidationKeyword string

const (
	KeywordType           PlanValidationKeyword = "type"
	KeywordEnum           PlanValidationKeyword = "enum"
	KeywordRequired       PlanValidationKeyword = "required"
	KeywordConst          PlanValidationKeyword = "const"
	KeywordPattern        PlanValidationKeyword = "pattern"
	KeywordAdditionalProp PlanValidationKeyword = "additionalProperties"
	KeywordMinItems       PlanValidationKeyword = "minItems"
	KeywordIfThenElse     PlanValidationKeyword = "if"
	KeywordMinLength      PlanValidationKeyword = "minLength"
	KeywordMinimum        PlanValidationKeyword = "minimum"
	KeywordMaximum        PlanValidationKeyword = "maximum"
	// KeywordPathPolicy is the stable Leamas extension for
	// repository-relative path rules JSON Schema cannot express
	// portably (no absolute paths, no parent traversal,
	// lexically clean). The keyword name mirrors the JSON Schema
	// convention of prefixing Leamas extensions with
	// x-leamas-.
	KeywordPathPolicy PlanValidationKeyword = "x-leamas-path-policy"
)

// PlanValidationCode is the stable, machine-readable diagnostic
// code emitted by the structural validator. The closed set is
// documented in ACT-LEAMAS-FACTORY-CLOSE-PLAN-CONTRACT-AUTHORITY01
// and ACT-LEAMAS-FACTORY-CLOSE-PLAN-CONTRACT-AUTHORITY01-
// CORRECTION01-STRUCTURAL-PARITY-CLOSURE01.
type PlanValidationCode string

const (
	PlanCodeInvalidJSON                PlanValidationCode = "invalid_json"
	PlanCodeUnsupportedContractVersion PlanValidationCode = "unsupported_contract_version"
	PlanCodeRequiredPropertyMissing    PlanValidationCode = "required_property_missing"
	PlanCodeInvalidType                PlanValidationCode = "invalid_type"
	PlanCodeInvalidEnum                PlanValidationCode = "invalid_enum"
	PlanCodeUnknownProperty            PlanValidationCode = "unknown_property"
	PlanCodeDuplicateProperty          PlanValidationCode = "duplicate_property"
	PlanCodeSemanticConstraintFailed   PlanValidationCode = "semantic_constraint_failed"
	// PlanCodeValueBelowMinLength reports that a supplied string
	// is shorter than the descriptor's MinLength. Replaces the
	// generic invalid_type classification so consumers can
	// distinguish length from type failures.
	PlanCodeValueBelowMinLength PlanValidationCode = "value_below_min_length"
	// PlanCodeValuePatternMismatch reports that a supplied
	// string fails the descriptor's Pattern. Replaces the
	// generic invalid_type classification so consumers can
	// distinguish pattern from type failures.
	PlanCodeValuePatternMismatch PlanValidationCode = "pattern_mismatch"
	// PlanCodeNumericBelowMinimum reports that a supplied integer
	// is below the descriptor's inclusive Minimum bound.
	PlanCodeNumericBelowMinimum PlanValidationCode = "numeric_below_minimum"
	// PlanCodeNumericAboveMaximum reports that a supplied integer
	// is above the descriptor's inclusive Maximum bound.
	PlanCodeNumericAboveMaximum PlanValidationCode = "numeric_above_maximum"
	// PlanCodePathPolicyViolation reports that a supplied
	// repository-relative path violates a path-policy rule that
	// JSON Schema cannot express portably (parent traversal,
	// lexical cleanliness, etc.).
	PlanCodePathPolicyViolation PlanValidationCode = "path_policy_violation"
	// PlanCodeForbiddenPresence reports that a field with a
	// PresenceForbidden ApplicabilityRule is present. The code
	// dominates the value-constraint classifications so a
	// forbidden field is reported consistently regardless of
	// the supplied value.
	PlanCodeForbiddenPresence PlanValidationCode = "forbidden_presence"
	// PlanCodeDuplicateApplicabilityRule is the stable code
	// raised when a descriptor field carries two or more
	// ApplicabilityRule entries that share (Sibling, Value).
	// Two rules for the same condition are ambiguous: a
	// descriptor inventory must declare at most one rule per
	// (Sibling, Value) pair so the walker and the example
	// generator agree on the contract.
	PlanCodeDuplicateApplicabilityRule PlanValidationCode = "duplicate_applicability_rule"
)
