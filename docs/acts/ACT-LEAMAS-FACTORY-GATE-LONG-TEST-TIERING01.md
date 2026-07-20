# ACT-LEAMAS-FACTORY-GATE-LONG-TEST-TIERING01

## Status: CLOSED

## Objective

Calibrate the dupcode long-test CI budget based on isolated runtime measurement.

## Runtime Evidence

| Metric | Value |
|--------|-------|
| Test | `TestV4BaselineDelta_LiveTreeMatchesCommittedBaseline` |
| Package | `./internal/factory/dupcode` |
| Observed duration | 4:18 (4.3 minutes) |
| Exit code | 0 (PASS) |
| Maximum RSS | 119,368 KB (~116 MB) |
| CPU utilization | 137% |
| GOMAXPROCS | (host default) |

### Calibration Calculation

- Observed duration: 4.3 minutes
- Registered ci_timeout = max(4.3 + 10, 4.3 × 1.5) = max(14.3, 6.45) = **15 minutes**
- Factory Long job timeout = 15 + 15 = **30 minutes**

## Changes Made

| File | Before | After |
|------|--------|-------|
| `.factory/long-tests-baseline.json` | `ci_timeout: "30m"` | `ci_timeout: "15m"` |
| `.github/workflows/factory.yml` | `timeout-minutes: 35` | `timeout-minutes: 30` |

## Next Steps

The suspended factorize baseline correction should resume immediately after this evidence closes.

## Closed by

ACT-LEAMAS-FACTORY-GATE-LONG-TEST-TIERING01 close report.
