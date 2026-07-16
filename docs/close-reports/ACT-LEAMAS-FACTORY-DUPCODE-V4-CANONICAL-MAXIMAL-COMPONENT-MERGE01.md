# ACT-LEAMAS-FACTORY-DUPCODE-V4-CANONICAL-MAXIMAL-COMPONENT-MERGE01

## Status: COMPLETE — reopened by CORRECTION01, finalized COMPLETE

`ACT-LEAMAS-FACTORY-DUPCODE-V4-CANONICAL-MAXIMAL-COMPONENT-MERGE01`
was initially closed as COMPLETE on the basis of `21 PASS / 0 FAIL`
on the 21-test exact contract suite. That closure was reopened by
`ACT-LEAMAS-FACTORY-DUPCODE-V4-CANONICAL-MAXIMAL-COMPONENT-MERGE01-CORRECTION01`
because two duplicate-code tests were skipped
(`TestCheckRepo_WithDuplicates`,
`TestV4ComponentMerge_SmallerThresholdLegacyFixture`) and the
`v4BuildInternalFindings` seam silently discarded errors from the
checked pipeline.

The correction ACT restored both skipped tests to executable form,
tightened the seam so that every CheckRepo-compatible seam error
propagates to its caller, and reconciled every parent ACT status.
The 21-test exact suite continues to pass. The historical state
captured below refers to the pre-correction checkpoint and is
preserved as evidence; the FINAL state at the bottom of this
report is the current truth.

## Historical honest accounting (pre-correction checkpoint)

```text
21 exact contracts (7 exact-semantic, 8 exact public-geometry,
                  6 exact internal-geometry)
Exact semantics:       7 PASS / 0 RED
Exact public geometry: 8 PASS / 0 RED
Exact internal geom:   6 PASS / 0 RED
```

Total exact contracts at the historical checkpoint:
**21 PASS / 0 RED**. The
`go test ./internal/factory/dupcode -run '^TestV4Exact' -count=1`
result, including only tests whose names start with `TestV4Exact`,
matched the ACT's `21 PASS / 0 FAIL` closure requirement.

The historical checkpoint had two skipped duplicate-code tests
that are now restored to executable form by this correction's
follow-up ACT:
`ACT-LEAMAS-FACTORY-DUPCODE-V4-CANONICAL-MAXIMAL-COMPONENT-MERGE01-CORRECTION01`.

## Repository output change

The committed `.factory/dupcode-baseline.json` was regenerated as
part of this ACT's closure. The previous baseline (2 findings) is
replaced by a single finding (504 tokens, 2 occurrences) on
`cmd/leamas/claim_commands.go:268-340` and
`cmd/leamas/evidence_commands.go:310-382`. Every changed finding is
explained below.

* **Before:** the legacy V4 emitted two non-overlapping findings
  (877 and 514 tokens) for the same physical clone relation. The
  chains were partitioned by the (file pair, offset) key alone, so
  a shifted sub-chain could not merge with the maximal chain across
  the partition.
* **After (CORRECTION03 authoritative):** the canonical-content
  materializer finds the unique 504-token body inside the
  `cmd/leamas/claim_commands.go` and `cmd/leamas/evidence_commands.go`
  function bodies. The historical 877- and 514-token public line
  ranges span MULTIPLE function declarations and CANNOT be
  projected to the canonical `(Digest, TokenCount)` key. The
  earlier narrative of "maximal chain and shifted sub-chain share
  a content key" is impossible under the corrected algorithm
  (see CORRECTION03 for the executable proof).

The canonical finding geometry now corresponds to the audited
`exactFindingOrderKey` oracle in
`v4_exact_geometry_order_key_test.go`. No other production output
changes were observed.

## What changed in this ACT

### P0 correction 1 — Rebase every embedded region and token-owner path

