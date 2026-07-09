# Factory: Go Coverage Gate

This document describes the Go test coverage measurement and threshold checking
infrastructure, including module-level breakdown and module floors.

## Overview

The coverage gate provides a way to measure and enforce minimum Go test coverage
thresholds. The current conservative ratchet threshold is **64%** total, with
conservative per-module floors to prevent individual modules from regressing.

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
6. Check against the configured thresholds (total and module floors)

### Output Format

**Terminal Output:**
```
Generating coverage profile...
coverage: total=66.6% min=64.0% OK
coverage: module cmd/leamas=52.0% min=50.0% OK
coverage: module internal/factory=69.7% min=67.0% OK
coverage: module internal/hulk=95.6% min=90.0% OK
coverage: module internal/web=74.6% min=70.0% OK
coverage: module internal/witness=85.4% min=80.0% OK
Coverage by module:
module                  coverage
cmd/leamas              52.0%
internal/factory        69.7%
internal/hulk           95.6%
internal/web            74.6%
internal/witness        85.4%
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

# With module floors (using individual --min-module flags)
leamas factory coverage \
  --profile .factory/coverage.out \
  --min-total 64 \
  --min-module cmd/leamas=50 \
  --min-module internal/factory=67 \
  --min-module internal/hulk=90 \
  --min-module internal/web=70 \
  --min-module internal/witness=80

# Using default module floors (convenience flag)
leamas factory coverage \
  --profile .factory/coverage.out \
  --min-total 64 \
  --default-module-floors

# Generate JSON output only (no terminal breakdown)
leamas factory coverage --profile .factory/coverage.out --min-total 64.0 \
  --min-module cmd/leamas=50 \
  --min-module internal/factory=67 \
  --min-module internal/hulk=90 \
  --min-module internal/web=70 \
  --min-module internal/witness=80 \
  --json-output .factory/coverage-summary.json

# Hide module breakdown on terminal
leamas factory coverage --profile .factory/coverage.out --min-total 64.0 \
  --min-module cmd/leamas=50 \
  --min-module internal/factory=67 \
  --min-module internal/hulk=90 \
  --min-module internal/web=70 \
  --min-module internal/witness=80 \
  --no-breakdown

# Check with a different threshold
leamas factory coverage --profile .factory/coverage.out --min-total 50.0 \
  --min-module cmd/leamas=40 \
  --min-module internal/factory=60
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
coverage: module cmd/leamas=52.0% min=50.0% OK
...
```

**Total Failure:**
```
coverage: threshold_fail: total coverage 63.9% is below minimum 64.0%
```

**Module Failure:**
```
coverage: module_threshold_fail: module cmd/leamas coverage 49.9% is below minimum 50.0%
```

### Current Thresholds

**Global Threshold:** 64%

**Module Floors:**

| Module | Floor | Current | Headroom |
|--------|-------|---------|----------|
| cmd/leamas | 50.0% | 52.0% | 2.0% |
| internal/factory | 67.0% | 69.7% | 2.7% |
| internal/hulk | 90.0% | 95.6% | 5.6% |
| internal/web | 70.0% | 74.6% | 4.6% |
| internal/witness | 80.0% | 85.4% | 5.4% |
| other | (report only) | 85.7% | N/A |

**Note:** The `other` module is report-only and not enforced. This allows flexibility
for new modules while still protecting known stable components.

### Why Module Floors?

The global threshold protects the overall repository, but it does not prevent a
single module from silently dropping while another module improves. Module floors
ensure that:

1. Individual modules don't regress below their current conservative levels
2. Coverage improvements in one module don't mask declines in another
3. Teams can see which specific module needs attention when coverage fails

### Threshold Selection

Module floors were set conservatively with ~2-6% headroom from current measured
coverage. These floors can be ratcheted later as coverage improves.

**Module threshold selection rule:**
- Module floors remain conservative and can be ratcheted later
- Floors are not raised without corresponding coverage improvements
- "other" is intentionally report-only to avoid blocking on new modules

### Raising Thresholds

To raise the global threshold, edit the Makefile:

```makefile
COVERAGE_MIN_TOTAL ?= 65
```

To raise a module floor, edit the Makefile:

```makefile
COVERAGE_MIN_INTERNAL_FACTORY ?= 68
```

Or override on the command line:

```bash
COVERAGE_MIN_CMD_LEAMAS=55 make coverage
```

Or pass directly to the CLI:

```bash
--min-module cmd/leamas=55
```

## Module Thresholds (Now Enforced)

Module-level thresholds are **now enforced** by default.

**Behavior:**
- `make coverage` enforces all module floors
- `leamas factory verify coverage` enforces all module floors
- Missing enforced modules fail closed (error, not warning)
- `other` remains report-only (no enforcement)

## Architecture

### Components

1. **`internal/factory/coverage/`** - Core coverage parsing and threshold logic
   - `Report` - Module breakdown report with JSON serialization
   - `Threshold` - Threshold configuration (with module floors)
   - `ModuleSummary` - Per-module coverage data
   - `ProfileReport` - Statement-weighted report with statement counts
   - `ParseProfile()` - Parse raw coverage profile for weighted aggregation
   - `CheckThreshold()` - Compare report against threshold (total + modules)
   - `DefaultModuleThresholds()` - Default conservative module floors
   - `DefaultThreshold()` - Full default threshold with total and modules
   - `Analyze()` - Full analysis pipeline: ParseProfile() -> statement-weighted report -> CheckThreshold()
   - `ClassifyModule()` - Map import path to module name
   - `ToJSON()` - Serialize report to JSON
   - `PrintModuleTable()` - Print human-readable module table

2. **`make coverage`** - Orchestrates test run and report generation
   - Runs `go test -coverprofile`
   - Invokes the CLI for analysis and JSON output
   - Passes module floors to CLI

3. **`leamas factory coverage`** - CLI command for coverage analysis
   - Accepts `--profile` and `--min-total` flags (required)
   - Accepts `--min-module` flags (repeatable, format: `module=threshold`)
   - Accepts `--default-module-floors` flag (convenience)
   - Accepts `--json-output` flag (optional)
   - `--breakdown` / `--no-breakdown` flags control terminal output
   - Returns exit code 1 on threshold failure

4. **`leamas factory verify coverage`** - Verifier for gate integration
   - Checks if profile exists
   - Validates against threshold (total + module floors)
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
- Enforce `other` module (remains report-only)
