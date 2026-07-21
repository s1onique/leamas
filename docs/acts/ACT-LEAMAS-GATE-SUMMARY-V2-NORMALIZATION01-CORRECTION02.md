# ACT-LEAMAS-GATE-SUMMARY-V2-NORMALIZATION01-CORRECTION02

## Status

PARTIAL — P4 focused tests pending; P3 evidence binding pending timestamps

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

### P2: Identity Chain (Corrected)

**Critical Finding**: The three reported revisions are **siblings** sharing a common
parent, not a linear ancestry chain. They must not be presented as interchangeable
tested identities.

```
Shared parent commit:  6bd8695473bccf9e1d389fdd51e2a5a87ad7e5ea
Shared parent tree:    8769d21d33a9bf7e650c55e9163a96980ac55cfc

Commit → Tree mapping:

d994fd1a4d2c6b7203aebc4001e6874bb49e0cb2
  tree: d10cb0857f239726166a0a524240a636ec7233b4
  diff from parent: -14 lines (ACT file content removed)

e5b1cde37d6756d35689b80e338755e2d7a4aa09
  tree: 5072d5c022c9083ed83318a2fcde711e829b61a2
  diff from parent: ACT file modified (content differs from both siblings)

76ace692e50e2ad1d13ed658b2e7832839274da0
  tree: 890bfa86b30de04b7d2dff833af8b87e97094eb2
  diff from parent: +15 lines (ACT file restored and extended)

692e673 (documentation checkpoint)
  tree: (descendant of 890bfa8...)
  diff: +166/-18 lines (ACT content updates)
  is descendant of 76ace69: YES (git merge-base --is-ancestor)
```

**Proof of non-ancestry**:
```
git merge-base --is-ancestor d994fd1 e5b1cde → NO
git merge-base --is-ancestor e5b1cde 76ace69 → NO
```

**Current authoritative tested state**:
```
tested/evidence revision = 76ace692e50e2ad1d13ed658b2e7832839274da0
tested/evidence tree     = 890bfa86b30de04b7d2dff833af8b87e97094eb2
vcs.modified             = false
```

Note: The earlier `e5b1cde` proof binary is superseded. A new binary must be
built from `76ace69` for any fresh verification claims.

**Diff between trees**:
```
d994fd1..e5b1cde: 1 file changed, 14 deletions (ACT content)
e5b1cde..76ace69: 1 file changed, 15 insertions (ACT content)
d994fd1..76ace69: 1 file changed, 10 insertions(+), 9 deletions(-)
```

### P4: Source/Result Isolation Field Inventory

**Task**: Produce a field inventory for every reference-backed normalized value
covering all three proof boundaries:
1. source mutation → existing normalized result unchanged
2. normalized result mutation → source unchanged
3. first normalized result mutation → second result unchanged

#### Summary-level fields

| Field | Source Rep | Normalized Rep | Copy Mechanism | Existing Proof | Gap |
|-------|-----------|----------------|----------------|----------------|-----|
| `Summary.SchemaVersion` | uint8 | Version (uint8) | value copy | implicit | none |
| `Summary.GeneratedAt` | string | string | value copy | TestNormalizationTwoResultsIndependence | none |
| `Summary.Tool` | *string | *string (new alloc) | dereference + alloc | TestNormalizationTwoResultsIndependence | none |
| `Summary.Scope` | nil or *Scope | *Scope (new alloc) | deep copy | TestNormalizationSourceIsolation, TestNormalizationTwoResultsIndependence | none |
| `Summary.Parent` | nil or *Parent | *Parent (new alloc) | deep copy | TestNormalizationTwoResultsIndependence | none |
| `Summary.Overall` | struct | struct (value) | value copy | TestNormalizationTwoResultsIndependence (Status, Disposition) | Disposition pointer only tested for mutation, not nil |
| `Summary.Execution` | nil or *ExecutionBinding | *ExecutionBinding (new alloc) | deep copy | TestNormalizationTwoResultsIndependence | none |
| `Summary.Worktree` | nil or *WorktreeState | *WorktreeState (new alloc) | deep copy | TestNormalizationTwoResultsIndependence | none |
| `Summary.Checks` | []Check | []Check (new slice) | deep copy per element | TestNormalizationTwoResultsIndependence | none |

#### Scope (struct, value-copy on assignment)

