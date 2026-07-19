# ACT-LEAMAS-FACTORY-DIGEST-STAGED-STATUS-CLASSIFICATION01 — Close Report

## Status

PARTIAL — CORRECTION01 + CORRECTION02 + CORRECTION03 are documented
and committed (commits `5587810`, `aa6687f`, `656ee35`, `d31b3dd`,
`00314cf`, and `247cc76`). The parent ACT's original
four-added/one-modified defect and the core manifest/classification
change are implemented, focused-tested, and self-hosting-proven.
The corrector ACTs address the reviewer findings; CORRECTION03 in
particular closes REVIEW_MAP path escaping, evidence hash-scope
migration to `normalized_digest_v3_sections`, and the
copy-coverage contract text. Canonical full-tree `make factorize`
/ `make gate` verification remains blocked on the previously
documented ACTs
(`ACT-LEAMAS-FACTORY-FACTORIZE-RUNNER-FIXTURE01`,
`ACT-LEAMAS-FACTORY-DUPCODE-PERF-RATCHET01`) and is explicitly
out of scope for this ACT and its corrections.

## Files changed (parent + CORRECTION01 + CORRECTION02)

* `internal/factory/digest/git_status_parser.go` (new) — shared
  NUL-delimited parser for `git diff --name-status -z` records,
  including typed `ChangeKind` constants (A/M/D/T/U/X/B plus rename
  /copy with score dropped) and strict error-on-malformed
  semantics.
* `internal/factory/digest/git_status_parser_test.go` (new) —
  24-row table-driven parser unit tests covering all status
  letters, paths with spaces/tabs/newlines/Unicode/leading
  dashes, multiple records, empty input, and every malformation.
* `internal/factory/digest/path_escape.go` (new) — canonical
  `PathEscape` / `ParseEscapedPath` form (escapes backslash, tab,
  CR, LF, and every control byte); symmetric round-trip.
* `internal/factory/digest/file_operations.go` — `ChangedFile`
  now carries explicit `Kind` and `OldPath`; `GetStagedFiles`
  and `GetDirtyFiles` consume the shared parser. Adds
  `RenameSimilarityThreshold = 30` and `detectArgs()`.
* `internal/factory/digest/review_manifest.go` — `BuildManifest`
  and `BuildRangeManifest` keep raw paths in their semantic
  fields; `PathEscape` is invoked only inside `RenderManifest`.
  `BuildRangeManifest` retains `OldPath` for both renames AND
  copies.
* `internal/factory/digest/review_types.go` — `ReviewChangedFile`
  carries `Path` / `OldPath` plus the new `StatusTypeChanged`,
  `StatusUnknown`, `StatusBrokenPair`; `FileStats` carries
  `TypeChangedFiles`, `UnknownFiles`, `BrokenPairFiles` plus the
  existing buckets.
* `internal/factory/digest/review_stats.go` — `ComputeStats` and
  `RenderStats` track the new kinds and emit the canonical v3
  `CHANGESET_STATS` key order.
* `internal/factory/digest/contract.go` — `ContractVersion = 3`
  with documented v3 schema (status alphabet, key order, rendered
  filename escaping, compatibility expectations for v2
  consumers).
* `internal/factory/digest/contract_test.go` — v3 contract tests
  (`TestContractVersion_IsThree`, `TestRenderStats_V3CanonicalKeyOrder`,
  `TestRenderStats_V3IncludesNewFields`); updates every literal
  `version: 2` assertion to v3.
* `internal/factory/digest/contract_integration_test.go`,
  `internal/factory/digest/review_integration_test.go` — same
  v3 contract update.
* `internal/factory/digest/range_types.go` — `GetRangeFiles`
  uses the shared parser; `statusToHuman` handles every status
  letter, including `T`/`U`/`X`/`B`.
* `internal/factory/digest/review_stats_test.go` — v3 key order
  via `ContractStatsKeysV3`.
* `internal/factory/digest/digest_status_path_escape_test.go`
  (new) — table-driven `PathEscape` round-trip unit tests plus
  staged-mode and range-mode rendering integration tests that
  assert exact equality on manifest lines and verify raw paths
  do not appear in any rendered section.
* `internal/factory/digest/digest_status_staged_test.go` (new) —
  7-staged integration tests; reproduces the original defect
  with the exact `internal/factory/gate/gate.go` + 4 new files
  fixture; reconciles manifest and statistics against literal
  `git diff --cached --name-status -z --find-renames=30%
  --find-copies=30% HEAD --`.
* `internal/factory/digest/digest_status_dirty_test.go` (new) —
  9-dirty integration tests covering the ACT contract table plus
  determinism.
* `internal/factory/digest/digest_status_evidence_hashes_test.go`
  (new) — evidence-hash regression tests pinning the manifest,
  statistics, and aggregate digest evidence hashes to the
  corrected status.
