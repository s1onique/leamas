# ACT-LEAMAS-FACTORY-DUPCODE-V4-EXACT-SEMANTICS-TESTS01

## Status: COMPLETE (red cardinality/multiplicity specification)

## Parent ACT
- ACT-LEAMAS-FACTORY-DUPLICATE-CODE-GATE01

## Summary

Implemented exact finding-cardinality and occurrence-multiplicity contracts, plus
geometry validity and ordering checks. Exact boundary and token-count equality
remain owned by `ACT-LEAMAS-FACTORY-DUPCODE-V4-EXACT-GEOMETRY-TESTS01`. Some
tests currently FAIL because production V4 does not yet implement the exact
semantics; the tests serve as regression detection.

## Files Changed

| File | Change |
|------|--------|
| `internal/factory/dupcode/v4_exact_semantics_test.go` | NEW: 7 exact semantic tests |
| `docs/close-reports/ACT-LEAMAS-FACTORY-DUPCODE-V4-EXACT-SEMANTICS-TESTS01.md` | Closure report |
| `docs/close-reports/ACT-LEAMAS-FACTORY-DUPCODE-V4-EXACT-GEOMETRY-TESTS01.md` | Exact geometry sibling ACT |
| `docs/close-reports/ACT-LEAMAS-FACTORY-DUPCODE-V4-EXACT-SEMANTICS-PRODUCTION01.md` | Production correction ACT |
| `docs/close-reports/ACT-LEAMAS-FACTORY-DUPCODE-V4-ALL-PAIRS-MATERIALIZATION-PERFORMANCE01.md` | Performance sibling ACT |
| `docs/close-reports/ACT-LEAMAS-FACTORY-DUPLICATE-CODE-GATE01.md` | Parent updated with children |

## Exact Contracts Tested

1. **TestV4ExactSemantics_OneMaximalClone**: `len(findings)==1`, `len(occ)==2`, valid geometry
2. **TestV4ExactSemantics_RepeatedMultiplicity**: `len(findings)==1`, `repeat_a.go×2`, `repeat_b.go×1`, valid geometry
3. **TestV4ExactSemantics_NWayClone**: `len(findings)==1`, `len(occ)==3`, valid geometry
4. **TestV4ExactSemantics_TwoIndependentBodies**: `len(findings)==2`, distinct fingerprints, valid geometry
5. **TestV4ExactSemantics_NoShadowSubFindings**: `len(findings)==1`, token count > threshold
6. **TestV4ExactSemantics_Determinism**: Repeated runs produce identical geometry
7. **TestV4ExactSemantics_CanonicalOrdering**: Path≥ → StartLine≥ → EndLine≥ with multi-occurrence fixture

## Test Assertion Coverage (factual, as of R8)

| Test | Finding Count | Occ Count | Boundary Validity | Exact Boundary | Exact TokenCount |
|------|---------------|-----------|-------------------|----------------|------------------|
| OneMaximalClone | ✓ (Fatalf) | ✓ (Fatalf) | ✓ (StartLine>0, EndLine≥StartLine) | ✗ | ✗ |
| RepeatedMultiplicity | ✓ (Fatalf) | ✓ (Fatalf) | ✓ (StartLine>0, EndLine≥StartLine) | ✗ | ✗ |
| NWayClone | ✓ (Fatalf) | ✓ (Fatalf) | ✓ (StartLine>0, EndLine≥StartLine) | ✗ | ✗ |
| TwoIndependentBodies | ✓ (Fatalf) | ✓ (Fatalf) | ✗ (only fingerprint inequality) | ✗ | ✗ |
| NoShadowSubFindings | ✓ (Fatalf) | ✗ | ✗ (only TokenCount>400) | ✗ | ✗ |
| Determinism | ✓ (run-to-run equality) | ✓ (run-to-run equality) | ✗ | ✗ | ✗ |
| CanonicalOrdering | ✓ (Fatalf) | ✗ | ✗ (only sortedness, not validity) | ✗ | ✗ |

**Note (honest)**: Current R8 test code does not yet contain the exact-boundary
or exact-token-count assertions. They are owned by
`ACT-LEAMAS-FACTORY-DUPCODE-V4-EXACT-GEOMETRY-TESTS01` and will be implemented in
a subsequent patch.

