# META-EPIC-LEAMAS-FACTORY-DUPCODE-SELF-HOSTED-CONVERGENCE01

## Closure Status

```
CLOSED
```

## Final Outcome

```text
Leamas successfully used its own duplicate detector to identify,
govern, remove, and ratchet away a real duplicate in Leamas itself.
```

## Chain of ACTs

The meta-epic links the canonical content merge V4 work with the
self-hosted remediation and convergence:

```text
V4 semantic and geometry correctness
    (ACT-LEAMAS-FACTORY-DUPCODE-V4-EXACT-SEMANTICS-PRODUCTION01,
     ACT-LEAMAS-FACTORY-DUPCODE-V4-EXACT-SEMANTICS-TESTS01)
        |
        v
all-pairs performance optimization
    (ACT-LEAMAS-FACTORY-DUPCODE-V4-ALL-PAIRS-MATERIALIZATION-PERFORMANCE01)
        |
        v
alignment guard and differential/fuzz proof
    (ACT-LEAMAS-FACTORY-DUPCODE-V4-ALL-PAIRS-MATERIALIZATION-PERFORMANCE01-CORRECTION02,
     ACT-LEAMAS-FACTORY-DUPCODE-V4-ALL-PAIRS-MATERIALIZATION-PERFORMANCE01-CORRECTION02-CORPUS-AND-EVIDENCE01,
     ACT-LEAMAS-FACTORY-DUPCODE-V4-ALL-PAIRS-MATERIALIZATION-PERFORMANCE01-CORRECTION02-R1-CROSS-REGION-PROOF01)
        |
        v
canonical maximal component merge and 504-token detection
    (ACT-LEAMAS-FACTORY-DUPCODE-V4-CANONICAL-MAXIMAL-COMPONENT-MERGE01
     and corrections 01-05)
        |
        v
self-hosted claim/evidence remediation
    (ACT-LEAMAS-FACTORY-DUPCODE-SELF-HOSTED-REMEDIATION01)
        |
        v
baseline and forensics convergence
    (ACT-LEAMAS-FACTORY-DUPCODE-SELF-HOSTED-BASELINE-CONVERGENCE01)
```

## Frozen Detector Delta

```text
removed findings:
    exactly fingerprint
    86fae794736b22ea7939fefe24346360250945fa9a13388152a5f3a96471354b
    token_count=504
    line_count=73
    occurrences:
        cmd/leamas/claim_commands.go:268-340
        cmd/leamas/evidence_commands.go:310-382

added findings:
    none

changed surviving findings:
    none

final findings (committed baseline):
    zero
```

## Detector vs Baseline State

The committed baseline at `.factory/dupcode-baseline.json` reports
zero findings and the live scan reports zero findings. The baseline
verify gate returns `status: ok` with no drift. The dupcode verify
gate returns `has_changes: false` because the live tree and committed
baseline are byte-equal on the same zero-finding state.

## Files Changed

* `cmd/leamas/record_show.go` (new, ~243 LOC)
* `cmd/leamas/record_show_test.go` (new, ~244 LOC)
* `cmd/leamas/record_show_errors_test.go` (new, ~201 LOC)
* `cmd/leamas/claim_show_characterization_test.go` (new, ~358 LOC)
* `cmd/leamas/evidence_show_characterization_test.go` (new, ~386 LOC)
* `cmd/leamas/cli_test_helpers_test.go` (modified, ~188 LOC)
* `cmd/leamas/claim_commands.go` (modified, ~296 LOC)
* `cmd/leamas/evidence_commands.go` (modified, ~338 LOC)
* `internal/factory/dupcode/v4_remediation_delta_test.go` (new)
* `internal/factory/dupcode/v4_self_hosted_fixture_test.go` (new)
* `internal/factory/dupcode/v4_pipeline_trace_test.go` (modified)
* `internal/factory/dupcode/v4_baseline_forensics_504_test.go` (modified)
* `internal/factory/dupcode/v4_baseline_forensics_504_trace_test.go` (modified)
* `internal/factory/dupcode/v4_baseline_forensics_504_maximality_test.go` (modified)
* `internal/factory/dupcode/v4_baseline_forensics_all_test.go` (modified)
* `internal/factory/dupcode/v4_baseline_forensics_facts_test.go` (modified)
* `internal/factory/dupcode/v4_baseline_delta_test.go` (modified)
* `internal/factory/dupcode/v4_baseline_audit_test.go` (modified)
* `internal/factory/dupcode/debug_test.go` (modified)
* `.factory/dupcode-baseline.json` (regenerated, zero findings)
* `docs/close-reports/ACT-LEAMAS-FACTORY-DUPCODE-SELF-HOSTED-BASELINE-CONVERGENCE01/`
  (new: failure inventory, baseline before/after, post-convergence
   gate log)
* `docs/close-reports/ACT-LEAMAS-FACTORY-DUPCODE-SELF-HOSTED-REMEDIATION01.md`
  (updated: closure status COMPLETE, convergence addendum)
* `docs/close-reports/META-EPIC-LEAMAS-FACTORY-DUPCODE-SELF-HOSTED-CONVERGENCE01.md`
  (new: this file)

## Verification

```text
go test ./internal/factory/dupcode                       = PASS
go test ./...                                          = PASS
go test -race ./cmd/leamas                             = PASS
go test -race ./internal/factory/dupcode -run ...       = PASS
./bin/leamas factory verify dupcode-baseline --json     = status:ok
./bin/leamas factory verify dupcode --json              = has_changes:false
make factorize                                         = PASS
make gate                                              = PASS
```

The final `.factory/gate-summary.json` reports:

```text
source_status       = present
overall_status      = pass
checks_failed       = 0
checks_unavailable  = 0
generated_at        = (concrete timestamp recorded by the gate)
```

## Reproduction commands

```bash
go test ./internal/factory/dupcode -count=1
go test ./... -count=1
go test -race ./cmd/leamas
go test -race ./internal/factory/dupcode \
    -run='^(TestRemediationDelta_|TestV4BaselineDelta_)' -count=1
./bin/leamas factory verify dupcode --json
./bin/leamas factory verify dupcode-baseline --json
make factorize
make gate
```