* `internal/factory/digest/digest_status_range_test.go` (new) —
  range-mode regression tests using **exact-equality** assertions
  for `Addition` / `Modification` / `Deletion` / `Rename`, plus
  `MixedAllKinds` (covering all four kinds in one commit) and
  `TypeChange` (regular file → symlink). All four kinds are
  load-bearing; vacuous `for ... range` loops that would pass on
  empty renders are no longer used.
* `internal/factory/digest/digest_test_helpers_test.go` (new) —
  `RunGitForTest` / `RunGitWithExitCodeForTest` test helpers.
* `docs/acts/ACT-LEAMAS-FACTORY-DIGEST-STAGED-STATUS-CLASSIFICATION01.md`
  (parent ACT body updated to PARTIAL — CORRECTION01 + 02).
* `docs/acts/ACT-LEAMAS-FACTORY-DIGEST-STAGED-STATUS-CLASSIFICATION01-CORRECTION01.md`
  (new) — the corrector 01 ACT.
* `docs/acts/ACT-LEAMAS-FACTORY-DIGEST-STAGED-STATUS-CLASSIFICATION01-CORRECTION02.md`
  (new) — the corrector 02 ACT.
* `docs/factory/digest.md` — adds a "Status classification"
  section documenting the v3 semantics, the explicit 30%
  similarity policy, the `PathEscape` rendering contract, and
  the canonical `X` (unknown change type) / `B` (pairing broken)
  Git meanings.
* `docs/factory/digest-contract.md` — bumps the contract to v3
  with documented schema, status alphabet, stats key order, and
  compatibility expectations for v2 consumers.
* `docs/close-reports/ACT-LEAMAS-FACTORY-DIGEST-STAGED-STATUS-CLASSIFICATION01.md`
  (this file).

## Behavior changed

### Before the parent ACT

```text
A  internal/factory/gate/gate.go
```

with stats

```text
added_files=5
modified_files=0
```

(Digest inferred `A` from `Tracked && StagedPresent && !UnstagedPresent`,
which misclassified every modified-then-staged existing file as an
addition.)

### After CORRECTION02 (current)

```text
M  internal/factory/gate/gate.go
A  new_one.go
A  new_two.go
A  new_three.go
A  new_four.go
```

with stats (v3 contract):

```text
files_changed=5
added_files=4
modified_files=1
deleted_files=0
type_changed_files=0
renamed_files=0
copied_files=0
unmerged_files=0
unknown_files=0
broken_pair_files=0
untracked_files=0
binary_files=0
generated_files=0
test_files=0
doc_files=0
source_files=5
config_files=0
```

The manifest now agrees path-for-path with the lowered-threshold
Leamas oracle
`git diff --cached --name-status -z --find-renames=30%
--find-copies=30% HEAD --`. The contract header reads
`LEAMAS_TARGETED_DIGEST_CONTRACT_VERSION: 3`.

A staged file whose name contains an embedded newline (e.g.
`weird\nfile\nname.go`) now renders on a single manifest line
in the canonical escaped form (e.g. `weird\\nfile\\nname.go`).
Reviewers can recover the original filename with `ParseEscapedPath`.
The raw path is preserved in `ReviewChangedFile.Path` /
`ChangedFile.Path` so that `ComputeStats`'s generated / binary
/ source / test / doc / config classification still addresses the
on-disk path.

A regular-file → symlink change renders as `T  linked.go` with
`type_changed_files=1`, in staged, dirty, and range modes.

## Exact commands run (parent + 01 + 02)

