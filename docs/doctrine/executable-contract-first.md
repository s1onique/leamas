# Doctrine: Executable Contract First

## Purpose

Establish **Executable Contract First** as the mandatory default workflow for behavior-changing work in Leamas.

Before production behavior is implemented or modified, the developer or coding agent must:

1. Identify the narrowest stable behavioral boundary.
2. Design an orthogonal, declarative test matrix.
3. Implement the relevant tests.
4. Run them and establish a meaningful RED state.
5. Implement the smallest coherent production change.
6. Establish focused GREEN.
7. Run affected subsystem tests and the repository gate.
8. Refactor only while the executable contract remains green.

## Applicability

This doctrine applies to all **behavior-changing tasks**, defined as changes that modify any externally or internally observable contract, including:

- CLI output or exit behavior
- Package API behavior
- Parsing or serialization
- State transitions
- Ordering or determinism
- Error classification
- Resource limits
- Timeout or cancellation behavior
- Process execution behavior
- Security policy
- Generated artifacts
- Configuration interpretation
- Verifier acceptance or rejection
- Compatibility behavior

A change does not need to alter a public Go symbol to be behavior-changing.

## Required Sequence

For behavior-changing work:

1. Inspect the current implementation, contract, and tests.
2. Identify the stable boundary.
3. Write the behavioral matrix.
4. Implement the relevant tests.
5. Run the focused tests.
6. Confirm RED for the intended reason.
7. Implement the smallest coherent production change.
8. Run the focused tests and confirm GREEN.
9. Run affected package or subsystem tests.
10. Run `make factorize`.
11. Run `make gate`.
12. Refactor only under passing tests.
13. Record evidence in the close report.

## Stable Boundary Guidance

The contract is written at the narrowest stable interface whose behavior should remain valid when implementation details are refactored.

| Change Type | Preferred stable boundary |
|------------|--------------------------|
| Duration formatting | Pure formatter |
| CLI result line | Command output contract |
| Config validation | Parser or validation API |
| Gate execution | Toolchain execution boundary |
| Markdown verifier | Verifier input/output contract |
| Helm configuration | Rendered manifest contract |
| Process timeout | Executor boundary and process-tree outcome |

Tests should primarily exercise observable behavior through stable interfaces rather than coupling themselves to implementation structure.

## Test Matrix Guidance

The following dimensions guide test case selection. Select relevant dimensions based on the specific change; do not mechanically generate all possible combinations.

| Dimension | Representative cases |
|-----------|---------------------|
| **Nominal** | Valid input and expected output |
| **Boundary** | Empty, zero, minimum, maximum, exact threshold |
| **Invalid** | Malformed, unsupported, contradictory |
| **Failure** | Dependency error, permission error, timeout |
| **State** | Initial, repeated, already completed, rejected |
| **Determinism** | Stable ordering, serialization, diagnostics |
| **Bounds** | Count, size, time, memory, concurrency |
| **Cancellation** | Before start, in flight, after completion |
| **Compatibility** | Existing format, migrated format, unknown fields |
| **Security** | Untrusted input, secret handling, fail-closed behavior |
| **Interaction** | Two dimensions whose combination changes semantics |

Orthogonality requires interaction cases to be explicit rather than accidentally combining unrelated dimensions.

## Test Quality Requirements

Tests must:

- Assert observable behavior
- Have clear names
- Expose meaningful inputs and expectations
- Be deterministic
- Be hermetic unless an integration boundary explicitly requires otherwise
- Avoid arbitrary sleeps
- Avoid uncontrolled wall-clock time
- Avoid ambient environment dependence
- Avoid uncontrolled randomness
- Avoid real external network services
- Avoid depending on test execution order
- Produce actionable failure output
- Preserve discovered defects as regression cases

Where time is part of the contract, inject or virtualize it where practical.

Where I/O is part of the contract, prefer an injected capability, local fake, temporary filesystem, or controlled subprocess seam.

Where concurrency is part of the contract, test the observable synchronization or lifecycle property rather than relying on timing luck.

### Public Behavior Over Private Structure

Tests should not normally assert:

- Private helper calls
- Exact internal call ordering
- Incidental allocation strategy
- Internal data structure choices
- Private method existence
- Implementation-only log lines
- Mocks that duplicate the implementation algorithm

Interaction-heavy mocks commonly couple tests to internal implementation and increase maintenance cost.

### Declarative Test Structure

Prefer table-driven tests where cases share execution logic:

```go
tests := []struct {
    name    string
    input   Input
    want    Output
    wantErr error
}{
    // ...
}
```

The reader should not need to trace substantial fixture logic to understand what each case proves.

### Test Expectation Changes

After production implementation begins, a new test expectation may be changed only when:

1. The expectation contradicts the accepted contract.
2. The test is technically incorrect.
3. The test is nondeterministic.
4. Implementation work exposes a genuine contract ambiguity requiring a recorded decision.

The reason must be recorded before or alongside the expectation change.

It is prohibited to silently change an expected result solely because the implementation produced something different.

## RED Requirements

A useful RED result demonstrates the behavioral gap. RED is a focused test run where at least one new or changed test fails because the required behavior is absent or incorrect.

The following do not normally constitute sufficient RED evidence:

