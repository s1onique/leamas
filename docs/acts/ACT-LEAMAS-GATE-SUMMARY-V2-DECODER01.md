# ACT-LEAMAS-GATE-SUMMARY-V2-DECODER01

## Title

Implement the frozen strict gate-summary v1/v2 decoder.

## Parent Epic

[`EPIC-LEAMAS-GATE-SUMMARY-SCHEMA-V2-ADOPTION01`](../epics/EPIC-LEAMAS-GATE-SUMMARY-SCHEMA-V2-ADOPTION01.md)

## Status

`IN PROGRESS`. The implementation and executable contract are under review.
`NORMALIZATION01` remains pending and is not ready.

## Problem

Leamas has frozen v1 and v2 gate-summary schemas and reader semantics, but
it does not yet have one authoritative bounded decoder that preserves the
version-dispatch boundary, rejects ambiguous JSON, validates the selected
schema, and translates structured validator failures into stable `GS_*`
diagnostics.

## Goal

Deliver an internal Go package that accepts structurally valid v1 and v2
wire documents and rejects invalid input at the exact stage frozen by
`CONTRACT01-CORRECTION03`, without performing semantic normalization.

## Scope

- Bounded `io.Reader` input with the 4 MiB + 1 sentinel.
- Complete JSON token scanning and trailing-value detection.
- Duplicate-key rejection at every object depth.
- Presence-aware `schema_version` probing and exact lexical dispatch.
- Embedded Draft 2020-12 v1/v2 schemas using
  `jsonschema/v6 v6.0.2` with `AssertFormat()`.
- Structural schema-error translation from structured fields only.
- Strict version-specific Go wire decoding.
- Stable diagnostic ordering and an internal stage observer for tests.
- Determinism, concurrency, resource-bound, negative, and benchmark evidence.

## Non-goals

- Semantic invariants or normalization.
- Digest rendering or CLI integration.
- Producer changes or source-document rewriting.
- Making `NORMALIZATION01` ready before this ACT has Git closure.
- Any gateway, provider-routing, database, or authentication behavior.

## Executable contract

### Stable boundary

The stable boundary is untrusted raw JSON entering `Decode(io.Reader)` and
leaving as exactly one of:

1. a sealed version-specific wire `Document`;
2. ordinary input `Diagnostics` with `Err == nil`; or
3. an operational `Err`, optionally accompanied by `GS_INTERNAL`.

The owning stages are:

```text
bounded read
→ syntax and complete-token scan
→ duplicate-key scan
→ version probe
→ version dispatch
→ selected-schema validation and structural translation
→ strict wire decode
```

### Test matrix

| Case | Dimension | Given | Expected owner/result | Downstream evidence |
|---|---|---|---|---|
| Size sentinel | reader bound | 4 MiB + 1 bytes with more source available | bounded read → `GS_DOCUMENT_TOO_LARGE` | no token/schema/wire work |
| Missing version | presence | no top-level member | version probe → `GS_VERSION_MISSING` | schema and wire uninvoked |
| Null version | presence/type | `schema_version: null` | version probe → `GS_INVALID_VERSION_TYPE` | schema and wire uninvoked |
| Container version | scanner state | `{}`, nested object, `[]`, or nested array | version probe → `GS_INVALID_VERSION_TYPE` | complete value consumed; schema uninvoked |
| Malformed number | JSON grammar | leading zero or plus | syntax → `GS_MALFORMED_JSON` | schema and wire uninvoked |
| Decimal/exponent | lexical type | valid non-integral number spelling | version probe → `GS_INVALID_VERSION_TYPE` | schema and wire uninvoked |
| Unsupported integer | dispatch | integral value outside `{1,2}` | dispatch → `GS_UNSUPPORTED_VERSION` | schema and wire uninvoked |
| Trailing value | EOF | a second token after the object | syntax → `GS_TRAILING_JSON` | schema and wire uninvoked |
| Malformed suffix | EOF | invalid token after the object | syntax → `GS_MALFORMED_JSON` | schema and wire uninvoked |
| Duplicate member | lexical identity | duplicate at any object depth | duplicate scan → `GS_DUPLICATE_KEY` | schema and wire uninvoked |
| Schema leaf | translation | every frozen structured row | exact code, path, fanout, and values | schema invoked; wire uninvoked |
| Impossible version leaf | integration drift | post-dispatch required/type/const failure | `GS_INTERNAL` plus operational `Err` | wire uninvoked |
| Schema/wire disagreement | integration drift | injected non-validation or strict-decode error | `GS_INTERNAL` plus wrapped `Err` | trace records invocation |
| Valid v1/v2 | happy path | every valid fixture | sealed `Document` | selected schema and wire invoked |
| Hostile nesting | panic safety | deep containers or wrapper tree | deterministic diagnostic | no panic |

