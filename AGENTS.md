# AGENTS.md

## Project

Leamas is a local-first, web-first, Go-only, single-binary verification witness for AI-assisted development loops.

## Read First

Before changing files, read:

- `docs/doctrine/agent-assisted-development.md`
- `docs/doctrine/go-only.md`
- `docs/doctrine/not-a-gateway.md`
- `docs/doctrine/verification-witness.md`
- `docs/doctrine/factory-meta-loop.md`
- `docs/factory/llm-friendliness.md`
- `docs/factory/tooling-boundaries.md`

## Non-Negotiable Rules

- No Python anywhere.
- Bash is glue only.
- New executable Bash scripts must stay at or below 50 meaningful LOC.
- Substantial automation belongs in Go.
- Keep files LLM-friendly: small, focused, readable, and non-minified.
- Do not add allowlists, bypasses, or exception lists to the LLM-friendliness gate.
- Do not add OAuth/OIDC/RBAC/database/gateway behavior by default.
- Leamas may implement a local witness proxy for capture/evidence, but it is not a provider router or model control plane.
- Do not claim verification passed unless it actually ran and passed.
- Do not force-push or suggest force-pushing as normal Factory workflow.

## Required Verification

Before closing an ACT, run:

```bash
make factorize
make gate
```

When Go code exists or changes, also run:

```bash
go test ./...
go vet ./...
CGO_ENABLED=0 go build -trimpath -o bin/leamas ./cmd/leamas
```

## Verifiers Are Go

All verifiers must be implemented in Go. Bash verifier scripts are forbidden.

- Use `leamas factory verify` for all verification.
- Bash `scripts/verify_*.sh` files are compatibility wrappers only (≤50 LOC).
- Git hooks may be Bash (they are executable programs).

## Close Reports

Every closed ACT must record:

- files changed
- behavior changed
- exact commands run
- honest results
- skipped or deferred checks
- follow-up ACTs

<!-- LEAMAS:EXECUTABLE-CONTRACT-FIRST:BEGIN -->
## Executable Contract First

For every behavior-changing task:

1. Inspect the existing behavioral contract and relevant tests.
2. Before editing production code, identify the narrowest stable boundary
   and design an orthogonal, declarative test matrix.
3. Implement the relevant tests and run them to establish RED for the
   intended behavioral reason.
4. Only then implement the smallest coherent production change.
5. Establish focused GREEN, run affected subsystem tests, and run the
   repository gate.
6. Refactor only while the executable contract remains green.

Test observable behavior rather than private implementation details.
Prefer table-driven tests where cases share execution logic. Keep tests
deterministic and explicit. Prefer injected capabilities or simple fakes
over interaction-heavy mocks. Do not weaken a correct test merely to make
an implementation pass. Document any exception to the RED requirement.
<!-- LEAMAS:EXECUTABLE-CONTRACT-FIRST:END -->

## If Doctrine Conflicts With Task

Stop and report the conflict. Do not silently override doctrine.
