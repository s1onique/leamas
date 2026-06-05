# Templates

Document templates for Leamas workflow.

## Available Templates

| Template | Purpose |
|----------|---------|
| [epic.md](epic.md) | High-level feature or capability description |
| [act.md](act.md) | Specific implementation task derived from an epic |
| [close-report.md](close-report.md) | Summary report when closing an act or epic |
| [reviewer-prompt.md](reviewer-prompt.md) | Prompt guidance for code reviewers |

## Usage

### Epic

For large features or capabilities that span multiple implementation tasks.

```bash
cp docs/templates/epic.md docs/epics/YOUR-EPIC-NAME.md
```

### Act

For specific, bounded implementation tasks.

```bash
cp docs/templates/act.md docs/acts/YOUR-ACT-NAME.md
```

### Close Report

When completing an epic or act, document the outcome.

```bash
cp docs/templates/close-report.md docs/close-reports/YOUR-REPORT.md
```

### Reviewer Prompt

Use the reviewer prompt template to provide context for code reviewers.

## Relationship

```
Epic (feature)
  └── Act 1 (implementation task)
  └── Act 2 (implementation task)
  └── ...
  └── Close Report (outcome)
```

## Notes

- Templates are copied, not referenced (Markdown doesn't support includes)
- Keep templates focused; detailed guidance belongs in doctrine
- Templates may evolve as the project matures
