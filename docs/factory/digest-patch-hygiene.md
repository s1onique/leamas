# Factory: PATCH_HYGIENE Specification

**ACT**: ACT-LEAMAS-FACTORY-DIGEST-PATCH-HYGIENE01

## Purpose

The `PATCH_HYGIENE` section reports deterministic patch quality checks. It uses `git diff --check` to detect:

- Trailing whitespace
- Conflict markers
- Other whitespace errors reported by Git

## Status Values

| Value | Meaning |
|-------|---------|
| `pass` | No whitespace errors or conflict markers detected |
| `fail` | Whitespace errors or conflict markers detected |
| `unavailable` | `git diff --check` could not be executed |

## Behavior by Mode

| Mode | Command |
|------|---------|
| `dirty` | Runs both `git diff --check` and `git diff --cached --check`, merges results |
| `staged` | Runs `git diff --cached --check` |
| `range` | Runs `git diff --check <range>` |

### Dirty Mode

Dirty mode can include both unstaged and staged changes. If both exist, the implementation:

1. Runs `git diff --check` (for unstaged)
2. Runs `git diff --cached --check` (for staged)
3. Merges results deterministically (staged first, then unstaged)

### Range Mode

For range mode, runs `git diff --check <range>` using the same resolved range used to build the digest (e.g., `HEAD~1..HEAD`).

## Key Order (Stable)

```
git_diff_check
whitespace_errors
conflict_markers
diagnostic_lines
```

Only renders `diagnostics:` block when `diagnostic_lines > 0`.

## Diagnostics Constraints

| Constraint | Value |
|------------|-------|
| Maximum lines | 20 |
| Maximum characters per line | 240 |
| Path normalization | Repo root replaced with `<repo>` |
| Sorting | Deterministic (staged first, then unstaged) |
| ANSI codes | Stripped |

## Classification Rules

Git's diagnostic wording may vary slightly, so classification is conservative:

- **`conflict_markers`**: Diagnostics containing "conflict marker"
- **`whitespace_errors`**: All other `git diff --check` diagnostics

This avoids overfitting to exact Git wording.

## Output Examples

### Pass

```
## PATCH_HYGIENE
git_diff_check=pass
whitespace_errors=0
conflict_markers=0
diagnostic_lines=0
```

### Fail with Diagnostics

```
## PATCH_HYGIENE
git_diff_check=fail
whitespace_errors=2
conflict_markers=1
diagnostic_lines=3
diagnostics:
  - file.go:12: trailing whitespace.
  - file.go:20: leftover conflict marker
  - README.md:4: trailing whitespace.
```

### Unavailable

```
## PATCH_HYGIENE
git_diff_check=unavailable
whitespace_errors=0
conflict_markers=0
diagnostic_lines=1
diagnostics:
  - git diff --check unavailable: <error>
```

## Implementation

- **Source**: `internal/factory/digest/patch_hygiene.go`
- **Tests**: `internal/factory/digest/patch_hygiene_test.go`
- **Integration tests**: `internal/factory/digest/patch_hygiene_integration_test.go`

## Constants

```go
const (
    PatchHygienePass        = "pass"
    PatchHygieneFail        = "fail"
    PatchHygieneUnavailable = "unavailable"
)

const MaxPatchHygieneDiagnostics = 20
const MaxDiagnosticLineLength = 240
```

## Type Definition

```go
type PatchHygiene struct {
    GitDiffCheck       string
    WhitespaceErrors   int
    ConflictMarkers    int
    DiagnosticLines    int
    Diagnostics       []string
}
```

## Related

- [Digest Contract](./digest-contract.md)
- [Digest Documentation](./digest.md)