| Command (with budget)                                    | Elapsed | Exit | Notes |
|----------------------------------------------------------|--------:|-----:|-------|
| `gofmt -w internal/factory/digest/*.go`                  | <0.1s   | 0    | reformatted |
| `go vet ./...`                                            | <2s     | 0    | clean |
| `go test ./internal/factory/digest -count=1`              | 3.5s    | 0    | full package, ~135 tests |
| `go test ./internal/factory/digest -count=5` (repeat)    | 11.4s   | 0    | 5 consecutive runs all green |
| `go test ./cmd/leamas -count=1`                            | 5.1s    | 0    | CLI wiring still works |
| `CGO_ENABLED=0 go build -trimpath -o bin/leamas ./cmd/leamas` | <3s  | 0    | 12,780,092 bytes, statically linked |
| `./bin/leamas factory verify llm-friendly`                | <1s     | 0    | "llm-friendly verification PASSED" |
| `./bin/leamas factory verify agent-context`               | <1s     | 0    | "agent-context verification PASSED" |
| `./bin/leamas factory verify forbidden-patterns`          | <1s     | 0    | "forbidden-patterns verification PASSED" |
| `git diff --check`                                         | <0.1s   | 0    | whitespace hygiene clean |
| `CGO_ENABLED=0 go build -trimpath -o /tmp/leamas-digest-correction02 ./cmd/leamas` | <3s | 0 | correction-02 self-hosting binary |
| `/tmp/leamas-digest-correction02 version`                | <0.1s   | 0    | reports `commit: d31b3dd872cc...` |
| `/tmp/leamas-digest-correction02 factory digest --range HEAD~1..HEAD --output ...` | 0.08s | 0 | self-hosting proof with current commit; v3 contract |
| `/tmp/leamas-digest-correction02 factory digest --staged --output ...` | 0.01s | 0 | empty staged; v3 contract version |
| `timeout 60 make factorize`                              | 60s     | 124 | Got past `agent-context`, `docs`, `doctrine`,
`doctrine-agent-contracts`, `domain-boundaries` (each
OK in 0.00s) before timing out on the heavier
duplicate-code phase. Same blocking previously documented. |
| `timeout 60 make gate`                                   | 60s     | 124 | Same blocking: gate re-runs the early OK
phases and then hangs on the live-tree
duplicate-code phase. Same blocking previously
documented. |
| `set -o pipefail; timeout 180 go test ./... 2>&1`              | 180s   | **rc=124** (timeout killed child) | 11 of 31 packages green (cmd/leamas, internal/execution,
internal/factory/{agentcontext, boundary, checks,
coverage, digest, docs, doctrine, doctrinecompiler}).
The remaining 20 packages (notably
internal/factory/{dupcode, gate, ...}) were not exercised.
The literal `rc=124` is recorded from the `timeout` signal. |

## Self-hosting proof (after CORRECTION02)

`/tmp/leamas-digest-correction02 version` reports
`commit: d31b3dd872cccd1101d20e8d3746aac034abcad2`, which matches
`git rev-parse HEAD`. The digest's range-mode proof for
`HEAD~1..HEAD` (the corrector 02 commit) shows the contract
header:

```text
LEAMAS_TARGETED_DIGEST_CONTRACT_VERSION: 3
LEAMAS_VERSION: 0.1.0+dev.d31b3dd872cc.20260719T083907Z
LEAMAS_COMMIT: d31b3dd872cccd1101d20e8d3746aac034abcad2
LEAMAS_BUILD_TIME: 2026-07-19T08:39:07Z
DIGEST_MODE: range
DIGEST_CREATED_AT: 2026-07-19T08:39:33Z
```

`CHANGESET_STATS` lists `type_changed_files=0`, `unknown_files=0`,
`broken_pair_files=0` — the new v3 keys are present in the
rendered output, not omitted.

The staged-mode proof for the current (empty) staging area
shows `CHANGESET_STATS` carrying the v3 keys with zero counts,
confirming the contract bump rather than the parent v2.

## Skipped / deferred checks

### Canonical full-tree verification

`make factorize` and `make gate` were each given a 60-second
budget and terminated by the timeout. Both got past the early OK
phases (`agent-context`, `docs`, `doctrine`,
`doctrine-agent-contracts`, `domain-boundaries`) and then hung
on the heavier live-tree duplicate-code phase. This is the same
blocking documented in prior ACTs and the corrector ACTs
explicitly mark it out of scope.

`go test ./...` was attempted with a 180-second budget and was
interrupted by the timeout signal. The bounded attempt completed
11 of 31 packages successfully; the remaining 20 packages
(notably `internal/factory/{dupcode, gate, ...}`) were not
exercised in this run. The literal `rc=124` from `timeout` is
recorded above as the bounded-attempt exit status.

### Evidence: behavior on baseline

* Before this ACT, `BuildManifest` used the boolean inference
  (`Tracked && StagedPresent && !UnstagedPresent` ⇒ `A`).
* After CORRECTION01, `BuildManifest` projects the explicit
  `Kind` populated from the structured `--name-status -z`
  parser; the presence flags are independent metadata for
  diff rendering.
* After CORRECTION02, semantic models (review types) keep raw
  paths; `PathEscape` is applied only at the rendering boundary,
  so `ComputeStats` and `BuildReviewMap` address the on-disk
  path.
* `TestBuildManifest_NoBooleanInference` (this ACT) asserts the
  inverse contract: a `ChangedFile` with presence flags but no
  `Kind` must not be classified as `A`/`M`. If the predicate ever
  re-appears, this test catches it.
* `TestRenderStats_V3CanonicalKeyOrder` asserts the v3 stats key
  sequence exactly; any reordering fails this test and is not
  permitted without bumping the contract version again.

## Follow-up ACTs

* `ACT-LEAMAS-FACTORY-FACTORIZE-RUNNER-FIXTURE01` —
  prerequisite for unblocking `make factorize`.
* `ACT-LEAMAS-FACTORY-DUPCODE-PERF-RATCHET01` — prerequisite for
  unblocking `make gate` and the heavier packages in
  `go test ./...`.

These remain blocked on the duplicate-code runtime and are
explicitly out of scope for this ACT and its corrections.
