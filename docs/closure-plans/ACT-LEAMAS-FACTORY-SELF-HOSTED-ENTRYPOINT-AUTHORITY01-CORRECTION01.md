# Closure Plan: ACT-LEAMAS-FACTORY-SELF-HOSTED-ENTRYPOINT-AUTHORITY01-CORRECTION01

This document is the human-readable companion to the strict
`ACT-LEAMAS-FACTORY-SELF-HOSTED-ENTRYPOINT-AUTHORITY01-CORRECTION01.json`
plan. The strict JSON plan is the freeze contract enforced by
`leamas factory close plan validate` and bound by the closure
manifest. This document carries the descriptive context that the
strict JSON cannot hold.

## Predecessor

| Identity | Value |
|---|---|
| Predecessor ACT | `ACT-LEAMAS-FACTORY-SELF-HOSTED-ENTRYPOINT-AUTHORITY01` |
| Predecessor declared subject OID | `06c5115a5d2e7c4f4a26f5c1e3b9a8d7c6e5f4a3` (does not exist) |
| Predecessor actual subject OID | `06c51158d104c20eec389736a2a0bcff06743630` |
| Predecessor subject tree | `897587b88dc06a6f40d68c796f4ed186dbd91b6e` |
| Predecessor plan first appeared in | `d20fc2c0f856b8a99330b626cd87fd256dc0a931` (after subject) |
| Predecessor plan tree | `49d10a413026ff5d655736dfb03da1ff0df1bae8` |

## Predecessor closure status

`INVALID`. Reasons:

- `declared_subject_object_missing`: the predecessor reported an
  implementation subject using
  `06c5115a5d2e7c4f4a26f5c1e3b9a8d7c6e5f4a3` which does not exist
  in this repository.
- `no_pre_subject_plan_freeze`: the predecessor closure plan first
  appears in `d20fc2c0f856b8a99330b626cd87fd256dc0a931`, which is
  AFTER the actual subject `06c51158d104c20eec389736a2a0bcff06743630`.
- `plan_contains_unresolved_placeholder`: the predecessor plan's
  `baseline.tree_oid` is `TO_BE_FILLED` which is a closure
  placeholder.
- `closure_manifest_missing`: no `docs/closure-manifests/<ACT>.json`
  was committed.
- `attestation_missing`: no post-closure attestation was committed.
- `annotated_tag_missing`: no annotated tag was created.

## Required behavioral checks

- `leamas doctor executable` reports verdict `authoritative` with a
  resolvable embedded commit and resolvable repository HEAD.
- `leamas doctor executable --json` returns the canonical
  machine-readable contract.
- Required capabilities (`factory_digest_auto_range`,
  `factory_self_hosted_authority`, `closure_protocol`) are satisfied.
- `leamas bootstrap self --exact` rebuilds a binary bound to the
  intended commit, records SHA-256, and verifies the embedded
  identity matches.
- A stale global `leamas` binary earlier in a controlled PATH cannot
  silently execute an obsolete contract.
- PATH ambiguity is diagnosed.
- Symlinked canonical binary is resolved correctly.

## Required executable-authority checks

- `factory_digest_auto_range` capability exposed at runtime.
- `factory_self_hosted_authority` capability exposed at runtime.
- `closure_protocol` capability exposed at runtime.

## Zero-range digest acceptance

```bash
./bin/leamas factory digest \
  --output /tmp/self-hosted-authority-correction01-digest.txt
```

No `--range` is permitted.

- Before closure: must fail closed with `ErrNoACTAuthority` (the
  current HEAD is evidence-only).
- After closure: must pass and include `## LIFECYCLE`,
  `AUTO_RANGE_STRATEGY`, `ACT_ID`, `RANGE_BASE`, `RANGE_HEAD`,
  `LIFECYCLE_FREEZE`, `LIFECYCLE_SUBJECT`, `LIFECYCLE_CLOSURE`,
  `INCLUDED_COMMITS`, `GENERATOR_COMMIT`, `REPOSITORY_HEAD`,
  `GENERATOR_STALE`.

## Timeout limits

- `go test -count=1 ...` : 600 seconds
- `go test -count=20 ...` : 600 seconds
- `go test -race -count=5 ...` : 600 seconds
- `go vet ...` : 300 seconds
- `go build ...` : 600 seconds
- `make gate-fast` : 600 seconds
- `git diff --check` : 60 seconds

## Excluded checks

- `factorize`: not required for this ACT; authority, digest, and
  closure tests cover the ACT scope. `factorize` is also explicitly
  refused in editor/Cline terminal contexts unless
  `LEAMAS_ALLOW_FULL_FACTORIZE=1` is supplied.
- `full_canonical_gate`: expensive verification is refused in
  editor/Cline terminal contexts by default.

## Expected closure artifacts

| Artifact | Path |
|---|---|
| Closure manifest | `docs/closure-manifests/ACT-LEAMAS-FACTORY-SELF-HOSTED-ENTRYPOINT-AUTHORITY01-CORRECTION01.json` |
| Close report | `docs/close-reports/ACT-LEAMAS-FACTORY-SELF-HOSTED-ENTRYPOINT-AUTHORITY01-CORRECTION01.md` |
| Lifecycle erratum | `docs/lifecycle-errata/ACT-LEAMAS-FACTORY-SELF-HOSTED-ENTRYPOINT-AUTHORITY01.json` |
| Annotated tag | `act/leamas-factory-self-hosted-entrypoint-authority01-correction01` |

## Lifecycle policy

- Freeze (F1) must precede subject (S1).
- Plan must be immutable after freeze.
- Verification must be bound to exact S1.
- Tag must be an annotated tag object.
- Peeled target must equal closure commit (C1).
- Zero-range digest must pass after closure without `--range`.
