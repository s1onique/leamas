# Factory: Go Coverage Gate

This document describes the Go test coverage measurement and threshold checking
infrastructure, including module-level breakdown.

## Overview

The coverage gate provides a way to measure and enforce minimum Go test coverage
thresholds. The current conservative ratchet threshold is **64%**.

## How to Run Coverage

### Quick Start

```bash
# Generate coverage profile, module breakdown, and check threshold
make coverage
```

This will:
1. Create the `.factory/` directory if needed
2. Run all tests with coverage instrumentation
3. Generate a coverage profile at `.factory/coverage.out`
4. Generate a JSON report at `.factory/coverage-summary.json`
5. Print total coverage and module breakdown
6. Check against the configured threshold (default: 64%)

### Output Format

**Terminal Output:**
```
Generating coverage profile...
coverage: total=66.6% min=64.0% OK
Coverage by module:
module                  coverage
cmd/leamas             52.0%
internal/factory       69.7%
internal/hulk          95.6%
internal/web           74.6%
internal/witness       85.4%
```

**JSON Output** (`.factory/coverage-summary.json`):
```json
{
  "schema_version": 2,
  "total_percent": 66.6,
  "total_covered": 2790,
  "total_statements": 4189,
  "modules": [
    {
      "module": "cmd/leamas",
      "percent": 52.0,
      "packages": 14,
      "covered_statements": 691,
      "total_statements": 1328
    },
    {
      "module": "internal/factory",
      "percent": 69.7,
      "packages": 30,
      "covered_statements": 1577,
      "total_statements": 2261
    }
  ]
}
```

### CLI Usage

```bash
# Check an existing coverage profile with module breakdown
leamas factory coverage --profile .factory/coverage.out --min-total 64.0

# Generate JSON output only (no terminal breakdown)
leamas factory coverage --profile .factory/coverage.out --min-total 64.0 \
  --json-output .factory/coverage-summary.json

# Hide module breakdown on terminal
leamas factory coverage --profile .factory/coverage.out --min-total 64.0 \
  --no-breakdown

# Check with a different threshold
leamas factory coverage --profile .factory/coverage.out --min-total 50.0
```

### Verify Coverage Status

```bash
# Run coverage verifier independently
leamas factory verify coverage
```

## Profile Locations

| File | Purpose |
|------|---------|
| `.factory/coverage.out` | Raw coverage profile from `go test -coverprofile` |
| `.factory/coverage-summary.json` | Machine-readable module breakdown |

These locations are:
- In `.gitignore`
- Local to the repository
- Not meant for external publishing

## Module Grouping

Leamas modules are defined as stable repo components, not Go modules in the `go.mod` sense.

### Module Mapping

| Import Path Prefix | Module Name |
|-------------------|------------|
| `github.com/s1onique/leamas/cmd/leamas` | `cmd/leamas` |
| `github.com/s1onique/leamas/internal/factory` | `internal/factory` |
| `github.com/s1onique/leamas/internal/hulk` | `internal/hulk` |
| `github.com/s1onique/leamas/internal/witness` | `internal/witness` |
| `github.com/s1onique/leamas/internal/web` | `internal/web` |
| (anything else) | `other` |

### Aggregation Semantics

Module percentages are computed using **statement-weighted aggregation** from the raw
Go coverage profile (`.factory/coverage.out`).

For each coverage block:
- `total_statements += numStatements`
- `covered_statements += numStatements` if `count > 0` (in atomic mode)

Module coverage = `covered_statements / total_statements * 100`

This is the exact statement-weighted coverage, not an approximation.

## Threshold Checking

### Output Format

**Pass:**
```
coverage: total=66.6% min=64.0% OK
```

**Fail:**
```
coverage: threshold_fail: total coverage 63.9% is below minimum 64.0%
```

### Current Threshold

The current conservative ratchet threshold is **64%** (raised from 60%).

**Threshold selection rule:**
- If total >= 67.0: threshold = 65
- If total >= 66.0: threshold = 64
- If total >= 65.0: threshold = 63
- Otherwise: threshold = 62

This means:
- The coverage gate actively enforces a minimum 64% coverage
- Headroom exists from the current ~66.6% measured coverage (2.6 percentage points)
- Module thresholds remain deferred for future ACTs

### Raising the Threshold

To raise the threshold, edit the Makefile:

```makefile
COVERAGE_MIN_TOTAL ?= 65
```

Or run with an override:

```bash
COVERAGE_MIN_TOTAL=65 make coverage
```

## Module Thresholds (Deferred)

Module-level thresholds are **not** enforced in this implementation.

Rationale:
- Adding visibility first before enforcement
- Module thresholds require careful consideration of what "good" means per module
- Teams need time to understand their module's coverage before being held accountable

A future ACT may add:
- Per-module minimum thresholds
- Module-level threshold failures in `make gate`
- Different thresholds for different modules

## Architecture

### Components

1. **`internal/factory/coverage/`** - Core coverage parsing and threshold logic
   - `Report` - Module breakdown report with JSON serialization
   - `Threshold` - Threshold configuration
   - `ModuleSummary` - Per-module coverage data
   - `ProfileReport` - Statement-weighted report with statement counts
   - `ParseProfile()` - Parse raw coverage profile for weighted aggregation
   - `CheckThreshold()` - Compare report against threshold
   - `Analyze()` - Full analysis pipeline: ParseProfile() -> statement-weighted report -> CheckThreshold()
   - `ClassifyModule()` - Map import path to module name
   - `ToJSON()` - Serialize report to JSON
   - `PrintModuleTable()` - Print human-readable module table

2. **`make coverage`** - Orchestrates test run and report generation
   - Runs `go test -coverprofile`
   - Invokes the CLI for analysis and JSON output

3. **`leamas factory coverage`** - CLI command for coverage analysis
   - Accepts `--profile` and `--min-total` flags (required)
   - Accepts `--json-output` flag (optional)
   - `--breakdown` / `--no-breakdown` flags control terminal output
   - Returns exit code 1 on threshold failure

4. **`leamas factory verify coverage`** - Verifier for gate integration
   - Checks if profile exists
   - Validates against threshold
   - Requires `make coverage` to be run first

### Gate Integration

The coverage verifier is **not** included in the default `make gate` run because:
- Running `go test -coverprofile` is expensive
- The expensive step lives in `make coverage`, not in the verifier
- This design avoids surprising slowdowns in the default workflow

To include coverage in the default gate:
1. Run `make coverage` first
2. Then run `make gate`

Or add coverage to the gate directly (future enhancement).

## Current Coverage

Measured weighted statement coverage:

| Module | Coverage |
|--------|----------|
| cmd/leamas | 52.0% |
| internal/factory | 69.7% |
| internal/hulk | 95.6% |
| internal/web | 74.6% |
| internal/witness | 85.4% |
| **Total** | **66.6%** |

Total statements: 4189 (2790 covered, 1399 uncovered)

## Non-Goals

This implementation does not:
- Upload coverage to external services
- Add coverage badges
- Force arbitrary coverage improvements
- Add CI-specific coverage publishing
- Implement module-level threshold failures
