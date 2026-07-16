# ACT-LEAMAS-FACTORY-DUPCODE-V4-REGION-BOUNDED-CHAIN-CONSTRUCTION02

## Status: PARTIAL — chain-construction rewrites pending

`ACT-LEAMAS-FACTORY-DUPCODE-V4-REGION-BOUNDED-CHAIN-CONSTRUCTION02`
is **PARTIAL**. This ACT applied every P0 production-correctness
correction called for in its task spec and routed both
`CheckRepo` and `v4PipelineInternal` through a single production
seam (`v4BuildInternalFindings`) that performs region ownership
filtering, region-annotated match generation, global
chain-pair/offset partitioning, maximal chain construction,
region-aware shadow suppression, occurrence extraction with
invariant enforcement, N-way merge, and deterministic finding
ordering. The Partial-patch focused tests (21 PASS / 0 RED) and
the occurrence-identity focused tests (9 PASS / 0 RED) all pass.
The exact-semantics contract `TwoIndependentBodies` now passes,
and the determinism contract passes.

The remaining RED contracts (17 of 21) require a deeper rewrite of
pair-evidence materialization and merge. The semantic, public-geometry,
and internal-geometry failures are distinct contract classes: only
`TwoIndependentBodies` and the three determinism contracts were green at
this checkpoint. Pair candidates that share a seed or current
`StableFingerprint` but have different canonical geometry are unresolved
geometry conflicts, not evidence that they should merge. The closure
criteria requiring `21 PASS / 0 FAIL` is not satisfied; this ACT remains
PARTIAL pending exact-content pair evidence and connected-component
materialization.

## Honest accounting

```text
21 exact contracts (7 exact-semantic, 8 exact public-geometry,
                  6 exact internal-geometry)
Exact semantics:       2 PASS / 5 RED  (TwoIndependentBodies, Determinism)
Exact public geometry: 1 PASS / 7 RED  (Determinism)
Exact internal geom:   1 PASS / 5 RED  (Determinism)
Focused partial-patch tests: 21 PASS / 0 RED
Focused occurrence tests:     9 PASS / 0 RED
```

Total exact contracts: **4 PASS / 17 RED**. The 5 RED exact-semantic
contracts are: `OneMaximalClone`, `RepeatedMultiplicity`,
`NWayClone`, `NoShadowSubFindings`, `CanonicalOrdering`. The 12
RED geometry contracts are the public- and internal-geometry
counterparts of the same defects, plus `TwoIndependentBodies`
(internal-geometry variant).

`TestV4ExactSemantics_TwoIndependentBodies` is the ONLY exact
contract that was RED in the parent ACT and is now GREEN under
this ACT's region-bounded chain construction. The remaining RED
contracts all share the same root cause: the chain construction
still emits multiple findings for what should be a single physical
clone relation.

## P0 corrections applied

### P0-1 — Content identity separated from region metadata

Introduced `v4RegionSeedMatch` to attach region identity to a
seed match without overwriting the original content-seed
fingerprint. The previous `encodeRegionChainSeed` /
`parseRegionChainSeed` colon-delimited encoding was deleted; no
delimiter-encoded paths participate in chain keys. `Match.SeedFingerprint`
remains the canonical content-identity carrier.

### P0-2 — All fingerprint buckets contribute globally

`v4BuildRegionBoundedChainInputs` aggregates every fingerprint
bucket into one combined match slice before partitioning by
structured chain-pair key. Two adjacent windows with distinct
seed fingerprints but the same region pair can now extend into
one maximal chain inside the partition.

### P0-3 — Deterministic chain-key iteration

`sortChainPairKeys` orders keys by
`(LeftRegion.Path, LeftRegion.Ordinal, RightRegion.Path,
RightRegion.Ordinal, Offset)` and `sortRegionAnnotatedMatches`
orders matches inside a partition by
`(Left.Path, Left.StartPos, Left.EndPos, Right.Path,
Right.StartPos, Right.EndPos, SeedFingerprint)`. No comparator
leaves distinct projected values equal.

### P0-4 — AST region inventory and exclusive ends

`buildRegions` now inventories:

* top-level function declarations;
* methods (still represented as `*ast.FuncDecl` with a receiver);
* nested function literals;
* function literals in package-level `var`/`const`/`type` declarations.

`inclusiveRegionEnd` converts the AST-exclusive `End()` to the
inclusive final-token boundary and advances past the
auto-inserted SEMICOLON that the Go scanner emits after the
closing `}` of a function body. The conversion does not include
the first token of the next declaration, behaves correctly at
EOF, and never includes package-declaration tokens.

### P0-5 — Non-overlapping token ownership

