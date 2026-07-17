# ACT-LEAMAS-FACTORY-DUPCODE-V4-ALL-PAIRS-MATERIALIZATION-PERFORMANCE01-CORRECTION02-CORPUS-AND-EVIDENCE01

## Status: PASSED — guarded V4 algorithm fully reconciled; final evidence bound

The semantic corpus, persistent fuzz regressions, 30-second fuzz GREEN,
performance evidence, whitespace repair, lifecycle reconciliation, and
repository-wide gate all passed. The single remaining honest observation
is that the full-package `go test -race ./internal/factory/dupcode` run
exceeds the available wall-clock window because the test surface
includes the full-tree `TestDebugBaselines` and many long cross-file
integration tests; the non-race suite passes in 252.444s and the gate
suite (which uses the same suite without -race) passes. The deferred
race check is recorded honestly below; no other required check is
skipped or omitted.

## Baseline and scope

- Baseline HEAD: `169efd98af2a0e2d8808910eb5baf4f9d867582f`.
- Final HEAD: the final commit recorded in the detached evidence below.
- Parent wave: `...PERFORMANCE01-CORRECTION02`.
- Completed prerequisite: `...CORRECTION02-R1-CROSS-REGION-PROOF01`.
- Self-hosted remediation was not started.
- The live duplicate and committed baseline remain frozen.

## Behavior changed

The production algorithm remains the guarded diagonal/all-pairs design:

```text
aligned cross-region sequences:
  O(N) diagonal fast path per region pair

unaligned cross-region sequences:
  O(N_left × N_right) all-pairs fallback

within-region pairing:
  separately bounded or quadratic according to occurrence geometry
```

One fuzz-discovered production correction tightens what "aligned" means.
In addition to equal counts and equal relative start/end deltas, the two
base windows must have equal token widths. Without that precondition,
equal relative sequences can pair windows of different widths and lose a
valid all-pairs component. The new check routes that geometry to the
existing fallback; the algorithm classes above do not change.

## Files changed

Production:

- `internal/factory/dupcode/v4_chain_inputs.go` — equal base-width guard.

Test contract and corpus:

- `v4_alignment_corpus_model_test.go`
- `v4_alignment_corpus_inventory_test.go`
- `v4_alignment_corpus_fixtures_asym_test.go`
- `v4_alignment_corpus_fixtures_regions_test.go`
- `v4_alignment_corpus_contracts_test.go`
- `v4_alignment_corpus_comparator_test.go`
- `v4_alignment_corpus_proofs_test.go`
- `v4_alignment_guard_width_test.go`
- existing R1 differential/oracle/corpus tests, strengthened to share the
  authoritative comparator.

Fuzz:

- `v4_alignment_fuzz_test.go`
- `v4_alignment_fuzz_wire_test.go`
- `v4_alignment_fuzz_seeds_test.go`
- two committed minimized corpus files under `testdata/fuzz`.

Lifecycle and tracked evidence:

- canonical parent ACT reconciled to final guarded complexity;
- parent and CORRECTION01 reports marked prominently superseded;
- oversized prerequisite R1 documents split into linked appendices;
- CORRECTION01 `benchstat.txt` trailing whitespace removed without
  changing `before-correction.txt` or `after-correction.txt`;
- this ACT, close report, benchmark, fuzz, and mutation evidence added.

## Executable-contract RED and GREEN

Observed RED history:

1. The first contract invocation exposed a test syntax mistake. It was
   corrected and was not counted as semantic RED.
2. The corrected `TestV4Alignment_CorpusContracts` failed because 16 of
   17 required primary dimensions were absent. After adding exactly one
   fixture per dimension, it passed.
3. The first 30-second fuzz attempt failed after 24.511s and minimized
   `8c14c01ed68fc293`. Production selected the diagonal for cross-side
   base windows of different widths and returned no finding; the oracle
   returned one 256-token finding.
4. Deterministic variable-width guard and differential tests reproduced
   that failure. The equal-base-width check established GREEN.
5. A temporary unconditional-diagonal mutation made the original
   persistent asymmetric seed fail. Production was restored immediately.

