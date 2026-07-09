# Close Report: ACT-LEAMAS-WITNESS-RUN-BUNDLE-CLI01

## Summary

Added a minimal CLI surface for local run bundles via `leamas witness run-bundle`
with create/list/show subcommands. The CLI wires the existing `internal/witness/runbundle`
package into `cmd/leamas` without adding witness proxy persistence, cockpit UI,
database persistence, or network behavior.

## Files Changed

| File | Change |
|------|--------|
| `cmd/leamas/witness.go` | Added `run-bundle` case to witness dispatch |
| `cmd/leamas/run_bundle.go` | New file with create/list/show command handlers |
| `cmd/leamas/run_bundle_test.go` | New file with 17 tests for CLI commands |
| `docs/factory/run-bundles.md` | Added CLI section with usage examples |
| `docs/close-reports/ACT-LEAMAS-WITNESS-RUN-BUNDLE-CLI01.md` | This close report |

## CLI Added

### Commands

```text
leamas witness run-bundle create --id <run-id> [--root <path>] [--tool-version <version>] [--json]
leamas witness run-bundle list [--root <path>] [--json] [--include-invalid]
leamas witness run-bundle show <run-id> [--root <path>] [--json]
```

### Features

- **Default root**: `.leamas/runs`
- **All commands support `--root`**: Explicit root directory specification
- **JSON output**: All commands support `--json` for machine-readable output
- **Run ID validation**: Uses `runbundle.ValidateRunID` to reject unsafe IDs
- **Root validation**: Rejects empty root directories
- **Error messages**: Clear, user-facing error messages without Go stack traces

## Behavior Proved

An operator can now:

1. **Create a local run bundle**: `leamas witness run-bundle create --id run-xyz`
2. **List local run bundles**: `leamas witness run-bundle list`
3. **Inspect run-bundle metadata**: `leamas witness run-bundle show run-xyz`
4. **Choose an explicit root directory**: `--root /path/to/runs`
5. **Stay entirely local-only and filesystem-only**: No network calls, database, or external services

## Tests Added

### Help and Dispatch Tests
- `TestRunBundleHelp`
- `TestRunBundleUnknownSubcommand`
- `TestRunBundleCreateRequiresID`
- `TestRunBundleShowRequiresID`

### Create Command Tests
- `TestRunBundleCreateCreatesBundle`
- `TestRunBundleCreateJSONOutput`
- `TestRunBundleCreateRejectsInvalidID`
- `TestRunBundleCreateRejectsEmptyRoot`

### List Command Tests
- `TestRunBundleListEmptyRoot`
- `TestRunBundleListShowsCreatedBundles`
- `TestRunBundleListJSONOutput`
- `TestRunBundleListIgnoresNonBundles`
- `TestRunBundleListSkipsInvalidBundles`

### Show Command Tests
- `TestRunBundleShowDisplaysMetadata`
- `TestRunBundleShowJSONOutput`
- `TestRunBundleShowRejectsInvalidID`
- `TestRunBundleShowRejectsMissingMetadata`
- `TestRunBundleShowRejectsSchemaMismatch`
- `TestRunBundleShowRejectsRunIDMismatch`

### Boundary Tests
- `TestRunBundleCLIDoesNotImportRuntimePackages`

## Verification Commands and Results

```bash
# Focused tests
go test ./cmd/leamas/... -run 'RunBundle' -count=1 -v

# Full test suite
go test ./...

# Vet check
go vet ./...

# Build
CGO_ENABLED=0 go build -trimpath -o bin/leamas ./cmd/leamas

# Factory checks
make factorize
make gate
```

## Skipped / Deferred

- **Run ID generation**: Not added; CLI requires explicit `--id`
- **Delete command**: Not added (read-only except create)
- **Edit/mutation command**: Not added (read-only except create)
- **Witness proxy persistence wiring**: Not added
- **Cockpit UI**: Not added
- **Database persistence**: Not added
- **Network behavior**: Not added

## Hard Stops Honored

| Requirement | Status |
|-------------|--------|
| No Python added | ✅ |
| No shell verifier logic added | ✅ |
| No Node/Vite/React/npm/yarn/pnpm added | ✅ |
| No database imports added | ✅ |
| No network imports added | ✅ |
| No cockpit imports added | ✅ |
| No witness proxy imports added | ✅ |
| No delete/edit command added | ✅ |
| No background daemon added | ✅ |
| No browser behavior added | ✅ |

## Not Added

- Run ID generation: CLI requires explicit `--id`
- Witness proxy persistence: Not in scope
- Cockpit UI: Not in scope
- Delete/mutation commands: Read-only except create

## Follow-up Candidates

| ACT | Description | Priority |
|-----|-------------|----------|
| `ACT-LEAMAS-WITNESS-CLAIM-EVIDENCE-SEED01` | Seed claim/evidence domain models | P1 |
| `ACT-LEAMAS-WITNESS-PROXY-INSPECT-CLI01` | CLI to inspect witness proxy captures | P1 |
| `ACT-LEAMAS-WEB-RUN-BUNDLE-LIST01` | Web cockpit run bundle list UI | P2 |
| `ACT-LEAMAS-HULK-RUN-BUNDLE-CORE01` | Hulk run bundle core integration | P2 |

Recommended next: `ACT-LEAMAS-WITNESS-CLAIM-EVIDENCE-SEED01`

Reason: Once bundles can be created/listed/shown, the next durable evidence primitive is typed claims and evidence links.

## Suggested Commit

```bash
git add cmd/leamas/run_bundle.go \
      cmd/leamas/run_bundle_test.go \
      cmd/leamas/witness.go \
      docs/factory/run-bundles.md \
      docs/close-reports/ACT-LEAMAS-WITNESS-RUN-BUNDLE-CLI01.md

git commit -m "ACT-LEAMAS-WITNESS-RUN-BUNDLE-CLI01 add run bundle CLI"
git push
```
