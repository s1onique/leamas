# Close Report: ACT-LEAMAS-FACTORY-DUPCODE-V4-EXACT-GEOMETRY-TESTS01-R8

## Status: COMPLETE

## Parent ACT

- ACT-LEAMAS-FACTORY-DUPCODE-V4-EXACT-GEOMETRY-TESTS01
- (lineage: ACT-LEAMAS-FACTORY-DUPLICATE-CODE-GATE01)

## Summary

R8 corrects the four remaining defects identified in the R8 review of the
geometry ACT:

1. Stale summary status calling the semantic-tests child `PARTIAL`.
2. Path-escape guard using a string prefix (`strings.HasPrefix(rel, "..")`)
   that incorrectly rejects legitimate local names such as `..generated.go`.
3. Projection diagnostic specified as a "complete set" without multiset
   semantics; repeated clones like `repeat_a.go × 2` could be silently
   collapsed.
4. Position fields (`StartPos`/`EndPos`) labeled as "exact byte offsets" but
   the public `Occurrence` type does not expose those fields and the internal
   values are 0-based token offsets, not byte offsets.

This patch is documentary for the geometry ACT plus a small executable-contract
fix for the path-escape guard in `internal/factory/dupcode/baseline_verify.go`,
with regression tests added to lock the corrected contract.

## Files Changed

This patch touches **12 files** (matches `git status --porcelain`):

### Modified (2)

| File | Change |
|------|--------|
| `internal/factory/dupcode/baseline_verify.go` | Replace `strings.HasPrefix(rel, "..")` with `filepath.IsLocal(rel)` in `NormalizeOccurrencePath`; drop unused `strings` import; expand the doc comment. |
| `docs/close-reports/ACT-LEAMAS-FACTORY-DUPLICATE-CODE-GATE01.md` | R1 parent ACT updated for the R1 child ACT set (semantic-tests, geometry-tests, production, performance). Not modified by R8 itself; included for inventory completeness. |

### New (10)

| File | Change |
|------|--------|
| `internal/factory/dupcode/baseline_verify_test.go` | NEW: tests pinning `NormalizeOccurrencePath` (local, `..` prefix, exact outside-root fallback). RED established for `..generated.go`; GREEN after fix. |
| `docs/close-reports/ACT-LEAMAS-FACTORY-DUPCODE-V4-EXACT-GEOMETRY-TESTS01.md` | R8 corrections: status fix, `filepath.IsLocal` path-escape contract, multiset diagnostic, position semantics tied to `Occurrence`. |
| `docs/close-reports/ACT-LEAMAS-FACTORY-DUPCODE-V4-EXACT-GEOMETRY-TESTS01-R8.md` | This close report. |
| `docs/close-reports/ACT-LEAMAS-FACTORY-DUPCODE-V4-EXACT-SEMANTICS-TESTS01.md` | R1 child ACT: cardinality/multiplicity/validity/sortedness tests. Pre-existing untracked file; long summary line wrapped for LLM-friendliness. |
| `docs/close-reports/ACT-LEAMAS-FACTORY-DUPCODE-V4-EXACT-SEMANTICS-PRODUCTION01.md` | R1 child ACT: production correction owner. Pre-existing untracked file; long summary line wrapped. |
| `docs/close-reports/ACT-LEAMAS-FACTORY-DUPCODE-V4-ALL-PAIRS-MATERIALIZATION-PERFORMANCE01.md` | R1 child ACT: performance sibling. Pre-existing untracked file; long summary line wrapped. |
| `internal/factory/dupcode/v4_exact_semantics_test.go` | Trimmed to cardinality tests + helpers (216 lines, was 448). |
| `internal/factory/dupcode/v4_exact_semantics_bodies_test.go` | NEW: body-separation tests (TwoIndependentBodies, NoShadowSubFindings). |
| `internal/factory/dupcode/v4_exact_semantics_determinism_test.go` | NEW: determinism test. |
| `internal/factory/dupcode/v4_exact_semantics_ordering_test.go` | NEW: canonical-ordering test + groupOccurrencesByPath helper. |

