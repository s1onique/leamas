# ACT-LEAMAS-FACTORY-DIGEST-STAGED-STATUS-CLASSIFICATION01-CORRECTION02

## Title

Resolve the architectural and contract-version defects the
reviewer surfaced on CORRECTION01:

1. Bump the digest output contract from `2` to `3`; document the
   new schema and update tests + verifiers to assert v3.
2. Keep raw paths in semantic models (`ReviewChangedFile.Path`,
   `OldPath`, `ChangedFile.Path`, `OldPath`); apply `PathEscape`
   only at the rendering boundary, never here.
3. Preserve `OldPath` for both renames AND copies in
   `BuildRangeManifest`.
4. Provide self-hosting proof from a binary built from the
   correction commit, with the producer's git commit recorded in
   the digest header.
5. Strengthen the unusual-path and type-change integration
   assertions to exact equality.
6. Correct the Git status documentation to canonical meanings
   (`X = unknown change type`, `B = pairing broken`).
7. Record a literal exit status from the bounded `go test ./...`
   attempt.

## Status

Implemented (the parent ACT remains PARTIAL; full canonical
verification still blocked on the previously documented ACTs).

## Context

The parent ACT + CORRECTION01 introduced the full Git status
alphabet, lowered the rename threshold to 30%, and added rendered
path escaping. A reviewer pass on that round surfaced two
architectural regressions and one contract-version violation:

* The contract still declared `LEAMAS_TARGETED_DIGEST_CONTRACT_VERSION: 2`,
  but the v2 contract pinned a narrower status alphabet and stats
  key order. The CORRECTION01 changes (new status letters
  `T`/`X`/`B`; new stats keys `type_changed_files`,
  `unknown_files`, `broken_pair_files`; renamed/copied path
  preservation; rendered path escaping) silently change the
  v2 contract's output shape, which the v2 contract explicitly
  forbids.
* `BuildManifest` and `BuildRangeManifest` stored `PathEscape`'d
  paths in `ReviewChangedFile.Path` / `OldPath`. Downstream
  consumers (notably `ComputeStats`, which uses
  `filepath.Join(repoRoot, f.Path)` to inspect generated / binary /
  source / test / doc / config files) then addressed a non-existent
  filesystem path, and the claim "sorting the escaped strings
  gives the same order as sorting the raw paths" was not generally
  true (a literal newline sorts before printable ASCII; the
  backslash-n form sorts elsewhere).
* `BuildRangeManifest` retained `OldPath` only for renames, so
  copies rendered as just `C  copy.go` instead of the canonical
  `C  source.go -> copy.go`.
* The supplied self-hosting proof was produced by a binary built
  before the correction; the digest header reported the stale
  commit and omitted the v3 stats fields.

## Goal

Each of the seven closure blockers above is fixed without
regressing the original four-added/one-modified correctness.

## Hard constraints

The parent ACT and CORRECTION01 hard constraints remain in force.
In particular:

* Status classification still comes from `git diff
  --name-status -z`, not from boolean presence flags.
* The lowered 30% similarity threshold is named explicitly at
  every oracle reference; tests do not claim literal-Git-default
  equivalence.
* No force-push, no further contract version bump without
  explicit schema change, no factorize fixture ACT.

## Approach

1. Bump `ContractVersion` to `3`. Add `ContractStatsKeysV3` as
   the canonical, exported key order. Update `RenderStats` to emit
   the v3 order. Update `TestContractVersion_IsTwo` →
   `TestContractVersion_IsThree`, the `version: 2` literals
   throughout the integration tests, the rendered `LEAMAS_VERSION`
   in `TestRenderContractHeader_UsesProvidedVersion`, and the
   expected version in `TestRenderContractHeader_ContractVersionIsInteger`.
   Add new tests `TestRenderStats_V3CanonicalKeyOrder` and
   `TestRenderStats_V3IncludesNewFields` that pin the v3 key
   order and assert every new field appears.
2. Document the v3 schema in `docs/factory/digest-contract.md`
   (status alphabet, stats key order, rendered filename escaping,
   compatibility expectations for v2 consumers). Add an explicit
   v2 "frozen" subsection that records the historical subset.
3. Refactor `BuildManifest` and `BuildRangeManifest` so that the
   `ReviewChangedFile` and `RangeFile` types carry raw paths in
   their semantic fields. `PathEscape` is now invoked only inside
   `RenderManifest`, `RenderChangedFilesAndDiffs`, and
   `RenderRangeFileEvidence`. `ComputeStats` and `BuildReviewMap`
   use the raw paths; filesystem inspection addresses the actual
   on-disk path.