### RED evidence

- Baseline command: `go test -count=1 ./internal/gatesummary/...`
  passed before the new matrix was added.
- First matrix run failed to compile because the internal `stage`,
  `decodeTrace`, and injected dependency boundary did not exist.
- After adding only that observer boundary, the focused run exposed the
  intended behavioral failures: operational errors had `Err == nil`,
  `Success()` ignored diagnostics, null/container versions crossed scanner
  state, keyword identity used raw suffixes, and malformed roots could panic.
- The review findings are additional RED evidence for `More()` misuse,
  missing root-wrapper validation, and incorrect lexical accounting.

### GREEN evidence

Focused GREEN currently includes:

```bash
go test -count=1 ./internal/gatesummary/...
go test -race -count=1 ./internal/gatesummary/...
go vet ./internal/gatesummary/...
go test -run='^$' -bench='BenchmarkDecode(V1Minimal|V2Minimal|V2Full)$' \
  -benchtime=1x -count=1 ./internal/gatesummary
```

Repository-wide verification and Git closure remain required before status
can change from `IN PROGRESS`.

### Exceptions

None.

## Acceptance Criteria

- [x] The 4 MiB bound reads exactly through the +1 sentinel and no farther.
- [x] Null and all required container-valued versions map to
      `GS_INVALID_VERSION_TYPE` without selected-schema invocation.
- [x] Scanner and strict decoder use next-token `io.EOF` checks, not
      `Decoder.More()` outside a container.
- [x] Schema translation validates the root wrapper and uses parsed base URL
      plus RFC 6901 fragment and keyword tokens.
- [x] Every frozen schema-error row has direct exact-code/path evidence.
- [x] Operational schema/wire drift sets `Result.Err`; `Success()` rejects
      diagnostics and zero documents.
- [x] All 41 committed fixtures assert owning stage, schema selection and
      invocation, wire invocation, and diagnostic paths.
- [x] The lexical matrix records 123 normative contract cases and 146
      template-expanded executions.
- [x] Reader sentinel, fanout order, collection values, fallback, `kind.Not`,
      impossible version, empty root, hostile nesting, and benchmarks have
      direct evidence.
- [x] `jsonschema/v6` is a direct dependency and module metadata is tidy.
- [ ] Required repository verification is complete with honest results.
- [ ] The full implementation changeset has been staged and reviewed through
      a staged Factory digest.
- [ ] The implementation has a forward commit.
- [ ] A separate closure-only status commit is made only if review permits;
      until then this ACT remains `IN PROGRESS` and `NORMALIZATION01` pending.

## Verification Commands

```bash
go test -count=1 ./internal/gatesummary/...
go test -race -count=1 ./internal/gatesummary/...
go test -run='^$' -bench=. -benchtime=1x -count=1 ./internal/gatesummary
go mod tidy
git diff -- go.mod go.sum
go mod verify
go test ./...
go vet ./...
CGO_ENABLED=0 go build -trimpath -o bin/leamas ./cmd/leamas
make factorize
make gate
git diff --check
git status --short
go run ./cmd/leamas factory digest --staged --output build/staged-digest.txt
```

## Reviewer Focus

Review presence-vs-null handling, complete consumption of version containers,
EOF classification, exact schema keyword identity, root-wrapper handling,
operational `Result` invariants, and proof that rejected versions cannot invoke
schema or wire decoding. Also confirm that no semantic normalization has
entered this ACT.

## Close Report Stub

The draft report is at
[`docs/close-reports/ACT-LEAMAS-GATE-SUMMARY-V2-DECODER01.md`](../close-reports/ACT-LEAMAS-GATE-SUMMARY-V2-DECODER01.md).
It is not a closure claim while this ACT remains `IN PROGRESS`.
