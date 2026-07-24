# Close Report: ACT-LEAMAS-FACTORY-CLOSURE-PROTOCOL-V1-ADOPTION01-CORRECTION04

## Summary

Operational CLI convergence with proper Git revision expressions and manifest authority for Closure Protocol V1.

## Files Changed

| File | Change |
|------|--------|
| `internal/factory/closure/validation.go` | Proper `^{type}` Git revision syntax |
| `internal/factory/closure/manifest.go` | Manifest loading, strict validation, SHA256Hex |
| `internal/factory/closure/plan_validation.go` | ValidatePlanFromBytes |
| `internal/factory/closure/chain.go` | ChainValidationRequest, types |

## Behavior Changed

### 1. Git Typed Revision Syntax

All Git revision expressions now use proper `^{type}` syntax:

| Operation | Expression |
|-----------|------------|
| Commit resolution | `F^{commit}`, `S^{commit}`, `C^{commit}` |
| Tree resolution | `F^{tree}`, `S^{tree}`, `C^{tree}` |
| Tag object | `refs/tags/<tag>^{tag}` |
| Tag peeled | `refs/tags/<tag>^{commit}` |

### 2. Repository Authority Internal to CLI

The CLI constructs the real Git client and derives repository root via `git rev-parse --show-toplevel`. Unknown storage format detection fails closed.

### 3. Strict Manifest Validation

`DecodeManifest` uses `json.Decoder.DisallowUnknownFields()` to reject unknown fields.

`ValidateManifestStrict` validates:
- `contract_version == 1`
- `act_id` non-empty
- `plan.path` non-empty, relative path
- `plan.sha256` is 64-char hex
- Freeze/subject OIDs are valid SHA-1
- Verdict is `pass` or `fail`

### 4. Annotated Tag Enforcement

`getTagInfo` requires:
1. `refs/tags/<tag>` exists
2. `refs/tags/<tag>^{tag}` resolves (annotated tag required)
3. `refs/tags/<tag>^{commit}` returns peeled target

### 5. Ancestry Error Propagation

`isAncestor` returns errors distinctly:
- `exit 0` → ancestor (true)
- `exit 1` → valid negative (false)
- `other` → Git execution error

## Verification Results

| Command | Result |
|---------|--------|
| `go test ./internal/factory/closure/...` | PASS |
| `make gate-fast` | PASS |

## Acceptance Criteria

| # | Criterion | Status |
|-:|-----------|--------|
| 1 | All Git typed revisions use valid `^{type}` syntax | ✓ |
| 2 | Real CLI initializes repository/Git execution authority | ✓ |
| 3 | Manifest strict validation with unknown field rejection | ✓ |
| 4 | Annotated tags enforced via `^{tag}` resolution | ✓ |
| 5 | Ancestry errors distinguishable from negative results | ✓ |
| 6 | All tests pass | ✓ |

## Successor

`ACT-LEAMAS-GATE-FAST-LONG-EXECUTION-BOUNDARY-CORRECTION01`
