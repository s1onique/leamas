# ACT-LEAMAS-FACTORY-FACTORIZE-CRITICAL-PATH-BASELINE01-CORRECTION01

## Status

**OPEN — BASELINE EVIDENCE AND INSTRUMENTATION CORRECTION REQUIRED**

## Parent ACT

`ACT-LEAMAS-FACTORY-FACTORIZE-CRITICAL-PATH-BASELINE01` (PARTIAL — CORRECTION01 OPEN)

## Epic

`EPIC-LEAMAS-FACTORY-FACTORIZE-LATENCY01`

## Starting Commit

`faa117a`

## Purpose

Correct the baseline ACT so that:

1. metrics represent what their fields claim to represent;
2. every evidence artifact binds to an exact repository subject;
3. the complete controlled benchmark matrix is executed;
4. required statistical and profiling analyses are produced;
5. canonical closure verification is fresh and passing;
6. the original ACT can be honestly classified as complete.

## Corrections Required

### P0 Blockers

| # | Finding |
|---|---------|
| P0-1 | Only 1 measurement sample collected (no cold/warm series) |
| P0-2 | CPU/memory metrics incorrectly use cumulative RUSAGE_SELF without deltas |
| P0-3 | Metrics-write failures do not fail the command |
| P0-4 | Subject identity fields (head_oid, tree_oid, subject_input_digest) are empty |
| P0-5 | scenario/sequence are hard-coded to "controlled-warm", 1 |
| P0-6 | Command fingerprint only hashes name + repo root |
| P0-7 | Gate summary evidence is stale and failing |

### P1 Blockers

| # | Finding |
|---|---------|
| P1-1 | Required metrics tests are absent |
| P1-2 | Unreachable cache-classification branch |
| P1-3 | Atomic writing uses predictable temp filenames |

## Implementation Order

```
1. tests: install the corrected metrics/evidence contract
2. fix: subject identity, scenario and sequence
3. fix: truthful resource scopes and command fingerprints
4. fix: fail-closed atomic artifact writing
5. measure: cold/warm/native matrix and profiling
6. docs: reconcile statistics, inventories and successor
7. close: run fresh canonical gates and regenerate evidence
```

## Semantic Scope

Default `make factorize` behavior must remain unchanged:
- Same verifier set
- Same canonical sorted output order
- Same verifier inputs
- Same thresholds
- Same sequential execution
- Same failure findings
- Same pass/fail classification
- No result cache
- No incremental indexing
- No worker pool

Metrics remain explicitly opt-in.

## Non-Goals

Do not implement:
- Verifier parallelism
- Worker pools
- Result caching
- Incremental duplicate-code indexes
- Duplicate-code semantic changes
- Threshold changes
- Baseline changes
- CI performance budgets

## Successor

To be determined after corrected measurements establish the actual critical path.

## References

- [Go Packages: Cmd.ProcessState](https://pkg.go.dev/os#Cmd.ProcessState)
- [Go Packages: CreateTemp](https://pkg.go.dev/os#CreateTemp)
- [man7: getrusage(2)](https://man7.org/linux/man-pages/man2/getrusage.2.html)
