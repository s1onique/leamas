# ACT-LEAMAS-GATE-SUMMARY-V2-DIGEST01 Close Report

## Status: CLOSED

## Close Identity

| Field | Value |
|-------|-------|
| close_commit_oid | cc596f99585a05112f8b587ac59071b5d55abf06 |
| close_tree_oid | 7596eb792b417e53a5b6c8e6f004756fca8fb3a9 |
| tag_object_oid | 5c1b078b3d961d5d3d49ac7056d85921e74fadd6 |
| tag_name | act/leamas-gate-summary-v2-digest01 |
| tag_points_to | cc596f9 (close commit) |
| worktree | CLEAN (git status --porcelain=v1 = empty) |

## Summary

Implemented shared gate_summary adapter for digest with v1/v2 rendering, arbitrary-precision integer comparison, and full test coverage.

## Files Changed

### Production Code
- `internal/factory/digest/digest.go` - Refactored to use shared adapter
- `internal/factory/digest/range.go` - Refactored to use shared adapter
- `internal/factory/digest/gate_summary.go` - New shared adapter
- `internal/factory/digest/gate_summary_render.go` - V2 rendering logic
- `internal/factory/digest/gate_summary_render_v1.go` - V1 rendering logic
- `internal/factory/digest/gate_summary_sanitize.go` - UTF-8 sanitization

### Test Code
- `internal/factory/digest/gate_summary_canonical_test.go` - Source-order canonicalization tests
- `internal/factory/digest/gate_summary_hash_basic_test.go` - Hash authority tests
- `internal/factory/digest/gate_summary_integration_test.go` - Integration tests
- `internal/factory/digest/gate_summary_parity_test.go` - Mode parity tests
- `internal/factory/digest/gate_summary_copy_sort_v2_test.go` - Direct V2 comparator tests
- `internal/factory/digest/gate_summary_render_test.go` - Render tests
- `internal/factory/digest/gate_summary_sanitize_test.go` - Sanitize tests
- `internal/factory/digest/gate_summary_source_test.go` - Source file tests

### Golden Files
- `internal/factory/digest/testdata/v1-full.golden.txt`
- `internal/factory/digest/testdata/v1-minimal.golden.txt`
- `internal/factory/digest/testdata/v2-clinemm-microc3.golden.txt`
- `internal/factory/digest/testdata/v2-full.golden.txt`
- `internal/factory/digest/testdata/v2-leamas-self-hosted.golden.txt`
- `internal/factory/digest/testdata/v2-minimal.golden.txt`

## Behavior Changed

1. **Shared adapter**: One `Decode → Normalize → Render` adapter for both v1 and v2
2. **Preserved v1 behavior**: Evidence fallback and zero-duration handling maintained
3. **Fixed v2 row geometry**: Added scope, lifecycle, and totals columns
4. **UTF-8 safety**: Bounded text rendering with sanitization
5. **Canonical ordering**: Full arbitrary-precision integer comparison for sorting
6. **Exact hashing**: gate_summary_sha256 bound to rendered section

## Exact Commands Run

### Tests
```bash
# count=1
go test -count=1 ./internal/factory/digest/... ./internal/gatesummary/...
# Result: PASS (4.317s digest, 0.375s gatesummary)

# count=20
go test -count=20 ./internal/factory/digest/... ./internal/gatesummary/...
# Result: PASS (80.819s digest, 6.898s gatesummary)

# race tests
go test -race -count=5 ./internal/factory/digest/... ./internal/gatesummary/...
# Result: PASS (23.877s digest, 9.605s gatesummary)

# go vet
go vet ./internal/factory/digest/... ./internal/gatesummary/... ./cmd/leamas/...
# Result: PASS
```

### Build
```bash
CGO_ENABLED=0 go build -buildvcs=true -trimpath -o /tmp/leamas-gate-summary-v2-digest01 ./cmd/leamas
```

## Honest Results

- All tests pass including count=20 and race detection
- go vet clean
- Proof binary built with VCS stamps
- LLM-friendly: New files PASS; pre-existing unrelated failure in `docs/acts/ACT-LEAMAS-FACTORY-DUPCODE-V4-CALL-SITE-OPTIMIZATION01.md`

## LLM-Friendly Non-Ownership Evidence

**Non-ownership proof**: The failing file `docs/acts/ACT-LEAMAS-FACTORY-DUPCODE-V4-CALL-SITE-OPTIMIZATION01.md` was last modified in commit `0c11336` (not part of this ACT's changes).

**Targeted verification**: All new `gate_summary*.go` files pass LLM-friendly verification.

**Baseline equivalence**: NOT PROVEN - the detached-worktree comparison was not run. The claim is downgraded to non-ownership evidence only.

## Tested Commit

- **Commit**: `2f55deb117697c1fd3bdda20b53713cce54b692e`
- **Tree**: `829797954561095011c091b909f3739698652e4f`
- **Proof binary**: `/tmp/leamas-gate-summary-v2-digest01`
- **Binary SHA256**: `3f53834fd8e3ac6d88f28bf79f023107104073c8da27d995bc846be37b9bda01`
- **VCS modified**: `false`
- **VCS revision**: `2f55deb117697c1fd3bdda20b53713cce54b692e`

## Direct Comparator Proofs

- `TestCopyAndSortChecksV2Duration` - Equal earlier keys (name, scope, status, evidence), duration=100 vs 200
- `TestCopyAndSortChecksV2ExitCode` - Equal earlier keys plus duration, exit_code=1 vs 2
- Both use `Decode → Normalize` to get valid Integer values, mutate test-owned copies, call `copyAndSortChecksV2` directly

## Verification Status

| Check | Status | Notes |
|-------|--------|-------|
| go test -count=1 | PASS | 4.317s digest, 0.375s gatesummary |
| go test -count=20 | PASS | 80.819s digest, 6.898s gatesummary |
| go test -race -count=5 | PASS | 23.877s digest, 9.605s gatesummary |
| go vet | PASS | All packages clean |
| make factorize | FAIL | 372.49s - unrelated llm-friendly issue |
| make gate-fast | FAIL | Unrelated llm-friendly issue (pre-existing) |
| make gate | NOT RUN | Refused in editor context per AGENTS.md |

## Factorize and Gate-Fast Results

**make factorize**: FAILED (372.49s)
- Cause: `llm-friendly FAILED` on `docs/acts/ACT-LEAMAS-FACTORY-DUPCODE-V4-CALL-SITE-OPTIMIZATION01.md`
- All other stages: PASS

**make gate-fast**: FAILED
- Cause: `llm-friendly FAILED` on `docs/acts/ACT-LEAMAS-FACTORY-DUPCODE-V4-CALL-SITE-OPTIMIZATION01.md`
- All other stages: PASS (go mod tidy, gofmt, go vet, go test -short, static build)

**Non-ownership**: The failing file was last modified in commit `0c11336` (not part of this ACT's changes).
This ACT did not introduce the failure.

## Follow-up ACTs

- ACT-LEAMAS-GATE-SUMMARY-V2-DOGFOOD01 - BLOCKED until this ACT closure
