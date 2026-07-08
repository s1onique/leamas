# Factory: Targeted Digest

**ACT**: ACT-LEAMAS-FACTORY-GO-DIGEST01

## Overview

The targeted digest is a reviewable artifact of repository changes, suitable for agent-assisted review workflows. It provides a structured view of what has changed in a Git repository.

## Command

```bash
# Generate digest of all changes (unstaged, staged, and untracked)
leamas factory digest --dirty --output build/digest.txt

# Generate digest of staged changes only
leamas factory digest --staged --output build/staged-digest.txt

# Via wrapper script
scripts/make_targeted_digest.sh --dirty --output build/digest.txt
```

## Modes

### Dirty Mode (`--dirty`)

Includes:
- Tracked files with unstaged changes
- Tracked files with staged changes  
- Untracked files (not ignored by Git)

### Staged Mode (`--staged`)

Includes:
- Tracked files with staged changes
- New staged files
- Deleted staged files

## Output Location

Digest output should be written to `build/` or another ignored artifact directory:

```bash
--output build/leamas-digest.txt
--output build/digest.txt
```

**Warning**: Do not commit generated digests to version control.

## Format

The digest is generated as Markdown with the following sections:

```markdown
# Targeted digest

Generated at: <RFC3339 UTC timestamp>
Repo: <absolute repo root>
Mode: dirty|staged

## Changed files
...

## Diffs
...

## Workflow anchors
No workflow anchors configured.
```

### Changed Files

Lists all changed files with metadata:

```
Makefile  [tracked, staged present: no, unstaged present: yes]
docs/foo.md  [tracked, staged present: yes, unstaged present: no]
new-file.md  [untracked, staged present: no, unstaged present: yes]
```

### Diffs

For tracked files:
- Staged diff if file has staged changes
- Unstaged diff if file has unstaged changes

For untracked files:
- Full content preview (text files)
- "(binary file)" summary (binary files)

Preview limits:
- Maximum 16 KiB per file
- Maximum 200 lines per file

## Cross-Project Usage

The digest command works from any Git repository, not just the Leamas repository. This makes it a reusable Factory primitive:

```bash
# In any Factory-managed project
leamas factory digest --dirty --output build/digest.txt
```

## Use Cases

### Agent-Assisted Review

Generate a digest before requesting agent review:

```bash
leamas factory digest --dirty --output build/review-digest.txt
```

### Pre-Commit Review

Review staged changes before committing:

```bash
leamas factory digest --staged --output build/staged-digest.txt
```

### CI/CD Integration

Capture repository state at build time:

```bash
leamas factory digest --dirty --output build/artifacts/digest.txt
```

## Related

- [Agent-Assisted Development](../doctrine/agent-assisted-development.md)
- [Factory Meta Loop](../doctrine/factory-meta-loop.md)
- [Tooling Boundaries](./tooling-boundaries.md)
