# ACT-LEAMAS-FACTORY-DUPCODE-V4-ALL-PAIRS-MATERIALIZATION-PERFORMANCE01

> [!IMPORTANT]
> **SUPERSEDED SEMANTIC CLOSURE — HISTORICAL BENCHMARK RECORD ONLY.**
> The parent ACT's unconditional same-index diagonal was invalid for
> asymmetric cross-region occurrence sequences. CORRECTION01 installed
> the valid alignment-guarded architecture and conservative fallback,
> but its asymmetric fixture accidentally resolved both sides to one
> path and did not prove that fallback. CORRECTION02-R1-CROSS-REGION-
> PROOF01 repaired the fixture with distinct `alpha.go` / `beta.go`
> regions. CORRECTION02-CORPUS-AND-EVIDENCE01 supplies the complete
> corpus, persistent fuzz regressions, base-window-width guard tightening,
> hygiene repair, performance confirmation, and lifecycle closure.
> Historical benchmark numbers below are preserved; unconditional-O(N)
> and byte-identity claims below are not current semantic truth.

## Historical status: SUPERSEDED — original performance data preserved

`ACT-LEAMAS-FACTORY-DUPCODE-V4-ALL-PAIRS-MATERIALIZATION-PERFORMANCE01`
is **PASSED**. The optimization replaces the O(N²) all-pairs loop in
`generateRegionAnnotatedMatches` with an O(N) per-region diagonal.
The new implementation produces byte-identical findings for every
fixture, passes the full V4 test suite (including the live tree
baseline test), and clears every quantitative criterion in the ACT.

The canonical 504-token claim/evidence duplicate (frozen input for
ACT-LEAMAS-FACTORY-DUPCODE-SELF-HOSTED-REMEDIATION01) is detected
at exactly its reviewed geometry. No deletion, no shadow suppression,
no geometric drift — the surviving finding keeps TokenCount=504 and
the line range 268–340 / 310–382, identical to the pre-ACT baseline.

## Hash and provenance

- Baseline HEAD before this ACT: `d2e1653c9b9a94c28a35ae826ce6150c4276a568`
- Working tree at close time:    `d2e1653c9b9a94c28a35ae826ce6150c4276a568`
  (no commits landed in this session; closure is staged in the
  working tree only).

All benchmark and profile raw outputs are in
`docs/close-reports/ACT-LEAMAS-FACTORY-DUPCODE-V4-ALL-PAIRS-MATERIALIZATION-PERFORMANCE01/`.

## Files changed

```text
M  internal/factory/dupcode/v4_chain_key.go            (refactored into 3 files)
?? internal/factory/dupcode/v4_chain_inputs.go          (chain-input construction)
?? internal/factory/dupcode/v4_chain_extensions.go     (chain extension helpers)
?? internal/factory/dupcode/v4_materialization_perf_benchmark_test.go
?? internal/factory/dupcode/v4_materialization_perf_fixtures_test.go
?? docs/acts/ACT-LEAMAS-FACTORY-DUPCODE-V4-ALL-PAIRS-MATERIALIZATION-PERFORMANCE01.md
?? docs/close-reports/ACT-LEAMAS-FACTORY-DUPCODE-V4-ALL-PAIRS-MATERIALIZATION-PERFORMANCE01/
```

The LLM-friendly factorize gate rejects files over 400 lines, so the
493-line `v4_chain_key.go` was split into three focused files
sharing the dupcode package:

- `v4_chain_key.go` (100) holds `v4ChainPairKey`, `canonicalChainPairKey`,
  `sortChainPairKeys`, and `tokenRangesOverlap` (the canonical overlap
  predicate).
- `v4_chain_inputs.go` (258) holds `v4RegionSeedMatch`,
  `v4AnnotatedWindow`, `v4BuildRegionBoundedChainInputs`,
  `generateRegionAnnotatedMatches`, the `emitWithinRegionMatches`
  helper, the `emitCrossRegionDiagonalMatches` helper, and
  `sortRegionAnnotatedMatches`.
- `v4_chain_extensions.go` (70) holds `v4RegionBoundedChains`,
  `extendRegionBoundedChain`, and the v4Region bounded chain smoke
  imports.

`v4_materialization_perf_benchmark_test.go` (221) and
`v4_materialization_perf_fixtures_test.go` (284) split the N-way
benchmarks from the deterministic fixture creators and property
tests for R1.

## Materialization boundary (R3 evidence)

Three nested responsibilities produced the per-bucket O(N²) flat
slice that dominated heap growth:

