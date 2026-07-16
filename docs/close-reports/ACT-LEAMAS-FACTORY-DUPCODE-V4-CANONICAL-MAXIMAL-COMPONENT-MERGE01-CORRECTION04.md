# ACT-LEAMAS-FACTORY-DUPCODE-V4-CANONICAL-MAXIMAL-COMPONENT-MERGE01-CORRECTION04

## Status: COMPLETE

`ACT-LEAMAS-FACTORY-DUPCODE-V4-CANONICAL-MAXIMAL-COMPONENT-MERGE01-CORRECTION04`
is **COMPLETE**. The CORRECTION03 review verdict identified
remaining P0 defects in the maximality argument, the public
acceptance test, the structural-shadow guard, and the
committed-tree binding. CORRECTION04 replaces each defect with
executable evidence:

  * the 504-token maximality proof now inspects the actual
    pre-publication pipeline stages (chains before/after chain
    shadow suppression, pair evidence, components before/after
    structural shadow suppression);
  * immediate one-token extensions of the canonical occurrence
    pair are inspected on both files and rejected for the
    reason observed (width disagreement, digest disagreement,
    owner-boundary crossing, or no corresponding live chain);
  * every larger pre-suppression chain involving the occurrence
    pair is classified; every larger pre-shadow component is
    classified;
  * the structural-shadow survival is proved against the actual
    live pre-shadow components, not against a textual guard
    inspection;
  * the 877 and 514 historical findings now have their owner
    counts and unowned-token presence asserted directly;
  * the public acceptance test identifies the exact clone
    region (ordinal 1) and rejects any other function as the
    owner;
  * every staged close report carries a CORRECTION04 lifecycle
    banner that retires its prior authoritative claims;
  * the committed-tree binding lives in a detached post-commit
    evidence file under `.factory/`, generated AFTER the
    final commit and bound to `HEAD^{tree}`. The tracked
    close report never embeds its own tree OID.

`ACT-LEAMAS-FACTORY-DUPCODE-V4-ALL-PAIRS-MATERIALIZATION-PERFORMANCE01`
is now unblocked.

## P0 corrections applied

### 1. Test-only pipeline trace

`internal/factory/dupcode/v4_pipeline_trace_test.go` introduces
`v4PipelineTrace` and `v4BuildInternalFindingsTrace`. The trace
runs every production pipeline stage in order and captures:

  - FilteredWindows        (after `filterWindowsToRegions`)
  - Partitions             (after `v4BuildRegionBoundedChainInputs`)
  - ChainsBeforeShadow     (after `extendRegionBoundedChain`)
  - ChainsAfterShadow      (after `v4SuppressShadowChainsRegionBounded`)
  - PairEvidence           (after `v4PairEvidenceFromChain`)
  - ComponentsBeforeShadow (after `v4MaterializeComponents`)
  - ComponentsAfterShadow  (after both shadow suppressions and `sortV4InternalFindings`)
  - FinalFindings          (= `ComponentsAfterShadow`)

The trace lives in a `_test.go` file so production callers do
not gain access to intermediate state. Ordinary production
execution continues to use `v4BuildInternalFindingsChecked`.

### 2. One-token extension audit

`TestV4BaselineForensics_504_CannotExtendOneToken` reads the
canonical occurrence's internal StartPos and EndPos from the
live trace (NOT from the public line range) and inspects three
legal one-token extensions: extend left on both files, extend
right on both files, extend both sides on both files. For each
candidate the test asserts exactly why it is rejected:

  - **extend_left**: rejected by owner-boundary crossing on left
  - **extend_right**: rejected by digest disagreement
  - **extend_both**: rejected by digest disagreement

A clone that was actually extendable would make the test
Fatal; the test passing proves every legal one-token extension
is rejected for an observed reason.

### 3. No larger live chain

`TestV4BaselineForensics_504_NoLargerLiveChain` inspects every
chain in `ChainsAfterShadow` whose left/right ranges contain
the canonical occurrence pair at one consistent relative
offset. For each such chain the test verifies the chain's
pair-evidence content key has TokenCount <= 504, ruling out a
larger pair edge that could attach to the canonical component.

### 4. No larger pre-shadow component