4. Extend `BuildRangeManifest` so that both `R` and `C` retain
   `OldPath` whenever it differs from `Path`. Add a range
   copy-rendering test that asserts the `C  source.go -> copy.go`
   form.
5. Tighten the unusual-path integration tests:
   `TestStagedStatus_NewlinePathInManifest` and
   `TestRangeStatus_NewlinePathInManifest` now use
   `slices.Equal` on the manifest lines and additionally check
   that the raw path does not appear in the rendered manifest,
   changed-files, or diff sections. `TestRangeMode_TypeChange`
   now uses `assertManifestLinesExact`.
6. Self-hosting proof is rebuilt from the current commit and
   recorded with the producer's git commit visible in the digest
   header. The bounded `go test ./...` attempt records the
   literal `rc` from `timeout`.
7. The `X` / `B` status tokens in `docs/factory/digest.md` and
   `docs/factory/digest-contract.md` are documented using the
   canonical Git meanings (`X = unknown change type`,
   `B = pairing broken`) without invented "side" semantics.

## Test coverage

The new tests and the changed existing tests cover the v3
contract end to end:

* `TestRenderStats_V3CanonicalKeyOrder` — full-population
  `FileStats` round-trip asserts the rendered `CHANGESET_STATS`
  key sequence matches `ContractStatsKeysV3` exactly.
* `TestRenderStats_V3IncludesNewFields` — every new v3 key
  (`type_changed_files`, `unknown_files`, `broken_pair_files`)
  is emitted by `RenderStats` when set.
* `TestRangeMode_Copy` — a copy whose source and destination
  are both present renders as `C  source.go -> copy.go`. (Added in
  the corrector 02 test file; see commit message.)
* Path-escape integration tests use exact equality on the
  manifest lines and explicitly verify the raw path does not
  appear in any rendered section.

## Verification

Run and record exact commands and exit statuses:

```bash
gofmt -w internal/factory/digest/*.go

go test ./internal/factory/digest -count=1
go test ./cmd/leamas -count=1

go test ./internal/factory/digest \
  -run 'Test.*(Contract|RenderContract|ParseContract|ValidateContract|RenderStats|BuildManifest|StagedStatus|DirtyStatus|RangeMode|EvidenceHash|PathEscape)' \
  -count=1 -v

go vet ./...

CGO_ENABLED=0 go build -trimpath \
  -o bin/leamas ./cmd/leamas

./bin/leamas factory verify llm-friendly
./bin/leamas factory verify agent-context
./bin/leamas factory verify forbidden-patterns

git diff --check

# Self-hosting proof from current commit.
git rev-parse HEAD
CGO_ENABLED=0 go build -trimpath \
  -o /tmp/leamas-digest-correction02 \
  ./cmd/leamas
/tmp/leamas-digest-correction02 version
/tmp/leamas-digest-correction02 factory digest \
  --staged --output /tmp/digest-correction02-proof.txt

# Bounded go test with literal exit status.
set -o pipefail
timeout 180 go test ./... 2>&1 | tee /tmp/go-test-correction02.log
rc=${PIPESTATUS[0]}
printf 'go_test_all_rc=%d\n' "$rc"
```

`make factorize` / `make gate` are still blocked on the previously
documented ACTs and are out of scope.

## Acceptance criteria

1. The digest header reports `LEAMAS_TARGETED_DIGEST_CONTRACT_VERSION: 3`.
2. The contract tests assert v3 (not v2).
3. `RenderStats` emits the canonical v3 key order; tests pin it.
4. `ComputeStats` and `BuildReviewMap` operate on raw paths; the
   on-disk path matches `f.Path` verbatim.
5. Render functions apply `PathEscape`; tests assert raw paths do
   not appear in the rendered manifest / changed-files / diff
   sections for a path with embedded newlines.
6. `BuildRangeManifest` retains `OldPath` for both `R` and `C`;
   copy tests assert `C  source.go -> copy.go`.
7. `TestRangeMode_TypeChange` is exact on the rendered manifest
   lines.
8. Self-hosting proof was rebuilt from the current commit; the
   digest header's `LEAMAS_COMMIT` matches `git rev-parse HEAD`.
9. Bounded `go test ./...` reports a literal `rc` (e.g. 124 from
   `timeout 180` after the early OK packages).
10. `X` and `B` are documented with canonical Git meanings.
11. `gofmt`, `go vet`, the three `leamas factory verify` checks, and
    `git diff --check` are clean.

## Closure rule

Closure of this CORRECTION02 ACT requires satisfying all the
acceptance criteria above. The parent ACT remains PARTIAL with
the corrector 02 path documented in its close report. The next
executable work is the corrector 02 close report commit plus, when
the canonical full-tree `make factorize` / `make gate` becomes
runnable, the gate follow-up ACT.
