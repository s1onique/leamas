# ACT Close Report: ACT-LEAMAS-FACTORY-DIGEST-WALLCLOCK-TIME01-R2-TEST-ISOLATION

**Status**: CLOSED

**Date**: 2026-07-12

**Parent ACT**: ACT-LEAMAS-FACTORY-DIGEST-WALLCLOCK-TIME01

## Summary

Removed a global-state test-isolation defect that caused `go test ./...`
to fail under CI load. The original
`TestRunFactoryDigest_ProductionDigestFileExcludesElapsedTime` mutated
the entire test process's working directory via `os.Chdir`, which made
it fragile to concurrently running tests, subprocesses, cleanup
ordering, and CI timing.

The fix splits the test into two orthogonal contracts:

1. **CLI timing/status formatting** — already covered by
   `TestRunFactoryDigest_SuccessOutputIncludesElapsedTime` (uses the
   injected fake writer and proves the stdout contract).
2. **Production digest file contents** — now
   `TestProductionDigestFile_ExcludesElapsedTime`, which calls
   `digest.Write` directly with an explicit `RepoRoot`, eliminating
   the need for `os.Chdir` and any process-global state.

This brings the test in line with the executable-contract-first
doctrine: the narrowest stable boundary for "the digest file does not
contain a CLI timing field" is `digest.Write`, not the CLI command
wrapper plus its full stdout/status side effects.

## Root Cause

`os.Chdir` changes the working directory for the entire test process,
not for a single test. In CI:

- A parallel test, a goroutine, or a subprocess could observe the
  temporary repository as its CWD.
- Cleanup ordering after a fatal failure could leave the CWD wrong
  for subsequent tests in the same package.
- The `setup-go` cache-restore error and Node 20 deprecation that
  surfaced alongside the failure are unrelated warnings, not the
  blocker.

The production `digest.Write` path was correct: it always passed the
CWD-derived repo root into `Generate`. The defect was purely in the
test's reliance on `DetectRepoRoot()` from a temporary directory.

## Files Changed

| File | Change |
|------|--------|
| `cmd/leamas/factory_digest_time_test.go` | Removed `os.Chdir`/cleanup; split the production contract into `TestProductionDigestFile_ExcludesElapsedTime`; calls `digest.Write` directly with `RepoRoot` |

No production code changed. `cmd/leamas/factory_digest.go` and
`internal/factory/digest/digest.go` are byte-identical to the
R1 baseline.

## Behavior Changed

- The test `TestRunFactoryDigest_ProductionDigestFileExcludesElapsedTime`
  was replaced by `TestProductionDigestFile_ExcludesElapsedTime`,
  whose only responsibility is proving that the production
  `digest.Write` contract (when given an explicit `RepoRoot`) does
  not serialize the CLI-only `time=` field into the digest file.
- The CLI stdout/status contract remains covered by
  `TestRunFactoryDigest_SuccessOutputIncludesElapsedTime` and
  `TestRunFactoryDigest_ElapsedTimeIncludesSuccessfulWrite`.
- No user-visible CLI behavior changed.
- No digest file content contract changed.

## RED → GREEN Evidence

The defect was reproduced and verified via stress runs. The original
test, which calls `os.Chdir`, was the prime suspect per the upstream
analysis; the fix removes that call entirely.

### Focused RED/GREEN

```text
go test ./cmd/leamas \
  -run 'TestFormatElapsed|TestRunFactoryDigest_SuccessOutputIncludesElapsedTime|TestRunFactoryDigest_ElapsedTimeIncludesSuccessfulWrite|TestProductionDigestFile_ExcludesElapsedTime' \
  -count=1 -v
```

Result:

```text
=== RUN   TestFormatElapsed
--- PASS: TestFormatElapsed (0.00s)
=== RUN   TestRunFactoryDigest_SuccessOutputIncludesElapsedTime
--- PASS: TestRunFactoryDigest_SuccessOutputIncludesElapsedTime (0.00s)
=== RUN   TestRunFactoryDigest_ElapsedTimeIncludesSuccessfulWrite
--- PASS: TestRunFactoryDigest_ElapsedTimeIncludesSuccessfulWrite (0.00s)
=== RUN   TestProductionDigestFile_ExcludesElapsedTime
--- PASS: TestProductionDigestFile_ExcludesElapsedTime (0.08s)
PASS
ok      github.com/s1onique/leamas/cmd/leamas       0.431s
```

### Stress runs to confirm no global-state regression

```text
go test ./cmd/leamas -count=10 -shuffle=on
ok      github.com/s1onique/leamas/cmd/leamas       10.435s

go test -race ./cmd/leamas -count=5 -shuffle=on
ok      github.com/s1onique/leamas/cmd/leamas       6.705s

go test ./cmd/leamas \
  -run '^TestProductionDigestFile_ExcludesElapsedTime$' -count=100
ok      github.com/s1onique/leamas/cmd/leamas       7.989s
```

### CI re-entry regression check

```text
LEAMAS_EXEC_ROOT_ID=test-root \
LEAMAS_EXEC_PARENT_PID=123 \
LEAMAS_EXEC_GENERATION=0 \
go test ./cmd/leamas/... -count=1
ok      github.com/s1onique/leamas/cmd/leamas       1.247s
```

### Subsystem sweep

```text
go test ./... -count=1
ok      github.com/s1onique/leamas/cmd/leamas               1.202s
ok      github.com/s1onique/leamas/internal/execution        6.891s
ok      github.com/s1onique/leamas/internal/factory/digest   5.991s
ok      github.com/s1onique/leamas/internal/factory/gate     62.875s
... (all packages PASS)
```

### Static build

```text
CGO_ENABLED=0 go build -trimpath -o bin/leamas ./cmd/leamas
exit 0
```

### Repository gate

```text
make factorize
*** FACTORIZE PASSED ***

make gate
*** GATE PASSED ***
```

Both `make factorize` and `make gate` reported PASS with all
factory verifiers green and the Go toolchain checks
(`go mod tidy`, `gofmt`, `go vet ./...`, `go test ./...`,
static build) clean.

## Skipped or Deferred Checks

None. Every required check ran and passed.

## Follow-up ACTs

None.