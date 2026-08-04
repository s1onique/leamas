# Closure Plan Semantic Validation Taxonomy

This document maps every semantic validation error reachable from `ValidatePlan`
to its canonical semantic path, code, and keyword. The taxonomy is
authoritative: every new semantic error must be added here with its exact
paths before being committed.

## Codes (String Wire Values)

| Code | Value | Meaning |
|------|-------|---------|
| `PlanCodeRequiredPropertyMissing` | `"required_property_missing"` | A required JSON property is absent |
| `PlanCodeSemanticConstraintFailed` | `"semantic_constraint_failed"` | A semantic constraint (enum, type, pattern, etc.) is violated |
| `PlanCodeInvalidEnum` | `"invalid_enum"` | Value not in the allowed enumeration |
| `PlanCodeInvalidType` | `"invalid_type"` | Value type does not match schema |
| `PlanCodeUnknownProperty` | `"unknown_property"` | Unexpected property in object |
| `PlanCodeDuplicateProperty` | `"duplicate_property"` | Property appears more than once |
| `PlanCodeInvalidJSON` | `"invalid_json"` | JSON syntax or parsing error |
| `PlanCodeUnsupportedContractVersion` | `"unsupported_contract_version"` | Contract version not supported |
| `PlanCodeDuplicateApplicabilityRule` | `"duplicate_applicability_rule"` | Descriptor field carries conflicting rules |

## Keywords

| Keyword | Value | Meaning |
|---------|-------|---------|
| `KeywordRequired` | `"required"` | The property must be present |
| `KeywordEnum` | `"enum"` | The value must be one of the accepted values |
| `KeywordType` | `"type"` | The value must be of the correct type |
| `KeywordPattern` | `"pattern"` | The value must match the regex pattern |
| `KeywordConst` | `"const"` | Value must equal constant |
| `KeywordMinItems` | `"minItems"` | Array must have at least N elements |
| `KeywordIfThenElse` | `"if/then/else"` | Conditional schema violation |
| `KeywordAdditionalProp` | `"additionalProperties"` | Unknown property found |

## Semantic Taxonomy (Plan Validation)

### Execution Mode Errors

| Semantic Path | Code | Keyword | Error Type | Trigger |
|--------------|------|---------|------------|---------|
| `/execution/mode` | `semantic_constraint_failed` | `enum` | `*ExecutionModeError` | Missing, empty, whitespace, or unknown execution mode |

### Policy Errors

| Semantic Path | Code | Keyword | Error Type | Trigger |
|--------------|------|---------|------------|---------|
| `/policy/<field>` | `required_property_missing` | `required` | `*PlanPolicyRequiredError` | Missing required policy field |

### Runner Authority Errors (Plan Declaration)

| Semantic Path | Code | Keyword | Error Type | Trigger |
|--------------|------|---------|------------|---------|
| `/runner_authority/mode` | `semantic_constraint_failed` | `type` | `*RunnerAuthorityError` | Unknown runner authority mode |
| `/runner_authority/tool` | `semantic_constraint_failed` | `type` | `*RunnerAuthorityError` | Tool block not allowed (subject_exact) or required (tool_release_exact) |
| `/runner_authority/tool/revision` | `semantic_constraint_failed` | `type` | `*RunnerAuthorityError` | Empty, wrong length, or non-hex revision |
| `/runner_authority/tool/tree_oid` | `semantic_constraint_failed` | `type` | `*RunnerAuthorityError` | Invalid OID length or non-hex |
| `/runner_authority/tool/binary_sha256` | `semantic_constraint_failed` | `type` | `*RunnerAuthorityError` | Empty, wrong length, or non-hex |
| `/runner_authority/vcs_revision` | `semantic_constraint_failed` | `type` | `*RunnerAuthorityError` | Revision mismatch in enforcement |
| `/runner_authority/vcs_modified` | `semantic_constraint_failed` | `type` | `*RunnerAuthorityError` | Runner built from modified sources |
| `/runner_authority/binary_sha256` | `semantic_constraint_failed` | `type` | `*RunnerAuthorityError` | Binary SHA256 mismatch |
| `/runner_authority/target_subject` | `semantic_constraint_failed` | `type` | `*RunnerAuthorityError` | Target subject commit empty |
| `/runner_authority/target_tree` | `semantic_constraint_failed` | `type` | `*RunnerAuthorityError` | Target tree empty |

### Check Errors

