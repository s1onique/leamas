# ACT-LEAMAS-FACTORY-DUPCODE-V4-ALL-PAIRS-MATERIALIZATION-PERFORMANCE01

## Status: PLANNED

## Parent ACT
- ACT-LEAMAS-FACTORY-DUPLICATE-CODE-GATE01

## Summary

Ownership established for the performance ACT concerning the V4 duplicate-code
detector's materialization strategy. This sibling ACT to
`ACT-LEAMAS-FACTORY-DUPCODE-V4-EXACT-SEMANTICS-TESTS01` must measure and reduce
or eliminate unnecessary all-pairs materialization while preserving the newly
specified N-way output semantics.

## Scope (when implemented)

1. **Measure cost of all-pairs materialization**: Benchmark V4 with varying file counts and clone body sizes
2. **Identify and quantify unnecessary all-pairs allocation**: Determine which pairwise outcomes are eventually coalesced or shadowed
3. **Reduce or eliminate unnecessary all-pairs materialization**: Preserve the N-way merge semantics but avoid O(N²) candidate storage where feasible
4. **Verify semantic equivalence**: All 7 exact semantic tests continue to pass after changes

## Dependencies

- ACT-LEAMAS-FACTORY-DUPCODE-V4-EXACT-SEMANTICS-PRODUCTION01 - Must be COMPLETE first
- ACT-LEAMAS-FACTORY-DUPCODE-V4-EXACT-GEOMETRY-TESTS01 - Must be COMPLETE first

## Mission Statement

This ACT's mission is not "test all-pairs materialization works" but instead
to MEASURE and REDUCE the cost of all-pairs materialization while preserving
the exact N-way output semantics.

## Created At

2026-07-16T07:18:30+03:00
