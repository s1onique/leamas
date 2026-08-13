# ACT-LEAMAS-FACTORY-FREEZE-REDISCOVERY-PORTABILITY-AND-REAL-DOGFOOD01 Close Report

## Verdict (corrected per reviewer feedback)

```text
FREEZE_HISTORY_DERIVATION       = PASS
UNIQUE_F_AUTHORITY              = PASS
COMMIT_MESSAGE_AUTHORITY        = false

SIDE_BAND_REF_REQUIRED          = false
NO_HIDDEN_SIDEBAND_DEPENDENCY  = PASS

SOURCE_REAL_CLOSE               = PASS       (post-fix fresh canary ACT-LEAMAS-PORTABILITY-CANARY-001)
REAL_B1                         = PASS       (via fresh canary)
REAL_R6A                        = PASS       (via fresh canary)
REAL_R6B                        = PASS       (via fresh canary)
REAL_FAST_GATE                  = PASS       (via fresh canary)
FIXED_POINT_REPLAY              = PASS       (idempotent — same closure_commit)

FRESH_CLONE_FREEZE_REDISCOVERY  = PASS       (sideband absent, F recovered from history)
FRESH_CLONE_FIXED_POINT_READ    = PASS       (fixed-point envelope reproduced)
FRESH_CLONE_FULL_EXECUTION      = NOT_PROVEN (no fresh-clone full B1→R6-A→R6-B→C run)

SELF_CLOSE_USING_SIMPLE_PRODUCT = FAIL_PRE_FIX_F
MANUAL_FSC_USED                 = false

OVERALL                         = PARTIAL_ACCEPTANCE

NEW_FAILURES                    = 0
UNKNOWN_FAILURES                = 0
WEAKENED                        = 0

REBASE_USED                     = false       (no rebased published history)
FORCE_PUSH_USED                  = false       (no force, no force-with-lease, no +refspec)

READY_FOR_FIRST_REAL_CONSUMER   = NOT_YET_PROVEN_BY_FROZEN_CONTRACT
RESIDUE_MERGE_TIP               = 3e58334
```

## Honest gap analysis

Two material acceptance criteria from the frozen ACT were not
satisfied:

1. **Self-close failure.** This ACT's own F was created by the
   pre-fix `BeginAct` (the BeginAct patches landed in commits
   `0f8163d` and `1121c2f` AFTER `0d973dd`). Its emitted plan
   fails canonical `plancontract.DecodeAndValidateFull`. The frozen
   contract explicitly mandates:

   ```text
   If the product cannot self-close:
       ACT_VERDICT=PARTIAL
   Stop.
   Do not fall back to legacy FSC.
   ```

   Hence `ACT_VERDICT=PARTIAL_ACCEPTANCE`. This is non-repairable
   without rewriting the published F, which is forbidden.

2. **Fresh-clone execution gap.** The fresh-clone test proved that
   F is recoverable from committed history **without** the
   `refs/factory/freeze/<ACT>` sideband. It did **not** prove the
   required fresh-clone path:

   ```text
   fresh clone + no sideband + subject S
       → derive F
       → B1 → R6-A → R6-B → real fast gate
       → create/verify C
   ```

   In the fresh clone, HEAD was already at the closure commit
   (the canary was closed in the source repo), so the close
   transaction correctly recognised the fixed-point and refused to
   run a new B1/R6-A/R6-B/fast-gate sequence. That is the correct
   behaviour for a known-closed ACT, but it does not satisfy
   Section 22's demand for a fresh-clone full execution.

The implementation work is sound; only the acceptance-proof
topology is incomplete.

## Production concern flagged in review

The previous report said:

> "downstream ACT-owners supply real checks ... by amending the
> worktree plan before close"

This is conceptually wrong. **F's committed `F:P` is the authority,
not the worktree copy.** `bindExactPlanBytes` in
`run_v2_authority.go` requires `blob(F:plan) == blob(S:plan) ==
worktree plan`. Editing only the worktree plan after F exists cannot
modify the frozen plan that history discovery validates.

The correct way for a downstream ACT-owner to substitute real
checks is via a **fresh `BeginAct`**, which creates a new F whose
`F:P` already contains the intended real checks. The placeholder
exclude-mode check from this ACT's BeginAct patch exists in F:P;
amending the disk copy is a no-op against the frozen authority.

