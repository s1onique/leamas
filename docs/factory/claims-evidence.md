# Claims and Evidence

Claims and evidence are Leamas's typed domain models for recording verification witness artifacts within run bundles.

## Purpose

Claims and evidence provide a local-first, filesystem-backed, portable, and JSON-readable structure for recording verification assertions and their supporting artifacts.

**This ACT does not evaluate claims. This ACT does not call LLMs. This ACT does not persist witness proxy traffic. This ACT does not add cockpit UI.**

## Claim Schema

```json
{
  "schema_version": "leamas.claim.v1",
  "id": "claim-gate-passed",
  "run_id": "run-20260709T071704Z-smoke01",
  "created_at": "2026-07-09T07:17:04Z",
  "updated_at": "2026-07-09T07:17:04Z",
  "statement": "The gate passed with zero failures",
  "status": "open",
  "verdict": "unreviewed",
  "evidence_ids": ["evidence-make-gate-output"],
  "notes": ""
}
```

### Claim Status Values

| Status | Description |
|--------|-------------|
| `open` | Claim requires evaluation |
| `supported` | Claim has sufficient supporting evidence |
| `rejected` | Claim has been refuted |
| `unknown` | Claim status is indeterminate |

### Verdict Values

| Verdict | Description |
|---------|-------------|
| `unreviewed` | Verdict has not been determined |
| `pass` | Evidence supports the claim |
| `fail` | Evidence contradicts the claim |
| `mixed` | Evidence is mixed |

## Evidence Schema

```json
{
  "schema_version": "leamas.evidence.v1",
  "id": "evidence-make-gate-output",
  "run_id": "run-20260709T071704Z-smoke01",
  "created_at": "2026-07-09T07:17:04Z",
  "kind": "command_output",
  "role": "primary",
  "title": "Make gate output",
  "relative_path": "verifier-results/make-gate.txt",
  "summary": "Exit code 0, no test failures",
  "metadata": {}
}
```

### Evidence Kind Values

| Kind | Description |
|------|-------------|
| `command_output` | Output from a command execution |
| `digest` | A digest file |
| `log` | A log file |
| `file` | A file artifact |
| `trace` | A trace file |
| `verifier_result` | Output from a verifier |

### Evidence Role Values

| Role | Description |
|------|-------------|
| `primary` | Primary evidence |
| `supporting` | Additional supporting evidence |
| `contradicting` | Evidence that contradicts |
| `context` | Contextual information |

## Filesystem Layout

```
.leamas/runs/<run-id>/
  metadata.json
  claims/
    <claim-id>.json
  evidence/
    <evidence-id>.json
```

## Safety Rules

### Claim/Evidence ID Contract

- Non-empty, max 128 characters
- Must start with `claim-` or `evidence-`
- Only `[a-zA-Z0-9._-]` characters
- No path separators, no traversal components

### Relative Path Contract

- Empty is allowed
- Must be local (no absolute paths)
- No traversal components

## Non-Goals

- Claim evaluation engine
- LLM scoring
- Witness proxy persistence wiring
- Database/sql, network calls

## References

- [Run Bundles](./run-bundles.md)
- [Verification Witness](../doctrine/verification-witness.md)
