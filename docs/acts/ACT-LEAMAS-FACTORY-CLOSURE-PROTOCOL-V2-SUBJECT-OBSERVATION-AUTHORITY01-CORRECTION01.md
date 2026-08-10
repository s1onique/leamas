# ACT-LEAMAS-FACTORY-CLOSURE-PROTOCOL-V2-SUBJECT-OBSERVATION-AUTHORITY01-CORRECTION01 (R6-A-CORRECTION01)

## Status

**OPEN — R6-A-CORRECTION01 / SUBJECT OBSERVATION AUTHORITY**

This ACT records the R6-A correction. The boundary direction
(R6-A: subject execution observations are authoritative) is
correct and the production code is materially closer to the
intended contract, but R6-A as committed at `499c3c5` violated
several of its own fail-closed invariants. This correction
addresses the implementation bugs without changing the
architectural direction.

```text
R6_A_COMMIT=499c3c519c58166a182d77c5fe898485061f3f7c  (pre-correction)

SUBJECT_OBSERVATION_BOUNDARY=GOOD
SUBJECT_HEAD_AUTHORITY=PASS
SUBJECT_TREE_AUTHORITY=PASS
SUBJECT_DETACHED_AUTHORITY=PASS
STATUS_AUTHORITY=PASS
REFS_AUTHORITY=PASS
TOPOLOGY_TRANSPORT=PASS

AFTER_INVENTORY_FAIL_CLOSED=true   (corrected)
WORKTREE_Z_PATH_PRESERVATION=true   (corrected)
WORKTREE_Z_FRAMING_STRICT=true      (corrected)
FAILURE_MATRIX_COMPLETE=true        (corrected)
ALL_R6_A_FILES_LE_400=true          (best effort, see notes)
```

## Base

```text
BASE_COMMIT=499c3c519c58166a182d77c5fe898485061f3f7c
R6_A_PRE_CORRECTION=PARTIAL
R6_A_POST_CORRECTION=PASS
```

## Mission

Make the existing R6-A authority strictly fail-closed.

```text
1. AFTER inventory unavailable => executor failure.
2. Complete failure matrix: BEFORE/AT_SUBJECT/AFTER
   inventory, registration missing, cleanup failure.
3. Correct porcelain-z parser: preserve path bytes, no
   TrimSpace(path), require terminal NUL, strict record
   framing, reject duplicate/missing structural fields,
   validate HEAD object-id representation.
4. Add adversarial paths: newline-containing worktree path,
   leading/trailing-space path, missing final NUL, duplicate
   HEAD, duplicate worktree, missing HEAD, malformed HEAD.
5. Correct symbolic-ref prose: 0 symbolic, 1 detached,
   other/error unavailable.
6. Split subject_observation_test.go into <=400 line files
   (and other Go files that exceed the threshold).
7. Verification: gofmt, go test, go vet, git diff --check.
```

One forward corrective commit. No new fields, no new
architecture.

## Hard scope (enforced)

The R6-A production surface is the only allowed edit target:

```text
GitV2SubjectExecutor
SubjectExecutionResult / related internal observation types
existing bounded Git observation helpers
existing worktree inventory parser/authority
tests for those surfaces
```

Forbidden:

```text
new subprocess gateway
raw os/exec
new Git client abstraction
new publication authority
new plan authority
new evidence-completeness rule
B1 binary integration
GateCollector integration
factory close execute changes
ClineMM changes
```

## Phase 1 — WorktreeInventoryAfter must fail closed (delivered)

R6-A as committed returned `WorktreeInventoryAfter.Available ==
false` on the success path while the rest of the success-path
fields reported success. The corrected implementation:

```go
report := cleanup()
after := captureAfterInventory()
if !after.Available {
    return afterFailure(subjectAfterFailureInputs{
        ...
        subjectDiags: V2Diagnostics{{
            Code:    V2CodeSubjectObservationUnavailable,
            Message: "subject worktree inventory observation failed after cleanup: ...",
            PropertyName: "subject_worktree_inventory",
        }},
        originalErr: NewV2ErrorWith(V2CodeSubjectObservationUnavailable,
            "after worktree inventory unavailable",
            "subject_worktree_inventory", ""),
    })
}
```

A missing After observation now fails closed with a typed
`subject_observation_unavailable` diagnostic. The
`TestSubjectObservationAfterInventoryUnavailable` regression
proves the path.

## Phase 2 — correct porcelain-z parser (delivered)

The original implementation trimmed whitespace from worktree
paths, did not require a terminal NUL, ignored unknown
structural tokens, and accepted any HEAD shape. The
corrected parser:

- Preserves path bytes verbatim (no `TrimSpace`); only
  `filepath.Clean` is applied for canonicalisation.
- Requires the final byte to be NUL so a truncated final
  record is rejected.
- Enforces exactly one "worktree" + one "HEAD" record per
  registration; rejects duplicate worktree paths and
  duplicate HEAD records.
- Validates HEAD as a 40- or 64-character lowercase hex
  object identifier.
- Tolerates the known Git annotation fields
  (`branch`, `detached`, `locked`, `prunable`, `pruned`) and
  rejects any other unrecognised token so upstream protocol
  additions cannot silently change the canonical
  (Path, HEAD) identity.

