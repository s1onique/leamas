# Execution Bounds

This document describes the runtime bounds enforced by the Leamas execution gateway (`internal/execution`).

## Overview

The execution gateway ensures all external process execution is bounded by finite limits on deadline, concurrency, total starts, logical depth, retained output, process-tree lifetime, and post-termination waiting.

## Default and Hard Maximum Bounds

| Dimension                         |     Default |                          Hard maximum |
| --------------------------------- | ----------: | ------------------------------------: |
| Concurrent external commands      |           4 |                                    16 |
| Start attempts per executor       |          64 |                                 1,024 |
| Logical task depth                |           8 |                                    32 |
| Command timeout                   | 120 seconds |                            10 minutes |
| Root execution deadline           | 120 seconds |                            10 minutes |
| Termination grace                 |   2 seconds |                            10 seconds |
| Post-`SIGKILL` wait              |    1 second |                             5 seconds |
| Combined retained stdout + stderr |       8 MiB |                                64 MiB |
| Go package parallelism, `-p`      |           4 |                  executor concurrency |
| Go test parallelism, `-parallel`  |           4 |                  executor concurrency |
| `GOMAXPROCS`                      |           4 |                  executor concurrency |
| Automatic retries                 |           0 | 0 unless explicitly implemented later |

## Effective Deadline Precedence

When executing a command, the effective deadline is computed as:

```
effective_deadline = min(
    parent context deadline (when present),
    Budget.Deadline,
    now + Request.Timeout
)
```

Rules:
1. `Request.Timeout == 0` means `DefaultTimeout`.
2. Negative timeouts are invalid and rejected.
3. Timeouts greater than `MaxPermittedTimeout` are rejected before starting.
4. A zero or expired `Budget.Deadline` is rejected.
5. If the effective deadline has already expired, no semaphore is acquired and no start counter is incremented.
6. Semaphore waiting, process startup, execution, graceful termination, forced termination, and I/O draining all count toward the root execution lifetime.

## Error Codes

The following codes distinguish failure causes:

| Code | Meaning |
|------|---------|
| `execution_cancelled` | Caller cancelled the context |
| `execution_timeout_exceeded` | Request timeout expired |
| `execution_deadline_exceeded` | Root deadline expired |
| `execution_task_depth_exceeded` | Task depth exceeds configured limit |
| `execution_output_limit_exceeded` | Combined output exceeded the cap |
| `execution_process_tree_cleanup_failed` | Process tree cleanup failed |
| `execution_concurrency_exhausted` | Concurrency limit reached |
| `execution_start_budget_exhausted` | Total start attempts exhausted |
| `nested_leamas_execution` | Attempted Leamas within Leamas |
| `execution_cycle_detected` | Logical command cycle detected |

## Concurrency Control

The executor uses a context-cancellable semaphore to bound concurrent commands:

- Uses a channel-based implementation with bounded capacity
- Acquisition waits until a slot is available or the context is cancelled
- Waiting is immediately interrupted when the context is cancelled
- No release operation is required to wake a cancelled waiter
- No goroutines leak when cancellation occurs

## Start Budget Semantics

- The start counter increments only after semaphore acquisition
- A failed `cmd.Start()` still counts against the budget
- Waiting for a slot does NOT count as a start
- The counter is cumulative across all commands in an executor's lifetime

## Task Depth

Task depth is an explicit field on execution requests:

- Zero means root command depth `1`
- Depth greater than `Budget.MaxTaskDepth` is rejected before semaphore acquisition
- Child orchestration must increment depth rather than reset it
- Depth is independent of active-command count

## Combined Output Semantics

The output limit applies to the combined size of stdout and stderr:

- Observed bytes count all output, including discarded bytes
- Retained bytes are stored in memory (limited to the cap)
- `OutputTruncated` is set immediately when observed output exceeds the cap
- Overflow triggers process-tree termination
- Commands that exceed output limits are never reported as successful

## Unix Process Group Cleanup

On Unix, every command runs in its own process group (`Setpgid: true`).

On timeout, cancellation, or output overflow:
1. Send `SIGTERM` to the process group
2. Wait up to `TerminationGrace`
3. If the group still exists, send `SIGKILL` to the process group
4. Wait up to `PostKillWait`
5. Verify the process group no longer exists

Benign conditions (successful cleanup):
- Process group already absent
- `ESRCH` while signalling or probing

`WaitDelay` is set to prevent blocking on inherited I/O descriptors.

## Windows Fail-Closed Behavior

Windows compilation is supported but execution is disabled:
- `Execute()` returns `execution_unknown` error
- No process is started
- The same public/internal method set is exposed as Unix

## Go Toolchain Hard Bounds

Every request created by `GoAdapter` is bounded even when the caller supplies no flags.

For `go build`, `go vet`, `go list`:
- `-p <executor MaxConcurrent>` is always present
- `GOMAXPROCS` is clamped in the environment

For `go test`:
- `-p <executor MaxConcurrent>` is always present
- `-parallel <executor MaxConcurrent>` is always present
- `-timeout <finite non-zero duration>` is always present
- `GOMAXPROCS` is clamped in the environment

Normalization rules:
- Missing values are added
- Unparsable values are rejected
- Zero and negative values are replaced
- Values above the limit are clamped
- Smaller valid values are preserved
- Duplicate forms are normalized to exactly one

## Make Hard Bounds

The `MakeAdapter` normalizes:
- `-j`, `-j0`, `-jN`, `-j=N`, `--jobs`, `--jobs=N`
- Bare or zero job counts become the configured finite limit
- Larger counts are clamped
- Smaller positive counts are preserved
- `MAKEFLAGS` and `MFLAGS` are sanitized

## Re-Entry Metadata

Every child command receives:
- `LEAMAS_EXEC_ROOT_ID=<root ID>`
- `LEAMAS_EXEC_PARENT_PID=<current Leamas PID>`
- `LEAMAS_EXEC_GENERATION=<parent generation + 1>`

Generation overflow is handled safely (wraps to max uint32).

Root IDs are unique using PID + cryptographic randomness.

## Prohibition on Direct Process APIs

The following are forbidden outside the execution gateway:

```go
os/exec.Command
os/exec.CommandContext
os.StartProcess
syscall.ForkExec
syscall.Exec
```

Allowed files (low-level platform implementations):
- `internal/execution/executor_unix.go`
- `internal/execution/executor_windows.go`
- `internal/execution/process_unix.go`

## Budget Validation

`Budget.Validate(now time.Time) error` rejects:
- Nil budgets
- Past deadlines
- Non-positive concurrency
- Non-positive starts
- Non-positive task depth
- Non-positive output limit
- Non-positive termination grace
- Values exceeding hard maxima
- Arithmetic overflow
