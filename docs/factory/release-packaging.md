# Release Packaging

This document describes the local release packaging workflow for the Leamas single static binary.

## Overview

Leamas uses a simple, local-first release packaging approach that produces deterministic artifacts without requiring hosted automation or external services.

## Release Artifacts

Default release output directory: `dist/`

Artifact layout:

```
dist/
  leamas_<version>_<goos>_<goarch>/
    leamas
    SHA256SUMS
    release.txt
```

Example:

```
dist/leamas_0.1.0_darwin_arm64/leamas
dist/leamas_0.1.0_darwin_arm64/SHA256SUMS
dist/leamas_0.1.0_darwin_arm64/release.txt
```

## Release Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `VERSION` | `dev` | Release version string |
| `COMMIT` | `git rev-parse --short=12 HEAD` | Git commit hash |
| `BUILD_DATE` | `date -u +%Y-%m-%dT%H:%M:%SZ` | Build timestamp |
| `DIST_DIR` | `dist` | Output directory |
| `GOOS` | `go env GOOS` | Target OS |
| `GOARCH` | `go env GOARCH` | Target architecture |

## Make Targets

### `make release-build`

Creates the release artifact directory and builds the static binary.

```bash
make release-build VERSION=0.1.0
```

Output:
- `dist/leamas_<version>_<goos>_<goarch>/leamas` - Static binary
- `dist/leamas_<version>_<goos>_<goarch>/release.txt` - Build metadata

### `make release-checksum`

Generates SHA-256 checksums for release artifacts.

```bash
make release-checksum VERSION=0.1.0
```

Output:
- `dist/leamas_<version>_<goos>_<goarch>/SHA256SUMS` - Checksum file

Supports both `sha256sum` (Linux) and `shasum -a 256` (macOS).

### `make release-verify`

Verifies release artifacts:
1. Binary exists and is executable
2. Binary runs without errors
3. Checksums match (if SHA256SUMS exists)

```bash
make release-verify VERSION=0.1.0
```

### `make release-clean`

Removes all release artifacts.

```bash
make release-clean
```

### `make release`

Convenience target that runs the full release workflow:

```bash
make release VERSION=0.1.0
```

Equivalently:

```bash
make release-build release-checksum release-verify VERSION=0.1.0
```

## Complete Workflow

### Create a release

```bash
# Clean any previous artifacts
make release-clean

# Build and verify
make release VERSION=0.1.0
```

### Inspect artifacts

```bash
# List all files
find dist -maxdepth 3 -type f -print

# View checksums
cat dist/leamas_0.1.0_darwin_arm64/SHA256SUMS

# View build metadata
cat dist/leamas_0.1.0_darwin_arm64/release.txt
```

### Verify checksum manually

```bash
# On Linux
sha256sum -c dist/leamas_0.1.0_darwin_arm64/SHA256SUMS

# On macOS
cd dist/leamas_0.1.0_darwin_arm64
shasum -a 256 -c SHA256SUMS
```

### Run the binary

```bash
dist/leamas_0.1.0_darwin_arm64/leamas version
dist/leamas_0.1.0_darwin_arm64/leamas --help
```

## Release Metadata

The `release.txt` file contains build information:

```
version=0.1.0
commit=083fb1cf7da1
build_date=2026-08-07T19:51:00Z
goos=darwin
goarch=arm64
```

## What Is Not Included

This release packaging does NOT include:
- GitHub Release publishing
- Automatic tag creation
- Multi-platform matrix builds
- Package manager formulas (Homebrew, apt, yum, etc.)
- Container images (Docker, Podman)
- Signed artifacts
- SLSA/provenance attestation
- SBOM generation
- Cross-compilation beyond host platform

These features may be added in future ACTs.
