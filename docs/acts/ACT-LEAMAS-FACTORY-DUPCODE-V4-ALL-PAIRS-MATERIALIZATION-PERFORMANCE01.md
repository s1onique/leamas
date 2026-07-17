# ACT-LEAMAS-FACTORY-DUPCODE-V4-ALL-PAIRS-MATERIALIZATION-PERFORMANCE01

## Status

PASSED — final guarded algorithm reconciled by
`ACT-LEAMAS-FACTORY-DUPCODE-V4-ALL-PAIRS-MATERIALIZATION-PERFORMANCE01-CORRECTION02-CORPUS-AND-EVIDENCE01`.

## Final algorithm

V4 candidate materialization groups owned raw windows by explicit syntax
region. For each canonical region pair it applies this guarded policy:

```text
aligned cross-region sequences:
  O(N) diagonal fast path per region pair

unaligned cross-region sequences:
  O(N_left × N_right) conservative all-pairs fallback

within-region pairing:
  separately bounded or quadratic according to occurrence geometry
```

The alignment guard requires:

1. equal occurrence counts;
2. equal base-window token widths;
3. equal per-index `StartPos` deltas from each side's base window;
4. equal per-index `EndPos` deltas from each side's base window.

If any condition is not proved, the all-pairs fallback is mandatory. The
historical unconditional same-index diagonal plan is superseded and is
not a supported implementation.

## Materialization boundary

1. `findCommonWindows` buckets raw windows by normalized token
   fingerprint.
2. `generateRegionAnnotatedMatches` resolves explicit token ownership,
   groups windows by `(Path, Ordinal)`, emits within-region candidates,
   and applies the guarded cross-region policy.
3. `v4BuildRegionBoundedChainInputs` partitions candidates by canonical
   `(LeftRegion, RightRegion, Offset)` and passes deterministic groups to
   chain construction.

Windows with no file analysis, windows outside every declared region,
and windows crossing ownership boundaries are discarded. They are never
silently assigned to ordinal 0.

## Correction lineage

- **Parent implementation:** introduced the performance fixtures and an
  unconditional O(N) same-index diagonal. That semantic plan was invalid
  for asymmetric region occurrence sequences.
- **CORRECTION01:** added `regionsArePositionallyAligned` and the
  conservative all-pairs fallback. The production guard was valid, but
  its asymmetric test fixture accidentally used one path on both sides
  and therefore did not exercise a cross-region pair.
- **CORRECTION02-R1-CROSS-REGION-PROOF01:** repaired the fixture to use
  `alpha.go` and `beta.go`, proved the fallback candidate geometry, and
  captured unconditional-diagonal mutation failure.
- **CORRECTION02-CORPUS-AND-EVIDENCE01:** supplied the 17-dimension
  declared-region corpus, complete structural comparator, ownership and
  shuffle proofs, persistent fuzz regressions, performance confirmation,
  whitespace repair, lifecycle reconciliation, and final evidence. Its
  fuzz RED also tightened the guard so cross-side base windows must have
  equal token widths before the diagonal is permitted.

Historical benchmark files remain evidence of the measured runs. Their
superseded semantic explanations are not the current algorithm contract.

## Canonical frozen duplicate

The self-hosted remediation input remains frozen:

```text
TokenCount = 504
cmd/leamas/claim_commands.go    = lines 268–340
cmd/leamas/evidence_commands.go = lines 310–382
```

`cmd/leamas/claim_commands.go`, `cmd/leamas/evidence_commands.go`, and
`internal/factory/dupcode/baseline.json` must not be changed by this ACT.

## Rendering evidence boundary

The differential corpus proves canonical internal structural equality.
Existing renderer contract tests independently prove deterministic text
and JSON projection.

The corpus does not claim byte equality because production text and JSON
renderers are not directly callable at the selected internal seam without
constructing a second orchestration path.

## Immediate successor

After final verification and detached evidence are complete,
`ACT-LEAMAS-FACTORY-DUPCODE-SELF-HOSTED-REMEDIATION01` may begin.
