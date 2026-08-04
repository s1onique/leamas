# Closure Plan Semantic Validation Taxonomy

This document maps every semantic validation error reachable from `ValidatePlan` to its canonical semantic path, code, and keyword. The taxonomy is authoritative: every new semantic error must be added here with its exact paths before being committed.

## Taxonomy Table

| Semantic Path | Code | Keyword | Error Type | Trigger |
|--------------|------|---------|------------|---------|
| `/execution/mode` | `semantic_constraint_failed` | `enum` | `*ExecutionModeError` | Missing, empty, whitespace, or unknown execution mode |
| `/policy/<field>` | `required_property_missing` | `required` | `*PlanPolicyRequiredError` | Missing required policy field |
| `/policy` | `semantic_constraint_failed` | `type` | `*PlanSemanticError` | Clean worktree requirement not met |
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
| `/checks/<i>/id` | `semantic_constraint_failed` | `pattern` | `*PlanSemanticError` | Invalid check ID format or placeholder |
| `/checks/<i>/mode` | `semantic_constraint_failed` | `type` | `*PlanSemanticError` | Unknown check mode |
| `/checks/<i>/reason` | `semantic_constraint_failed` | `required` | `*PlanSemanticError` | Exclude reason missing, multiline, too long, or placeholder |
| `/checks/<i>/argv` | `semantic_constraint_failed` | `type` | `*PlanSemanticError` | Empty argv or exceeds limit |
| `/checks/<i>/argv/<j>` | `semantic_constraint_failed` | `type` | `*PlanSemanticError` | Empty argv element, contains NUL, or placeholder |
| `/checks/<i>/working_directory` | `semantic_constraint_failed` | `type` | `*PlanSemanticError` | Invalid path (absolute, escapes, not clean) |
| `/checks/<i>/timeout_seconds` | `semantic_constraint_failed` | `type` | `*PlanSemanticError` | Timeout out of bounds |
| `/checks/<i>/environment` | `semantic_constraint_failed` | `type` | `*PlanSemanticError` | Nil environment or exceeds entry limit |
| `/artifacts/<i>/id` | `semantic_constraint_failed` | `pattern` | `*PlanSemanticError` | Invalid artifact ID format or placeholder |
| `/artifacts/<i>/path` | `semantic_constraint_failed` | `type` | `*PlanSemanticError` | Invalid path (absolute, escapes, not clean) |
| `/artifacts/<i>/required` | `semantic_constraint_failed` | `type` | `*PlanSemanticError` | Missing required field |
| `/artifacts/<i>/max_bytes` | `semantic_constraint_failed` | `type` | `*PlanSemanticError` | Non-positive max_bytes |
| `/artifacts/<i>/media_type` | `semantic_constraint_failed` | `type` | `*PlanSemanticError` | Empty or placeholder media type |

## Codes

| Code | Value | Meaning |
|------|-------|---------|
| `PlanCodeRequiredPropertyMissing` | `1001` | A required JSON property is absent |
| `PlanCodeSemanticConstraintFailed` | `2001` | A semantic constraint (enum, type, pattern, etc.) is violated |

## Keywords

| Keyword | Value | Meaning |
|---------|-------|---------|
| `KeywordRequired` | `"required"` | The property must be present |
| `KeywordEnum` | `"enum"` | The value must be one of the accepted values |
| `KeywordType` | `"type"` | The value must be of the correct type |
| `KeywordPattern` | `"pattern"` | The value must match the regex pattern |

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
    return []PlanValidationError{{...}}
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