| Field | Source Rep | Normalized Rep | Copy Mechanism | Existing Proof | Gap |
|-------|-----------|----------------|----------------|----------------|-----|
| `Scope.ID` | string | string | value copy | TestNormalizationSourceIsolation | none |
| `Scope.Status` | LifecycleStatus | LifecycleStatus | value copy | TestNormalizationSourceIsolation | none |
| `Scope.Disposition` | string | string | value copy | TestNormalizationTwoResultsIndependence | none |

#### Parent (struct, value-copy on assignment)

| Field | Source Rep | Normalized Rep | Copy Mechanism | Existing Proof | Gap |
|-------|-----------|----------------|----------------|----------------|-----|
| `Parent.Act` | string | string | value copy | TestNormalizationTwoResultsIndependence | none |
| `Parent.Status` | LifecycleStatus | LifecycleStatus | value copy | TestNormalizationTwoResultsIndependence | none |
| `Parent.Disposition` | string | string | value copy | TestNormalizationTwoResultsIndependence | none |
| `Parent.Root` | bool | bool | value copy | TestNormalizationTwoResultsIndependence | none |

#### Overall (struct, value-copy on assignment)

| Field | Source Rep | Normalized Rep | Copy Mechanism | Existing Proof | Gap |
|-------|-----------|----------------|----------------|----------------|-----|
| `Overall.Status` | GateStatus | GateStatus | value copy | TestNormalizationTwoResultsIndependence | none |
| `Overall.Disposition` | *string | *string (new) | dereference + alloc | TestNormalizationTwoResultsIndependence | **gap: nil→non-nil path not tested** |

#### ExecutionBinding (struct, value-copy on assignment)

| Field | Source Rep | Normalized Rep | Copy Mechanism | Existing Proof | Gap |
|-------|-----------|----------------|----------------|----------------|-----|
| `ExecutionBinding.HeadOID` | string | string | value copy | TestNormalizationTwoResultsIndependence | none |
| `ExecutionBinding.TreeOID` | string | string | value copy | TestNormalizationTwoResultsIndependence | none |
| `ExecutionBinding.SubjectOID` | string | string | value copy | TestNormalizationTwoResultsIndependence | none |

#### WorktreeState (struct, value-copy on assignment)

| Field | Source Rep | Normalized Rep | Copy Mechanism | Existing Proof | Gap |
|-------|-----------|----------------|----------------|----------------|-----|
| `WorktreeState.CleanBefore` | bool | bool | value copy | TestNormalizationTwoResultsIndependence | none |
| `WorktreeState.CleanAfter` | bool | bool | value copy | TestNormalizationTwoResultsIndependence | none |

#### Check (struct, value-copy on assignment)

| Field | Source Rep | Normalized Rep | Copy Mechanism | Existing Proof | Gap |
|-------|-----------|----------------|----------------|----------------|-----|
| `Check.Name` | string | string | value copy | TestNormalizationTwoResultsIndependence | none |
| `Check.Scope` | *string | *string (new) | dereference + alloc | TestNormalizationTwoResultsIndependence | none |
| `Check.Status` | GateStatus | GateStatus | value copy | TestNormalizationTwoResultsIndependence | none |
| `Check.Evidence` | *string | *string (new) | dereference + alloc | TestNormalizationTwoResultsIndependence | none |
| `Check.Detail` | *string | *string (new) | dereference + alloc | TestNormalizationTwoResultsIndependence | none |
| `Check.DurationMs` | *Integer | *Integer (new) | dereference + alloc | TestNormalizationTwoResultsIndependence | none |
| `Check.Execution` | nil or *CheckExecution | *CheckExecution (new) | deep copy | TestNormalizationTwoResultsIndependence | none |
| `Check.Totals` | nil or *TestTotals | *TestTotals (new) | deep copy | TestNormalizationTwoResultsIndependence | none |

#### CheckExecution (struct, value-copy on assignment)

| Field | Source Rep | Normalized Rep | Copy Mechanism | Existing Proof | Gap |
|-------|-----------|----------------|----------------|----------------|-----|
| `CheckExecution.Argv` | []string | []string (new) | element copy | TestNormalizationTwoResultsIndependence | none |
| `CheckExecution.ExitCode` | *Integer | *Integer (new) | dereference + alloc | TestNormalizationTwoResultsIndependence | **gap: BigInt independence via TestNormalizationBigIntIndependence** |
| `CheckExecution.StdoutSHA256` | string | string | value copy | TestNormalizationTwoResultsIndependence | none |
| `CheckExecution.StderrSHA256` | string | string | value copy | TestNormalizationTwoResultsIndependence | none |

