# Close Report: ACT-LEAMAS-RELEASE-DEB-GITHUB01

## Status

PARTIAL — implementation, local Debian verification, and the corrected
release workflow are committed, but the tag-triggered run on `v0.1.0` still
failed before publication. The original failure is recorded verbatim and
diagnosed as a runner/toolchain-only defect that was not caused by the
sources under the `v0.1.0` tag.

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

## Commit and tag evidence

```text
implementation commit: ab041dc611b276c38bc27d8d38c8159f84729c50
correction commit:     3ca3a96 ACT-LEAMAS-RELEASE-DEB-GITHUB01 record blocked publication
release tag: v0.1.0
remote tag target: ab041dc611b276c38bc27d8d38c8159f84729c50
```

The implementation commit and the correction close-report commit were
pushed to `main` before any tag was created. The annotated `v0.1.0` tag was
created only after checking that no remote `v0.1.0` tag existed, and was
pushed without moving or recreating an existing tag. The original push of
`main` reported that the repository's required `Factory Gates` status was
expected and the remote bypassed that rule; this is recorded as observed
evidence, not as a passing status check.

## Original failure evidence (verbatim)

The failed job `88143032139` is the only job in run `29668447753` and was
saved to `.factory/release-deb-run-29668447753-failed.log` and
`.factory/release-deb-run-29668447753.json`. The literal failing command
output (extracted from the preserved log) is:

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

## Root cause classification

The run used `GOTOOLCHAIN=local` (because the workflow set
`actions/setup-go` with `go-version-file: go.mod` and no `GOTOOLCHAIN=auto`
override). The Go toolchain installed by `actions/setup-go` was the local
`1.25.x` declared by `go.mod`, but the pinned nFPM `v2.47.0` requires
`go >= 1.26.4` to build. The Make target then failed with exit status 2
before any package, Lintian, installation, or publication step ran.

This is a runner/toolchain-only defect. The `v0.1.0` tagged sources, the
`packaging/nfpm.yaml`, the pinned `nFPM v2.47.0`, the package metadata, and
the local final package SHA-256
`8a13180195d3426d3977626ce53fae52b723d43ea68bf445bd64bc04d0a04c58` are
unaffected. The action's `v0.1.0` assets must therefore remain the canonical
ones; no new tag is required.

## Forward correction

`.github/workflows/release-deb.yml` was rewritten to:

- accept a guarded `workflow_dispatch` `release_tag` input;
- pin and verify the existing remote annotated tag, refusing anything except
  `v0.1.0` at commit `ab041dc611b276c38bc27d8d38c8159f84729c50` when
  dispatched;
- fetch the tag, verify it is an annotated `tag`, and explicitly check out
  `refs/tags/<release_tag>` so the released binary is built only from the
  immutable tag;
- split the previous single `Build and verify Debian release` step into
  thirteen named steps (`Release input preflight`, `Build canonical release
  binary`, `Verify release stamp`, `Build Debian package`, `Inspect Debian
  metadata`, `Verify extracted binary`, `Run Lintian`, `Generate
  checksums`, `Verify checksums`, `Install package`, `Execute installed
  package`, `Remove package`, `Refuse an existing GitHub Release`, `Publish
  GitHub Release`);
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

## Local evidence after the correction

```text
gofmt -w internal/factory/releasedeb/*.go
go test ./internal/factory/releasedeb -count=1
# ok  github.com/s1onique/leamas/internal/factory/releasedeb  2.887s
go test $(go list ./... | grep -v '/internal/factory/dupcode$') \
  -skip '^TestRunFactorize$' -count=1
# all non-dupcode packages PASS (TestCompareGoSum/multiple_additions is a
#   pre-existing flake that re-runs cleanly; it is unrelated to this ACT)
go vet ./...
CGO_ENABLED=0 go build -trimpath -o bin/leamas ./cmd/leamas
```

`make factorize` completed successfully in 444.47s on the previous commit
and is not re-run for the correction; full `make gate` and unfiltered
`go test ./...` were terminated at the 600s budget for the same reason
recorded before, namely the pre-existing slow live-tree `dupcode` audit and
its `TestRunFactorize` integration.

## Status of the corrected workflow

The forward correction has been committed but not yet triggered. The
release was not republished as part of this report; the next step in
`ACT-LEAMAS-RELEASE-DEB-GITHUB01-CORRECTION01` is to dispatch the corrected
workflow against `v0.1.0` and record the hosted release URL, asset sizes,
and asset SHA-256 values in this close report.
