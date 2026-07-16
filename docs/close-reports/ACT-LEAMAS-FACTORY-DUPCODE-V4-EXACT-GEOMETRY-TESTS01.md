# ACT-LEAMAS-FACTORY-DUPCODE-V4-EXACT-GEOMETRY-TESTS01

## Status: OPEN

## Parent ACT
- ACT-LEAMAS-FACTORY-DUPCODE-V4-EXACT-SEMANTICS-TESTS01

## Summary

Explicit ownership ACT for exact geometry assertion tests for the V4 duplicate-code
detector. The exact-semantics test ACT is COMPLETE as a red cardinality and
multiplicity specification. It asserts cardinality, multiplicity, selected
boundary-validity checks, and sortedness, but does not assert exact fixture
projections.

This ACT specifies exact expected geometry BEFORE production correction so
production is developed against both cardinality and exact boundary contracts.

## Dependency Direction

```
EXACT-SEMANTICS-TESTS01
        ↓
EXACT-GEOMETRY-TESTS01 (this ACT)
        ↓
EXACT-SEMANTICS-PRODUCTION01
        ↓
ALL-PAIRS-MATERIALIZATION-PERFORMANCE01
```

The geometry ACT depends on the semantic-tests ACT only. It does NOT depend on production correction.
Production depends on both test contracts. Performance depends on production.

## Test Requirements

### Exact Boundaries (with position fields)
Geometry ACT must freeze the real public/internal model contract rather than
reinterpret field names.

The public `Occurrence` type exposed by the detector currently declares only:

```go
type Occurrence struct {
    Path      string
    StartLine int
    EndLine   int
}
```

It does NOT expose `StartPos` or `EndPos`. The geometry ACT therefore MUST NOT
assert equality on `StartPos`/`EndPos` against `Occurrence`; those fields are
not part of the public contract.

The internal `maximalOccurrence` (and `rawWindow`) types used by V4 carry
`StartPos` and `EndPos` as **0-based token offsets** into the per-file
token slice produced by `go/scanner`, NOT byte offsets. (See
`internal/factory/dupcode/coalesce.go`: `// StartPos and EndPos are token
offsets (0-based).`) They are an internal deduplication coordinate, not a
public contract, and they intentionally do not survive projection from
`maximalOccurrence` to the public `Occurrence`.

For each contract test, the following must be equality-checked against the
public `Occurrence` plus the enclosing `Finding`:

```text
- Path       (fixture-root-relative, slash-normalized, deterministic)
- StartLine  (exact, 1-based per go/token FileSet)
- EndLine    (exact, 1-based per go/token FileSet)
- TokenCount (exact, on Finding)
```

If StartPos/EndPos testing is later added (e.g. via a stable-internal
projection), it must:

* Be documented as 0-based token offsets into the file's normalized token
  slice, not byte offsets.
* Not assume those fields exist on the public `Occurrence` until that type
  is extended.

### Stable Path Normalization (Implementation Contract)

Containment must be a true path-component check, not a string-prefix test.
`strings.HasPrefix(rel, "..")` is wrong: it rejects legitimate local names
whose names happen to start with `..` (e.g. `..generated.go`). Use
`filepath.IsLocal`, which guarantees lexically that the path is nonempty,
non-absolute, and contains no parent-directory component.

```go
func normalizeFixturePath(fixtureRoot, occurrencePath string) (string, error) {
    rel, err := filepath.Rel(fixtureRoot, occurrencePath)
    if err != nil {
        return "", fmt.Errorf("make fixture-relative path: %w", err)
    }
    if !filepath.IsLocal(rel) {
        return "", fmt.Errorf("occurrence path escapes fixture root: %q", rel)
    }
    return filepath.ToSlash(rel), nil
}
```

Returning an error is preferable to returning `""`, which loses the reason
for rejection and introduces an artificial projection value. `filepath.Rel`
legitimately returns paths such as `../b/c` when the target is outside the
base; `filepath.ToSlash` performs the required platform-separator
normalization.

Expected values are `a.go`, `repeat_a.go`, and similar stable names — never the
machine-specific temporary directory path produced by `t.TempDir()`. Paths that
escape the fixture root (e.g. via `..`) must be rejected, while names such
as `..generated.go` that are local single-name paths must be accepted.

### Independent Token-Count Oracle
Expected token counts MUST NOT be derived from the same production normalization
or tokenization path under test. Acceptable oracles:

* Fixed audited constants for immutable fixture literals (preferred).
* An independent lexical oracle (a different tokenizer implementation).
* A fixture builder that returns both source and independently constructed
  expected token metadata in one call.

