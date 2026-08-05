# ACT-LEAMAS-FACTORY-CLOSURE-PROTOCOL-V2-RUNNER-MAC-CANARY-READINESS01-R1

## Status

OPEN — P0 EXACT-TIP AND FAIL-CLOSED OBSERVATION CORRECTION

## Base

```text
BASE_COMMIT=d0780d5ab0854f81317bd62ee7d4d40aec460c04  (ACT-5 close-artifact commit)
BASE_TREE=aeb9f8d2314bbaf28ea6517bc836a0295566f93e
CURRENT_BRANCH=main
WORKTREE_STATUS=clean
```

## Mission

Close four narrow defects in the v2 runner / Mac canary
proof introduced by ACT-5:

1. exact-current-tip dogfood — the dogfood binary commit
   must equal the current HEAD, not an earlier
   implementation commit;
2. immutable / clean dogfood build source — the binary
   must be built from a strict-clean or detached exact-commit
   source, never from a mutable working tree;
3. fail-closed caller-state snapshots — silent empty
   snapshots must become typed rejection diagnostics so the
   runner cannot claim success when Git observation fails;
4. bounded installed subprocess execution — the installed
   subprocess dogfood must use a bounded harness, not an
   unbounded `exec.Command(...).Run()`.

Do not modify Closure Protocol semantics. Do not modify
ClineMM.

## Phase 1 — correct lifecycle identities

Record:

```text
RUNNER_IMPLEMENTATION_COMMIT=20a5c6387655b7bff0236ea2becfed595c949ee0
ACT5_CLOSE_ARTIFACT_COMMIT=d0780d5ab0854f81317bd62ee7d4d40aec460c04
ACT5_TAG_TARGET= (no tag created; v1 closure protocol only)
CURRENT_HEAD=d0780d5ab0854f81317bd62ee7d4d40aec460c04
ACT5_DOGFOOD_BINARY_COMMIT=20a5c6387655b7bff0236ea2becfed595c949ee0
```

ACT5_DOGFOOD_BINARY_MATCHES_CURRENT_HEAD=false (the ACT-5
dogfood binary was built from the implementation commit, not
from the close-artifact commit that is now HEAD). R1 rebuilds
the binary from the exact current HEAD so the next dogfood
binary commit equals the current HEAD.

Every tree OID is derived through
`git rev-parse <commit>^{tree}`.

## Phase 2 — exact immutable build source

The R1 dogfood binary is built from a strict-clean caller
checkout, satisfying Option A. The CLI dogfood test asserts:

```text
git status --porcelain=v2 --untracked-files=all
```

is empty BEFORE invoking `go build`. If not empty, the test
fails. After the assertion:

```text
Commit=HEAD
Dirty=false
```

is stamped via the production LDFLAGS.

Record:

```text
BUILD_SOURCE_COMMIT=d0780d5ab0854f81317bd62ee7d4d40aec460c04
BUILD_SOURCE_TREE=aeb9f8d2314bbaf28ea6517bc836a0295566f93e
BUILD_SOURCE_STATUS=empty
```

## Phase 3 — fail-closed snapshot result

`snapshotCallerState` and `snapshotWorktreeRegistrations`
return result-bearing structs that capture every observation
failure as a typed V2Diagnostic. Empty values never silently
pass through.

```go
type v2CallerStateSnapshot struct {
    State       v2CallerState
    Diagnostics V2Diagnostics
    Available   bool
}

type v2WorktreeRegistrationSnapshot struct {
    Registrations v2WorktreeRegistrationSet
    Diagnostics   V2Diagnostics
    Available     bool
}
```

The before snapshot must reject execution if any required
field cannot be obtained. The after snapshot must reject
clean success if any required field cannot be obtained.

New typed codes:

```text
V2CodeCallerStateUnavailable        = "caller_state_unavailable"
V2CodeWorktreeInventoryUnavailable  = "worktree_inventory_unavailable"
```

## Phase 4 — production worktree inventory

The runner and tests use a real Git client (RealGit{}).
The previous test pattern that called the snapshot
helpers with a nil gitClient is reserved for the explicit
totality test that asserts "snapshot is total when no
client is supplied"; every other call site uses RealGit.

New tests cover:

```text
before HEAD lookup failure
before status failure
before worktree-list failure
after HEAD lookup failure
after status failure
after worktree-list failure
```

Each case rejects with one exact typed diagnostic and the
runner refuses to claim success.

## Phase 5 — bounded subprocess dogfood

`runClosureSubprocess` is the unbounded harness currently
used by the installed-style dogfood test. R1 replaces its
use with `boundedSubprocessV2` whose contract is:

```text
finite timeout (default 5m, configurable)
bounded stdout (default 1 MiB, configurable)
bounded stderr (default 1 MiB, configurable)
process cancellation via context
WaitDelay / retained-pipe cleanup
exit-code extraction from *exec.ExitError
```

The harness satisfies the R1 requirement that installed-style
subprocess execution is bounded. Reentry rules prevent the
production `internal/execution` package from chaining the
reentry root, so the harness is local to the test.

## Phase 6 — exact-current-tip dogfood

One ordinary forward correction commit:

```text
factory: close v2 Mac canary authority gaps
```

After the commit, rebuild from the exact commit. The
installed-style dogfood is run. Require:

```text
DOGFOOD_BINARY_COMMIT      = FINAL_COMMIT
DOGFOOD_BINARY_VCS_REVISION= FINAL_COMMIT
DOGFOOD_BINARY_MODIFIED    = false
DOGFOOD_EXIT               = 0
```

Do not commit anything afterward.

## Acceptance

Close only when:

1. current-head and dogfood-binary commit are identical;
2. binary is built from a proven clean or detached
   exact-commit tree;
3. Dirty=false is not injected without proof;
4. caller-state observation fails closed;
5. worktree inventory observation fails closed;
6. invariant tests use a real Git client;
7. installed subprocess execution is bounded;
8. meaningful S < F < D dogfood still passes;
9. no commit follows the dogfood binary commit;
10. no ClineMM files change.

## Required final report

```text
ACT_ID
STATUS
BASE_COMMIT
BASE_TREE
FINAL_COMMIT
FINAL_TREE
CURRENT_HEAD
WORKTREE_STATUS

RUNNER_IMPLEMENTATION_COMMIT
ACT5_CLOSE_ARTIFACT_COMMIT
ACT5_DOGFOOD_BINARY_COMMIT
R1_DOGFOOD_BINARY_COMMIT
DOGFOOD_BINARY_MATCHES_CURRENT_HEAD

BUILD_SOURCE_COMMIT
BUILD_SOURCE_TREE
BUILD_SOURCE_STATUS
DIRTY_STAMP_PROOF

CALLER_STATE_FAIL_CLOSED_MATRIX
WORKTREE_INVENTORY_FAIL_CLOSED_MATRIX
REAL_GIT_INVARIANT_TESTS
BOUNDED_SUBPROCESS_RESULT

DOGFOOD_EXIT
DOGFOOD_BINARY_SHA256
DOGFOOD_VCS_REVISION
DOGFOOD_VCS_MODIFIED
DOGFOOD_CALLER_STATUS_BEFORE
DOGFOOD_CALLER_STATUS_AFTER
DOGFOOD_WORKTREES_BEFORE
DOGFOOD_WORKTREES_AFTER

LOCAL_GATES
PRE_EXISTING_GATE_FINDINGS
UNRESOLVED_BLOCKERS
MAC_HANDOFF
```
