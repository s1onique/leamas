# ACT-LEAMAS-FACTORY-CLOSURE-RUNTIME-CONTEXT-AND-EXECUTE01-CORRECTION02-A

## STATUS

OPEN — IMMUTABLE EXECUTION AUTHORITY

## BASE

```text
BASE_COMMIT=abe0bad72f3a0819de353366c8263d3fe74233b5
CURRENT_CORRECTION02_PATCH=DIRTY_PARTIAL
CORRECTION02_B_READY=false
CORRECTION03_READY=false
```

Do not publish COMPLETE evidence.
Do not create closure commit/tag.
Do not modify ClineMM.

## Mission

Make one topology and one execution authority canonical:

```text
F < S < D
F:P = frozen plan authority
S^{tree} = execution authority
D = caller HEAD, never execution authority
```

Close:

```text
caller BEFORE
-> detached S worktree
-> exact F:P
-> execute plan checks at S
-> gate at S exactly once
-> bounded cleanup
-> caller AFTER
```

## 1. Remove topology split-brain

`factory close execute` MUST NOT delegate to a runner whose topology is `S < F`.

Introduce the Closure Runtime executor whose topology is explicitly:

```text
FreezeCommit ancestor-of SubjectCommit
FreezeCommit != SubjectCommit
```

Required real-Git matrix:

```text
F < S        PASS
F == S       reject exit 2
S < F        reject exit 2
F unrelated S reject exit 2
missing F    reject exit 2
missing S    reject exit 2
```

Required umbrella:

```text
TestClosureExecuteTopologyAuthority
```

## 2. Fix refs observation

Do not parse the undocumented/default `git for-each-ref` presentation.

Invoke an explicit machine format, NUL-framed, containing exactly:

```text
refname
objectname
```

Parse fail-closed.

Required real-Git test:

```text
create repo
create branch + annotated tag
capture refs
mutate one ref
capture again
Diff -> caller_refs_changed
```

Required failure rows:

```text
git failure
malformed framing
empty malformed record
duplicate ref
invalid OID
```

Observation failure => `Available=false`.

Remove this fail-open condition:

```text
if before refs == "" -> ignore drift
```

Empty valid repository and unavailable observation must be distinguishable.

## 3. Subject-worktree authority

Create detached temporary worktree at S.

Prove:

```text
HEAD == S
HEAD^{tree} == S^{tree}
symbolic-ref exit 1
worktree path outside caller repo
```

Cleanup uses a fresh bounded context:

```text
git worktree remove --force
git worktree prune
filesystem cleanup
```

Git documents linked worktrees as separate working trees associated with one repository; use `worktree list --porcelain -z` for the stable machine-readable inventory.

Required umbrella:

```text
TestClosureSubjectWorktreeAuthority
```

## 4. Execute exact F:P at S

Resolve only:

```text
F:P -> blob OID
cat-file blob OID -> exact bytes
production plan decoder -> plan
```

Run checks only with:

```text
execution_root = S worktree
```

Hermetic fixture:

```text
F: plan introduced
S: subject-only sentinel introduced
D: descendant-only sentinel introduced
caller HEAD = D
```

Check must prove:

```text
S sentinel visible
D sentinel absent
execution tree == S^{tree}
```

Required umbrella:

```text
TestClosureExecuteChecksAgainstSubjectTree
```

## 5. Real check/result bijection

Do NOT infer plan membership from result uniqueness.

Build expected IDs from the decoded frozen plan.

Require:

```text
len(results) == len(plan.checks)
each plan ID appears exactly once
no unknown result ID
no duplicate result ID
result order == plan order
mode == frozen plan mode
```

Reject unknown modes.

Required umbrella:

```text
TestClosureCheckResultBijection
```

## 6. Exclude semantics

For:

```text
mode=exclude
```

the command MUST NOT execute.

Use an excluded command that would create a sentinel file.

Assert:

```text
sentinel absent
result exists exactly once
result.mode=exclude
result.outcome=excluded
```

## 7. Bounded process authority

Reuse one production bounded executor.

Required matrix:

```text
success
nonzero exit
stdout exactly cap
stdout cap+1
stderr exactly cap
stderr cap+1
simultaneous overflow
timeout
cancellation
spawn failure
retained descendant pipe
```

Required umbrellas:

```text
TestClosureBoundedExecutionMatrix
TestClosureWaitDelayRetainedPipe
```

