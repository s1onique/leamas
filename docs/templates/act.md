# ACT Template

> Define a small actionable, concrete, time-bound increment.

---

## Title

<!-- Short imperative title (e.g., "Add user authentication") -->

## Parent Epic

<!-- Link to parent epic or N/A if standalone -->

## Problem

<!-- Why is this ACT needed? What specific issue does it address? -->

## Goal

<!-- What will be different after this ACT is done? -->

## Scope

<!-- What is included in this ACT? Be specific. -->

## Non-goals

<!-- What is explicitly excluded? -->

## Executable contract

### Stable boundary

<What stable behavioral boundary is being changed or protected?>

### Test matrix

| Case | Dimension | Given | When | Expected |
|---|---|---|---|---|

### RED evidence

- Command:
- Expected failing case:
- Observed reason:
- Evidence:

### GREEN evidence

- Focused command:
- Affected subsystem command:
- Repository gate command:

### Exceptions

If there are no exceptions to the RED/GREEN cycle, write `None.` below and
stop. Otherwise each exception MUST include all four fields below.

| Category | Justification | Verification | Why no regression test |
|---|---|---|---|
| <!-- e.g. docs-only / formatting / metadata / dead-code / mechanical-refactor / emergency --> | <!-- Why warranted? --> | <!-- What ran instead? --> | <!-- Why not practical? --> |

## Acceptance Criteria

- [ ] <!-- Criterion 1 -->
- [ ] <!-- Criterion 2 -->
- [ ] <!-- Criterion 3 -->

## Verification Commands

<!-- Commands or steps to verify the ACT works as intended -->

```bash
# Your verification commands here
```

## Reviewer Focus

<!-- What should the reviewer pay most attention to? What is risky or non-obvious? -->

## Close Report Stub

<!-- Placeholder for close report — fill after ACT is verified -->

> **Summary:** <!-- 1-2 sentence description -->
> **Files changed:** <!-- List files -->
> **Behavior changed:** <!-- What changed from user perspective -->
> **Verification:** <!-- Commands run and results -->
> **Follow-up ACTs:** <!-- None / List -->
