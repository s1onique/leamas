# ACT-LEAMAS-FACTORY-DUPCODE-SELF-HOSTED-REMEDIATION01

## Closure Status

```
COMPLETE
```

Removed the canonical 504-token duplicate code shared by
`cmd/leamas/claim_commands.go` and `cmd/leamas/evidence_commands.go`,
introducing one coherent shared abstraction. At the remediation commit,
the committed baseline was intentionally left stale and the successor ACT
owned forensics-suite reconciliation. That successor subsequently
completed the convergence work documented in the Convergence Addendum.

## Frozen Before Finding

```
TokenCount = 504, LineCount = 73
cmd/leamas/claim_commands.go: lines 268–340
cmd/leamas/evidence_commands.go: lines 310–382
fingerprint = 86fae794736b22ea7939fefe24346360250945fa9a13388152a5f3a96471354b
```

Captured under `docs/close-reports/ACT-LEAMAS-FACTORY-DUPCODE-SELF-HOSTED-REMEDIATION01/`.

## Responsibility Comparison Summary

Both commands share: repository-root resolution, target-path calculation,
filesystem writes (none), exit-code selection. Differences are per-entity
labeling and type-specific callbacks.

| Responsibility | claim | evidence | Shared after refactor |
|---------------|-------|----------|----------------------|
| Input flags | `--root`, `--run-id`, `--json`, `<claim-id>` | `--root`, `--run-id`, `--json`, `<evidence-id>` | command-specific spec |
| Record loading | `store.ReadClaim` | `store.ReadEvidence` | `recordShowSpec.ReadRecord` |
| Text output labels | Claim:/Run:/Status:/... | Evidence:/Run:/Kind:/... | `RenderText` callback |
| JSON wrapper | `{ok, claim}` | `{ok, evidence}` | `RenderJSON` callback |
| Error nouns | claim | evidence | `<kind>` interpolation |

## Selected Abstraction

```go
type recordShowSpec struct {
    KindName    string
    PosArgLabel string
    ValidateID  func(rawID string) error
    NotFoundErr error
    ReadRecord  func(store claim.Store, rawID string) (any, error)
    RenderText  func(w io.Writer, bundlePath string, record any)
    RenderJSON  func(w io.Writer, record any) error
}

func runRecordShow(args []string, spec recordShowSpec) int { /* shared orchestration */ }
```

Placement: `cmd/leamas/record_show.go`. CLI-orchestration specific;
does not justify a new internal package.

## Rejected Alternatives

* `runCommonCommand(..., isClaim bool)` — hides per-entity policy behind
  a boolean flag.
* Single `writeRecord(..., evidenceMode, overwrite, jsonMode)` — same
  issue. Named function fields with typed policy and named entry per
  command make semantic branches visible at call site.

## Characterization Test Coverage

Command-level (claim): `TestClaimShow_*` (13 tests covering success,
missing args, invalid IDs, repo not found, not found, filesystem
writes, stdout/stderr, flag errors).

Command-level (evidence): `TestEvidenceShow_*` (14 tests, including
`TestEvidenceShow_DiffersFromClaimShow`).

Shared-operation unit tests: `TestRunRecordShow_*` (8 tests covering
success, validation, error translation, repository errors, bundle
mutation prevention).

All command-level tests exercise real paths with stdout/stderr capture
and exit code inspection.

## Public Behavior Preservation

* Command names: `claim show`, `evidence show`
* Flags: `--root`, `--run-id`, `--json`; positional arg: `<kind>-id`
* Defaults: `defaultRunBundleRoot`; exit codes: 0 success, 1 error
* Text lines: ordered and labelled identically to pre-refactor
* JSON schema: `{ok, claim}` / `{ok, evidence}` envelopes unchanged
* Stdout/stderr: success on stdout, errors on stderr
* File locations: unchanged
* Serialized record schema: `*claim.Claim` and `*claim.Evidence` unchanged
* `show` is read-only; identifier rules (`claim-`/`evidence-` prefix) preserved
* Deterministic output (`TestRunRecordShow_DeterministicOutput`)

## Side-Effect Contract

`TestClaimShow_NoFilesystemWrites`, `TestEvidenceShow_NoFilesystemWrites`,
and `TestRunRecordShow_NoBundleMutation` assert byte-for-byte equality of
bundle directory listings before and after commands.

## Detector Delta

