# ACT-LEAMAS-GATE-SUMMARY-V2-EXACT-GEOMETRY-TESTS01-R9-CORRECTION01

## Title

Correct the R9 closure and establish narrow contract reconnaissance for
diagnostic ordering proof tests.

## Parent Epic

[`EPIC-LEAMAS-GATE-SUMMARY-SCHEMA-V2-ADOPTION01`](../epics/EPIC-LEAMAS-GATE-SUMMARY-SCHEMA-V2-ADOPTION01.md)

## Parent ACT

[`ACT-LEAMAS-GATE-SUMMARY-V2-EXACT-GEOMETRY-TESTS01`](./ACT-LEAMAS-GATE-SUMMARY-V2-EXACT-GEOMETRY-TESTS01.md)
(ACT itself; R9 is the correction target)

## Status

`CLOSED (PARTIAL — focused executable proof delivered; repository-wide commands not verified within external execution budget)`

## Problem

R9 claimed `CLOSED` but did not satisfy its objective:

- **Objective**: Create diagnostic ordering proof tests for multi-diagnostic scenarios.
- **Actual outcome**: All eight test files were deleted; no tests remain. The
  passing existing suite proves only that the pre-R9 baseline was restored.

The honest status should be `PARTIAL — test-design attempt reverted; objective
not delivered`, not `CLOSED`.

Additionally, the R9 close report contains five concrete defects:

1. **`CLOSED` overstates the outcome.** Must be `PARTIAL` or `CLOSED–NO-OP`.
2. **"Repository is clean" is false.** One untracked file remains:
   `docs/close-reports/ACT-...-R9.md`.
3. **"Files Changed: None" is inaccurate.** One file changed:
   the close report itself.
4. **Digest gate summary predates the work.** Cannot support an R9
   acceptance claim.
5. **"Deferred to CI" is too weak.** The 300-second timeout must be
   attributed to the external execution budget, not delegated.

## Goal

Establish narrow contract reconnaissance before expanding diagnostic
ordering tests. The first increment must be one small test file, not
another speculative batch.

## Scope

This ACT:

1. Corrects the R9 close report status and evidence.
2. Adds one valid V2 test-document builder.
3. Adds one passing two-diagnostic ordering proof.
4. Adds one structural decode-failure test that proves Decode returns
   `Success() == false`, `Err == nil`, no usable Document, and the
   expected wire-stage diagnostics. Normalization is caller-gated on
   `Decode.Success()`.
5. Identifies canonical entry points, existing constructors, and the
   diagnostic-precedence authority.
6. Runs focused package tests, genuinely cheap checks, and repository
   gates.
7. Records the full-suite command as verified, failed, or explicitly
   unavailable with exact timeout ownership.

This ACT does **not** add the full 8-file diagnostic ordering matrix.
That belongs to `EXACT-GEOMETRY-TESTS01-R10` after this correction is
closed.

## Non-goals

- No broad diagnostic ordering matrix (deferred to R10).
- No speculative test batch creation.
- No changes to production code.

## Entry-Point Inventory (Reconnaissance)

### Decode entry point

```go
// internal/gatesummary/decoder.go
func Decode(r io.Reader) Result
```

`Result` contains `Document`, `Diagnostics` (wire-level), and `Err`.

### Normalize entry point

```go
// internal/gatesummary/normalize.go
func Normalize(doc Document) NormalizationResult
```

`NormalizationResult` contains `Summary`, `Diagnostics` (semantic), and `Err`.

### Diagnostic-precedence authority

```go
// internal/gatesummary/diagnostic.go
var codePrecedence = map[string]int{...}
```

Lower rank = emitted earlier. Frozen at 1–27 (27 diagnostic codes).

### Existing test constructors and helpers

- `readFixture(t testing.TB, path string) []byte` — loads JSON from
  `testdata/`.
- `mustValidationError(t *testing.T, err error) *jsonschema.ValidationError`
  — typed error assertion.
- Inline JSON literals with `strings.NewReader()`.
- `Decode()` → `Result.Document` → `Normalize()` pipeline.

### Known construction constraints (from R9 reconnaissance)

- `minimalV2Wire()` does not exist; create checks inline with full fields.
- A pass check **must** have `exit_code: 0` or `extras.ExitCode` set to
  an integer zero, otherwise `GS_PASS_EXIT_CODE_MISMATCH` fires.
- A skip check **must** have `exit_code: null`.
- A fail check is valid with `exit_code: null` (infrastructure failure) or
  non-zero.
- Wire-level diagnostics (structural failures) appear in `Result.Diagnostics`
  during decode, before normalization is invoked.
- Normalization diagnostics appear in `NormalizationResult.Diagnostics`
  after normalization.

## Executable Contract

### Stable boundary

The stable boundary is the existing `internal/gatesummary` package:
its `Decode()`, `Normalize()`, and diagnostic registry.

### RED step

The R9 attempt discovered:

1. Pass checks without `exit_code` trigger `GS_PASS_EXIT_CODE_MISMATCH`.
2. Decode failures and normalization diagnostics occupy different stages.
3. The assumed helper API (`minimalV2Wire`) does not exist.
4. Diagnostic ordering is controlled by `codePrecedence` map.

### GREEN step

