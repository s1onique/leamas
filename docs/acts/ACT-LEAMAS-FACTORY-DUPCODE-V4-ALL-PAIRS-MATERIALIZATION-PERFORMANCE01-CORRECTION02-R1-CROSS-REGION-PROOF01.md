# ACT-LEAMAS-FACTORY-DUPCODE-V4-ALL-PAIRS-MATERIALIZATION-PERFORMANCE01-CORRECTION02-R1-CROSS-REGION-PROOF01

## Status

READY

## Parent correction wave

`ACT-LEAMAS-FACTORY-DUPCODE-V4-ALL-PAIRS-MATERIALIZATION-PERFORMANCE01-CORRECTION02`

## Baseline

```text
HEAD = a66b3729d43a709db999b5e6a0d33bf344760cf9
working tree = clean
index = clean
```

## Intent

Repair and prove the single highest-priority defect in the
CORRECTION01 test suite:

> The asymmetric leading-extra regression fixture must represent two
> distinct production syntax regions, activate the unaligned
> cross-region all-pairs fallback, preserve the offset-100 maximal
> chain, and fail when the original unconditional diagonal behavior
> is temporarily restored.

This ACT deliberately does not attempt the full adversarial corpus,
committed fuzz corpus, benchmark refresh, historical-document
reconciliation, or final performance-ACT closure.

## Scope

This ACT owns only:

1. correcting the asymmetric fixture's paths (R1);
2. proving its region ownership (R2);
3. proving its alignment classification (R3);
4. proving the expected candidate geometry (R4);
5. proving final production/oracle equivalence (R5);
6. proving that the test rejects the original unconditional
   diagonal (R7);
7. installing a three-case minimal differential table (R6);
8. committing this minimal proof on a clean repository state (R12).

## Out of scope

The following remain assigned to a later correction ACT:

* the complete 17-dimension deterministic corpus;
* shuffled-input proof;
* unowned-window proof;
* same-path/different-region-ordinal proof;
* generalized corpus structural validation;
* fuzz serialization redesign;
* 30-second fuzz execution;
* committed `testdata/fuzz` regression input;
* benchmark regeneration;
* CORRECTION01 benchmark-summary whitespace cleanup;
* parent and child lifecycle-document reconciliation;
* final performance-ACT closure.

No item above may be claimed as completed by this ACT.

## R1 — Correct the asymmetric fixture

The CORRECTION01 fixture `v4AsymmetricLeadingExtra()` used a
single-path helper `mkLeft(starts)` for both sides. The right side
silently re-used the left side's path (`alpha.go`), the two sides
resolved to the same production syntax region, and the alignment
guard was never consulted.

This ACT replaces the one-path helper with an explicit
path-aware constructor:

```go
func makeRawWindows(path string, starts []int) []v4RawWindow {
    // Construct windows using the supplied path.
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

Both `alpha.go` and `beta.go` exist in:

```go
PerPathLength
```

and in the generated analysis and file inventories.

A window's side is NEVER inferred from its position or slice
membership. The path is stored in every raw window.

## R2 — Prove fixture preconditions

`TestV4Alignment_AsymmetricLeadingExtra_FixtureContract` asserts
before running the detector:

```text
len(left windows)  = 3
len(right windows) = 4

every left path  = alpha.go
every right path = beta.go

left region path  = alpha.go
right region path = beta.go

left region ID != right region ID
```

The test also asserts the exact start-position sequences:

```text
left  = [0, 1, 2]
right = [50, 100, 101, 102]
```

The test fails with a specific diagnostic if both sides accidentally
use the same path again.

## R3 — Prove the alignment guard rejects the fixture

`TestV4Alignment_AsymmetricLeadingExtra_AlignmentGuardRejects`
builds the same annotated-window and region-index inputs used by
production and asserts:

```go
regionsArePositionallyAligned(leftIndexes, rightIndexes, annotatedWindows) == false
```

The failure diagnostic prints:

* left region ID;
* right region ID;
* left starts;
* right starts;
* observed alignment result.

This assertion occurs independently of final oracle equality.

## R4 — Prove the conservative candidate geometry

`TestV4Alignment_AsymmetricLeadingExtra_ConservativeCandidateGeometry`
calls the production candidate generator
(`v4BuildRegionBoundedChainInputs`) — the equivalent production
seam — and asserts the candidate set contains these exact matches:

```text
alpha.go start 0 ↔ beta.go start 100
alpha.go start 1 ↔ beta.go start 101
alpha.go start 2 ↔ beta.go start 102
```

Every match must have:

```text
offset = 100
left region path  = alpha.go
right region path = beta.go
```

The test asserts these three matches belong to one constant-offset
partition. Canonicalized candidate values match exactly.

## R5 — Prove final canonical equivalence

`TestV4Alignment_AsymmetricLeadingExtra_ProductionEqualsOracle`
runs:

```text
production guarded pipeline
legacy all-pairs test oracle
```

against the corrected fixture and compares the complete canonical
internal values available at this seam, including:

* finding count;
* stable fingerprint;
* token count;
* line count;
* complete occurrence count;
* occurrence paths;
* occurrence start and end token positions;
* occurrence start and end lines;
* occurrence ordering;
* finding ordering;
* errors or diagnostics.

The comparison is described as "structurally equal"; the test does
not render the findings as text or JSON, it compares the live
canonical structs field-by-field.

Acceptable closure wording:

```text
The production and legacy-oracle canonical internal findings are
structurally equal for the corrected asymmetric cross-region fixture.
```

## R6 — Three-case minimal differential table

`TestV4Alignment_MinimalCrossRegionCorpus` runs three table-driven
cases:

1. `AlignedDistinctRegions`

   * equal cardinality;
   * identical relative positions;
   * guard returns true;
   * diagonal path is valid.

2. `AsymmetricLeadingExtraRight`

   * `alpha=[0,1,2]`;
   * `beta=[50,100,101,102]`;
   * guard returns false;
   * offset-100 chain survives.

3. `AsymmetricLeadingExtraLeft`

   * mirror of case 2;
   * guard returns false;
   * corresponding off-index maximal chain survives.

Every case has a unique name. For every case, the test asserts its
intended guard result before comparing production with the oracle.
The test is NOT the complete CORRECTION02 corpus.

## R7 — Mutation proof

The ACT temporarily changed the production selection logic from:

```go
if regionsArePositionallyAligned(...) {
    emitCrossRegionDiagonalMatches(...)
} else {
    emitCrossRegionAllPairsMatches(...)
}
```

to unconditional diagonal selection and ran:

```text
go test ./internal/factory/dupcode \
  -run='^TestV4Alignment_(AsymmetricLeadingExtra|MinimalCrossRegionCorpus)'
