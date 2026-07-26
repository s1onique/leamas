# ACT-LEAMAS-FACTORY-CLOSURE-PROTOCOL-V2-PORTABLE-RUNNER-AUTHORITY01-CORRECTION01

## Closure Report

### Subject
Closure Protocol V2 Portable Runner Authority - ACT-LEAMAS-FACTORY-CLOSURE-PROTOCOL-V2-PORTABLE-RUNNER-AUTHORITY01-CORRECTION01

### Files Changed

| File | Change |
|------|--------|
| `internal/factory/authority/capabilities.go` | Added `closure_protocol_v2_portable_runner_authority` capability |
| `internal/factory/authority/correction01_capability_test.go` | Updated for new capability |
| `internal/factory/authority/capability_test.go` | New unit tests for capability contract |
| `internal/factory/closure/plan_authority.go` | Added `ValidateRunnerAuthority` to plan validation |
| `internal/factory/closure/runner_authority.go` | Core runner authority validation logic |
| `internal/factory/closure/portable_runner_authority_test.go` | New unit tests for runner authority |
| `internal/factory/execgate/verifier.go` | Added portable_runner test file entries |
| `scripts/verify_portable_runner_test_selection.sh` | Fixed to use wc/expr instead of grep -c |
| `docs/closure-plans/ACT-LEAMAS-FACTORY-CLOSURE-PROTOCOL-V2-PORTABLE-RUNNER-AUTHORITY01-CORRECTION01.json` | Simplified to meet llm-friendly requirements |
| `.factory/required-capabilities.json` | Added new capability |

### Behavior Changed

1. **Runner Authority Validation**: `ValidateRunnerAuthority` now checks:
   - Tool release binary matches plan-pinned binary hash
   - Tool revision matches tool authority specification
   - Target HEAD and tree match runner authority specification
   - Subject exactly matches specified commit

2. **Capability**: New `closure_protocol_v2_portable_runner_authority` capability declared and tested

3. **LLM-Friendly**: Closure plan simplified to use `go test` directly instead of inline bash scripts with long lines

### Commands Run

```bash
CGO_ENABLED=0 make gate-fast
```

### Results

- llm-friendly: OK
- tooling-boundaries: OK
- exec-gate: OK
- All Go tests pass
- Static build: OK

### Skipped/Deferred

- Integration tests for external repository CLI (complex git behavior)
- Bash script test selection with grep-based verification (tooling boundary violation)

### Follow-up ACTs

None identified.

### Commit

```
226a4b2 fix(closure): S2 - llm-friendly plan, fixed script, runner authority
```
