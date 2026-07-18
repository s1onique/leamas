# ACT-LEAMAS-FACTORY-DUPCODE-V4-ALL-PAIRS-MATERIALIZATION-PERFORMANCE01-CORRECTION02-CORPUS-AND-EVIDENCE01

## Status: PASSED — guarded V4 algorithm fully reconciled; final evidence bound

The semantic corpus, persistent fuzz regressions, 30-second fuzz GREEN,
performance evidence, whitespace repair, lifecycle reconciliation, and
repository-wide gate passed. The final closure correction now replaces the
previously deferred race statement with the explicit one-test bounded race
alternative, records the literal full-package timeout without calling it a
pass, and corrects the gate-summary generator defect exposed after the stale
artifact was invalidated. No duplicate-detection behavior, corpus semantics,
fuzz wire format, benchmark fixture, baseline, or remediation target changed.

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

Final closure correction:

- `cmd/leamas/factory.go` — bind literal gate execution to summary writing;
- `cmd/leamas/factory_gate_run_test.go` — injected pass/fail and write-error contract;
- `internal/factory/gate/run_summary.go` — write one observed aggregate gate result;
- `internal/factory/gate/run_summary_test.go` — canonical field/count matrix;
- `internal/factory/gate/summary.go` — share the non-recursive artifact writer;
- this ACT, close report, and gate-summary Factory documentation — corrected
  race and evidence-generator acceptance wording.

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

Original corpus, fuzz, and benchmark evidence remains as recorded above. The
closure correction ran the required matrix against the staged patch:

```text
gofmt -l internal/factory/dupcode                    PASS (empty)
gofmt -l <changed Go files>                          PASS (empty)
git diff --check                                     PASS
git diff --cached --check                            PASS
go test ./internal/factory/dupcode                   PASS (142.951s)
go test ./internal/factory/gate ./cmd/leamas         PASS (80.782s / 3.644s)
go test ./...                                        PASS (all packages)
go vet ./...                                         PASS
CGO_ENABLED=0 go build ./...                         PASS
make factorize                                       PASS (15/15)
./bin/leamas factory verify dupcode-baseline         PASS (36.14s)
make gate                                            PASS
.factory/gate-summary.json                           regenerated 2026-07-18T02:07:42Z,
                                                      overall_status=pass
```

The focused gate-summary contract established semantic RED on missing
`WriteGateRunSummary` / `runFactoryGate`, then GREEN in 0.403s / 0.256s.
The literal full and bounded race outcomes are recorded separately below.

## Final race and evidence closure

The first literal complete command was given the required package timeout:

```text
go test -race -timeout=30m ./internal/factory/dupcode
  FAIL: package timeout after 1800.456s (real 1801.00s)
  last executing test:
    TestV4BaselineDelta_LiveTreeMatchesCommittedBaseline (4m42s)
  TestDebugBaselines had completed and printed baselinesEqual: EQUAL
  race diagnostics before timeout: 0
```

A three-minute verbose diagnostic independently identified
`TestDebugBaselines` inside its live-tree `CheckReport` scan. The permitted
bounded alternative omitted exactly that one test and no production code:

```text
go test -race -timeout=30m -skip='^TestDebugBaselines$' ./internal/factory/dupcode
  PASS (1666.860s; real 1667.70s; race diagnostics: 0)

go test -timeout=30m -run='^TestDebugBaselines$' ./internal/factory/dupcode
  PASS (39.122s; real 39.76s)

go test -race -run='^TestV4Alignment_' ./internal/factory/dupcode
  PASS (1.864s; real 2.27s)
```

This is a bounded alternative, not a successful literal full-package race
run. Every package test except exactly `TestDebugBaselines` remained under
race instrumentation, the excluded test passed without `-race`, and no
production file was tagged or excluded.

Both persistent corpus regressions passed as ordinary tests. Their SHA-256
values remain:

```text
3fc61698be2e22940717afed57b7f563e49a15201582e060ee1e212da4a30a70
8c14c01ed68fc29339f518ff8f9de3a669294b7947e8170496be3506bcb30b5b
```

The stale summary was removed at `2026-07-18T01:39:28Z`. The ensuing literal
`make gate` passed, but `.factory/gate-summary.json` remained missing. That
observed RED established the evidence-generator defect. Focused executable
contracts first failed because `WriteGateRunSummary` and `runFactoryGate` did
not exist, then passed after the smallest correction. A rebuilt binary and
literal `make gate` regenerated a canonical `pass` summary at
`2026-07-18T01:53:18Z`; final post-commit regeneration is detached evidence.

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

No check is silently deferred. The literal complete package race command
timed out and is not claimed as a pass; the explicitly permitted one-test
bounded alternative is the accepted package race evidence. The immediate
successor `ACT-LEAMAS-FACTORY-DUPCODE-SELF-HOSTED-REMEDIATION01` may begin only
after the closure commit and detached final evidence are complete.