`v4FileAnalysis.TokenOwner` is a per-token innermost-owner array
where the innermost executable region wins. `windowFitsRegion`
verifies that every token in `[start, end]`, inclusive, belongs
to the same owning region; windows crossing a region boundary OR
between owned and unowned token ranges are rejected. Package
tokens carry the zero value (`Path == ""`) and windows anchored
on package tokens are rejected.

### P0-6 — Same-region partial clones preserved

The unconditional `if leftRegion == rightRegion { continue }`
rejection was removed. Same-region pairs now require disjoint
token ranges (`tokenRangesOverlap == false`); overlapping or
identical physical ranges are still rejected via the symmetric
predicate. The within-file disjoint clone case (RepeatedMultiplicity
B1 vs B2) is preserved.

### P0-7 — Shadow suppression correctness

* The single-chain short-circuit (`if len(chains) < 2 { return }`)
  was removed; structural self-match validation runs for one or
  more chains.
* The deterministic equal-range tie-break (`deterministicEqualTieBreak`)
  compares `(offset, TokenSpan, LineSpan, content hash, original
  input ordinal, canonical chain string)` and is total.
* `chainContainsRange` and `chainRangeRelationBetween` agree on
  the strict-containment rule: outer must contain inner on both
  sides AND at least one side must be strictly larger.
* All overlap checks route through the symmetric
  `tokenRangesOverlap` predicate.

### P0-8 — Structured occurrence identity + invariants

`maximalOccurrenceKey{Path, StartPos, EndPos}` replaced the
colon-delimited string encoding. `assertOccurrenceIdentityInvariants`
runs in `v4OccurrenceFromChain` BEFORE dedup and in
`v4MergeToNWayClone` before merging any group's occurrences; the
legacy `lineOccurrenceKey` helper was deleted. Two occurrences
sharing the token-position key but disagreeing on line geometry
fail closed; two disjoint occurrences in one file are retained.

### P0-9 — Non-mutating region filter

`filterWindowsToRegions` allocates a fresh `make([]rawWindow, 0,
len(wins))` slice for the kept windows; the input `windowMap`
and its slice values are NOT modified.

### P0-10 — One shared production pipeline

`v4BuildInternalFindings(windowMap, analyses)` is the one
production-owned internal seam. Both `CheckRepo` and
`v4PipelineInternal` route through it; the legacy
`v4BuildChainsWithPartitioning` path is preserved only as a
helper for legacy narrow tests via `legacyV4CoalesceFindings`.

### P0-11 — Pair-independent maximal content identity

`v4ContentIdentityFromChain` derives the stable fingerprint
exclusively from the ordered content-seed sequence and
relative-advancement tuples of the chain's matches. File path,
region ordinal, line number, pair orientation, offset, and map
iteration order are excluded.

## Files changed in the parent ACT checkpoint

The parent manifest classification is reconciled with the staged tree:

| Path | Parent checkpoint status | Change |
|---|---|---|
| `internal/factory/dupcode/v4_regions.go` | NEW | Region inventory, owner array, non-mutating filter, semicolon boundary. |
| `internal/factory/dupcode/v4_chain_key.go` | NEW | Region seed matches, structured chain partitions, deterministic ordering. |
| `internal/factory/dupcode/v4_shadow_suppression.go` | NEW | Region-aware structural shadow suppression. |
| `internal/factory/dupcode/v4_occurrences.go` | MODIFIED | Structured occurrence identity and invariant checks. |
| `internal/factory/dupcode/v4_internal_pipeline.go` | NEW | Shared production orchestration seam. |
| `internal/factory/dupcode/check.go` | MODIFIED | Public path routed through the shared seam. |
| `internal/factory/dupcode/coalesce.go` | MODIFIED | Legacy compatibility path retained. |
| `internal/factory/dupcode/v4_coalesce.go` | MODIFIED | Production merge projection changes. |
| `internal/factory/dupcode/v4_exact_geometry_internal_helpers_test.go` | MODIFIED | Internal fixture path and seam wiring. |
| `internal/factory/dupcode/v4_occurrence_identity_test.go` | NEW | Occurrence identity contracts. |
| `internal/factory/dupcode/v4_shadow_suppression_test.go` | NEW | Shadow and overlap contracts. |
| `docs/close-reports/ACT-LEAMAS-FACTORY-DUPCODE-V4-REGION-BOUNDED-CHAIN-CONSTRUCTION01.md` | NEW | Parent continuation report. |
| `docs/close-reports/ACT-LEAMAS-FACTORY-DUPCODE-V4-REGION-BOUNDED-CHAIN-CONSTRUCTION02.md` | NEW | This checkpoint report. |

## Verification commands run at the parent checkpoint