### Note on scope

The geometry ACT change was scoped to the R8 corrections, but `make factorize`
also flagged pre-existing LLM-friendliness issues in three sibling ACTs and in
the exact-semantics test file. To keep the patch landed atomically and the
factorize gate green, those issues were addressed as small R1/R2 cleanups.
They are listed above under "Pre-existing untracked file" entries.

## Behavior Changed

### Geometry ACT

1. Summary no longer calls the semantic-tests ACT `PARTIAL`. It now states
   the semantic-tests ACT is COMPLETE as a red cardinality/multiplicity
   specification, and clarifies what it asserts vs. defers.
2. "Stable Path Normalization" section now:
   - explicitly forbids the `strings.HasPrefix(rel, "..")` pattern,
   - mandates `filepath.IsLocal(rel)` as a true path-component containment
     check,
   - shows the error-returning `normalizeFixturePath` shape,
   - accepts legitimate local names like `..generated.go`.
3. "Red-Reachability" section is now titled "Nonfatal Multiset Projection
   Diagnostic" and:
   - compares canonicalized projection multisets (or sorted slices),
   - enumerates `repeat_a.go × 2`, `repeat_b.go × 1` as a case where
     mathematical set semantics would be wrong,
   - lists the four independent reports (cardinality mismatch, missing
     instances, unexpected instances, field-level differences).
4. "Exact Boundaries" section now:
   - quotes the real public `Occurrence` type,
   - explicitly forbids equality on `StartPos`/`EndPos` against the public
     contract,
   - documents the internal `StartPos`/`EndPos` as 0-based token offsets (not
     byte offsets) per the comment in `internal/factory/dupcode/coalesce.go`,
   - restricts equality to `Path`, `StartLine`, `EndLine`, and
     `Finding.TokenCount`.
5. Files-to-Update, Scope, Closure Criteria, and diagnostic-field-list all
   updated to match the public-contract-only position semantics.

### `NormalizeOccurrencePath` (production code)

Before:
```go
func NormalizeOccurrencePath(root, p string) string {
    rel, err := filepath.Rel(root, p)
    if err == nil && !strings.HasPrefix(rel, "..") && !filepath.IsAbs(rel) {
        return filepath.ToSlash(rel)
    }
    return filepath.ToSlash(p)
}
```

After:
```go
func NormalizeOccurrencePath(root, p string) string {
    rel, err := filepath.Rel(root, p)
    if err == nil && filepath.IsLocal(rel) {
        return filepath.ToSlash(rel)
    }
    return filepath.ToSlash(p)
}
```

`filepath.IsLocal` lexically guarantees the relative path is nonempty,
non-absolute, and contains no parent-directory component. The string-prefix
guard `strings.HasPrefix(rel, "..")` is removed (and the `strings` import is
no longer needed) because it would have rejected legitimate local names such
as `..generated.go` that happen to start with `..` but are not escapes.

The `error` return shape recommended by the verdict (e.g.
`normalizeFixturePath(fixtureRoot, occurrencePath string) (string, error)`)
is captured in the geometry ACT as the contract for test-side helpers; the
production-side `NormalizeOccurrencePath` keeps its string-returning shape
because all its callers (only `GenerateCanonicalBaseline`) treat the
fallback-to-original as a recoverable baseline-encoding decision rather than
a hard error. The boundary guarantee — escape detection is correct — is now
test-pinned regardless of return shape.

## Verification Evidence

### `make factorize`

```
Running factory factorize...
  agent-context: OK
  docs: OK
  doctrine: OK
  doctrine-agent-contracts: OK
  domain-boundaries: OK
  dupcode: OK
  dupcode-baseline: OK
  exec-gate: OK
  executable-contract-first: OK
  forbidden-patterns: OK
  git-hooks: OK
  language: OK
  llm-friendly: OK
  static-binary: OK
  tooling-boundaries: OK

*** FACTORIZE PASSED ***
```