1. R9 close report corrected.
2. One valid V2 test-document builder created.
3. One passing two-diagnostic ordering proof created.
4. One structural decode-failure test created proving `Err == nil`,
   `Success() == false`, no usable Document, and expected diagnostics.
   Caller-gating of Normalize on Decode.Success() documented.
5. Focused package tests pass.
6. Full-suite gate status recorded honestly.

### Commands

```bash
# Focused package suite
go test -count=1 ./internal/gatesummary/...

# Full suite (may time out)
go test -count=1 ./...
# If timeout: record as NOT VERIFIED with attribution to external
# execution budget; Go default is 10 minutes.

# Build
CGO_ENABLED=0 go build -trimpath -o bin/leamas ./cmd/leamas

# Repository gates
make gate
make factorize

# Genuinely cheap checks
git diff --check
./bin/leamas factory verify llm-friendly
```

## Acceptance Criteria

- [x] R9 close report status changed from `CLOSED` to
      `PARTIAL — test-design attempt reverted; objective not delivered`.
- [x] R9 close report records the close report file as the single
      changed artifact.
- [x] R9 close report attributes the 300-second timeout to the external
      execution budget, not Go's default 10-minute timeout.
- [x] One V2 test-document builder function created that always produces
      internally consistent checks (with correct `exit_code` per status).
      Valid helpers: `passCheckForTest`, `failWithoutExitForTest`,
      `failNonzeroForTest`, `skipCheckForTest`, `unavailableCheckForTest`.
      `failNonzeroForTest` panics on zero exit code.
- [x] One two-diagnostic ordering proof test that verifies complete
      diagnostic identities (Code and Path) appear in precedence order.
      `TestDiagnosticPrecedenceEndToEnd` verifies GS_DUPLICATE_CHECK_NAME (rank 15)
      precedes GS_PASS_EXIT_CODE_MISMATCH (rank 16).
- [x] One structural decode-failure test proving Decode returns
      `Success() == false`, `Err == nil`, no usable Document, and
      the expected wire-stage diagnostics.
      `TestStructuralDecodeRejection` verifies all four conditions.
- [x] The test documents that normalization is caller-gated on
      `Decode.Success()`. `TestCallerGatingBothBranches` exercises both branches.
- [x] Focused package tests pass: `go test -count=1 ./internal/gatesummary/...`
      VERIFIED: PASS (0.373s)
- [x] Build succeeds: `CGO_ENABLED=0 go build -trimpath -o bin/leamas ./cmd/leamas`
      VERIFIED: SUCCESS
- [x] Full-suite status recorded as VERIFIED, FAILED, or NOT VERIFIED
      with exact timeout attribution. NOT VERIFIED: external 300s timeout.
- [x] `make gate` and `make factorize` status recorded honestly (if timed out,
      NOT VERIFIED with exact timeout attribution). NOT VERIFIED: external 300s timeout.
- [x] New test file stays ≤ 64 KiB and ≤ 400 lines.
      VERIFIED: 299 lines < 400 lines.
- [x] No new files exceed LLM-friendliness limits.
      VERIFIED: LLM-friendly check passed on new files.

## Verification Commands

```bash
# Focused package suite
go test -count=1 -v ./internal/gatesummary/...

# Build
CGO_ENABLED=0 go build -trimpath -o bin/leamas ./cmd/leamas

# Full suite (with timeout attribution if applicable)
go test -count=1 ./...
# Record: VERIFIED | FAILED | NOT VERIFIED (external Ns timeout)

# Repository gates
make gate
make factorize
# Record: VERIFIED | NOT VERIFIED (external Ns timeout)
# If timeout: NOT VERIFIED — all N observed checks passed, but the
# external execution budget expired before final exit status captured.

# Genuinely cheap checks
git diff --check
./bin/leamas factory verify llm-friendly
```

## Reviewer Focus

- **Status honesty.** The R9 close report must not say `CLOSED` when
  the objective was not delivered.
- **Incremental scope.** One small test file, not another eight-file
  speculative batch.
- **Correct construction.** Pass checks must have `exit_code: 0`; skip
  checks must have `exit_code: null`.
- **Decode-stage contract.** Structural decode rejection: `Err == nil`,
  `Success() == false`, no usable Document, wire-stage diagnostics
  present. Normalization is caller-gated on `Decode.Success()`.
- **Precedence proof.** Two diagnostics must appear in the order
  dictated by `codePrecedence`, with Code and Path asserted.

## Close Report Stub

> **Summary:** Corrected R9 closure and established narrow contract
> reconnaissance for diagnostic ordering tests.
>
> **Files changed:**
> - `docs/close-reports/ACT-LEAMAS-GATE-SUMMARY-V2-EXACT-GEOMETRY-TESTS01-R9.md`
>   (status corrected, evidence corrected, timeout attributed)
> - `internal/gatesummary/*_test.go` (one small test file added)
>
> **Behavior changed:** None (test code only).
>
> **Verification:**
> - Focused package tests: VERIFIED
> - Build: VERIFIED
> - Full suite: VERIFIED | FAILED | NOT VERIFIED (external Ns timeout)
> - Documentation gates: VERIFIED
>
> **Closure:** PARTIAL — reconnaissance established; full diagnostic
> ordering matrix deferred to R10.
