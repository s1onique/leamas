# R1 cross-region proof ACT — closure appendix

This appendix preserves R10-R12, acceptance, shortcut, and successor text
split from the original ACT document solely to satisfy the 400-line
LLM-friendliness contract. The canonical ACT links here.

## R10 — Preserve the frozen remediation target

The following files were NOT modified:

```text
cmd/leamas/claim_commands.go
cmd/leamas/evidence_commands.go
internal/factory/dupcode/baseline.json
```

The live detector retains:

```text
TokenCount = 504

claim_commands.go:
    268–340

evidence_commands.go:
    310–382
```

`TestV4BaselineDelta_LiveTreeMatchesCommittedBaseline` PASSES,
confirming the canonical 504-token claim/evidence duplicate is
intact at its reviewed geometry.

## R11 — Minimal documentation

This ACT adds its canonical ACT and close-report Markdown files. The
close report states explicitly that this ACT closes only the R1
cross-region regression-proof defect. It does not close CORRECTION02,
CORRECTION01, the parent performance ACT, or the self-hosted-remediation
prerequisite. Lifecycle reconciliation belongs to the successor ACT.

## R12 — Commit closure

The focused test and documentation changes are committed on a clean
repository state. After committing, `git status --porcelain=v1` is empty.
Detached evidence binds to the literal final `HEAD`.

## Acceptance criteria

This ACT is PASSED only when:

1. the asymmetric right-side-extra fixture uses `alpha.go` and `beta.go`;
2. the sides resolve to distinct production region IDs;
3. the guard returns `false`;
4. the all-pairs candidate set contains the complete offset-100 run;
5. production canonical output structurally equals the oracle;
6. the mirrored left-side-extra fixture also passes;
7. the aligned distinct-region case proves the fast path remains covered;
8. the unconditional-diagonal mutation makes the regression fail;
9. the restored guarded implementation passes;
10. all focused and repository-wide verification passes;
11. the 504-token live finding remains unchanged;
12. the final commit and detached evidence bind to a clean literal `HEAD`.

## Prohibited shortcuts

This ACT verified none of the prohibited shortcuts:

- It did not use the same path for both sides.
- It did not test only the alignment predicate without final output.
- It did not test only final equality without fixture contracts.
- It did not infer fallback execution merely because output matched.
- It did not omit the exact offset-100 candidate assertions.
- It did not skip the mirrored asymmetric case.
- It actually mutated branch selection for the mutation proof.
- It did not commit the deliberately broken mutation.
- It did not claim completion of the broader CORRECTION02 wave.
- It did not begin self-hosted remediation.

## Immediate successor

After this ACT passes, execute
`ACT-LEAMAS-FACTORY-DUPCODE-V4-ALL-PAIRS-MATERIALIZATION-PERFORMANCE01-CORRECTION02-CORPUS-AND-EVIDENCE01`.

That successor owns the remaining corpus dimensions, structural inventory,
persistent fuzz regressions, 30-second fuzzing, benchmark confirmation,
whitespace cleanup, lifecycle reconciliation, and final performance-ACT
closure. Only after it passes may
`ACT-LEAMAS-FACTORY-DUPCODE-SELF-HOSTED-REMEDIATION01` begin.
