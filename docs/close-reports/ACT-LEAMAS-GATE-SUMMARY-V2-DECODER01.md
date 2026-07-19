# ACT-LEAMAS-GATE-SUMMARY-V2-DECODER01 — Draft Close Report

> **Status:** `IN PROGRESS`.
>
> This is a verification record and closure draft, not a claim that the ACT
> is closed. The implementation must be committed and reviewed before any
> separate closure-only status change. `NORMALIZATION01` remains pending.

## Summary

The working implementation provides a strict, bounded, versioned v1/v2
gate-summary decoder. It performs complete envelope scanning, duplicate-key
rejection, exact version dispatch, embedded Draft 2020-12 schema validation,
structured schema-error translation, and strict version-specific Go decoding.

The correction review was resolved in place; no correction ACT was created.
The active contract now exists at
`docs/acts/ACT-LEAMAS-GATE-SUMMARY-V2-DECODER01.md`.

## Status and Git disposition

- Contract base: `a9af32318be3709ac408b122f0028cb45469a82a`.
- ACT status: `IN PROGRESS`.
- `NORMALIZATION01`: `PENDING`, blocked on DECODER01 closure.
- Implementation commit: pending at the time of this draft.
- Closure-only status commit: deferred unless a later review permits closure.
- Force-push, amend, and history rewriting: not used.

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
- 2,048-level input nesting and 20,000 wrapper levels without panic;
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

### Focused GREEN

```bash
go test -count=1 ./internal/gatesummary/...
# PASS: ok github.com/s1onique/leamas/internal/gatesummary 0.186s

go test -race -count=1 ./internal/gatesummary/...
# PASS: ok github.com/s1onique/leamas/internal/gatesummary 2.138s

go vet ./internal/gatesummary/...
# PASS: no output
```

Benchmark smoke ran once per benchmark and passed:

```text
BenchmarkDecodeV1Minimal-24  1  2306421 ns/op  828192 B/op  10872 allocs/op
BenchmarkDecodeV2Minimal-24  1   565352 ns/op  117112 B/op   1386 allocs/op
BenchmarkDecodeV2Full-24     1   696003 ns/op  122576 B/op   2844 allocs/op
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

The final quick checks passed:

```bash
go vet ./...
CGO_ENABLED=0 go build -trimpath -o bin/leamas ./cmd/leamas
test -z "$(gofmt -l internal/gatesummary)"
git diff --check
./bin/leamas factory verify docs
```

Observed results: every command above exited 0. The docs verifier printed
`docs verification PASSED`.

`go test ./...` did not pass. It reached the repository's 10-minute package
timeout in
`internal/factory/dupcode.TestV4BaselineDelta_LiveTreeMatchesCommittedBaseline`.
The nested gate test also reported the known forbidden exec call and the
oversized `docs/factory/digest-contract.md`.

`make factorize` completed all verifiers and failed after 480.43 seconds on
two retained baseline findings:

- `internal/factory/digest/digest_test_helpers_test.go` directly calls
  `os/exec.Command` outside `internal/execution`;
- `docs/factory/digest-contract.md` has 420 lines, above the 400-line limit.

All DECODER01 files passed the LLM-friendliness check after the draft's prior
long line was repaired. A direct final llm-friendly verification reported only
`docs/factory/digest-contract.md`.

`make gate` ran after the final Go changes. It completed Factory verification,
reported the same two baseline findings, then reported three additional
working-tree/baseline conditions:

- the unstaged `go.mod`/`go.sum` dependency delta appears changed after tidy;
- pre-existing `internal/factory/digest/contract_test.go` is not gofmt-clean;
- its `go test ./...` phase did not complete before the command was killed at
  the execution tool's 10-minute limit.

The gate did complete `go mod tidy` and `go vet ./...` successfully before the
timeout. No repository-wide green claim is made.

## Deferred and out of scope

- Semantic validation and normalization remain `NORMALIZATION01` work.
- Digest, CLI, producer, dogfood, conformance, and release integration remain
  in their named downstream ACTs.
- Fuzz campaign execution is deferred to `CONFORMANCE01`; deterministic
  hostile-nesting no-panic tests are present here.
- The staged digest, implementation commit, and any closure-only commit were
  not yet complete when this draft was written.

## Follow-up disposition

Keep `DECODER01` as `IN PROGRESS` and `NORMALIZATION01` as `PENDING` until a
later closure review explicitly authorizes the status transition. No
closure-only status commit should be inferred from the implementation commit.
