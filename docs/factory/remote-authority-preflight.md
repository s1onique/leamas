# Remote Authority Preflight (Factory Doctrine)

## Status

**DOCUMENTED — proposed, not yet enforced.**

This doctrine records a Factory rule that was identified as a hard prerequisite
during `ACT-LEAMAS-CORRECTION12-PUBLIC-V3-RECONCILIATION-AND-MAC-HANDOFF03`.
A previous reconciliation ACT was built on a stale remote-tracking ref and
reported `PUBLIC_UNIQUE=1` instead of the true `PUBLIC_UNIQUE=3`, which
mis-sized every downstream phase. This rule prevents that failure mode.

## Rule

**REMOTE_AUTHORITY_PREFLIGHT**

Any ACT whose acceptance criteria depend on the current state of a remote
Git ref **MUST** establish current remote authority before using that ref
for any of:

- divergence counts
- ancestry claims
- merge/rebase planning
- commit accounting
- transfer planning
- release authority

A freshly-fetched tracking ref is the only acceptable source. A
`git rev-parse origin/<branch>` reading that has not been preceded in the
same ACT by a verified `git fetch` is **NOT** acceptable evidence.

## Required evidence

The remote-advertised ref and the freshly-fetched tracking ref **MUST**
agree byte-for-byte:

```text
remote_advertised_ref == freshly_fetched_tracking_ref
```

This equality must be checked explicitly in the ACT log, not assumed.

## Canonical machine sequence

```bash
git ls-remote --symref origin HEAD
git ls-remote origin refs/heads/<ref>
git fetch origin --prune --tags
git rev-parse origin/<ref>
```

Compare the `ls-remote` value with the post-fetch `rev-parse` value.
If they differ, stop and resolve the authority ambiguity before
proceeding.

## Why this is required

`git fetch` updates remote-tracking refs and obtains the objects needed
for their histories. A `git rev-parse origin/<ref>` reading taken
before the fetch reflects the last-known state of the local clone, not
the current state of the remote.

When a remote branch has advanced silently between two ACTs, the local
tracking ref carries the stale tip. Any divergence calculation
(`git rev-list --left-right --count`) made against that stale tip
under-reports the actual remote-side delta, which under-dimensions
reconciliation phases that target the conflict surface, conflict
resolution, and bundle transfer.

The failure mode is silent: stale-ref reconciliation looks like a
clean trivial merge because the remote's new commits are not yet
visible to the local clone.

## What this rule does NOT cover

- Verifying remote authentication or transport security.
- Resolving multi-remote authority (only one remote is canonical per ACT).
- Substituting `git ls-remote` for `git fetch` as the authority step.
  `ls-remote` is the **advertised** value; the local clone's tracking
  ref must still be updated before it is treated as authoritative.

## Reference incidents

- `ACT-LEAMAS-HISTORY-RECONCILIATION-AND-MAC-HANDOFF01` — reported
  `PUBLIC_UNIQUE=1` based on a stale `origin/main` tip; the real
  value was `3` (one v2 canary + two gatesummary v3 commits). The
  reconciliation plan was dimensioned for the wrong graph.
- `ACT-LEAMAS-CORRECTION12-PUBLIC-V3-RECONCILIATION-AND-MAC-HANDOFF03`
  — this ACT re-measured at the start and again immediately before the
  real merge, and detected `PUBLIC_UNCHANGED` both times. The
  re-measurement caught a stale-ref regression that would otherwise
  have propagated to the merged tree.

## Enforcement

This rule is currently documented only. A future ACT
(`factory: implement REMOTE_AUTHORITY_PREFLIGHT enforcement`) may add a
verifier that inspects ACT manifests and the close-report evidence to
flag ACTs that consume remote ref state without a documented preflight
in the same ACT.
