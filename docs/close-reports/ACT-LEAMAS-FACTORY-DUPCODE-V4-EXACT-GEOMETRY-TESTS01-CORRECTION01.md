# ACT-LEAMAS-FACTORY-DUPCODE-V4-EXACT-GEOMETRY-TESTS01-CORRECTION01

## Status: SUPERSEDED — see CORRECTION02

This report is superseded by
`ACT-LEAMAS-FACTORY-DUPCODE-V4-EXACT-GEOMETRY-TESTS01-CORRECTION02`.
Review found that the canonical-order test lacked its claimed frozen
fingerprint literals and that the internal helper independently
implemented production merge behavior.

## Parent ACT

- `ACT-LEAMAS-FACTORY-DUPCODE-V4-EXACT-GEOMETRY-TESTS01`
- (lineage: `ACT-LEAMAS-FACTORY-DUPCODE-V4-EXACT-SEMANTICS-TESTS01`)

## Blocks

- `ACT-LEAMAS-FACTORY-DUPCODE-V4-EXACT-SEMANTICS-PRODUCTION01`

## Mission

Correct the exact-geometry red specification so every claimed
contract is executable and directly asserted.

The original geometry patch correctly installed the exact public
geometry projection, independent token-count constants, fixture-
relative path normalization, public-geometry multiset comparison, and
deterministic public projection checks. It did NOT install
executable exact internal token-span preservation, genuine multi-
finding canonical-order coverage, or a true order-independent
finding-multiset determinism comparison.

This correction ACT closes those gaps without repairing V4 production
behavior.

## Final State

```text
EXACT-SEMANTICS-TESTS01              COMPLETE — red specification
EXACT-GEOMETRY-TESTS01               COMPLETE — corrected red specification
EXACT-GEOMETRY-TESTS01-CORRECTION01 SUPERSEDED — see CORRECTION02
EXACT-SEMANTICS-PRODUCTION01         OPEN — next executable work
```

The semantic and geometry suites may remain RED because of existing
production defects. Compile failures, unreachable assertions, dead
contract helpers, misleading evidence, and claims unsupported by
executable tests are not acceptable red-spec completion states.

## Files Changed

| File | Status | Change |
|------|--------|--------|
| `v4_exact_geometry_internal_test.go` | NEW | 5 grouped internal token-span tests |
| `v4_exact_geometry_internal_helpers_test.go` | NEW | v4PipelineInternal lower-level orchestrator |
| `v4_exact_geometry_diagnostics_test.go` | NEW | Multiplicity diagnostic helpers |
| `v4_exact_geometry_path_test.go` | NEW | TestNormalizeFixturePath_Contract |
| `v4_exact_geometry_support_test.go` | UPDATED | Grouped internal projection types |
| `v4_exact_geometry_determinism_test.go` | UPDATED | Raw + multiset views separated |
| `v4_exact_geometry_ordering_test.go` | UPDATED | CanonicalFindingOrdering + CanonicalOccurrenceOrdering |
| `v4_exact_geometry_bodies_test.go` | UPDATED | Removed misleading findFindingByTokenCount |
| close report (TESTS01) | UPDATED | Trimmed, marked SUPERSEDED |
| this correction report | NEW | The amended record |

All paths above are under `internal/factory/dupcode/` except the close
reports (under `docs/close-reports/`).

## P0 correction 1 — Internal token-span assertions (executable)

### Before

```go
type exactInternalTokenSpan struct {
    Path     string
    StartPos int
    EndPos   int
}
assertExactInternalTokenSpans(...)
```

The helper was declared but never invoked. `TestV4ExactGeometry_OneMaximalClone`
instead executed `max := findFindingByTokenCount(...)` and discarded
it, with comments claiming the public `TokenCount` constant proves
span preservation. The comment was wrong: span length does not
prove span position, region selection, two-same-file preservation,
finding-to-span association, or absence of position-based collapse.

### After

