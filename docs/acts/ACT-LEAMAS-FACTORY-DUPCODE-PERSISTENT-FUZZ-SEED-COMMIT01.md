# ACT-LEAMAS-FACTORY-DUPCODE-PERSISTENT-FUZZ-SEED-COMMIT01

## Title

Commit the persistent fuzz seed for `FuzzV4RegionPairingEquivalentToAllPairs`
and root-anchor the `.gitignore` `testdata/` rule.

## Parent Epic

Closes the seam left by
`ACT-LEAMAS-FACTORY-DUPCODE-V4-ALL-PAIRS-MATERIALIZATION-PERFORMANCE01-CORRECTION02-CORPUS-AND-EVIDENCE01`:
that ACT documented the canonical fuzz seed hash
`3fc61698be2e22940717afed57b7f563e49a15201582e060ee1e212da4a30a70`
(see `docs/close-reports/...CORPUS-AND-EVIDENCE01.detached-evidence.txt`)
but the corresponding corpus file was never committed, so a clean CI
checkout failed `TestV4Alignment_PersistentFuzzSeedFile`.

## Problem

CI surfaces the seed file as missing, and the diagnosis (CI/Go/GitHub
Actions were all blamed in turn) is local: the root `.gitignore`
contained

```gitignore
testdata/
```

which matches directories named `testdata` at any depth. That blanket
rule silently ignored every Go package's `testdata/` directory,
including `internal/factory/dupcode/testdata/fuzz/...`, so the seed
intended to be committed never reached the repository.

A second, latent defect: the nested `internal/execution/testdata/`
directory houses a built ELF binary (`testhelper/main`, ~3 MB) that the
exception `!internal/execution/testdata/testhelper/main.go` cannot
re-include (Git does not re-include files while a parent directory is
excluded). With the deep `testdata/` rule the binary was implicitly
ignored; widening the rule to only the root would expose it to the
LLM-friendly gate.

## Goal

After this ACT:

1. The canonical persistent fuzz seed at
   `internal/factory/dupcode/testdata/fuzz/FuzzV4RegionPairingEquivalentToAllPairs/3fc61698be2e2294`
   is committed.
2. The root `.gitignore` rule is root-anchored (`/testdata/`) so Go
   package-level `testdata/` directories can carry committed
   fixtures without re-introducing a blanket deep-ignore.
3. The build artifact under `internal/execution/testdata/testhelper/`
   remains ignored so the LLM-friendly gate stays green.

## Scope

- `internal/factory/dupcode/testdata/fuzz/FuzzV4RegionPairingEquivalentToAllPairs/3fc61698be2e2294`
  (new file, source-controlled 199-byte corpus entry).
- `.gitignore` (small surgical patch: root-anchor the `testdata/`
  pattern and add an explicit ignore for the documented
  `testhelper/main` build artifact).

## Non-goals

- No production-code change. No test-code change outside the dupcode
  package — the contract
  `TestV4Alignment_PersistentFuzzSeedFile` already exists and asserts
  the contract; this ACT merely satisfies it durably.
- No cleanup of the now-redundant
  `!internal/factory/doctrinecompiler/testdata/` re-include patterns.
  Removing them is R2 cleanup; keeping them is harmless and out of
  scope.
- No change to the LLM-friendliness verifier's internal `ignoredDirs`
  list; the binary is suppressed at the `.gitignore` layer instead.

## Executable contract

### Stable boundary

`internal/factory/dupcode/testdata/fuzz/FuzzV4RegionPairingEquivalentToAllPairs/<hex>`
holds a single, deterministic, source-controlled Go fuzz seed. The
contract is enforced by the existing
`TestV4Alignment_PersistentFuzzSeedFile` test plus the persistent
`hash(name) = v4PersistentFuzzSeedHash` invariant on the seed body.

### Test matrix

| Case | Dimension | Given | When | Expected |
|---|---|---|---|---|
| 1 | seed present locally | working tree contains the canonical bytes | `go test ./internal/factory/dupcode -run TestV4Alignment_PersistentFuzzSeedFile` | PASS |
| 2 | seed content matches contract | first 16 hex chars of SHA256(bytes) equal `3fc61698be2e2294` | same run | the equality assertion inside the test passes (no separate external check needed) |
| 3 | seed visible to git | `.gitignore` no longer blanket-ignores nested `testdata/` | `git check-ignore` and `git ls-files --error-unmatch` on the seed path | not ignored, listed at stage 0 with non-zero SHA |
| 4 | build artifact still ignored | the ELF `internal/execution/testdata/testhelper/main` is a local build product | `git check-ignore -v` on that path | exits 0, reports line 19 of `.gitignore` |
| 5 | LLM-friendly gate stays green | full repo, including the committed seed and the locally-built ELF when present | `make factorize` | all 15 verifiers report OK (specifically `llm-friendly` no longer flags the hidden binary) |
| 6 | source files visible | `internal/execution/testdata/testhelper/main.go` must remain committed | `git ls-files` | present, mode 100644 |