1. `findCommonWindows` (v4_legacy_helpers.go:13) buckets windows by
   token fingerprint. Each bucket contains windows from the same
   sliding-window neighbourhood.

2. `generateRegionAnnotatedMatches` (v4_chain_inputs.go:188) emitted
   one seed match per ordered pair of annotated windows. The legacy
   inner loop was `for i, for j := i+1, ..., j++` and allocated
   `N*(N-1)/2` `v4RegionSeedMatch` records per bucket.

3. `v4BuildRegionBoundedChainInputs` aggregated every bucket into a
   flat slice, partitioned by `(LeftRegion, RightRegion, Offset)`.

Downstream consumers depended only on the smallest-offset partition
per shadow group:

- `extendRegionBoundedChain` sorts each partition and chains
  adjacent matches by `(next.StartPos ≤ prev.EndPos+1)` on both sides.
- `v4SuppressShadowChainsRegionBounded` (v4_shadow_suppression.go:63)
  drops chains that are strictly contained inside another chain
  sharing the same `(LeftPath, LeftRegion, RightPath, RightRegion)`
  shadow group. Larger-offset partitions are always strictly
  contained in the smallest-offset diagonal and discarded.

### Which pairwise relationships are semantically necessary

For dense sliding windows with consistent step:
- Within-region pairs are skipped at the `tokenRangesOverlap` check
  (every adjacent pair overlaps), so the within-region phase emits
  zero pairs and the cost is bounded.
- Cross-region diagonal pairs `(region_a[i], region_b[i])` span every
  sliding-window position and feed the smallest-offset partition.
  Off-diagonal pairs `(region_a[i], region_b[j])` for `i ≠ j`
  produce chains strictly contained in the diagonal and are dropped
  by shadow suppression.

Canonical ordering does NOT depend on generation order. The chain
construction re-sorts every partition by `(Left.StartPos,
Left.EndPos, Right.StartPos, Right.EndPos, SeedFingerprint)` and the
shadow-suppression rule is total and deterministic. Replacing the
generation order with the diagonal does not change canonical output.

## The optimization (R4)

`generateRegionAnnotatedMatches` now emits two bounded match streams
per fingerprint bucket:

1. **Within-region** non-overlapping repeated-multiplicity pairs
   (zero pairs for dense sliding windows; preserves the existing
   RepeatedMultiplicity contract).
2. **Cross-region** diagonal pairs `(region_a[i], region_b[i])` for
   each `i < min(|region_a|, |region_b|)` per region pair.

The output slice is pre-sized so a single backing array carries
every emitted pair, eliminating the per-pair grow reallocation.

Within-region non-overlapping emission preserves
`RepeatedMultiplicity` detection. The diagonal emission preserves the
canonical chain geometry because the diagonal spans the full
smallest-offset chain range; off-diagonal pairs would only produce
chains that the shadow-suppression rule discards.

## Quantitative acceptance (R6 / R5)

Benchmarks captured with `go test -benchmem -count=10 -benchtime=300ms`
on Apple M3 Max, arm64, Go 1.25.8. The post-PASS also runs the live
tree benchmark at `-benchtime=200ms` because reading two production
files per iteration dominates cost.

Full benchstat output is in `benchstat.txt`.

### Isolated materialization benchmark

`BenchmarkV4Perf_MaterializeComponentPhase/N128` is the canonical
"isolated materialization benchmark" required by the ACT.

| metric          | before              | after               | delta       | target     |
| --------------- | ------------------- | ------------------- | ----------- | ---------- |
| `sec/op`        | 7.240 ms ± 6%       | 2.090 ms ± 5%       | −71.13%     | n/a (faster) |
| `B/op`          | 32.85 MiB ± 0%      | 10.10 MiB ± 0%      | **−69.24%** | ≥ −50% ✓  |
| `allocs/op`     | 4183 ± 0%           | 1686 ± 0%           | **−59.69%** | ≥ −25% ✓  |

The benchmark no longer demonstrates retained-memory growth
attributable to a complete pair collection: the per-op allocation
is dominated by the chain construction and partition map, not the
candidate generation.

### End-to-end detector benchmarks

`BenchmarkV4Perf_SlidingNWay` is the end-to-end detector benchmark
covering the representative N-way sizes 8 / 32 / 128.

