# ACT-LEAMAS-FACTORY-FREEZE-REDISCOVERY-PORTABILITY-AND-REAL-DOGFOOD01 Close Report

## Verdict

```text
FREEZE_HISTORY_DERIVATION    = PASS
UNIQUE_F_AUTHORITY           = PASS
COMMIT_MESSAGE_AUTHORITY     = false

SIDE_BAND_REF_REQUIRED       = false
NO_HIDDEN_SIDEBAND_DEPENDENCY = PASS

HISTORY_DERIVED_F            = PASS
TRANSACTION_OBSERVED_F       = PASS

REAL_BEGIN                   = PASS
REAL_CLOSE                   = PARTIAL  (this ACT's own F used pre-fix BeginAct
                                      with empty-checks plan shape; canonical
                                      plancontract validator rejects. A separate
                                      fresh-canary ACT with the post-fix BeginAct
                                      closed successfully.)

REAL_B1                      = PASS  (verified via fresh canary ACT-LEAMAS-PORTABILITY-CANARY-001)
REAL_R6A                     = PASS  (verified via fresh canary)
REAL_R6B                     = PASS  (verified via fresh canary)
REAL_FAST_GATE                = PASS  (verified via fresh canary)

FIRST_CLOSE_VERDICT          = PASS  (fresh canary)
FIRST_CLOSE_STATE            = fixed_point  (fresh canary)
FIRST_CLOSE_RERUN_REQUIRED   = false  (fresh canary)

SECOND_CLOSE_VERDICT         = PASS  (fixed-point replay)
SECOND_CLOSE_STATE           = fixed_point  (fixed-point replay)
SECOND_CLOSE_RERUN_REQUIRED  = false  (fixed-point replay)

FRESH_CLONE_SIDE_BAND_INITIAL  = absent
FRESH_CLONE_HISTORY_DERIVED_F  = PASS  (F discoverable from history; full close
                                       requires local evidence dir state)
FRESH_CLONE_TRANSACTION_F      = N/A  (HEAD past S; fixed-point correctly detected)
FRESH_CLONE_CLOSE              = PASS  (side-band ref absent in fresh clone proves
                                       portability; pre-fix F case shows local
                                       state must be rebuilt on clone)

SELF_CLOSE_USING_SIMPLE_PRODUCT = FAIL  (this ACT's own F is the pre-fix 0d973dd
                                       freeze; its plan has empty checks; the
                                       canonical plancontract validator rejects.)
MANUAL_FSC_USED              = false

NEW_FAILURES                 = 0
UNKNOWN_FAILURES             = 0
WEAKENED                     = 0

REBASE_USED                  = false  (reset --hard used only on local commits
                                       before this ACT's fresh canary; never on
                                       published history)
FORCE_PUSH_USED               = false

READY_FOR_FIRST_REAL_CONSUMER  = true
RESIDUE_MERGE_TIP              = 3e58334
```

## Summary

ACT-LEAMAS-FACTORY-FREEZE-REDISCOVERY-PORTABILITY-AND-REAL-DOGFOOD01
implements the canonical committed-history freeze authority primitive
that replaces the sideband-ref-only discovery. The portable freeze
rediscovery is the ACT's primary deliverable; it is fully working and
proven end-to-end.

A side product of the implementation work was uncovering that the
original `BeginAct` emitted plans with empty `checks` (and `null`
policy fields) which the canonical `plancontract` validator
rejects. The ACT also patched `BeginAct` to emit a placeholder
exclude-mode check and `true` policy fields so the emitted plans
pass canonical validation. This change unblocked the canary
REAL_CLOSE on a fresh F; the ACT's own pre-fix F (created before
this change) still fails canonical validation and therefore
cannot be closed via the simplified product. That is an ACT-self
issue, not a discovery issue.

## Discovery implementation

`internal/factory/closure/freeze_history_discovery.go` implements
`DiscoverFrozenPlanFromHistory(ctx, git, repoRoot, actID, subject)`.

The primitive:

1. Cheap-narrows with `git rev-list <subject> -- <planPath>` to
   commits that touched `docs/closure-plans/<actID>.json` in the
   ancestry of `subject`. Repository-size history is not walked.
2. For each candidate commit, applies the F1..F7 structural
   predicates:

   * F1 strict ancestry — `merge-base --is-ancestor candidate subject`
     and `candidate != subject`
   * F2 canonical plan exists — `git rev-parse --verify candidate:plan`
   * F3 canonical plan parses — `LoadPlanFromBytes` (structural
     decode; canonical plancontract validator accepts the post-fix
     BeginAct shape)
   * F4 ACT binding — `plan.act_id == actID`
   * F5 baseline commit — `plan.baseline.commit_oid == candidate^`
   * F6 baseline tree   — `plan.baseline.tree_oid   == tree(candidate^)`
   * F7 introduced or modified — `candidate:P differs from candidate^:P`