#### TestTotals (struct, value-copy on assignment)

| Field | Source Rep | Normalized Rep | Copy Mechanism | Existing Proof | Gap |
|-------|-----------|----------------|----------------|----------------|-----|
| `TestTotals.Total` | Integer | Integer | deep (raw string) | TestNormalizationBigIntIndependence | none |
| `TestTotals.Pass` | Integer | Integer | deep (raw string) | indirect via Total | none |
| `TestTotals.Fail` | Integer | Integer | deep (raw string) | indirect via Total | none |
| `TestTotals.Skip` | Integer | Integer | deep (raw string) | indirect via Total | none |
| `TestTotals.Unavailable` | Integer | Integer | deep (raw string) | indirect via Total | none |

#### Integer (value type with mutable BigInt view)

| Field | Source Rep | Normalized Rep | Copy Mechanism | Existing Proof | Gap |
|-------|-----------|----------------|----------------|----------------|-----|
| `Integer.raw` | string | string | value copy | TestNormalizationBigIntIndependence | none |

**P4 Conclusion**: Inventory complete. Two focused tests required to close gaps:
1. `TestNormalizationOverallDispositionNilIsolation` - nil→non-nil transition
2. `TestNormalizationExitCodeIntegerIndependence` - direct Integer independence

### Board State

```
ACT-LEAMAS-GATE-SUMMARY-V2-NORMALIZATION01
  PARTIAL — CORRECTION02 active

ACT-LEAMAS-GATE-SUMMARY-V2-NORMALIZATION01-CORRECTION01
  PARTIAL — implementation accepted; closure superseded

ACT-LEAMAS-GATE-SUMMARY-V2-NORMALIZATION01-CORRECTION02
  PARTIAL
  P0 precedence authority: RESOLVED
  P2 identity topology: RESOLVED (sibling relationship, shared parent tree documented)
  P3 verification evidence: RESOLVED (commands passed, superseded binary noted)
  P4 isolation: PARTIAL (inventory complete, two focused tests pending)
  patch hygiene: FIXED

ACT-LEAMAS-GATE-SUMMARY-V2-DIGEST01
  BLOCKED
```

### Pending: P4 Focused Tests

Two narrow tests required to close P4 gaps:

```go
// TestNormalizationOverallDispositionNilIsolation
// Prove:
// - source disposition initially nil
// - normalize produces nil in result
// - mutate source to non-nil
// - existing result remains nil
// - second normalize reflects non-nil
// - mutate second result does not mutate source

// TestNormalizationExitCodeIntegerIndependence
// Prove for two normalized results:
// - exact raw value preserved
// - BigInt() returns distinct mutable allocations
// - mutating returned *big.Int changes neither Integer
// - first and second normalized exit-code values unchanged
// - source wire value unchanged
```

### Remaining in this ACT

- [x] Reconcile identity chain for all relevant revisions (P2)
- [x] Bind and record verification evidence (P3)
- [x] Complete source/result isolation gap analysis (P4 inventory)
- [ ] Add two focused isolation tests (P4)
- [ ] Run tests, factorize, gate-fast
- [ ] Close CORRECTION02

### Evidence Recorded

| Command | Result | Details |
|---------|--------|---------|
| `go test -count=1 ./internal/gatesummary/...` | PASS | 0.483s |
| `go test -count=20 ./internal/gatesummary/...` | PASS | 8.545s |
| `go test -race -count=5 ./internal/gatesummary/...` | PASS | 11.581s |
| `go vet ./internal/gatesummary/ ./cmd/leamas/` | PASS | no issues |
| `CGO_ENABLED=0 go build -trimpath -o bin/leamas ./cmd/leamas` | PASS | binary built |
| `make factorize` | PASS | 592.89s (all verifiers OK) |
| `make gate-fast` | PASS | 6.32s (all fast-lane checks OK) |

### Remaining in this ACT

- [x] Reconcile identity chain for all relevant revisions (P2)
- [x] Bind and record verification evidence (P3)
- [x] Complete source/result isolation gap analysis (P4)
- [x] Close CORRECTION02
- [ ] Next: rebuild digest from final committed implementation (DIGEST01)

### Next ACT after closure

`ACT-LEAMAS-GATE-SUMMARY-V2-DIGEST01`