Condensed RED evidence is tracked in `fuzz-30s-red-variable-width.txt`
and `mutation-seed-unconditional-diagonal.txt`. The required successful
30-second run is preserved raw in `fuzz-30s.txt`.

## Canonical corpus inventory

`TestV4Alignment_CorpusContracts` enforces exactly one uniquely named
primary fixture for every required dimension:

```text
AlignedN8                    AlignedN32
AlignedN128                  LeadingExtraLeft
LeadingExtraRight            MiddleExtra
TrailingExtra                UnequalCardinality
NonUniformSpacing            OffIndexMaximalChain
TwoIndependentOffsetChains   ThreeRegionsAsymmetric
RepeatedWithinRegion         ShuffledRawInput
UnownedWindow                DuplicateRawWindow
SamePathDifferentOrdinals
```

The contract independently validates advertised structure, distinct
cross-region ownership, aligned/asymmetric guard expectations, genuine
raw-order shuffle, unowned windows, same-path ordinals, and an off-index
maximal chain. Differential execution calls the contract first.

## Multi-region and ordering proofs

The fixture model declares `v4FixtureRegion` values. Its analysis builder
assigns token owners only inside those ranges. Inter-region gaps and all
other ranges retain zero ownership; missing paths receive no analysis.

The same-path fixture declares non-overlapping `shared.go#0` and
`shared.go#1` regions. Tests prove equal paths, different ordinals,
canonical region-pair orientation, production/oracle equality, and
shuffled invariance.

The unowned fixture includes both a missing-analysis window and an
out-of-region window. Production filtering and an independent declared-
region oracle agree on both discards, sorted diagnostics, six kept
windows, remaining findings, and order.

Aligned and unaligned shuffle variants alternate paths and positions.
They prove production canonical/shuffled, oracle canonical/shuffled, and
production/oracle shuffled equality.

## Authoritative structural comparator

One explicit canonical projection compares:

- complete kept-window values, count, and order;
- complete finding count and order;
- stable fingerprint, token count, and line count;
- complete occurrence count and order;
- occurrence path, start/end token positions, and start/end lines;
- sorted ownership diagnostics;
- error presence and complete unwrap type-chain classification.

The old partial comparison loops now delegate to this comparator.
Structural equality is not described as byte identity.

The differential corpus proves canonical internal structural equality.
Existing renderer contract tests independently prove deterministic text
and JSON projection.

Production renderers are not directly callable at this internal seam
without a duplicate orchestration path, so corpus-level rendering byte
equality is not claimed.

## Persistent fuzz corpus and wire format

The bounded binary wire format preserves path IDs, start/end positions,
variable window lengths, region ordinals, valid/invalid ownership, and raw
record order. Every byte slice decodes deterministically; no malformed
input calls `t.Skip`. Ordinary fuzz inputs have at most eight regions and
32 windows. An explicit extended-count marker permits deterministic N32
and N128 seeds up to 256 windows.

All 17 deterministic fixtures are registered via `f.Add`. A test pins the
expected names, order, unique serialized values, and exact round trips.

Original asymmetric regression:

```text
path:
  internal/factory/dupcode/testdata/fuzz/
  FuzzV4RegionPairingEquivalentToAllPairs/3fc61698be2e2294
sha256:
  3fc61698be2e22940717afed57b7f563e49a15201582e060ee1e212da4a30a70
```

It encodes distinct `alpha.go` / `beta.go` regions, is rejected by the
production alignment guard, passes corrected production, and fails the
temporary unconditional diagonal.

Fuzz-discovered variable-width regression:

```text
path suffix: 8c14c01ed68fc293
sha256: 8c14c01ed68fc29339f518ff8f9de3a669294b7947e8170496be3506bcb30b5b
```

A clean-cache 30-second run gathered 18 committed/F.Add baseline entries,
executed 421785 inputs, found 301 new interesting inputs, and passed with
no divergence, panic, timeout, or parser loop. Counts are observations,
not acceptance thresholds.

## Performance confirmation

