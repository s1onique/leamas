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

## Status classification

`CHANGESET_MANIFEST` and `CHANGESET_STATS` always reflect Git's
authoritative status for each path. The digest does not infer the
classification from boolean presence flags.

### Staged mode (`--staged`)

For staged changes the manifest status is sourced directly from
`git diff --cached --name-status -z --find-renames --find-copies
<base> --` where `<base>` is `HEAD` for normal repositories and the
empty-tree SHA otherwise. The result therefore agrees path-for-path
with `git diff --cached --name-status HEAD --` in normal
repositories.

### Dirty mode (`--dirty`)

For dirty mode the tracked-path status is the *net* change relative
to `HEAD`, obtained from
`git diff --name-status -z --find-renames --find-copies HEAD --`.
Untracked files come from `git ls-files --others`. A staged rename
followed by an unstaged edit at the destination still renders as
`R old -> new` (the net change), not `M`, because the underlying
file is the renamed destination.

### Status tokens

The digest passes the full set of Git `--name-status -z` status
letters through verbatim: `A` (added), `M` (modified), `D`
(deleted), `T` (type change, e.g. regular file to symlink or
submodule), `R` (renamed), `C` (copied), `U` (unmerged),
`X` (unknown change type), `B` (pairing broken), plus `?` for
untracked files (sourced from `ls-files --others`, not from Git's
diff). Rename/copy tokens like `R100`/`C075` are normalised to
`R`/`C` (the similarity score is dropped).

`CHANGESET_STATS` tracks each kind in its own field
(`type_changed_files`, `unknown_files`, `broken_pair_files`,
etc.) so reviewers can spot type-change and unknown-status
records at a glance.

### NUL-safe path handling

All Git output is parsed from NUL-delimited streams
(`git diff --name-status -z`, `git ls-files --others -z`,
`git diff --name-only -z`). Paths containing spaces, tabs,
newlines, Unicode or leading dashes are preserved.

### Path escaping in rendered output

A filename containing a literal newline or tab would, if written
naively into the digest, split a single manifest or changed-files
record across multiple visual lines and silently corrupt the
output. Git itself uses a `-z` form precisely so callers can decide
how to render; the digest therefore writes every rendered path
through `PathEscape` before it reaches the output buffer. The
encoding covers backslash, tab, CR, LF, and every other control
byte (0x00..0x1f, 0x7f), and is symmetric: `ParseEscapedPath` inverts
the encoding exactly. Printable UTF-8 (including leading dashes,
spaces, Unicode, etc.) passes through unchanged.

This means a path such as `weird\nfile\nname.go` (an actual
filename with embedded newlines) renders as a single manifest
entry `M  weird\\nfile\\nname.go`, and reviewers can recover the
original filename by running `ParseEscapedPath` on the rendered
path. Tests covering this round-trip and the integration tests for
newline-bearing staged/range paths are part of the parser and
digest integration test matrix.

### Similarity threshold (Leamas policy)

The digest always invokes Git with **explicit
`--find-renames=30% --find-copies=30%`** on every mode that gathers
status from `git diff`. The default Git similarity threshold is 50%,
which is too strict for the common "rename then a small edit at the
destination" case (it would degrade to an `A` + `D` pair). Lowering
to 30% keeps that case classified as `R`.

Because of this Leamas policy, the authoritative oracle for digest
reconciliation tests and self-hosting proofs is the explicit
thresholded command, **not** plain `git diff --name-status`:

```bash
git diff   --cached   --name-status   -z   --find-renames=30%   --find-copies=30%   HEAD   --
```

Stated informally: the manifest status agrees with Git's
`--name-status -z` output **at the 30% threshold**, not at Git's
default. Tests and reviewers should not treat the two as
interchangeable; for example, a rename with similarity 40% reports
as `R` in the digest but as an `A` + `D` pair in plain
`git diff --name-status`.

### Staged and unstaged presence

The staged and unstaged presence flags on each file are populated
separately from `git diff --cached --name-only` and
`git diff --name-only`. They are independent of the manifest status
and describe whether staged and/or unstaged patches exist for the
same path. The diff renderer uses them to attach the right patches
even when the net change is a single letter.

## Contract Header

Every digest begins with a versioned contract header that provides metadata about the digest producer and format. See [Digest Contract](./digest-contract.md) for full documentation.

```
LEAMAS_TARGETED_DIGEST_CONTRACT_VERSION: 1
LEAMAS_VERSION: <version>
LEAMAS_COMMIT: <commit>
LEAMAS_BUILD_TIME: <build_time>
DIGEST_MODE: <dirty|staged|range|auto>
DIGEST_CREATED_AT: <UTC RFC3339 timestamp>
```

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
