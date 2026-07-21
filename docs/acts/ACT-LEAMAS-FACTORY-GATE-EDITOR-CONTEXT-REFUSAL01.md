# ACT-LEAMAS-FACTORY-GATE-EDITOR-CONTEXT-REFUSAL01

**Status:** PARTIAL — CORRECTION01 implementation complete; canonical closure pending
**Priority:** P0
**Type:** Local feedback-loop correctness and performance protection
**Date:** 2026-07-21
**Target branch:** `main`

## 1. Objective

Prevent accidental execution of the expensive canonical Leamas gate from VSCodium, VS Code, or Cline-driven terminal sessions.

---

## 2. Implementation Summary

### 2.1 Files Changed

| File | Change |
|------|--------|
| `make/long-tests.mk` | Split `gate` into guard-first wrapper + `gate-canonical` target |
| `Makefile` | Added new phony targets to declaration |
| `.clinerules/leamas.md` | Updated to direct routine work to `make gate-fast` |
| `AGENTS.md` | Updated to use `make gate-fast` for routine work |
| `.vscode/settings.json` | New file: workspace terminal marker `LEAMAS_GATE_CALLER=codium` |
| `internal/execution/exectest/outcome.go` | Outcome types and error types (split from exectest.go) |
| `internal/execution/exectest/bounded_output.go` | runState and boundedWriter (split from exectest.go) |
| `internal/execution/exectest/environment.go` | mergeEnv helper (split from exectest.go) |
| `internal/execution/exectest/run_make.go` | Generic `Run()` + `RunMake()` adapter (split from exectest.go) |
| `internal/execution/exectest/request.go` | Request struct and helpers (split from exectest.go) |
| `internal/factory/execgate/verifier.go` | Updated allow-list for split files |
| `internal/factory/gate/gate_routing_test.go` | Sentinel routing tests, outcome classification tests |
| `internal/factory/gate/gate_integration_test.go` | Real public target tests, deterministic environment |

### 2.2 Command Contract

| Command | Purpose | Allowed in Codium/Cline |
|--------|--------|------------------------:|
| `make gate-fast` | Normal interactive feedback | Yes |
| `make gate-dupcode` | Explicit focused dupcode verification | Yes |
| `make gate` | Canonical full repository verification | No, by default |
| `LEAMAS_ALLOW_FULL_GATE=1 make gate` | Deliberate canonical verification | Yes |
| `make gate-canonical` | Internal canonical implementation target | Not documented |

### 2.3 Guard Variables

- `LEAMAS_GATE_CALLER`: Explicit marker (accepted: empty, cline, codium, vscode, editor)
- `LEAMAS_ALLOW_FULL_GATE`: Override (accepted: empty, 0, 1)
- Fallback detection: `TERM_PROGRAM=vscode|vscodium|codium` or `VSCODE_PID` set

### 2.4 Outcome Classification

The `exectest.Run` function provides bounded, race-free outcome classification:

| Outcome | Description |
|--------|-------------|
| `OutcomeSuccess` | Command completed with exit code 0 |
| `OutcomeExitFailure` | Command exited with non-zero code |
| `OutcomeSpawnFailure` | Command failed to start |
| `OutcomeTimeout` | Context deadline exceeded |
| `OutcomeCancelled` | Context cancelled |
| `OutcomeOutputOverflow` | Output exceeded limit |
| `OutcomeWaitDelay` | Retained descriptor cleanup |
| `OutcomeExecutionError` | Unexpected execution error |

---

## 3. Tests

### 3.1 Guard truth table

16 cases covering explicit callers, fallback signals, override handling,
empty values, and invalid-value fail-closed behavior.

### 3.2 Production-equivalent routing

- editor context refuses before canonical execution
- explicit override executes canonical exactly once
- gate-fast executes without canonical or dupcode routing
- an unset caller permits canonical routing

### 3.3 Execution outcome classification

- timeout
- explicit cancellation
- genuine executable spawn failure
- stdout overflow with bounded capture
- retained-descriptor WaitDelay
- non-zero target exit

### 3.4 Real public target

`make gate` in editor context returns exit code 2, emits REFUSED and
gate-fast guidance on stderr, and emits no canonical, dupcode, or PASS marker.

---

## 4. Verification Results

### 4.1 Test Execution

```bash
go test ./internal/factory/gate/...
# All tests pass
```

### 4.2 Factory Fast Gate

```bash
make gate-fast
*** GATE PASSED ***
```

### 4.3 Canonical Gate

**Status:** Available via explicit override

```bash
LEAMAS_ALLOW_FULL_GATE=1 make gate  # Runs canonical gate
```

---

## 5. Remaining Work

Canonical closure requires:

1. Observe `LEAMAS_GATE_CALLER=codium` in an actual Cline terminal
2. Run `LEAMAS_ALLOW_FULL_GATE=1 make gate` on committed tree
3. Record tested commit and tree OIDs
4. Generate authoritative gate summary and clean-range digest
5. Commit close report

---

## 6. References

- [GNU Make sequential execution](https://www.gnu.org/software/make/manual/make.html)
- [VS Code terminal environment variables](https://code.visualstudio.com/updates/v1_18)
- [Cline workspace rules](https://docs.cline.bot/core-workflows/using-commands)
- [Go exec package](https://pkg.go.dev/os/exec)