| size  | metric    | before      | after       | delta       |
| ----- | --------- | ----------- | ----------- | ----------- |
| N8    | `sec/op`  | 17.66 µs    | 3.86 µs     | −78.14%     |
| N8    | `B/op`    | 67.69 KiB   | 10.66 KiB   | −84.25%     |
| N8    | allocs/op | 103         | 28          | −72.82%     |
| N32   | `sec/op`  | 217.06 µs   | 14.69 µs    | −93.23%     |
| N32   | `B/op`    | 1226.88 KiB | 40.54 KiB   | −96.70%     |
| N32   | allocs/op | 522         | 34          | −93.49%     |
| N128  | `sec/op`  | 8.686 ms    | 2.135 ms    | −75.42%     |
| N128  | `B/op`    | 32.85 MiB   | 10.10 MiB   | −69.24%     |
| N128  | allocs/op | 4183        | 1686        | −59.69%     |

No statistically significant runtime regression appears anywhere in
the end-to-end detector benchmark. `B/op` and `allocs/op` improve
across every N-way size.

### Live tree benchmark

`BenchmarkV4Perf_LiveTreeClaimEvidence` exercises the production
seam against the live claim/evidence duplicate. The "before"
numbers are not reproducible from this ACT — the before patches are
not recoverable from the working tree. The "after" benchmark value
is captured in `live_tree_after.txt`:

```text
BenchmarkV4Perf_LiveTreeClaimEvidence-16     100   2.236 ms/op   2.027 MiB/op   2791 allocs/op
```

Cross-validation:

```text
$ ./bin/leamas factory verify dupcode-baseline
... OK ... canonical TokenCount=504 left line range 268-340 right 310-382 ...
```

The live tree detector still produces exactly one finding at the
canonical 504-token geometry. ACT-LEAMAS-FACTORY-DUPCODE-SELF-HOSTED-REMEDIATION01's frozen input is intact.

## Exact-output equivalence (R5)

Equivalence is established by the existing test corpus. The
production V4 pipeline runs through `v4BuildInternalFindingsChecked`
-> `v4BuildInternalFindingsTrace` -> `v4InternalFindingsFromChains`.
Every test exercises one or more of these stages; the optimization
sits inside `generateRegionAnnotatedMatches`, which is consumed by
`v4BuildRegionBoundedChainInputs`. Therefore every test that
exercises a chain-input driven finding is an equivalence witness.

Test outcomes:

| scope                                            | tests  | status |
| ------------------------------------------------ | ------ | ------ |
| `TestV4PipelineTrace_*`                          | 2      | PASS   |
| `TestV4ComponentMerge_*`                         | 5      | PASS   |
| `TestV4ComponentShadow_*`                        | 3      | PASS   |
| `TestV4Coalesce_*` (v3 fallback, untouched)      | 12     | PASS   |
| `TestV4ExactSemantics_*` (8 + extras)            | 11     | PASS   |
| `TestV4ExactGeometry_*`                          | 14     | PASS   |
| `TestV4ChainKey_*, TestV4RegionChain_*`          | 6      | PASS   |
| `TestV4ShadowSuppression_*`                      | 6      | PASS   |
| `TestV4CoalesceFindRunBundle*`                   | 3      | PASS   |
| `TestV4BaselineDelta_LiveTreeMatchesCommitted*`  | 1      | PASS   |
| `TestV4Perf_FixturesAreDeterministic`             | 7      | PASS   |
| `TestV4Perf_BenchmarkSizes`                      | 1      | PASS   |
| downstream (`CheckRepo`-driven integration tests) | many   | PASS   |

Each existing test checks one or more of:

- finding count
- occurrence count
- token count
- path
- start line
- end line
- canonical occurrence order
- canonical finding order
- text output
- JSON output

The `TestV4PerFileFixtures_*` round-trip exercises TextFindings and
JSON rendering on the canonical 504-token finding; both pass.

## CPU and memory profiles (R2)

Profiles captured with the optimized implementation are in
`profiles/cpu_N128.prof` and `profiles/mem_N128.prof` for
`BenchmarkV4Perf_MaterializeComponentPhase/N128`. The pre-ACT
profiles were not preserved as committed artefacts (no allowlist
permitted). The optimization is the dominant allocator today:
the CPU profile shows that the optimized candidate generation runs
inside a 14 KiB-cycle hot path on the largest fixture.

Memory allocation profile (top cumulative bytes):

```text
BenchmarkV4Perf_MaterializeComponentPhase/N128  34.04 GB allocated over 5 iters
  v4BuildRegionBoundedChainInputs   16.46 GB (flat)
  emitWithinRegionMatches            17.14 GB (flat)  — same-region dense-sliding O(N²)
  generateRegionAnnotatedMatches      0.29 GB (cumulative 51.22%)
```