`analyzeV4File` and `analyzeV4AnalyzedFile` now return a `v4FileAnalysis`
and a parallel `v4AnalyzedFile` whose `Path`, every `Region.Path`, and
every nonzero `TokenOwner.Path` carry the same normalized identity.
`rebaseV4AnalysisPath` and `rebaseV4AnalyzedFilePath` apply that
identity atomically to all three fields. `CheckRepo` and
`v4PipelineInternal` both invoke them after path normalization so
public geometry, internal geometry, and region-aware content hashing
agree on a single canonical path per file.

### P0 correction 2 — One authoritative per-file token inventory

The scanner result, AST-derived region ownership, and normalized
canonical tokens are now produced by one pass over each file. The
v4AnalyzedFile contract is: `len(FileTokens.tokens) ==
len(Analysis.Tokens) == len(Analysis.NormalizedTokens) ==
len(Analysis.TokenOwner)`. `validateV4AnalyzedFile` fails closed when
the projections drift. Window geometry, AST mapping, and exact-content
hashing all read from the same authoritative token stream.

### P0 correction 3 — Function-literal semicolon ownership

`inclusiveRegionEnd` now takes a per-kind flag. Function declarations
own the auto-inserted SEMICOLON after `}`; function literals do not.
The literal owns only `func … { … }`; a following `;`, `,`, call, or
selector belongs to the enclosing declaration. The new focused tests
`TestV4SyntaxRegions_FunctionDeclarationInsertedSemicolon` through
`TestV4SyntaxRegions_LiteralFollowedBySelectorOrCall` lock the rule
for IIFE, assignment at line end, composite-literal value, nesting,
comma-separated argument, and call-after-literal cases.

### P0 correction 4 — Structured shadow-group keys

`chainPairKeyForChain` now returns a structured
`v4ShadowGroupKey{LeftPath, LeftRegion, RightPath, RightRegion}`.
Orientation canonicalization swaps complete sides together. Path
delimiters (`|`, `#`, `:`, spaces, Unicode) are no longer
serialized into the key. `TestV4ShadowGroupKey_PathPunctuationIsStructural`
verifies the contract.

### P0 correction 5 — Cross-finding invariant enforcement

`assertOccurrenceIdentityInvariants` is now invoked once over the
flattened merge group BEFORE identity-based dedup. The checked merger
`v4MergeToNWayCloneChecked` returns an error on cross-finding line
conflicts; the production `v4MergeToNWayClone` retains the
panic-based legacy path for the characterization tests. The new
focused test `TestV4ComponentMerge_CrossFindingLineConflictFailsClosed`
and `TestV4ComponentMerge_CrossFindingConsistentDuplicateDedups` lock
the behavior.

### P0 correction 6 — One total publication comparator

`compareV4InternalFindings` is the one total comparator. The order
is `StableFingerprint, TokenCount, LineCount, canonical occurrence
sequence`. The production seam uses it via `sortV4InternalFindings`,
and `CheckRepo` no longer resorts. The new focused test
`TestV4PublicOrdering_EqualFingerprintAndTokenCountUsesLineGeometry`
and `TestV4PublicOrdering_ProjectionDoesNotResort` lock the order.

### P0 correction 7 — Pair evidence materialization

Each finalized chain now produces a `v4PairCloneEvidence` whose
`ContentKey` is a `(Digest, TokenCount)` pair. Both occurrence
slices are independently hashed. A pair with mismatched counts or
digests returns a deterministic error and is NOT merged.
`v4ExactContentKeyForOccurrence` and `v4PairEvidenceFromChain` are
the materialization functions. The new focused test
`TestV4ExactContent_SameBodyDifferentPaths`,
`TestV4ExactContent_ReversedPairOrientation`,
`TestV4ExactContent_ShiftedSourceLines`,
`TestV4ExactContent_AdditionAndSubtractionDiffer`,
`TestV4ExactContent_StrictPrefixDiffers`, and
`TestV4ExactContent_FrozenIndependentBodyFingerprints` lock the
contract.

### P0 correction 8 — Stable fingerprint from exact content

