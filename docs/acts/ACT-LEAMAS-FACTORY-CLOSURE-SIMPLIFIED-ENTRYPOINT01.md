# ACT-LEAMAS-FACTORY-CLOSURE-SIMPLIFIED-ENTRYPOINT01

## STATUS

```text
CLOSED (bootstrap exception)
```

## MISSION

Introduce one canonical Leamas operation that accepts a committed
development subject plus ACT intent and internally owns the entire
closure transaction — replacing the manual F/S/C choreography that
currently requires the agent to author freeze and closure commits.

Conceptual UX:

```bash
leamas factory close \
    --act ACT-LEAMAS-FACTORY-CLOSURE-SIMPLIFIED-ENTRYPOINT01 \
    --subject <committed-S> \
    --lane fast
```

The coding agent supplies ONLY:

```text
ACT identity
committed subject
requested lane
delegated publication policy
```

The agent does NOT supply:

```text
freeze commit
closure commit
plan JSON
manifest JSON
subject worktree
binary identity
gate counts
B2 completeness
execution observations
fixed-point state
```

The product internally establishes:

```text
remote/base authority
subject commit + tree
freeze authority
exact-subject isolated worktree
exact-subject Leamas binary
one real requested gate lane
verbatim process/gate capture
authoritative execution observations
derived completeness
manifest/report
closure authority
fixed_point / rerun_required
delegated publication
```

No synthetic evidence. No stubbed B1. No caller-created subject
worktree. No manually supplied closure commit.

## BOOTSTRAP EXCEPTION

This ACT alone may use the existing explicit F/S/C choreography
to close itself. The bootstrap exception is one-time and
documented explicitly in the close report as:

```text
THIS_IS_THE_FINAL_BOOTSTRAP_ACT_USING_AGENT_ORCHESTRATED_FSC
```

Bootstrap rules:

```text
BOOTSTRAP_MANUAL_FSC_ALLOWED=true   # this ACT only
no rebase
no force push
real existing closure verifier
ordinary forward publication
```

After publication, the policy flips:

```text
BOOTSTRAP_MANUAL_FSC_ALLOWED=false
CANONICAL_AGENT_CLOSURE=simplified-entrypoint
```

## OUT OF SCOPE

- The 73 unrelated baseline failures observed in
  ACT-LEAMAS-MACOS-PATH-AUTHORITY-RESIDUE-CONVERGENCE01.
- Runner hermeticity.
- New lanes beyond `fast` in this ACT (deferred).
- Closure evidence byte-stability beyond what is required for
  fixed-point convergence.
- Removal of existing `plan validate | run | verify | render |
  tag create | status | chain | attest` commands. They remain
  available as internal / debugging primitives.

## NON-GOALS

- Replace the F/S/C model internally; F/S/C stays.
- Reopen any closed ACT.
- Force-push history.
- Rewrite the closure protocol.
- Manually publish the parked residue convergence subject
  (3e58334). That happens through the new entrypoint AFTER this
  ACT lands.

## SUCCESS CRITERIA

The product ACT is PASS only when:

```text
single canonical closure entrypoint exists
agent input does not require F/C OIDs
exact committed S is authoritative
real isolated execution works
real fast gate is captured
completeness is derived
fixed-point state is derived
rerun_required works
delegated publication works
non-fast-forward publication fails closed
production canary PASS
existing explicit machinery still works
bootstrap F/S/C closure PASS
origin/main published by ordinary forward update
```

## POST-LANDING IMMEDIATE DOGFOOD

Once the simplified entrypoint is on `main`, the first real
consumer is the already-waiting residue convergence subject
`3e58334`. Topology:

```text
d0a03fc main
   ├─ ACT-LEAMAS-FACTORY-CLOSURE-SIMPLIFIED-ENTRYPOINT01 (lands first)
   └─ ACT-LEAMAS-MACOS-PATH-AUTHORITY-RESIDUE-CONVERGENCE01 line:
      a81d88d (parent ACT)
      3e58334 (CORRECTION01)
```

Merge strategy: ordinary forward merge on a continuation branch,
no rebase. Both `3e58334` and `closure-tooling-main` remain
ancestors. Then rerun the residue focused matrix on the combined
subject and invoke the simplified closure against it.

## COMMIT

Implementation subject is one or more bounded forward commits
on top of `d0a03fc`. The bootstrap close is the explicit
F/S/C choreography on top of the implementation subject.
