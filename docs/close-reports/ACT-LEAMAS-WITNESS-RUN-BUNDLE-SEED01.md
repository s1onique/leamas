# Close Report: ACT-LEAMAS-WITNESS-RUN-BUNDLE-SEED01

## Summary

Introduced the `internal/witness/runbundle` package providing a minimal
run bundle core: typed metadata, safe filesystem layout, JSON writing/loading,
deterministic path validation, and comprehensive tests. This is the filesystem
foundation for Leamas's durable local evidence substrate.

## Files Changed

| File | Change |
|------|--------|
| `internal/witness/runbundle/metadata.go` | Types, SchemaVersion, NewMetadata, MarshalJSON, StrictDecode |
| `internal/witness/runbundle/path.go` | ValidateRunID, BundlePath, and all validation errors |
| `internal/witness/runbundle/bundle.go` | Create, Open, Bundle struct, CreateOptions, subdirs |
| `internal/witness/runbundle/bundle_test.go` | 14 tests for Create and Open operations |
| `internal/witness/runbundle/path_test.go` | 8 tests for path validation and RunID safety |
| `docs/factory/run-bundles.md` | Factory documentation for run bundles |
| `docs/close-reports/ACT-LEAMAS-WITNESS-RUN-BUNDLE-SEED01.md` | This close report |

## API Added

### Types
```go
type RunID string
type SchemaVersion = "leamas.runbundle.v1"
type Bundle struct { Root, ID, Path string }
type Metadata struct { SchemaVersion, RunID, CreatedAt, Tool, Doctrine }
type ToolInfo struct { Name, Version }
type Doctrine struct { LocalOnly, ReadOnly, NoDatabase }
type CreateOptions struct { Root, RunID, Now, ToolName, Version }
```

### Functions
```go
func ValidateRunID(id RunID) error
func BundlePath(root string, id RunID) (string, error)
func Create(opts CreateOptions) (Bundle, error)
func Open(root string, id RunID) (Bundle, *Metadata, error)
func NewMetadata(runID RunID, now time.Time, toolName, toolVersion string) Metadata
func (m Metadata) MarshalJSON() ([]byte, error)
func StrictDecode(data []byte) (*Metadata, error)
```

### Errors
```go
var ErrEmptyRoot, ErrEmptyRunID, ErrRunIDTooLong, ErrRunIDNotLocal,
    ErrRunIDTraversal, ErrRunIDAbsolute, ErrRunIDReserved,
    ErrRunIDInvalidChar, ErrRunIDNoPrefix,
    ErrSchemaVersionMismatch, ErrRunIDMismatch,
    ErrMissingMetadata, ErrMetadataReadError, ErrMetadataDecodeError
```

## Layout Contract

`Create()` creates the following directory structure:

```
<root>/
  <run-id>/
    metadata.json        (0644)
    claims/              (0755)
    evidence/           (0755)
    digests/            (0755)
    traces/             (0755)
    verifier-results/   (0755)
```

## Tests Added

### Path Tests
- `TestValidateRunIDAcceptsSafeIDs` - Accepts valid run IDs
- `TestValidateRunIDRejectsUnsafeIDs` - Rejects invalid run IDs
- `TestValidateRunIDLength` - Enforces 128 char max length
- `TestBundlePathStaysUnderRoot` - Path stays under root
- `TestBundlePathRejectsTraversalID` - Rejects traversal
- `TestBundlePathRejectsAbsoluteID` - Rejects absolute paths
- `TestBundlePathRejectsEmptyRoot` - Rejects empty root
- `TestBundlePathRequiresValidRunID` - Validates run ID

### Bundle Tests
- `TestCreateBundleCreatesExpectedLayout` - Creates all subdirs
- `TestCreateBundleWritesMetadata` - Writes correct metadata.json
- `TestCreateBundleUsesDeterministicClock` - Uses injected time
- `TestCreateBundleRejectsUnsafeRunID` - Validates run ID
- `TestCreateBundleRejectsEmptyRoot` - Validates root
- `TestCreateBundleDoesNotCreateOutsideRoot` - Path safety
- `TestOpenBundleReadsMetadata` - Reads metadata correctly
- `TestOpenBundleRejectsUnknownMetadataFields` - Strict decode
- `TestOpenBundleRejectsSchemaVersionMismatch` - Schema validation
- `TestOpenBundleRejectsRunIDMismatch` - Run ID validation
- `TestOpenBundleRejectsMissingMetadata` - Missing file handling
- `TestOpenBundleRejectsEmptyRoot` - Empty root handling
- `TestOpenBundleRejectsInvalidRunID` - Invalid ID handling
- `TestMetadataJSONFormat` - JSON format validation

## Verification Commands and Results

```bash
# Focused tests
go test ./internal/witness/runbundle/... -v

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

- **CLI wiring** - Deferred to `ACT-LEAMAS-WITNESS-RUN-BUNDLE-CLI01`
- **Witness proxy persistence** - Not in scope for this seed
- **Cockpit UI** - Not in scope for this seed
- **Full claim/evidence modeling** - Not in scope for this seed
- **Trace capture storage** - Not in scope for this seed
- **Export/import archive format** - Not in scope for this seed
- **Symlink resolution** - Lexical path safety only in this seed

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
| No cmd/leamas imports | ✅ |

## No CLI Wiring Added

No CLI wiring was added in this ACT. The `internal/witness/runbundle`
package can be used programmatically by importing it.
CLI wiring is deferred to `ACT-LEAMAS-WITNESS-RUN-BUNDLE-CLI01`.

## Follow-up Candidates

| ACT | Description | Priority |
|-----|-------------|----------|
| `ACT-LEAMAS-WITNESS-RUN-BUNDLE-CLI01` | Add CLI to create/list/inspect local run bundles | P0 |
| `ACT-LEAMAS-WITNESS-CLAIM-EVIDENCE-SEED01` | Seed claim/evidence domain models | P1 |
| `ACT-LEAMAS-WITNESS-PROXY-INSPECT-CLI01` | CLI to inspect witness proxy captures | P1 |
| `ACT-LEAMAS-WEB-RUN-BUNDLE-LIST01` | Web cockpit run bundle list UI | P2 |
| `ACT-LEAMAS-HULK-RUN-BUNDLE-CORE01` | Hulk run bundle core integration | P2 |

## Suggested Commit

```bash
git add internal/witness/runbundle \
      docs/factory/run-bundles.md \
      docs/close-reports/ACT-LEAMAS-WITNESS-RUN-BUNDLE-SEED01.md

git commit -m "ACT-LEAMAS-WITNESS-RUN-BUNDLE-SEED01 add local run bundle seed"
git push
```
