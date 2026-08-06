# ACT-LEAMAS-FACTORY-CLOSURE-PROTOCOL-V2-VERIFIER-MAC-HANDOFF01-CORRECTION02A

## Status

OPEN — BUILD AND SUBPROCESS EVIDENCE

## Base

```text
BASE_COMMIT=c62d110c01ab3d4786a53ef8fea1d11b4c8e80e8
BASE_TREE=5a463325fdbd6532a6b3df1507e7de385c67c524
BASE_IN_ANCESTRY=true
WORKTREE_STATUS=clean
```

## Mission

Close the P0 build-evidence defects of
ACT-LEAMAS-FACTORY-CLOSURE-PROTOCOL-V2-VERIFIER-MAC-HANDOFF01-CORRECTION01
without re-opening the filesystem confinement work. The
scope is intentionally narrow:

1. the dogfood binary is written OUTSIDE the detached
   build worktree, not inside it;
2. the detached build source status, HEAD commit, HEAD
   tree, and detached state are observed (not constants)
   before and after the build;
3. both runner and verifier bounded subprocess results
   are asserted in full;
4. the 646-line correction01 test is split into
   LLM-friendly files;
5. dogfood evidence fields are typed and validated
   before publication.

No filesystem confinement redesign in this ACT.
No tag metadata parsing in this ACT.
No public-CLI mutation matrix in this ACT.

## Acceptance

```text
BUILD_OUTPUT_OUTSIDE_SOURCE=true
BUILD_SOURCE_STATUS_BEFORE=clean
BUILD_SOURCE_STATUS_AFTER=clean
RUNNER_EXIT=0
RUNNER_TIMED_OUT=false
RUNNER_STDOUT_TRUNCATED=false
RUNNER_STDERR_TRUNCATED=false
RUNNER_ERROR_PRESENT=false
VERIFIER_EXIT=0
VERIFIER_TIMED_OUT=false
VERIFIER_STDOUT_TRUNCATED=false
VERIFIER_STDERR_TRUNCATED=false
VERIFIER_ERROR_PRESENT=false
LLM_FRIENDLY_ACT_FILES=PASS
```

## Publication

Exactly one forward commit:

```text
factory: prove clean v2 verifier dogfood build
```

No later close-artifact commit.

## Expected final status

```text
STATUS=PASS
DOGFOOD_BINARY_COMMIT=FINAL_COMMIT
COMMITS_AFTER_FINAL=0
```
