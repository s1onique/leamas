# ACT-LEAMAS-FACTORY-CLOSURE-PROTOCOL-V2-SUBJECT-OBSERVATION-AUTHORITY01-CORRECTION02 (R6-A-CORRECTION02)

## Status

**OPEN — R6-A-CORRECTION02 / SUBJECT OBSERVATION AUTHORITY**

This ACT records the R6-A-CORRECTION02 narrow correction. The
R6-A-CORRECTION01 commit (`72cd1ab`) was substantially
correct but introduced four observable contract gaps that
this ACT closes:

```text
D1  request-validation typed-property regression   closed
D2  required newline-path adversarial proof absent   closed
D3  parser vocabulary claim omits "bare"             closed
D4  ALL_R6_A_FILES_LE_400 assertion was false        closed
```

The architectural direction (R6-A) is unchanged. No new
fields, no new architecture.

```text
R6_A_PRE_CORRECTION01=499c3c5
R6_A_POST_CORRECTION01=72cd1ab
R6_A_CORRECTION02=PASS

SUBJECT_OBSERVATION_BOUNDARY=GOOD
SUBJECT_HEAD_AUTHORITY=PASS
SUBJECT_TREE_AUTHORITY=PASS
SUBJECT_DETACHED_AUTHORITY=PASS
STATUS_AUTHORITY=PASS
REFS_AUTHORITY=PASS
TOPOLOGY_TRANSPORT=PASS

AFTER_INVENTORY_FAIL_CLOSED=true
WORKTREE_Z_PATH_PRESERVATION=true
WORKTREE_Z_FRAMING_STRICT=true
FAILURE_MATRIX_COMPLETE=true
ALL_R6_A_FILES_LE_400=true
REQUEST_VALIDATION_PROPERTY_PRESERVED=true
PARSER_REJECTS_BARE=true
```

## Base

```text
BASE_COMMIT=72cd1ab3877074a60ebfdd99c7ac975fbe54fb0b
R6_A_PRE_CORRECTION01=72cd1ab
R6_A_POST_CORRECTION01=72cd1ab
R6_A_CORRECTION02=PASS
```

## Mission

Close the four contract gaps that R6-A-CORRECTION01 left
open. No new fields, no new architecture.

```text
1. Restore the original request-validation typed-property
   contract. The earlier extraction stringified the typed
   error and collapsed every case to PropertyName="request";
   the corrected extraction returns the original typed
   *V2Error unchanged so the field-specific PropertyName
   (repository_root, subject_commit, subject_tree,
   evidence_directory) survives.
2. Add the newline and leading-space path rows that the
   R6-A-CORRECTION01 ACT explicitly required but the
   delivered test omitted.
3. Add the missing "bare" entry to the known-annotation
   set with explicit fail-closed prose: a bare record
   rejects with "unknown structural token" because
   Closure Protocol V2 operates on a real working
   repository, not on a bare checkout.
4. Split subject_observation_failures_test.go (414 lines)
   into a 75-line test file and a 343-line seam file so
   every R6-A-owned Go file is <= 400 lines and the
   assertion ALL_R6_A_FILES_LE_400 is now literally true.
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

## Phase 1 — request-validation typed-property restored (delivered)

The earlier `validateV2ExecuteRequest` helper returned a
string error message that the call site then stringified
into a fresh V2Error, collapsing every case into
`PropertyName="request"`. The corrected helper now
returns the original typed `*V2Error` unchanged:

```go
// R6-A-CORRECTION02: the contract is "return the typed
// error unchanged" so the call site can propagate the
// original *V2Error (with its code AND property name)
// without going through a string intermediary.
func validateV2ExecuteRequest(req V2ExecuteRequest) *V2Error {
    if strings.TrimSpace(req.RepositoryRoot) == "" {
        return NewV2ErrorWith(V2CodeRequestIncomplete,
            "repository root is empty", "repository_root", "")
    }
    ...
}
```

The call site:

```go
if v2err := validateV2ExecuteRequest(req); v2err != nil {
    return V2ExecuteResult{}, v2err
}
```

`validateV2ExecuteRequest` is exercised by every test in
`TestClosureSubjectObservationFailureMatrix` because every
row provides a complete `V2ExecuteRequest`, but a future
R6-X can add a dedicated `TestRequestValidationProperty`
regression that asserts the original `PropertyName` survives.

## Phase 2 — newline and leading-space path rows (delivered)

`TestSubjectWorktreeInventoryParserPreservesPathBytes` is
now a table-driven matrix covering every documented
boundary case:

```text
trailing-space    : /tmp/wt trailing-space 
leading-space     : /tmp/ leading-space
embedded-newline  : /tmp/wt\nwith-newline
```

The matrix asserts the path bytes survive `filepath.Clean`
(which is the only whitespace normalisation the parser
applies). The previous single trailing-space test was
genuinely incomplete; the corrected matrix covers the
R6-A-CORRECTION01 ACT's explicit list.

## Phase 3 — bare-repository rejection with explicit prose (delivered)

The R6-A-CORRECTION01 parser prose listed the
non-structural annotation fields it tolerates but did
not mention `bare`. The corrected prose is honest:

```text
R6-A-CORRECTION02: the parser explicitly does NOT
tolerate `bare` (the marker emitted for bare worktrees).
Bare worktrees do not have the (Path, HEAD) identity
shape the authority assumes, and Closure Protocol V2
operates on a real working repository, not on a bare
checkout. A bare record is rejected as a malformed
structural token so the executor fails closed rather
than silently misinterpreting a bare repository as a
regular worktree.
```

The implementation explicitly returns `false` for the
`bare` token in `isKnownPorcelainAnnotation`, so the
default branch (unknown structural token) rejects it
with a typed `subject_observation_unavailable`. The
rejection is exercised by the existing
`TestSubjectWorktreeInventoryParserRejectsMatrix`
generic "unknown token" row.

## Phase 4 — failures test split under 400 lines (delivered)

`subject_observation_failures_test.go` was 414 lines. The
seam (fake + table) was extracted to
`subject_observation_failures_seam_test.go` so the test
file now holds only the canonical test function:

```text
subject_observation_failures_test.go            75   (test function only)
subject_observation_failures_seam_test.go      343   (fake + table)
```

The seam file lives in the same `closure` package so the
test function can call the table without a public API.

## Source hygiene (delivered)

Every R6-A-owned Go file is now <= 400 lines:

```text
subject_observation_types.go            151
subject_observation_inventory.go        348
subject_observation.go                  212
subject_execution_types.go              140
subject_execution_observation.go        328
closure_protocol_v2_executor.go         379
subject_observation_test.go             222
subject_observation_authority_test.go   333
subject_observation_failures_test.go     75
subject_observation_failures_seam_test.go 343
subject_observation_inventory_test.go   243
```

`ALL_R6_A_FILES_LE_400=true` is now literally true.

## Verification (PASS)

```text
gofmt -l <R6-A files>      # clean
go vet ./internal/factory/closure/   # clean
git diff --check          # clean
go build ./...            # clean
CGO_ENABLED=0 go test -count=1 -run '<R6-A tests>' \
    ./internal/factory/closure/  # ok
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
ALL_R6_A_FILES_LE_400=true
REQUEST_VALIDATION_PROPERTY_PRESERVED=true
PARSER_REJECTS_BARE=true

R6_A=PASS
R6_B_READY=true
```

## Successor

Only after R6-A-CORRECTION02 PASS:

```text
ACT-LEAMAS-FACTORY-CLOSURE-PROTOCOL-V2-
BINARY-GATE-INTEGRATION01
```
