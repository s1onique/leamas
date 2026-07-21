# ACT-LEAMAS-GATE-SUMMARY-V2-NORMALIZATION01-CORRECTION02

## Status

IN PROGRESS

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

The `expectedRanks` map (lines 123–151) was removed. The test now
performs only structural assertions:

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

**Deferred to next ACT**:
- P2 identity chain reconciliation
- P3 verification evidence recording (forward commits)
- P4 source/result isolation gap analysis
- DIGEST01 mark READY

## Next ACT

`ACT-LEAMAS-GATE-SUMMARY-V2-DIGEST01` (READY after corrections pass)
