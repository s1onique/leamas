# ACT-LEAMAS-RELEASE-DEB-GITHUB01

## Goal

Build and publish the first Linux amd64 Debian package for Leamas as GitHub
Release `v0.1.0`.

The Debian data payload installs the canonical release binary at
`/usr/bin/leamas`. The package asset is
`leamas_0.1.0_amd64.deb`, accompanied by `SHA256SUMS`.

## Decisions

| Decision | Contract |
|---|---|
| Release | `v0.1.0` |
| Debian package | `leamas_0.1.0_amd64.deb` |
| Build authority | Existing `make release-build` target |
| Target | `GOOS=linux`, `GOARCH=amd64`, Debian architecture `amd64` |
| Packager | nFPM `v2.47.0`, invoked with `go run` and not installed globally |
| Runner | `ubuntu-24.04` |
| Publication | Pushed stable `v*` tag; GitHub CLI with `--verify-tag` and no `--clobber` |
| License | Apache-2.0, selected by the repository owner and committed verbatim in `LICENSE` |

The Debian package includes Debian copyright/changelog and the Lintian
metadata needed for the intentionally static Go binary. These are package
documentation/quality metadata, not runtime configuration, users, maintainer
scripts, services, or dependencies.

## Implementation boundary

The `.deb` layer never invokes a second Leamas product build. `package-deb`
validates the release inputs, invokes `release-build` and
`release-stamp-verify`, and gives nFPM the resulting binary at
`dist/leamas_<version>_linux_amd64/leamas`.

The Go `leamas factory verify release-deb` verifier owns package inspection,
Lintian invocation, extraction, static-binary checks, byte-identity checks,
APT install/removal smoke checks, checksum generation, and publication
preflight. Publication preflight fails closed for dirty trees, local or
remote tags not bound to `HEAD`, missing remote tags, and duplicate asset
basenames.

## Negative contract matrix

The executable contract tests cover rejection of:

- empty, placeholder, malformed, prerelease, and build-metadata versions;
- non-Linux or non-amd64 build targets;
- missing license metadata or non-authoritative license text;
- absent canonical release binaries and wrong embedded versions;
- wrong Debian architecture or missing `/usr/bin/leamas` payloads;
- extracted binaries whose bytes differ from the canonical release binary;
- dirty publication trees, tags not at `HEAD`, missing remote tags, and
  duplicate release asset names.

## Closure rule

This ACT is not closed until the implementation is committed, local package
and APT smoke verification passes, the tag-triggered GitHub Actions workflow
publishes `v0.1.0`, the assets are independently downloaded and checked, the
downloaded package installs and executes, removal succeeds, and the exact tag,
commit, asset names, sizes, and SHA-256 values are recorded in the close
report.

The known slow full-tree `dupcode` limitation from the wall-clock ACT must be
reported honestly if full `make factorize`, `make gate`, or unfiltered
`go test ./...` cannot complete in the available verification budget.
