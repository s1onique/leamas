# Close Report: ACT-LEAMAS-GATE-SUMMARY-V2-DECODER01

> **Status:** `CLOSED (PARTIAL — retained unrelated baseline failures)`.
>
> The decoder contract and both implementation commits are reviewed and
> committed. Repository-wide failures retained below predate and remain
> unrelated to DECODER01. `NORMALIZATION01` is now `READY`.

## Summary

The implementation provides a strict, bounded, versioned v1/v2
gate-summary decoder. It performs complete envelope scanning, iterative
duplicate-key rejection, exact version dispatch, embedded Draft 2020-12 schema
validation, structured schema-error translation, and strict version-specific
Go decoding with exact, non-narrowing wire integers.

The correction review was resolved in place; no correction ACT was created.
The active contract now exists at
`docs/acts/ACT-LEAMAS-GATE-SUMMARY-V2-DECODER01.md`.

## Status and Git disposition

```text
implementation commit = 37d9720
working tree after implementation = clean
staged digest before implementation commit = reviewed
```

- Contract base: `a9af32318be3709ac408b122f0028cb45469a82a`.
- Implementation commit: `37d972053a472aeacbc5588f2be7960d3c177950`.
- Forward P0 fix: `a14a3d70b058cab83cc652b6bd79b22c27b3b7a2`.
- Forward-fix staged digest: reviewed; 10 files, no untracked content,
  `git_diff_check=pass`, and no dependency delta.
- Forward range digest: `37d9720..a14a3d7`, reviewed with the same manifest
  and patch-hygiene result.
- Closure-only status commit: this commit.
- ACT status: `CLOSED (PARTIAL — retained unrelated baseline failures)`.
- `NORMALIZATION01`: `READY`.
- Force-push, amend, reset, and history rewriting: not used.

## Files changed

### Documentation and dependency metadata

- `docs/acts/ACT-LEAMAS-GATE-SUMMARY-V2-DECODER01.md`
- `docs/close-reports/ACT-LEAMAS-GATE-SUMMARY-V2-DECODER01.md`
- `docs/epics/EPIC-LEAMAS-GATE-SUMMARY-SCHEMA-V2-ADOPTION01.md`
- `go.mod`
- `go.sum`

### Production package

- `internal/gatesummary/decode_trace.go`
- `internal/gatesummary/decoder.go`
- `internal/gatesummary/diagnostic.go`
- `internal/gatesummary/document.go`
- `internal/gatesummary/duplicate_keys.go`
- `internal/gatesummary/envelope.go`
- `internal/gatesummary/reader.go`
- `internal/gatesummary/schema_compile.go`
- `internal/gatesummary/schema_embed.go`
- `internal/gatesummary/schema_error_identity.go`
- `internal/gatesummary/schema_error_translate.go`
- `internal/gatesummary/schema_validate.go`
- `internal/gatesummary/strict_decode.go`
- `internal/gatesummary/version.go`
- `internal/gatesummary/wire_integer.go`
- `internal/gatesummary/wire_v1.go`
- `internal/gatesummary/wire_v2.go`

The package embeds the already committed schema files under
`internal/gatesummary/schema/`; those schema contracts were not changed.

### Tests

- `internal/gatesummary/concurrency_test.go`
- `internal/gatesummary/corpus_stage_test.go`
- `internal/gatesummary/corpus_test.go`
- `internal/gatesummary/decoder_benchmark_test.go`
- `internal/gatesummary/decoder_contract_test.go`
- `internal/gatesummary/determinism_test.go`
- `internal/gatesummary/duplicate_keys_contract_test.go`
- `internal/gatesummary/duplicate_keys_test.go`
- `internal/gatesummary/envelope_contract_test.go`
- `internal/gatesummary/envelope_test.go`
- `internal/gatesummary/reader_contract_test.go`
- `internal/gatesummary/reader_test.go`
- `internal/gatesummary/schema_compile_test.go`
- `internal/gatesummary/schema_error_structure_test.go`
- `internal/gatesummary/schema_error_table_test.go`
- `internal/gatesummary/schema_error_translate_test.go`
- `internal/gatesummary/testhelpers_test.go`
- `internal/gatesummary/version_generated_test.go`
- `internal/gatesummary/version_test.go`
- `internal/gatesummary/wire_integer_contract_test.go`
- `internal/gatesummary/wire_v1_test.go`
- `internal/gatesummary/wire_v2_test.go`

