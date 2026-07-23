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

## Verification

During ordinary implementation and correction loops, run `CGO_ENABLED=0 make gate-fast`.

When changing dupcode-related code, also run `CGO_ENABLED=0 make gate-dupcode`.

**Expensive verification** (canonical gate, factorize) is refused in
Codium/VS Code/Cline terminal contexts to prevent accidental expensive
execution during interactive development loops.

Routine instructions must not recommend:

- `make gate`
- `make factorize`
- `make gate-dupcode`
- `go test ./...`

unless the task is explicitly in closure or expensive-verification mode.

For deliberate expensive verification, use explicit overrides:

- `LEAMAS_ALLOW_FULL_GATE=1 make gate`
- `LEAMAS_ALLOW_FULL_FACTORIZE=1 make factorize`

A refusal from `make gate` or `make factorize` is not a PASS and must
never be reported as successful verification.

## Git Safety

Do not force-push. Prefer forward corrective commits.

## Closure Protocol v1

New ACTs MUST use Closure Protocol v1 via
`leamas factory close plan|run|verify|render|tag|status`. The
authoritative verification record is the compact manifest at
`docs/closure-manifests/<ACT-ID>.json`. Never embed future closure
identities or raw evidence in committed documents and never move or
force-push ACT tags.
