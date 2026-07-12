# ACT-LEAMAS-COMPILER-VERSION-STAMPING01

## Title

Stamp developer and release Leamas binaries with SemVer-compatible
versions; separate version stamping from build provenance.

## Parent Epic

`ACT-CIRCUS-FACTORY-BOOTSTRAP01` bootstrap recovery.

## Problem

A locally installed Leamas binary currently reports
`version=dev, commit=<hash>, built=<timestamp>`. That string violates
SemVer (which requires `MAJOR.MINOR.PATCH`) and is rejected by the
doctrine compiler compatibility check:

```text
compiler version unknown; refusing to verify lock with non-empty constraint ">=0.1.0"
```

The compiler is doing the right thing: refusing to prove that
`version=dev` satisfies `>=0.1.0` weakens the doctrine-lock
compatibility guarantee. The defect is that the Leamas build pipeline
produces a binary that cannot participate in its own compatibility
protocol.

## Goal

1. Every installable or distributable Leamas binary carries a real
   SemVer version string.
2. Compatible development builds auto-derive
   `0.1.0+dev.<commit>.<timestamp>` so a bare `make build` succeeds.
3. Release builds require an explicit strict SemVer `VERSION=...`;
   `make release VERSION=dev` fails closed.
4. Version stamping and build provenance (commit, time, dirty) remain
   separate fields.
5. Compatibility behaviour is provable by tests and live build output.

## Scope

In:

- Build-time auto-stamping in the Makefile for development builds.
- Release-time guard rejecting placeholder versions.
- New `internal/version` helpers producing the effective SemVer from
  the declared version plus provenance fields.
- Wiring of every `factory doctrine` command to the same effective
  version.
- Pre-release-aware SemVer comparison so `0.1.0+dev.<commit>` satisfies
  `>=0.1.0` while `0.1.0-dev.<commit>` (pre-release form) does not.
- Documentation updates in `docs/factory/version.md` and
  `docs/doctrine/doctrine-compiler.md`.
- Version command output that exposes both `version` (effective) and
  `declared_version` so reviewers can see when the stamp was derived.

Out:

- Full SemVer range matching (`^`, `~`, boolean operators) — already
  noted as deferred in `docs/doctrine/doctrine-compiler.md`.
- Multi-platform release publishing.
- A new Circus bootstrap attempt (handled by
  `ACT-CIRCUS-FACTORY-BOOTSTRAP01`).

## Executable contract

### Stable boundary

The contract lives in two places that are reachable from any entry
point:

1. The `leamas version` subcommand: it always reports the
   effective SemVer used by compatibility checks, plus
   the declared version that was stamped into the binary.
2. `internal/factory/doctrinecompiler.CheckCompilerCompatibility`,
   the single compatibility oracle used by `plan`, `compile`,
   and `verify`.

### Behavioural matrix

| Declared (`version` link) | Derived effective | Satisfies `>=0.1.0`? |
|---------------------------|-------------------|----------------------|
| `dev` (auto stamp)        | `0.1.0+dev.<commit>` | yes                  |
| `0.1.0` (release)         | `0.1.0`           | yes                  |
| `0.2.0`                   | `0.2.0`           | yes                  |
| `0.1.0+dev.abc`           | `0.1.0+dev.abc`   | yes                  |
| `0.0.9+dev.abc`           | `0.0.9+dev.abc`   | no (below floor)     |
| `0.1.0-dev.abc`           | `0.1.0-dev.abc`   | no (pre-release)     |
| `dev` (raw)               | `dev`             | no                   |
| empty                     | empty             | no                   |
| `unknown`                 | `unknown`         | no                   |
| `1.5.0`                   | `1.5.0`           | yes                  |

### Tests to add

- `internal/factory/doctrinecompiler/compat_test.go`:
  - `TestCheckCompilerCompatibility_BuildMetadataAccepted`
  - `TestCheckCompilerCompatibility_BuildMetadataBelowFloorRejected`
  - `TestCheckCompilerCompatibility_PreReleaseRejected`
  - `TestCheckCompilerCompatibility_UnknownPrereleaseRejected`
- `internal/version/effective_test.go`:
  - `TestEffective_DevDerivesSemVerCompatible`
  - `TestEffective_EmptyDerivesSemVerCompatible`
  - `TestEffective_UnknownDerivesSemVerCompatible`
  - `TestEffective_DeclaredUnchangedWhenAlreadySemVer`
  - `TestEffective_CommitSanitized`
  - `TestEffective_BuildMetadataIsSemVerSafe`
- `cmd/leamas/version_cli_test.go`:
  - update if the line-oriented output schema gains
    `declared_version:` (kept additive).

## Non-goals

- Force-pushing, branch-protection changes, GitHub policy changes.
- Multi-platform release build matrix (separate ACT).
- Adopting `Masterminds/semver` — a `cmd:` external dependency would
  violate the no-extra-deps posture outside the existing dependency
  tree.

## Verification

Before close:

```bash
make factorize
make gate
go test ./...
go vet ./...
CGO_ENABLED=0 go build -trimpath -o bin/leamas ./cmd/leamas
make build
./bin/leamas version
./bin/leamas version --json
make release VERSION=dev   # must FAIL
make release VERSION=0.1.0 # must PASS
```