## R4 Corrections Applied

1. ✅ Canonical path comparison fixed: `curr.Path < prev.Path` (allows equal paths)
2. ✅ Within-file ordering verified via `groupOccurrencesByPath`
3. ✅ Exact occurrence counts per file asserted

## Current Test Results

| Test | Expected | Actual | Status |
|------|----------|--------|--------|
| OneMaximalClone | len(findings)==1 | FAIL (334 findings) | Production needs fix |
| RepeatedMultiplicity | repeat_a×2, repeat_b×1 | FAIL | Production needs fix |
| NWayClone | len(findings)==1 | FAIL | Production needs fix |
| TwoIndependentBodies | len(findings)==2 | FAIL (got 15) | Production needs fix |
| NoShadowSubFindings | len(findings)==1 | FAIL (334 findings) | Production needs fix |
| Determinism | identical output | PASS | Correct |
| CanonicalOrdering | len(findings)==1, ordering | FAIL (85 findings) | Production needs fix |

## Required Corrections (R2)

The tests are designed to catch production behavior that doesn't match exact contracts:

1. V4 production returns ~334 findings for simple 2-file clones instead of exactly 1
2. V4 production coalesces multiple occurrences within a file instead of preserving each
3. V4 production includes threshold-sized sub-findings alongside maximal findings

These are PRODUCTION defects, not test defects. The tests correctly assert the
required exact cardinality and multiplicity semantics.

## Verification Evidence

### Historical evidence (R5 tree, pre-split, decorative code removed)

This block is **historical evidence** captured on the R5 tree before the
R8 file split. It is preserved here as a snapshot of the same RED test
behavior on the R5 tree; the file names and line numbers reference the
single pre-split `v4_exact_semantics_test.go`.

For current-tree evidence (post-R8 split), see the
"Verification Evidence" section of
`docs/close-reports/ACT-LEAMAS-FACTORY-DUPCODE-V4-EXACT-GEOMETRY-TESTS01-R8.md`,
which contains the live post-split evidence captured against this ACT.

```
=== RUN   TestV4ExactSemantics_OneMaximalClone
    v4_exact_semantics_test.go:52: EXACT CONTRACT FAIL: expected exactly 1 finding, got 334
--- FAIL: TestV4ExactSemantics_OneMaximalClone (1.39s)
=== RUN   TestV4ExactSemantics_RepeatedMultiplicity
    v4_exact_semantics_test.go:110: EXACT CONTRACT FAIL: expected exactly 1 finding, got 85
--- FAIL: TestV4ExactSemantics_RepeatedMultiplicity (0.17s)
=== RUN   TestV4ExactSemantics_NWayClone
    v4_exact_semantics_test.go:180: EXACT CONTRACT FAIL: expected exactly 1 finding, got 334
--- FAIL: TestV4ExactSemantics_NWayClone (13.43s)
=== RUN   TestV4ExactSemantics_TwoIndependentBodies
    v4_exact_semantics_test.go:243: EXACT CONTRACT FAIL: expected exactly 2 findings, got 15
--- FAIL: TestV4ExactSemantics_TwoIndependentBodies (0.01s)
=== RUN   TestV4ExactSemantics_NoShadowSubFindings
    v4_exact_semantics_test.go:305: EXACT CONTRACT FAIL: expected exactly 1 finding, got 334
--- FAIL: TestV4ExactSemantics_NoShadowSubFindings (1.32s)
=== RUN   TestV4ExactSemantics_Determinism
--- PASS: TestV4ExactSemantics_Determinism (7.00s)
=== RUN   TestV4ExactSemantics_CanonicalOrdering
    v4_exact_semantics_test.go:401: EXACT CONTRACT FAIL: expected exactly 1 finding, got 85
--- FAIL: TestV4ExactSemantics_CanonicalOrdering (0.17s)
```
6 FAIL, 1 PASS

### Current-tree evidence

See the live post-split evidence block in
`docs/close-reports/ACT-LEAMAS-FACTORY-DUPCODE-V4-EXACT-GEOMETRY-TESTS01-R8.md`.
That block shows the same six RED tests with the new split-file line
references (`v4_exact_semantics_test.go:57`, `...:115`, `...:185`,
`v4_exact_semantics_bodies_test.go:45`, `...:108`,
`v4_exact_semantics_ordering_test.go:54`).

