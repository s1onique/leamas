# AGENTS.md

## Project

Leamas is a local-first, web-first, Go-only, single-binary verification witness for AI-assisted development loops.

## Read First

Before changing files, read:

- `docs/doctrine/agent-assisted-development.md`
- `docs/doctrine/agent-authority-delegation.md`
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

## Authority Delegation

Follow the current ACT's explicit authority.

Do not infer commit authority from permission to edit or test.
Do not infer push authority from permission to commit.
Do not infer tag authority from permission to commit.

In interactive agent/editor context, do not run make factorize, make gate-dupcode, or make gate unless the current ACT explicitly authorizes that exact command.

Do not run make factorize unless explicitly authorized by the current ACT.
Do not run make gate-dupcode unless explicitly authorized by the current ACT.
Do not run make gate unless explicitly authorized by the current ACT.

Focused checks explicitly required by the ACT remain allowed.

When not authorized, report the command as not run.

A validated Factory closure plan may itself authorize execution. Authority comes from the contract, not from caller identity. Do not substitute ad-hoc interactive commands for plan authority.

Editor checkpoints, restore points, and Compare operations are not Git commits and do not grant Git publication authority.

Successful tests do not imply commit authority. Commit only when the ACT delegates commit authority.

## Required Verification

Routine implementation loop:

```bash
CGO_ENABLED=0 make gate-fast
```

When Go code exists or changes, also run:

```bash
go test ./...
go vet ./...
CGO_ENABLED=0 go build -trimpath -o bin/leamas ./cmd/leamas
```

## Closure Verification

For routine ACT closure, use the fast lane only: `CGO_ENABLED=0 make gate-fast`.

Expensive canonical gates (`make factorize`, `make gate-dupcode`, `make gate`) require explicit authorization from the active ACT. In editor/Cline context they are refused by default.

A refusal from an expensive gate is NOT RUN, not PASS, and must never be reported as successful verification.

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

New ACTs MUST use Closure Protocol v1:

- Freeze the closure plan at `docs/closure-plans/<ACT-ID>.json` before the subject commit.
- Run the frozen plan, generate a compact manifest in a detached directory, render the deterministic report, and commit both at `docs/closure-manifests/<ACT-ID>.json` and `docs/close-reports/<ACT-ID>.md`.
- Create the immutable annotated tag with `leamas factory close tag create`.
- Derive lifecycle state with `leamas factory close status`.
- Never embed future closure, tree, or tag-object identities in the committed plan, manifest, or report.
- Never embed raw command output, full targeted digests, absolute host paths, or secret environment values in the committed manifest.
- Never move or force-push ACT tags; corrections are new tags.

Legacy report-only ACTs may continue to exist for historical ACTs but are deprecated for new ACTs.

<!-- LEAMAS:EXECUTABLE-CONTRACT-FIRST:BEGIN -->
## Executable Contract First

For every behavior-changing task:

1. Inspect the existing behavioral contract and relevant tests.
2. Before editing production code, identify the narrowest stable boundary and design an orthogonal, declarative test matrix.
3. Implement the relevant tests and run them to establish RED for the intended behavioral reason.
4. Only then implement the smallest coherent production change.
5. Establish focused GREEN, run affected subsystem tests, and run the repository gate.
6. Refactor only while the executable contract remains green.

Test observable behavior rather than private implementation details.
Prefer table-driven tests where cases share execution logic.
Keep tests deterministic and explicit.
Prefer injected capabilities or simple fakes over interaction-heavy mocks.
Do not weaken a correct test merely to make an implementation pass.
Document any exception to the RED requirement.
<!-- LEAMAS:EXECUTABLE-CONTRACT-FIRST:END -->

## If Doctrine Conflicts With Task

Stop and report the conflict. Do not silently override doctrine.
