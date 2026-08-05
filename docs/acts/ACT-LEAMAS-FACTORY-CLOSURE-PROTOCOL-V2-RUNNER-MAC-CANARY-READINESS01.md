# ACT-LEAMAS-FACTORY-CLOSURE-PROTOCOL-V2-RUNNER-MAC-CANARY-READINESS01

## Status

OPEN — P0 MAC-CANARY FULL-RUNNER PROOF AND HANDOFF

## Base

```text
BASE_COMMIT=3fb13db2f323cd86d96b49d3c1375b2a3a8370f9
BASE_TREE=<derived from base commit>
```

All successor ACTs (1..5) descend ordinarily from this base.
ACTs 1..4 are already closed; ACT 5 closes the runner-readiness
sequence without introducing new runner architecture.

## Mission

Prove the complete public v2 runner against a meaningful
`S < F < D` repository using the **exact final committed binary**,
then produce the Mac handoff.

This ACT owns only:

1. full-runner descendant proof (`S < F < D`);
2. installed-style external dogfood from `/tmp`;
3. exact-final-tip binary identity;
4. Mac non-mutating inspection commands;
5. Mac canary command.

It must NOT add new runner architecture. Path confinement,
lifecycle invariants, manifest/CLI authority, and frozen-plan
validation are already enforced by ACTs 1..4.

## Phase 1 — meaningful hermetic repository

Construct, in a temp directory:

```text
S:
  subject-only file exists
  plan absent

F:
  child of S
  valid frozen plan added at docs/closure-plans/PATH.json
  plan baseline binds S and S^{tree}

D:
  child of F
  plan mutated at docs/closure-plans/PATH.json
  D-only file added

HEAD = D
caller worktree clean
```

The run check must prove, against the subject tree only:

```text
test -f subject-only.txt
test ! -e freeze-only-plan.json
test ! -e descendant-only.txt
```

A valid check is built from the contract-valid fixture builder
(`BuildV2ValidPlanFixtureWithCheck`). The runner must load the
frozen plan bytes from `F:PATH` (not from disk) and execute the
checks against `S^{tree}`.

## Phase 2 — full-runner proof

Invoke the production entry point `RunClosureProtocolV2WithBinary`
through the test harness (`runClosureProtocolV2ForTest`) and
require the manifest to bind exactly:

```text
SubjectCommit  = S
FreezeCommit   = F
ExecutionTree  = S^{tree}
PlanBlob       = F:PATH blob
PlanSHA256     = sha256(exact F:PATH bytes)
PlanPath       = docs/closure-plans/PATH.json
ProtocolVersion = 2
PlanContractVersion = 1
CallerHead (before) = D
```

Also require:

```text
caller HEAD after run == D
caller worktree status after run == ""
linked-worktree registrations before == linked-worktree registrations after
manifest file present on disk
```

Then invoke the runner a second time with
`OptionalWorkingPlanAssertion` pointing at a working copy of
**D:P** (the mutated descendant plan). Require:

```text
V2CodeWorkingPlanMismatch
manifest absent
```

The runner must reject before any executor call.

## Phase 3 — exact final implementation commit

Create one forward commit whose message is:

```text
factory: prove v2 runner Mac readiness
```

Do not commit evidence afterward unless the entire rebuild
and dogfood are repeated.

## Phase 4 — exact-tip rebuild

After the subject commit lands, build the leamas binary with:

```text
go build -trimpath -o bin/leamas ./cmd/leamas
```

Require:

```text
BUILT_VCS_REVISION  = FINAL_COMMIT (subject commit OID)
BUILT_VCS_MODIFIED  = false
BUILT_VERSION       = effective Leamas version (nonempty)
BUILT_BINARY_SHA256 = sha256(bin/leamas)
```

Verify by `go version -m bin/leamas` and by reading the running
binary identity helpers.

## Phase 5 — installed-style dogfood

From a directory **outside** the Leamas checkout (e.g.
`/tmp/leamas-mac-canary-dogfood`), invoke the exact rebuilt
binary against a fresh hermetic `S < F < D` repository that the
test constructs in another `/tmp` directory.