### `make gate`

```
*** GATE FAILED ***
make: *** [gate] Error 1
```

**Status: FAILED AS EXPECTED.** The gate fails because `go test ./...` fails on
the 6 `TestV4ExactSemantics_*` tests that intentionally expose production
defects; this is the documented state of
`ACT-LEAMAS-FACTORY-DUPCODE-V4-EXACT-SEMANTICS-TESTS01`. The test for
`Determinism` continues to pass. The gate cannot close until the exact-geometry
red specification is implemented and the subsequent production ACT makes both
the semantic and geometry contracts green.

```
=== RUN   TestV4ExactSemantics_OneMaximalClone
--- FAIL: TestV4ExactSemantics_OneMaximalClone (1.96s)
    v4_exact_semantics_test.go:57: EXACT CONTRACT FAIL: expected exactly 1 finding, got 334
=== RUN   TestV4ExactSemantics_RepeatedMultiplicity
--- FAIL: TestV4ExactSemantics_RepeatedMultiplicity (0.19s)
    v4_exact_semantics_test.go:115: EXACT CONTRACT FAIL: expected exactly 1 finding, got 85
=== RUN   TestV4ExactSemantics_NWayClone
--- FAIL: TestV4ExactSemantics_NWayClone (15.42s)
    v4_exact_semantics_test.go:185: EXACT CONTRACT FAIL: expected exactly 1 finding, got 334
=== RUN   TestV4ExactSemantics_TwoIndependentBodies
--- FAIL: TestV4ExactSemantics_TwoIndependentBodies (0.03s)
    v4_exact_semantics_bodies_test.go:45: EXACT CONTRACT FAIL: expected exactly 2 findings, got 15
=== RUN   TestV4ExactSemantics_NoShadowSubFindings
--- FAIL: TestV4ExactSemantics_NoShadowSubFindings (2.32s)
    v4_exact_semantics_bodies_test.go:108: EXACT CONTRACT FAIL: expected exactly 1 above-threshold cross-file finding, got 334
=== RUN   TestV4ExactSemantics_Determinism
--- PASS: TestV4ExactSemantics_Determinism
=== RUN   TestV4ExactSemantics_CanonicalOrdering
--- FAIL: TestV4ExactSemantics_CanonicalOrdering (0.21s)
    v4_exact_semantics_ordering_test.go:54: EXACT CONTRACT FAIL: expected exactly 1 finding, got 85
```

### `go vet ./...`

Exit 0, no diagnostics.

### `CGO_ENABLED=0 go build -trimpath -o bin/leamas ./cmd/leamas`

Exit 0, binary produced at `bin/leamas`.

### Targeted RED → GREEN for the path-escape fix

RED (before production fix):

```
=== RUN   TestNormalizeOccurrencePath_LocalNameStartingWithDotDot
    baseline_verify_test.go:78: NormalizeOccurrencePath("/tmp/.../repo", "/tmp/.../repo/..generated.go") = "/tmp/.../repo/..generated.go", want "..generated.go" (local file whose name starts with ".." must be accepted)
--- FAIL: TestNormalizeOccurrencePath_LocalNameStartingWithDotDot (0.00s)
```

GREEN (after production fix):

