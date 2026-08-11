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
- Bash only for tiny glue and Git hooks.
- New executable Bash scripts must be ≤50 meaningful LOC.
- All verifiers must be in Go. Bash verifier scripts are forbidden.
- Bash `scripts/verify_*.sh` files are compatibility wrappers only.

## LLM-Friendliness

- Keep files small and reviewable.
- Do not add minified committed assets.
- Do not add allowlists or bypasses to the LLM-friendliness gate.
- Split large files instead of weakening thresholds.

## Execution Authority

The current ACT is authoritative.

Never run `make factorize`, `make gate-dupcode`, or `make gate` in Cline/editor context unless the current ACT explicitly authorizes that exact command.

When not authorized, report it as NOT RUN.

Do not infer Git commit, push, tag, or history-rewrite authority from permission to edit or test.

Cline checkpoints are not Git commits and do not grant Git publication authority.

Successful tests do not imply commit authority. Commit only when the ACT delegates commit authority.

## Verification

During ordinary implementation and correction loops, run `CGO_ENABLED=0 make gate-fast`.

A refusal from `make factorize`, `make gate-dupcode`, or `make gate` is NOT RUN, not PASS, and must never be reported as successful verification.

## Git Safety

Do not force-push. Prefer forward corrective commits.

## Closure Protocol v1

New ACTs MUST use Closure Protocol v1 via
`leamas factory close plan|run|verify|render|tag|status`. The
authoritative verification record is the compact manifest at
`docs/closure-manifests/<ACT-ID>.json`. Never embed future closure
identities or raw evidence in committed documents and never move or
force-push ACT tags.