Cases 1, 3, 4, 5, 6 were exercised during this ACT and all PASSED.
Case 2 is delegated to the existing
`TestV4Alignment_PersistentFuzzSeedFile` assertion; no new external
check is added in this ACT.

## Approach

1. Establish RED by running the existing contract test on a clean
   checkout; confirm it fails with
   `no such file or directory` for the seed path.
2. Generate the canonical seed locally with a `seedgenerator`-tagged
   test in the `dupcode` package, mirroring the contract test's
   own encoder to guarantee identical bytes. Confirm SHA256 matches
   the documented value.
3. Delete the one-shot generator test (no committed mutator; the
   build tag also defends against accidental future commits).
4. Replace `testdata/` with `/testdata/` in `.gitignore` and add an
   explicit ignore for `internal/execution/testdata/testhelper/main`
   to keep the LLM-friendly gate green.
5. Stage the new seed and the `.gitignore` patch. Confirm
   `git ls-files --stage` and `git check-ignore -v` semantics.

## Verification (exact commands)

```bash
# RED witness — the contract test fails before the file exists
go test ./internal/factory/dupcode \
    -run 'TestV4Alignment_PersistentFuzzSeedFile$' -count=1 -v

# Generate the seed via a one-shot, tag-guarded test
go test -tags seedgenerator ./internal/factory/dupcode \
    -run '^TestGeneratePersistentFuzzSeed$' -count=1 -v
sha256sum \
    internal/factory/dupcode/testdata/fuzz/FuzzV4RegionPairingEquivalentToAllPairs/3fc61698be2e2294
# expected: 3fc61698be2e22940717afed57b7f563e49a15201582e060ee1e212da4a30a70

# GREEN witness — both contract tests now pass
go test ./internal/factory/dupcode \
    -run 'TestV4Alignment_(PersistentFuzzSeedFile|AsymmetricPersistentSeedContract)$' \
    -count=1 -v

# Repository semantics
git check-ignore -v \
    internal/factory/dupcode/testdata/fuzz/FuzzV4RegionPairingEquivalentToAllPairs/3fc61698be2e2294
git check-ignore -v internal/execution/testdata/testhelper/main

git add .gitignore \
    internal/factory/dupcode/testdata/fuzz/FuzzV4RegionPairingEquivalentToAllPairs/3fc61698be2e2294
git ls-files --stage \
    internal/factory/dupcode/testdata/fuzz/FuzzV4RegionPairingEquivalentToAllPairs/3fc61698be2e2294

# Full factory factorize and toolchain checks
make factorize
go vet ./...
CGO_ENABLED=0 go build -trimpath -o bin/leamas ./cmd/leamas
```

`make gate` was started. All 15 Factory verifiers (agent-context,
docs, doctrine, doctrine-agent-contracts, domain-boundaries, dupcode,
dupcode-baseline, exec-gate, executable-contract-first,
forbidden-patterns, git-hooks, language, llm-friendly, static-binary,
tooling-boundaries) reported OK in 415.32 s.
Toolchain (`go mod tidy`, `gofmt`, `go vet ./...`) all OK. The full
`go test ./...` invocation timed out on
`TestV4BaselineDelta_LiveTreeMatchesCommittedBaseline`, an unrelated
pre-existing live-tree scan that runs in 204 s when invoked in
isolation but exceeds the 10-minute package timeout under the
default parallel `go test` load. ACT-LEAMAS-FACTORY-WALLCLOCK-TIMINGS01
already documents this exact environmental limitation as
out-of-scope; the test itself passes in isolation. See the close
report for the truncated gate results.

## Risks

- Root-anchoring `testdata/` widens what nested `testdata/`
  directories Git will track. Every currently-existing nested
  `testdata/` (`internal/factory/doctrinecompiler/testdata/`,
  `internal/execution/testdata/`) was manually inspected; no new
  untracked tree would appear that the gate hasn't already
  accounted for.
- The test-stage `make gate` run timed out on
  `TestV4BaselineDelta_LiveTreeMatchesCommittedBaseline` independent
  of this ACT. The failure was reproduced as pre-existing via the
  WCT-01 close-report.

## Follow-ups

| ACT | Description | Priority |
|-----|-------------|----------|
| (none required) | The redundant re-include patterns for `internal/factory/doctrinecompiler/testdata/` could be collapsed in a future R1 cleanup patch. | low |
| (pre-existing) | `TestV4BaselineDelta_LiveTreeMatchesCommittedBaseline` parallel-timeout per WCT-01. | medium |
