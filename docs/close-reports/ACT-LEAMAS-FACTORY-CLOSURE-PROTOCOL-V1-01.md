# ACT-LEAMAS-FACTORY-CLOSURE-PROTOCOL-V1-01 Close Report

## Verdict

FAIL

## Subject

- Commit: `22d21726582febfb43ceb9fb56d49b470fdf83b6`
- Tree: `5f76d0162804a417d4f95795dd4ae0af5380db36`

## Plan

- Path: `docs/closure-plans/ACT-LEAMAS-FACTORY-CLOSURE-PROTOCOL-V1-01.json`
- SHA-256: `abf89f664dd19b7bf99be85a6b1e0c382526344f21dccbe164fdc7ea6d3deef0`

## Checks

Ordered results: 7.

| Check | Result | Duration | Exit |
|---|---|---:|---:|
| focused-count-1 | PASS | 14273ms | 0 |
| focused-count-20 | PASS | 224277ms | 0 |
| focused-race-5 | FAIL | 10ms | 2 |
| vet | NOT_RUN_DUE_TO_PRIOR_FAILURE | 0ms | — |
| build | NOT_RUN_DUE_TO_PRIOR_FAILURE | 0ms | — |
| gate-fast | NOT_RUN_DUE_TO_PRIOR_FAILURE | 0ms | — |
| diff-check | NOT_RUN_DUE_TO_PRIOR_FAILURE | 0ms | — |

## Artifacts

None.

## Excluded checks

- `dupcode` — No dupcode-owned source or registration changed.

## Patch hygiene

- Git diff check: PASS
- Diagnostics: 0
- Tracked full digest policy: PASS
- Closure-policy diagnostics: 0

## Runner identity

- Leamas version: `0.1.0+dev.22d21726582f.20260723T112502Z`
- Binary SHA-256: `085f0f7713685220f21aad20e090a0227a42b91b426047aff27f1b6d2f47642c`
- VCS revision: `22d21726582febfb43ceb9fb56d49b470fdf83b6`
- VCS modified: `false`

## Lifecycle transition

Verification state: IMPLEMENTED

The immutable closure tag is created after this report and manifest are committed. The annotated-tag object identity remains external Git evidence.
