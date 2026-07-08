# Close Report: ACT-LEAMAS-FACTORY-PREVENT-FORCE-PUSH-GO-VERIFY01

Date: 2026-07-08

## Summary

Added local Git safety rails against force-pushes with hook and installer as tiny Bash glue and hook verifier implemented in Go.

## Files Changed

### Created

- `githooks/pre-push` - Bash pre-push hook (24 LOC)
- `scripts/install_git_hooks.sh` - Bash installer (6 LOC)
- `internal/factory/githooks/check.go` - Go verifier
- `internal/factory/githooks/check_test.go` - Verifier tests
- `internal/factory/githooks/hook_functional_test.go` - Functional pre-push hook tests
- `docs/factory/git-safety.md` - Documentation

### Modified

- `cmd/leamas/main.go` - Added `git-hooks` verify command
- `Makefile` - Added `verify-git-hooks` and `install-git-hooks` targets
- `scripts/quality_gate.sh` - Added required files and Git hooks verification
- `docs/factory/tooling-boundaries.md` - Documented allowed Bash glue
- `docs/doctrine/go-only.md` - Added verifier rule
- `AGENTS.md` - Added verifiers section
- `.clinerules/leamas.md` - Added verifier rule
- `docs/doctrine/agent-assisted-development.md` - Added verifier rule

## Behavior Changed

- `leamas factory verify git-hooks` now verifies:
  - `githooks/pre-push` exists and is executable
  - `scripts/install_git_hooks.sh` exists and is executable
  - `core.hooksPath` equals `githooks`
  - Hook contains protected branch refs and merge-base check
  - No `scripts/verify_git_hooks.sh` exists (Bash verifier forbidden)
  - Both Bash files are ≤50 meaningful LOC

## Exact Commands Run

```bash
chmod +x githooks/pre-push
chmod +x scripts/install_git_hooks.sh
go test ./...
go vet ./...
CGO_ENABLED=0 go build -trimpath -o bin/leamas ./cmd/leamas
./bin/leamas factory verify git-hooks
make verify-git-hooks
make factorize
make gate
```

## Honest Results

| Command | Result |
|---------|--------|
| `go test ./...` | PASSED |
| `go vet ./...` | PASSED |
| `CGO_ENABLED=0 go build` | PASSED |
| `leamas factory verify git-hooks` | PASSED |
| `make verify-git-hooks` | PASSED |
| `make verify-tooling-boundaries` | PASSED |
| `make verify-agent-context` | PASSED |
| `make verify-llm-friendly` | PASSED |
| `make factorize` | PASSED |
| `make gate` | PASSED |

## Skipped or Deferred

- None

## Follow-Up

- **ACT-LEAMAS-FACTORY-GO-VERIFIERS01**: Migrate remaining grandfathered Bash verifiers into Go