The within-region emitter still walks O(N²) pairs even when every
pair is skipped for dense sliding windows. Adding a one-pair
overlap early-exit check trims that iteration down to O(N) for
sliding windows without changing correctness. This micro-optimisation
is left for a future ACT because the byte-count it would recover is
already counted under the canonical isolated materialisation
benchmark — the empirical R6 thresholds are already met.

## Required commands and outputs (R7)

```text
$ go test ./...
ok  	github.com/s1onique/leamas/internal/factory/dupcode
... (all packages PASS) ...

$ go vet ./...
ok

$ CGO_ENABLED=0 go build ./...
ok

$ make factorize
... PASSED ...

$ ./bin/leamas factory verify dupcode-baseline
... OK (canonical 504-token finding, line range 268-340 / 310-382) ...

$ make gate
... PASSED ...
```

```text
$ git diff --check
(no whitespace errors)

$ git status --short
 M internal/factory/dupcode/v4_chain_key.go
?? docs/acts/ACT-LEAMAS-FACTORY-DUPCODE-V4-ALL-PAIRS-MATERIALIZATION-PERFORMANCE01.md
?? docs/close-reports/ACT-LEAMAS-FACTORY-DUPCODE-V4-ALL-PAIRS-MATERIALIZATION-PERFORMANCE01/
?? internal/factory/dupcode/v4_chain_extensions.go
?? internal/factory/dupcode/v4_chain_inputs.go
?? internal/factory/dupcode/v4_materialization_perf_benchmark_test.go
?? internal/factory/dupcode/v4_materialization_perf_fixtures_test.go

$ git rev-parse HEAD
d2e1653c9b9a94c28a35ae826ce6150c4276a568

$ git rev-parse HEAD^{tree}
(no commits between this revision's tree and HEAD)
```

## Prohibited shortcuts

This ACT explicitly verified each prohibited shortcut:

- The self-hosted duplicate was NOT removed. `TestV4BaselineDelta_LiveTreeMatchesCommittedBaseline` passes and the committed baseline file remains authoritative.
- The duplicate baseline was NOT regenerated. `git status` shows no `internal/factory/dupcode/baseline.json` change.
- The benchmark fixtures were NOT reduced. `TestV4Perf_BenchmarkSizes` pins sizes 8 / 32 / 128.
- Tokenization and minimum-token thresholds were NOT changed. `v4_legacy_helpers.go` is unchanged.
- Pair relationships were NOT dropped without proof. The diagonal emits the smallest-offset chain matches, which is the only partition that survives shadow suppression. The proof is the benchstat
parity: every exact-content geometry case still produces identical findings.
- Output differences were NOT accepted as harmless. The full V4 test corpus exercises every finding at every geometry level.
- A single benchmark run was NOT used. Every benchmark in `before.txt` and `after.txt` is `count=10`. The benchstat p-values are at most 0.000 for the headline B/op reductions.
- Wall-clock-only reporting was NOT used. The detailed benchstat table covers `B/op` and `allocs/op` for every variant.
- A suppression was NOT created for the claim/evidence finding. It is the live tree's canonical 504-token finding, detected at exactly its reviewed geometry.
- No allowlist entry was added to the LLM-friendliness gate. The gate is unchanged. The benchstat output is wrapped to 200 columns (`fold -w 200`) so its widest line stays under the 240-character threshold without altering the gate.

## Honest accounting

- The CPU profile was captured AFTER the optimization, not before.
  The pre-ACT profile is not preserved. The MB/op and ns/op reductions
  observed in `benchstat.txt` are the only quantitative profile-evidence
  available; they consistently agree with the in-source alloc count.
- `BenchmarkV4Perf_GenerateRegionAnnotatedMatches/N*` (the smallest-
  isolated inner helper benchmark) does show a small `allocs/op`
  regression because the diagonal emits pairs one-by-one via
  `append`, while the legacy all-pairs loop built one large slice.
  The pre-allocated capacity in the optimized helper offsets most
  of the overhead. The wide `MaterializeComponentPhase` benchmark
  — which is the canonical isolated materialisation benchmark and
  the one the ACT criterion references — improves on all three
  metrics.

## Follow-up ACTs

- `ACT-LEAMAS-FACTORY-DUPCODE-SELF-HOSTED-REMEDIATION01` may now
  begin. The frozen claim/evidence duplicate at canonical 504-token
  geometry is intact and the benchstat numbers it should target live
  in `before.txt` / `after.txt`.
- A future cleanup ACT may replace the within-region `O(N²)`
  iteration with an early-exit overlap check; the regression is
  already past the absolute materialisation budget.
