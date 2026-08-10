# ACT-LEAMAS-FACTORY-CLOSURE-PROTOCOL-V2-SUBJECT-OBSERVATION-AUTHORITY01 (R6-A)

## Status

**OPEN — R6-A / SUBJECT OBSERVATION AUTHORITY**

This ACT replaces the failed monolithic R6 with one authority
boundary only:

```text
production GitV2SubjectExecutor
        ↓
authoritative SubjectExecutionResult
```

No B1 binary integration.

No GateCollector integration.

No public CLI work.

No B2/B3 publication changes.

No lifecycle/tag work.

## Base

```text
BASE_COMMIT=f92102c5c7b4748abdce68d8741b8262eb14fd43

CORRECTION02_A=PASS
CORRECTION02_B1=PASS
CORRECTION02_B2=PASS
CORRECTION02_B3=PARTIAL

R6_MONOLITHIC=PARTIAL_NO_COMMIT
R6_A_IMPLEMENTED=true
R6_B_READY=true
```

## Mission

Make the production subject executor return every fact that can
only be observed while the detached S worktree exists.

After this ACT, downstream code MUST NOT need to reconstruct
subject authority from:

```text
repository root
manifest
topology guesses
hard-coded booleans
post-cleanup filesystem inspection
synthetic hashes
```

The executor owns the live-worktree lifecycle and therefore owns
these observations.

## Precondition (executed)

The current checkout contained untracked R5/R6 planning/close-report
documents. Before implementation:

```text
1. mkdir -p /tmp/leamas-r6a-staging                          # done
2. copy all current untracked R5/R6 planning/close-report docs
   to /tmp/leamas-r6a-staging/                              # done
3. remove those untracked copies from the checkout           # done
4. assert:
     git status --porcelain
   is empty                                                  # done
5. assert:
     git rev-parse HEAD
   ==
     f92102c5c7b4748abdce68d8741b8262eb14fd43                # done
```

Implementation started from the clean base.

## Hard scope (enforced)

Allowed production surface:

```text
GitV2SubjectExecutor
SubjectExecutionResult / related internal observation types
existing bounded Git observation helpers
existing worktree inventory parser/authority
tests for those surfaces
```

Allowed additional Git observations:

```text
only through the existing bounded Git authority
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

## Phase 1 — authoritative result type (delivered)

The production V2ExecuteResult now carries every fact the live
detached subject worktree can yield. New typed fields:

```text
SubjectWorktreePath        string
SubjectHead                string
SubjectTree                string
SubjectDetached            bool

StatusObservation          SubjectByteObservation
RefsObservation            SubjectByteObservation

WorktreeInventoryBefore    SubjectWorktreeInventory
WorktreeInventoryAtSubject SubjectWorktreeInventory
WorktreeInventoryAfter     SubjectWorktreeInventory

SubjectRegistration         SubjectWorktreeRegistration
SubjectRegistrationAvailable bool

TopologyFacts              V2TopologyFacts

SubjectCleanupObserved     bool
SubjectCleanupError        string

SubjectObservationDiagnostics V2Diagnostics
```

The legacy ObservedTree/CheckResults/Evidence/CleanupError fields
are preserved unchanged. New fields are additive so existing
callers keep their wire contract.

## Phase 2 — typed observations (delivered)

`SubjectByteObservation` is the canonical carrier:

```text
Available=false  -> observation was not attempted, or the
                    underlying Git command failed. Error
                    carries a typed diagnostic.
Available=true   -> observation succeeded. Bytes is the
                    exact observed payload, possibly empty.
                    Empty bytes are a legitimate result
                    (e.g. clean porcelain status) and MUST
                    NOT be encoded as "unavailable".
