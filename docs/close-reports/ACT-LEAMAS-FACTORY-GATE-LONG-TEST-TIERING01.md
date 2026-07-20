# Close Report: ACT-LEAMAS-FACTORY-GATE-LONG-TEST-TIERING01

## Files Changed

- `.factory/long-tests-baseline.json` - calibrated ci_timeout from 30m to 15m
- `.github/workflows/factory.yml` - reduced job timeout from 35m to 30m

## Behavior Changed

CI budget is now calibrated to actual observed runtime with appropriate margin.

## Exact Commands Run

```bash
# Isolated test measurement
git status --short && git rev-parse HEAD
# Output: e1a3fa5535fdb82d15e683699875272363872f8e

mkdir -p /tmp/leamas-long-test-evidence

/usr/bin/time -v \
  go test \
    -count=1 \
    -timeout=90m \
    -run '^TestV4BaselineDelta_LiveTreeMatchesCommittedBaseline$' \
    ./internal/factory/dupcode \
  2>&1 | tee \
    /tmp/leamas-long-test-evidence/dupcode-live-tree.log
```

## Results

- **Test duration**: 4:18 (wall clock)
- **User time**: 346.80s
- **System time**: 7.99s
- **Maximum RSS**: 119,368 KB
- **Exit code**: 0 (PASS)

## Decision Rule Applied

```
registered ci_timeout = max(observed duration + 10 minutes, observed duration × 1.5)
                      = max(4.3 + 10, 4.3 × 1.5)
                      = max(14.3, 6.45)
                      = 15 minutes

Factory Long job timeout = test budget + job allowance
                          = 15 + 15
                          = 30 minutes
```

## Skipped or Deferred

None.

## Follow-up ACTs

None.

## Host Identity

Local development machine (x86_64 Linux).
