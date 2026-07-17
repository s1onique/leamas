# ACT-LEAMAS-FACTORY-DUPCODE-V4-ALL-PAIRS-MATERIALIZATION-PERFORMANCE01-CORRECTION02-R1-CROSS-REGION-PROOF01

## Status: PASSED — corrected asymmetric fixture cross-region proof; mutation evidence captured

`ACT-LEAMAS-FACTORY-DUPCODE-V4-ALL-PAIRS-MATERIALIZATION-PERFORMANCE01-CORRECTION02-R1-CROSS-REGION-PROOF01`
is **PASSED**. The single highest-priority defect in the CORRECTION01
test suite is repaired and proved:

> The asymmetric leading-extra regression fixture now represents two
> distinct production syntax regions, activates the unaligned
> cross-region all-pairs fallback, preserves the offset-100 maximal
> chain, and fails when the original unconditional diagonal behavior
> is temporarily restored.

This ACT closes only the R1 cross-region regression-proof defect.
It does NOT close CORRECTION02, CORRECTION01, the parent performance
ACT, or the self-hosted-remediation prerequisite.

## The defect that motivated this ACT

The CORRECTION01 test suite introduced an asymmetric fixture
`v4AsymmetricLeadingExtra()` that was supposed to prove the
alignment guard correctly rejects cross-region sequences that are
not position-by-position aligned:

```text
left  starts:  [0, 1, 2]
right starts:  [50, 100, 101, 102]
```

The fixture was constructed with a single-path helper that hard-coded
`Path: "alpha.go"` for both sides. The right side silently re-used
the left side's path. Both windows resolved to the same production
syntax region (`alpha.go#0`), the alignment guard was never
consulted, and the regression proof passed without ever exercising
the cross-region all-pairs fallback.

The new tests in this ACT exercise:

* two distinct production regions (`alpha.go#0` and `beta.go#0`);
* the alignment guard rejecting the asymmetric sequences;
* the conservative all-pairs candidate generator producing the
  offset-100 chain;
* final canonical equality between production and the legacy
  all-pairs oracle;
* the mirrored left-side-extra case;
* the unconditional-diagonal mutation that fails the asymmetric
  test.

## Files changed

```text
M  internal/factory/dupcode/v4_alignment_differential_test.go
?? internal/factory/dupcode/v4_alignment_cross_region_fixtures_test.go    (232 lines)
?? internal/factory/dupcode/v4_alignment_cross_region_asymmetric_test.go  (337 lines)
?? internal/factory/dupcode/v4_alignment_cross_region_corpus_test.go      (145 lines)
?? docs/acts/.../ACT-LEAMAS-FACTORY-DUPCODE-V4-ALL-PAIRS-MATERIALIZATION-
   PERFORMANCE01-CORRECTION02-R1-CROSS-REGION-PROOF01.md
?? docs/close-reports/.../ACT-LEAMAS-FACTORY-DUPCODE-V4-ALL-PAIRS-MATERIALIZATION-
   PERFORMANCE01-CORRECTION02-R1-CROSS-REGION-PROOF01.md
```

The 232+337+145 line split (all under the 400-line LLM-friendly
threshold) was driven by the factorize gate. The fixtures and shared
helpers live in `v4_alignment_cross_region_fixtures_test.go`; the
R2-R5 asymmetric tests live in
`v4_alignment_cross_region_asymmetric_test.go`; the R6 three-case
minimal differential table lives in
`v4_alignment_cross_region_corpus_test.go`.

## R1 — Corrected the asymmetric fixture

The CORRECTION01 fixture `v4AsymmetricLeadingExtra()` was rewritten
to use a new path-aware constructor:

```go
func makeRawWindows(path string, starts []int) []v4RawWindow {
    out := make([]v4RawWindow, 0, len(starts))
    for i, sp := range starts {
        out = append(out, v4RawWindow{
            Path:      path,
            StartPos:  sp,
            EndPos:    sp + 80,
            StartLine: 100 + i,
            EndLine:   100 + i + 80,
        })
    }
    return out
}
```

The fixture now constructs:

```text
left:
    path   = alpha.go
    starts = [0, 1, 2]

right:
    path   = beta.go
    starts = [50, 100, 101, 102]
```

Both paths are present in `PerPathLength` and in the synthetic
analyses. A window's side is NEVER inferred from its position in
some enclosing fixture; the path is stored in every raw window.

## R2 — Fixture-contract test

`TestV4Alignment_AsymmetricLeadingExtra_FixtureContract` asserts
the corrected fixture's preconditions before the candidate generator
runs:

* left window count = 3, right window count = 4;
* every left path is `alpha.go`, every right path is `beta.go`;
* left region path = `alpha.go`, right region path = `beta.go`;
* left region ID != right region ID;
* left starts = `[0, 1, 2]`, right starts = `[50, 100, 101, 102]`.

The test fails with a specific diagnostic if both sides accidentally
use the same path again — e.g.
`left region alpha.go#0 == right region alpha.go#0 (fixture collapsed
to a single region)`.

## R3 — Alignment-guard-rejection test

`TestV4Alignment_AsymmetricLeadingExtra_AlignmentGuardRejects`
builds the same annotated-window and region-index inputs the
production candidate generator sees, and asserts:

```go
regionsArePositionallyAligned(idxA, idxB, annotated) == false
```

The failure diagnostic prints every value a reviewer needs:

```text
alignment guard returned true for the asymmetric fixture
  leftRegion   = alpha.go#0
  rightRegion  = beta.go#0
  left starts  = [0, 1, 2]
  right starts = [50, 100, 101, 102]
  observed     = true
  expected     = false
```

This assertion runs INDEPENDENTLY of the final canonical comparison;
the guard verdict is observable on its own.

## R4 — Conservative-candidate-geometry test

`TestV4Alignment_AsymmetricLeadingExtra_ConservativeCandidateGeometry`
calls the production candidate generator
(`v4BuildRegionBoundedChainInputs`) — the equivalent production
seam — and asserts the candidate set contains these exact matches:

```text
alpha.go start 0 ↔ beta.go start 100    offset = 100
alpha.go start 1 ↔ beta.go start 101    offset = 100
alpha.go start 2 ↔ beta.go start 102    offset = 100
```

Each match carries the correct `LeftRegion` (`alpha.go#0`) and
`RightRegion` (`beta.go#0`) identity, and all three matches belong
to one constant-offset partition keyed by `(alpha.go#0, beta.go#0,
offset=100)`.

The test fails with:

```text
candidate set missing required match alpha.go@0 ↔ beta.go@100
  combined = 3 matches
```

when the conservative all-pairs fallback does not produce the
offset-100 chain — exactly what the unconditional-diagonal mutation
caused in R7.

## R5 — Production-equals-oracle test

`TestV4Alignment_AsymmetricLeadingExtra_ProductionEqualsOracle`
runs:

* production guarded pipeline (`v4BuildInternalFindings`);
* legacy all-pairs test oracle
  (`v4BuildInternalFindingsOracle` driven by
  `v4GenerateAllPairsMatchesOracle`);

against the corrected fixture. The comparison covers every canonical
internal value the seam surfaces:

* finding count;
* `StableFingerprint`;
* `TokenCount`;
* `LineCount`;
* occurrence count;
* occurrence path;
* occurrence StartPos / EndPos;
* occurrence StartLine / EndLine;
* canonical ordering of findings and occurrences.

The assertion is described as "structurally equal"; the test does
NOT render the findings as text or JSON, it compares the live
canonical structs field-by-field. Acceptance wording:

> The production and legacy-oracle canonical internal findings are
> structurally equal for the corrected asymmetric cross-region
> fixture.

## R6 — Three-case minimal differential table

`TestV4Alignment_MinimalCrossRegionCorpus` runs three table-driven
cases, each with a unique name:

1. `AlignedDistinctRegions`

   * `alpha = [0, 1, 2]`, `beta = [100, 101, 102]`;
   * guard returns true;
   * offset-100 chain at offset 100 from the diagonal;
   * diagonal path is valid.

2. `AsymmetricLeadingExtraRight`

   * `alpha = [0, 1, 2]`, `beta = [50, 100, 101, 102]`;
   * guard returns false;
   * offset-100 chain survives.

3. `AsymmetricLeadingExtraLeft`

   * `alpha = [50, 100, 101, 102]`, `beta = [0, 1, 2]`;
   * guard returns false;
   * offset -100 (canonicalized) chain survives.

Every case asserts its intended guard verdict BEFORE comparing
production with the oracle, so the failure diagnostic localises a
regression to one row.

This is intentionally NOT the complete CORRECTION02 corpus. The
remaining adversarial dimensions, committed fuzz regression, and
30-second fuzz run belong to the successor ACT
`ACT-LEAMAS-FACTORY-DUPCODE-V4-ALL-PAIRS-MATERIALIZATION-PERFORMANCE01-CORRECTION02-CORPUS-AND-EVIDENCE01`.

## R7 — Mutation proof

The ACT temporarily changed the production selection logic in
`internal/factory/dupcode/v4_chain_inputs.go` from:

```go
if regionsArePositionallyAligned(idxA, idxB, annotatedWindows) {
    emitCrossRegionDiagonalMatches(fp, ridA, ridB, idxA, idxB, annotatedWindows, &out)
} else {
    emitCrossRegionAllPairsMatches(fp, ridA, ridB, idxA, idxB, annotatedWindows, &out)
}
```

to unconditional diagonal selection:

```go
emitCrossRegionDiagonalMatches(fp, ridA, ridB, idxA, idxB, annotatedWindows, &out)
```

and ran:

```text
go test ./internal/factory/dupcode \
  -run='^TestV4Alignment_(AsymmetricLeadingExtra|MinimalCrossRegionCorpus)'
```

Exit code: 1.

Failing tests:

* `TestV4Alignment_AsymmetricLeadingExtra_ConservativeCandidateGeometry`
  — `candidate set missing required match alpha.go@0 ↔ beta.go@100,
  combined = 3 matches`
* `TestV4Alignment_AsymmetricLeadingExtra_ProductionEqualsOracle`
  — finding-count drift prod=2 ora=4, fingerprint drift, token-count
  drift, line-count drift, token-position drift, line-range drift.
* `TestV4Alignment_MinimalCrossRegionCorpus/AsymmetricLeadingExtraRight`
  — `constant-offset partition (offset=100) did not survive,
  partitions = alpha.go#0/beta.go#0@50=1, alpha.go#0/beta.go#0@99=2`
* `TestV4Alignment_MinimalCrossRegionCorpus/AsymmetricLeadingExtraLeft`
  — `constant-offset partition (offset=-100) did not survive,
  partitions = alpha.go#0/beta.go#0@-99=2, alpha.go#0/beta.go#0@-50=1`
* `TestV4Alignment_AsymmetricLeadingExtra_Regression`
  — occurrence-count drift, fingerprint drift, token-count drift,
  occurrence drift at the new geometry.

The production source was restored immediately. The same command
re-ran and passed:

```text
ok  	github.com/s1onique/leamas/internal/factory/dupcode	0.388s
```

The temporary mutation was NOT committed.

## R8 — No production change

The expected production delta was:

```text
none
```

The repaired fixture did not expose a real production divergence.
The temporary mutation evidence in R7 is captured as test-side
proof only; the production source is byte-identical to its pre-ACT
state. `git diff internal/factory/dupcode/v4_chain_inputs.go` after
restoration was empty.

## R9 — Focused verification

```text
gofmt -w internal/factory/dupcode/v4_alignment_*_test.go      OK
test -z "$(gofmt -l internal/factory/dupcode/v4_alignment_*_test.go)"   clean

go test ./internal/factory/dupcode \
  -run='^TestV4Alignment_(AsymmetricLeadingExtra|MinimalCrossRegionCorpus|RegionsArePositionallyAligned)'
                                                             PASS

go test ./internal/factory/dupcode                            PASS  (143.658s)
go test -race ./internal/factory/dupcode \
  -run='^TestV4Alignment_(AsymmetricLeadingExtra|MinimalCrossRegionCorpus|RegionsArePositionallyAligned)'
                                                             PASS  (2.258s)

git diff --check                                             clean
make factorize                                               PASS
```

Required repository-wide commands:

```text
go test ./...                                                PASS
go vet ./...                                                 OK
CGO_ENABLED=0 go build ./...                                OK

./bin/leamas factory verify dupcode-baseline                 OK  (canonical 504-token claim/evidence finding intact)
make gate                                                    PASS
```

## R10 — Frozen remediation target preserved

The following files were NOT modified:

```text
cmd/leamas/claim_commands.go
cmd/leamas/evidence_commands.go
internal/factory/dupcode/baseline.json
```

The live detector retains:

```text
TokenCount = 504

claim_commands.go:
    268–340

evidence_commands.go:
    310–382
```

`TestV4BaselineDelta_LiveTreeMatchesCommittedBaseline` PASSES,
confirming the canonical 504-token claim/evidence duplicate is
intact at its reviewed geometry.

## R11-R12, shortcuts, accounting, and successor

The remaining historical closure text was split without changing its
meaning to keep this canonical close report below the 400-line
LLM-friendliness limit. It is preserved in:

```text
docs/close-reports/
  ACT-LEAMAS-FACTORY-DUPCODE-V4-ALL-PAIRS-MATERIALIZATION-
  PERFORMANCE01-CORRECTION02-R1-CROSS-REGION-PROOF01/
    close-closure-appendix.md
```

That appendix records documentation and commit closure, prohibited
shortcuts, honest accounting, and successor scope. The split is
historical-document hygiene only.
