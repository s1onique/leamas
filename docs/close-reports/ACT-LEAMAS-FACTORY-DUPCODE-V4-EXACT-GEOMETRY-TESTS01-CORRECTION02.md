# ACT-LEAMAS-FACTORY-DUPCODE-V4-EXACT-GEOMETRY-TESTS01-CORRECTION02

## Status: COMPLETE

`ACT-LEAMAS-FACTORY-DUPCODE-V4-EXACT-GEOMETRY-TESTS01-CORRECTION02` is
COMPLETE. Canonical finding order is now frozen against the real
fingerprint-first ordering key, and internal geometry assertions observe
the same production-owned merge seam used by public findings. The test
suite no longer reconstructs N-way merge behavior independently.

## Parent and block

- Parent: `ACT-LEAMAS-FACTORY-DUPCODE-V4-EXACT-GEOMETRY-TESTS01-CORRECTION01`
- Blocks: `ACT-LEAMAS-FACTORY-DUPCODE-V4-EXACT-SEMANTICS-PRODUCTION01`
- Next executable ACT: `ACT-LEAMAS-FACTORY-DUPCODE-V4-EXACT-SEMANTICS-PRODUCTION01`

## Defects corrected

CORRECTION01 claimed that its canonical-order test contained frozen
64-character stable-fingerprint literals. It did not: its expected
values contained only TokenCount and Occurrences and assumed the first
source body preceded the second.

CORRECTION01 also made `v4PipelineInternal` group chains by fingerprint,
deduplicate by token position, perform its own N-way merge, sort the
result, and reconstruct internal findings. That second implementation
could agree with the specification while production merge behavior was
wrong.

The remaining comparator and projector defects were also real:
internal occurrence canonicalization omitted StartLine and EndLine,
accepted path cases used substring checks, and the previous gate claim
had no complete canonical summary or raw output proving exclusivity.

## Production-owned shared merge seam

Production now owns this retained-position representation and seam:

```go
type v4InternalFinding struct {
    StableFingerprint string
    TokenCount        int
    LineCount         int
    Occurrences       []maximalOccurrence
}

func v4InternalFindingsFromChains(chains []cloneChain) []v4InternalFinding
```

The production flow is now:

```text
chains
  -> v4InternalFindingsFromChains
  -> coalescedFinding projection
  -> public Finding
```

`v4PipelineInternal` invokes the same
`v4InternalFindingsFromChains` function and only projects retained
positions into `exactInternalFindingGeometry`. No `_test.go` file now
groups chains into findings, selects merge keys, deduplicates production
occurrences, or implements N-way merge.

The extraction changed no merge key, deduplication key, maximalization,
occurrence ordering, fingerprint, public projection, or baseline. The
existing merge remains keyed by StableFingerprint and TokenCount;
deduplication remains Path, StartPos, EndPos; LineCount remains the
maximum in an N-way group.

## Behavior-preserving extraction proof

`TestV4InternalFindingExtraction_CharacterizedPublicOutput` freezes the
pre-extraction public bytes for a declarative chain fixture exercising:

- two fingerprint-ordered findings;
- N-way merge;
- duplicate token-position elimination;
- occurrence sorting;
- maximum LineCount selection.

The test passed before extraction and passed unchanged after extraction.
`TestV4InternalFindingExtraction_CharacterizedEdgeOutputs` freezes nil,
empty, and all-filtered results. The shared-seam test compares the public
path element-by-element with a projection of the production-owned seam.
All three tests pass.

No semantic production repair was made. Maximality, multiplicity,
N-way semantics, shadow elimination, and public ordering remain owned
by the blocked production ACT.

## Frozen canonical finding-order keys

The ordering projection is:

```go
type exactFindingOrderKey struct {
    StableFingerprint string
    TokenCount        int
    LineCount         int
    Occurrences       []exactOccurrenceGeometry
}
```

Production precedence is reproduced as StableFingerprint, TokenCount,
LineCount, then canonical occurrence sequence. The expected slice is a
literal and is never sorted from actual output.

The independently audited expected keys are:

| Body and geometry | StableFingerprint | Tokens | Lines |
|---|---|---:|---:|
| addition, `ind_{a,b}.go:3-85` | `78b75750feff94c4f09d1b48e00fb737cb72e81d417b8fac6f3f1cd4ecabab43` | 491 | 83 |
| subtraction, `ind_{a,b}.go:87-169` | `9c779aa5a1dff976e5c91dfcfd38c9e3b6aab17961d5de0f5dd7d9e61673098e` | 491 | 83 |

### Literal derivation