No new subprocess gateway.

## 8. Gate runs at S exactly once

Gate request MUST be:

```text
RepositoryRoot = caller repository
SubjectRoot = detached S worktree
```

Assert the underlying runner sees S worktree as cwd.

Invocation count exactly 1.

Required umbrella:

```text
TestClosureGateRunsAgainstSubject
```

## 9. Caller-state barrier

Capture BEFORE and AFTER:

```text
HEAD
HEAD^{tree}
porcelain-v2
refs
worktree inventory
```

Success requires byte/identity equality.

Required matrix:

```text
HEAD drift
tree drift
status drift
refs drift
worktree leak
BEFORE unavailable
AFTER unavailable
```

All => exit 4.

## 10. Keep evidence incomplete

The current `ClosureEvidenceEx` is experimental and MUST NOT become publication authority in this ACT.

Specifically:

```text
DeriveClosureEvidenceCompleteness (canonical) remains INCOMPLETE
DeriveClosureEvidenceCompletenessEx not wired to CLI/runner
no COMPLETE evidence publication
```

Before CORRECTION02-B, separately fix:

```text
true plan/result bijection
binary source tree/clean/detached predicates
binary outside-all-worktrees predicate
runtime caller identity predicates
```

## Required umbrellas

```text
TestClosureExecuteTopologyAuthority
TestClosureCallerStateAuthority
TestClosureSubjectWorktreeAuthority
TestClosureFrozenPlanRawBytes
TestClosureFrozenPlanTrailingNewline
TestClosureExecuteChecksAgainstSubjectTree
TestClosureCheckResultBijection
TestClosureBoundedExecutionMatrix
TestClosureWaitDelayRetainedPipe
TestClosureGateRunsAgainstSubject
```

Do not satisfy an umbrella by merely calling an old test or renaming it.
Each umbrella must exercise the production path relevant to its claim.

## Verification

```text
go test -race -count=1 ./internal/factory/closure/...
go test -count=1 ./cmd/leamas/...
go vet ./internal/factory/closure/... ./cmd/leamas/...
gofmt -l <ACT-owned files>
git diff --check
```

Run focused Factory verifiers permitted by editor policy.

Do not run:

```text
make factorize
make gate-dupcode
make gate
```

unless explicitly authorized.

## Publication

One implementation commit:

```text
factory: execute closure plans against immutable subjects
```

Optional docs-only report commit.

No closure commit.
No annotated tag.

## Acceptance

```text
TOPOLOGY=F_LT_S
OLD_S_LT_F_RUNNER_NOT_USED=true

REAL_GIT_REFS_CAPTURE=PASS
REFS_OBSERVATION_FAIL_CLOSED=true
REFS_DRIFT=PASS

SUBJECT_WORKTREE_AUTHORITY=PASS
EXECUTION_TREE_EQUALS_S_TREE=true

FROZEN_PLAN_BYTES_FROM_F=true
FROZEN_PLAN_TRAILING_NEWLINE=PASS

CHECKS_EXECUTE_AT_S=true
D_ONLY_CONTENT_INVISIBLE=true
EXCLUDED_CHECKS_NOT_EXECUTED=true
CHECK_RESULT_BIJECTION=PASS

BOUNDED_EXECUTION_MATRIX=PASS
WAIT_DELAY_RETAINED_PIPE=PASS

GATE_EXECUTES_AT_S=true
GATE_INVOCATIONS=1

CALLER_STATE_UNCHANGED=PASS
WORKTREE_LEAK=false

CANONICAL_EVIDENCE_COMPLETENESS=INCOMPLETE
CORRECTION02_B_READY=true

CLINEMM_FILES_CHANGED=none
WORKTREE_STATUS=clean
```

## Stop conditions

Close PARTIAL if any are true:

```text
factory close execute still delegates to S<F topology
refs authority uses default for-each-ref output
empty refs is used as "observation unavailable"
tests use a fake refs grammar not used by production
bijection checks only uniqueness
checks/gate execute in caller checkout
D-only content visible
retained-pipe proof absent
COMPLETE evidence becomes publishable
```

## Successor

After PASS:

```text
ACT-...-CORRECTION02-B
```

CORRECTION02-B owns only:

```text
exact-S binary authority
corrected completeness predicate
coordinated evidence publication
hermetic public dogfood
negative public CLI matrix
```
