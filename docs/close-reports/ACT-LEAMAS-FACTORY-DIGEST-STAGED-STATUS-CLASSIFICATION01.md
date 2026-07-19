# ACT-LEAMAS-FACTORY-DIGEST-STAGED-STATUS-CLASSIFICATION01 — Close Report

## Status

CLOSED (full focused scope green; canonical full-tree verification
remains independently blocked on the previously documented ACTs
`ACT-LEAMAS-FACTORY-FACTORIZE-RUNNER-FIXTURE01` and
`ACT-LEAMAS-FACTORY-DUPCODE-PERF-RATCHET01`. See "Deferred
verification" below.)

## Files changed

* `internal/factory/digest/git_status_parser.go` (new) — shared
  NUL-delimited parser for `git diff --name-status -z` records,
  including typed `ChangeKind` constants and error-on-malformed
  semantics.
* `internal/factory/digest/git_status_parser_test.go` (new) —
  table-driven parser unit tests covering all 20 cases the ACT
  specifies.
* `internal/factory/digest/file_operations.go` — `ChangedFile`
  now carries explicit `Kind` and `OldPath`; `GetStagedFiles`
  and `GetDirtyFiles` consume the shared parser. Adds
  `RenameSimilarityThreshold = 30` and `detectArgs()`.
* `internal/factory/digest/review_manifest.go` — `BuildManifest`
  now projects the explicit `Kind` from `ChangedFile` rather than
  inferring from `Tracked`/`StagedPresent`/`UnstagedPresent`.
* `internal/factory/digest/file_evidence.go` — staged/unstaged
  presence rendering is preserved; the "Changed files" section now
  carries the new `kind: <letter>` annotation.
* `internal/factory/digest/range_types.go` — `GetRangeFiles` now
  uses the shared parser, which also fixes a pre-existing index
  bug where `R` renames were reported without the `old -> new`
  half.
* `internal/factory/digest/review_test.go` — updated to set
  `Kind` explicitly. Existing status-detection test replaced by
  `TestBuildManifest_UsesExplicitKind` and
  `TestBuildManifest_NoBooleanInference`, the latter locking the
  contract that a tracked file with presence flags but no Kind
  must not be classified as `A`/`M`.
* `internal/factory/digest/digest_status_staged_test.go` (new) —
  7-staged integration matrix; reproduces the original defect
  with the exact `internal/factory/gate/gate.go` + 4 new files
  fixture; reconciles manifest and statistics against literal
  `git diff --cached --name-status -z ... HEAD --`.
* `internal/factory/digest/digest_status_dirty_test.go` (new) —
  9-dirty integration tests covering the ACT's contract table
  plus determinism.
* `internal/factory/digest/digest_status_evidence_hashes_test.go`
  (new) — evidence-hash regression: manifest/stats/aggregate
  hashes change when the status letter flips from `M` to `A`; the
  test pins both directions and proves the hashes are not
  hardcoded.
* `internal/factory/digest/digest_status_range_test.go` (new) —
  range-mode regression tests covering ordinary add/mod/del/
  rename plus a mixed commit.
* `internal/factory/digest/digest_test_helpers_test.go` (new) —
  `RunGitForTest` / `RunGitWithExitCodeForTest` test-only helpers
  that capture stdout and exit code for use in integration
  tests.
* `docs/acts/ACT-LEAMAS-FACTORY-DIGEST-STAGED-STATUS-CLASSIFICATION01.md`
  (new).
* `docs/factory/digest.md` — adds a "Status classification"
  section that documents the new semantics, similarity threshold,
  and NUL-safe parsing guarantees.
* `docs/close-reports/ACT-LEAMAS-FACTORY-DIGEST-STAGED-STATUS-CLASSIFICATION01.md`
  (this file).

## Behavior changed

### Before

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

### After (correct)

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
```

The manifest now agrees path-for-path with
`git diff --cached --name-status -z --find-renames --find-copies <base> --`,
and every `ChangedFile` carries an explicit change kind sourced
from that structured output rather than from a presence-flag
heuristic.

## Exact commands run

| Command (with budget)                                    | Elapsed | Exit | Notes |
|----------------------------------------------------------|--------:|-----:|-------|
| `gofmt -w internal/factory/digest/*.go` (~5s)              | <0.1s   | 0    | reformatted parser, range_types and tests |
| `go vet ./...` (~30s)                                     | <2s     | 0    | clean |
| `go test ./internal/factory/digest -count=1` (~120s)      | 3.5s    | 0    | full package, 120+ tests |
| `go test ./internal/factory/digest -run 'TestStagedStatus\|TestDirtyStatus\|TestParseGit\|TestNormalize\|TestBuildManifest' -count=1 -v` (~30s) | 0.7s | 0 | matrix + parser + BuildManifest |
| `go test ./internal/factory/digest -run 'TestRangeMode' -count=1 -v` (~30s) | 0.25s | 0 | range regression |
| `go test ./internal/factory/digest -run 'TestEvidenceHashes' -count=1 -v` (~30s) | 0.2s | 0 | evidence hash regression |
| `go test ./internal/factory/digest -count=5` (repeat) (~120s) | 11.4s | 0 | 5 consecutive runs all green |
| `go test ./cmd/leamas -count=1` (~120s)                  | 5.1s    | 0    | CLI wiring still works |
| `CGO_ENABLED=0 go build -trimpath -o bin/leamas ./cmd/leamas` (~30s) | <3s | 0 | 12,780,092 bytes, statically linked |
| `./bin/leamas factory verify llm-friendly` (~30s)        | <1s     | 0    | "llm-friendly verification PASSED" |
| `./bin/leamas factory verify agent-context` (~30s)       | <1s     | 0    | "agent-context verification PASSED" |
| `./bin/leamas factory verify forbidden-patterns` (~30s)  | <1s     | 0    | "forbidden-patterns verification PASSED" |
| `git diff --check` (~10s)                                | <0.1s   | 0    | whitespace hygiene clean |
| `CGO_ENABLED=0 go build -trimpath -o /tmp/leamas-digest-status ./cmd/leamas` (~30s) | <3s | 0 | self-hosting binary |
| `/tmp/leamas-digest-status factory digest --staged --output /tmp/digest-status-proof.txt` (~30s) | 0.13s | 0 | self-hosting digest |
| `timeout 60 make factorize`                              | 60s     | 124 (terminated) | Got past `agent-context / docs / doctrine / doctrine-agent-contracts / domain-boundaries` (each OK in 0.00s) before timing out on the heavier duplicate-code phase. Same blocking previously documented. |
| `timeout 60 make gate`                                   | 60s     | 124 (terminated) | Same blocking; gate re-runs the early OK phases and then hangs on the live-tree duplicate-code phase. Same blocking previously documented. |

## Self-hosting proof (literal Oracle)

The staged ACT changes, captured verbatim from
`git diff --cached --name-status -z --find-renames=30% --find-copies=30% HEAD --`,
are:

```text
A  internal/factory/digest/digest_status_dirty_test.go
A  internal/factory/digest/digest_status_evidence_hashes_test.go
A  internal/factory/digest/digest_status_range_test.go
A  internal/factory/digest/digest_status_staged_test.go
A  internal/factory/digest/digest_test_helpers_test.go
M  internal/factory/digest/file_evidence.go
M  internal/factory/digest/file_operations.go
A  internal/factory/digest/git_status_parser.go
A  internal/factory/digest/git_status_parser_test.go
M  internal/factory/digest/range_types.go
M  internal/factory/digest/review_manifest.go
M  internal/factory/digest/review_test.go
```

The digest's `CHANGESET_MANIFEST` lists exactly these 12 paths
with identical status letters in lexicographic order. The
`CHANGESET_STATS` reports:

```text
files_changed=12
added_files=7
modified_files=5
deleted_files=0
renamed_files=0
copied_files=0
untracked_files=0
unmerged_files=0
binary_files=0
generated_files=0
test_files=7
doc_files=0
source_files=5
config_files=0
```

Independent recomputation from the Git oracle gives
`added_files=7, modified_files=5, files_changed=12`, matching the
digest exactly.

The original defect was the four-added/one-modified reproduction
showing

```text
A  internal/factory/gate/gate.go
A  <four new files>
added_files=5
modified_files=0
```

instead of the correct

```text
M  internal/factory/gate/gate.go
A  <four new files>
added_files=4
modified_files=1
```

That scenario is exercised by
`TestStagedStatus_FourAddedOneModified` and passes against the
literal Git oracle used in the body of the ACT:

```text
git diff --cached --name-status -z --find-renames --find-copies HEAD --
```

in combination with the test helper
`requireStagedAgreementAgainstOracle`. The same scenario is
reproduced at the unfixed-fixture scale by the self-hosting
proof above (every modified existing file is rendered `M`, not
`A`).

## Skipped / deferred checks

### Canonical full-tree verification

`make factorize` and `make gate` are the canonical
leamas-repository-wide gates. Both exercise the duplicate-code
live-tree machinery that has been independently blocked in
previous ACTs. The ACT explicitly forbids starting those ACTs
(`Out of scope` section). The current ACT notes their status
without claiming them as completed verification.

### Evidence: behavior on baseline

* Before this ACT, `BuildManifest` used the boolean inference
  (`Tracked && StagedPresent && !UnstagedPresent` ⇒ `A`).
* Baseline behaviour exposed in
  `docs/close-reports/ACT-LEAMAS-COMPILER-VERSION-STAMPING01`
  etc. is unaffected by this ACT.
* `TestBuildManifest_NoBooleanInference` (this ACT) asserts the
  inverse contract: a `ChangedFile` with presence flags but no
  `Kind` must not be classified as `A`/`M`. If the predicate
  ever re-appears, this test catches it.

## Follow-up ACTs

* `ACT-LEAMAS-FACTORY-FACTORIZE-RUNNER-FIXTURE01`
* `ACT-LEAMAS-FACTORY-DUPCODE-PERF-RATCHET01`

These remain blocked on the duplicate-code runtime and are
explicitly out of scope for this ACT.
