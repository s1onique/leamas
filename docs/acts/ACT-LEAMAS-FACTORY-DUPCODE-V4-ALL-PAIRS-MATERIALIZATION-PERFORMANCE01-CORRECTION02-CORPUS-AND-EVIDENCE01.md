# ACT-LEAMAS-FACTORY-DUPCODE-V4-ALL-PAIRS-MATERIALIZATION-PERFORMANCE01-CORRECTION02-CORPUS-AND-EVIDENCE01

## Status

PASSED

## Parent correction wave

`ACT-LEAMAS-FACTORY-DUPCODE-V4-ALL-PAIRS-MATERIALIZATION-PERFORMANCE01-CORRECTION02`

## Completed prerequisite

`ACT-LEAMAS-FACTORY-DUPCODE-V4-ALL-PAIRS-MATERIALIZATION-PERFORMANCE01-CORRECTION02-R1-CROSS-REGION-PROOF01`

## Baseline

```text
HEAD = 169efd98af2a0e2d8808910eb5baf4f9d867582f
tracked worktree = clean
index = clean
pre-existing detached R1 evidence = one known untracked path
```

## Intent

Close the guarded V4 materialization performance lineage with:

1. exactly one primary fixture for each of the 17 required semantic
   dimensions;
2. structural fixture contracts before differential execution;
3. explicit multi-region ownership, including unowned spans and one
   path with different ordinals;
4. one authoritative complete internal structural comparator;
5. genuine aligned and unaligned shuffled-input proofs;
6. a bounded, deterministic, semantically rich fuzz wire format;
7. complete named deterministic fuzz seeds and persistent minimized
   regressions;
8. a successful clean-cache 30-second fuzz run;
9. accepted-baseline and forced-all-pairs N128 benchmark confirmation;
10. benchmark whitespace and lifecycle-document reconciliation;
11. complete repository verification and commit-bound detached evidence.

## Final algorithm contract

```text
aligned cross-region sequences:
  O(N) diagonal fast path per region pair

unaligned cross-region sequences:
  O(N_left × N_right) conservative all-pairs fallback

within-region pairing:
  separately bounded or quadratic according to occurrence geometry
```

Alignment requires equal counts, equal base-window widths, and equal
per-index start/end deltas. A 30-second fuzz RED in this ACT exposed the
missing base-width precondition; the smallest guard tightening routes
that geometry to the existing conservative fallback.

## Canonical corpus dimensions

```text
AlignedN8
AlignedN32
AlignedN128
LeadingExtraLeft
LeadingExtraRight
MiddleExtra
TrailingExtra
UnequalCardinality
NonUniformSpacing
OffIndexMaximalChain
TwoIndependentOffsetChains
ThreeRegionsAsymmetric
RepeatedWithinRegion
ShuffledRawInput
UnownedWindow
DuplicateRawWindow
SamePathDifferentOrdinals
```

Each name identifies one primary fixture. The variable-width fuzz
regression is a separately documented variant and does not satisfy
inventory cardinality.

## Differential evidence boundary

The authoritative projection compares:

- complete kept-window structure and discard decisions;
- complete finding count and order;
- stable fingerprint, token count, and line count;
- complete occurrence count and order;
- occurrence path, token positions, and line range;
- sorted ownership diagnostics;
- error presence and type-chain classification.

The differential corpus proves canonical internal structural equality.
Existing renderer contract tests independently prove deterministic text
and JSON projection.

No corpus-level byte identity is claimed because production text and JSON
renderers are not directly callable at the selected internal seam without
constructing a duplicate orchestration path.

## Frozen self-hosted target

This ACT must not modify:

```text
cmd/leamas/claim_commands.go
cmd/leamas/evidence_commands.go
internal/factory/dupcode/baseline.json
```

The required live result remains:

```text
TokenCount = 504
claim_commands.go    = lines 268–340
evidence_commands.go = lines 310–382
```

## Final evidence and race closure correction

The final closure correction keeps the guarded V4 implementation, corpus,
fuzz wire format, benchmark fixtures, baseline, and 504-token remediation
target unchanged.

The literal 30-minute full-package race command timed out after 1800.456s.
`TestDebugBaselines` completed its live-tree scan before the package timeout;
the timeout dump named
`TestV4BaselineDelta_LiveTreeMatchesCommittedBaseline` as the executing test.
No race diagnostic occurred. The accepted one-test bounded alternative then
ran every package test except exactly `TestDebugBaselines` under `-race` and
passed in 1666.860s. The excluded test passed separately without `-race`, and
the focused `TestV4Alignment_` race proof also passed.

Invalidating the stale gate summary exposed a separate closure-evidence defect:
`make gate` passed but did not recreate `.factory/gate-summary.json`. The
closure patch makes the literal gate write one aggregate, observed gate-result
check after execution. This avoids recursive gate invocation while providing a
fresh RFC3339 timestamp and canonical `pass` / `fail` status. Focused tests pin
pass, fail, timestamp, duration, source-presence, unavailable-count, and
summary-write-failure behavior.

## Immediate successor

Only after this ACT is committed, verified, and bound by detached final
evidence may
`ACT-LEAMAS-FACTORY-DUPCODE-SELF-HOSTED-REMEDIATION01` begin.
