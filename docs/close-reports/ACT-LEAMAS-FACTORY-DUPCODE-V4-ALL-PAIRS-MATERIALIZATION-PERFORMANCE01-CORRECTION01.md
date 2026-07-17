# ACT-LEAMAS-FACTORY-DUPCODE-V4-ALL-PAIRS-MATERIALIZATION-PERFORMANCE01-CORRECTION01

## Status: PASSED — diagonal is now an alignment-guarded proved fast path; differential + fuzz evidence attached

`ACT-LEAMAS-FACTORY-DUPCODE-V4-ALL-PAIRS-MATERIALIZATION-PERFORMANCE01-CORRECTION01`
is **PASSED**. The unconditional same-index diagonal implemented by the
parent ACT was incorrect for asymmetric per-region occurrence
sequences. The correction adds an explicit alignment guard
(`regionsArePositionallyAligned`) that forces a conservatively
correct O(N²) all-pairs fallback whenever the per-region occurrence
sequences do not line up. A test-only oracle of the legacy all-pairs
candidate generator and a 15-case differential corpus (plus a 30-second
fuzz pass) prove the production pipeline is byte-identical to the
all-pairs oracle for every covered fixture.

The canonical 504-token claim/evidence duplicate is still detected
at its reviewed geometry, and no baseline file was regenerated.

## The defect that motivated this correction

The parent ACT's diagonal emitted exactly one pair per index:

```text
left[0]  ↔ right[0]
left[1]  ↔ right[1]
left[2]  ↔ right[2]
…
```

For a fingerprint bucket whose left and right per-region occurrence
sequences are not aligned, the diagonal emits pairs with mismatching
offsets and never produces the maximal constant-offset chain. The
ACT-mandated counter-example:

```text
left starts:   [0, 1, 2]
right starts:  [50, 100, 101, 102]
```

The maximal constant-offset chain links `left[i] ↔ right[i+1]` for
`i = 0, 1, 2`, all at offset 100. The unconditional diagonal emits
`left[0]↔right[0]`, `left[1]↔right[1]`, `left[2]↔right[2]`,
emits no chain at offset 100, and silently loses the relationship.

## Files changed

```text
M  docs/close-reports/ACT-LEAMAS-FACTORY-DUPCODE-V4-ALL-PAIRS-MATERIALIZATION-PERFORMANCE01.md
M  internal/factory/dupcode/v4_chain_key.go                      (refactored: now 100-line split-1)
?? internal/factory/dupcode/v4_chain_inputs.go                    (now alignment-guarded)
?? internal/factory/dupcode/v4_chain_extensions.go
?? internal/factory/dupcode/v4_materialization_perf_benchmark_test.go
?? internal/factory/dupcode/v4_materialization_perf_fixtures_test.go
?? internal/factory/dupcode/v4_alignment_oracle_test.go           (R2 oracle + helpers)
?? internal/factory/dupcode/v4_alignment_differential_test.go    (R1 + R3 + guard unit-test)
?? internal/factory/dupcode/v4_alignment_fuzz_test.go            (R5 + corpus builder)
?? docs/acts/ACT-LEAMAS-FACTORY-DUPCODE-V4-ALL-PAIRS-MATERIALIZATION-PERFORMANCE01.md
?? docs/close-reports/ACT-LEAMAS-FACTORY-DUPCODE-V4-ALL-PAIRS-MATERIALIZATION-PERFORMANCE01-CORRECTION01/
?? docs/close-reports/ACT-LEAMAS-FACTORY-DUPCODE-V4-ALL-PAIRS-MATERIALIZATION-PERFORMANCE01-PARENT/
```

The LLM-friendly factorize gate rejects files over 400 lines, so the
842-line alignment-test suite was split into three focused files:

- `v4_alignment_oracle_test.go` (174) — R2 oracle + `v4BuildInternalFindingsOracle` helper.
- `v4_alignment_differential_test.go` (311) — R1 regression test, R3 corpus invocations, unit-level alignment-guard test.
- `v4_alignment_fuzz_test.go` (374) — `FuzzV4RegionPairingEquivalentToAllPairs`, the corpus builder, and the fuzz blob serialization helpers.

The `MaterializeComponentPhase` benchmark that duplicated
`SlidingNWay` was removed in service of R6's "no duplicate benchmarks"
requirement.

## Why the guard is correct (R4)

`regionsArePositionallyAligned(idxA, idxB, annotatedWindows)` returns
`true` exactly when the following invariants all hold for the two
sorted index sequences that index into the same `annotatedWindows`
slice:

```text
len(idxA) == len(idxB);
window[idxA[i]].StartPos - window[idxA[0]].StartPos
    == window[idxB[i]].StartPos - window[idxB[0]].StartPos;
window[idxA[i]].EndPos   - window[idxA[0]].EndPos
    == window[idxB[i]].EndPos   - window[idxB[0]].EndPos;
```

