# ACT-LEAMAS-FACTORY-SELF-HOSTED-ENTRYPOINT-AUTHORITY01-CORRECTION01 Close Report

## Verdict

PASS

## Subject

- Commit: `a9753acc347f26aa444aa2c99b01162c532f0136`
- Tree: `e8279b82593d13d44f62436142da15016ee4b62a`

## Plan

- Path: `docs/closure-plans/ACT-LEAMAS-FACTORY-SELF-HOSTED-ENTRYPOINT-AUTHORITY01-CORRECTION01.json`
- SHA-256: `688c7049f01cf85e81ec1729aedb3184a0bb439528e65edb4e8716aa2e271d9f`

## Checks

Ordered results: 7.

| Check | Result | Duration | Exit |
|---|---|---:|---:|
| focused-count-1 | PASS | 11519ms | 0 |
| focused-count-20 | PASS | 221135ms | 0 |
| focused-race-5 | PASS | 62029ms | 0 |
| vet | PASS | 2595ms | 0 |
| build | PASS | 192ms | 0 |
| gate-fast | PASS | 26453ms | 0 |
| diff-check | PASS | 21ms | 0 |

## Artifacts

| Artifact | Status | SHA-256 | Bytes |
|---|---|---|---:|
| manifest | PASS | c5469053af59e903da93807547d2faa798b0a5aa5c5f8930ce1db47777bb9b68 | 13814 |
| report | PASS | f0342d9d945b66e8d3f610afa7616e3260abf50e8a590ad092954d9168fbbb75 | 1644 |
| erratum | PASS | a0a2e757e17e4a2d2ccd2dd4ec3774b58add44ee518822f12e79bedc7fd30c43 | 2443 |

## Excluded checks

- `dupcode` — No dupcode-owned source or registration changed.

## Patch hygiene

- Git diff check: PASS
- Diagnostics: 0
- Tracked full digest policy: PASS
- Closure-policy diagnostics: 0

## Runner identity

- Leamas version: `0.1.0+dev.a9753acc347f.20260724T085252Z`
- Binary SHA-256: `247acdbf62342f915a701e6798c59b8c5cfd0190f48a42af8faf594b78c02448`
- VCS revision: `a9753acc347f26aa444aa2c99b01162c532f0136`
- VCS modified: `false`

## Lifecycle transition

Verification state: VERIFIED

The immutable closure tag is created after this report and manifest are committed. The annotated-tag object identity remains external Git evidence.
