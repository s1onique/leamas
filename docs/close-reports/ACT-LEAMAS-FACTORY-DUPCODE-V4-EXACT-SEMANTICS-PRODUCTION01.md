# ACT-LEAMAS-FACTORY-DUPCODE-V4-EXACT-SEMANTICS-PRODUCTION01

## Status: OPEN

## Parent ACT
- ACT-LEAMAS-FACTORY-DUPLICATE-CODE-GATE01

## Summary

Production correction ACT for the V4 duplicate-code detector to implement exact
semantic contracts. The exact-semantics test suite
(`ACT-LEAMAS-FACTORY-DUPCODE-V4-EXACT-SEMANTICS-TESTS01`) and the exact-geometry
test suite (`ACT-LEAMAS-FACTORY-DUPCODE-V4-EXACT-GEOMETRY-TESTS01`) expose the
following production defects that must be corrected.

## Production Defects to Fix

### 1. Excessive Finding Cardinality
**Current behavior**: V4 returns ~334 findings for simple 2-file clones
**Required behavior**: Exactly 1 maximal finding

### 2. Coalesced Occurrence Multiplicity
**Current behavior**: Multiple occurrences within a file are coalesced into one
**Required behavior**: Each distinct occurrence must be preserved and counted
**Example**: `repeat_a.go × 2, repeat_b.go × 1` (3 total occurrences)

### 3. Shadow Sub-findings
**Current behavior**: Threshold-sized prefixes/suffixes/interior windows emitted alongside maximal
**Required behavior**: Only the maximal clone finding, no sub-findings

### 4. N-way Clone Materialization
**Current behavior**: N-way clones produce pairwise residue alongside coalesced result
**Required behavior**: Exactly 1 N-way finding with all N occurrences

### 5. Independent Body Separation
**Current behavior**: 15 findings for 2 independent clone bodies
**Required behavior**: Exactly 2 findings, one per maximal body

## Dependencies

- **ACT-LEAMAS-FACTORY-DUPCODE-V4-EXACT-SEMANTICS-TESTS01** (cardinality/validity tests exist)
- **ACT-LEAMAS-FACTORY-DUPCODE-V4-EXACT-GEOMETRY-TESTS01** (exact geometry tests exist)

Both test contracts must specify expected behavior before production is corrected.

## Downstream Follow-up

- ACT-LEAMAS-FACTORY-DUPCODE-V4-ALL-PAIRS-MATERIALIZATION-PERFORMANCE01 (performance sibling)

## Required Outcomes

When this ACT is closed:
- `ACT-LEAMAS-FACTORY-DUPCODE-V4-EXACT-SEMANTICS-TESTS01` is already COMPLETE (cardinality/multiplicity covered)
- `ACT-LEAMAS-FACTORY-DUPCODE-V4-EXACT-GEOMETRY-TESTS01` is already COMPLETE as a red specification
- All installed geometry assertions pass against corrected production
- All 7 semantic tests pass
- Full `go test ./...` passes
- Full `make gate` passes

## Scope

- Production V4 algorithm in `internal/factory/dupcode/`

## Created At

2026-07-16T07:36:00+03:00