| Semantic Path | Code | Keyword | Error Type | Trigger |
|--------------|------|---------|------------|---------|
| `/act_id` | `semantic_constraint_failed` | `type` | `*PlanSemanticError` | Invalid ACT ID format or placeholder |
| `/baseline/commit_oid` | `semantic_constraint_failed` | `type` | `*PlanSemanticError` | Invalid commit OID format or placeholder |
| `/baseline/tree_oid` | `semantic_constraint_failed` | `type` | `*PlanSemanticError` | Invalid tree OID format or placeholder |
| `/checks/<i>/id` | `semantic_constraint_failed` | `pattern` | `*PlanSemanticError` | Invalid check ID format or placeholder |
| `/checks/<i>/mode` | `semantic_constraint_failed` | `type` | `*PlanSemanticError` | Unknown check mode |
| `/checks/<i>/reason` | `semantic_constraint_failed` | `required` | `*PlanSemanticError` | Exclude reason missing, multiline, too long, or placeholder |
| `/checks/<i>/argv` | `semantic_constraint_failed` | `type` | `*PlanSemanticError` | Empty argv or exceeds limit |
| `/checks/<i>/argv/<j>` | `semantic_constraint_failed` | `type` | `*PlanSemanticError` | Empty argv element, contains NUL, or placeholder |
| `/checks/<i>/working_directory` | `semantic_constraint_failed` | `type` | `*PlanSemanticError` | Invalid path (absolute, escapes, not clean) |
| `/checks/<i>/timeout_seconds` | `semantic_constraint_failed` | `type` | `*PlanSemanticError` | Timeout out of bounds |
| `/checks/<i>/environment` | `semantic_constraint_failed` | `type` | `*PlanSemanticError` | Nil environment or exceeds entry limit |

### Artifact Errors

| Semantic Path | Code | Keyword | Error Type | Trigger |
|--------------|------|---------|------------|---------|
| `/artifacts/<i>/id` | `semantic_constraint_failed` | `pattern` | `*PlanSemanticError` | Invalid artifact ID format or placeholder |
| `/artifacts/<i>/path` | `semantic_constraint_failed` | `type` | `*PlanSemanticError` | Invalid path (absolute, escapes, not clean) |
| `/artifacts/<i>/required` | `semantic_constraint_failed` | `required` | `*PlanSemanticError` | Missing required field |
| `/artifacts/<i>/max_bytes` | `semantic_constraint_failed` | `type` | `*PlanSemanticError` | Non-positive max_bytes |
| `/artifacts/<i>/media_type` | `semantic_constraint_failed` | `type` | `*PlanSemanticError` | Empty or placeholder media type |

## Structural Taxonomy (Parse/Decode Validation)

| Semantic Path | Code | Keyword | Trigger |
|--------------|------|---------|---------|
| (root) | `invalid_json` | `type` | JSON syntax error |
| `/contract_version` | `unsupported_contract_version` | `type` | Contract version not "1" |
| `/contract_version` | `invalid_type` | `type` | Contract version wrong type |
| `/checks` | `semantic_constraint_failed` | `minItems` | Empty checks array |
| `/artifacts` | `semantic_constraint_failed` | `minItems` | Artifacts exceeds limit |

## Runtime-Only Diagnostics

These diagnostics arise only at runtime observation, not from plan validation:

| Semantic Path | Code | Keyword | Trigger |
|--------------|------|---------|---------|
| N/A (runtime) | N/A | N/A | Runner binary SHA256 mismatch |
| N/A (runtime) | N/A | N/A | VCS revision mismatch |
| N/A (runtime) | N/A | N/A | VCS modified sources detected |

## Diagnostic Extraction

Every typed semantic error implements `planDiagnosticSource`:

```go
type planDiagnosticSource interface {
    PlanDiagnostics() []PlanValidationError
}
```

The composed pipeline uses `errors.As` to extract diagnostics:

```go
func semanticDiagnostics(err error) []PlanValidationError {
    var source planDiagnosticSource
    if errors.As(err, &source) {
        return clonePlanValidationErrors(source.PlanDiagnostics())
    }
    // Fallback for unknown errors
    return []PlanValidationError{{
        InstancePath: "",
        Code:        PlanCodeSemanticConstraintFailed,
        Keyword:     KeywordType,
        Message:     err.Error(),
    }}
}
```

## Mutation Isolation

`PlanDiagnostics()` returns deep copies of diagnostics. Callers can mutate the returned `AcceptedValues` slice without affecting the error's internal state.

## File Organization

- `plan_semantic_error.go` - `PlanSemanticError`, `PlanSemanticMultiError`
- `plan_semantic_composed.go` - `semanticDiagnostics` extractor
- `execution_mode.go` - `ExecutionModeError`
- `runner_authority.go` - `RunnerAuthorityError`
- `plan_policy.go` - `PlanPolicyRequiredError`
- `plan_semantic_clone.go` - `clonePlanValidationError`, `clonePlanValidationErrors`
- `plan_semantic_pointer.go` - JSON Pointer helpers
- `plan_semantic_validation_*.go` - Semantic validation functions

## Test Reconciliation

The semantic taxonomy document is reconciled with Go constants in `plan_semantic_matrix_test.go`:

```go
// TestSemanticPathMatrix verifies every error type's InstancePath and Code
// against the documented taxonomy.
```