- Syntax errors
- A missing import
- Broken test setup
- An unrelated existing failure
- A deliberately impossible assertion
- A missing API where the behavior could reasonably have been tested through an existing seam

A compile failure may be accepted when introducing a genuinely new typed contract and no meaningful behavioral seam exists yet. The ACT must state why.

## GREEN Requirements

GREEN is the new executable contract passing after the smallest coherent production implementation has been added.

GREEN is not completion by itself. Affected subsystem tests and the repository gate must also pass.

## Refactoring Rules

Refactor only while the executable contract remains green. The following are permitted under passing tests and do not require a new RED/GREEN cycle:

- Internal structure reorganization
- Helper extraction
- Naming improvements
- Documentation additions
- Performance optimizations that do not alter observable contracts

Dependency updates are NOT pure refactors. They are excluded from this list
because they can change transitive behavior (resolved versions, transitive
deps, build flags, platform-specific code paths). Dependency updates require
their own RED/GREEN cycle unless the update is a no-op metadata refresh with
no transitive impact, in which case the exception rules below apply.

The following require a new RED before GREEN cycle:

- Behavioral contract changes
- API signature changes
- CLI output format changes
- Error behavior changes
- Dependency updates with potential transitive impact

## Permitted Exceptions

The RED requirement may be marked not applicable for:

- Documentation-only changes
- Comment-only changes
- Formatting-only changes
- Generated-file refreshes with no behavioral change
- Dependency metadata refreshes with no behavioral change
- Pure mechanical refactors already protected by sufficient executable contracts
- Deletion of unreachable dead code with existing proof
- Emergency remediation where reproducing the defect would create unacceptable immediate risk

Every exception must include:

- Exception category
- Justification
- Verification performed instead
- Explanation of why no behavioral regression test was practical

Defect fixes are not generally exempt. A regression test should reproduce the defect first whenever practical.

## Evidence Requirements

Every closed ACT must record:

- Command and failing test name for RED
- Expected behavior and observed failure reason for RED
- Focused command and result for GREEN
- Affected subsystem command and result for GREEN
- Repository gate command and result for GREEN

## Examples for Leamas-Specific Change Types

### Duration Formatting

**Boundary:** Pure formatter function
**Test Matrix:**

| Case | Input | Expected |
|------|-------|----------|
| Zero | 0 | "0s" |
| Seconds | 30 | "30s" |
| Minutes | 90 | "1m30s" |
| Hours | 3661 | "1h1m1s" |
| Negative | -1 | error |

### CLI Result Line

**Boundary:** Command output contract
**Test Matrix:**

| Case | Given | When | Expected |
|------|-------|------|----------|
| Success | - | exit 0 | stdout contains "OK" |
| Failure | - | exit 1 | stderr contains error |
| Timeout | - | timeout | exit code matches project timeout policy |

Note: exit code 124 is the convention used by GNU `timeout(1)` and many
POSIX shells, but it is not a universal Leamas contract. Each verifier or
executor that introduces a timeout must define and assert its own exit
contract; tests must assert THAT contract, not blindly assume 124.

### Process Timeout

**Boundary:** Executor boundary and process-tree outcome
**Test Matrix:**

| Case | Given | When | Expected |
|------|-------|------|----------|
| Normal | valid cmd | completes | exit 0 |
| Timeout | long cmd | timeout 1s | exit code per executor contract, no zombie |
| Kill | hanging | timeout | process tree cleaned |

## Anti-Patterns

The following patterns violate this doctrine:

1. **Writing tests against private implementation details** — Tests must exercise observable behavior through stable interfaces.

2. **Excessive mocks and call-sequence assertions** — Prefer injected capabilities or simple fakes over interaction-heavy mocks.

3. **Large overlapping test matrices** — Select orthogonal dimensions; explicit interaction cases only.

4. **Tests that fail only because an API does not compile yet** — Use existing seams where possible; document if genuinely new contract.

5. **Expectations silently rewritten to fit the implementation** — Record the reason for any expectation change.

6. **Ceremonial test-first compliance** — Require stable-boundary identification, a behavioral matrix, and an intended-reason RED explanation.

## Agent Contract

### Always

- Follow the required sequence for behavior-changing work.
- Identify the stable boundary before writing tests.
- Design orthogonal, declarative test matrices.
- Establish RED for the intended behavioral reason before production code.
- Record RED and GREEN evidence in the close report.
- Refactor only while tests remain green.

### Never

- Weaken a correct test to make an implementation pass.
- Claim verification passed unless tests actually ran and passed.
- Skip tests when "it's obvious" the implementation is correct.
- Write tests that couple to private implementation details.
- Use mocks that duplicate the implementation algorithm.

### Ask / Escalate

- If the change scope makes orthogonal testing impractical.
- If the stable boundary is unclear.
- If a test expectation needs to change after implementation begins.

### Verification Hooks

- `go test ./path/...` — Focused tests for the affected package or
  subsystem. Run this first after the implementation change.
- `go test ./...` — Full test sweep, used to confirm no unrelated
  subsystem regressions. Run this before `make gate`.
- `make factorize` — Repository structure verification (Factory
  policy verifiers only; no Go toolchain checks).
- `make gate` — Full quality gate (Factory verifiers plus Go toolchain:
  `go mod tidy`, `gofmt`, `go vet ./...`, `go test ./...`, static build).
