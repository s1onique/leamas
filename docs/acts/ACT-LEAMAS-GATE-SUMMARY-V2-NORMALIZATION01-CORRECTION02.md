# ACT-LEAMAS-GATE-SUMMARY-V2-NORMALIZATION01-CORRECTION02

## Status

PARTIAL — P0 resolved; P2/P3/P4 remaining

## Motivation

Review of CORRECTION01 closure rejected the ACT as unconditional due to
four blocking findings:

1. **P0 — Precedence authority duplicated**: The test
   `TestNormalizationDiagnosticOrderingPreservesPrecedenceAuthority`
   contained a literal 27-entry `expectedRanks` map that duplicated the
   production `codePrecedence` table, directly contradicting the
   requirement that production remains the sole ordering authority.

2. **P2 — Identity record internally contradictory**: The reported
   identities (close commit, evidence binary, implementation commit,
   trees) do not form a coherent chain. Cannot be dismissed as
   alternative labels.

3. **P3 — Required verification evidence missing**: The close report
   claimed all commands succeeded but did not record results for
   `make factorize`, `make gate`, `go test -short`, agent-context
   verification, forbidden-patterns verification, and worktree
   cleanliness.

4. **P4 — Source-isolation coverage overclaimed**: The tests establish
   useful result-to-result independence but source/result isolation is
   narrower than claimed for all mutable or pointer-backed fields.

## Scope

This ACT is narrowly bounded to:

1. Remove the copied precedence table and replace the vacuous
   authority test with a structural meta-test that checks rank
   uniqueness without duplicating `codePrecedence`.
2. Analyze and address P4 source/result isolation coverage gaps.
3. Reconcile identity chain for baseline, implementation, tested,
   evidence, close, tree, tag, and binary commits.
4. Run and record every required verification command.
5. Correct the close report using forward commits.
6. Mark `DIGEST01` READY only after those corrections pass.

The 41-case corpus and semantic matrices remain frozen unless a
correction test proves an actual defect.

## Changes

### P0 Fix: Remove duplicated precedence authority

**File**: `internal/gatesummary/normalization_diagnostic_ordering_combo_test.go`

Two changes:

1. Removed the literal 27-entry `expectedRanks` map (lines 123–151) that
   duplicated the production `codePrecedence` table.

2. Deleted `TestNormalizationDiagnosticOrderingUsesProductionAuthority` entirely.
   The surrounding black-box ordering tests already provide the meaningful proof
   of ordering authority. The pointer check proved only that the map exists,
   not that diagnostic sorting uses it.

The remaining structural test `TestNormalizationDiagnosticOrderingPreservesPrecedenceAuthority`
now performs only structural assertions without duplicating `codePrecedence`:

- No empty code strings in `codePrecedence`.
- All ranks are positive integers.
- All ranks are unique (no duplicate rank assignments).
- Sanity check that at least 27 codes are present.

The production `codePrecedence` map remains the single source of truth.
Tests must never reproduce the code→rank mapping.

## Verification

```text
go test -count=1 ./internal/gatesummary/...          PASS (0.442s)
go vet ./internal/gatesummary/ ./cmd/leamas/         PASS
CGO_ENABLED=0 go build -buildvcs=true -trimpath      PASS
make factorize                                       PASS (589.04s)
make gate-fast                                       PASS (fast lane)
```

**Gate results**: All verifiers passed including agent-context,
docs, doctrine, domain-boundaries, exec-gate, executable-contract-first,
forbidden-patterns, git-hooks, language, llm-friendly, long-test-policy,
static-binary, tooling-boundaries, go mod tidy, gofmt, go vet, go test
-short, static build.

### Evidence Identity Chain

```
CORRECTION02 commit:       d994fd1a4d2c6b7203aebc4001e6874bb49e0cb2
CORRECTION02 tree:         d10cb0857f239726166a0a524240a636ec7233b4

Proof binary vcs.revision: e5b1cde37d6756d35689b80e338755e2d7a4aa09
Proof binary tree:          5072d5c022c9083ed83318a2fcde711e829b61a2

Binary:                    /tmp/leamas-correction02
Binary SHA256:             031bd74c1f198e3a791678aec66690f4a52a9ebd9458b502fddfdd494d1d9da4
vcs.modified:              false
vcs.time:                  2026-07-21T14:43:29Z
```

### Board State

```
ACT-LEAMAS-GATE-SUMMARY-V2-NORMALIZATION01
  PARTIAL — CORRECTION02 active

ACT-LEAMAS-GATE-SUMMARY-V2-NORMALIZATION01-CORRECTION01
  PARTIAL — implementation accepted; closure superseded

ACT-LEAMAS-GATE-SUMMARY-V2-NORMALIZATION01-CORRECTION02
  PARTIAL
  P0a copied precedence table: RESOLVED
  P0b vacuous pointer test: RESOLVED
  P2 identity reconciliation: OPEN
  P3 evidence recording: OPEN
  P4 isolation gap analysis: OPEN

ACT-LEAMAS-GATE-SUMMARY-V2-DIGEST01
  BLOCKED
```

### Remaining in this ACT

- Reconcile identity chain for all relevant revisions
- Bind and record verification evidence
- Complete source/result isolation gap analysis
- Close CORRECTION02

### Next ACT after closure

`ACT-LEAMAS-GATE-SUMMARY-V2-DIGEST01`