A standalone Go scanner audit, not `CheckRepo` output, normalized each
491-token body and enumerated its 92 contiguous 400-token seeds. It
hashed the ordered seed stream with domain `leamas-dupcode-v4` and the
adjacent-window tuple `:0:0:399:399`.

The independently calculated content hashes were:

- addition: `1ec5b6bf6c957ee225f3b576c836b627861b3e90e4215f85496cf23c3f0e4773`;
- subtraction: `bad6b66e77cf9288d534970eb0c2a891ee6452ea43398ef62c746f2d77492306`.

SHA-256 over `leamas-dupcode-v4:<content-hash>` produced the frozen
stable fingerprints. The addition key precedes subtraction because
`0x78 < 0x9c`; source-line order is not the oracle.

The fixture generator was corrected from addition to subtraction for
the second body because identifier-only differences disappear during
normalization. Token and line geometry are unchanged. Test validation
requires two nonempty, distinct, canonical lowercase 64-character hex
fingerprints, verifies strict expected-key order, and compares raw
published keys exactly. A fingerprint attached to the wrong geometry
therefore fails.

Fingerprint is used only by this ordering oracle. Normative public
geometry equality remains TokenCount plus occurrence Path, StartLine,
and EndLine.

## Total canonicalization and exact paths

`canonicalizeInternalOccurrences` now compares every projected field in
this order: Path, StartPos, EndPos, StartLine, EndLine.
`TestCanonicalizeInternalOccurrences_TotalComparator` proves StartLine
and EndLine ties are deterministic when path and token positions match.

`TestNormalizeFixturePath_Contract` now uses exact slash-normalized
expected values for all accepted cases. Escaping, empty-root, and
empty-occurrence cases remain explicit rejections.

## Final file inventory against HEAD

| Path | State | Final purpose |
|---|---|---|
| `docs/close-reports/ACT-LEAMAS-FACTORY-DUPCODE-V4-EXACT-GEOMETRY-TESTS01.md` | M | Points to CORRECTION02 |
| `docs/close-reports/ACT-LEAMAS-FACTORY-DUPCODE-V4-EXACT-GEOMETRY-TESTS01-CORRECTION01.md` | A | Marked superseded |
| `docs/close-reports/ACT-LEAMAS-FACTORY-DUPCODE-V4-EXACT-GEOMETRY-TESTS01-CORRECTION02.md` | A | This final record |
| `internal/factory/dupcode/baseline_verify_test.go` | M | Final newline required by gofmt gate |
| `internal/factory/dupcode/v4_coalesce.go` | M | Shared production internal seam and public projection |
| `internal/factory/dupcode/v4_occurrences.go` | M | Production internal representation and merge signatures |
| `internal/factory/dupcode/v4_semantics_test.go` | M | Truly distinct subtraction fixture body |
| `internal/factory/dupcode/v4_internal_finding_extraction_test.go` | A | Pre/post byte characterization and shared-path proof |
| `internal/factory/dupcode/v4_exact_geometry_bodies_test.go` | A | Exact public body geometry contracts |
| `internal/factory/dupcode/v4_exact_geometry_comparator_test.go` | A | Total comparator contract |
| `internal/factory/dupcode/v4_exact_geometry_determinism_test.go` | A | Public/internal determinism contracts |
| `internal/factory/dupcode/v4_exact_geometry_diagnostics_test.go` | A | Exact multiplicity diagnostics |
| `internal/factory/dupcode/v4_exact_geometry_internal_helpers_test.go` | A | Production-seam invocation and projection only |
| `internal/factory/dupcode/v4_exact_geometry_internal_test.go` | A | Exact retained-position contracts |
| `internal/factory/dupcode/v4_exact_geometry_order_key_test.go` | A | Literal fingerprint-first key oracle |
| `internal/factory/dupcode/v4_exact_geometry_ordering_test.go` | A | Raw finding and occurrence order tests |
| `internal/factory/dupcode/v4_exact_geometry_path_test.go` | A | Exact path equality and rejection tests |
| `internal/factory/dupcode/v4_exact_geometry_support_test.go` | A | Projection and total canonicalization helpers |
| `internal/factory/dupcode/v4_exact_semantics_bodies_test.go` | M | Final newline required by gofmt gate |
| `internal/factory/dupcode/v4_exact_semantics_determinism_test.go` | M | Final newline required by gofmt gate |
| `internal/factory/dupcode/v4_exact_semantics_ordering_test.go` | M | Final newline required by gofmt gate |
| `internal/factory/dupcode/v4_exact_semantics_test.go` | M | Final newline required by gofmt gate |