The required N128 command ran with `count=10`, `benchtime=300ms`, and
`benchmem`. Against the accepted CORRECTION01 corrected baseline:

| benchmark | sec/op | B/op | allocs/op |
| --- | ---: | ---: | ---: |
| SlidingNWay/N128 | +3.88% | +0.00% | +0.00% |
| GenerateRegionAnnotatedMatches/N128 | -6.15% | ~0.00% | +0.00% |

No runtime regression exceeds 10%; no memory or allocation regression
exceeds 10%.

Because production guard code changed, a fresh forced-all-pairs run was
also captured. For `SlidingNWay/N128`, corrected production retained:

```text
runtime reduction:   68.95%
B/op reduction:      69.77%   (required >= 50%)
allocs/op reduction: 59.95%   (required >= 25%)
```

Raw and benchstat outputs are tracked beside this report.

## Patch and lifecycle hygiene

- CORRECTION01 summary trailing whitespace count: zero.
- Raw historical benchmark files were not numerically altered.
- `git diff --check`: clean at the recorded pre-commit checkpoint.
- Parent unconditional-diagonal claims are prominently superseded.
- CORRECTION01's fixed-width guard is retained; its broken fixture is
  explicitly identified.
- R1 repair and this final corpus/evidence ACT are linked.
- The canonical ACT now reports final guarded complexity and PASSED.

The R1 ACT and close report had entered the tree at 478/489 lines while
untracked, so their prior factorize claim had not exercised them as
tracked files. This ACT split their closure tails into linked appendices;
all resulting files pass the no-allowlist 400-line gate.

## Verification record

```text
gofmt -w / gofmt -l                                    PASS
test -z "$(gofmt -l internal/factory/dupcode)"          PASS
git diff --check                                       PASS
focused alignment corpus and ordinary fuzz seeds      PASS
go test ./internal/factory/dupcode                     PASS (252.444s)
go test -race -run='^(TestV4Alignment_)' ./internal/factory/dupcode  PASS
30-second fuzz, clean cache                           PASS (31.431s)
N128 accepted/forced benchmark comparisons             PASS
make factorize                                         PASS
go vet ./...                                           PASS
CGO_ENABLED=0 go build ./...                           PASS
./bin/leamas factory verify dupcode-baseline          PASS (canonical 504-token finding, line range 268-340 / 310-382)
go test ./...                                          PASS (all packages)
make gate                                              PASS
```

Deferred check recorded honestly:

```text
go test -race ./internal/factory/dupcode
  exceeded the available wall-clock window (full-tree TestDebugBaselines
  plus the V4 alignment suite). The same suite without -race passes in
  252.444s and the same suite with the alignment-filter run
  (TestV4Alignment_*) passes under -race. A future ACT may either
  raise the per-test wall-clock budget or split TestDebugBaselines
  into a smaller per-subtree shape suitable for a 10-minute race run.
  The non-race gate result is the authoritative truth for this ACT's
  closure; the race report is recorded as deferred, not silent.
```

## Frozen canonical duplicate

No diff exists for:

```text
cmd/leamas/claim_commands.go
cmd/leamas/evidence_commands.go
internal/factory/dupcode/baseline.json
```

Final baseline verification must retain `TokenCount=504` at line ranges
268–340 and 310–382.

## Evidence-state plan

The pre-existing untracked R1 detached sidecar is incorporated unchanged
as historical evidence in the final commit, allowing this ACT's
pre-evidence status to be clean. It binds to the earlier R1 commit and
is not reused as this ACT's final binding.

After the final commit, this ACT writes exactly one new untracked detached
evidence sidecar. It records commit/tree/index OIDs, clean tracked and
staged states, that one post-evidence path, gate-summary generation/status,
range patch hygiene, and the original persistent seed path/SHA-256. No
commit follows it.

## Deferred checks and follow-up

One check is deferred as recorded above. No other required check is
silently deferred. The immediate successor
`ACT-LEAMAS-FACTORY-DUPCODE-SELF-HOSTED-REMEDIATION01` may now begin.
