# ACT Close Report: ACT-LEAMAS-FACTORY-DIGEST-WALLCLOCK-TIME01

**Status**: CLOSED

**Date**: 2026-07-12

## Summary

Added elapsed execution time to successful `leamas factory digest` CLI status output.

The timing starts at the beginning of the digest command handler and is measured
after the digest write callback returns successfully. The elapsed value is
rendered as seconds with exactly two decimal places in a `time=<seconds>s` field
immediately before the final `OK` token.

## R1 Update

Reviewer feedback found that the first wrapper-level tests did not fully prove
that timing includes successful digest writing or that production digest output
excludes the CLI timing field.

R1 added a deterministic clock seam and stronger tests:

- `runFactoryDigestWithClock` lets tests inject `now` and `since` while
  production still calls `time.Now` and `time.Since`.
- `TestRunFactoryDigest_ElapsedTimeIncludesSuccessfulWrite` fails if elapsed
  time is measured before the digest write callback completes.
- `TestRunFactoryDigest_ProductionDigestFileExcludesElapsedTime` invokes the
  production `digest.Write` path against a temporary Git repository and checks
  the generated digest file with a precise timing-field regexp.

## Files Changed

| File | Change |
|------|--------|
| `cmd/leamas/factory_digest.go` | Added timing and the injectable clock seam |
| `cmd/leamas/factory_digest_time_test.go` | Added elapsed-time and production-writer tests |
| `docs/close-reports/ACT-LEAMAS-FACTORY-DIGEST-WALLCLOCK-TIME01.md` | Added this close report |

## Behavior Changed

- Successful `leamas factory digest` executions now emit a final status line:
  ```text
  digest: mode=auto output=/path/to/digest.txt time=0.01s OK
  ```
- `OK` remains the final token.
- The timing field is included only in CLI operational output.
- Generated digest file content does not receive the `time=` field.
- Elapsed time is formatted with `fmt.Sprintf("%.2fs", d.Seconds())`, not
  `time.Duration.String()`.

## Executable Contract

- Pure `formatElapsed` table coverage for:
  - `0.00s`
  - `0.01s`
  - `1.00s`
  - `1.01s`
  - `121.42s`
- CLI success-output coverage for:
  ```regexp
  ^digest: mode=\S+ output=.+ time=\d+\.\d{2}s OK\n$
  ```
- Deterministic timing-boundary coverage proves elapsed measurement happens
  after successful digest writing.
- Production `digest.Write` coverage proves the generated digest file does not
  contain a CLI timing field matching:
  ```regexp
  (?:^|[ \n])time=\d+\.\d{2}s(?:[ \n]|$)
  ```

## RED Evidence

Initial focused test run before production implementation failed for the
intended reason:

```text
cmd/leamas/factory_digest_test.go:186:13: undefined: formatElapsed
FAIL	github.com/s1onique/leamas/cmd/leamas [build failed]
```

## Verification Commands and Results

- `go test ./cmd/leamas -run 'TestFormatElapsed\|TestRunFactoryDigest_SuccessOutputIncludesElapsedTime\|TestRunFactoryDigest_DigestFileExcludesElapsedTime' 2>&1`
  - FAIL before implementation: `undefined: formatElapsed`
- Same focused command after implementation
  - PASS
- `gofmt -w .../cmd/leamas/factory_digest.go .../cmd/leamas/factory_digest_test.go`
  - PASS
- `go test ./cmd/leamas -run 'TestFormatElapsed\|TestRunFactoryDigest' 2>&1`
  - PASS
- `go test ./... 2>&1`
  - PASS before adding this close report
- `go vet ./... 2>&1`
  - PASS
- `CGO_ENABLED=0 go build -trimpath -o bin/leamas ./cmd/leamas 2>&1`
  - PASS
- `make factorize 2>&1`
  - PASS before adding this close report
- `make gate 2>&1`
  - PASS before adding this close report
- `make factorize 2>&1 && make gate 2>&1`
  - FAIL after adding this close report: LLM-friendly long-line violation
- `awk 'length($0)>240 {print NR ":" length($0) ":" $0}' .../ACT-LEAMAS-FACTORY-DIGEST-WALLCLOCK-TIME01.md`
  - PASS after wrapping the long line; no output
- `make factorize 2>&1`
  - PASS after close-report fix
- `make gate 2>&1`
  - PASS after close-report fix
- `./bin/leamas factory digest --output /tmp/leamas-digest-wallclock.txt 2>&1`
  - PASS; emitted `digest: mode=auto output=/tmp/leamas-digest-wallclock.txt time=0.08s OK`
- R1 focused test command:
  `go test ./cmd/leamas -run 'TestFormatElapsed\|TestRunFactoryDigest_SuccessOutputIncludesElapsedTime\|TestRunFactoryDigest_ElapsedTimeIncludesSuccessfulWrite\|TestRunFactoryDigest_ProductionDigestFileExcludesElapsedTime' 2>&1`
  - PASS
- R1 `go test ./... 2>&1`
  - FAIL once: exec-gate rejected direct `os/exec.Command` in the new test
- R1 test fix
  - Replaced direct `os/exec.Command` with the `internal/execution` gateway
- R1 `go test ./... 2>&1`
  - PASS
- R1 `go vet ./... 2>&1`
  - PASS
- R1 `CGO_ENABLED=0 go build -trimpath -o bin/leamas ./cmd/leamas 2>&1`
  - PASS
- R1 `make factorize 2>&1`
  - PASS before this close-report update
- R1 `make gate 2>&1`
  - PASS before this close-report update

## Skipped or Deferred Checks

None.

## Follow-up ACTs

None.