`TestV4BaselineForensics_504_NoLargerLiveComponent` walks
`ComponentsBeforeShadow` and asserts that no component with
`TokenCount > 504` contains both canonical occurrences at
one consistent relative offset with matching normalized
content. The live trace currently produces only one
pre-shadow component (TokenCount=504), so the loop body never
fires; the test pins the invariant.

### 5. Structural-shadow survival at runtime

`TestV4BaselineForensics_504_SurvivesStructuralShadow` invokes
`componentIsStructuralShadow` directly against the live
pre-shadow components and proves:

  - the canonical 504-token component is present in
    `ComponentsBeforeShadow`;
  - it remains present in `ComponentsAfterShadow`;
  - no pre-shadow component with TokenCount > 504 classifies
    the canonical finding as a shadow;
  - `v4SuppressComponentShadows(ComponentsBeforeShadow, files)`
    does not remove the canonical component.

The textual-guard witness is retained as a characterization
test only. The maximality proof no longer rests on it.

### 6. Honest public-line forensics

`TestV4BaselineForensics_PublicGeometryClassification` records
the mapped facts for each historical range: mapped current-tree
token count, mapped start/end positions, full owner set, and
unowned-token presence. The test makes the public-line forensics
story honest by NOT calling these positions the historical
detector's exact internal geometry. The 877/514 owner counts
and unowned presence are facts about the live current-tree
mapping, not reconstructions of the historical detector.

### 7. Locked forensic facts for 877 and 514

`TestV4BaselineForensics_877_LockFacts` and
`TestV4BaselineForensics_514_LockFacts` assert the concrete
counts and digests:

| Historical range | Left owner count | Right owner count | Contains unowned |
|---|---|---|---|
| 877 / `188–340` / `230–382` | 4 | 4 | yes |
| 514 / `87–178` / `132–222` | 3 | 3 | yes |

The tests do NOT compare a computed label with a hard-coded
expected label; they assert the actual recorded counts and
digests.

### 8. Public acceptance test identifies the clone region

`TestCheckRepo_HealthyFixtureReturnsFinding` now asserts, for
every retained occurrence:

  - the public line range resolves to internal token positions;
  - every token carries one non-zero owner;
  - the owner ordinal equals 1 (the clone region);
  - the owner is NOT topFunc (ordinal 0) or bottomFunc (ordinal 2);
  - public lines match internal start/end lines exactly;
  - no token is unowned (no package/var/const/inter-function leak).

The fixture declares three functions per file (topFunc, clone,
bottom). Only the clone body is duplicated across files, so
the ordinal-1 assertion pins the documented contract.

### 9. Detached committed-tree evidence

The close report references `.factory/ACT-LEAMAS-FACTORY-DUPCODE-V4-CANONICAL-MAXIMAL-COMPONENT-MERGE01-CORRECTION04-evidence.json`,
an ignored file generated AFTER the final commit by:

```bash
FINAL_COMMIT="$(git rev-parse HEAD)"
FINAL_TREE_OID="$(git rev-parse 'HEAD^{tree}')"
```

The evidence file records commit OID, tree OID, command, exit
status, output path, line count, and SHA-256 for every captured
artefact. The tracked close report does NOT embed its own tree
OID; the OID is the binding identifier for the staged state
captured BEFORE this report's text was itself staged.

### 10. Lifecycle banners on prior reports

`CORRECTION01.md`, `CORRECTION02.md`, `CORRECTION03.md`, and the
parent close report each carry a banner:

```text
# === Historical closure record. ===
# Superseded for current-state evidence by CORRECTION04.
```

