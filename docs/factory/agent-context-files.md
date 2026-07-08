# Agent Context Files

Leamas uses checked-in agent context files so coding agents receive consistent project instructions.

## Files

| File | Purpose |
|------|---------|
| `AGENTS.md` | Tool-agnostic repository instructions for coding agents |
| `.clinerules/leamas.md` | Cline-specific persistent project rules |

## Source of Truth

Doctrine files under `docs/doctrine/` are the source of truth.

`AGENTS.md` and `.clinerules/leamas.md` must summarize and point to doctrine. They must not contradict doctrine.

## Maintenance Rules

- Keep context files short.
- Link to doctrine instead of copying long doctrine text.
- Do not include secrets or local machine paths.
- Do not encode temporary task state.
- Do not weaken gates from agent context files.
