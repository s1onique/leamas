# Close Report: ACT-LEAMAS-FACTORY-GO-DIGEST01

## Summary

Ported `scripts/make_targeted_digest.sh` from substantial Bash (262 LOC) to a reusable Leamas Go command. The targeted digest is now a first-class Leamas Factory capability usable across all Factory-managed projects.

## Files Changed

### New Files
- `internal/factory/digest/digest.go` - Core digest types and generation logic
- `internal/factory/digest/git.go` - Git operations wrapper
- `internal/factory/digest/preview.go` - File preview and binary detection
- `internal/factory/digest/digest_integration_test.go` - Comprehensive tests
- `docs/factory/digest.md` - Factory digest documentation

### Modified Files
- `cmd/leamas/main.go` - Added `leamas factory digest` command
- `scripts/make_targeted_digest.sh` - Reduced from 262 LOC to 19 LOC tiny wrapper
- `Makefile` - Added `make digest` target
- `docs/factory/tooling-boundaries.md` - Updated to reflect digest migration
- `docs/doctrine/agent-assisted-development.md` - Added digest to verification hooks

## Behavior Changed

- `leamas factory digest --dirty` now generates digests via Go instead of Bash
- `leamas factory digest --staged` now generates staged-only digests via Go
- Digest output includes: timestamp, repo root, mode, changed files with metadata, diffs/previews, workflow anchors
- Ignored files are excluded from digest
- Binary files are summarized instead of previewed
- Output directory is created automatically if needed

## Commands Run

```bash
# Build and test
CGO_ENABLED=0 go build -trimpath -o bin/leamas ./cmd/leamas
go test ./...
go vet ./...

# Manual verification
./bin/leamas factory digest --dirty --output build/leamas-go-digest-dirty.txt
./bin/leamas factory digest --staged --output build/leamas-go-digest-staged.txt
scripts/make_targeted_digest.sh --dirty --output build/leamas-wrapper-digest.txt
make digest
make factorize
make gate
```

## Verification Results

All verification commands passed:

```bash
go test ./...                          # PASSED
go vet ./...                           # PASSED
gofmt -l .                            # OK (no output)
CGO_ENABLED=0 go build -o bin/leamas  # PASSED
make factorize                        # PASSED
make gate                             # PASSED
./bin/leamas factory digest --dirty   # WORKING
./bin/leamas factory digest --staged  # WORKING
scripts/make_targeted_digest.sh       # WORKING (tiny 19 LOC wrapper)
```

## Skipped/Deferred

None.

## Decisions Made

- Digest logic lives in `internal/factory/digest/` package
- Public API uses boring, testable types (Mode, Options, ChangedFile)
- File preview capped at 16 KiB / 200 lines for LLM-friendliness
- Binary files are summarized instead of previewed
- Bash wrapper reduced from 262 LOC to 19 LOC (92% reduction)

## Follow-up ACTs

- **ACT-LEAMAS-FACTORY-GO-VERIFIERS01** - Continue migrating remaining Factory verifier/gate logic into Go (already in progress)

## Notes

- The digest command works from any Git repository, not just Leamas
- Digest output should be written to `build/` or ignored artifact directories
- Bash wrapper reduced from 262 LOC to 19 LOC (92% reduction)
- All substantial digest logic now lives in Go under `internal/factory/digest/`