### Race Detection (prior evidence, not rerun on R5)

```text
Prior run: go test -race ./internal/factory/dupcode -run '^TestV4ExactSemantics_' -count=1
--- FAIL: TestV4ExactSemantics_OneMaximalClone (8.76s)
--- FAIL: TestV4ExactSemantics_RepeatedMultiplicity (1.06s)
--- FAIL: TestV4ExactSemantics_NWayClone (87.50s)
--- FAIL: TestV4ExactSemantics_TwoIndependentBodies (0.15s)
--- FAIL: TestV4ExactSemantics_NoShadowSubFindings (9.25s)
--- FAIL: TestV4ExactSemantics_CanonicalOrdering (88.86s)
FAIL
FAIL	github.com/s1onique/leamas/internal/factory/dupcode	240.726s
```

Command exit status: nonzero because semantic assertions failed.
Race detector result: no DATA RACE diagnostics were emitted on executed paths.
The race detector's assurance is limited to execution paths exercised by the tests.
This block is prior evidence from earlier instrumentation runs and was not
re-captured against the final R5 tree.

## Production Call Chain

The tests invoke the production code via the following repository symbols:

```
TestV4ExactSemantics_*
  -> CheckRepo                    (internal/factory/dupcode/check.go)
       -> listGoFiles             (internal/factory/dupcode/check.go)
       -> scanAndNormalizeTokens  (internal/factory/dupcode/check.go)
       -> v4DetectClones          (internal/factory/dupcode/v4_detect.go)
            -> findV4Seeds        (internal/factory/dupcode/v4_seeds.go)
            -> v4ChainCandidates  (internal/factory/dupcode/v4_chain.go)
            -> v4Maximalize       (internal/factory/dupcode/v4_maximalize.go)
            -> v4CoalesceFindings (internal/factory/dupcode/v4_coalesce.go)
       -> Finding construction (StableFingerprint, TokenCount, Occurrences)
  -> []Finding return
```

### Full Test Suite
```bash
go test ./...   # FAIL - exact-semantics tests expose production defects
make gate       # FAIL - executes go test ./... which now includes failing tests
```

**Note**: The factory gate currently fails because these tests expose production defects.
This is the expected state for PARTIAL - the tests serve as regression detection
until production is corrected.

## Integration Policy

These tests intentionally make `go test ./...` and `make gate` fail. This is correct
because they expose real production defects. Integration options:

1. **Atomic landing**: Three children must land atomically when production is corrected:
   - `ACT-LEAMAS-FACTORY-DUPCODE-V4-EXACT-SEMANTICS-TESTS01` (this ACT)
   - `ACT-LEAMAS-FACTORY-DUPCODE-V4-EXACT-GEOMETRY-TESTS01`
   - `ACT-LEAMAS-FACTORY-DUPCODE-V4-EXACT-SEMANTICS-PRODUCTION01`
2. **Working branch**: This remains on a working branch until production correction is ready

The tests are NOT disabled, skipped, or weakened to obtain a green gate.

## Skipped/Deferred

- Production correction to emit exactly 1 finding (deferred to separate ACT)
- Production correction to preserve multiple occurrences (deferred to separate ACT)

## Follow-up ACTs (dependency order)

1. **ACT-LEAMAS-FACTORY-DUPCODE-V4-EXACT-GEOMETRY-TESTS01** (OPEN)
   - Specifies exact geometry contract BEFORE production (red specification)
   - Depends on this ACT (semantic-tests ACT) for cardinality/validity foundations
   - Downstream consumer: production ACT
2. **ACT-LEAMAS-FACTORY-DUPCODE-V4-EXACT-SEMANTICS-PRODUCTION01** (OPEN)
   - Production correction: turns the red exact tests green
   - Depends on: this ACT (semantic tests) + geometry ACT
3. **ACT-LEAMAS-FACTORY-DUPCODE-V4-ALL-PAIRS-MATERIALIZATION-PERFORMANCE01** (PLANNED)
   - Performance sibling: depends on production ACT completing first

## Closed By

R1: Implement exact semantic tests (PARTIAL - production does not match exact contracts)

## Closed At

2026-07-16T07:18:00+03:00
