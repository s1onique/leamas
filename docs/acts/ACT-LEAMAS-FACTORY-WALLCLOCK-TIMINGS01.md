# ACT-LEAMAS-FACTORY-WALLCLOCK-TIMINGS01

## Goal

Add wall-clock durations to the `leamas factory factorize` text progress
output so operators can see which checks dominate the run, while keeping
timings out of every deterministic / JSON / evidence / fingerprint
contract.

## Acceptance criteria

1. Every factorize check prints elapsed wall-clock duration.
2. Successful and failed checks use the same format.
3. Overall factorize duration is printed on success and failure.
4. Durations use seconds with exactly two decimal places.
5. Execution order and exit codes remain unchanged.
6. JSON and deterministic evidence contracts remain unchanged.
7. Tests use an injected clock and contain no sleeps.
8. `make factorize`, `make gate`, and `go test ./...` remain green.

## Scope

Do **not** add performance thresholds in this ACT. First collect
comparable Mint and Mac measurements; a subsequent ratchet can identify
regressions without guessing acceptable limits.

## Implementation boundary

Instrument the shared check-execution boundary (`runCheck`,
`runFactorize` in `internal/factory/gate/factorize.go`) rather than
adding timing inside every verifier. The public `RunFactorize(root)`
becomes a thin wrapper that injects the system clock and writes to
`os.Stdout`; tests inject a fake clock and a `bytes.Buffer`.
