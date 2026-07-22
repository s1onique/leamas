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

Do not run `make gate` as a routine local feedback command.

`make gate` is the canonical full gate and is intentionally refused in
Codium/VS Code/Cline terminal contexts.

Run the full gate only when the ACT explicitly requires canonical closure
evidence, using:

    LEAMAS_ALLOW_FULL_GATE=1 make gate

A refusal from `make gate` is not a PASS and must never be reported as
successful verification.

**Temporary Policy (ACT-LEAMAS-FACTORY-FACTORIZE-DUPCODE-SHARED-SCAN01):**

```text
make factorize:
  NOT REQUIRED for ordinary local ACT closure
  reason: still exceeds the accepted local-feedback budget
  shared-scan duplication: resolved by SHARED-SCAN01
  required only for controlled performance, CI, or explicitly scoped evidence
```

Canonical CI or release workflows may continue to invoke `make factorize`.

## Git Safety

Do not force-push. Prefer forward corrective commits.
