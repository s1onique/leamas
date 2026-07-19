# Close Report: ACT-LEAMAS-RELEASE-DEB-GITHUB01

## Status

PARTIAL — implementation, commit, tag push, and final local Debian
verification completed. The tag-triggered GitHub Actions release failed in
`Build and verify Debian release`, so no GitHub Release was created and this
ACT is not closed.

## Commit and tag evidence

```text
implementation commit: ab041dc611b276c38bc27d8d38c8159f84729c50
release tag: v0.1.0
remote tag target: ab041dc611b276c38bc27d8d38c8159f84729c50
```

The annotated tag was created only after checking that no remote `v0.1.0` tag
existed, and was pushed without moving or recreating an existing tag. The
implementation commit was pushed to `main` first. The main push reported that
the repository's required `Factory Gates` status was expected and the remote
bypassed that rule; this is recorded as observed evidence, not as a passing
status check.

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
- `README.md` and `docs/README.md` — current status and documentation index.

## Behavior changed

`make package-deb` validates a strict stable SemVer, Linux amd64 target,
Apache-2.0 license text, and pinned nFPM version before invoking the existing
`release-build` and `release-stamp-verify` targets. nFPM consumes the resulting
canonical binary and creates `dist/leamas_<version>_amd64.deb` with
`/usr/bin/leamas`.

The Go Factory verifier checks Debian metadata, payload executable shape,
Lintian, extraction, static amd64 ELF properties, binary version/commit/build
stamps, canonical/extracted SHA-256 identity, APT installation and removal,
and basename-only release checksums. Publication preflight checks clean Git
state, exact local tag-to-HEAD binding, pushed remote tag binding, and unique
asset basenames. The existing release binary remains the only Leamas product
compilation authority.

## Final local package evidence

Commands run successfully after commit `ab041dc`:

```text
make release-clean
make release-deb VERSION=0.1.0 GOOS=linux GOARCH=amd64
(cd dist && sha256sum --check SHA256SUMS)
make package-deb-install-smoke VERSION=0.1.0 GOOS=linux GOARCH=amd64
(cd dist && sha256sum --check SHA256SUMS)
test ! -e /usr/bin/leamas
```

Observed final package metadata:

```text
Package: leamas
Version: 0.1.0-1
Architecture: amd64
Section: devel
Priority: optional
Payload executable: /usr/bin/leamas
```

Observed final local asset evidence:

```text
leamas_0.1.0_amd64.deb
size: 3745486 bytes
sha256: 8a13180195d3426d3977626ce53fae52b723d43ea68bf445bd64bc04d0a04c58

SHA256SUMS
size: 89 bytes

canonical release binary: dist/leamas_0.1.0_linux_amd64/leamas
size: 8982712 bytes
sha256: bc6e7232f1464d687384cab6775d53edc3b57033eb7a4742478beaf8fe19dc06
```

`file` reported the canonical binary as an ELF 64-bit x86-64 statically
linked stripped executable. `ldd` reported `not a dynamic executable`. The APT
smoke installed the package, selected `/usr/bin/leamas` despite a pre-existing
`/usr/local/bin/leamas` on the development machine, ran `leamas doctor`, and
removed the package. The final package verifier reported only Lintian
warnings (`initial-upload-closes-no-bugs` and `no-manual-page`); no Lintian
errors occurred.

## Tests and repository verification

Successful focused and subsystem checks:

```text
gofmt -w cmd/leamas/factory_verify_dispatch_test.go \
  cmd/leamas/factory_verify_release_deb.go internal/factory/releasedeb/*.go
go test ./cmd/leamas ./internal/factory/releasedeb -count=1
go test $(go list ./... | grep -v '/internal/factory/dupcode$') \
  -skip '^TestRunFactorize$' -count=1
go vet ./...
CGO_ENABLED=0 go build -trimpath -o bin/leamas ./cmd/leamas
```

`make factorize` completed successfully in `444.47s`; its two dominant checks
were `dupcode` (`222.34s`) and `dupcode-baseline` (`220.54s`). The first full
`make gate` attempt exposed and was corrected for the new verifier count. The
final `make gate` rerun completed its factorize, Factory checks, `go mod tidy`,
`gofmt`, and `go vet ./...` phases, then was terminated at the 600-second
budget while running the unfiltered `go test ./...` phase. The standalone
`timeout 300s go test ./...` likewise did not complete. These are not claimed
as passing full-tree checks; the pre-existing slow live-tree duplicate-code
and factorize-related tests remain the known limitation.

## GitHub Actions publication evidence

The pushed tag started:

```text
workflow: Release Debian package
run: https://github.com/s1onique/leamas/actions/runs/29668447753
job: https://github.com/s1onique/leamas/actions/runs/29668447753/job/88143032139
head_sha: ab041dc611b276c38bc27d8d38c8159f84729c50
conclusion: failure
failed step: Build and verify Debian release
```

The preceding checkout/tag validation, repository Go setup, Lintian install,
and practical release checks succeeded. The package build/verification step
exited with code 2; subsequent install, release-conflict, and publication
steps were skipped. Public GitHub API evidence showed no `v0.1.0` Release after
the failed run. The unauthenticated API did not expose the detailed job log,
so the precise command-level failure inside that step remains unresolved and
must not be invented.

## Not completed

- No GitHub Release URL exists for `v0.1.0`.
- No release assets were independently downloaded.
- No downloaded asset installation/removal evidence exists.
- `SHA256SUMS` was not compared against a GitHub-hosted asset.

## Follow-up ACT required before closure

Investigate the exact GitHub `Build and verify Debian release` exit-2 log,
correct the workflow or runner incompatibility in a forward commit, and use a
safe release process that does not move or recreate the existing `v0.1.0` tag.
Only after a successful tag-bound workflow, independent asset download,
checksum verification, installation, execution, and removal may this ACT be
closed.
