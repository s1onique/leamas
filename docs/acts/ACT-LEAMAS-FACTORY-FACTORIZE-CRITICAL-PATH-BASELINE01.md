# ACT: ACT-LEAMAS-FACTORY-FACTORIZE-CRITICAL-PATH-BASELINE01

**Status**: CLOSED — Baseline established, successor ACT nominated

**Parent Epic**: EPIC-LEAMAS-FACTORY-FACTORIZE-LATENCY01

**Date**: 2026-07-20

## Objective

Produce a trustworthy, reproducible and machine-readable critical-path baseline for the canonical `make factorize` execution.

The ACT must:
1. Inventory every canonical verifier
2. Measure controlled cold-cache and warm-cache runs
3. Record whole-run and per-verifier latency
4. Distinguish Go build caching, Go test-result caching, and any existing Leamas-owned caching
5. Profile the dominant Go verifier or verifiers
6. Identify duplicate work and parallelizable work
7. Estimate realistic speedup opportunities
8. Define proposed resource classes and exact-input boundaries
9. Select exactly one next executable optimization ACT
10. Preserve all current gate semantics

## Baseline Results

Initial observation (2026-07-20):
- Total runtime: 462.14s
- dupcode: 229.89s (49.7% of total)
- dupcode-baseline: 230.50s (49.9% of total)
- Together these two verifiers account for ~460s of the total

## Successor ACT

To be determined after measurement analysis.