```text
gofmt -w <changed files>                                            PASS
gofmt -l .                                                          empty
go vet ./internal/factory/dupcode                                   PASS
go build ./...                                                       PASS
go test ./internal/factory/dupcode -run '^TestV4SuppressShadowChains_|^TestV4ChainPairKeyForChain_|^TestV4ChainRangeRelationBetween_|^TestV4TokenRangesOverlap_' -count=1 -v   PASS (12/12)
go test ./internal/factory/dupcode -run '^TestV4Occurrence' -count=1 -v   PASS (9/9)
go test ./internal/factory/dupcode -run '^TestV4ExactSemantics_' -count=1 -v   PARTIAL (2 PASS / 5 RED)
go test ./internal/factory/dupcode -run '^TestV4ExactGeometry_' -count=1 -v   PARTIAL (1 PASS / 7 RED)
go test ./internal/factory/dupcode -run '^TestV4ExactGeometryInternal_' -count=1 -v   PARTIAL (1 PASS / 5 RED)
```

A baseline regeneration was NOT performed. The behaviour change
that this ACT introduces is partial; the corrected repository
findings must be reviewed before any baseline update. The same
prerequisite guards the duplicate-code baseline regeneration
that the parent ACT invoked.

## Why this ACT cannot close with 21 PASS

The five RED exact-semantic contracts all share the same root
cause: the chain construction still emits multiple findings for
what should be a single physical clone relation. Specifically:

* `OneMaximalClone`: the chain construction emits two findings
  when the fixture contains one clone body.
* `RepeatedMultiplicity`: the within-file (region 0 of `repeat_a.go`,
  region 1 of `repeat_a.go`) chain and the cross-file
  (region 0 of `repeat_a.go`, region 0 of `repeat_b.go`) chain
  have different `TokenSpan` values and therefore do not merge
  through `v4MergeFindings`.
* `NWayClone`: the three pairwise chain keys produce three
  findings whose `TokenSpan` differs by one or two tokens because
  the intersection of each file pair's window range is shorter
  than the union of all three files.
* `NoShadowSubFindings`: same defect as `OneMaximalClone`,
  compounded by a residual internal-geometry shadow emission.
* `CanonicalOrdering`: depends on the multiplicity-merge defect.

To resolve these defects, the follow-up must materialize each finalized
pair as exact evidence: both file-local token slices must have equal
counts and equal normalized-content digests. The merge identity is the
structured `(Digest, TokenCount)` key; `StableFingerprint` alone is not a
merge key. Validated pair edges must then form deterministic N-way
connected components, followed by structural shadow proof. This is a
canonical-content follow-up to the region-bounded construction, not a
fingerprint-only change.

## Follow-up

The next ACT must:

1. Rebase every embedded region and token-owner path identity, and use one
authoritative per-file token inventory for window geometry and hashing.
2. Materialize exact pair evidence with the structured `(Digest, TokenCount)`
identity. A same-fingerprint/different-count case must fail closed as an
unresolved canonical-geometry conflict.
3. Build deterministic N-way connected components from validated pair edges,
preserve same-file multiplicity and legitimate partial clones, and apply
structural post-component shadow suppression only where containment,
relative offset, content, and multiplicity are all proven.
4. Re-run all 21 exact contracts and the legacy V4 contracts before any
baseline regeneration.

`ACT-LEAMAS-FACTORY-DUPCODE-V4-ALL-PAIRS-MATERIALIZATION-PERFORMANCE01`
remains blocked behind the parent ACT
`ACT-LEAMAS-FACTORY-DUPCODE-V4-EXACT-SEMANTICS-PRODUCTION01`.

## Checkpointed at

2026-07-16T16:30:00+03:00

## Reconciliation note (added by CANONICAL-MAXIMAL-COMPONENT-MERGE01-CORRECTION01)

The historical PARTIAL state captured above remains the truthful
record of this ACT's checkpoint. The blocking correctness work
that was unresolved at this checkpoint has since been completed by
the follow-up ACT
`ACT-LEAMAS-FACTORY-DUPCODE-V4-CANONICAL-MAXIMAL-COMPONENT-MERGE01`.
The 21-test exact contract suite is now 21 PASS / 0 FAIL on the
final tree; the previously-blocked
`ACT-LEAMAS-FACTORY-DUPCODE-V4-ALL-PAIRS-MATERIALIZATION-PERFORMANCE01`
is unblocked.

Historical command results above remain valid as checkpoint
evidence. The follow-up state is determined by
`docs/close-reports/ACT-LEAMAS-FACTORY-DUPCODE-V4-EXACT-SEMANTICS-PRODUCTION01.md`
and the closing report for the correction ACT.
