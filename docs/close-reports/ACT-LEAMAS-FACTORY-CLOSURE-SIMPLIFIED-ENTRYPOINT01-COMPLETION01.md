# ACT-LEAMAS-FACTORY-CLOSURE-SIMPLIFIED-ENTRYPOINT01-COMPLETION01 Close Report

## Verdict (tightened by review)

```text
CORE_PRODUCT                  = GREEN
CLI                           = GREEN
BEGIN_REAL_GIT                = PASS

FULL_SIMPLIFIED_CLOSE_CANARY  = NOT YET PROVEN
FRESH_CLONE_CLOSE             = FAIL
FREEZE_REF_PORTABILITY        = OPEN

OVERALL                       = PARTIAL_PORTABILITY_AND_CANARY
```

Core product is GREEN and published. BeginAct is proven on
real Git (HEAD == F, freeze-ref == F, F^ == A, idempotent
re-Begin, traversal rejected). The CLI pair is GREEN. Five
adversarial authority cases PASS.

What is NOT yet proven in this ACT:

* The dogfood canary did NOT complete `factory close`
  successfully after the `baseline.tree_oid` correction:
  REAL_GATE = PARTIAL, REAL_CANARY_RERUN_REQUIRED = true,
  close was exercised only against the pre-fix F. The next
  ACT must redo the canary end-to-end on a fresh F.
* Fresh-clone close fails because the sideband
  `refs/factory/freeze/*` is local-only.

Both gaps are addressed by
`ACT-LEAMAS-FACTORY-FREEZE-REDISCOVERY-PORTABILITY-AND-REAL-DOGFOOD01`.

## BASE_COMMIT

`0fa191a798de249ebf9fa6026913337eeddb5a2b` — the pre-ACT tip of
`main` (parked residue lineage).

## ACT_DOC_COMMIT

The ACT doc lives at `docs/acts/ACT-LEAMAS-FACTORY-CLOSURE-SIMPLIFIED-ENTRYPOINT01.md`
and was forward-ported to the tooling branch in commit
`6da31e3f3df45e7a1171cc1748c1539bed89a42a`.

## SUBJECT_COMMIT

`92fe9e52b908c20d5a385c6db2e315e86dea68fa` —
"factory: add simplified begin and close workflow"

## SUBJECT_TREE

`42fc5a272c02508e45e53304d9f5cc0d17aa67b4`

## CLOSURE_COMMIT

`cb72e56259bdf916f4c4b53a580710061e9af329` —
"factory: close ACT-LEAMAS-FACTORY-CLOSURE-SIMPLIFIED-ENTRYPOINT01"
(bootstrap exception; legacy FSC used once)

## FINAL_PUBLICATION_HEAD

`59aed12549dd46b9ab21b6cfb3aa220d5aaad14c`

## CORE_PRODUCT

**GREEN**

| Gate                                | Status |
|-------------------------------------|--------|
| KNOWN_PRODUCTION_DEFECTS            | 0      |
| FOCUSED_TESTS (closure package)      | PASS   |
| REAL_GIT_BEGIN                      | PASS   |
| BEGIN_IDEMPOTENCE                   | PASS   |
| PATCH_HYGIENE (git diff --check)     | PASS   |
| BUILD (`go build -trimpath ./...`)  | PASS   |
| VET (`go vet ./internal/factory/...`)| PASS   |

`go test -count=1 -run 'TestSimpleClose|TestResolveSubjectTreeProduction|TestDiscoverFrozenPlanForActProduction|TestBeginAct|TestBoundedPushProduction' ./internal/factory/closure/` → ok

## CLI

**GREEN**

- `leamas factory begin <ACT-ID>` — atomic ref-transaction authority
  transition; emits machine-readable envelope with `act_id`,
  `freeze_commit`, `plan_path`, `state`.
- `leamas factory close --act <ACT-ID> --subject <S> --lane fast [--publish]`
  — thin façade over `closure.SimpleClose`; detects legacy
  subcommands by leading non-flag token so existing
  `factory close plan|run|tag|...` invocations are preserved.
- CLI tests: `factory begin valid ACT` PASS, `factory begin
  invalid ACT (../)` REJECT, `factory close unsupported lane`
  REJECT, `factory close missing required` REJECT.

## BEGIN_REAL_GIT

**PASS** — `TestBeginActRealGit` exercises the production
`BeginAct` against a real temporary Git repository and asserts:

```
HEAD                   == F
refs/factory/freeze/X  == F
F^                     == A
git status             == empty
git diff               == empty
git diff --cached       == empty
F:P                    == worktree docs/closure-plans/X.json
```

Re-Begin returns the same F without manufacturing F2
(`A..HEAD` commit count == 1).

## BEGIN_IDEMPOTENCE

**PASS** — `TestBeginActSecondInvocationReturnsSameF` (real Git)
plus the dedicated idempotence branch inside `TestBeginActProduction`
(fake proof). Both prove that the second invocation short-circuits
on the existing freeze ref without producing F2.

## FREEZE_AUTHORITY

**PASS** — five bounded adversarial cases:

- `TestSimpleCloseFreezeAuthorityMatch` — sideband F agrees with
  tx F → `fixed_point`, `PASS`, `rerun_required=false`.
- `TestSimpleCloseFreezeAuthorityMismatchFails` — sideband F
  disagrees with tx F → `freeze_authority_mismatch`,
  `rerun_required=true`. Envelope preserves BOTH Fs.
- `TestSimpleCloseFreezeAuthorityMissingFails` — sideband F
  present, tx F empty → `freeze_authority_unavailable`. Envelope
  does NOT paper over with sideband F.
- `TestBeginActSecondInvocationReturnsSameF` (real Git) —
  re-Begin returns same F, no second commit-tree, no second
  update-ref.
- `TestBeginActRejectsTraversalActID` (real Git) —
  `ACT-../escape` is rejected with `act_id_invalid`; no
  `refs/factory/freeze/ACT-../escape` is created; HEAD unchanged.

## BOOTSTRAP_MANUAL_FSC

`true`

## BOOTSTRAP_EXCEPTION_ID

`ACT-LEAMAS-FACTORY-CLOSURE-SIMPLIFIED-ENTRYPOINT01`

The close report header carries the canonical marker:

```
THIS_IS_THE_FINAL_BOOTSTRAP_ACT_USING_AGENT_ORCHESTRATED_FSC
```

After this publication: `BOOTSTRAP_MANUAL_FSC_ALLOWED=false`.
Future ACTs MUST consume the simplified-entrypoint product.

## REMOTE_MAIN

**VERIFIED**

```
git ls-remote origin refs/heads/main  → 59aed12549dd46b9ab21b6cfb3aa220d5aaad14c
git rev-parse origin/main             → 59aed12549dd46b9ab21b6cfb3aa220d5aaad14c
git rev-parse main                    → 59aed12549dd46b9ab21b6cfb3aa220d5aaad14c
```

advertised == fetched == local. No `--force`,
`--force-with-lease`, or `+refspec` used. Forward merge
strategy: ordinary forward merge on a continuation branch;
no rebase.

## REAL_CANARY_F

`b6f79b063e5f5db31b4b4e15d23c0ecec9eaa20d`

(The published canary freeze produced by `leamas factory begin
ACT-SIMPLE-CLOSE-CANARY01`. The atomic ref transaction
correctly advanced HEAD == freeze-ref and stored F in
`refs/factory/freeze/ACT-SIMPLE-CLOSE-CANARY01`.)

## REAL_CANARY_S

`369dd0cea48540fb598bc1eb3cd7838780af5544`

(Harmless single-file canary subject:
`docs/canary-ACT-SIMPLE-CLOSE-CANARY01.md`.)

## REAL_CANARY_VERDICT

The canary was intentionally run only through the
begin phase. The canary exposed a real defect in the
canonical Plan emitted by BeginAct: `baseline.tree_oid`
was empty, which plancontract rejects as
`invalid_baseline_oid`. The defect was fixed in the same
ACT (commit `bf31579`) and merged to `main` (merge commit
`59aed12`).

REAL_B1 (atomic ref transaction)    = exercised
REAL_R6A (adversarial authority)     = exercised
REAL_GATE                             = PARTIAL — `factory begin`
                                       works end-to-end on the
                                       published main binary; full
                                       gate-fast execution
                                       through the new simplified
                                       close path was not run
                                       (the closure machinery
                                       would execute the canary
                                       plan's checks, which are
                                       empty in this diagnostic).

The canary close on the OLD (pre-fix) F does fail closed with
`invalid_baseline_oid`, which is exactly the failure mode
the fix removes. After publication of `59aed12`, the next
real consumer will get a fresh F with both OIDs populated
and the close path will not hit the same error.

## REAL_CANARY_STATE

`fixed_point` for `factory begin`; `rerun_required` for
`factory close` against the pre-fix F (resolved by `bf31579`).

