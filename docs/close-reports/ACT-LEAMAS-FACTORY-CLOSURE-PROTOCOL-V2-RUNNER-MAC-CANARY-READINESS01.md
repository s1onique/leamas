# ACT-LEAMAS-FACTORY-CLOSURE-PROTOCOL-V2-RUNNER-MAC-CANARY-READINESS01 Close Report

## Verdict

PASS

## Subject

- Commit: `20a5c6387655b7bff0236ea2becfed595c949ee0`
- Tree: `1ebf1e660019749b2a0a8177eba9b358d1d9237a`

## Plan

- Path: `docs/closure-plans/ACT-LEAMAS-FACTORY-CLOSURE-PROTOCOL-V2-RUNNER-MAC-CANARY-READINESS01.json`
- SHA-256: (ACT document carries no frozen plan; the runner-readiness ACT is
  proof-only and is closed through the v2 hermetic + CLI subprocess tests,
  not through the v1 closure verifier.)

## Checks

Ordered results: 5.

| Check | Result | Duration | Exit |
|---|---|---:|---:|
| v2-mac-canary-full-runner-descendant-proof | PASS | <1s | 0 |
| v2-mac-canary-working-assertion-rejects-descendant-plan | PASS | <1s | 0 |
| v2-mac-canary-no-caller-state-drift | PASS | <1s | 0 |
| v2-mac-canary-no-worktree-leak | PASS | <1s | 0 |
| cli-v2-mac-canary-dogfood | PASS | 1.5s | 0 |

## Artifacts

None committed. The installed-style dogfood evidence (manifest, stdout,
stderr, evidence directory) lives in `/tmp/leamas-mac-canary-dogfood` and
is intentionally NOT committed to the Leamas repository. The Mac canary
handoff treats the ClineMM repository similarly: evidence and manifest
paths must live outside `<clinemm>`.

## Excluded checks

None.

## Patch hygiene

- Git diff check: PASS
- Diagnostics: 0
- Tracked full digest policy: PASS
- Closure-policy diagnostics: 0

## Runner identity

- Leamas version: `0.1.0+dev`
- Binary SHA-256: `59838de51b3725b4e326ed660aceaf20e6decc2bf286462f1ef10f40a9468e06`
- VCS revision: `20a5c6387655b7bff0236ea2becfed595c949ee0`
- VCS modified: `false`

## Lifecycle transition

Verification state: VERIFIED

## Final report fields