### No-Shadow Maximality
Replace `TokenCount > 400` with exact equality against expected maximal token
count, proving the sole finding equals the complete maximal clone.

### Red-Reachability: Nonfatal Multiset Projection Diagnostic
Fatally failing on cardinality means geometry assertions never run while
production over-emits (Go's `Fatalf` invokes `FailNow`, which halts the test
goroutine). Use nonfatal **multiset** projection comparison:

```text
expected projection multiset
  vs.
all actual normalized projections
```

The diagnostic compares **canonicalized projection multisets** (or
equivalently, sorted slices), not mathematical sets. Multiplicity matters
because repeated clones intentionally require:

```text
repeat_a.go × 2
repeat_b.go × 1
```

A set-based implementation could accidentally collapse repeated expected
or actual records, hiding real production defects.

The diagnostic independently reports:

* Cardinality mismatch (counts of canonicalized records).
* Missing projection instances (expected records not present in actual).
* Unexpected projection instances (actual records not present in expected).
* Field-level differences for matched instances (Path, StartLine, EndLine,
  TokenCount). `StartPos`/`EndPos` are reported only if and when the public
  `Occurrence` type is extended to expose them; until then, equality is
  constrained to the public fields.

This allows geometry assertions to execute without silently filtering away
production defects. Do NOT heuristically select a candidate by "largest token
count"; that would encode current implementation behavior into the oracle.

## Files to Update

- `internal/factory/dupcode/v4_exact_semantics_test.go`
  - Define `assertExactFindingProjection` helper operating on the public
    `Occurrence` (`Path`, `StartLine`, `EndLine`) plus `Finding.TokenCount`.
    Do not introduce assertions on `StartPos`/`EndPos` against the public
    `Occurrence`; those fields are not on the contract surface.
  - Call helper from all six geometry-bearing semantic tests
  - Use stable, fixture-root-relative path projection
  - Pair every cardinality test with a nonfatal geometry diagnostic
  - Source expected token counts from an independent oracle, not the SUT path

## Scope

1. Derive exact fixture geometry independently of detector output
2. Call `assertExactFindingProjection` from OneMaximalClone, RepeatedMultiplicity, NWayClone, NoShadowSubFindings
3. Compare complete two-finding projection for independent bodies
4. Replace `TokenCount > 400` with exact token-count equality
5. Make canonical ordering an exact expected-slice comparison
6. Use stable, fixture-root-relative path projection
7. Replace `Fatalf` cardinality gates so geometry assertions run during red
8. Keep position equality anchored to the fields actually declared on
   `Occurrence` (`StartLine`, `EndLine`); do not silently re-read token
   positions as byte offsets

## Closure Criteria (Geometry ACT COMPLETE)

The geometry ACT is COMPLETE when:

1. Exact independently-derived projections are implemented in the test file
2. All required exact equality assertions execute (whether red or green)
3. Any failures are attributable to the documented production gaps, not to fixture or assertion defects
4. The exact projection helper (`assertExactFindingProjection` or equivalent) is in place and wired into **all six geometry-bearing semantic tests** (excluding only `Determinism`):
   - `TestV4ExactSemantics_OneMaximalClone`
   - `TestV4ExactSemantics_RepeatedMultiplicity`
   - `TestV4ExactSemantics_NWayClone`
   - `TestV4ExactSemantics_TwoIndependentBodies` (complete two-finding projection)
   - `TestV4ExactSemantics_NoShadowSubFindings`
   - `TestV4ExactSemantics_CanonicalOrdering` (exact expected slice comparison)
5. Token-count and boundary equality assertions are in place
6. Position equality uses the public `Occurrence` fields (`StartLine`,
   `EndLine`); internal `StartPos`/`EndPos` are NOT asserted on the public
   contract
7. Stable path projection uses `filepath.IsLocal` containment and avoids
   machine-specific temp paths; legitimate local names such as
   `..generated.go` are accepted
8. Cardinality gated nonfatally so geometry assertions execute during red,
   and the multiset projection diagnostic independently reports cardinality,
   missing instances, unexpected instances, and field-level differences

## Production ACT OWNERSHIP

The production ACT owns turning all exact tests green:

- `ACT-LEAMAS-FACTORY-DUPCODE-V4-EXACT-SEMANTICS-PRODUCTION01` COMPLETE means:
  - All 7 exact semantic tests pass
  - `go test ./...` green
  - `make gate` green
  - Geometry ACT installation matches corrected behavior

## Created At

2026-07-16T08:08:00+03:00