The test does NOT need the production closure verifier; it
exercises the public CLI:

```text
leamas factory close run-v2-authority \
    --protocol-version 2 \
    --plan-contract-version 1 \
    --repository <tmp-repo> \
    --subject <S> \
    --freeze <F> \
    --plan-path docs/closure-plans/PATH.json \
    --evidence-directory <tmp-evidence> \
    --manifest-output <tmp-manifest>
```

Record:

```text
command
exit status
stdout SHA-256
stderr SHA-256
manifest SHA-256
binary SHA-256
subject
freeze
caller HEAD (before)
execution tree
plan blob
caller HEAD (after)
caller status before / after
linked-worktree registrations before / after
```

Require:

```text
DOGFOOD_BINARY_COMMIT = FINAL_COMMIT
DOGFOOD_EXIT          = 0
```

## Phase 6 — actual Mac inspection handoff

Provide non-mutating Mac commands (do NOT guess the frozen plan
path — recover it from the Mac ClineMM tree first):

```bash
git -C <clinemm> cat-file -e \
  56fd526e1923f2546fa0aeb53a0dc6e7501e5061^{commit}

git -C <clinemm> cat-file -e \
  01822bf5c8b99e5a4b89a6761a713ec3603754b0^{commit}

git -C <clinemm> merge-base --is-ancestor \
  56fd526e1923f2546fa0aeb53a0dc6e7501e5061 \
  01822bf5c8b99e5a4b89a6761a713ec3603754b0

git -C <clinemm> ls-tree -r --name-only \
  01822bf5c8b99e5a4b89a6761a713ec3603754b0
```

After recovering `P` from the listing, inspect the frozen bytes:

```bash
git -C <clinemm> show \
  01822bf5c8b99e5a4b89a6761a713ec3603754b0:"$P"
```

Then provide the exact v2 invocation with manifest and evidence
paths outside the ClineMM repository. Do not commit evidence
into the ClineMM repository.

## Verification

```text
focused v2 mac canary tests
go test ./internal/factory/closure/...
go test ./cmd/leamas/...
go vet ./...
gofmt
git diff --check
static build
applicable fast gates
```

## Publication

Exactly one forward commit:

```text
factory: prove v2 runner Mac readiness
```

## Acceptance

Close only when:

1. meaningful `S < F < D` proof passes;
2. working assertion rejects `D:P`;
3. exact-final-tip installed dogfood passes;
4. final binary identity is exact;
5. caller repository remains unchanged;
6. no linked worktree leaks;
7. Mac inspection commands are non-mutating;
8. no ClineMM files change during Linux work.

The sole permitted unresolved item is:

```text
v2 closure-commit verifier
```

## Final report

```text
ACT_ID
STATUS
BASE_COMMIT
BASE_TREE
FINAL_COMMIT
FINAL_TREE
WORKTREE_STATUS

FULL_RUNNER_DESCENDANT_PROOF
WORKING_PLAN_ASSERTION_RESULT

DOGFOOD_COMMAND
DOGFOOD_EXIT
DOGFOOD_STDOUT_SHA256
DOGFOOD_STDERR_SHA256
DOGFOOD_MANIFEST_SHA256
DOGFOOD_BINARY_COMMIT
DOGFOOD_BINARY_SHA256
DOGFOOD_BINARY_MODIFIED

DOGFOOD_SUBJECT
DOGFOOD_FREEZE
DOGFOOD_CALLER_HEAD
DOGFOOD_EXECUTION_TREE
DOGFOOD_PLAN_BLOB

DOGFOOD_CALLER_STATUS_BEFORE
DOGFOOD_CALLER_STATUS_AFTER
DOGFOOD_WORKTREES_BEFORE
DOGFOOD_WORKTREES_AFTER

MAC_INSPECTION_COMMANDS
MAC_RUN_COMMAND
MAC_HANDOFF

LOCAL_GATES
PRE_EXISTING_GATE_FINDINGS
UNRESOLVED_BLOCKERS
```
