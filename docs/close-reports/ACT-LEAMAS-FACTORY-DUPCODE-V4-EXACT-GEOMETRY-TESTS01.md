# ACT-LEAMAS-FACTORY-DUPCODE-V4-EXACT-GEOMETRY-TESTS01

## Status

**SUPERSEDED — see correction report.**

This ACT's closure record is now amended by
ACT-LEAMAS-FACTORY-DUPCODE-V4-EXACT-GEOMETRY-TESTS01-CORRECTION02, which
supersedes CORRECTION01. The latest correction freezes literal
fingerprint-first finding-order keys, routes internal assertions
through the production-owned merge seam shared with public findings,
makes internal canonicalization total, uses exact path equality, and
provides complete final gate evidence.

See CORRECTION02 for the executable assertions, exact ordering-key
literals, final evidence, and final file inventory.

This file is retained (modified tracked) so the original close intent
remains visible alongside the correction. The status line below
reflects the original closure; the correction report owns the
corrected record.

## Status (original closure)

COMPLETE (red exact-geometry specification)

## Parent ACT

- ACT-LEAMAS-FACTORY-DUPCODE-V4-EXACT-SEMANTICS-TESTS01
- (lineage: ACT-LEAMAS-FACTORY-DUPLICATE-CODE-GATE01)

## Summary

The original ACT installed the exact geometry contract tests for the
V4 duplicate-code detector as a red specification. It froze the exact
public projection (Path, StartLine, EndLine, TokenCount), an internal
token-span projection (Path, StartPos, EndPos), stable fixture-root-
relative path projection via `filepath.IsLocal`, multiset comparison,
and an independent oracle (`countIndependentTokensForSpan`).

## Files originally changed

| File | Original change |
|------|-----------------|
| `v4_exact_geometry_support_test.go` | NEW: projection types, frozen token-count constants, normalizeFixturePath, canonicalize helpers, assertExactFindingProjection, countIndependentTokensForSpan |
| `v4_exact_geometry_bodies_test.go` | NEW: OneMaximalClone, RepeatedMultiplicity, NWayClone, TwoIndependentBodies, NoShadowSubFindings |
| `v4_exact_geometry_determinism_test.go` | NEW: Determinism with 5 repeated runs and raw + canonical comparison |
| `v4_exact_geometry_ordering_test.go` | NEW: CanonicalOrdering with prev/curr projection printing on inversion |
| this close report | NEW |

All paths above are under `internal/factory/dupcode/` except the close
report.

## Amendment file inventory

The amendment by ACT-LEAMAS-FACTORY-DUPCODE-V4-EXACT-GEOMETRY-TESTS01-
CORRECTION01 adds the following executable assertions and split
helpers:

| File | Amendment |
|------|-----------|
| `v4_exact_geometry_internal_test.go` | NEW: 5 grouped internal StartPos / EndPos tests |
| `v4_exact_geometry_internal_helpers_test.go` | NEW: v4PipelineInternal lower-level orchestrator |
| `v4_exact_geometry_diagnostics_test.go` | NEW: multiplicity diagnostic helpers (sorted keys, expected N / actual M) |
| `v4_exact_geometry_path_test.go` | NEW: TestNormalizeFixturePath_Contract |
| `v4_exact_geometry_determinism_test.go` | UPDATED: Internal Determinism; raw + multiset views separated |
| `v4_exact_geometry_ordering_test.go` | UPDATED: CanonicalFindingOrdering + CanonicalOccurrenceOrdering |
| `v4_exact_geometry_bodies_test.go` | UPDATED: removed misleading `findFindingByTokenCount` |
| `v4_exact_geometry_support_test.go` | UPDATED: grouped internal projection types, assertExactInternalFindingGeometry |
| this close report | UPDATED |
| `*-CORRECTION01.md` close report | NEW |

All paths above are under `internal/factory/dupcode/` except the close
reports (under `docs/close-reports/`).

## Verification status (as amended)

- `make factorize`: PASS
- `go vet ./...`: PASS
- Static build: PASS (`CGO_ENABLED=0 go build -trimpath -o bin/leamas ./cmd/leamas`)
- `go test ./internal/factory/dupcode -run '^TestV4ExactGeometry'`:
  RED for body / internal / ordering tests (production geometry
  defects); PASS for Determinism, Internal Determinism, and the path
  contract test
- `make gate`: FAIL only from the documented red tests

The correction report owns the corrected, current evidence.

## Skipped / Deferred

Production correction is deferred to
`ACT-LEAMAS-FACTORY-DUPCODE-V4-EXACT-SEMANTICS-PRODUCTION01`.

## Ownership transfer

This ACT is amended to a corrected red specification by
ACT-LEAMAS-FACTORY-DUPCODE-V4-EXACT-GEOMETRY-TESTS01-CORRECTION02.
The subsequent production ACT remains responsible for turning the red
tests green.

## Closed At (original)

2026-07-16T09:53:00+03:00

## Amended At

2026-07-16T11:19:00+03:00 (see CORRECTION02)
