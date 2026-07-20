# Factorize Verifier Inventory

## Canonical Verifiers (15 total, sorted by name)

| Ordinal | ID | Implementation | Execution Form | Go Test Caching | -count=1 |
|---------|-----|----------------|----------------|------------------|----------|
| 1 | agent-context | internal/factory/agentcontext | In-process | N/A | N/A |
| 2 | docs | internal/factory/docs | In-process | N/A | N/A |
| 3 | doctrine | internal/factory/doctrine | In-process | N/A | N/A |
| 4 | doctrine-agent-contracts | internal/factory/doctrine | In-process | N/A | N/A |
| 5 | domain-boundaries | internal/factory/boundary | In-process | N/A | N/A |
| 6 | dupcode | internal/factory/dupcode | go test | Disabled | Yes |
| 7 | dupcode-baseline | internal/factory/dupcode | go test | Disabled | Yes |
| 8 | exec-gate | internal/factory/execgate | In-process | N/A | N/A |
| 9 | executable-contract-first | internal/factory/doctrine | In-process | N/A | N/A |
| 10 | forbidden-patterns | internal/factory/forbidden | In-process | N/A | N/A |
| 11 | git-hooks | internal/factory/githooks | In-process | N/A | N/A |
| 12 | language | internal/factory/language | In-process | N/A | N/A |
| 13 | llm-friendly | internal/factory/llmfriendly | In-process | N/A | N/A |
| 14 | static-binary | internal/factory/staticbinary | In-process | N/A | N/A |
| 15 | tooling-boundaries | internal/factory/tooling | In-process | N/A | N/A |

## Baseline Timing Observations (2026-07-20)

| Verifier | Duration | % of Total |
|----------|----------|------------|
| dupcode | 229.89s | 49.7% |
| dupcode-baseline | 230.50s | 49.9% |
| static-binary | 1.30s | 0.3% |
| exec-gate | 0.30s | 0.1% |
| llm-friendly | 0.11s | 0.02% |
| forbidden-patterns | 0.02s | 0.004% |
| language | 0.01s | 0.002% |
| Others | ~0.00s | ~0% |
| **TOTAL** | **462.14s** | **100%** |

## Key Findings

1. **dupcode and dupcode-baseline together account for ~99.6% of total runtime**
2. Both use `go test -count=1`, disabling test-result caching
3. All other verifiers complete in <2s total
4. No parallelism is currently implemented

## Resource Classification

| Class | Verifiers |
|-------|-----------|
| cpu-heavy | dupcode, dupcode-baseline |
| normal | static-binary, exec-gate, llm-friendly |
| tiny-io | All others |

## Cache Boundaries

- **dupcode**: Unknown - requires input boundary analysis
- **dupcode-baseline**: Unknown - requires input boundary analysis
- **All others**: Fast enough that caching is unnecessary

## Concurrency Safety

- All verifiers are currently run sequentially
- dupcode and dupcode-baseline are independent and could run in parallel
- No shared mutable state detected between verifiers
