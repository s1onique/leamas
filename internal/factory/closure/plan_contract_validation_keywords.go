package closure

// plan_contract_validation_keywords.go centralises the
// PlanValidationKeyword type and its closed constant set. Splitting
// the keywords out of plan_contract_validation.go keeps the main
// validator file under the LLM-friendly 400-line threshold while
// preserving the single closure over the descriptor that
// ACT-LEAMAS-FACTORY-CLOSE-PLAN-CONTRACT-AUTHORITY01 requires.

// PlanValidationKeyword mirrors the JSON Schema keyword taxonomy.
// The closed set is extended to cover the v1 schema's value-level
// constraints (minLength, minimum, maximum) so the structural
// validator, the JSON Schema generator, and downstream consumers
// agree on the keyword vocabulary.
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
)
