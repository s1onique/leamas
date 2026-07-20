# Factorize Critical Path Baseline

## Summary

This document records the critical-path baseline for `make factorize` execution.

## Environment

| Property | Value |
|----------|-------|
| Go Version | go1.25.12 linux/amd64 |
| GOMAXPROCS | (default) |
| Logical CPUs | 24 |
| OS | Linux Mint 22.3 (Zena) |
| Host Class | development-workstation |

## Baseline Measurements

### Controlled-Warm Observations (2026-07-20)

| Metric | Value |
|--------|-------|
| Total Runtime | 462.14s |
| dupcode | 229.89s |
| dupcode-baseline | 230.50s |
| static-binary | 1.30s |
| exec-gate | 0.30s |
| llm-friendly | 0.11s |
| All others | <0.05s |

## Critical Path Analysis

### Primary Bottleneck

**dupcode + dupcode-baseline** account for **99.6%** of total runtime.

These two verifiers:
1. Run sequentially (can run in parallel)
2. Both use `go test -count=1` (no test-result caching)
3. Process the same repository files independently

### Theoretical Speedup Opportunities

| Optimization | Estimated Speedup | Notes |
|-------------|------------------|-------|
| Parallel dupcode + dupcode-baseline | ~2x | Independent verifiers |
| Exact-input result caching | High | Unchanged-tree benefit |
| Incremental dupcode indexing | Medium | Changed-tree benefit |

### Concurrency Estimates

| Workers | Lower Bound | Upper Bound | Notes |
|---------|-------------|-------------|-------|
| 1 (current) | 462s | 462s | Sequential baseline |
| 2 | 231s | 231s | dupcode + dupcode-baseline parallel |
| 4+ | 231s | 231s | Limited by serial verifiers (~2s) |

**Realistic bound: 231s with 2 workers (50% reduction)**

## Recommendations

### Immediate Next ACT

**ACT-LEAMAS-FACTORY-FACTORIZE-BOUNDED-PARALLELISM01**

Rationale:
1. Evidence shows ~50% of runtime in two independent verifiers
2. Implementation is straightforward (bounded concurrency)
3. Low risk (verifiers are independent)
4. Provides immediate ~2x speedup

### Non-Goals for Next ACT

- Do NOT implement result caching yet (requires boundary analysis)
- Do NOT modify dupcode algorithm (requires separate ACT)
- Do NOT add CI latency enforcement

## Measurement Uncertainty

| Source | Impact | Mitigation |
|--------|--------|------------|
| Single measurement | High | Need controlled-cold/warm series |
| Host load variation | Medium | Use coefficient of variation |
| Go build cache state | High | Use isolated GOCACHE for benchmarks |