```
=== RUN   TestNormalizeOccurrencePath_LocalPaths
=== RUN   TestNormalizeOccurrencePath_LocalPaths/sibling
=== RUN   TestNormalizeOccurrencePath_LocalPaths/nested
=== RUN   TestNormalizeOccurrencePath_LocalPaths/deeper_nesting
--- PASS: TestNormalizeOccurrencePath_LocalPaths (0.00s)
    --- PASS: TestNormalizeOccurrencePath_LocalPaths/sibling (0.00s)
    --- PASS: TestNormalizeOccurrencePath_LocalPaths/nested (0.00s)
    --- PASS: TestNormalizeOccurrencePath_LocalPaths/deeper_nesting (0.00s)
=== RUN   TestNormalizeOccurrencePath_LocalNameStartingWithDotDot
--- PASS: TestNormalizeOccurrencePath_LocalNameStartingWithDotDot (0.00s)
=== RUN   TestNormalizeOccurrencePath_OutsideRootFallback
=== RUN   TestNormalizeOccurrencePath_OutsideRootFallback/sibling-of-root_(Rel_returns_../...)
=== RUN   TestNormalizeOccurrencePath_OutsideRootFallback/absolute_path_outside_root
--- PASS: TestNormalizeOccurrencePath_OutsideRootFallback (0.00s)
    --- PASS: TestNormalizeOccurrencePath_OutsideRootFallback/sibling-of-root_(Rel_returns_../...) (0.00s)
    --- PASS: TestNormalizeOccurrencePath_OutsideRootFallback/absolute_path_outside_root (0.00s)
PASS
ok  	github.com/s1onique/leamas/internal/factory/dupcode	0.410s
```

The outside-root fallback is asserted exactly (`got != filepath.ToSlash(p)`
fails the test), not merely "not local"; the `..generated.go` regression
case passes; the three local-path table cases pass.

## Final Disposition (R8 Review Items)

| Item | Disposition |
|------|-------------|
| Geometry summary status | Corrected: PARTIAL → COMPLETE as a red cardinality/multiplicity specification, with the listed assertions enumerated. |
| Path containment implementation | Corrected in both production code (`filepath.IsLocal`) and ACT document. Tests pin the corrected contract. |
| Complete-set matching semantics | Corrected: now specified as multiset / canonicalized sorted-slice comparison, with the four independent diagnostic reports listed. |
| Position-field meaning | Corrected: tied to the public `Occurrence` contract (`Path`, `StartLine`, `EndLine`); `StartPos`/`EndPos` documented as 0-based token offsets on the internal types only. |

## Geometry ACT Lifecycle

The geometry ACT remains OPEN because the exact projection helpers,
independent token-count constants, stable path projections, and nonfatal
multiset diagnostics have not yet been implemented.

It can become COMPLETE as a red specification before production is
corrected. The production ACT owns turning those installed assertions
green.

## Skipped / Deferred

- Implementation of the geometry ACT as a red specification
  (`ACT-LEAMAS-FACTORY-DUPCODE-V4-EXACT-GEOMETRY-TESTS01`) is the
  next executable ACT, not production. It must land before
  `ACT-LEAMAS-FACTORY-DUPCODE-V4-EXACT-SEMANTICS-PRODUCTION01`.
- Production correction (turning the RED semantic and geometry tests
  green) is deferred to
  `ACT-LEAMAS-FACTORY-DUPCODE-V4-EXACT-SEMANTICS-PRODUCTION01`.
- All-pairs materialization performance work is deferred to
  `ACT-LEAMAS-FACTORY-DUPCODE-V4-ALL-PAIRS-MATERIALIZATION-PERFORMANCE01`.
- `go test ./...` and `make gate` failures from the 6 RED
  `TestV4ExactSemantics_*` tests are not skipped or weakened; they remain
  as regression detection for the production correction ACT.

## Follow-up ACTs (dependency order)

1. **`ACT-LEAMAS-FACTORY-DUPCODE-V4-EXACT-GEOMETRY-TESTS01`** — install
   the exact-projection red specification (next executable work).
   Depends on: the corrected geometry ACT (this R8 patch) + the
   cardinality/validity tests from `...-EXACT-SEMANTICS-TESTS01`.
2. **`ACT-LEAMAS-FACTORY-DUPCODE-V4-EXACT-SEMANTICS-PRODUCTION01`** —
   turn the RED semantic and geometry tests green.
   Depends on: this ACT (geometry) + the semantic-tests ACT.
3. **`ACT-LEAMAS-FACTORY-DUPCODE-V4-ALL-PAIRS-MATERIALIZATION-PERFORMANCE01`** —
   measure and reduce unnecessary all-pairs materialization.
   Depends on: production ACT completing first.

## Closed At

2026-07-16T09:05:00+03:00