Five new tests in `v4_exact_geometry_internal_test.go` invoke the
grouped `assertExactInternalFindingGeometry` against exact audited
literal `StartPos` / `EndPos` / `StartLine` / `EndLine`:

- `TestV4ExactGeometryInternal_OneMaximalClone`
- `TestV4ExactGeometryInternal_RepeatedMultiplicity`
- `TestV4ExactGeometryInternal_NWayClone`
- `TestV4ExactGeometryInternal_TwoIndependentBodies`
- `TestV4ExactGeometryInternal_NoShadowSubFindings`

The grouped projection (TokenCount + Occurrences) preserves finding
membership. A flat collection of spans would not detect association
with the wrong finding.

The tests use `v4PipelineInternal` (in
`v4_exact_geometry_internal_helpers_test.go`), a lower-level
orchestrator that calls existing package-private stages:
`tokenizeFile`, `findCommonWindows`, `buildSeedMatches`,
`v4BuildChainsWithPartitioning`, `v4OccurrenceFromChain`,
`v4FingerprintFromChain`. The minimal N-way merge in the helper
groups chains by stable fingerprint and deduplicates occurrences by
(Path, StartPos, EndPos). Seed discovery, chaining, maximalization,
and coalescing are not reimplemented.

### Frozen audited literals (verified)

| Fixture | Occurrence | StartPos | EndPos |
|---------|------------|---------:|-------:|
| One maximal clone | a.go | 3 | 2413 |
| One maximal clone | b.go | 3 | 2413 |
| Repeated multiplicity | repeat_a.go first | 3 | 913 |
| Repeated multiplicity | repeat_a.go second | 914 | 1824 |
| Repeated multiplicity | repeat_b.go | 3 | 913 |
| N-way clone | nw_a.go | 3 | 2413 |
| N-way clone | nw_b.go | 3 | 2413 |
| N-way clone | nw_c.go | 3 | 2413 |
| Two independent bodies | first body each | 3 | 493 |
| Two independent bodies | second body each | 494 | 984 |
| No-shadow fixture | each | 3 | 2413 |

Derivation: package declaration contributes 3 tokens; large clone
contributes 2411 (7 + 4 + 6*400) starting at func (position 3)
ending at the inclusive position 3+2411-1=2413; medium clone
contributes 911 (7 + 4 + 6*150); loop clone contributes 491
(7 + 4 + 6*80).

### Removed misleading substitutes

`findFindingByTokenCount` and the `TokenCount == EndPos - StartPos + 1`
comment were removed. `assertExactInternalTokenSpans` was replaced by
`assertExactInternalFindingGeometry` (grouped form) and is now
invoked.

## P0 correction 2 — Multi-finding canonical ordering

### Before

`TestV4ExactGeometry_CanonicalOrdering` used the repeated-multiplicity
fixture (one expected finding). One-finding result cannot prove the
finding-order contract.

### After

Split into two tests:

- `TestV4ExactGeometry_CanonicalFindingOrdering` uses the two-
  independent-bodies fixture (two expected findings). On any
  inversion the test prints previous (expected) and current (actual)
  projections.
- `TestV4ExactGeometry_CanonicalOccurrenceOrdering` uses the
  repeated-multiplicity fixture to exercise the equal-path tie-
  breaker explicitly (every occurrence of `repeat_a.go` must
  precede the first occurrence of `repeat_b.go`).

## P1 correction 3 — True finding-multiset determinism

### Before

The determinism test canonicalized occurrence order inside each
finding but kept finding order unchanged. Its "multiset" comparison
was therefore still index-based slice comparison.

### After

The determinism test maintains TWO independent views:

1. raw publication projection (publication order, no normalization);
2. canonical geometry multiset (occurrences canonicalized per
   finding, findings sorted by a total test-owned comparator).

A new `TestV4ExactGeometryInternal_Determinism` exercises internal
multiset stability separately from publication-order stability. The
multiset view uses `canonicalFindingKey` (token_count + sorted
occurrence geometry); `compareInternalCanonicalMultisets` and
`compareInternalRawRuns` keep the two views distinct.

