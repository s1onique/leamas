# Close Report: ACT-LEAMAS-FACTORY-DIGEST-SMART-DEFAULTS01

## ACT Summary

**Title**: Add smart digest defaults  
**Status**: Closed  
**Date**: 2026-07-08

## Goal

Make `leamas factory digest --output <path>` the default recommended command with smart behavior:

- If working tree has changes → dirty digest
- If working tree is clean → `HEAD~1..HEAD` commit range digest

## Files Changed

1. **internal/factory/digest/digest.go** - Core digest generation with auto mode logic
2. **internal/factory/digest/resolve.go** - Auto mode resolution (Mode types, ResolveAutoMode)
3. **internal/factory/digest/range.go** - Range digest support (GetRangeFiles, RenderRangeDigest)
4. **internal/factory/digest/digest_auto_test.go** - Tests for auto mode behavior
5. **cmd/leamas/main.go** - CLI with auto mode, --range flag, mutually exclusive validation
6. **Makefile** - Updated `make digest` to use smart defaults
7. **docs/factory/digest.md** - Updated documentation with smart defaults explanation

## Behavior Changed

### Before
```bash
leamas factory digest --dirty --output build/digest.txt  # Required --dirty
leamas factory digest --staged --output build/staged.txt  # Required --staged
```

### After
```bash
leamas factory digest --output build/digest.txt  # Smart auto mode (default)
leamas factory digest --dirty --output build/digest.txt  # Explicit still works
leamas factory digest --staged --output build/staged.txt  # Explicit still works
leamas factory digest --range HEAD~1..HEAD --output build/range.txt  # New range mode
```

### Smart Auto Mode Output
```markdown
Mode: dirty
Resolved from: auto
Reason: working tree has changes
```
or
```markdown
Mode: range
Range: HEAD~1..HEAD
Resolved from: auto
Reason: working tree clean; showing previous commit
```

### Error Handling
- `--dirty` + `--staged` → error
- `--dirty` + `--range` → error
- `--staged` + `--range` → error
- Clean repo with only initial commit → honest error message

## Technical Implementation

### Mode Types
```go
ModeAuto   Mode = "auto"   // New default
ModeDirty  Mode = "dirty"  // Existing
ModeStaged Mode = "staged" // Existing
ModeRange  Mode = "range"  // New
```

### Auto Resolution Logic
1. Check `git diff --cached --quiet` for staged changes
2. Check `git diff --quiet` for unstaged changes
3. Check `git ls-files --others --exclude-standard` for untracked files
4. If dirty → ModeDirty with reason "working tree has changes"
5. If clean → ModeRange with `HEAD~1..HEAD`, reason "working tree clean; showing previous commit"

### NUL-Delimited File Parsing
All Git file list commands now use `-z` flag with `splitNULList()` to handle filenames with spaces correctly.

## Exact Commands Run

```bash
go test ./...                                    # All tests pass
go vet ./...                                     # No issues
gofmt -l .                                      # Clean
CGO_ENABLED=0 go build -trimpath -o bin/leamas ./cmd/leamas  # Builds

# Functional verification
./bin/leamas factory digest --output build/leamas-auto-digest.txt
./bin/leamas factory digest --dirty --output build/leamas-dirty-digest.txt
./bin/leamas factory digest --dirty --staged --output build/invalid.txt  # Expected failure

make digest                                     # Uses smart defaults
make factorize                                  # All verifiers pass
make gate                                       # Full quality gate passes
```

## Verification Results

| Check | Result |
|-------|--------|
| go test ./... | PASSED |
| go vet ./... | PASSED |
| gofmt -l . | Clean |
| Static build | PASSED |
| factorize | PASSED |
| gate | PASSED |
| Smart auto mode (dirty tree) | PASSED |
| Smart auto mode (clean tree) | PASSED |
| Explicit --dirty | PASSED |
| Explicit --staged | PASSED |
| Mutually exclusive error | PASSED |
| Files with spaces | PASSED |
| Initial commit error | PASSED |

## Example Output

### Auto Dirty Mode
```
# Targeted digest

Generated at: 2026-07-08T18:28:07Z
Repo: /Volumes/UserData/Users/chistyakov/Projects/SPbNIX/leamas
Mode: dirty
Resolved from: auto
Reason: working tree has changes

## Changed files
Makefile  [tracked, staged present: no, unstaged present: yes]
...
```

### Auto Clean Mode (Previous Commit)
```
# Targeted digest

Generated at: 2026-07-08T18:30:00Z
Repo: /path/to/repo
Mode: range
Range: HEAD~1..HEAD
Resolved from: auto
Reason: working tree clean; showing previous commit

## Changed files
file2.txt  [modified]
...
```

## Skipped/Deferred

- Initial commit fallback with empty tree hash (deferred to follow-up)
- Configurable range anchors (out of scope)
- Multi-commit history summaries (out of scope)

## Follow-up ACTs

- ACT-LEAMAS-FACTORY-FORBIDDEN-SCOPE-CONTRACT01 (suggested)

## Notes

- Wrapper script `scripts/make_targeted_digest.sh` requires no changes - delegates to Go
- LLM-friendliness gate required splitting digest.go into multiple files:
  - digest.go (core + file discovery)
  - resolve.go (auto mode resolution)
  - range.go (range digest support)
  - digest_auto_test.go (smart mode tests)
