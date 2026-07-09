# DEPENDENCY_DELTA Section

The `DEPENDENCY_DELTA` section answers: **"Did this patch change project dependencies or toolchain constraints?"**

This section provides a deterministic delta of Go module changes, enabling reviewers to quickly identify dependency updates that may require careful attention.

## Placement in Digest

In digest v2, sections appear in this order:

```
...
## PUBLIC_SURFACE_DELTA
...
## DEPENDENCY_DELTA  ← NEW
...
## Changed files
...
```

## Format

```markdown
## DEPENDENCY_DELTA
ecosystem=go
source_status=<status>
go_mod_changed=<bool>
go_sum_changed=<bool>
module_path_changed=<bool>
go_version_changed=<bool>
toolchain_changed=<bool>
requires_added=<count>
requires_removed=<count>
requires_modified=<count>
replaces_added=<count>
replaces_removed=<count>
replaces_modified=<count>

module:
  before=<module path>
  after=<module path>

go_version:
  before=<version>
  after=<version>

toolchain:
  before=<toolchain>
  after=<toolchain>

requires_added:
  - <module> <version>
  - ...

requires_removed:
  - <module> <version>
  - ...

requires_modified:
  - <module> <old_version> -> <new_version>
  - ...

replaces_added:
  - <module> <version>
  - ...

replaces_removed:
  - <module> <version>
  - ...

replaces_modified:
  - <module> <old_version> -> <new_version>
  - ...
```

## Fields

| Field | Description |
|-------|-------------|
| `ecosystem` | Always "go" for this section |
| `source_status` | "present" when go.mod/go.sum changed, "absent" otherwise |
| `go_mod_changed` | True if any go.mod directive changed |
| `go_sum_changed` | True if go.sum entries added/removed |
| `module_path_changed` | True if module directive changed |
| `go_version_changed` | True if go directive changed |
| `toolchain_changed` | True if toolchain directive changed |
| `requires_added/removed/modified` | Count of require directive changes |
| `replaces_added/removed/modified` | Count of replace directive changes |

## Tracked Changes

### Module Directives
- `module` - The module path
- `go` - Go version requirement
- `toolchain` - Toolchain version

### Require Directives
- New dependencies added
- Dependencies removed
- Version upgrades/downgrades

### Replace Directives
- New replace directives
- Replace directives removed
- Replace target changes

### go.sum Changes
- New `module@version hash` entries
- Entries removed (tracked via requires_removed)

## Excluded Checks

The following are explicitly **not** implemented:

- Vulnerability scanning
- License compliance checks
- SBOM generation
- Network lookups
- Transitive dependency analysis
- Module downloads

## Implementation

Uses `golang.org/x/mod/modfile` for Go module parsing.

## Evidence Hash

The section content is included in `EVIDENCE_HASHES` for reproducibility verification:

```markdown
dependency_delta_sha256=<sha256 hex>
```

## Example

### No dependency changes
```markdown
## DEPENDENCY_DELTA
ecosystem=go
source_status=absent
go_mod_changed=false
go_sum_changed=false
module_path_changed=false
go_version_changed=false
toolchain_changed=false
requires_added=0
requires_removed=0
requires_modified=0
replaces_added=0
replaces_removed=0
replaces_modified=0

module:
  before=
  after=

go_version:
  before=
  after=

toolchain:
  before=
  after=

requires_added:
  - none

requires_removed:
  - none

requires_modified:
  - none

replaces_added:
  - none

replaces_removed:
  - none

replaces_modified:
  - none
```

### With dependency changes
```markdown
## DEPENDENCY_DELTA
ecosystem=go
source_status=present
go_mod_changed=true
go_sum_changed=true
module_path_changed=false
go_version_changed=true
toolchain_changed=false
requires_added=1
requires_removed=0
requires_modified=2
replaces_added=0
replaces_removed=0
replaces_modified=0

module:
  before=github.com/example/mymodule
  after=github.com/example/mymodule

go_version:
  before=1.21
  after=1.22

toolchain:
  before=
  after=

requires_added:
  - github.com/example/foo v1.2.3

requires_removed:
  - none

requires_modified:
  - golang.org/x/tools v0.33.0 -> v0.34.0
  - github.com/spf13/cobra v1.7.0 -> v1.8.0

replaces_added:
  - none

replaces_removed:
  - none

replaces_modified:
  - none
```

## Design Rationale

1. **Reviewer Focus**: Helps reviewers quickly identify dependency updates that may introduce new code paths or vulnerabilities.

2. **Deterministic**: Uses official Go module parser for reliable parsing.

3. **Minimal Overhead**: Only processes go.mod and go.sum, not entire dependency graph.

4. **Network-Free**: Does not download or analyze remote modules.

## Related Sections

- [PUBLIC_SURFACE_DELTA](./digest-public-surface-delta.md) - Precedes DEPENDENCY_DELTA
- [EVIDENCE_HASHES](./digest-evidence-hashes.md) - Includes DEPENDENCY_DELTA in hash

## Contract Version

This section is part of digest v2. It is not present in v1 digests.
