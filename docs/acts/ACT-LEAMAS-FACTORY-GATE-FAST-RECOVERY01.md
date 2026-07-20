# ACT-LEAMAS-FACTORY-GATE-FAST-RECOVERY01

## Summary

Hotfix to eliminate dupcode execution from the developer fast lane and remove nested full-registry execution from `TestRunFactorize`.

## Problem

`make gate-fast` was timing out because:
1. `RunGateFast` executed all verifiers including expensive dupcode verifiers
2. `TestRunFactorize` called `AllVerifiers()` against the live repository, causing nested full-scan execution

## Solution

### 1. Added typed verifier lane metadata

```go
type VerifierLane string

const (
    VerifierLaneFast    VerifierLane = "fast"
    VerifierLaneDupcode VerifierLane = "dupcode"
)

type Verifier struct {
    Name      string
    Run       func(root string) []checks.Finding
    Lane      VerifierLane  // NEW: lane assignment
    Execution ExecutionDefinition
    Cache     CacheSemantics
}
```

### 2. Assigned lanes to verifiers

- `dupcode` → `VerifierLaneDupcode`
- `dupcode-baseline` → `VerifierLaneDupcode`
- All other verifiers → `VerifierLaneFast`

### 3. Added lane filtering functions

```go
func FastVerifiers() []Verifier
func DupcodeVerifiers() []Verifier
```

### 4. Refactored RunGateFast

- Runs only fast-lane verifiers
- Reports explicit SKIP messages for dupcode-lane verifiers
- Output:
  ```
  dupcode: SKIP: expensive verifier lane; run make gate-dupcode
  dupcode-baseline: SKIP: expensive verifier lane; run make gate-dupcode
  ```

### 5. Added dedicated gate-dupcode lane

```bash
make gate-dupcode  # Runs exactly dupcode + dupcode-baseline
```

### 6. Removed nested full-scan from TestRunFactorize

- Replaced live `AllVerifiers()` call with fixture verifiers
- Test now completes in milliseconds
- Added lane validation tests: `TestVerifierLanes`, `TestSelectVerifiers`

### 7. Updated CLI

Added `--lane` flag to `factory gate`:
```bash
leamas factory gate --lane=fast    # Fast lane only
leamas factory gate --lane=dupcode # Dupcode lane only
```

## Files Changed

| File | Change |
|------|--------|
| `internal/factory/gate/gate.go` | Added `VerifierLane` type, `FastVerifiers()`, `DupcodeVerifiers()`, `RunGateDupcode()`, updated `RunGateFast` |
| `internal/factory/gate/verifiers.go` | Added `Lane` field to all verifier definitions |
| `internal/factory/gate/gate_test.go` | Removed live-repo `TestRunFactorize`, added lane validation tests, added fixture-based `TestRunFactorizeFixtures` |
| `cmd/leamas/factory.go` | Added `--lane` flag parsing and lane dispatch |
| `make/long-tests.mk` | Added `gate-dupcode` target |
| `Makefile` | Added `gate-dupcode` to PHONY targets and help |

## Commands Run

```bash
# Focused tests
go test ./internal/factory/gate/... -run 'TestRunFactorize|TestVerifierLane|TestSelectVerifiers'

# Build and gate-fast verification
CGO_ENABLED=0 go build -trimpath -o bin/leamas ./cmd/leamas
./bin/leamas factory gate --lane=fast
```

## Results

- ✅ `make gate-fast` does not execute dupcode or dupcode-baseline
- ✅ `TestRunFactorize` does not use `AllVerifiers()` or scan live repository
- ✅ `gate-fast` reports exactly two expensive-verifier skips
- ✅ `make gate-dupcode` executes both duplicate-code verifiers
- ✅ All focused gate tests pass (0.007s)

## Notes

The `go test -short ./...` in the toolchain fast mode still takes time. This is a separate concern from the dupcode lane issue - the original task was to eliminate dupcode from the fast developer loop. CI can parallelize `Factory Fast`, `Factory Dupcode`, and `Factory Long` as separate required jobs.
