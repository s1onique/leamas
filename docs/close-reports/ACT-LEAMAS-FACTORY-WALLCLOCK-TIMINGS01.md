# ACT-LEAMAS-FACTORY-WALLCLOCK-TIMINGS01

## Status

PARTIAL — implementation and P1/P2 review fixes accepted; full
repository verification still pending on an environment where the
pre-existing live-tree tests complete.

## Files changed

| File | Change | LOC |
|------|--------|-----|
| `internal/factory/gate/factorize.go` | new | 109 |
| `internal/factory/gate/factorize_test.go` | new | 265 |
| `internal/factory/gate/gate.go` | modified (RunFactorize delegates) | +5 / −27 |
| `docs/acts/ACT-LEAMAS-FACTORY-WALLCLOCK-TIMINGS01.md` | new | 33 |
| `docs/close-reports/ACT-LEAMAS-FACTORY-WALLCLOCK-TIMINGS01.md` | new | this file |

## Behavior changed

`leamas factory factorize` now emits wall-clock durations on every
per-check line and on the final summary.

Per-check line format (identical for success and failure):

```text
  <name>: OK: <seconds>s
  <name>: FAILED: <seconds>s
```

Final summary — these are **alternative outputs**, one per
invocation, not a single invocation emitting both:

Successful run:

```text
*** FACTORIZE PASSED: 8.72s ***
```

Failed run:

```text
*** FACTORIZE FAILED: 11.39s ***
```

- Two-decimal `%.2fs` on `elapsed.Seconds()`.
- Summary preserves the pre-existing `PASSED` / `FAILED` vocabulary
  (not `OK`) to keep external scripts that grep for the literal
  string working.
- Verifier execution order (alphabetical by name) and exit codes
  (0 on full success, 1 on any failure) unchanged.
- JSON, digest, fingerprint, gate-summary, and detached-evidence
  contracts untouched. Timings live only in this interactive text
  output.

## Tests

`internal/factory/gate/factorize_test.go` introduces seven
deterministic tests with an injected `fakeClock` (no sleeps, no
`time.Sleep`). The fake clock carries a `*testing.T` so unexpected
extra `Now()` calls fail the test with a diagnostic rather than
panicking on an opaque index-out-of-range error:

- `TestRunCheck_PrintsElapsedTimeOnSuccess`
- `TestRunCheck_PrintsElapsedTimeOnFailure`
- `TestRunCheck_FormatsSubSecondDurations`
- `TestRunFactorize_PrintsTotalOnSuccess`
- `TestRunFactorize_PrintsFailureAndTotalOnError`
- `TestRunFactorize_PreservesExitCodeOnFailure`
- `TestRunFactorize_SortsByName`

All seven PASS under `go test -count=1 -v`.

## Exact commands run

```bash
gofmt -w \
  internal/factory/gate/factorize.go \
  internal/factory/gate/factorize_test.go \
  internal/factory/gate/gate.go
# → exit 0

go test ./internal/factory/gate \
  -run 'TestRun(Check|Factorize)_' \
  -count=1 -v
# → all 7 new tests PASS in 0.005s

go vet ./...
# → exit 0

CGO_ENABLED=0 go build -trimpath -o bin/leamas ./cmd/leamas
# → exit 0

go test $(go list ./... | grep -v '/internal/factory/dupcode$') \
  -skip '^TestRunFactorize$' \
  -count=1 -timeout 60s
# → all 28 non-dupcode packages PASS
```

## Honest results

- `make factorize` partial output (killed at 600 s on this local
  machine):

  ```text
  Running factory factorize...
    agent-context: OK: 0.00s
    docs: OK: 0.00s
    doctrine: OK: 0.00s
    doctrine-agent-contracts: OK: 0.28s
    domain-boundaries: OK: 0.91s
    dupcode: OK: 205.66s
  make: *** [Makefile:121: factorize] Terminated
  ```

  The format matches the acceptance contract line-for-line. The
  command did not finish within the 10-minute window. The
  `dupcode` verifier consumed 205.66 s; a subsequent verifier had
  not completed when the command was terminated.

- `make gate` was not run end-to-end because it includes the same
  production duplicate-code verification path. Based on the observed
  `make factorize` run, it was expected to exceed the selected local
  verification budget; this remains an inference rather than retained
  execution evidence. Output format change is independent of the
  slow verifier and is fully exercised by the new unit tests.