## Behavior changed

1. `Decode(io.Reader)` reads at most 4 MiB + 1 bytes. A sentinel-reader test
   proves that an additional source byte is not requested.
2. The envelope scanner distinguishes an absent discriminator from present
   JSON `null` and fully consumes object or array discriminator values.
3. `null`, `{}`, nested `{ "schema_version": 2 }`, `[]`, `[2]`, and an array
   containing that nested object all produce `GS_INVALID_VERSION_TYPE`.
4. Trailing input uses a next-token request and requires `io.EOF`; malformed
   trailing syntax and a second JSON value remain distinct outcomes.
5. Schema keyword identity is the exact tuple of parsed base URL and decoded
   RFC 6901 fragment plus keyword tokens. Raw suffix matching is gone.
6. A missing or malformed root `*kind.Schema` wrapper maps to `GS_INTERNAL`.
   Empty nested wrappers remain `GS_SCHEMA_VIOLATION` as frozen.
7. A post-dispatch required/type/const version failure maps to `GS_INTERNAL`
   and is surfaced as an operational `Result.Err`.
8. Non-validation schema failures and schema/wire disagreements set wrapped
   operational errors and may carry one `GS_INTERNAL` diagnostic.
9. `Result.Success()` now requires nil `Err`, no diagnostics, and a non-zero
   document version.
10. `jsonschema/v6 v6.0.2` is a direct module dependency; `x/text` remains
    indirect.
11. `WireInteger` preserves exact `json.Number` spellings and exposes explicit
    `BigInt` and checked `Int64` conversions. All schema-unbounded evidence
    integers use it; `decodeStrict` enables `UseNumber()`.
12. Duplicate detection now uses an explicit frame stack, per-object key maps,
    and linked JSON Pointer tokens. It has no call recursion proportional to
    input nesting and does not copy the full ancestor path at every level.
13. The full `Decode` pipeline exercises the deepest object and array nesting
    accepted by the active `encoding/json` syntax implementation and returns
    the exact version-type diagnostic without panic.

## Executable evidence

### Stage ownership

The internal unexported `decodeTrace` records:

```text
owning stage
selected version
schema invoked yes/no
wire decode invoked yes/no
```

`TestCorpusStageEvidence` has exactly 41 rows and asserts those fields plus
success/rejection, exact diagnostic code, and diagnostic path for every
committed fixture. Rejected version forms assert that both schema and wire
decoding remain uninvoked.

### Lexical accounting

The frozen lexical families and implementation expansion are recorded
separately:

```text
normative lexical contract cases       = 123
implemented template-expanded executions = 146

whitespace executions            = 100
leading-zero/plus executions     = 12
decimal/exponent executions      = 18
unsupported-integer executions   = 16
```

### Required negative evidence

Direct tests cover:

- all six required null/container discriminator forms;
- reader sentinel behavior;
- exact EOF and malformed-suffix behavior;
- every frozen schema-error table row;
- required/additional-property multi-name fanout and ordering;
- structured collection-limit expected/observed values;
- unknown leaf fallback;
- impossible post-dispatch version failure;
- missing, malformed, and empty root wrappers;
- `kind.Not` keyword identity;
- exact base-URL rejection of a suffix spoof;
- deepest syntax-accepted object and array input through full `Decode`, plus
  20,000 schema-error wrapper levels, without panic;
- exact preservation at `int64` max, max + 1, 512 digits, `int64` min, and
  min - 1 where the selected schemas permit negatives;
- injected schema, bootstrap, and wire operational failures;
- v1 minimal, v2 minimal, and v2 full benchmark smoke.

## Executable-contract cycle

### Baseline

Before the review matrix was added:

```bash
go test -count=1 ./internal/gatesummary/...
# PASS: ok github.com/s1onique/leamas/internal/gatesummary 0.157s
```

### RED

The first matrix run failed to compile because the internal observer and
injected capability boundary did not exist. After adding that boundary only,
the next focused run failed for the intended behavior: operational errors had
nil `Err`, `Success()` accepted diagnostics, container versions corrupted
scanner state, keyword identity was textual, and a nil root kind panicked.