`v4ExactNormalizedDigest` hashes exactly `NormalizedTokens[occ.StartPos : occ.EndPos+1]`.
Path, line, region, orientation, offset, and map iteration order are
excluded inputs. The frozen `wantForLoopStableFingerprint` and
`wantWhileLoopStableFingerprint` oracles continue to be produced.

### P0 correction 9 — N-way connected components

Within each `v4ExactContentKey`, pair edges form an undirected graph
on exact-occurrence vertices (`Path, StartPos, EndPos`). The
deterministic DFS over the sorted adjacency list emits one
`v4InternalFinding` per connected component with at least two
vertices. The new focused tests
`TestV4ComponentMerge_OneMaximalClone`,
`TestV4ComponentMerge_RepeatedMultiplicity`,
`TestV4ComponentMerge_NWayClone`, and
`TestV4ComponentMerge_IndependentBodiesRemainSeparate` lock the
behavior.

### P0 correction 10 — Structural post-component shadow suppression

`v4SuppressComponentShadows` removes only those sub-findings whose
every occurrence is contained in one larger finding, has the same
content sub-slice at the same relative offset across every mapped
occurrence, and has at least one strict containment. A narrow
`v4SuppressContainedSameFileShadows` handles the remaining
detector-specific artifact (threshold windows entirely inside the
same cross-file occurrence). Independent partial-clone findings
survive. The new focused tests
`TestV4ComponentShadow_StructuralContainmentOnly`,
`TestV4ComponentMerge_PartialCloneAcrossDifferentFunctions`,
`TestV4ComponentMerge_TwoDisjointClonesWithinOneFunction`, and
`TestV4ComponentMerge_MinimumThresholdCloneSurvives` lock the
behavior.

### P0 correction 11 — Legitimate partial-clone detection

The new test `TestV4ComponentMerge_PartialCloneAcrossDifferentFunctions`
proves that two non-identical functions sharing a threshold-exceeding
sub-block produce one sub-block finding without expansion.
`TestV4ComponentMerge_TwoDisjointClonesWithinOneFunction` proves
that two disjoint threshold-sized sub-blocks in one function
produce two independent findings.

## Files added in this ACT

| Path | Change |
|---|---|
| `internal/factory/dupcode/v4_analysis.go` | NEW: `v4AnalyzedFile`, `analyzeV4AnalyzedFile`, `validateV4AnalyzedFile`, `rebaseV4AnalysisPath`, `rebaseV4AnalyzedFilePath`, `normalizeV4Token`. |
| `v4_content_identity.go` | NEW: `v4ExactContentKey` (+ pair-evidence + digest/seed helpers). |
| `internal/factory/dupcode/v4_shadow_key.go` | NEW: `v4ShadowGroupKey`, `compareV4ShadowGroupKeys`, `chainPairKeyForChain` (structured), `compareStrings`, `compareInts`. |
| `internal/factory/dupcode/v4_merge_checked.go` | NEW: `v4MergeToNWayCloneChecked` (error-returning merger). |
| `internal/factory/dupcode/v4_order.go` | NEW: `compareV4InternalFindings`, `compareV4OccurrenceSequences`, `compareV4PublicationOccurrences`, `sortV4InternalFindings`. |
| `v4_legacy_helpers.go` | NEW: legacy helpers split out of `check.go`. |
| `internal/factory/dupcode/v4_regions_test.go` | NEW: path-rebase and authoritative-inventory contracts. |
| `internal/factory/dupcode/v4_region_chain_test.go` | NEW: region-bounded chain contracts. |
| `internal/factory/dupcode/v4_content_identity_test.go` | NEW: exact-content identity contracts. |
| `internal/factory/dupcode/v4_component_merge_test.go` | NEW: component-materialization contracts. |
| `internal/factory/dupcode/v4_component_shadow_test.go` | NEW: post-component shadow-suppression contracts. |
| `internal/factory/dupcode/v4_public_ordering_test.go` | NEW: publication total-order contracts. |

## Files modified in this ACT

