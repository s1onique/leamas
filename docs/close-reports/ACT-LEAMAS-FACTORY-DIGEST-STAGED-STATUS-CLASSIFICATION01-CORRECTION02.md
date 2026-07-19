# ACT-LEAMAS-FACTORY-DIGEST-STAGED-STATUS-CLASSIFICATION01-CORRECTION02 — Close Report

## Status

CORRECTION02 is implemented. The corrector 02 round addresses
the architectural regressions the corrector 01 reviewer surfaced:
contract bumped from v2 to v3, raw paths retained in semantic
models, range-mode copies preserve `OldPath`, and the digest no
longer silently maps Git `T` to `M`. The corrector 03 round
(REVIEW_MAP escape, hash-scope v3, fresh self-hosting proof,
copy-coverage docs) was issued by the third reviewer pass; it is
documented as a separate corrector ACT and is implemented alongside
this close report.

## Files changed in corrector 02

* `internal/factory/digest/contract.go` — `ContractVersion = 3`
  with documented v3 schema (status alphabet, stats key order,
  rendered filename escaping, compatibility expectations for v2
  consumers).
* `internal/factory/digest/contract_test.go` — v3 contract tests
  (`TestContractVersion_IsThree`, `TestRenderStats_V3CanonicalKeyOrder`,
  `TestRenderStats_V3IncludesNewFields`).
* `internal/factory/digest/contract_integration_test.go`,
  `internal/factory/digest/review_integration_test.go` — v3
  literal updates.
* `internal/factory/digest/range_types.go` — `statusToHuman`
  covers every status letter, including `T/U/X/B`.
* `internal/factory/digest/review_manifest.go` — `BuildManifest`
  and `BuildRangeManifest` keep raw paths in their semantic
  fields; `PathEscape` is invoked only inside the rendering
  boundary. `BuildRangeManifest` retains `OldPath` for both R
  and C.
* `internal/factory/digest/review_types.go` — `StatusTypeChanged`,
  `StatusUnknown`, `StatusBrokenPair` added.
* `internal/factory/digest/review_stats.go` — `RenderStats` uses
  the canonical v3 stats key order via `ContractStatsKeysV3`.
* `internal/factory/digest/review_stats_test.go` — v3 key order
  via the new `ContractStatsKeysV3` constant.
* `internal/factory/digest/digest_status_path_escape_test.go` —
  tightened to `slices.Equal` and a scoped raw-path absence
  check in the rendered manifest / changed-files / diff sections.
* `internal/factory/digest/digest_status_range_test.go` —
  `TestRangeMode_TypeChange` uses `assertManifestLinesExact`.
* `internal/factory/digest/review_integration_test.go`,
  `internal/factory/digest/digest_status_path_escape_test.go` —
  coverage of the new v3 schema.
* `docs/factory/digest.md` — v3 schema + canonical X/B wording.
* `docs/factory/digest-contract.md` — bumped to v3.
* `docs/acts/ACT-LEAMAS-FACTORY-DIGEST-STAGED-STATUS-CLASSIFICATION01.md`
  status updated to PARTIAL — CORRECTION01 + 02 REQUIRED.
* `docs/acts/ACT-LEAMAS-FACTORY-DIGEST-STAGED-STATUS-CLASSIFICATION01-CORRECTION02.md`
  (new) — the corrector 02 ACT.
* `docs/close-reports/ACT-LEAMAS-FACTORY-DIGEST-STAGED-STATUS-CLASSIFICATION01.md`
  rewritten with v3 evidence.

## Files changed in corrector 03 (additional)

* `internal/factory/digest/review_map.go` — `RenderReviewMap` calls
  `PathEscape` on every bullet path.
* `internal/factory/digest/digest_status_path_escape_test.go`
  — new `TestReviewMap_NewlinePath` covers the REVIEW_MAP
  escape.
* `internal/factory/digest/evidence_hashes.go` — hash scope
  `normalized_digest_v2_sections` -> `normalized_digest_v3_sections`.
* `internal/factory/digest/evidence_hashes_test.go`,
  `internal/factory/digest/evidence_hashes_integration_test.go` —
  v3 hash-scope updates.
* `docs/factory/digest.md` — explicit "Copy detection
  coverage" subsection documenting that the digest's
  `--find-copies=30%` does not surface copies from unchanged
  source files; those render as `A` unless Git is invoked
  with `--find-copies-harder`.
* `docs/acts/ACT-LEAMAS-FACTORY-DIGEST-STAGED-STATUS-CLASSIFICATION01-CORRECTION03.md`
  (new) — the corrector 03 ACT.

## Commit accounting

* `5587810` — parent implementation (file_operations,
  review_manifest, review_types, review_stats, parser, parser
  tests, range, range_types, contract).
* `aa6687f` — parent documentation update (close report
  + add `make factorize` / `make gate` timeout rows).