The P0 follow-up matrix failed at the intended schema/wire boundary. Documents
containing `9223372036854775808`, a 512-digit integer, or
`-9223372036854775809` where negatives are permitted passed the selected schema
and then produced strict `int64` overflow plus `GS_INTERNAL`. The direct API
matrix separately failed to compile with `undefined: WireInteger`. Recursive
`walkForDuplicates` was structural RED from review; the deepest-input test was
already non-panicking under Go's growable stack, so that test is retained as
observable regression evidence rather than misreported as behavioral RED.

### Focused GREEN

```bash
go test -count=1 ./internal/gatesummary/...
# PASS: ok github.com/s1onique/leamas/internal/gatesummary 0.349s

go test -race -count=1 ./internal/gatesummary/...
# PASS: ok github.com/s1onique/leamas/internal/gatesummary 2.693s

go vet ./internal/gatesummary/...
# PASS: no output
```

Benchmark smoke ran once per benchmark and passed:

```text
BenchmarkDecodeV1Minimal-24  1  2523683 ns/op  829008 B/op  10878 allocs/op
BenchmarkDecodeV2Minimal-24  1   543551 ns/op  123360 B/op   1396 allocs/op
BenchmarkDecodeV2Full-24     1   834064 ns/op  192424 B/op   2952 allocs/op
```

These one-iteration figures are smoke evidence, not performance thresholds.

## Module verification

Commands run:

```bash
go mod tidy
git diff -- go.mod go.sum
go mod verify
```

Observed results:

- `go mod tidy`: exit 0.
- `jsonschema/v6 v6.0.2`: direct requirement.
- `golang.org/x/text v0.14.0`: indirect requirement.
- `go mod verify`: `all modules verified`.

## Repository verification

The refreshed quick checks passed:

```bash
go vet ./...
CGO_ENABLED=0 go build -trimpath -o bin/leamas ./cmd/leamas
go mod tidy -diff
go mod verify
test -z "$(gofmt -l internal/gatesummary)"
git diff --check
```

Observed results: every command above exited 0; `go mod tidy -diff` emitted no
module delta, and `go mod verify` printed `all modules verified`.

`go test ./...` did not pass. The command reached Go's 10-minute package
timeout in
`internal/factory/dupcode.TestV4BaselineDelta_LiveTreeMatchesCommittedBaseline`.
Its nested gate test reported the two retained baseline findings below. The
changed `internal/gatesummary` package passed in 0.265 seconds in that run.

`make factorize` completed all verifiers and failed after 494.67 seconds on
two retained baseline findings:

- `internal/factory/digest/digest_test_helpers_test.go` directly calls
  `os/exec.Command` outside `internal/execution`;
- `docs/factory/digest-contract.md` has 420 lines, above the 400-line limit.

All DECODER01 files passed the LLM-friendliness verifier; its only reported
finding was the unrelated 420-line baseline file above.

`make gate` ran after the final Go changes. It completed Factory verification,
reported those same two findings, and then ran the Go toolchain. `go mod tidy`
and `go vet ./...` passed. The gofmt phase reported the pre-existing
`internal/factory/digest/contract_test.go`. The execution tool killed the gate
while `go test ./...` was still running, so the gate did not complete and no
repository-wide green claim is made.

## Deferred and out of scope

- Semantic validation and normalization remain `NORMALIZATION01` work.
- Digest, CLI, producer, dogfood, conformance, and release integration remain
  in their named downstream ACTs.
- Fuzz campaign execution is deferred to `CONFORMANCE01`; deterministic
  hostile-nesting no-panic tests are present here.
- The retained exec-boundary, LLM-friendliness, baseline gofmt, and dupcode
  timeout findings are unrelated and remain owned outside DECODER01.

## Closure range

- Contract base: `a9af323`.
- Implementation checkpoint: `37d9720`.
- Forward P0 fix: `a14a3d7`.
- Lifecycle reconciliation: this closure-only commit.

## Follow-up disposition

`DECODER01` is `CLOSED (PARTIAL — retained unrelated baseline failures)`.
`NORMALIZATION01` is `READY`; semantic validation and normalization remain
strictly downstream and were not introduced by this ACT.
