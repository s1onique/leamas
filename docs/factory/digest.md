# Factory: Targeted Digest

**ACT**: ACT-LEAMAS-FACTORY-GO-DIGEST01, ACT-LEAMAS-FACTORY-DIGEST-SMART-DEFAULTS01

## Overview

The targeted digest is a reviewable artifact of repository changes, suitable for agent-assisted review workflows. It provides a structured view of what has changed in a Git repository.

## Command

```bash
# Generate digest with smart defaults (recommended)
# - Dirty working tree → dirty digest
# - Clean working tree → previous commit digest (HEAD~1..HEAD)
leamas factory digest --output build/digest.txt

# Explicit dirty mode: includes unstaged, staged, and untracked changes
leamas factory digest --dirty --output build/digest.txt

# Explicit staged mode: includes only staged changes
leamas factory digest --staged --output build/staged-digest.txt

# Explicit range mode: include changes in revision range
leamas factory digest --range HEAD~3..HEAD --output build/range-digest.txt

# Via wrapper script
scripts/make_targeted_digest.sh --output build/digest.txt
```

## Smart Defaults

The default behavior (`leamas factory digest --output <path>`) provides the most useful digest automatically:

1. **If working tree has changes** (staged, unstaged, or untracked):
   - Generates a dirty digest
   - Includes all tracked and untracked changes

2. **If working tree is clean**:
   - Generates a commit range digest for `HEAD~1..HEAD`
   - Shows the previous committed change

This makes it suitable for both pre-commit and post-commit review workflows.

## Modes

### Auto Mode (Default)

Smart mode that automatically selects the best digest based on repository state:

- **Dirty tree**: Shows all working tree changes
- **Clean tree**: Shows `HEAD~1..HEAD` (previous commit)

Output includes resolution information:
```markdown
Mode: dirty
Resolved from: auto
Reason: working tree has changes
```

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

### Range Mode (`--range`)

Includes changes between two commits/refs:
```bash
leamas factory digest --range HEAD~1..HEAD --output build/digest.txt
leamas factory digest --range v1.0..v2.0 --output build/digest.txt
leamas factory digest --range abc123..def456 --output build/digest.txt
```

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
Mode: auto|dirty|staged|range
Range: HEAD~1..HEAD  (only for range mode)
Resolved from: auto  (only for auto mode)
Reason: <resolution reason>  (only for auto mode)

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

For range mode:
```
Makefile  [modified]
docs/foo.md  [added]
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
leamas factory digest --output build/digest.txt
```

## Use Cases

### Agent-Assisted Review

Generate a digest before requesting agent review:

```bash
leamas factory digest --output build/review-digest.txt
```

### Pre-Commit Review

Review staged changes before committing:

```bash
leamas factory digest --staged --output build/staged-digest.txt
```

### Post-Commit Review

Review the last committed change after pushing to a clean working tree:

```bash
git status  # should show "working tree clean"
leamas factory digest --output build/last-commit-digest.txt
```

### CI/CD Integration

Capture repository state at build time:

```bash
leamas factory digest --output build/artifacts/digest.txt
```

## Related

- [Agent-Assisted Development](../doctrine/agent-assisted-development.md)
- [Factory Meta Loop](../doctrine/factory-meta-loop.md)
- [Tooling Boundaries](./tooling-boundaries.md)