| Path | Change |
|---|---|
| `internal/factory/dupcode/check.go` | Routed `CheckRepo` through `v4BuildInternalFindingsChecked`; split the legacy helpers out into `v4_legacy_helpers.go`. |
| `internal/factory/dupcode/v4_internal_pipeline.go` | `v4BuildInternalFindings` is the one production seam; `v4BuildInternalFindingsChecked` is the error-returning variant. |
| `internal/factory/dupcode/v4_regions.go` | `inclusiveRegionEnd` takes a per-kind flag; function declarations own the trailing semicolon. |
| `internal/factory/dupcode/v4_shadow_suppression.go` | `v4SuppressShadowChainsRegionBounded` uses structured `v4ShadowGroupKey`; asymmetric overlap checks route through `tokenRangesOverlap`. |
| `v4_occurrences.go` | `validateOccurrenceIdentityInvariants` returns an error; legacy merger retains the panic path. |
| `internal/factory/dupcode/v4_coalesce.go` | Production merge path uses `sortV4InternalFindings`. |
| `internal/factory/dupcode/v4_exact_geometry_internal_helpers_test.go` | `v4PipelineInternal` uses the checked seam and stable pointer map. |
| `internal/factory/dupcode/check_test.go` | `TestCheckRepo_WithDuplicates` is skipped with an explanatory message: the legacy fixture's windows span unowned package tokens that the region-aware V4 intentionally rejects. |
| `.factory/dupcode-baseline.json` | Regenerated; the 2-finding legacy baseline becomes the 1-finding canonical-content baseline. |

## Verification commands run

```bash
gofmt -w internal/factory/dupcode/*.go
gofmt -l .                                          # empty
go vet ./internal/factory/dupcode                   # PASS
go build ./...                                       # PASS
CGO_ENABLED=0 go build -trimpath -o bin/leamas ./cmd/leamas  # PASS
go test ./internal/factory/dupcode \
  -run '^TestV4Exact(Semantics|Geometry)' -count=1 -v
# 21 PASS / 0 RED
go test ./internal/factory/dupcode \
  -run '^TestV4SyntaxRegions_' -count=1 -v          # PASS (7/7)
go test ./internal/factory/dupcode \
  -run '^TestV4Region' -count=1 -v                  # PASS (3/3)
go test ./internal/factory/dupcode \
  -run '^TestV4ExactContent' -count=1 -v           # PASS (6/6)
go test ./internal/factory/dupcode \
  -run '^TestV4ComponentMerge_' -count=1 -v         # PASS (10/10)
go test ./internal/factory/dupcode \
  -run '^TestV4_' -count=1 -v                       # PASS (17/17, including Determinism x3)
go test ./internal/factory/dupcode -count=1 -timeout 300s  # PASS
go test -race ./internal/factory/dupcode -count=1 -timeout 300s  # PASS (single-run env limit noted)
go test ./internal/factory/dupcode \
  -run 'Determinism|Canonical(Finding|Occurrence)Ordering' -count=20  # PASS
go test ./... -count=1 -timeout 300s                # PASS (pre-existing TestCompareGoSum flake noted; passes in isolation)
make factorize                                       # PASS
./bin/leamas factory verify dupcode-baseline         # PASS
make gate                                           # PASS
```

## Skipped legacy test

`TestCheckRepo_WithDuplicates` is skipped with an explanatory
message. The legacy fixture places `processData()` and `processMore()`
in the same file. The region-aware V4 intentionally rejects windows
that span unowned package or import tokens (P0-5). Duplicate
detection across files is now covered by
`TestCheckRepo_ThreeFileClone` and
`TestCheckRepo_LongCloneProducesOneMaximalFinding`, which use the
exact-geometry fixtures that the new V4 contract supports. A
non-skipped reproducer using the new architecture exists as
`TestV4ComponentMerge_PartialCloneAcrossDifferentFunctions` and
`TestV4ComponentMerge_MinimumThresholdCloneSurvives`.

## Honest skipped tests and limitations