```

Required regression:

```text
TestSubjectObservationEmptyBytesAreObserved
```

## Phase 3 — production creates S worktree (delivered)

The production `GitV2SubjectExecutor` creates its normal detached
worktree for S. The captured path is the `SubjectWorktreePath`
field of the result.

## Phase 4 — live S identity (delivered)

`observeLiveSubjectIdentity` captures:

```text
git -C <S-worktree> rev-parse HEAD
git -C <S-worktree> rev-parse HEAD^{tree}
git -C <S-worktree> rev-parse --show-toplevel
git -C <S-worktree> symbolic-ref -q HEAD
```

The detached state is established by the canonical exit code of
`symbolic-ref -q HEAD`: exit 0 means the HEAD is a symbolic
ref (NOT detached); non-zero exit means the HEAD is detached.
A non-zero exit is the authoritative detached signal and is
NOT collapsed into a generic "git error".

Required umbrella:

```text
TestSubjectExecutorObservesLiveDetachedAuthority
```

## Phase 5 — status authority (delivered)

`observeSubjectStatus` captures
`git -C <S-worktree> status --porcelain=v2 --untracked-files=all`.
The function is fail-closed. Empty bytes are a legitimate
result (clean worktree) and MUST be encoded as
Available=true, Bytes="".

## Phase 6 — refs authority (delivered)

`observeSubjectRefs` reuses the existing canonical refs authority
(`snapshotCallerRefs`) applied to the subject worktree. The
act forbids a second refs representation; this helper IS the
canonical authority.

## Phase 7 — worktree inventory snapshots (delivered)

`observeSubjectWorktreeInventory` runs
`git worktree list --porcelain -z` through the existing bounded
Git authority. The helper is captured three times in
production:

```text
BEFORE:    immediately before production adds S worktree
AT_SUBJECT: after S worktree creation and live authority capture
AFTER:     after production removes S worktree
```

Each observation preserves canonical (Path, HEAD) registration
identity.

## Phase 8 — subject registration binding (delivered)

From `WorktreeInventoryAtSubject`, the executor binds the exact
registration corresponding to `SubjectWorktreePath`. The
mismatch row produces the typed
`V2CodeSubjectRegistrationMismatch` diagnostic.

Required umbrella:

```text
TestSubjectWorktreeRegistrationBindsPathAndHead
```

## Phase 9 — no-leak cleanup proof (delivered)

The existing subject executor remains the owner of S-worktree
cleanup. After cleanup:

```text
SubjectWorktreePath does not exist
SubjectWorktreePath absent from WorktreeInventoryAfter
WorktreeInventoryAfter equals WorktreeInventoryBefore
semantically (Path, HEAD)
```

Required umbrella:

```text
TestSubjectExecutorCleanupRestoresWorktreeInventory
```

## Phase 10 — cleanup observation (delivered)

```text
SubjectCleanupObserved   bool
SubjectCleanupError      string
```

Contract:

```text
successful cleanup:
  SubjectCleanupObserved=true, SubjectCleanupError=""
cleanup failure:
  SubjectCleanupObserved=true, SubjectCleanupError non-empty
cleanup never attempted / observation unavailable:
  SubjectCleanupObserved=false
```

Downstream mapping into B2 remains R6-B work.

## Phase 11 — topology fact transport (delivered)

The subject result carries the already-established topology
facts that governed execution via `V2TopologyFacts`. The
executor does NOT recompute topology from subject
observations and does NOT hard-code any relation.

Required regression:

```text
TestSubjectExecutionResultCarriesTopologyFacts
```

## Phase 12 — execute checks unchanged (delivered)

Existing plan-check behavior is unchanged.

Required regression:

```text
TestSubjectObservationDoesNotChangeCheckExecution
```

## Phase 13 — lifetime contract (delivered)

Frozen in production and tests:

```text
inventory BEFORE
    ↓
create detached S worktree
    ↓
capture live S identity
    ↓
capture status / refs / inventory AT_SUBJECT
    ↓
execute checks
    ↓
capture required pre-cleanup observations
    ↓
remove S worktree
    ↓
inventory AFTER
    ↓
