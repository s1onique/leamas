# Close Report: ACT-LEAMAS-FACTORY-RELEASE-PACKAGING01

## Summary

Added local release build, install verification, and checksum packaging for the Leamas static binary. The release workflow produces deterministic artifacts under `dist/` without requiring hosted automation.

## Files Changed

| File | Change |
|------|--------|
| `Makefile` | Added release targets: `release-build`, `release-checksum`, `release-verify`, `release-clean`, `release` |
| `docs/factory/release-packaging.md` | New documentation for release workflow |
| `docs/close-reports/ACT-LEAMAS-FACTORY-RELEASE-PACKAGING01.md` | This close report |
| `docs/factory/branch-protection-proof.md` | Fixed timestamp typo: `2026-08-07` → `2026-07-08` |
| `.gitignore` | Added `dist/` to exclude release artifacts from Git tracking |

## Artifact Layout

```
dist/
  leamas_<version>_<goos>_<goarch>/
    leamas          # Static binary
    SHA256SUMS      # SHA-256 checksum
    release.txt     # Build metadata
```

## Exact Release Commands

```bash
# Full release workflow
make release VERSION=0.0.0-test

# Or step by step
make release-clean
make release-build VERSION=0.0.0-test
make release-checksum VERSION=0.0.0-test
make release-verify VERSION=0.0.0-test
```

## Verification Commands and Results

```bash
# Test release build
make release-clean
make release VERSION=0.0.0-test
```

Expected output:
- `dist/leamas_0.0.0-test_darwin_arm64/leamas` exists
- `dist/leamas_0.0.0-test_darwin_arm64/SHA256SUMS` exists
- `dist/leamas_0.0.0-test_darwin_arm64/release.txt` exists

```bash
# Inspect artifacts
find dist -maxdepth 3 -type f -print
cat dist/leamas_0.0.0-test_$(go env GOOS)_$(go env GOARCH)/SHA256SUMS
cat dist/leamas_0.0.0-test_$(go env GOOS)_$(go env GOARCH)/release.txt

# Run binary
dist/leamas_0.0.0-test_$(go env GOOS)_$(go env GOARCH)/leamas version
```

```bash
# Quality gates
make factorize
make gate
go test ./...
go vet ./...
```

All tests pass.

## What Is Intentionally Not Included

- GitHub Release publishing
- Automatic tag creation
- Multi-platform matrix builds
- Package manager formulas (Homebrew, apt, yum, etc.)
- Container images (Docker, Podman)
- Signed artifacts
- SLSA/provenance attestation
- SBOM generation
- Cross-compilation beyond host platform

## Follow-up ACT Candidates

- `ACT-LEAMAS-FACTORY-GITHUB-RELEASE-PUBLISH01` - Manual GitHub release publishing workflow
- `ACT-LEAMAS-FACTORY-MULTIPLATFORM-RELEASE01` - Multi-platform build matrix
- `ACT-LEAMAS-FACTORY-ARTIFACT-SIGNING01` - Artifact signing and verification

## Stop Condition Met

- [x] `make release-clean` works
- [x] `make release VERSION=0.0.0-test` builds artifact directory
- [x] Release artifact contains `leamas` binary
- [x] Release artifact contains `SHA256SUMS`
- [x] Release artifact contains `release.txt`
- [x] Release binary is executable
- [x] Release binary runs a safe no-side-effect command
- [x] Checksum verification passes
- [x] Release docs exist
- [x] Close report exists
- [x] Branch-protection timestamp typo fixed
- [x] `make factorize` passes
- [x] `make gate` passes
- [x] `go test ./...` passes
- [x] `go vet ./...` passes
- [x] No GitHub release published
- [x] No tags created automatically
- [x] No Hulk/witness/web cockpit work started
