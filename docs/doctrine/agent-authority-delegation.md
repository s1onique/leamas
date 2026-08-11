# Doctrine: Agent Authority Delegation

This doctrine defines how authority is delegated to interactive coding agents
and editors that operate Leamas under an ACT. It complements
[agent-assisted-development.md](agent-assisted-development.md).

## D1 — Authority is explicit

Consequential operations require explicit delegation from the current
authoritative task contract (the active ACT).

The following authorities are independent:

```text
mutation
verification
expensive verification
commit
push
tag
closure / publication
history mutation
```

One authority MUST NOT imply another.

Editing a file does not grant commit authority. Committing does not
grant push, tag, or history-rewrite authority. Running focused tests
does not grant authority to run aggregate expensive gates.

## D2 — Default-deny publication authority

Unless the active ACT explicitly delegates otherwise, persistent agent
context MUST assume:

```text
EDIT_ALLOWED=true
COMMIT_ALLOWED=false
PUSH_ALLOWED=false
TAG_ALLOWED=false
HISTORY_REWRITE_ALLOWED=false
```

An ACT may override individual fields explicitly (for example,
`commit: exactly_one`). The absence of a publication section MUST
NOT be interpreted as permission to commit, push, tag, or rewrite
history.

## D3 — Checkpoint is not publication

Editor checkpoints, restore points, Compare operations, local undo
state, or equivalent workspace snapshots are NOT Git publication
authority.

They MUST NOT imply any of:

```text
git add
git commit
git tag
git push
```

## D4 — Verification authority is tiered

Authority for verification is tiered:

```text
TIER_0_READ_ONLY
    source inspection
    git diff / status / show / log
    deterministic read-only inspection

TIER_1_FOCUSED
    ACT-owned unit tests
    named focused test umbrellas
    focused Factory verifiers explicitly listed by the ACT
    gofmt / diff-check for ACT-owned files

TIER_2_AFFECTED
    affected-package tests
    affected-package vet / build / race tests
    only when the ACT delegates them

TIER_3_EXPENSIVE_CANONICAL
    make factorize
    make gate-dupcode
    make gate
    full expensive dupcode lanes
    any future canonical aggregate lane designated expensive
```

For interactive coding-agent / editor ACT execution:

```text
TIER_3_EXPENSIVE_CANONICAL = DENY
```

unless the current ACT explicitly authorizes the exact command or
lane.

## D5 — Frozen-plan execution is distinct

This rule MUST NOT prevent Leamas from executing a gate that is
explicitly declared by an immutable validated closure plan.

The authority model is:

```text
interactive agent invents expensive command
    -> unauthorized

current ACT explicitly delegates exact expensive command
    -> authorized

validated immutable closure plan contains exact command
    -> authorized Factory execution
```

Authority comes from the contract, not from caller identity alone.

This doctrine does not change R6 execution semantics. It only
records that frozen-plan authority is distinct from interactive
authority.

## D6 — NOT RUN is evidence

When an expensive gate is not authorized, the honest result is:

```text
make factorize=NOT RUN
make gate-dupcode=NOT RUN
make gate=NOT RUN
```

Reporting NOT RUN is valid evidence. An agent MUST NOT run a
forbidden command merely so it can report PASS.

## D7 — No inferred commit after success

The following reasoning is forbidden:

```text
tests passed -> therefore commit
task looks complete -> therefore commit
```

The correct sequence is:

```text
tests passed
    consult ACT publication authority
    commit only if delegated
```

A successful verification result MUST NOT silently upgrade to commit
authority.

## D8 — Persistent agent context contract

Persistent agent context files (`AGENTS.md`, `.clinerules/leamas.md`)
MUST:

- refuse to grant expensive-gate authority implicitly;
- refuse to grant commit, push, or tag authority implicitly;
- distinguish editor checkpoints from Git publication;
- require explicit ACT authorization for each expensive command;
- allow a validated closure plan to be its own authority source;
- allow truthful NOT RUN reporting.

The agent-context verifier enforces this contract deterministically
against canonical anchor phrases. The verifier MUST NOT accept
unguarded substrings such as `make factorize` or `make gate` as proof
of correct policy.

## See Also

- [agent-assisted-development.md](agent-assisted-development.md)
- [factory-meta-loop.md](factory-meta-loop.md)
- [verification-witness.md](verification-witness.md)
- [docs/factory/agent-context-files.md](../factory/agent-context-files.md)
