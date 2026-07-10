# ACT-LEAMAS-EXECUTION-HARD-BOUNDS-RUNTIME01-R4: Adversarial Test Process Tree Fix

## Summary

R4 corrections for adversarial process tree proof tests. Fixed the `sleep-grandchild` mode to properly create a 3-level blocking process tree that can be terminated by the executor.

## Problem Statement

After R3 was committed, an expert review identified that the test helper's `sleep-grandchild` mode had a critical bug:

1. **Waiting bug**: Parent called `cmd.Wait()` on child, causing premature exit when grandchild was killed
2. **Deadlock detection**: Go's runtime detected `select{}` as a deadlock and terminated the process
3. **Manifest not synced**: Child processes weren't flushing their PID records to disk before parent exited

## Root Cause Analysis

1. **Waiting bug**: The `sleep-grandchild-child` case called `cmd.Wait()` on its child, waiting for it to exit. When the executor killed the grandchild with SIGKILL, the child exited, causing the parent to also exit (no more work to do).

2. **Go deadlock**: `select{}` with no cases triggers Go's runtime deadlock detection when there are no other goroutines doing work.

3. **File sync**: Children opened, wrote, and closed the manifest file, but the OS might not have flushed to disk before the parent killed them.

## Fixes Applied

### 1. Removed wait() calls in sleep-grandchild modes

```go
// Before (buggy)
spawnChild("sleep-grandchild-child")
cmd.Wait()  // <-- This causes premature exit

// After (fixed)
spawnChild("sleep-grandchild-child")  // Don't wait - let all processes run independently
```

### 2. Fixed Go deadlock detection

```go
// Before (triggers deadlock)
func sleepForever() {
    select {}  // Go runtime detects this as deadlock
}

// After (no deadlock)
func sleepForever() {
    for {
        time.Sleep(24 * time.Hour)  // Has work to do, no deadlock
    }
}
```

### 3. Added explicit file sync

```go
func recordPID(role string, mode string) {
    // ... write record ...
    
    // Sync to ensure data is flushed before process continues
    if err := f.Sync(); err != nil {
        // error handling
    }
    if err := f.Close(); err != nil {
        // error handling
    }
}
```

### 4. Added startup delays for grandchildren

```go
case "sleep-grandchild-child":
    recordPID("child", mode)
    time.Sleep(10 * time.Millisecond)  // Allow time for grandchild to start
    spawnChild("sleep-grandchild-grandchild")
    time.Sleep(10 * time.Millisecond)  // Allow time for grandchild to record
    sleepForever()
```

### 5. Removed duplicate locateHelperBinary function

Removed the duplicate `locateHelperBinary` from `adversarial_harness_parse.go` and updated callers to use `getHelperPath()` from the executor package.

### 6. Added gitignore exception for testhelper source

Added exception to `.gitignore` to track the adversarial test helper source file:

```
!internal/execution/testdata/testhelper/main.go
```

## Files Changed

- `internal/execution/testdata/testhelper/main.go` - New file with fixed test helper
- `internal/execution/adversarial_cancel_test.go` - Updated cancel delay timing
- `internal/execution/adversarial_harness_executor.go` - Fixed duplicate function, improved auto-build
- `internal/execution/adversarial_harness_parse.go` - Removed duplicate locateHelperBinary
- `internal/execution/adversarial_harness_types.go` - Added expected roles
- `internal/execution/adversarial_output_test.go` - Minor cleanup
- `internal/execution/adversarial_timeout_test.go` - Updated for macOS behavior
- `.gitignore` - Added exception for testhelper source

## Test Results

All adversarial tests pass:

```
=== RUN   TestAdversarialCallerCancellation
    adversarial_cancel_test.go:72: TestAdversarialCallerCancellation: PASSED - elapsed 1.522781416s, records=3
--- PASS: TestAdversarialCallerCancellation (1.53s)

=== RUN   TestAdversarialNonZeroExitWithChild
    adversarial_cancel_test.go:127: TestAdversarialNonZeroExitWithChild: PASSED - exit code 42, records=2
--- PASS: TestAdversarialNonZeroExitWithChild (0.04s)

=== RUN   TestAdversarialProcessGroupIsolation
--- PASS: TestAdversarialProcessGroupIsolation (0.42s)

=== RUN   TestAdversarialOutputOverflowWithDescendants
--- PASS: TestAdversarialOutputOverflowWithDescendants (0.09s)

=== RUN   TestAdversarialIgnoreSIGTERMViaGoHelper
--- PASS: TestAdversarialIgnoreSIGTERMViaGoHelper (0.93s)

=== RUN   TestAdversarialHeldOutputDescriptor
--- PASS: TestAdversarialHeldOutputDescriptor (0.82s)

=== RUN   TestAdversarialTimeoutDirectSleep
--- PASS: TestAdversarialTimeoutDirectSleep (0.21s)

=== RUN   TestAdversarialTimeoutChildTree
--- PASS: TestAdversarialTimeoutChildTree (0.51s)

=== RUN   TestAdversarialTimeoutGrandchildTree
--- PASS: TestAdversarialTimeoutGrandchildTree (1.22s)
```

## Commands Run

```bash
# Build helper
go build -o internal/execution/testdata/testhelper/main internal/execution/testdata/testhelper/main.go

# Run adversarial tests
go test ./internal/execution/... -run TestAdversarial -v -count=1

# Stress tests
go test ./internal/execution/... -run TestAdversarial -count=5

# Coverage check
go test ./... -covermode=atomic -coverprofile coverage.out
go run ./cmd/leamas factory coverage --profile coverage.out --min-total 64

# Verification
make factorize
go vet ./...
go test ./...
```

## Behavior Changed

- **sleep-grandchild**: Now creates a proper 3-level blocking process tree (parent → child → grandchild) that stays alive until killed by the executor
- **manifest records**: All 3 PIDs are recorded and synced to disk before any process is terminated
- **timeout tests**: Now properly verify that a 3-level process tree is created and all processes are terminated on timeout

## Verification Status

- [x] `go test ./internal/execution/...` - All tests pass
- [x] `go test ./...` - All tests pass
- [x] `go vet ./...` - No issues
- [x] `gofmt` - All files formatted
- [x] `make factorize` - All checks pass
- [x] `go build` - Static build succeeds
- [x] Stress tests (5 iterations) - All pass

## Follow-up Actions

- Gate CI status check needs to pass on remote
- Consider adding stress tests to the standard test suite (currently manual)
