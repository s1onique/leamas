# ACT-LEAMAS-FACTORY-CLOSURE-DIGEST-AUTHORITY-CONVERGENCE01 Close Report

## Verdict

FAIL

## Subject

- Commit: `98c6c3d364d790a999a840ffa34c11a1053deecf`
- Tree: `1f7a4618a2043314e54d5164e68dbba28115404f`

## Plan

- Path: `docs/closure-plans/ACT-LEAMAS-FACTORY-CLOSURE-DIGEST-AUTHORITY-CONVERGENCE01.json`
- SHA-256: `c1c3046ac61505444ec7ddadb2925849e71859cb5105429f5192e86e6c833c63`

## Checks

Ordered results: 9.

| Check | Result | Duration | Exit |
|---|---|---:|---:|
| gofmt | PASS | 66ms | 0 |
| vet | PASS | 464ms | 0 |
| test-digest | PASS | 5891ms | 0 |
| test-authority | PASS | 1521ms | 0 |
| test-closure | PASS | 5035ms | 0 |
| test-cmd | PASS | 14842ms | 0 |
| build | PASS | 1019ms | 0 |
| gate-fast | PASS | 23973ms | 0 |
| diff-check | PASS | 9ms | 0 |

## Artifacts

| Artifact | Status | SHA-256 | Bytes |
|---|---|---|---:|
| manifest | MISSING | — | 0 |
| report | MISSING | — | 0 |
| erratum | MISSING | — | 0 |

## Excluded checks

None.

## Patch hygiene

- Git diff check: PASS
- Diagnostics: 0
- Tracked full digest policy: PASS
- Closure-policy diagnostics: 0

## Runner identity

- Leamas version: `0.1.0+dev.98c6c3d364d7.20260724T114436Z`
- Binary SHA-256: `f730c41e1b0a4008fb277d1603e969027f4c644fad5f3355cc68f76e85244e8f`
- VCS revision: `98c6c3d364d790a999a840ffa34c11a1053deecf`
- VCS modified: `false`

## Lifecycle transition

Verification state: IMPLEMENTED

The immutable closure tag is created after this report and manifest are committed. The annotated-tag object identity remains external Git evidence.
