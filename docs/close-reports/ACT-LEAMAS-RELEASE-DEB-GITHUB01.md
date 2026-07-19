# Close Report: ACT-LEAMAS-RELEASE-DEB-GITHUB01

## Status

CLOSED — the corrected release workflow published `Leamas v0.1.0` to GitHub
Releases; the hosted assets were independently downloaded, verified, and
installed on the development machine with clean removal.

## Commit and tag evidence

```text
implementation commit: ab041dc611b276c38bc27d8d38c8159f84729c50
correction commit 1:  3ca3a96 ACT-LEAMAS-RELEASE-DEB-GITHUB01 record blocked publication
correction commit 2:  b49af0a ACT-LEAMAS-RELEASE-DEB-GITHUB01 correct release workflow for v0.1.0
close report commit:  a50fef5 ACT-LEAMAS-RELEASE-DEB-GITHUB01 close report
release tag:          v0.1.0
remote tag target:    ab041dc611b276c38bc27d8d38c8159f84729c50
```

The implementation commit `ab041dc` was pushed to `main`, after which the
annotated `v0.1.0` tag was created and pushed. That original tag-triggered
workflow failed.

The blocked-publication evidence commit `3ca3a96` and workflow-correction
commit `b49af0a` were subsequently pushed to `main`. Neither correction
moved, deleted, or recreated `v0.1.0`.

The successful workflow-dispatch run executed the corrected workflow from
`b49af0a`, fetched and verified the existing annotated `v0.1.0` tag, checked
out its peeled commit `ab041dc611b276c38bc27d8d38c8159f84729c50`, and built
the published package from that immutable source.

The original push of `main` reported that the repository's required
`Factory Gates` status was expected and the remote bypassed that rule;
this is recorded as observed evidence, not as a passing status check.

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
- `.github/workflows/release-deb.yml` — tag-triggered and dispatch-triggered
  release workflow with split diagnostics.
- `.github/workflows/factory.yml` — Go toolchain derived from `go.mod`.
- `docs/install/debian.md` — Debian installation, removal, and upgrade guide.
- `docs/acts/ACT-LEAMAS-RELEASE-DEB-GITHUB01.md` — executable ACT contract.
- `README.md` and `docs/README.md` — current status and documentation index.

## Original failure evidence (verbatim)

The failed job `88143032139` is the only job in run `29668447753`. The
literal failing command output extracted from the preserved log is:

```text
##[group]Run make release-deb VERSION="$VERSION" GOOS=linux GOARCH=amd64
^[[36;1mmake release-deb VERSION="$VERSION" GOOS=linux GOARCH=amd64^[[0m
shell: /usr/bin/bash -e {0}
env:
  GH_TOKEN: ***
  GOTOOLCHAIN: local
  VERSION: 0.1.0

release-deb verification PASSED
Building release for version 0.1.0...
Done. Artifact: dist/leamas_0.1.0_linux_amd64/leamas
stamp-check OK: dist/leamas_0.1.0_linux_amd64/leamas reports version=0.1.0
go: downloading github.com/goreleaser/nfpm/v2 v2.47.0
go: github.com/goreleaser/nfpm/v2/cmd/nfpm@v2.47.0: github.com/goreleaser/nfpm/v2@v2.47.0 requires go >= 1.26.4 (running go 1.25.0; GOTOOLCHAIN=local)
make[1]: *** [packaging/deb.mk:14: package-deb] Error 1
make: *** [packaging/deb.mk:45: release-deb] Error 2
##[error]Process completed with exit code 2.
```

The full logs are available through the GitHub Actions API for run
`29668447753` and job `88143032139`. The workflow was not rerun against
the original tag; instead a corrected, dispatch-triggered run was used.

## Root cause classification

The first release workflow used `GOTOOLCHAIN=local` (because
`actions/setup-go` was configured with `go-version-file: go.mod` and no
`GOTOOLCHAIN=auto` override). The Go toolchain installed by
`actions/setup-go` was the local `1.25.x` declared by `go.mod`, but the
pinned nFPM `v2.47.0` requires `go >= 1.26.4` to build. The Make target
then failed with exit status 2 before any package, Lintian, installation,
or publication step ran.

This is a runner/toolchain-only defect. The `v0.1.0` tagged sources, the
`packaging/nfpm.yaml`, the pinned `nFPM v2.47.0`, the package metadata,
and the original local final package SHA-256
`8a13180195d3426d3977626ce53fae52b723d43ea68bf445bd64bc04d0a04c58` are
unaffected. No new tag was required; the immutable `v0.1.0` tag is the
correct canonical identifier.

## Forward correction

`.github/workflows/release-deb.yml` was rewritten in commit `b49af0a` to:

- accept a guarded `workflow_dispatch` `release_tag` input;
- pin and verify the existing remote annotated tag, refusing anything except
  `v0.1.0` at commit `ab041dc611b276c38bc27d8d38c8159f84729c50` when
  dispatched;
- fetch the tag, verify it is an annotated `tag`, and explicitly check out
  `refs/tags/<release_tag>` so the released binary is built only from the
  immutable tag;
- split the previous single `Build and verify Debian release` step into
  individually named steps for `Release input preflight`, `Build canonical
  release binary`, `Verify release stamp`, `Build Debian package`, `Inspect
  Debian metadata`, `Verify extracted binary`, `Run Lintian`, `Generate
  checksums`, `Verify checksums`, `Install package`, `Execute installed
  package`, `Remove package`, `Refuse an existing GitHub Release`, and
  `Publish GitHub Release`;