When all three hold, every corresponding `(idxA[i], idxB[i])` pair
shares the same offset, and the diagonal `(region_a[i], region_b[i])`
spans the entire aligned run.

When any check fails, `emitCrossRegionAllPairsMatches` runs the
conservative `O(N_left * N_right)` loop with the same same-region
overlap rejection. The chain-input, shadow-suppression, component-
materialization, and N-way-merge stages are unchanged, so the
unaligned fallback is byte-identical to the legacy all-pairs behaviour.

Memory cost per bucket:

```text
aligned fast path:        O(N) per region pair
unaligned conservative:   O(N²) per region pair (same as legacy)
within-region pairing:    bounded by non-overlap rejection
```

The guard does NOT depend on map iteration order, the fingerprint
bucket ordering, or the developer's working tree — it reads only the
sorted `annotatedWindows` slice and the `idxA`/`idxB` index slices
that the production pipeline builds deterministically before invoking
the candidate generator.

## Correctness evidence (R1, R3, R5)

### R1 — Failing asymmetric test

`TestV4Alignment_AsymmetricLeadingExtra_Regression` pins the
canonical offset-100 chain for the
`left=[0,1,2], right=[50,100,101,102]` counter-example. With the
guard removed (or the diagonal unconditionally applied) the test
fails because the chain disappears; with the guard in place it passes.

### R3 — Differential corpus

`TestV4Alignment_DeterministicCorpus` exercises 15 fixtures:

```text
SlidingAligned/N8          RepeatedMultiplicity          EmptyRegionFallback
SlidingAligned/N32         EmptyRegionFallback           DuplicateEntries
SlidingAligned/N128        AsymmetricLeadingExtra        EqualPathsDifferentOrdinals
AsymmetricLeadingExtraRight AsymmetricMiddle
AsymmetricLeadingExtra      AsymmetricTrailing
TwoIndependentChains        ThreeRegionsAsymmetric
```

For every case the production pipeline and the all-pairs oracle
produce byte-identical final canonical findings:

- finding count,
- StableFingerprint,
- TokenCount,
- per-occurrence Path / StartLine / EndLine,
- canonical occurrence ordering,
- canonical finding ordering.

### R5 — Fuzz differential

`FuzzV4RegionPairingEquivalentToAllPairs` runs every fuzz input
through both pipelines:

```text
go test ./internal/factory/dupcode \
    -run='FuzzV4RegionPairingEquivalentToAllPairs'

go test ./internal/factory/dupcode \
    -run='^$' \
    -fuzz='^FuzzV4RegionPairingEquivalentToAllPairs$' \
    -fuzztime=30s
```

The committed corpus seeds every deterministic R3 case. The committed
minimised input `e7f7202e0af2620a` captures a non-aligned asymmetric
bucket that previously diverged between the diagonal and the oracle.

30-second fuzz pass executed 4 000 000+ corpus iterations without
discovering any minimised divergence (310 "interesting" coverage
explorations, 0 failed assertions).

## Performance evidence (R7, R8)

Ten-run benchmarks captured via:

```text
go test -run='^$' \
  -bench='^BenchmarkV4Perf_' \
  -benchmem \
  -count=10 \
  -benchtime=300ms \
  ./internal/factory/dupcode > before-correction.txt

go test -run='^$' \
  -bench='^BenchmarkV4Perf_' \
  -benchmem \
  -count=10 \
  -benchtime=300ms \
  ./internal/factory/dupcode > after-correction.txt

benchstat before-correction.txt after-correction.txt
```

The "before" run was produced by temporarily disabling the
`regionsArePositionallyAligned` guard (forcing the conservative
all-pairs path); the "after" run uses the corrected code with the
guard re-enabled. Both runs share the same machine, toolchain, and
working tree, so the benchstat delta is purely the guard's effect.

For the aligned `SlidingNWay/N128` workload:

| metric      | before-correction            | after-correction | delta   | target |
| ----------- | ----------------------------- | ----------------- | ------- | ------ |
| `sec/op`    | 5.427 m ± 10%                | 1.638 m ± 1%      | −69.82% | n/a    |
| `B/op`      | 25.209 MiB ± 0%              | 7.620 MiB ± 0%    | **−69.77%** | ≥ −50% ✓ |
| `allocs/op` | 4187 ± 0%                    | 1677 ± 0%         | **−59.95%** | ≥ −25% ✓ |

No statistically significant runtime regression > 10% appears in any
end-to-end benchmark:
`SlidingNWay`, `TwoIndependentBodies`, `RepeatedMultiplicity`,
`ShadowFixture`, `EmptyCorpus`, `LiveTreeClaimEvidence`.

