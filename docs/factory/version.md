# Leamas Version Command

The `leamas version` command prints build metadata for the leamas binary.

## Usage

```bash
leamas version
leamas version --json
```

## Output

### Default (line-oriented)

```
version: dev
commit: 73a2b7ffb1a6
build_time: 2026-07-09T17:30:00Z
```

### JSON Format

```bash
leamas version --json
```

```json
{
  "version": "dev",
  "commit": "73a2b7ffb1a6",
  "build_time": "2026-07-09T17:30:00Z"
}
```

## Build Metadata

The command reports three pieces of information:

| Field | Description |
|-------|-------------|
| `version` | Software version (default: `dev`) |
| `commit` | VCS commit hash (default: `unknown`) |
| `build_time` | Build timestamp in RFC3339/UTC (default: `unknown`) |

## Release Builds

For release builds, inject metadata via linker flags:

```bash
VERSION=0.1.0
COMMIT=$(git rev-parse --short=12 HEAD)
BUILD_TIME=$(date -u +%Y-%m-%dT%H:%M:%SZ)

MODULE_PATH=github.com/s1onique/leamas

CGO_ENABLED=0 go build -ldflags "\
  -X '${MODULE_PATH}/internal/version.Version=${VERSION}' \
  -X '${MODULE_PATH}/internal/version.Commit=${COMMIT}' \
  -X '${MODULE_PATH}/internal/version.BuildTime=${BUILD_TIME}'" \
  -o bin/leamas ./cmd/leamas
```

Or use the Makefile:

```bash
make build VERSION=0.1.0
```

## Notes

- Build time must be RFC3339/UTC format when injected
- In development, `runtime/debug.ReadBuildInfo()` provides fallback VCS info
- The command exits 0 for all valid outputs
- Only returns error for impossible formatting failures