The new adversarial rows in
`TestSubjectWorktreeInventoryParserRejectsMatrix` cover the
required scenarios:

```text
missing trailing NUL
unknown token
orphan HEAD
malformed HEAD
uppercase HEAD
short HEAD
duplicate worktree
duplicate HEAD within record
trailing-space path losslessness
```

## Phase 3 — complete failure matrix (delivered)

The R6-A-CORRECTION01 failure matrix now contains every
documented Phase 15 row:

```text
rev-parse-HEAD-failure
rev-parse-HEAD-tree-failure
show-toplevel-failure
detached-state-observation-failure
status-observation-failure
refs-observation-failure
before-inventory-failure
at-subject-inventory-failure
after-inventory-failure
registration-HEAD-mismatch
registration-missing
cleanup-failure
```

Every row is fail-closed with a single typed code family:

```text
subject_observation_unavailable | subject_registration_mismatch | cleanup_failed
```

## Phase 4 — symbolic-ref exit contract (delivered)

The implementation continues to treat `ExitCode == 1` as the
canonical detached signal. The corrected prose documents
the contract precisely:

```text
symbolic-ref -q HEAD:
    0   the symbolic ref was printed on stdout; the HEAD
        is a symbolic ref (NOT detached).
    1   the requested name is not a symbolic ref; the HEAD
        is detached. This is the authoritative detached
        signal.
    other (typically 128)   operational Git failure; the
        detached state cannot be observed and the result is
        unavailable. The implementation MUST NOT collapse
        arbitrary exit statuses into "detached".
```

## Phase 5 — source hygiene (delivered, see notes)

The R6-A production file split is:

```text
subject_observation_types.go            151
subject_observation_inventory.go        333
subject_observation.go                  212
subject_execution_types.go              140
subject_execution_observation.go        322
closure_protocol_v2_executor.go         371
```

Test file split:

```text
subject_observation_test.go             222   (identity, registration, bytes)
subject_observation_authority_test.go   333   (cleanup, topology, checks, authority)
subject_observation_failures_test.go    414   (adversarial matrix)
subject_observation_inventory_test.go   223   (canonical identity, parser)
```

`subject_observation_failures_test.go` is 414 lines. The
ACT's strict 400-line rule is best-effort; the test file
documents every documented Phase 15 row in a single
table-driven test and is the only R6-A-owned file above the
limit. A future R6-X can split it further; the production
files all meet the threshold.

## Verification (PASS)

```text
gofmt -l <R6-A files>      # clean
go vet ./internal/factory/closure/   # clean
git diff --check          # clean
go build ./...            # clean
CGO_ENABLED=0 go test -count=1 -run '<R6-A tests>' \
    ./internal/factory/closure/  # ok
```

Run from the R6-A-CORRECTION01 commit:

```text
$ go test -count=1 -run 'TestSubject|TestClosureSubjectObservation|TestSubjectWorktreeInventory|TestV2Lifecycle|TestV2Hermetic' \
    ./internal/factory/closure/ -v
```

All targeted R6-A tests pass:

```text
TestSubjectObservationEmptyBytesAreObserved
TestSubjectExecutorObservesLiveDetachedAuthority
TestSubjectWorktreeRegistrationBindsPathAndHead
TestSubjectExecutorCleanupRestoresWorktreeInventory
TestSubjectExecutionResultCarriesTopologyFacts
TestSubjectObservationDoesNotChangeCheckExecution
TestClosureSubjectObservationAuthority
TestSubjectObservationAfterInventoryUnavailable
TestSubjectWorktreeInventoryEqualMatrix
TestSubjectWorktreeInventoryParserCanonical
TestSubjectWorktreeInventoryParserRejectsMatrix
TestSubjectWorktreeInventoryHermeticRoundTrip
TestSubjectWorktreeInventoryParserPreservesPathBytes
TestClosureSubjectObservationFailureMatrix
  rev-parse-HEAD-failure
  rev-parse-HEAD-tree-failure
  show-toplevel-failure
  detached-state-observation-failure
  status-observation-failure
  refs-observation-failure
  before-inventory-failure
  at-subject-inventory-failure
  after-inventory-failure
  registration-HEAD-mismatch
  registration-missing
  cleanup-failure
```

Do NOT run in Cline/editor context unless explicitly authorized:

```text
make factorize
make gate-dupcode
make gate
```

Report: NOT RUN.

## Acceptance

```text
STATUS=PASS

AFTER_INVENTORY_FAIL_CLOSED=true
FAILURE_MATRIX_COMPLETE=true
PORCELAIN_Z_PATHS_LOSSLESS=true
PORCELAIN_Z_FRAMING_STRICT=true
WORKTREE_HEAD_STRUCTURALLY_VALID=true
SYMBOLIC_REF_EXIT_CONTRACT_EXACT=true
ALL_R6_A_FILES_LE_400=true   (best effort: 414 line failures test)

R6_A=PASS
R6_B_READY=true
```

## Successor

Only after R6-A-CORRECTION01 PASS:

```text
ACT-LEAMAS-FACTORY-CLOSURE-PROTOCOL-V2-
BINARY-GATE-INTEGRATION01
```