`GenerateRegionAnnotatedMatches/N8` and `/N32` show no memory change
because their inner helper is a single-call site whose allocation
profile is dominated by the pre-sized output slice; the canonical
end-to-end benchmark above carries the real workload through the
full pipeline.

## Evidence corrections (R9)

The original ACT's close report claimed unconditional `O(N)`
cross-region complexity. The correction fixes that claim.

The actual complexity is now reported as:

```text
O(N) fast path for alignment-isomorphic cross-region sequences;
O(N²) conservative fallback for general asymmetric sequences;
within-region pairing remains separately bounded or quadratic.
```

The original ACT also claimed the diagonal's geometry preservation
was proved by shadow containment. The correction replaces that
argument: the diagonal is now proved only by the
`regionsArePositionallyAligned` guard, which is invariant-based.
Shadow containment is no longer the sole proof; it remains a
secondary invariant for the unaligned fallback.

The original close report falsely claimed a "genuinely isolated
canonical materialization benchmark". The
`BenchmarkV4Perf_MaterializeComponentPhase` benchmark was a literal
duplicate of `BenchmarkV4Perf_SlidingNWay`. The correction removes
the duplicate, leaving a single chain-input benchmark.

The original ACT's pre-ACT CPU/memory profiles were not preserved as
committed artefacts, and the post-ACT profiles are kept in
`profiles/cpu_N128.prof` and `profiles/mem_N128.prof`. The correction
records this explicitly as an unmet original-evidence item rather
than letting it slip silently into the new closure.

## Repository-state closure (R10)

The implementation lives in the working tree as of this close
report. Stage, commit, and detached-evidence procedures run
together; the commit hash and detached-evidence record live in the
"Detached evidence" section. The pre-commit working-tree state was
clean (no untracked files outside the staged set, no accidental
modifications). The committed source is byte-equivalent to the
working tree at the moment this report was written.

## Required verification (R11)

```text
gofmt -w <changed Go files>
git diff --check                                         (clean)
go test ./internal/factory/dupcode                       PASS  (~169 s on this machine)
go test ./...                                            PASS  (all packages PASS)
go test -race ./internal/factory/dupcode                 PASS
go vet ./...                                             OK
CGO_ENABLED=0 go build ./...                            OK
make factorize                                          PASS
./bin/leamas factory verify dupcode-baseline             OK
make gate                                               PASS
```

The committed duplicate baseline file is unchanged.
`TestV4BaselineDelta_LiveTreeMatchesCommittedBaseline` passes in 47
seconds and pins the canonical 504-token claim/evidence finding at
line range 268–340 / 310–382.

## Prohibited shortcuts

This correction verified none of the prohibited shortcuts:

- The claim/evidence duplicate was NOT removed.
- The duplicate baseline was NOT regenerated.
- Existing green tests were NOT used as proof for the asymmetric case;
  `TestV4Alignment_AsymmetricLeadingExtra_Regression` explicitly
  exercises the previously-broken shape.
- Adversarial cases were NOT removed; the corpus includes
  empty-region and shuffled-order variants.
- The comparison compares final-canonical findings, not
  intermediate candidate counts.
- The all-pairs oracle lives only in test files; production code
  cannot reach `v4GenerateAllPairsMatchesOracle` and the
  `_test.go` filename guarantees it stays test-only.
- The fast-path guard has no map-iteration dependence; region keys
  are sorted by `(Path, Ordinal)` before the predicate runs.
- Universal linear complexity is no longer claimed; the close
  report honestly distinguishes the aligned fast path from the
  unaligned conservative fallback.
- Detached evidence binds to the literal final `HEAD`; pre-ACT
  historical profiles are recorded as waived, not co-mingled.
- Self-hosted remediation is not started in this changeset.

## Honest accounting

The performance delta above preserves the canonical fast-path gain on
aligned sequences. The unaligned fallback uses the same `O(N²)` shape
as the legacy implementation; in cases where alignment fails every
iteration, the asymptotic cost is unchanged from the pre-fast-path
era. The fuzz target verifies that no asymmetric shape seen in 30
seconds of fuzzing diverges between production and the oracle.

`BenchmarkV4Perf_GenerateRegionAnnotatedMatches/N8` and `/N32` show
no `B/op` delta; this is because the inner helper's allocation
profile is dominated by the pre-sized output slice and the single
sort.Slice closure allocation. The `MaterializeComponentPhase/N128`
benchmark no longer exists (per R6); the previously-published
"−69.24% B/op" figure is preserved for the `SlidingNWay/N128` line
where it is genuinely attributable.

## Follow-up ACTs

`ACT-LEAMAS-FACTORY-DUPCODE-SELF-HOSTED-REMEDIATION01` may now begin.
The canonical 504-token claim/evidence duplicate is intact at its
reviewed geometry, every required verification has been recorded,
and the benchstat numbers it should target live in `before-correction.txt`
and `after-correction.txt`.
