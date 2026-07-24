# Close Report: ACT-LEAMAS-FACTORY-CLOSURE-PROTOCOL-V1-ADOPTION01-CORRECTION03

## Summary

Repository-bound Closure Protocol V1 implementation with mechanical chain verification. All verdicts derived from Git operations rather than supplied strings.

## Files Changed

| File | Change |
|------|--------|
| `internal/factory/closure/chain.go` | OID validation, attestation types, plan self-reference check |
| `internal/factory/closure/validation.go` | Repository-bound chain verification |
| `internal/factory/closure/plan_validation.go` | Recursive forbidden key detection |
| `internal/factory/closure/plan_validation_test.go` | Plan structure tests |
| `cmd/leamas/factory_close.go` | Chain/attest CLI commands |

## Behavior Changed

### 1. Repository Storage Format Authority

`DetectStorageFormat` queries `git rev-parse --show-object-format=storage`:

| Format | OID length |
|--------|------------|
| `sha1` | 40 hex chars |
| `sha256` | 64 hex chars |

### 2. Git-Bound Object Resolution

Each identity resolved and type-checked through Git:
- `git rev-parse --verify <rev>^{commit}` → commit type
- `git rev-parse --verify <rev>^{tree}` → tree type
- `git cat-file -t <oid>` → object type verification

### 3. Mechanical Chain Verification

`VerifyChain` proves:
- `F != S`
- `F` is ancestor of `S`
- `F` is ancestor of `C`
- `S` is ancestor of `C`
- Plan bytes at `F` equal plan bytes at `S`
- Tag is annotated
- Tag peeled target equals `C`

### 4. Structural Plan Validation

`ValidatePlanStructure` recursively inspects JSON keys, rejects:
- `freeze_commit`, `freeze_tree`
- `subject_commit`, `subject_tree`
- `closure_commit`, `closure_tree`
- `tag_oid`, `tag_object_oid`, `tag_target`, `peeled_target`

Regardless of value (SHA-1, SHA-256, null, number, array, nested).

## Verification Results

| Command | Result |
|---------|--------|
| `go test ./internal/factory/closure/...` | PASS |
| `make gate-fast` | PASS |
| `bin/leamas factory close chain --help` | PASS |
| `bin/leamas factory close attest --help` | PASS |

## Acceptance Criteria

| # | Criterion | Status |
|-:|-----------|--------|
| 1 | Repository storage format controls OID validation | ✓ |
| 2 | Every identity resolved and type-checked | ✓ |
| 3 | Ancestry, trees, plan bytes mechanically verified | ✓ |
| 4 | Lightweight tags rejected | ✓ |
| 5 | Forbidden plan keys rejected structurally | ✓ |
| 6 | All tests pass | ✓ |

## Files Structure

```
internal/factory/closure/
├── chain.go           # OID validation, types, plan check
├── validation.go      # Git-bound verification
├── plan_validation.go # Structural key detection
└── *_test.go         # Tests

cmd/leamas/
└── factory_close.go   # CLI commands
```

## Successor

`ACT-LEAMAS-GATE-FAST-LONG-EXECUTION-BOUNDARY-CORRECTION01`
