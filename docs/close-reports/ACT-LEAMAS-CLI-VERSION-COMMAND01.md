# ACT-LEAMAS-CLI-VERSION-COMMAND01 Close Report

## Summary

Added `leamas version` command that prints build metadata (version, commit,
build_time) in both line-oriented and JSON formats. Implemented as a small,
maintainable Go package with linker-injectable variables and fallback to
`runtime/debug.ReadBuildInfo()`.

## Files Changed

| File | Change |
|------|--------|
| `internal/version/version.go` | New - version package with `Get()` function |
| `internal/version/version_test.go` | New - tests for version package |
| `cmd/leamas/version.go` | New - `handleVersion()` CLI handler |
| `cmd/leamas/version_cli_test.go` | New - CLI integration tests |
| `cmd/leamas/main.go` | Modified - wired version command |
| `Makefile` | Modified - added LDFLAGS for version injection |
| `docs/factory/version.md` | New - documentation |

## Command Examples

```bash
# Line-oriented output (default)
leamas version
# Output:
# version: dev
# commit: 73a2b7ffb1a6
# build_time: 2026-07-09T10:24:46Z

# JSON output
leamas version --json
# Output:
# {
#   "version": "dev",
#   "commit": "73a2b7ffb1a6",
#   "build_time": "2026-07-09T10:24:46Z"
# }
```

## Verification Results

| Check | Result |
|-------|--------|
| `go test ./...` | PASS |
| `go vet ./...` | PASS |
| `make factorize` | PASS |
| `make gate` | PASS |
| `make build VERSION=0.1.0` | PASS |

### Build with injected metadata:
```bash
make build VERSION=0.1.0
./bin/leamas version
# Output:
# version: 0.1.0
# commit: 73a2b7ffb1a6
# build_time: 2026-07-09T17:30:00Z
```

## Non-Goals (Not Implemented)

- Semantic versioning policy
- Release automation
- GitHub release publishing
- Large metadata subsystem
- Verbose dependency/module build info

## R1 Fixes Applied

1. **Hardened tests** - Added tests for injected values and extracted
   `FromSettings()` function for pure fallback logic testing
2. **BUILD_TIME naming** - Standardized on `BUILD_TIME` across Makefile,
   release.txt, and version output (removed duplicate `BUILD_DATE`)
3. **Environment-sensitive test fixes** - Avoided assertions that Get()
   must return "unknown" since VCS fallback is legitimate

## Design Decisions

1. **Separate version.go file** - Split from main.go to keep files under
   400 lines for LLM-friendliness
2. **`Get()` function name** - Avoided collision with the `Info` struct type
3. **`FromSettings()` exported** - Enables pure unit testing of fallback logic
4. **JSON as optional flag** - `--json` only added to version command