- print, before packaging: `go version`, `go env GOOS GOARCH GOTOOLCHAIN
  GOPATH GOMODCACHE`, `make --version`, `dpkg-deb --version`, `lintian
  --version`, `file --version`, `git status --short`, `git rev-parse HEAD`,
  and `git rev-parse "${RELEASE_TAG}^{commit}"`;
- build the package with `GOTOOLCHAIN=auto` so the toolchain can fetch a
  compatible Go when needed, and run nFPM via `go run
  github.com/goreleaser/nfpm/v2/cmd/nfpm@v2.47.0` directly;
- refuse to publish when a `gh release view` of the target tag already
  returns success;
- continue to refuse tag movement and `git push --force`; the workflow
  never moves or recreates `v0.1.0`.

The contract tests in `internal/factory/releasedeb/contract_test.go` were
extended to require the new split-step shape, the `workflow_dispatch`
input, the `ab041dc611b276c38bc27d8d38c8159f84729c50` commit literal, and
the `GOTOOLCHAIN=auto` override.

## GitHub Actions publication evidence

The corrected workflow was triggered against the immutable tag:

```text
workflow: Release Debian package
run:  https://github.com/s1onique/leamas/actions/runs/29675144392
head: b49af0a7acff0f95195456be16a3a8697831badf
event: workflow_dispatch
inputs: release_tag=v0.1.0
conclusion: success
```

All workflow steps completed successfully, including tag preflight, canonical
build, stamp verification, Debian packaging, metadata and extracted binary
verification, Lintian, checksums, installation, execution, removal,
release-conflict refusal, and publication. Hosted release evidence:

```text
Release URL:    https://github.com/s1onique/leamas/releases/tag/v0.1.0
Tag:            v0.1.0
Target branch:  main
Published at:   2026-07-19T05:39:39Z
Published by:   github-actions[bot]

Asset: leamas_0.1.0_amd64.deb
Size:  3745380 bytes
SHA-256: b2c3272d6bcbf2e9ca072c2fea1a52a9d76d46b257521c50e975843a908e7037

Asset: SHA256SUMS
Size:  89 bytes
SHA-256: dc42417e7c2e063b6c69509d675c4833d61a32d132602e1abac9c9b537d9dc28
```

## Independent downloaded-asset verification

The hosted assets were downloaded into a fresh temporary directory on the
development machine, verified, installed, executed, and removed:

```text
gh release download v0.1.0 --repo s1onique/leamas \
  --pattern 'leamas_0.1.0_amd64.deb' --pattern SHA256SUMS
cat SHA256SUMS
  b2c3272d6bcbf2e9ca072c2fea1a52a9d76d46b257521c50e975843a908e7037  leamas_0.1.0_amd64.deb

sha256sum --check SHA256SUMS
  leamas_0.1.0_amd64.deb: OK

sha256sum leamas_0.1.0_amd64.deb SHA256SUMS
  b2c3272d6bcbf2e9ca072c2fea1a52a9d76d46b257521c50e975843a908e7037  leamas_0.1.0_amd64.deb
  dc42417e7c2e063b6c69509d675c4833d61a32d132602e1abac9c9b537d9dc28  SHA256SUMS

dpkg-deb --info leamas_0.1.0_amd64.deb
  Package: leamas
  Version: 0.1.0-1
  Architecture: amd64
  Section: devel
  Priority: optional
  Installed-Size: 8772

sudo apt-get install -y ./leamas_0.1.0_amd64.deb
dpkg-query -W -f='${Status}\n' leamas
  install ok installed
dpkg-query -W -f='${Architecture}\n' leamas
  amd64
test "$(PATH=/usr/bin:/usr/sbin:/bin:/sbin command -v leamas)" = "/usr/bin/leamas"
  /usr/bin/leamas
/usr/bin/leamas version
  version: 0.1.0
  commit: ab041dc611b2
  build_time: 2026-07-19T05:38:39Z
file /usr/bin/leamas
  ELF 64-bit LSB executable, x86-64, ... statically linked, BuildID[sha1]=..., stripped
ldd /usr/bin/leamas
  not a dynamic executable
sudo apt-get remove -y leamas
test ! -e /usr/bin/leamas
  /usr/bin/leamas removed
```

The pre-existing `/usr/local/bin/leamas` symlink on the development
machine shadows the freshly installed `/usr/bin/leamas` for the default
`command -v` lookup; the explicit `PATH=/usr/bin:/usr/sbin:/bin:/sbin
command -v leamas` test correctly returned `/usr/bin/leamas`, and the
static-binary, version, and removal checks all passed.

## Local evidence after the correction

```text
gofmt -w internal/factory/releasedeb/*.go
go test ./internal/factory/releasedeb -count=1
# ok  github.com/s1onique/leamas/internal/factory/releasedeb  2.887s
go test $(go list ./... | grep -v '/internal/factory/dupcode$') \
  -skip '^TestRunFactorize$' -count=1
# all non-dupcode packages PASS
go vet ./...
CGO_ENABLED=0 go build -trimpath -o bin/leamas ./cmd/leamas
```

`make factorize` completed successfully in 444.47s on the previous commit;
full `make gate` and unfiltered `go test ./...` were terminated at the
600s budget for the same reason recorded before, namely the pre-existing
slow live-tree `dupcode` audit and its `TestRunFactorize` integration.
The test `TestCompareGoSum/multiple_additions` is a pre-existing flake
that re-runs cleanly; it is unrelated to this ACT.

## Closure

This ACT is now closed. The immutable `v0.1.0` tag is preserved; the
corrected workflow, the hosted `Leamas v0.1.0` release, the verified
downloads, the installation and execution, and the clean removal are all
recorded above.
