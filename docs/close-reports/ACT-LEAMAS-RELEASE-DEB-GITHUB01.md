# Close Report: ACT-LEAMAS-RELEASE-DEB-GITHUB01

## Status

OPEN — implementation and local Debian verification are complete; GitHub
publication and independent downloaded-asset verification remain pending.

## Files changed

- `LICENSE` — official Apache License 2.0 text, committed verbatim.
- `packaging/nfpm.yaml` — Debian metadata and payload mapping.
- `packaging/deb.mk` — Debian package, inspection, verification, smoke, and
  release targets included by `Makefile`.
- `packaging/changelog` — nFPM changelog source for the first Debian release.
- `packaging/debian-copyright` — Debian copyright-format metadata.
- `packaging/lintian-overrides` — static Go binary Lintian justification.
- `internal/factory/releasedeb/*.go` — Go package and publication verifier.
- `cmd/leamas/factory_verify_release_deb.go` and `cmd/leamas/main.go` — Factory
  verifier command wiring.
- `.github/workflows/release-deb.yml` — tag-triggered release workflow.
- `.github/workflows/factory.yml` — Go toolchain derived from `go.mod`.
- `docs/install/debian.md` — Debian installation, removal, and upgrade guide.
- `docs/acts/ACT-LEAMAS-RELEASE-DEB-GITHUB01.md` — executable ACT contract.
- `README.md` — current implementation status and quick start.

## Behavior changed

`make package-deb` now validates a strict stable SemVer, Linux amd64 target,
Apache-2.0 license text, and pinned nFPM version before invoking the existing
`release-build` and `release-stamp-verify` targets. nFPM consumes the resulting
canonical binary and creates `dist/leamas_<version>_amd64.deb` with
`/usr/bin/leamas`.

The Go Factory verifier checks Debian metadata, payload executable shape,
Lintian, extraction, static amd64 ELF properties, binary version/commit/build
stamps, canonical/extracted SHA-256 identity, APT installation and removal,
and basename-only release checksums. Publication preflight checks clean Git
state, exact local tag-to-HEAD binding, pushed remote tag binding, and unique
asset basenames.

## Local verification evidence

Final successful commands and observations recorded so far:

```text
make release-deb VERSION=0.1.0 GOOS=linux GOARCH=amd64
# PASS: package-deb, dpkg-deb inspection, extraction/hash verification,
#       Lintian --fail-on error, and dist/SHA256SUMS generation.

make package-deb-install-smoke VERSION=0.1.0 GOOS=linux GOARCH=amd64
# PASS: APT install, /usr/bin/leamas selection, version/commit/build-time
#       checks, static-binary check, `leamas doctor`, removal, and no
#       /usr/bin/leamas afterward.

(cd dist && sha256sum --check SHA256SUMS)
# PASS: leamas_0.1.0_amd64.deb: OK

file dist/leamas_0.1.0_linux_amd64/leamas
# observed: ELF 64-bit x86-64, statically linked, stripped

ldd dist/leamas_0.1.0_linux_amd64/leamas || true
# observed: not a dynamic executable
```

The final local package metadata is:

```text
Package: leamas
Version: 0.1.0-1
Architecture: amd64
Section: devel
Priority: optional
Payload executable: /usr/bin/leamas
```

The final local package SHA-256 at the last successful build was recorded in
`dist/SHA256SUMS`; it must be copied here after the final clean release build
and independently compared with the downloaded GitHub asset.

## Tests

The new `internal/factory/releasedeb` contract suite passed its focused run and
covers invalid inputs, malformed release stamps, missing/wrong package
artifacts, extracted-byte mismatch, publication dirtiness/tag binding, missing
remote tags, and duplicate asset names.

## Required checks still to record

- [ ] Commit implementation on a clean tree.
- [ ] Run and record `make factorize` and `make gate` honestly.
- [ ] Run and record `go test ./...`, `go vet ./...`, and static build.
- [ ] Push annotated `v0.1.0` without moving an existing tag.
- [ ] Record successful GitHub Actions run and release URL.
- [ ] Independently download both release assets; record asset sizes and
  SHA-256 values.
- [ ] Install, execute, and remove the independently downloaded package.

If the known slow full-tree `dupcode` path prevents an end-to-end factorize,
gate, or unfiltered test run, the exact command, timeout, and observed
limitation must remain explicitly recorded rather than marked passed.