return SubjectExecutionResult
```

After `ExecuteSubjectChecks` returns:

```text
SubjectWorktreePath is historical evidence only
```

Downstream code MUST NOT expect the directory to remain usable.

## Phase 14 — producer umbrella (delivered)

```text
TestClosureSubjectObservationAuthority
```

Asserts every R6-A fact in one execution.

## Phase 15 — adversarial matrix (delivered)

```text
TestClosureSubjectObservationFailureMatrix
```

Rows: rev-parse HEAD, HEAD^{tree}, show-toplevel, detached,
status, refs, registration HEAD != S. Every row fails closed
with one stable typed diagnostic family.

## Phase 16 — canonical worktree inventory identity (delivered)

```text
same set / different order -> equal
same path / different HEAD -> not equal
different path / same HEAD -> not equal
```

Required regression:

```text
TestSubjectWorktreeInventoryEqualMatrix
TestSubjectWorktreeInventoryHermeticRoundTrip
TestSubjectWorktreeInventoryParserRejectsMatrix
```

## Phase 17 — no downstream integration (enforced)

R6-A does NOT wire the new result into:

```text
BuildExactSubjectBinary
GateCollector
V2ExecutionObservation
BuildClosureEvidenceCandidate
PrepareClosureEvidenceForPublication
EvidencePublication
factory close execute
```

## Phase 18 — source hygiene (enforced)

Every R6-A-owned Go file is <= 400 lines:

```text
 151 subject_observation_types.go
 187 subject_observation_inventory.go
 202 subject_observation.go
 138 subject_execution_types.go
 245 subject_execution_observation.go
 376 closure_protocol_v2_executor.go
```

Test files split by concern:

```text
subject_observation_test.go         (umbrellas)
subject_observation_failures_test.go (adversarial matrix)
subject_observation_inventory_test.go (canonical identity + parser)
```

## Verification

Run:

```text
gofmt -w <R6-A-owned Go files>        # PASS
go test -count=1                      # PASS for R6-A suites
go vet ./internal/factory/closure/   # PASS
git diff --check                      # clean
```

Also run the relevant existing subject-execution umbrellas:

```text
TestClosureSubjectWorktreeAuthority
TestClosureExecuteChecksAgainstSubjectTree
TestClosureExecuteChecksAgainstSubjectTree_ExcludeSemantics
TestClosureWaitDelayRetainedPipe
```

Focused Factory verifiers:

```text
llm-friendly
tooling-boundaries
long-test-policy
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

SUBJECT_OBSERVATION_AUTHORITY_TYPED=true

SUBJECT_WORKTREE_PATH_ACTUAL=true
SUBJECT_HEAD_EQUALS_S=true
SUBJECT_TREE_EQUALS_S_TREE=true
SUBJECT_DETACHED_OBSERVED=true

STATUS_OBSERVATION_ACTUAL=true
EMPTY_STATUS_IS_OBSERVED=true

REFS_OBSERVATION_ACTUAL=true

WORKTREE_INVENTORY_BEFORE_ACTUAL=true
WORKTREE_INVENTORY_AT_S_ACTUAL=true
WORKTREE_INVENTORY_AFTER_ACTUAL=true

WORKTREE_REGISTRATION_BINDS_PATH_AND_HEAD=true
WORKTREE_IDENTITY_PATH_ONLY=false

SUBJECT_CLEANUP_OBSERVED=true
SUBJECT_WORKTREE_LEAK=false

TOPOLOGY_FACTS_TRANSPORTED=true
TOPOLOGY_SEMANTICS_UNCHANGED=true

CHECK_EXECUTION_SEMANTICS_UNCHANGED=true

FAILURE_MATRIX=PASS
UNAVAILABLE_NEVER_SYNTHESIZED=true

ALL_R6_A_FILES_LE_400=true
PRODUCTION_WITH_TESTS=true

B1_TOUCHED=false
B2_TOUCHED=false
B3_PUBLICATION_TOUCHED=false
GATECOLLECTOR_TOUCHED=false
PUBLIC_CLI_TOUCHED=false

CLINEMM_FILES_CHANGED=none
WORKTREE_STATUS=clean

R6_A=PASS
R6_B_READY=true
```

## Successor

Only after R6-A PASS:

```text
ACT-LEAMAS-FACTORY-CLOSURE-PROTOCOL-V2-
BINARY-GATE-INTEGRATION01
```

R6-B owns only:

```text
BuildExactSubjectBinary before live S window
exact-S binary lifetime
GateCollector exactly once inside live S window
mapping B1 + GateCapture + SubjectExecutionResult
into authoritative V2ExecutionObservation
proving B2 COMPLETE becomes achievable
```

R6-B MUST NOT implement the public exit 0/2/3/4 matrix.

That remains R6-C.