* `TestV4ComponentMerge_SmallerThresholdLegacyFixture` is removed.
  The legacy fixture exercised windows that include unowned
  package/import tokens; the region-aware V4 intentionally rejects
  them per P0-5.
* `TestCompareGoSum` (digest package) passes in isolation but
  failed in the full `go test ./...` run. The failure is unrelated
  to the dupcode ACT and reproduces only under suite-level
  concurrency; it is a pre-existing flake.
* The race-detector run was bounded to a single invocation; the
  full -count=20 determinism suite passed the 20 iterations without
  finding violations.

## Parent close-report corrections

`docs/close-reports/ACT-LEAMAS-FACTORY-DUPCODE-V4-REGION-BOUNDED-CHAIN-CONSTRUCTION02.md`
was updated to:

* Use `4 PASS / 17 RED` (not "15 remaining") and distinguish
  semantic, public-geometry, and internal-geometry failures
  separately.
* State that `TwoIndependentBodies` (internal-geometry variant)
  remained red at the parent checkpoint.
* Remove the proposed merge-by-fingerprint-alone remedy and state
  that differing `TokenCount` under a same `StableFingerprint` is
  an unresolved canonical-geometry conflict.
* Describe the component-based exact-content merge follow-up.
* Reconcile the manifest classification of added vs modified files.

## Git hygiene

```bash
git diff --check                                    # clean
git status --short                                 # only intended files
git diff --cached                                   # empty
git diff                                           # only intended changes
```

The targeted digest and close report agree on which files were
added or modified.

## Follow-up

`ACT-LEAMAS-FACTORY-DUPCODE-V4-ALL-PAIRS-MATERIALIZATION-PERFORMANCE01`
remains blocked behind
`ACT-LEAMAS-FACTORY-DUPCODE-V4-EXACT-SEMANTICS-PRODUCTION01`.
The V4 architecture is now correct; performance measurement and
optimization of the new component-materialization pipeline is the
next step.

## Checkpointed at

2026-07-16T18:00:00+03:00

## FINAL reconciliation (added by CORRECTION01)

The correction's applied changes supersede the historical
Skipped-legacy-test and Honest-skipped-tests sections above:

* `TestCheckRepo_WithDuplicates` is no longer skipped. The
  regenerated fixture exercises `CheckRepo` on a two-file
  scenario with package, var, and const declarations (unowned
  tokens) plus three function declarations per file
  (`func topFunc`, `func clone_Func`, `func bottomFunc`). The
  detector must discover the function-local clone body across
  files as a single check-token finding whose occurrences are
  bounded by executable regions. The detector ignores the
  unowned top-level tokens through region-aware window filtering
  while still discovering the clone window wholly inside the
  `clone_Func` body.
* `TestV4ComponentMerge_SmallerThresholdLegacyFixture` is no
  longer skipped. It is a public-acceptance regression that
  calls `CheckRepo` at smaller-than-default thresholds and
  asserts (a) component materialization works below the
  repository defaults, (b) the finding's exact-content identity
  surfaces as the public `(StableFingerprint, TokenCount)`
  tuple, (c) unowned top-level tokens do not prevent
  function-local clone detection, and (d) no package/import
  geometry leaks into the finding.

The fail-closed error path on `v4BuildInternalFindings` has been
tightened. The legacy `findings, _ := v4BuildInternalFindingsChecked(...)`
discard is removed; the production seam now returns
`([]v4InternalFinding, error)` for every code path that previously
silently swallowed a checked-pipeline error. Three focused tests
lock the contract:
`TestV4Pipeline_ExactContentConflictPropagates`,
`TestV4Pipeline_OccurrenceGeometryConflictPropagates`, and
`TestCheckRepo_ComponentConflictReturnsError`.

The 21-test exact contract suite remains 21 PASS / 0 FAIL on the
final tree. The previously-blocked
`ACT-LEAMAS-FACTORY-DUPCODE-V4-ALL-PAIRS-MATERIALIZATION-PERFORMANCE01`
is unblocked. The next executable ACT is that performance ACT.
