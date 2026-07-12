# Leamas Version Command

The `leamas version` command prints build metadata for the leamas
binary. The schema separates the **effective SemVer** (used by the
doctrine compatibility oracle) from the **declared version** that
was passed to the build, and from the immutable **provenance**
fields (commit, build_time, dirty).

## Usage

```bash
leamas version
leamas version --json
```

## Output

### Default (line-oriented)

A development build with the default `VERSION=dev`:

```
version: 0.1.0+dev.92b459cf0806.20260712T064118Z
declared_version: dev
commit: 92b459cf0806
build_time: 2026-07-12T06:41:18Z
```

A release build with `VERSION=0.1.0`:

```
version: 0.1.0
commit: 92b459cf0806
build_time: 2026-07-12T06:41:23Z
```

A release build with a dirty tree (`vcs.modified=true`):

```
version: 0.1.0
commit: 92b459cf0806
build_time: 2026-07-12T06:41:23Z
dirty: true
```

The `declared_version:` line is emitted only when the stamp was
auto-derived; release builds where the declared value matches the
effective value omit it for a stable schema.

The `dirty:` line is emitted only when `runtime/debug.ReadBuildInfo()`
reports `vcs.modified=true`. Defaults to omitted on clean builds.

### JSON format

```bash
leamas version --json
```

```json
{
  "version": "0.1.0+dev.92b459cf0806.20260712T064118Z",
  "declared_version": "dev",
  "commit": "92b459cf0806",
  "build_time": "2026-07-12T06:41:18Z"
}
```

(`dirty` is omitted because `Info.MarshalJSON` suppresses it when it
equals `"false"`; the development build defaults to a clean tree.)

The `declared_version` and `dirty` fields are JSON-omitempty and are
omitted from the wire form when they equal the version default or
are false respectively. This means a release build on a clean
tree yields:

```json
{
  "version": "0.1.0",
  "commit": "92b459cf0806",
  "build_time": "2026-07-12T06:41:23Z"
}
```

## Build metadata fields

| Field             | Description                                                |
|-------------------|------------------------------------------------------------|
| `version`         | Effective SemVer used by the doctrine compatibility check. Auto-derived for development builds to `<SemVerCompatible>+dev.<commit>.<ts>`. |
| `declared_version`| Literal `VERSION=` value passed to the build. Only emitted (JSON: omitted) when it differs from `version`. |
| `commit`          | VCS commit hash. Always present. Default: `unknown`.        |
| `build_time`      | Build timestamp in RFC3339/UTC. Default: `unknown`.        |
| `dirty`           | VCS dirty marker. Default: `false` (omitted from wire form). Auto-populated from `runtime/debug.ReadBuildInfo()`'s `vcs.modified`. |

## Linker injection contract

The Makefile injects **four** linker variables via `-ldflags`:

```make
-X 'github.com/s1onique/leamas/internal/version.Version=…'
-X 'github.com/s1onique/leamas/internal/version.DeclaredVersion=…'
-X 'github.com/s1onique/leamas/internal/version.Commit=…'
-X 'github.com/s1onique/leamas/internal/version.BuildTime=…'
```

The `Dirty` variable is **not** injected. The authoritative value
comes from `runtime/debug.ReadBuildInfo()`'s `vcs.modified` setting,
which the modern Go toolchain populates automatically when the binary
is built with `-buildvcs=true` (the default). Capturing this at
build time instead of computing it from `git status --porcelain`
in the Makefile keeps the contract robust regardless of how the
binary is produced.

## Why separation matters

Keeping the SemVer (`version` + `declared_version`) distinct from
the provenance (`commit`, `build_time`, `dirty`) is required by
the R2 acceptance set:

- Build metadata after `+` has no precedence per SemVer 2.0.0 §10,
  so the effective comparison stays correct even when the stamp
  carries a `+dev.<commit>.<ts>` suffix.
