# Close Report: ACT-LEAMAS-DOMAIN-BOUNDARY-ASSERTION-EXPAND01

## Summary

Expanded the Factory domain-boundary verifier to include CLI runtime file policies, distinguishing between protected internal domain packages and legitimate CLI runtime imports for server/runtime functionality.

## Files Changed

1. `internal/factory/boundary/boundary.go` - Extended with FilePolicy type and CLI runtime policies
2. `internal/factory/boundary/boundary_test.go` - Added 20 comprehensive tests for CLI runtime policies
3. `docs/factory/domain-boundaries.md` - Updated documentation with CLI runtime file policies
4. `docs/close-reports/ACT-LEAMAS-DOMAIN-BOUNDARY-ASSERTION-EXPAND01.md` - This close report

## Policy Changes

### New FilePolicy Type

Added a new `FilePolicy` struct to support exact-file or path-scoped policies:

```go
type FilePolicy struct {
    Name              string
    File              string
    AllowedImports    map[string]bool
    ForbiddenImports  map[string]string
    ForbiddenContains []string
}
```

### New CLI Runtime File Policies

Added policies for two CLI runtime files:

1. `cmd/leamas/cockpit.go` - cockpit-cli-runtime
2. `cmd/leamas/witness.go` - witness-cli-runtime

### CLI Runtime Allowed Imports

CLI runtime files may now import:
- `context`, `errors`, `fmt`, `net`, `net/http`, `os`, `os/signal`, `strconv`, `strings`, `syscall`, `time`

### CLI Runtime Forbidden Imports

CLI runtime files must NOT import:
- `database/sql`, `os/exec`, `embed`, `html/template`, `text/template`

### CLI Runtime Allowed Internal Imports

CLI runtime files may import:
- `github.com/s1onique/leamas/internal/web/cockpit`
- `github.com/s1onique/leamas/internal/witness/proxy`

### CLI Runtime Forbidden Internal Imports

CLI runtime files must NOT import:
- `github.com/s1onique/leamas/internal/hulk/runbundle`
- `github.com/s1onique/leamas/internal/hulk/claimevidence`

### Provider/Auth/Database Forbidden Substrings

CLI runtime files must not import packages containing:
- `openai`, `anthropic`, `litellm`, `ollama`, `gemini`, `bedrock`, `azure`, `oauth`, `oidc`, `jwt`, `session`, `cookie`, `sqlite`, `postgres`, `mysql`

## Verifier Command

```bash
leamas factory verify domain-boundaries
```

## Make Target

```bash
make verify-domain-boundaries
```

## Gate Integration

The verifier remains integrated with:
- `make gate`
- `make factorize`

## Verification Commands and Results

### Test Results

```bash
go test ./internal/factory/boundary/... -v
```

All 20 tests passed:
- TestCurrentRepoPoliciesPass ✓
- TestHulkRunbundleAllowsSort ✓
- TestHulkClaimevidenceAllowsSort ✓
- TestWitnessProxyAllowsNetHTTP ✓
- TestWitnessProxyAllowsHttputil ✓
- TestCockpitAllowsEmbed ✓
- TestCockpitAllowsEncodingJSON ✓
- TestCockpitAllowsNetHTTP ✓
- TestFindingsDeterministic ✓
- TestResultOK ✓
- TestMissingDirectoryDetection ✓
- TestMissingCLIRuntimeFileDetection ✓
- TestCockpitCLIAllowsContext ✓
- TestCockpitCLIAllowsNetHTTP ✓
- TestCockpitCLIAllowsOSSignal ✓
- TestCockpitCLIAllowsInternalCockpit ✓
- TestWitnessCLIAllowsContext ✓
- TestWitnessCLIAllowsNetHTTP ✓
- TestWitnessCLIAllowsOSSignal ✓
- TestWitnessCLIAllowsInternalWitnessProxy ✓
- TestCLIRuntimeRejectsDatabaseSQL ✓
- TestCLIRuntimeRejectsOsExec ✓
- TestCLIRuntimeRejectsProviderImport ✓
- TestCLIRuntimeRejectsAuthImport ✓
- TestHulkStillRejectsNetHTTP ✓
- TestHulkStillRejectsTime ✓
- TestWebCockpitStillRejectsHttputil ✓
- TestWitnessProxyStillRejectsDatabaseSQL ✓
- TestCLIRuntimeRejectsForbiddenInternal ✓
- TestTestFilesIgnored ✓

### Build Test

```bash
CGO_ENABLED=0 go build -trimpath -o bin/leamas ./cmd/leamas
```

Build succeeded.

### Domain Boundaries Verification

```bash
./bin/leamas factory verify domain-boundaries
```

Verification PASSED.

### Make Targets

```bash
make verify-domain-boundaries  # PASSED
make factorize                 # PASSED
make gate                      # PASSED
```

### Go Toolchain

```bash
go test ./...   # All tests passed
go vet ./...    # All vet checks passed
```

## Skipped/Deferred Items

The following ACTs were explicitly NOT started per the task hard stop:

- ACT-LEAMAS-WITNESS-PROXY-INSPECT-CLI01
- ACT-LEAMAS-WEB-COCKPIT-BROWSER-OPEN01
- ACT-LEAMAS-DOMAIN-BOUNDARY-RUNTIME-SMOKE01

Additional deferred candidates:
- Runtime smoke tests for CLI boundaries
- Integration tests for actual CLI commands

## Acceptance Checklist

- [x] Existing domain-boundary verifier still exists
- [x] Verifier still uses Go AST/import parsing
- [x] Existing internal protected package policies still pass
- [x] CLI runtime file policies exist for cockpit.go and witness.go
- [x] Missing CLI runtime files are detected
- [x] CLI runtime files allow legitimate local server imports
- [x] CLI runtime files reject database/sql
- [x] CLI runtime files reject os/exec
- [x] CLI runtime files reject provider/control-plane imports
- [x] CLI runtime files reject auth/session-like imports
- [x] Hulk packages still reject net/http/time/database imports
- [x] Web cockpit still rejects reverse-proxy imports
- [x] Witness proxy still rejects database/provider imports
- [x] *_test.go files remain ignored
- [x] Findings are deterministic
- [x] Docs updated
- [x] Close report exists
- [x] `leamas factory verify domain-boundaries` passes
- [x] `make verify-domain-boundaries` passes
- [x] `make factorize` passes
- [x] `make gate` passes
- [x] `go test ./...` passes
- [x] `go vet ./...` passes
- [x] No product CLI behavior is added
- [x] No browser auto-open is added
- [x] No persistence/auth/database/provider-routing work is started

## Commit

```bash
git add internal/factory/boundary docs/factory/domain-boundaries.md docs/close-reports/ACT-LEAMAS-DOMAIN-BOUNDARY-ASSERTION-EXPAND01.md
git commit -m "ACT-LEAMAS-DOMAIN-BOUNDARY-ASSERTION-EXPAND01 expand CLI runtime boundaries"
```

## Next Candidates

1. ACT-LEAMAS-WITNESS-PROXY-INSPECT-CLI01
2. ACT-LEAMAS-WEB-COCKPIT-BROWSER-OPEN01
3. ACT-LEAMAS-DOMAIN-BOUNDARY-RUNTIME-SMOKE01
