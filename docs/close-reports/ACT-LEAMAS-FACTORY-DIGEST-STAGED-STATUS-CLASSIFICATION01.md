# ACT-LEAMAS-FACTORY-DIGEST-STAGED-STATUS-CLASSIFICATION01 — Close Report

## Status

PARTIAL. The original four-added/one-modified defect and the core
manifest/classification change are implemented, focused-tested, and
self-hosting-proven. A reviewer pass surfaced six P1/P2 contract
defects which are addressed by
[`ACT-LEAMAS-FACTORY-DIGEST-STAGED-STATUS-CLASSIFICATION01-CORRECTION01`](../acts/ACT-LEAMAS-FACTORY-DIGEST-STAGED-STATUS-CLASSIFICATION01-CORRECTION01.md),
also implemented and committed. Canonical full-tree
`make factorize` / `make gate` verification remains blocked on the
previously-documented duplicate-code ACTs and is explicitly out of
scope for this ACT and its correction.

## Files changed (parent ACT 01)

* `internal/factory/digest/git_status_parser.go` (new) — shared
  NUL-delimited parser for `git diff --name-status -z` records,
  including typed `ChangeKind` constants and strict error-on-malformed
  semantics.
* `internal/factory/digest/git_status_parser_test.go` (new) —
  24-row table-driven parser unit tests covering all status
  letters (A/M/D/T/U/X/B), renames and copies with various scores,
  paths with spaces/tabs/newlines/Unicode/leading dashes,
  multiple records, empty input, and every malformation.
* `internal/factory/digest/file_operations.go` — `ChangedFile`
  now carries explicit `Kind` and `OldPath`; `GetStagedFiles`
  and `GetDirtyFiles` consume the shared parser. Adds
  `RenameSimilarityThreshold = 30` and `detectArgs()`.
* `internal/factory/digest/review_manifest.go` — `BuildManifest`
  projects the explicit `Kind` rather than inferring from
  boolean presence flags; uses `PathEscape` for rendered paths.
* `internal/factory/digest/review_types.go` — `ReviewChangedFile`
  carries `Path` / `OldPath` plus the new `StatusTypeChanged`,
  `StatusUnknown`, `StatusBrokenPair`; `FileStats` carries
  `TypeChangedFiles`, `UnknownFiles`, `BrokenPairFiles` plus the
  existing buckets.
* `internal/factory/digest/review_stats.go` — `ComputeStats` and
  `RenderStats` track the new kinds and emit `type_changed_files`,
  `unknown_files`, `broken_pair_files`.
* `internal/factory/digest/file_evidence.go` — staged/unstaged
  presence rendering is preserved; "Changed files" carries the
  new `kind: <letter>` annotation; rendered paths go through
  `PathEscape`.
* `internal/factory/digest/range_types.go` — `GetRangeFiles` uses
  the shared parser; `statusToHuman` and `BuildRangeManifest`
  handle every status letter, including `T`/`U`/`X`/`B` which
  were previously silently mapped to "modified".
* `internal/factory/digest/review_test.go` — updated to set
  `Kind` explicitly. Existing status-detection test replaced by
  `TestBuildManifest_UsesExplicitKind` and
  `TestBuildManifest_NoBooleanInference`.
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
  statistics, and aggregate digest hashes to the corrected
  status.
* `internal/factory/digest/digest_status_range_test.go` (new) —
  range-mode regression tests using **exact-equality** assertions
  for `Addition` / `Modification` / `Deletion` / `Rename`, plus
  `MixedAllKinds` (covering all four kinds in one commit) and
  `TypeChange` (regular file → symlink). All four kinds are
  load-bearing; vacuous `for ... range` loops that would pass on
  empty renders are no longer used.
* `internal/factory/digest/digest_status_path_escape_test.go`
  (new) — table-driven `PathEscape` round-trip unit tests plus
  staged-mode and range-mode rendering integration tests
  covering paths with embedded newlines.
* `internal/factory/digest/path_escape.go` (new) — canonical
  `PathEscape` / `ParseEscapedPath` form, with `\n` / `\r` /
  `\t` / `\\` / `\xNN` escapes; symmetric.
* `internal/factory/digest/digest_test_helpers_test.go` (new) —
  `RunGitForTest` / `RunGitWithExitCodeForTest` test helpers.
* `docs/acts/ACT-LEAMAS-FACTORY-DIGEST-STAGED-STATUS-CLASSIFICATION01.md`
  (parent ACT, updated to PARTIAL in CORRECTION01).
* `docs/acts/ACT-LEAMAS-FACTORY-DIGEST-STAGED-STATUS-CLASSIFICATION01-CORRECTION01.md`
  (new) — the corrector ACT.
* `docs/factory/digest.md` — adds a "Status classification"
  section documenting the new semantics, the explicit 30%
  similarity policy, and the `PathEscape` rendering contract.
* `docs/close-reports/ACT-LEAMAS-FACTORY-DIGEST-STAGED-STATUS-CLASSIFICATION01.md`
  (this file; updated to PARTIAL after the corrector ACT
  landed).

## Behavior changed (after CORRECTION01)

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
which misclassifies every modified-then-staged existing file as an
addition.)

### After CORRECTION01

```text
M  internal/factory/gate/gate.go
A  new_one.go
A  new_two.go
A  new_three.go
A  new_four.go
```

with stats

```text
added_files=4
modified_files=1
type_changed_files=0
renamed_files=0
copied_files=0
untracked_files=0
unmerged_files=0
unknown_files=0
broken_pair_files=0
```