```
removed: fingerprint=86fae794736b22ea7939fefe24346360250945fa9a13388152a5f3a96471354b
         token_count=504, line_count=73
         cmd/leamas/claim_commands.go:268-340
         cmd/leamas/evidence_commands.go:310-382
added:   none
changed: none
```

Live: 1 finding → 0 findings.

## No Detection Evasion

Detector not modified. Baseline not regenerated. No allowlist, threshold
change, tokenization change, or syntax-region change. Finding disappeared
because duplicated implementation now lives in exactly one place
(`runRecordShow` in `cmd/leamas/record_show.go`).

## Detector-Delta Proof

`v4_remediation_delta_test.go` asserts:
1. No finding carries the frozen fingerprint.
2. No finding with TokenCount=504 has both legacy command paths.

Pre-existing forensics tests in `internal/factory/dupcode/` pin historical
findings to specific line ranges invalidated by re-flow. Successor ACT
owns reconciliation.

## Expected Intermediate Failures at the Remediation Commit

| Check | Failure | Owner |
|-------|---------|-------|
| `make factorize` | dupcode-baseline drift | BASELINE-CONVERGENCE01 |
| `make gate` | dupcode-baseline drift | BASELINE-CONVERGENCE01 |
| forensics tests | line range changes | BASELINE-CONVERGENCE01 |

All other checks pass: `agent-context`, `docs`, `doctrine`,
`doctrine-agent-contracts`, `domain-boundaries`, `dupcode` (live verifier
zero findings), `exec-gate`, `executable-contract-first`, `forbidden-patterns`,
`git-hooks`, `language`, `llm-friendly`, `static-binary`, `tooling-boundaries`.

## Doctrine Conflict Disclosure

R10 demands removing the frozen 504-token finding. R12 forbids modifying
`internal/factory/dupcode/` production code except for narrowly-scoped
delta test. Acceptance criterion 12 demands all non-baseline checks pass.

Pre-existing forensics tests lock 504 finding to specific fingerprint,
line range, and geometry. Once this ACT removes the finding and re-flows
the files, those locks reference code that no longer exists.

Resolution: added narrowly-scoped `TestRemediationDelta_504FindingRemoved`
per R12; documented failures for successor ACT.

## Production Files Changed

* `cmd/leamas/record_show.go` (new, ~243 LOC) — shared orchestration
* `cmd/leamas/record_show_test.go` (new, ~244 LOC) — happy-path unit tests
* `cmd/leamas/record_show_errors_test.go` (new, ~201 LOC) — error-path tests
* `cmd/leamas/claim_show_characterization_test.go` (new, ~358 LOC)
* `cmd/leamas/evidence_show_characterization_test.go` (new, ~386 LOC)
* `cmd/leamas/cli_test_helpers_test.go` (modified) — added helpers
* `cmd/leamas/claim_commands.go` (modified) — `runWitnessClaimShow` delegates
* `cmd/leamas/evidence_commands.go` (modified) — symmetric refactor
* `internal/factory/dupcode/v4_remediation_delta_test.go` (new, ~109 LOC)

## Successor Ownership — Fulfilled

`ACT-LEAMAS-FACTORY-DUPCODE-SELF-HOSTED-BASELINE-CONVERGENCE01` was assigned:
* Review of exact detector delta (already proven here)
* Regeneration of canonical duplicate baseline
* Reconciliation of pre-existing forensics tests
* Baseline-diff validation
* Full green `make factorize`, baseline verification, `make gate`
* Final meta-epic closure

This successor subsequently completed all assigned items as documented in
its close report.

## Convergence Addendum

Successor `BASELINE-CONVERGENCE01` completes the remediation cycle.

### Intermediate state preserved

```
live findings:       0
baseline findings:   1 (historical 504-token, stale in .factory/dupcode-baseline.json)
factorize:           FAIL on dupcode-baseline (expected drift)
go test ./...:       FAIL on forensics tests pinning historical line ranges
```

### Detector result after convergence

```
live findings:   0
baseline:        0 (committed baseline after convergence)
```

### Full gate after convergence

```
make factorize   = PASS
make gate        = PASS
go test ./...    = PASS
go test -race    = PASS
```

### Reproduction commands

```bash
go test ./internal/factory/dupcode -count=1
go test ./... -count=1
./bin/leamas factory verify dupcode --json
./bin/leamas factory verify dupcode-baseline --json
make factorize
make gate
```