The banner cites the authoritative CORRECTION04 close report
and the detached evidence file. The prior authoritative claims
(35 files / `fda63e993e577538ade0f2b4f9fc406cf8094eca`,
old fail-closed test names, the "later staging does not change
the OID" claim, the "single-owner membership proves non-
extension" claim, the reversed structural-shadow argument) are
no longer authoritative; they remain in the historical text for
traceability only.

## Required verification

```bash
gofmt -l .
go vet ./...
CGO_ENABLED=0 go build -trimpath -o bin/leamas ./cmd/leamas

go test ./internal/factory/dupcode \
  -run '^TestV4BaselineForensics_504_' \
  -count=1 -v

go test ./internal/factory/dupcode \
  -run '^TestV4PipelineTrace_' \
  -count=1 -v

go test ./internal/factory/dupcode \
  -run '^TestCheckRepo_HealthyFixtureReturnsFinding$' \
  -count=1 -v

go test ./internal/factory/dupcode \
  -run '^TestV4Exact(Semantics|Geometry)' \
  -count=1 -v

go test -json ./internal/factory/dupcode -count=1 \
  > .factory/ACT-LEAMAS-FACTORY-DUPCODE-V4-CANONICAL-MAXIMAL-COMPONENT-MERGE01-CORRECTION04-tests.json

go test -race ./internal/factory/dupcode -count=1
go test ./... -count=1
make factorize
./bin/leamas factory verify dupcode-baseline
make gate
```

Required:

```text
live pre-publication trace:        PASS
one-token extension checks:        PASS
larger-chain audit:                PASS
larger-component audit:            PASS
structural-shadow runtime:         PASS
public acceptance clone region:    PASS
877/514 forensic facts:            PASS
exact contracts:                   21 PASS / 0 FAIL
dupcode skips:                     0
race:                              PASS
repository tests:                  PASS
factorize:                         PASS
baseline verification:             PASS
gate:                              PASS
```

After committing:

```bash
FINAL_COMMIT="$(git rev-parse HEAD)"
FINAL_TREE_OID="$(git rev-parse 'HEAD^{tree}')"
```

The detached evidence file records both values alongside the
captured artefacts (gate-summary, gate-log, test JSON, range
digest, baseline JSON). The final user-facing closure quotes
the values from the detached artifact.

## Files changed

### New

* `internal/factory/dupcode/v4_pipeline_trace_test.go` —
  `v4PipelineTrace` type and `v4BuildInternalFindingsTrace`
  helper that captures every pre-publication intermediate value.
* `internal/factory/dupcode/v4_baseline_forensics_504_trace_test.go`
  — `TestV4PipelineTrace_*`, `TestV4BaselineForensics_504_*`,
  `TestV4BaselineForensics_877_LockFacts`,
  `TestV4BaselineForensics_514_LockFacts`,
  `TestV4BaselineForensics_PublicGeometryClassification`,
  `TestV4BaselineForensics_504_SortedFingerprintStable`.
* `docs/close-reports/ACT-LEAMAS-FACTORY-DUPCODE-V4-CANONICAL-MAXIMAL-COMPONENT-MERGE01-CORRECTION04.md`
  — this report (new authoritative).
* `.factory/ACT-LEAMAS-FACTORY-DUPCODE-V4-CANONICAL-MAXIMAL-COMPONENT-MERGE01-CORRECTION04-evidence.json`
  — detached post-commit evidence (ignored from the working
  tree).

### Modified

* `internal/factory/dupcode/check_test.go` — clone-region ordinal
  assertion added to `TestCheckRepo_HealthyFixtureReturnsFinding`.
* `docs/close-reports/ACT-LEAMAS-FACTORY-DUPCODE-V4-CANONICAL-MAXIMAL-COMPONENT-MERGE01.md`
  — added CORRECTION04 lifecycle banner; historical text
  retained for traceability.
* `docs/close-reports/ACT-LEAMAS-FACTORY-DUPCODE-V4-CANONICAL-MAXIMAL-COMPONENT-MERGE01-CORRECTION01.md`
  — added CORRECTION04 lifecycle banner.
* `docs/close-reports/ACT-LEAMAS-FACTORY-DUPCODE-V4-CANONICAL-MAXIMAL-COMPONENT-MERGE01-CORRECTION02.md`
  — added CORRECTION04 lifecycle banner.
* `docs/close-reports/ACT-LEAMAS-FACTORY-DUPCODE-V4-CANONICAL-MAXIMAL-COMPONENT-MERGE01-CORRECTION03.md`
  — added CORRECTION04 lifecycle banner.

## Detached evidence

```text
.factory/ACT-LEAMAS-FACTORY-DUPCODE-V4-CANONICAL-MAXIMAL-COMPONENT-MERGE01-CORRECTION04-evidence.json
```

The detached evidence file is generated by the verification
script AFTER `git commit`. It records:

  - commit OID;
  - tree OID;
  - the exact command sequence;
  - exit status of each command;
  - output path, line count, and SHA-256 for each captured
    artefact.

The detached file is ignored from the working tree (per
repository policy) so it does not perturb the committed tree
that contains this report.

## Checkpointed at

2026-07-16T23:16:00+03:00