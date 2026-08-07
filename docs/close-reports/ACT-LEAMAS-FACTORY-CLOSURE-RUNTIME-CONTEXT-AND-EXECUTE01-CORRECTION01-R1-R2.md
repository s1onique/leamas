# ACT-LEAMAS-FACTORY-CLOSURE-RUNTIME-CONTEXT-AND-EXECUTE01-CORRECTION01-R1-R2

## ID

ACT-LEAMAS-FACTORY-CLOSURE-RUNTIME-CONTEXT-AND-EXECUTE01-CORRECTION01-R1-R2

## STATUS

PASS (foundation proofs closed)

## BASE

```text
BASE_COMMIT=f5e4a465c07c4481303208ba2608f960b5145503
FINAL_COMMIT=209f375
FINAL_TREE=<see commit>
WORKTREE_STATUS=clean
```

## Proof claims (literal)

```text
RUNTIME_CONTEXT_HAPPY_PATH=PASS
RUNTIME_CONTEXT_HAPPY_PATH_PRESENT=true
RUNTIME_CONTEXT_HAPPY_PATH_PASS=true
FAKE_GIT_SUPPORTS_FROZEN_PLAN_LOOKUP=true
RAW_PLAN_TRAILING_NEWLINE_PROVED=true (raw bytes are preserved verbatim)

GATE_REQUEST_FULL_IDENTITY=PASS
GATE_REQUEST_IDENTITY_ALL_FIELDS=true
GATE_REQUEST_ARGV_ORDERED=true
GATE_REQUEST_MISMATCH_CALLS_REMAIN_ONE=true
GATE_REQUEST_MISMATCH_MATRIX=PASS

START_FAILURE_STDOUT_EMPTY=true
START_FAILURE_STDERR_NONEMPTY=true
START_FAILURE_STDERR_DIAGNOSTIC=true
START_FAILURE_EXIT_127=true
START_FAILURE_ERR_NONNIL=true

WAIT_DELAY_RETENTION_TEST=PASS (default 5s)
WAIT_DELAY_BOUND_PROVED=true
```

## Phase results

### Phase 1: RUNTIME_CONTEXT_HAPPY_PATH_RESTORED

The `happy_path` sub-test in `TestClosureRuntimeContextMatrix`
was restored. The fake `runtimeRevParseFake` now handles
`git rev-parse --verify --end-of-options F:P` and returns a
synthetic 40-char OID. The resolver's full production sequence
is exercised end-to-end:

```text
git status --porcelain=v2
git rev-parse --show-object-format
git rev-parse --verify --end-of-options <freeze>^{commit}
git rev-parse --verify --end-of-options <subject>^{commit}
git rev-parse --verify --end-of-options <freeze>^{tree}
git rev-parse --verify --end-of-options <subject>^{tree}
git merge-base --is-ancestor <freeze> <subject>
git rev-parse --verify --end-of-options <freeze>:<plan-path>
git cat-file blob <plan-blob>
```

All matrix cases pass:
- happy_path: PASS
- dirty_worktree: PASS
- unsupported_format: PASS
- freeze_not_ancestor: PASS
- freeze_equals_subject: PASS

### Phase 2: GATE_REQUEST_FULL_IDENTITY

`sameGateRequest` now compares every identity-bearing field:

```text
RepositoryRoot
SubjectRoot
EvidenceDir
RunID
MakeExecutable argv element-by-element in declaration order
```

Two calls that differ in any of these fields return
`ErrCollectorRequestMismatch`. Runner invocations remain
exactly 1.

### Phase 3: START_FAILURE_STREAM_SEMANTICS

`OsRunner.Run` on `cmd.Start()` failure now writes the
diagnostic to `Stderr` (not `Stdout`) and returns:

```text
ExitCode=127
Err != nil
Stdout=[]
Stderr=<non-empty diagnostic>
TimedOut=false
Canceled=false
StdoutTrunc=false
StderrTrunc=false
```

`OsRunner.WaitDelay` defaults to 5 seconds, bounding
post-cancellation process delay and inherited/unclosed
I/O-pipe delay.

## LOCAL_GATES

```text
go build ./... OK
go vet ./internal/factory/closure/... ./cmd/leamas/... OK
gofmt -l <ACT-owned files> OK
go test -count=1 ./internal/factory/closure/evidence/ PASS
go test -count=1 -run TestClosureRuntimeContextMatrix PASS
```

## REFUSED_EXPENSIVE_GATES

```text
make gate, make factorize, make gate-dupcode
```

## UNRESOLVED_BLOCKERS

```text
subject worktree, frozen-plan execution, real binary authority,
caller-state proof, coordinated publication, and hermetic dogfood
remain for CORRECTION02
```

## CLINEMM_FILES_CHANGED

```text
none
```

## Follow-up ACTs

```text
ACT-LEAMAS-FACTORY-CLOSURE-RUNTIME-CONTEXT-AND-EXECUTE01-CORRECTION02
```

CORRECTION02 owns subject-worktree authority, frozen-plan
execution against S^{tree}, real exact-subject binary authority,
caller-state proof, coordinated publication, and hermetic
end-to-end dogfood.
