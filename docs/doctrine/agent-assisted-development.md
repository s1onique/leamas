# Doctrine: Agent-Assisted Development

LLMs and agents are useful Factory participants, but they are not trusted authorities.

## Core Principle

Agents may propose, edit, and review, but every claim must be grounded in evidence.

## Tooling Language Rules for Agents

Agents must not create Python files.

Agents must not create long Bash scripts.

If an automation task is too large for a tiny Bash wrapper, agents must implement it in Go or ask for an ACT/ADR update.

## Agent Contract

### Always

- Read the active ACT and relevant doctrine before changing files.
- Keep changes inside the ACT scope.
- Separate observed facts from assumptions and recommendations.
- Preserve local-first, web-first, Go-only, and single-binary constraints.
- Report verification honestly, including skipped or deferred checks.
- Prefer small corrective R1/R2 patches when review finds drift.
- Link decisions to ADRs or propose ADR updates when behavior changes.
- Prefer Go for substantial automation.
- Keep Bash wrappers small and boring.
- Treat Python as forbidden unless a future ADR changes doctrine.

### Never

- Invent files, tests, command outputs, commits, or verification results.
- Claim a gate passed unless it was actually run and passed.
- Add OAuth/OIDC/RBAC/database/gateway behavior by default.
- Weaken a verifier or quality gate to make a patch pass.
- Convert Leamas into a generic LLM gateway, provider router, or control plane.
- Hide uncertainty or silently broaden the ACT.
- Create `*.py` files.
- Create new Bash scripts over 50 meaningful LOC.
- Hide implementation logic in shell helpers.

### Ask / Escalate

- If the ACT conflicts with doctrine.
- If a requested change requires weakening a gate.
- If a feature appears to require non-local, multi-user, or hosted behavior.
- If evidence is missing for a closure claim.
- If sensitive data capture/redaction policy is unclear.

### Verification Hooks

- `scripts/verify_doctrine_agent_contracts.sh`
- `scripts/verify_forbidden_patterns.sh`
- `scripts/verify_factory_docs.sh`
- close reports
- future claim-grounding verifier
