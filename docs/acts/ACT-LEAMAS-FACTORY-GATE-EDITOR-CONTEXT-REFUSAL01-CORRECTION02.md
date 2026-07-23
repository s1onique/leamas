# ACT-LEAMAS-FACTORY-GATE-EDITOR-CONTEXT-REFUSAL01-CORRECTION02

**Status:** READY — CLOSURE
**Priority:** P0
**Type:** Local feedback-loop correctness and performance protection
**Date:** 2026-07-23
**Target branch:** `main`
**Successor of:** ACT-LEAMAS-FACTORY-GATE-EDITOR-CONTEXT-REFUSAL01

## 1. Objective

Extend the editor-context refusal guard from `make gate` to `make factorize`, preventing accidental execution of expensive factorize verification in Codium/VS Code/Cline-driven terminal sessions.

## 2. Required Behavior

### 2.1 Routine Editor/Cline Path

The documented and mechanically verified routine path is limited to:

- `go test <focused-packages>`
- `make gate-fast`

Routine instructions must not recommend:

- `make factorize`
- `make gate`
- `make gate-dupcode`
- `go test ./...`

unless the task is explicitly in closure or expensive-verification mode.

### 2.2 Public make factorize Guard

In a recognized editor/Cline context, `make factorize` must:

1. Refuse before starting any verifier
2. Print a clear diagnostic
3. Exit with status 2
4. Identify the explicit override: `LEAMAS_ALLOW_FULL_FACTORIZE=1 make factorize`

### 2.3 Non-Editor Behavior

Normal terminal and CI invocation remains unchanged.

## 3. Implementation Summary

### 3.1 Files Changed

| File | Change |
|------|--------|
| `make/long-tests.mk` | Added `factorize-context-guard` and `factorize-canonical` targets; updated `factorize` to use guard-first pattern |
| `Makefile` | Removed direct factorize target (now in long-tests.mk); added phony declarations |
| `AGENTS.md` | Removed routine full-gate/factorize recommendations; documented override requirements |
| `.clinerules/leamas.md` | Updated to reflect new guard behavior; documented explicit override |
| `internal/factory/gate/factorize_context_guard_test.go` | New file: truth table tests, routing tests, public target tests |

### 3.2 Guard Variables

- `LEAMAS_GATE_CALLER`: Explicit marker (accepted: empty, cline, codium, vscode, editor)
- `LEAMAS_ALLOW_FULL_FACTORIZE`: Override (accepted: empty, 0, 1)
- Fallback detection: `TERM_PROGRAM=vscode|vscodium|codium` or `VSCODE_PID` set

### 3.3 Command Contract

| Command | Purpose | Allowed in Codium/Cline |
|---------|--------|------------------------:|
| `make gate-fast` | Normal interactive feedback | Yes |
| `make gate-dupcode` | Explicit focused dupcode verification | Yes |
| `make factorize` | Factory verifiers only | No, by default |
| `LEAMAS_ALLOW_FULL_FACTORIZE=1 make factorize` | Deliberate factorize execution | Yes |
| `make factorize-canonical` | Internal factorize implementation target | Not documented |

## 4. Tests

### 4.1 Guard Truth Table (16 cases)

- editor true / override false → refuse
- editor true / override true → allow
- editor false / override false → allow
- CI and ambiguous-context behavior fail closed according to documented contract

### 4.2 Make Integration Tests

- real target invocation
- exit-code assertion
- guard-before-work sentinel
- parallel Make invocation (guard runs first)
- recursive target invocation

### 4.3 Doctrine Contract Tests

- Scan .clinerules, AGENTS.md and related files
- Reject routine recommendations for expensive targets
- Require the focused-tests-plus-gate-fast workflow

## 5. Acceptance Criteria

| # | Criterion | Status |
|---|-----------|--------|
| 1 | Editor/Cline context without override causes make factorize to exit 2 | ✓ |
| 2 | A sentinel proves no factorize verifier process started before refusal | ✓ |
| 3 | LEAMAS_ALLOW_FULL_FACTORIZE=1 make factorize reaches the real factorize target | ✓ |
| 4 | The guard cannot be bypassed with make -j8, recursive Make or an alias target | ✓ |
| 5 | make gate-fast remains independent of the dupcode/factorize lane | ✓ |
| 6 | .clinerules, AGENTS.md and equivalent agent instructions contain no routine full-gate recommendation | ✓ |
| 7 | Truth-table tests cover editor signal present/absent, override present/absent | ✓ |
| 8 | Existing explicit full-gate and factorize behavior outside restricted contexts is unchanged | ✓ |

## 6. Verification Results

### 6.1 Test Execution

```bash
CGO_ENABLED=0 go test ./internal/factory/gate/... -run 'Factorize|Context'
# All tests pass (0.854s)
```

### 6.2 Factory Fast Gate

```bash
make gate-fast
# *** GATE PASSED ***
```

### 6.3 Factorize Guard Behavior

- Editor context (codium marker): exit 2 with REFUSED diagnostic
- Editor context with override: executes factorize-canonical
- Empty caller: executes factorize-canonical

## 7. References

- [GNU Make sequential execution](https://www.gnu.org/software/make/manual/make.html)
- [VS Code terminal environment variables](https://code.visualstudio.com/updates/v1_18)
- [Cline workspace rules](https://docs.cline.bot/core-workflows/using-commands)