- The `declared_version` reflects the value the operator asked
  for (the literal `VERSION=` build argument), distinct from the
  effective version (what the oracle sees).
- Commit, build_time, and dirty fields are separate so the lock
  file can record them as immutable provenance.

## Version derivation policy

For development builds, the effective SemVer follows the rule:

```text
SemVerCompatible+dev.<short-commit>.<build-timestamp>
```

where `SemVerCompatible = "0.1.0"`. When the commit or build
timestamp are unavailable (e.g. a test-only build with no
`-ldflags`), the helper degrades gracefully: the commit segment
is dropped if the input is `unknown` or empty; the timestamp
segment likewise. The base (`SemVerCompatible`) is always
present so the stamp satisfies the canonical `>=0.1.0` floor.

For release builds, both `Version` and `DeclaredVersion` are
injected with the same strict SemVer, so `EffectiveFrom()` leaves
the value unchanged.

## Strict SemVer enforcement

Both the runtime `IsValidSemVer` parser (SemVer 2.0.0 grammar
from semver.org) and the Makefile `STAMP_REGEX` guard
implement the **same** grammar:

- arbitrary text (`banana`, `1oops`)
- missing patch (`1.2`)
- leading zeros (`01.2.3`, `1.02.3`, `1.2.03`)
- numeric prerelease identifiers with leading zeros (`1.2.3-01`,
  `1.2.3-alpha.01`)
- trailing `+` (`1.2.3+`)
- extra dot components (`1.2.3.4`)
- empty prerelease/build identifiers (`1.2.3-alpha..1`,
  `1.2.3+build..42`)
- surrounding whitespace (no implicit trim; both validators reject it)

`make release VERSION=...` enforces this in addition to the
placeholder rejection (`''`, `dev`, `unknown`). Both `make
install` and `make release` chain through `make stamp-check`,
which inspects the built binary and confirms its `version:`
value matches the strict SemVer 2.0.0 grammar.

`make install` (without `VERSION=`) **is** permitted because the
runtime derivation turns the `dev` placeholder into a
SemVer-compatible effective version, which then passes
`stamp-check`. Only `make release` requires an explicit strict
SemVer.

## Release builds

For release builds, inject metadata via linker flags:

```bash
VERSION=0.1.0
COMMIT=$(git rev-parse --short=12 HEAD)
BUILD_TIME=$(date -u +%Y-%m-%dT%H:%M:%SZ)

MODULE_PATH=github.com/s1onique/leamas

CGO_ENABLED=0 go build -ldflags "\
  -X '${MODULE_PATH}/internal/version.Version=${VERSION}' \
  -X '${MODULE_PATH}/internal/version.DeclaredVersion=${VERSION}' \
  -X '${MODULE_PATH}/internal/version.Commit=${COMMIT}' \
  -X '${MODULE_PATH}/internal/version.BuildTime=${BUILD_TIME}'" \
  -o bin/leamas ./cmd/leamas
```

Or use the Makefile:

```bash
make build                          # default development stamp
make build VERSION=0.1.0            # release-style stamp
make release VERSION=0.1.0          # full release with stamp-check
make install                        # install development stamp
make install VERSION=0.1.0          # install release stamp
make stamp-check STAMP_BINARY=path/to/leamas
```

`make stamp-check STAMP_BINARY=...` audits any binary.

## Integer-overflow safety

`compareSemver` and `comparePrerelease` use **string-based**
numeric comparison (via `compareNumeric` and the `comparePrerelease`
helper). No numeric identifier is ever converted to a Go `int`, so
any SemVer that satisfies the grammar — including identifiers
larger than `math.MaxInt64` — are ordered correctly. This was the
R2.2 acceptance criterion.

## Notes

- Build time must be RFC3339/UTC format when injected.
- In development, `runtime/debug.ReadBuildInfo()` provides
  fallback VCS info for commit, build_time, and dirty.
- The command exits 0 for all valid outputs.
- Only returns error for impossible formatting failures.