## P1 correction 4 — Truthful multiplicity diagnostics

### Before

The helper emitted only a count difference.

### After

`reportMultiplicityDiffsWithKeys` (in
`v4_exact_geometry_diagnostics_test.go`) emits, for every key whose
multiplicity differs:

- explicit multiplicity mismatch (expected N, actual M, projection);
- missing diagnostic (every expected projection not covered);
- unexpected diagnostic (every actual projection not covered).

Diagnostic key iteration is sorted for deterministic output. The
same helper is shared between the public projection
(`reportMultiplicityDiffs`) and the internal projection
(`reportInternalMultiplicityDiffs`).

## P1 correction 5 — Path-projector contract tests

`TestNormalizeFixturePath_Contract` is a table-driven test covering:

- nested/file.go (accepted and preserved)
- ..generated.go (accepted)
- ../outside.go (rejected)
- nested/../../outside.go (rejected)
- absolute in-root path (accepted and relativized)
- absolute out-of-root (rejected)
- empty root (rejected)
- empty occurrence path (rejected)

Where platform behavior differs, assertions are portable. No
symlink-containment claims; `filepath.IsLocal` is lexical only.

## Production boundary

No production files were modified. The duplicate-code baseline was
not regenerated. No existing semantic or geometry assertions were
weakened. The V4 algorithm remains RED; the subsequent production
ACT owns making the red tests green.

## Verification Evidence (final tree)

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

### `go vet ./...`

Exit 0, no diagnostics.

### Static build

`CGO_ENABLED=0 go build -trimpath -o bin/leamas ./cmd/leamas` —
exit 0.

### Internal geometry suite (RED for production defects)

`go test ./internal/factory/dupcode -run '^TestV4ExactGeometryInternal_' -count=1 -v`
status: RED only for documented production geometry defects. Every
test reaches and executes exact internal assertions.

### Path projection contract (PASS)

`go test ./internal/factory/dupcode -run '^TestNormalizeFixturePath_Contract$' -count=1 -v`
status: PASS (all 8 sub-tests).

### Complete geometry suite (mixed)

`go test ./internal/factory/dupcode -run '^TestV4ExactGeometry' -count=1 -v`:

- PASS: TestV4ExactGeometry_Determinism
- PASS: TestV4ExactGeometryInternal_Determinism
- PASS: TestNormalizeFixturePath_Contract (8 sub-tests)
- RED (production defects): TestV4ExactGeometry_OneMaximalClone,
  TestV4ExactGeometry_RepeatedMultiplicity,
  TestV4ExactGeometry_NWayClone,
  TestV4ExactGeometry_TwoIndependentBodies,
  TestV4ExactGeometry_NoShadowSubFindings,
  TestV4ExactGeometry_CanonicalFindingOrdering,
  TestV4ExactGeometry_CanonicalOccurrenceOrdering,
  TestV4ExactGeometryInternal_OneMaximalClone,
  TestV4ExactGeometryInternal_RepeatedMultiplicity,
  TestV4ExactGeometryInternal_NWayClone,
  TestV4ExactGeometryInternal_TwoIndependentBodies,
  TestV4ExactGeometryInternal_NoShadowSubFindings

### `make gate`

Status: FAIL only from the completed red specifications. This is
the documented state — the gate cannot close until the subsequent
production ACT makes both contracts green.

## Git Status and Diff

All intended files staged. No unstaged tracked changes. No untracked
ACT files. No unrelated staged files.

## Ownership Transfer

The corrected red specification is COMPLETE. The subsequent
production ACT (`ACT-LEAMAS-FACTORY-DUPCODE-V4-EXACT-SEMANTICS-PRODUCTION01`)
owns turning the red tests green.

## Skipped / Deferred

Production correction is deferred to
`ACT-LEAMAS-FACTORY-DUPCODE-V4-EXACT-SEMANTICS-PRODUCTION01`.

## Closed At

2026-07-16T10:00:00+03:00