```text
ACT_ID=ACT-LEAMAS-FACTORY-CLOSURE-PROTOCOL-V2-RUNNER-MAC-CANARY-READINESS01
STATUS=PASS
BASE_COMMIT=3fb13db2f323cd86d96b49d3c1375b2a3a8370f9
BASE_TREE=7d76e1aefb9a3e7d34a4f7c1c2d1bf2c9d4ac83f
FINAL_COMMIT=20a5c6387655b7bff0236ea2becfed595c949ee0
FINAL_TREE=1ebf1e660019749b2a0a8177eba9b358d1d9237a
WORKTREE_STATUS=clean

FULL_RUNNER_DESCENDANT_PROOF=PASS (TestV2MacCanaryFullRunnerDescendantProof)
WORKING_PLAN_ASSERTION_RESULT=PASS (TestV2MacCanaryWorkingAssertionRejectsDescendantPlan rejects with V2CodeWorkingPlanMismatch before any executor call; manifest absent)

DOGFOOD_COMMAND=cd /tmp/leamas-mac-canary-dogfood/cwd && /home/chistyakov/Projects/leamas/bin/leamas factory close run-v2-authority --protocol-version 2 --plan-contract-version 1 --repository <repo> --subject 6037a3b9c1b162a1323981f3303e24a54941b03e --freeze e2b5a0e42a289d05a6e10168c6256a371f67d425 --plan-path docs/closure-plans/MAC-CANARY-DOGFOOD.json --evidence-directory /tmp/leamas-mac-canary-dogfood/evidence --manifest-output /tmp/leamas-mac-canary-dogfood/manifest.json
DOGFOOD_EXIT=0
DOGFOOD_STDOUT_SHA256=a12b7cb43c9d9134b5bb1b35e9096b66775d9e92e7611d1cc92b02edd6782a87
DOGFOOD_STDERR_SHA256=59639d002d532247796ad28dc3cfcae6f099e45503e6bbda45ca91fd251504c7
DOGFOOD_MANIFEST_SHA256=883a66b05bbef539a62fca91cf83da917415d9beab4b833ab409e86b80968916
DOGFOOD_BINARY_COMMIT=20a5c6387655b7bff0236ea2becfed595c949ee0
DOGFOOD_BINARY_SHA256=59838de51b3725b4e326ed660aceaf20e6decc2bf286462f1ef10f40a9468e06
DOGFOOD_BINARY_MODIFIED=false

DOGFOOD_SUBJECT=6037a3b9c1b162a1323981f3303e24a54941b03e
DOGFOOD_FREEZE=e2b5a0e42a289d05a6e10168c6256a371f67d425
DOGFOOD_CALLER_HEAD=f6a28493779a23c0378e92d71bdd94efeafac389
DOGFOOD_EXECUTION_TREE=299576640e00495c1f4c04aa44366bae62876de8
DOGFOOD_PLAN_BLOB=111a356b88f5669011640c1c42cfb53e7bae053d

DOGFOOD_CALLER_STATUS_BEFORE=""
DOGFOOD_CALLER_STATUS_AFTER=""
DOGFOOD_WORKTREES_BEFORE=1 entry (main worktree only)
DOGFOOD_WORKTREES_AFTER=1 entry (main worktree only)

MAC_INSPECTION_COMMANDS=
  git -C <clinemm> cat-file -e 56fd526e1923f2546fa0aeb53a0dc6e7501e5061^{commit}
  git -C <clinemm> cat-file -e 01822bf5c8b99e5a4b89a6761a713ec3603754b0^{commit}
  git -C <clinemm> merge-base --is-ancestor 56fd526e1923f2546fa0aeb53a0dc6e7501e5061 01822bf5c8b99e5a4b89a6761a713ec3603754b0
  git -C <clinemm> ls-tree -r --name-only 01822bf5c8b99e5a4b89a6761a713ec3603754b0
  (after recovering P from the listing)
  git -C <clinemm> show 01822bf5c8b99e5a4b89a6761a713ec3603754b0:"$P"

MAC_RUN_COMMAND=
  /home/chistyakov/Projects/leamas/bin/leamas factory close run-v2-authority \
      --protocol-version 2 \
      --plan-contract-version 1 \
      --repository <clinemm> \
      --subject 56fd526e1923f2546fa0aeb53a0dc6e7501e5061 \
      --freeze 01822bf5c8b99e5a4b89a6761a713ec3603754b0 \
      --plan-path <P> \
      --evidence-directory <outside-clinemm-evidence> \
      --manifest-output <outside-clinemm-manifest.json>

MAC_HANDOFF=
  All five inspection commands above are non-mutating. The runner
  invocation uses the exact-final-tip binary built from
  20a5c6387655b7bff0236ea2becfed595c949ee0 (vcs.modified=false).
  Evidence and manifest paths MUST live outside <clinemm>. The
  frozen plan path P is recovered from the ls-tree listing, not
  guessed.

LOCAL_GATES=gofmt OK, go vet ./... OK, go test -count=1 ./internal/factory/closure/... OK (incl. TestV2MacCanary*), go test -count=1 -run 'TestClosureCLIV2MacCanaryDogfood' ./cmd/leamas/... OK, static build OK
PRE_EXISTING_GATE_FINDINGS=llm-friendly long_line complaints in ACT-LEAMAS-FACTORY-CLOSURE-PROTOCOL-V2-RUNNER-AUTHORITY-CORRECTION01/02/03 docs (unrelated to this ACT); forbidden-patterns pre-existing platform-specific build_ignored files (unrelated to this ACT)
UNRESOLVED_BLOCKERS=v2 closure-commit verifier (sole permitted unresolved item per ACT 5 acceptance)
```

## Closure

The runner-readiness sequence (ACTs 1..5) closes successfully.
ACT 5 contributes:

- `internal/factory/closure/v2_mac_canary_test.go` — four
  hermetic S < F < D tests (full-runner descendant proof,
  working-plan assertion rejection, no caller-state drift,
  no worktree leak). The run check, built from the contract-valid
  fixture builder, proves the executor ran against S^{tree} only.

- `cmd/leamas/factory_close_v2_mac_canary_test.go` — installed-style
  external dogfood that builds the exact-final-tip binary via
  the production LDFLAGS and invokes `factory close run-v2-authority`
  from a temp directory outside the Leamas checkout, against a
  fresh hermetic S < F < D repo in another temp directory. The
  build is routed through `internal/execution` so the exec-gate
  stays green.

- `docs/acts/ACT-LEAMAS-FACTORY-CLOSURE-PROTOCOL-V2-RUNNER-MAC-CANARY-READINESS01.md`
  — the ACT document and Mac inspection handoff.

The binary built from FINAL_COMMIT reports
`vcs.modified=false` and `vcs.revision=20a5c6387655b7bff0236ea2becfed595c949ee0`.
The Linux-side dogfood succeeds end-to-end (`DOGFOOD_EXIT=0`)
without mutating the caller repository. The Mac handoff provides
non-mutating inspection commands and a single run command whose
evidence and manifest paths live outside `<clinemm>`.

The sole remaining blocker — the v2 closure-commit verifier —
is intentionally deferred and documented above.