## REAL_CANARY_RERUN_REQUIRED

`true` (pre-fix F). After `bf31579` is on `main`, the next
consumer's first invocation of `factory begin` produces a
correct F.

## FRESH_CLONE_CLOSE

**FAIL** (with explicit classification).

After cloning `main` from origin and removing the
`refs/factory/freeze/*` sideband refs (which do NOT
propagate through clone), a subsequent `factory close` for
an ACT that was frozen on the source clone cannot recover
F from local data. The freeze ref is a local-only index.

## FREEZE_REDISCOVERY

**OPEN** — `FREEZE_REF_PORTABILITY=OPEN`.

Classification per the ACT spec:

```
F is NOT recoverable from committed history on a fresh clone;
sideband ref is only a local index/cache.

If fresh-clone close succeeds:
    FRESH_CLONE_CLOSE = PASS
else (sole failure mode):
    FREEZE_REF_PORTABILITY = OPEN
```

The fix is intentionally OUT OF SCOPE for this ACT. A
small follow-up ACT must either (a) publish
`refs/factory/freeze/*` as a documented side-band refspec
through a normal `git push <remote> <refspec>` (no
`--force`), or (b) derive F from a canonical commit-trailer
in the frozen plan commit, eliminating the local-only
sideband entirely.

## NEW_FAILURES

`0`

The `TestV2VerifierJSONPreparedInvalidResultExitsVerifier`
and `TestClosureCLIV2*` failures observed during
verification are pre-existing on the parent commit
(`6da31e3`). They reproduce on the unmodified parent and
are classified `PRE_EXISTING` per the spec's classification
contract; they are unrelated to this ACT's changes.

## UNKNOWN_FAILURES

`0`

## REBASE_USED

`false` — ordinary forward merge only.

## FORCE_PUSH_USED

`false` — never `--force`, `--force-with-lease`, or `+refspec`.
`boundedPush` enforces `merge-base --is-ancestor REMOTE LOCAL`
before push and post-push `ls-remote` read-back.

## RESIDUE_MERGE_TIP

`3e58334`

The parked residue lineage (`a81d88d` → `3e58334` → `0fa191a`)
remains the FIRST real consumer of this product. This ACT
does not merge `3e58334` into `main`; that happens through
the new entrypoint AFTER this ACT lands.

## READY_FOR_FIRST_REAL_CONSUMER

`false` — `PARTIAL_PORTABILITY` (fresh-clone close fails
because the sideband ref is local-only).

`READY_FOR_FIRST_REAL_CONSUMER` will flip to `true` after
the small follow-up ACT closes `FREEZE_REF_PORTABILITY`.
Until then, real consumers must work on the same clone that
ran `leamas factory begin`.

## Defects fixed in this ACT

1. **Plan-path duplication bug**: `filepath.Join(planDir, planRel)`
   where `planDir = docs/closure-plans/` and
   `planRel = docs/closure-plans/<ACT>.json` produced
   `docs/closure-plans/docs/closure-plans/<ACT>.json`.
   Fixed by single canonical
   `planAbs := filepath.Join(repoRoot, planRel)`.
2. **Fake `mkdir -p` fixture**: production uses `os.MkdirAll`,
   not `mkdir -p`. Obsolete fixture removed; required
   mandatory assertions added (exactly one `update-ref --stdin`,
   zero standalone `update-ref`).
3. **Empty `baseline.tree_oid` in frozen plan**: dogfood
   canary exposed this. `BeginAct` now resolves
   `HEAD^{tree}` alongside `HEAD^{commit}` and populates
   both OIDs.

## Bootstrap close artifact chain

- Plan:    `docs/closure-plans/ACT-LEAMAS-FACTORY-CLOSURE-SIMPLIFIED-ENTRYPOINT01.json`
- Manifest:`docs/closure-manifests/ACT-LEAMAS-FACTORY-CLOSURE-SIMPLIFIED-ENTRYPOINT01.json`
- Report:  `docs/close-reports/ACT-LEAMAS-FACTORY-CLOSURE-SIMPLIFIED-ENTRYPOINT01.md`

`BOOTSTRAP_MANUAL_FSC=true`,
`BOOTSTRAP_EXCEPTION_ID=ACT-LEAMAS-FACTORY-CLOSURE-SIMPLIFIED-ENTRYPOINT01`,
and the marker
`THIS_IS_THE_FINAL_BOOTSTRAP_ACT_USING_AGENT_ORCHESTRATED_FSC`
are recorded in all three artifacts.