```

The mutation evidence:

```text
temporary mutation
  - Phase 2: cross-region candidates. Unconditional diagonal
    selection (regression mutation — proves the guard matters).
  - The `if regionsArePositionallyAligned(...) { ... } else { ... }`
    branch was replaced with a single
    `emitCrossRegionDiagonalMatches(...)` call.
exact command
  - go test ./internal/factory/dupcode \
      -run='^TestV4Alignment_(AsymmetricLeadingExtra|MinimalCrossRegionCorpus)'
exit code
  - 1
failing test
  - TestV4Alignment_AsymmetricLeadingExtra_ConservativeCandidateGeometry
  - TestV4Alignment_AsymmetricLeadingExtra_ProductionEqualsOracle
  - TestV4Alignment_MinimalCrossRegionCorpus/AsymmetricLeadingExtraRight
  - TestV4Alignment_MinimalCrossRegionCorpus/AsymmetricLeadingExtraLeft
  - TestV4Alignment_AsymmetricLeadingExtra_Regression
failure diagnostic (excerpt)
  - candidate set missing required match alpha.go@0 ↔ beta.go@100
    combined = 3 matches
  - AsymmetricLeadingExtraRight: constant-offset partition
    (offset=100) did not survive
    partitions = alpha.go#0/beta.go#0@50=1, alpha.go#0/beta.go#0@99=2
  - AsymmetricLeadingExtraLeft: constant-offset partition
    (offset=-100) did not survive
    partitions = alpha.go#0/beta.go#0@-99=2, alpha.go#0/beta.go#0@-50=1
```

The production source was restored immediately and the same command
reran; all tests passed:

```text
ok  	github.com/s1onique/leamas/internal/factory/dupcode	0.388s
```

The temporary mutation was NOT committed.

## R8 — No production change by default

The expected production delta was:

```text
none
```

The repaired fixture did not expose a real production divergence.
The temporary mutation evidence in R7 is captured as test-side
proof only; the production source is byte-identical to its
pre-ACT state.

## R9 — Focused verification

Required focused commands:

```text
gofmt -w internal/factory/dupcode/v4_alignment_*_test.go
test -z "$(gofmt -l internal/factory/dupcode/v4_alignment_*_test.go)"

go test ./internal/factory/dupcode \
  -run='^TestV4Alignment_(AsymmetricLeadingExtra|MinimalCrossRegionCorpus|RegionsArePositionallyAligned)'

go test ./internal/factory/dupcode
go test -race ./internal/factory/dupcode \
  -run='^TestV4Alignment_(AsymmetricLeadingExtra|MinimalCrossRegionCorpus|RegionsArePositionallyAligned)'

git diff --check
make factorize
```

All focused commands PASS.

Required repository-wide commands:

```text
go test ./...
go vet ./...
CGO_ENABLED=0 go build ./...

./bin/leamas factory verify dupcode-baseline
make gate
```

All repository-wide commands PASS.

## R10-R12, acceptance, shortcuts, and successor

The remaining historical contract text was split without changing its
meaning to keep this canonical ACT below the 400-line LLM-friendliness
limit. It is preserved in:

```text
docs/close-reports/
  ACT-LEAMAS-FACTORY-DUPCODE-V4-ALL-PAIRS-MATERIALIZATION-
  PERFORMANCE01-CORRECTION02-R1-CROSS-REGION-PROOF01/
    act-closure-appendix.md
```

That appendix records the frozen 504-token target, documentation and
commit closure, acceptance criteria, prohibited shortcuts, and immediate
successor. The split is documentation-only.