The manifest now agrees path-for-path with
`git diff --cached --name-status -z --find-renames=30%
--find-copies=30% HEAD --` (the lowered-threshold Leamas oracle).

A staged file whose name contains an embedded newline (e.g.
`weird\nfile\nname.go`) now renders on a single manifest line in
the canonical escaped form (e.g. `weird\\nfile\\nname.go`),
because every rendered path goes through `PathEscape`. Reviewers
can recover the original filename with `ParseEscapedPath`.

A regular-file → symlink change renders as `T  linked.go`,
end-to-end through staged, dirty, and range modes, instead of the
silently-incorrect `M  linked.go`.

## Exact commands run (parent ACT 01 + CORRECTION01)

| Command (with budget)                                    | Elapsed | Exit | Notes |
|----------------------------------------------------------|--------:|-----:|-------|
| `gofmt -w internal/factory/digest/*.go`                  | <0.1s   | 0    | reformatted |
| `go vet ./...`                                            | <2s     | 0    | clean across the CORRECTION01 changes too |
| `go test ./internal/factory/digest -count=1`              | 3.5s    | 0    | full package, ~135 tests |
| `go test ./cmd/leamas -count=1`                            | 5.1s    | 0    | CLI wiring still works |
| `CGO_ENABLED=0 go build -trimpath -o bin/leamas ./cmd/leamas` | <3s  | 0    | 12,780,092 bytes, statically linked |
| `./bin/leamas factory verify llm-friendly`                | <1s     | 0    | "llm-friendly verification PASSED" |
| `./bin/leamas factory verify agent-context`               | <1s     | 0    | "agent-context verification PASSED" |
| `./bin/leamas factory verify forbidden-patterns`          | <1s     | 0    | "forbidden-patterns verification PASSED" |
| `git diff --check`                                         | <0.1s   | 0    | whitespace hygiene clean |
| `CGO_ENABLED=0 go build -trimpath -o /tmp/leamas-digest-status ./cmd/leamas` | <3s | 0 | self-hosting binary |
| `/tmp/leamas-digest-status factory digest --staged --output /tmp/digest-status-proof.txt` | 0.17s | 0 | self-hosting staged digest |
| `/tmp/leamas-digest-status factory digest --range HEAD~1..HEAD --output /tmp/...` | 0.02s | 0 | self-hosting range digest, with `T` rendered on regular→symlink |
| `timeout 60 make factorize`                              | 60s     | 124 (terminated) | Got past `agent-context`, `docs`, `doctrine`,
`doctrine-agent-contracts`, `domain-boundaries` (each OK in
0.00s) before timing out on the heavier duplicate-code
phase. Same blocking previously documented. |
| `timeout 60 make gate`                                   | 60s     | 124 (terminated) | Same blocking: gate re-runs the early OK
phases and then hangs on the live-tree duplicate-code
phase. Same blocking previously documented. |
| `timeout 480 go test ./...`                              | ~470s before interrupt | (interrupted) | 11 of 31 packages green
(cmd/leamas, internal/execution,
internal/factory/{agentcontext, boundary, checks,
coverage, digest, docs, doctrine, doctrinecompiler}).
The remaining 20 packages (notably
internal/factory/{dupcode, gate, ...}) were not exercised. |

## Self-hosting proof (literal Oracle after CORRECTION01)

The staged ACT + CORRECTION01 changes, captured verbatim from
`git diff --cached --name-status -z --find-renames=30% --find-copies=30% HEAD --`,
are enumerated in the previous digest manifest output. Independent
status count from the oracle equals `added_files + modified_files =
N`, matching the digest's `CHANGESET_STATS` exactly.

The corrector ACT also runs the new symlink test in
`/tmp/test_digest` to confirm range mode renders `T  linked.go`
(not `M  linked.go`).

## Skipped / deferred checks

### Canonical full-tree verification

`make factorize` and `make gate` were each given a 60-second budget
and terminated by the timeout. Both got past the early OK phases
(`agent-context`, `docs`, `doctrine`, `doctrine-agent-contracts`,
`domain-boundaries`) and then hung on the heavier live-tree
duplicate-code phase. This is the same blocking documented in
prior ACTs; the parent ACT explicitly forbids starting those ACTs
(`Out of scope` section) and the corrector ACT records it the same
way.

`go test ./...` was attempted with a 480-second budget and was
interrupted after the early packages completed (11 of 31 green)
because the heavier dupcode tests do not finish in that budget on
this host. The parent ACT body required the bounded attempt; the
corrector ACT records the exit honestly. The remaining 20 packages
were not exercised in this run.

### Evidence: behavior on baseline

* Before this ACT, `BuildManifest` used the boolean inference
  (`Tracked && StagedPresent && !UnstagedPresent` ⇒ `A`).
* After CORRECTION01, `BuildManifest` projects the explicit
  `Kind` populated from the structured `--name-status -z`
  parser; the presence flags are independent metadata for
  diff rendering.
* `TestBuildManifest_NoBooleanInference` asserts the inverse
  contract: a `ChangedFile` with presence flags but no `Kind`
  must not be classified as `A`/`M`. If the predicate ever
  re-appears, this test catches it.

## Follow-up ACTs

* `ACT-LEAMAS-FACTORY-FACTORIZE-RUNNER-FIXTURE01` —
  prerequisite for unblocking `make factorize`.
* `ACT-LEAMAS-FACTORY-DUPCODE-PERF-RATCHET01` — prerequisite for
  unblocking `make gate` and the heavier packages in
  `go test ./...`.

These remain blocked on the duplicate-code runtime and are
explicitly out of scope for this ACT and the corrector ACT.