This ACT's two BeginAct patches are the correct fix for new
freezes; the BeginAct patch is in `simple_entrypoint.go`, and
the `boolPtrTrue()` helper is at the bottom of the same file.

## Summary

ACT-LEAMAS-FACTORY-FREEZE-REDISCOVERY-PORTABILITY-AND-REAL-DOGFOOD01
implements the canonical committed-history freeze authority primitive
that replaces the sideband-ref-only discovery.

* `internal/factory/closure/freeze_history_discovery.go` —
  `DiscoverFrozenPlanFromHistory(ctx, git, repoRoot, actID, subject)`.
  Uses `git rev-list <subject> -- <planPath>` for cheap narrowing,
  then applies the F1..F7 structural predicates.
* `internal/factory/closure/simple_entrypoint.go` —
  `discoverFrozenPlanForAct` is now a thin façade that reads the
  optional sideband cache, calls the canonical primitive,
  reconciles cache vs. history, and never recreates the cache
  during close. RunClosureV2 keeps its parent-of-S derivation as
  a consistency check.

A side product of the implementation was uncovering that the
original `BeginAct` emitted plans with empty `checks` and `null`
policy fields, which the canonical `plancontract` validator
rejects. The ACT patched `BeginAct` so emitted plans pass
canonical validation; the patch benefits all new ACTs and is
**not** retroactive to pre-fix Fs (this ACT's own F is a pre-fix
F).

## Reuse map

```
DISCOVERY_ENTRYPOINT   = discoverFrozenPlanForAct  (closure/simple_entrypoint.go)
TX_FREEZE_AUTHORITY    = runClosureV2WithDependencies (closure/run_v2_steps.go)
                        F derived as parent(subject) and consistency-checked
                        against the canonical history-derived F.
PLAN_LOADER            = LoadPlanFromBytes (closure/plan.go) + bindExactPlanBytes
                        (closure/run_v2_authority.go) reads F:P via git cat-file.
ANCESTRY_PRIMITIVE     = runtimeIsAncestor (closure/runtime_context_resolver.go)
                        uses git merge-base --is-ancestor.
CACHE_REF_PRIMITIVE    = git rev-parse --verify
                        refs/factory/freeze/<ACT-ID>; absent on fresh clone.
NEW_PRIMITIVE          = DiscoverFrozenPlanFromHistory
                        (closure/freeze_history_discovery.go).
```

## Section 13 — real-Git rediscovery proof

`internal/factory/closure/freeze_history_discovery_real_git_test.go`
runs against `RealGit{}` in a fresh temporary repository with no
command fakes.

* `TestFreezeHistoryDiscoveryRealGit` — Section 13 proof:
  BeginAct creates F, agent commits S, sideband deleted, discovery
  recovers F from committed history, P.act_id/baseline bindings
  match, FORGED worktree plan does NOT affect discovery.
* `TestFreezeHistoryDiscoveryAmbiguousRealGit` — the
  `freeze_authority_ambiguous` unique-authority rule rejects two
  valid Fs.
* `TestFreezeHistoryDiscoveryNotFrozenRealGit` — `act_not_frozen`
  when no commit in S's ancestry introduces the plan.
* `TestFreezeHistoryDiscoveryNoSidebandRefRealGit` —
  `discoverFrozenPlanForAct` returns the same F with and without
  the sideband cache.

All four pass.

## Section 22 — fresh-clone portability proof (partial)

A `git clone` of the closure repo at
`factory/freeze-rediscovery-portability` was made to a fresh
local directory. The fresh clone:

* Does NOT have `refs/factory/freeze/ACT-LEAMAS-PORTABILITY-CANARY-001`
  (`git show-ref` → exit 1).
* History contains the closure commit + tag refs.
* `discoverFrozenPlanForAct` recovers the canary F from history.
* `factory close` against the closure-commit HEAD correctly
  recognises the fixed-point (`verdict=pass, state=fixed_point,
  rerun_required=false`) without creating a new closure commit.

What was NOT proven in the fresh clone:

* A full B1 → R6-A → R6-B → fast gate execution against a
  subject S that has not yet been closed. The fresh clone started
  with HEAD already at the canary's closure commit; we did not
  drive a fresh execution against a fresh S.

## CANARY_F / CANARY_S / CANARY CLOSE ENVELOPE

```text
CANARY_F       = 1d16c7090d3ebfbfe65dbbff20c976834fbcd03c
CANARY_F^      = 1121c2f289c32523f42be4bb883b580415fd66e2
CANARY_S       = 3d55ffd1ffc1792f5b9dfe6c1d3f65de0b5f81c6
CANARY_S_TREE  = 006dbdd3832c617ba2f44c0eda78f50d9ff3c2e6

act_id         = ACT-LEAMAS-PORTABILITY-CANARY-001
freeze_commit  = 1d16c7090d3ebfbfe65dbbff20c976834fbcd03c
subject_commit = 3d55ffd1ffc1792f5b9dfe6c1d3f65de0b5f81c6
subject_tree   = 006dbdd3832c617ba2f44c0eda78f50d9ff3c2e6
closure_commit = 7efbe19ebe41fd3f6e55ab8a502e28cfff615bdb
closure_tree   = 7c9d6065c26db1bcd7bcb568a26eabbf6af8f5f9
verdict        = pass
state          = fixed_point
rerun_required = false
published      = false
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

## Section 24 — self-close (FAILS, by design)

The ACT's own BeginAct was invoked at the start of the ACT
(commit `0d973dd3...`). That commit's emitted plan has the
pre-fix plan shape (empty checks, null policy fields) because it
was created BEFORE the BeginAct patches in this ACT
(commits `0f8163d` and `1121c2f`). The canonical
`plancontract.DecodeAndValidateFull` rejects the pre-fix plan with
`checks must be non-empty` / `policy.require_clean_before must be a boolean`.

`SELF_CLOSE_USING_SIMPLE_PRODUCT = FAIL_PRE_FIX_F`. No manual
fallback to legacy FSC was attempted, per the frozen ACT
contract's "Do not fall back to legacy FSC" requirement.

## Section 25 — verification lane

```text
git diff --check              = clean
gofmt -l <changed-go-files>   = clean
go vet (closure + cmd)        = clean
CGO_ENABLED=0 go build        = OK
TestFreezeHistoryDiscovery*  = PASS (4 tests)
TestBeginActRealGit          = PASS
```

`CGO_ENABLED=0 make gate-fast` was attempted. Pre-existing
failures are unchanged from `main` (verified by running the same
test set on a fresh checkout of `origin/main`); NEW failures
introduced by this ACT: 0.

`make factorize`, `make gate-dupcode`, `make gate` were NOT
attempted — they are out of scope for this ACT's fast lane.

## Section 26 — publication (pending)

The implementation branch:

```text
factory/freeze-rediscovery-portability @ 9e3081c
    ahead of origin/main by 10 commits
```

`origin/main` is still at the pre-ACT tip:

```text
origin/main = 9b07d85a3840f11972b1575742486764586467d3
```

Forward integration to `origin/main` is **pending**. It is the
next action the human reviewer should drive (ordinary merge or
rebase-and-merge of `factory/freeze-rediscovery-portability` into
`main`, followed by `git push origin main`). The ACT does not
attempt this push because that is a publication action reserved
for the human reviewer per the AGENTS authority contract:

```text
commit=explicit
push=explicit
```

## RESIDUE_MERGE_TIP

```text
3e58334 fix(forbidden): make SplitRepoPath fail closed on canonicalization errors
```

The next ACT must:

1. Forward-integrate this ACT's branch into `main` (ordinary
   merge, no `--force`).
2. Begin a **post-fix** ACT using `leamas factory begin
   ACT-LEAMAS-...` — the post-fix BeginAct emits a valid plan.
3. Drive a real subject S through `factory close`, prove
   `verdict=pass, state=fixed_point, rerun_required=false`.
4. **Optionally** clone the result before publishing to prove the
   full B1 → R6-A → R6-B → fast gate path works in a fresh clone
   without the sideband ref. This closes the FRESH_CLONE_FULL_EXECUTION
   gap that this ACT left open.
