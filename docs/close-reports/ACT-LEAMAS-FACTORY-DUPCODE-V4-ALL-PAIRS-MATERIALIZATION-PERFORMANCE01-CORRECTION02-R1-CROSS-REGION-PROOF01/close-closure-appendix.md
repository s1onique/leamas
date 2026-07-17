# R1 cross-region proof close report — closure appendix

This appendix preserves R11-R12, shortcut, accounting, and successor text
split from the original close report solely to satisfy the 400-line
LLM-friendliness contract. The canonical close report links here.

## R11 — Documentation

The ACT added canonical ACT and close-report Markdown. It closed only the
R1 cross-region regression-proof defect, not CORRECTION02, CORRECTION01,
the parent performance ACT, or self-hosted remediation. Historical parent
report reconciliation remained assigned to the successor ACT.

## R12 — Commit closure

The focused test and documentation changes were committed on a clean
repository state. Detached evidence recorded:

```text
commit_oid     = the literal R1 HEAD commit
head_tree_oid  = git rev-parse HEAD^{tree}
index_tree_oid = git write-tree
head_tree_oid == index_tree_oid
```

No R1 commit was made after its detached evidence was written.

## Prohibited shortcuts

The R1 ACT verified none of its prohibited shortcuts:

- The fixture did not use the same path for both sides.
- Guard rejection was paired with final output and fixture contracts.
- Final equality was paired with preconditions and candidate geometry.
- Exact fallback candidate membership and partition existence were pinned.
- The mirrored asymmetric case was covered.
- The mutation was actually applied, observed failing, then restored.
- The deliberately broken mutation was not committed.
- Broad fuzz and corpus work remained deferred to the successor ACT.
- Self-hosted remediation did not begin.

## Honest accounting

CORRECTION01's asymmetric fixture reused the left-side path on its right
side, collapsing both sides to one syntax region. The R1 ACT introduced a
path-aware constructor and independent fixture, guard, candidate geometry,
final structural equality, and mutation assertions. Together they proved
the corrected fixture exercises the cross-region all-pairs fallback.

The test files remained individually below 400 lines. The canonical
504-token claim/evidence duplicate stayed intact at lines 268–340 and
310–382, and the committed baseline was unchanged.

## Follow-up ACTs

The successor was
`ACT-LEAMAS-FACTORY-DUPCODE-V4-ALL-PAIRS-MATERIALIZATION-PERFORMANCE01-CORRECTION02-CORPUS-AND-EVIDENCE01`.
It owns remaining corpus dimensions, persistent fuzzing, benchmark and
whitespace evidence, lifecycle reconciliation, and final closure. Only
after that successor passes may
`ACT-LEAMAS-FACTORY-DUPCODE-SELF-HOSTED-REMEDIATION01` begin.
