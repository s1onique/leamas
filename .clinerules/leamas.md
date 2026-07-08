# Cline Rules for Leamas

Follow `AGENTS.md` first.

Leamas uses Factory discipline. Doctrine lives under `docs/doctrine/`.

## Required Behavior

- Read `AGENTS.md` before editing.
- Keep patches scoped to the active ACT.
- Do not invent command outputs, files, tests, commits, or verification results.
- Report uncertainty.
- Prefer small R1/R2 cleanup patches over broad rewrites.

## Language Boundary

- No Python anywhere.
- Go for product code, labs, verifiers, digest tools, and substantial automation.
- Bash only for tiny glue.
- New executable Bash scripts must be ≤50 meaningful LOC.

## LLM-Friendliness

- Keep files small and reviewable.
- Do not add minified committed assets.
- Do not add allowlists or bypasses to the LLM-friendliness gate.
- Split large files instead of weakening thresholds.

## Verification

Run:

```bash
make factorize
make gate
```

Do not claim success for checks that were skipped, deferred, or not run.

## Git Safety

Do not force-push. Prefer forward corrective commits.
