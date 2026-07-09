# Close Report: ACT-LEAMAS-DOMAIN-BOUNDARY-ASSERTION-EXPAND01

## Summary

Expanded the Factory domain-boundary verifier to include CLI runtime file policies, distinguishing between protected internal domain packages and legitimate CLI runtime imports for server/runtime functionality.

## Files Changed

1. `internal/factory/boundary/boundary.go` - Extended with FilePolicy type, CLI runtime policies, and isStandardLibrary helper
2. `internal/factory/boundary/check.go` - New file with checkFile, checkFileForCLI, checkPackage, checkCLIFile functions
3. `internal/factory/boundary/boundary_test.go` - Core integration tests for protected packages
4. `internal/factory/boundary/boundary_cli_test.go` - CLI-specific tests (split to meet LLM-friendliness ≤400 lines)
5. `internal/factory/boundary/boundary_cli_reject_test.go` - CLI reject and allowlist enforcement tests (new file)
6. `internal/factory/boundary/boundary_hulk_test.go` - Updated to use checkPackage with repoRoot parameter
7. `internal/factory/boundary/boundary_witness_test.go` - Updated to use checkPackage with repoRoot parameter
8. `internal/factory/boundary/boundary_cockpit_test.go` - Updated to use checkPackage with repoRoot parameter
9. `docs/factory/domain-boundaries.md` - Updated documentation with CLI runtime file policies
10. `docs/close-reports/ACT-LEAMAS-DOMAIN-BOUNDARY-ASSERTION-EXPAND01.md` - This close report

## Policy Changes

### R1: Allowlist Enforcement (Fixed)

The verifier now properly enforces AllowedImports for standard-library imports:

1. **ForbiddenContains** is checked first for ALL imports (third-party and standard library)
2. **AllowedImports** is checked only for standard-library imports that pass ForbiddenContains
3. **ForbiddenImports** is checked only for standard-library imports that pass both checks

This ensures that:
- Third-party provider imports (e.g., `github.com/someone/openai-sdk`) are caught by ForbiddenContains
- Unlisted standard-library imports are caught by AllowedImports enforcement
- Tests now use `repoRoot()` instead of `Check(".")` for correct root directory

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

## Verification Commands and Results

### Test Results

```bash
go test ./internal/factory/boundary/... -v
```

All 32 tests passed:
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
- TestCLIRuntimeRejectsUnlistedStdlib ✓ (NEW)
- TestCLIRuntimeRejectsDatabaseSQL ✓
- TestCLIRuntimeRejectsOsExec ✓
- TestCLIRuntimeRejectsProviderImport ✓
- TestCLIRuntimeRejectsAuthImport ✓
- TestHulkStillRejectsNetHTTP ✓
- TestHulkStillRejectsTime ✓
- TestWebCockpitStillRejectsHttputil ✓
- TestWitnessProxyStillRejectsDatabaseSQL ✓
- TestCLIRuntimeRejectsForbiddenInternal ✓
- TestBoundaryTestFilesIgnored ✓
- TestHulkRejectsUnlistedStdlib ✓ (NEW)

### Build Test

```bash
CGO_ENABLED=0 go build -trimpath -o bin/leamas ./cmd/leamas
```

Build succeeded.

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
- [x] AllowedImports is enforced for standard-library imports
- [x] New tests prove allowlist enforcement works
- [x] CLI allow tests use repoRoot() instead of Check(".")
- [x] Docs updated
- [x] Close report exists with accurate file list
- [x] `make factorize` passes
- [x] `make gate` passes
- [x] `go test ./...` passes
- [x] `go vet ./...` passes
- [x] No product CLI behavior is added

## Next Candidates

1. ACT-LEAMAS-WITNESS-PROXY-INSPECT-CLI01
2. ACT-LEAMAS-WEB-COCKPIT-BROWSER-OPEN01
3. ACT-LEAMAS-DOMAIN-BOUNDARY-RUNTIME-SMOKE01
