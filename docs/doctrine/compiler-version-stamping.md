# Compiler Version Stamping

This document describes how Leamas stamps development and release
binaries with a SemVer-compatible effective version so the doctrine
compiler's compatibility check sees a real `MAJOR.MINOR.PATCH`
string at every invocation.

The doctrine compiler's `CheckCompilerCompatibility` oracle refuses
placeholder versions (`dev`, `unknown`, empty) when the constraint
is non-empty. Without an effective stamp, a developer's locally
installed Leamas cannot participate in its own compatibility
protocol even though the source it carries implements the correct
contract.

## Effective version derivation

Leamas splits build metadata into two surfaces:

| Field             | Role                                                       |
|-------------------|------------------------------------------------------------|
| `version`         | Effective SemVer used by `CheckCompilerCompatibility`. Auto-derived for development builds to `<SemVerCompatible>+dev.<commit>.<ts>`. |
| `declared_version`| Literal `VERSION=` value passed to the build. Surfaced only when it differs from the effective version. |
| `commit`          | VCS commit hash. Always present, separate from the SemVer. |
| `build_time`      | Build timestamp in RFC3339/UTC. Always present, separate. |

The development-build stamp follows:

```text
0.1.0+dev.<short-commit>.<build-timestamp>
```

The base version `0.1.0` matches the canonical `>=0.1.0` floor.
Build metadata after `+` has no precedence per SemVer 2.0.0 §10,
so the stamp is comparable to `0.1.0` and satisfies the floor.

When the commit or build timestamp are unavailable (e.g. a
test-only build without `-ldflags`), the helper degrades
gracefully: unknown/empty provenance segments are dropped, and
the base version alone is returned.

For release builds, the declared value is preserved verbatim.

## Building a development binary

```bash
make build                 # auto-stamps from VERSION=dev
./bin/leamas version
# version: 0.1.0+dev.92b459cf0806.20260712T064118Z
# declared_version: dev
# commit: 92b459cf0806
# build_time: 2026-07-12T06:41:18Z
```

The Makefile injects both `Version` and `DeclaredVersion` via
`-ldflags`. `internal/version.Get()` returns both; the CLI command
prints only the derived `declared_version:` line when the stamp was
auto-derived.

## Building a release binary

Release builds require an explicit SemVer; the build refuses
`VERSION=dev`, empty, or `unknown`:

```bash
make release VERSION=0.1.0
```

The guard lives in the `release-build` recipe. The stamp-check
verifier (`make stamp-check STAMP_BINARY=…`) confirms that an
existing artefact reports a real SemVer before it is shipped.

## Injection mechanism

The version is injected via the standard `-ldflags` mechanism in
`internal/version`:

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

Tests that exercise doctrine commands in a subprocess inject the
same concrete version. The compiler's `init()` wires
`version.Get().Version` (which is the effective stamp) into the
compatibility check so production binaries always run with the
correct identity.

## Acceptance criteria

The contract enforced by this document and its companion tests is
specified in `ACT-LEAMAS-COMPILER-VERSION-STAMPING01`:

1. Release binaries report strict SemVer such as `0.1.0`.
2. Compatible development binaries report strict SemVer with
   build metadata, such as `0.1.0+dev.<commit>`.
3. Commit, build timestamp, and dirty status remain separate
   provenance fields.
4. Raw `dev`, empty, or otherwise unparsable versions still fail
   against non-empty constraints.
5. `0.1.0+dev.<commit>` satisfies `>=0.1.0`.
6. `0.0.9+dev.<commit>` does not satisfy `>=0.1.0`.
7. `0.1.0-dev.<commit>` does not satisfy `>=0.1.0`.
8. `plan`, `compile`, and `verify` apply the identical effective-
   version policy.
9. The lock records both the effective compiler version and
   immutable build provenance.
10. Packaging/install targets fail when they would emit an
    unstamped `dev` binary.