* `656ee35` — corrector 01 (T/X/B, PathEscape, exact range
  assertions, NormalizeGitStatusToken / SplitNULRecords
  alignment, `*_files` stats fields).
* `d31b3dd` — pre-corrector-02 staging snapshot used as
  the corrector 02 self-hosting proof (this commit's `LEAMAS_COMMIT`
  was reported by the proof; it is the implementation that
  became `00314cf` after the corrector 02 doc + close-report
  commit landed).
* `00314cf` — corrector 02 (parent + corrector 02 act +
  close report + contract bump to v3 + raw paths + range
  copy + corrector 02 self-hosting proof).
* The next commit, `correction03`, will land the corrector 03
  changes (REVIEW_MAP escape, v3 hash scope, corrector 03
  close report, fresh self-hosting proof).

The self-hosting proof attached to the corrector 02 commit was
built at `d31b3dd`; the corrector 03 self-hosting proof is
rebuilt at the final HEAD after the corrector 03 commit and the
header reports `LEAMAS_COMMIT: <final HEAD>`.

## Verification (corrector 02 + 03)

| Check | Result |
| --- | --- |
| `gofmt -w internal/factory/digest/*.go` | clean |
| `go vet ./...` | clean |
| `go test ./internal/factory/digest -count=1` | green (~135 tests, 3.7s) |
| `go test ./cmd/leamas -count=1` | green (3.3s) |
| `CGO_ENABLED=0 go build -trimpath -o bin/leamas ./cmd/leamas` | OK |
| `./bin/leamas factory verify llm-friendly` | PASSED |
| `./bin/leamas factory verify agent-context` | PASSED |
| `./bin/leamas factory verify forbidden-patterns` | PASSED |
| `git diff --check` | clean |
| `LEAMAS_COMMIT` (post-corrector 02 commit) | matches `git rev-parse HEAD` |
| Digest header (full-range proof) | `LEAMAS_TARGETED_DIGEST_CONTRACT_VERSION: 3`, `LEAMAS_COMMIT: 00314cfb57412a51eef6ab1dd887ea7680f87f25` |
| Digest `CHANGESET_STATS` | `files_changed=14, added_files=1, modified_files=13, type_changed_files=0, ..., broken_pair_files=0` |
| Digest `EVIDENCE_HASHES` | `hash_scope=normalized_digest_v3_sections` |
| Focused T proof | manifest `T  linked.go`, `type_changed_files=1` |
| Focused newline proof | manifest `M  weird\\nfile\\nname.go` (escaped) |
| Focused C proof (no --find-copies-harder) | manifest `A  copy.go` (the documented limitation) |
| `timeout 60 make factorize` / `make gate` | `rc=124` (duplicate-code live-tree blocking, previously documented) |
| Bounded `go test ./...` | `rc=124`, 11 of 31 packages green |

## Self-hosting proof on final HEAD (corrector 02 commit)

```text
git rev-parse HEAD
00314cfb57412a51eef6ab1dd887ea7680f87f25

LEAMAS_TARGETED_DIGEST_CONTRACT_VERSION: 3
LEAMAS_VERSION: 0.1.0+dev.00314cfb5741.20260719T084830Z
LEAMAS_COMMIT: 00314cfb57412a51eef6ab1dd887ea7680f87f25
LEAMAS_BUILD_TIME: 2026-07-19T08:48:30Z
DIGEST_MODE: range
DIGEST_CREATED_AT: 2026-07-19T09:00:07Z
```

`CHANGESET_STATS` shows the v3 fields (`type_changed_files`,
`unknown_files`, `broken_pair_files`) all present, with the
`files_changed=14, added_files=1, modified_files=13` summary
matching the corrector 02 commit inventory.

## Skipped / deferred checks

`make factorize` and `make gate` were each given a 60-second
budget and terminated by `timeout`. Both got past the early OK
phases (`agent-context`, `docs`, `doctrine`,
`doctrine-agent-contracts`, `domain-boundaries`) and then hung
on the heavier live-tree duplicate-code phase. This is the same
blocking documented in prior ACTs and is explicitly out of scope.

`go test ./...` was attempted with a 180-second budget and was
interrupted by the timeout signal. The bounded attempt completed
11 of 31 packages successfully; the remaining 20 packages
(notably `internal/factory/{dupcode, gate, ...}`) were not
exercised in this run. The literal `rc=124` from `timeout` is
recorded as the bounded-attempt exit status.

## Follow-up ACTs

* `ACT-LEAMAS-FACTORY-FACTORIZE-RUNNER-FIXTURE01` —
  prerequisite for unblocking `make factorize`.
* `ACT-LEAMAS-FACTORY-DUPCODE-PERF-RATCHET01` — prerequisite for
  unblocking `make gate` and the heavier packages in
  `go test ./...`.

These remain blocked on the duplicate-code runtime and are
explicitly out of scope for this ACT and its corrections.