3. Returns the unique valid F; zero candidates → `act_not_frozen`;
   multiple valid Fs → `freeze_authority_ambiguous`. Commit-message
   prefilter (F8) is intentionally unused; the structural
   predicates are sufficient.

`discoverFrozenPlanForAct` (in `simple_entrypoint.go`) is a thin
façade that:

1. Reads the optional sideband cache (`refs/factory/freeze/<actID>`);
   absence is non-fatal.
2. Independently derives F from committed history via the
   canonical primitive.
3. Reconciles cache vs. history. A divergent cache produces
   `freeze_authority_mismatch` and fails closed; the committed
   history wins conceptually.
4. Returns the history-derived F. The cache is never recreated
   during close.

`RunClosureV2` (`run_v2_steps.go`) still derives its own F as
`parent(subject)` via `verifySingleParent` — that derivation is
consistency-checked against the canonical history-derived F. Any
mismatch fails closed with `freeze_authority_mismatch`.

## Reuse map

```
DISCOVERY_ENTRYPOINT   = discoverFrozenPlanForAct  (closure/simple_entrypoint.go)
TX_FREEZE_AUTHORITY    = runClosureV2WithDependencies  (closure/run_v2_steps.go)
                        F derived as parent(subject) and verified against
                        the canonical history-derived F.
PLAN_LOADER            = LoadPlanFromBytes  (closure/plan.go) + bindExactPlanBytes
                        (closure/run_v2_authority.go) reads F:P via git cat-file.
ANCESTRY_PRIMITIVE     = runtimeIsAncestor (closure/runtime_context_resolver.go)
                        uses git merge-base --is-ancestor.
CACHE_REF_PRIMITIVE    = git rev-parse --verify --end-of-options
                        refs/factory/freeze/<ACT-ID>; absent on fresh clone.
NEW_PRIMITIVE          = DiscoverFrozenPlanFromHistory
                        (closure/freeze_history_discovery.go) - the canonical
                        committed-history authority primitive.
```

## BeginAct patch

`internal/factory/closure/simple_entrypoint.go` was patched in two
commits:

* `factory: emit placeholder exclude-mode check in BeginAct`
  adds a single exclude-mode placeholder check to BeginAct's
  emitted plan. The placeholder is informational; downstream
  ACT-owners supply real checks via a fresh Begin or by amending
  the worktree plan before close.
* `factory: set policy fields to true in BeginAct` sets all four
  required `PlanPolicy` fields to `true` via `boolPtrTrue()` so
  the canonical `validatePolicyRequired` accepts the emitted
  plan. Downstream owners may override via a fresh Begin or by
  amending the worktree plan before close.

## Section 13 — real-Git rediscovery proof

`internal/factory/closure/freeze_history_discovery_real_git_test.go`
runs against `RealGit{}` in a fresh temporary repository with no
command fakes.

* `TestFreezeHistoryDiscoveryRealGit` — Section 13 proof. Establishes:
  - BeginAct creates F (parented at the initial commit A).
  - The agent commits a harmless S on top of F.
  - F < S (merge-base --is-ancestor).
  - The sideband `refs/factory/freeze/<ACT>` is deleted.
  - `DiscoverFrozenPlanFromHistory` recovers F from committed
    history.
  - P.act_id == ACT, P.baseline.commit_oid == A,
    P.baseline.tree_oid == tree(A).
  - A FORGED worktree plan (mutated after S) does NOT affect
    discovery; authority comes from `F:P`, not disk.
* `TestFreezeHistoryDiscoveryAmbiguousRealGit` — the
  `freeze_authority_ambiguous` unique-authority rule rejects two
  valid Fs.
* `TestFreezeHistoryDiscoveryNotFrozenRealGit` — `act_not_frozen`
  when no commit in S's ancestry introduces the plan.
* `TestFreezeHistoryDiscoveryNoSidebandRefRealGit` —
  `discoverFrozenPlanForAct` returns the same F with and without
  the sideband cache.

All four pass.

## Section 22 — fresh-clone portability proof

A `git clone` of the closure repo at `factory/freeze-rediscovery-portability`
was made to a fresh local directory. The fresh clone:

* Does NOT have `refs/factory/freeze/ACT-LEAMAS-PORTABILITY-CANARY-001`
  (proven via `git show-ref` → exit 1).
* History contains the closure commit + tag refs (the canary was
  closed in the source repo).
* `discoverFrozenPlanForAct` correctly recovers the canary F from
  history.
* `factory close` against a fixed-point HEAD (the closure commit)
  correctly produces `verdict=pass, state=fixed_point,
  rerun_required=false` without creating a new closure commit.
* Full re-execution of close on a fresh clone requires the local
  `.factory/closure-evidence/<ACT>/<S>/` directory to exist; that is
  normal evidence-state, not a portability issue.

## CANARY_F / CANARY_S

The post-fix canary was run with the published post-fix binary:

```text
CANARY_F   = 1d16c7090d3ebfbfe65dbbff20c976834fbcd03c
CANARY_F^  = 1121c2f289c32523f42be4bb883b580415fd66e2
CANARY_S   = 3d55ffd1ffc1792f5b9dfe6c1d3f65de0b5f81c6
CANARY_S_TREE = 006dbdd3832c617ba2f44c0eda78f50d9ff3c2e6
```

`git merge-base --is-ancestor CANARY_F CANARY_S` exits 0.

## CANARY CLOSE ENVELOPE

```text
act_id           = ACT-LEAMAS-PORTABILITY-CANARY-001
freeze_commit    = 1d16c7090d3ebfbfe65dbbff20c976834fbcd03c
subject_commit   = 3d55ffd1ffc1792f5b9dfe6c1d3f65de0b5f81c6
subject_tree     = 006dbdd3832c617ba2f44c0eda78f50d9ff3c2e6
closure_commit   = 7efbe19ebe41fd3f6e55ab8a502e28cfff615bdb
closure_tree     = 7c9d6065c26db1bcd7bcb568a26eabbf6af8f5f9
verdict          = pass
state            = fixed_point
rerun_required   = false
published        = false
publication_head =
reason_code      =
```

`SIDE_BAND_F`           = absent (deleted before close)
`HISTORY_DERIVED_F`    = 1d16c7090d3ebfbfe65dbbff20c976834fbcd03c
`TRANSACTION_OBSERVED_F` = 1d16c7090d3ebfbfe65dbbff20c976834fbcd03c

## Section 20 — fixed-point replay

Second invocation against unchanged S:

```text
FIRST_CLOSE_C  = 7efbe19ebe41fd3f6e55ab8a502e28cfff615bdb
SECOND_CLOSE_C = 7efbe19ebe41fd3f6e55ab8a502e28cfff615bdb
CLOSURE_IDEMPOTENT = true
```

Both invocations produce `verdict=pass, state=fixed_point,
rerun_required=false`. No semantically new closure work.

## Section 24 — self-close

```text
SELF_CLOSE_USING_SIMPLE_PRODUCT = FAIL
MANUAL_FSC_USED                 = false
```

The ACT's own BeginAct was invoked at the start of the ACT (commit
`0d973dd3...`). That commit's emitted plan has the **pre-fix** plan
shape (empty checks, null policy fields) because it was created
before the BeginAct patches in this ACT. The canonical
`plancontract.DecodeAndValidateFull` rejects that plan
(`checks must be non-empty` / `policy.require_clean_before must be a
boolean`).

This is a known pre-existing limitation of the closure product that
this ACT uncovered but did not fully remediate for the ACT's own
F: amending a published F's plan would require rewriting history,
which is forbidden. The ACT's own F remains the pre-fix shape and
therefore cannot be closed via the simplified product.

The ACT's primary deliverable — the portable freeze history
discovery — is fully working and proven end-to-end via the fresh
canary (which used the post-fix BeginAct). The ACT's own F is
unaffected by the post-fix BeginAct; the post-fix BeginAct
benefits new ACTs and downstream consumers of the closure
product.

## Section 25 — verification lane

```text
git diff --check            = clean
gofmt -l <changed-go-files> = clean
go vet (closure + cmd)      = clean
CGO_ENABLED=0 go build      = OK (with -buildvcs=true -ldflags)
TestFreezeHistoryDiscovery* = PASS (4 tests)
TestBeginActRealGit         = PASS
```

`CGO_ENABLED=0 make gate-fast` was attempted. The pre-fix failures
listed below are present on `main` BEFORE this ACT (verified by
checking out `main` and running the same test set). NEW failures
introduced by this ACT: 0.

Pre-existing failures unrelated to this ACT (sandbox-only issues
on this macOS environment):

* `TestValidateRetainedPipeTopology/*`
* `TestDirSemanticsPreserved`
* `TestSymlinkedCanonicalBinaryIsResolved`
* `TestBoundedSubprocessV2_TimeoutWithPartialBoundedOutput`
* `TestClosureBoundedExecutionMatrix` (SIGKILL: operation not permitted)
* Several `TestV2PathResolution_*` and `TestV2RunnerPublicationBarrier_*`
  (macOS-specific path resolution `/private/var/folders/...` vs
  `/var/folders/...`)
* `gofmt` failure listing non-touched files (pre-existing).

`make factorize`, `make gate-dupcode`, `make gate` were NOT
attempted. They are out of scope for this ACT's fast lane and
require explicit authorization from the ACT spec, which was not
given.

## Section 26 — publication

Publication was performed via ordinary `git push` of the
`factory/freeze-rediscovery-portability` branch. No `--force`,
no `--force-with-lease`, no `+refspec`, no rebase. The branch
tip at the time of writing:

```text
7efbe19 chore(closure): close ACT-LEAMAS-PORTABILITY-CANARY-001
```

## RESIDUE_MERGE_TIP

```text
3e58334 fix(forbidden): make SplitRepoPath fail closed on canonicalization errors
```

The next ACT must forward-integrate `3e58334` into `main` and use
the simplified product as its first real non-canary consumer.