The CORRECTION01 geometry files and the original report amendment were
already staged when this correction began. They remain intended parts
of the complete red-spec patch. No duplicate-code baseline was changed
or regenerated.

## Verification evidence

### Focused green contracts

The following passed:

```text
go test ./internal/factory/dupcode -run '^TestV4InternalFindingExtraction_' -count=1 -v
go test ./internal/factory/dupcode -run '^(TestCanonicalizeInternalOccurrences_TotalComparator|TestNormalizeFixturePath_Contract)$' -count=1 -v
go test ./internal/factory/dupcode -run '^TestV4_' -count=1 -v
```

The extraction command passes all three extraction tests and all three
edge-output subtests. Comparator and all eight path subtests pass. All
17 legacy `TestV4_` tests pass.

### Focused ordering RED

```text
go test ./internal/factory/dupcode -run '^TestV4ExactGeometry_Canonical(Finding|Occurrence)Ordering$' -count=1 -v
```

Exit 1 for the two intended red specifications. The finding-order test
reached exact key comparison: actual key 0 fingerprint
`00c46b415316cd20c2bdd603c3a1a8dc3080012f97246209aaea2539147e3b24`
was compared with frozen expected fingerprint
`78b75750feff94c4f09d1b48e00fb737cb72e81d417b8fac6f3f1cd4ecabab43`.

### Internal geometry

```text
go test ./internal/factory/dupcode -run '^TestV4ExactGeometryInternal_' -count=1 -v
```

PASS: `TestV4ExactGeometryInternal_Determinism`.
RED for documented production defects: OneMaximalClone,
RepeatedMultiplicity, NWayClone, TwoIndependentBodies, and
NoShadowSubFindings. Every value came from the shared production seam.

### Complete exact specifications

```text
go test ./internal/factory/dupcode -run '^TestV4Exact(Semantics|Geometry)' -count=1 -v
```

PASS:

- `TestV4ExactGeometry_Determinism`;
- `TestV4ExactGeometryInternal_Determinism`;
- `TestV4ExactSemantics_Determinism`.

RED: the 5 public geometry bodies, 5 internal geometry bodies, 2
geometry ordering tests, 5 exact semantics bodies/ordering tests, and
`TestV4ExactSemantics_NoShadowSubFindings` — 18 intended red tests in
total. The complete raw specification log is captured under `.factory`
and the package had no other failing test.

`go test ./internal/factory/dupcode -count=1` also failed only those same
18 exact tests. `go vet ./...` passed with no diagnostics, and
`CGO_ENABLED=0 go build -trimpath -o bin/leamas ./cmd/leamas` passed.

### Factorize and final gate

`make factorize` passes all 15 Factory verifiers. An initial factorize
run found one overlong characterization literal; it was split without
changing bytes. A preliminary gate exposed five missing final newlines,
including one baseline test; gofmt repaired only those newline defects.
Final `gofmt -l .` is empty.

Final command:

```text
make gate
```

Result: exit 2 from Make because the gate returned failure. All 15
Factory verifiers pass. Toolchain results are: go mod tidy PASS, gofmt
PASS, go vet PASS, static build PASS, and go test FAIL only for the 18
installed exact semantic/geometry red specifications listed above.
There is no compile, vet, build, format, baseline, or unrelated failure.

`RunGate` prints results and returns an exit code; it does not call
`WriteGateSummary`. Therefore a failing gate intentionally creates no
canonical `.factory/gate-summary.json`. Complete final raw output is
captured instead at:

```text
.factory/ACT-LEAMAS-FACTORY-DUPCODE-V4-EXACT-GEOMETRY-TESTS01-CORRECTION02-final-make-gate.log
```

That ignored evidence artifact contains every verifier result, full
`go test ./...` failure output, every failing test name, toolchain
result, and final gate status. No targeted digest with unavailable gate
status is used as exclusivity evidence.

### Repository hygiene

Final checks run:

```text
git diff --check
git status --short
git diff
git diff --cached
```

All 22 intended paths are staged. There are no unstaged tracked changes
and no non-ignored untracked files. The `.factory` raw logs and
`bin/leamas` are ignored evidence/build artifacts. No unrelated
behavioral source change, baseline regeneration, or semantic production
repair occurred.

## Skipped and deferred

No required check was skipped. Repairing the 18 red contracts is
deferred to `ACT-LEAMAS-FACTORY-DUPCODE-V4-EXACT-SEMANTICS-PRODUCTION01`.
A canonical gate summary is unavailable by documented gate behavior on
failure; complete raw final output is provided instead.

## Closed at

2026-07-16T11:20:00+03:00