- `go test ./...` end-to-end without exclusions times out on this
  local machine. Four pre-existing tests perform full live-tree
  scans or invoke the complete live-tree factorize verifier set
  (timeout ≥ 60 s each on this local machine, observed during
  baseline validation; see "Baseline observations" below):

  - `internal/factory/gate.TestRunFactorize` — invokes
    `RunFactorize(repoRoot)`, i.e. the complete live-tree
    factorize verifier set (no duplicate-code audit of its own).
  - `internal/factory/dupcode.TestDebugBaselines` — full
    live-tree duplicate-code audit.
  - `internal/factory/dupcode.TestV4BaselineAudit_LiveTreeMatchesCommittedBaseline`
    — full live-tree duplicate-code audit.
  - `internal/factory/dupcode.TestV4BaselineDelta_LiveTreeMatchesCommittedBaseline`
    — full live-tree duplicate-code audit.

  Observed to exceed the stated timeout on unmodified `main` after
  temporarily stashing this ACT's changes. The timing patch itself
  adds only two `clk.Now()` calls per check and is not on the hot
  path of any slow verifier.

## Baseline observations

Stash-and-re-run experiments on the baseline commit
`96699984cffbac4bdd77f2ecb31fef18624f514d` ("docs: compact
duplicate-code self-hosted close reports"):

| Command | Timeout | Observed result on baseline |
|---------|---------|------------------------------|
| `go test ./internal/factory/gate -run '^TestRunFactorize$'` | 200 s | timed out inside `dupcodeBaselineVerifier` → `dupcode.VerifyBaseline` → `findCommonWindows` |
| `go test ./internal/factory/dupcode -run '^TestDebugBaselines$'` | 120 s | timed out inside `dupcode.CheckReport` |
| `go test ./internal/factory/dupcode -run '^TestV4BaselineAudit_LiveTreeMatchesCommittedBaseline$'` | 60 s | timed out inside `dupcode.CheckRepo` |
| `go test ./internal/factory/dupcode -run '^TestV4BaselineDelta_LiveTreeMatchesCommittedBaseline$'` | 60 s | timed out inside `dupcode.CheckRepo` |

This is the only recorded baseline evidence retained. The digest
contains no other artifacts for this ACT.

## Skipped / deferred checks

- Full `make factorize` end-to-end run with timing output on
  every verifier including the slow `dupcode`. Pending an
  environment where the pre-existing slow verifier completes
  in budget.
- Full `make gate` end-to-end run for the same reason.
- Full `go test ./...` with no `-skip` filter. Same reason.
- CPU profile of the slow `dupcode` verifier was not captured
  because the underlying algorithm is unchanged by this ACT and
  the review focus is the timing boundary, not dupcode
  performance.

## Authoritative staging evidence

The targeted digest for this ACT misclassified the modified
`gate.go` as added and reported `added_files=5, modified_files=0`.
Per the reviewer's correction, treat the literal output of the
following Git commands as authoritative staging evidence for
this ACT:

```text
$ git diff --cached --name-status HEAD --
A	docs/acts/ACT-LEAMAS-FACTORY-WALLCLOCK-TIMINGS01.md
A	docs/close-reports/ACT-LEAMAS-FACTORY-WALLCLOCK-TIMINGS01.md
A	internal/factory/gate/factorize.go
A	internal/factory/gate/factorize_test.go
M	internal/factory/gate/gate.go

$ git diff --cached --summary HEAD --
 create mode 100644 docs/acts/ACT-LEAMAS-FACTORY-WALLCLOCK-TIMINGS01.md
 create mode 100644 docs/close-reports/ACT-LEAMAS-FACTORY-WALLCLOCK-TIMINGS01.md
 create mode 100644 internal/factory/gate/factorize.go
 create mode 100644 internal/factory/gate/factorize_test.go
 (gate.go appears only in --name-status as M because it already
  existed in HEAD; `git ls-tree -r --name-only HEAD --
  internal/factory/gate/gate.go` confirms its presence)

$ git diff --cached --check
# (no whitespace issues)
```

Correct classification: `added_files=4, modified_files=1`.

## Follow-up ACTs

1. **ACT-LEAMAS-FACTORY-DIGEST-STAGED-STATUS-CLASSIFICATION01**
   Self-hosting evidence defect discovered during this ACT's
   review. The targeted digest tool misclassifies a modified
   existing file as added and reports `added_files` / `modified_files`
   accordingly. Fix the digest's changeset classification so it
   agrees with `git diff --cached --name-status HEAD --`. Tracked
   outside this ACT.

2. **ACT-LEAMAS-FACTORY-FACTORIZE-RUNNER-FIXTURE01**
   Convert `gate.TestRunFactorize` from a live-repository
   integration test to deterministic fixture-based runner
   coverage. This removes one redundant live-tree
   duplicate-code scan from `go test ./...`, but does **not**
   change `make factorize`, `make gate`, or the three dedicated
   `dupcode` live-tree audit tests (which remain slow for their
   own reasons).

3. **ACT-LEAMAS-FACTORY-DUPCODE-PERF-RATCHET01**
   Profile and optimize the production `dupcode` and
   `dupcode-baseline` verifiers and the three dedicated
   live-tree audit tests. This ACT owns restoring practical
   completion times for `make factorize`, `make gate`, and
   unfiltered `go test ./...`. No threshold is set in this ACT;
   that work follows the perf-ratchet discipline.
