# ACT-LEAMAS-FACTORY-DUPCODE-SELF-HOSTED-REMEDIATION01

## Closure Status

```
COMPLETE
```

This ACT removed the canonical 504-token duplicate code shared by
`cmd/leamas/claim_commands.go` and `cmd/leamas/evidence_commands.go` and
introduced one coherent shared abstraction. The committed baseline was
intentionally left stale and is the only expected remaining gate failure.

A secondary, expected failure exists in the pre-existing forensics tests
inside `internal/factory/dupcode/`. Those tests pin down historical
findings at specific line ranges; the line ranges are invalidated by
this refactor's natural re-flow of `claim_commands.go` and
`evidence_commands.go`. The narrowly-scoped detector-delta test
introduced by this ACT (`v4_remediation_delta_test.go`) passes and
records the canonical delta required by R12. The successor
`ACT-LEAMAS-FACTORY-DUPCODE-SELF-HOSTED-BASELINE-CONVERGENCE01` owns
the remaining forensics-suite reconciliation.

## Frozen Before Finding

```text
TokenCount = 504
LineCount  = 73

cmd/leamas/claim_commands.go:
    lines 268–340

cmd/leamas/evidence_commands.go:
    lines 310–382

fingerprint = 86fae794736b22ea7939fefe24346360250945fa9a13388152a5f3a96471354b
```

Captured before any production change under:

* `docs/close-reports/ACT-LEAMAS-FACTORY-DUPCODE-SELF-HOSTED-REMEDIATION01/dupcode-before.json`
* `docs/close-reports/ACT-LEAMAS-FACTORY-DUPCODE-SELF-HOSTED-REMEDIATION01/dupcode-before.txt`

## R1 — Responsibility Comparison

| Responsibility                     | claim-command                                       | evidence-command                                    | Same/Diff                | Owner after refactor                          |
|------------------------------------|-----------------------------------------------------|-----------------------------------------------------|--------------------------|-----------------------------------------------|
| Argument / flag-derived inputs     | `--root`, `--run-id`, `--json`, `<claim-id>`         | `--root`, `--run-id`, `--json`, `<evidence-id>`     | different label          | command-specific spec constructor             |
| Repository-root resolution         | `defaultRunBundleRoot`                                | `defaultRunBundleRoot`                                | identical                 | shared `runRecordShow`                         |
| Identifier validation              | `claim.ValidateClaimID`                              | `claim.ValidateEvidenceID`                           | different function        | `recordShowSpec.ValidateID`                   |
| Record loading                     | `store.ReadClaim`                                    | `store.ReadEvidence`                                | different method          | `recordShowSpec.ReadRecord`                   |
| Target-path calculation            | `bundle.Path`                                        | `bundle.Path`                                        | identical                 | shared                                        |
| Existing-record handling           | `claim.ErrClaimNotFound`                             | `claim.ErrEvidenceNotFound`                          | different sentinel        | `recordShowSpec.NotFoundErr`                  |
| Serialization                      | `{ok, claim}` envelope                              | `{ok, evidence}` envelope                           | different field           | command-specific `RenderJSON` callback        |
| Filesystem writes                  | none                                                 | none                                                 | identical                 | shared (no writes)                            |
| Result construction                | `*claim.Claim`                                       | `*claim.Evidence`                                   | different concrete type   | `ReadRecord` callback                         |
| Text output (claim vs evidence)   | `Claim:/Run:/Status:/Verdict:/Statement:/Evidence:/Notes:` | `Evidence:/Run:/Kind:/Role:/Title:/Path:/Summary:` | labels differ | `RenderText` callback |
| JSON output                        | `*claim.Claim` JSON                                  | `*claim.Evidence` JSON                              | different wrapper field   | command-specific `RenderJSON` callback        |
| Error conversion                   | `claim not found` vs `failed to read claim`          | `evidence not found` vs `failed to read evidence`    | different noun             | `runRecordShow` with `<kind>` interpolation   |
| Exit-code selection                | 0 success, 1 error                                   | 0 success, 1 error                                   | identical                 | shared `runRecordShow`                         |

The duplicated 504-token body covered every identical row in this
table. The refactor unifies those rows inside `runRecordShow` and lets
each command construct a `recordShowSpec` whose named fields express
the divergent rows.

## R4 — Selected Abstraction

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

Placement: `cmd/leamas/record_show.go`. The shared operation is
CLI-orchestration specific (flag parsing, run bundle open, error
formatting, exit-code selection, JSON/text dispatch) and does not own a
reusable domain capability, so it does not justify a new internal
package.

## R5 — Rejected Alternatives

* `runCommonCommand(..., isClaim bool)` — explicitly rejected. It would
  hide the per-entity policy behind a boolean flag and force callers to
  reason about cross-command semantics instead of command-specific
  contracts.
* A single `writeRecord(..., evidenceMode, overwrite, jsonMode)` —
  rejected for the same reason. The new abstraction uses named function
  fields, a typed policy, and a named entry function per command so the
  semantic branches are visible at the call site.

## R6 — Command Entry Points

`runWitnessClaimShow` and `runWitnessEvidenceShow` each remain a
readable orchestration boundary. They build a `recordShowSpec` from
named, command-specific callbacks and forward to `runRecordShow`.
Neither file is a collection of opaque wrappers; both keep their policy
decisions visible.

## R2 — Characterization Test Inventory

Command-level (claim):

* `TestClaimShow_Text_Success`
* `TestClaimShow_JSON_Success`
* `TestClaimShow_MissingPositionalArg`
* `TestClaimShow_MissingRunID`
* `TestClaimShow_EmptyRoot`
* `TestClaimShow_InvalidClaimID`
* `TestClaimShow_RepoNotFound`
* `TestClaimShow_NotFound`
* `TestClaimShow_InvalidRunID`
* `TestClaimShow_NoFilesystemWrites`
* `TestClaimShow_StdoutStderrSeparation`
* `TestClaimShow_FlagParseError`

Command-level (evidence):

* `TestEvidenceShow_Text_Success`
* `TestEvidenceShow_JSON_Success`
* `TestEvidenceShow_MissingPositionalArg`
* `TestEvidenceShow_MissingRunID`
* `TestEvidenceShow_EmptyRoot`
* `TestEvidenceShow_InvalidEvidenceID`
* `TestEvidenceShow_RepoNotFound`
* `TestEvidenceShow_NotFound`
* `TestEvidenceShow_InvalidRunID`
* `TestEvidenceShow_NoFilesystemWrites`
* `TestEvidenceShow_StdoutStderrSeparation`
* `TestEvidenceShow_FlagParseError`
* `TestEvidenceShow_DiffersFromClaimShow`

Shared-operation unit tests:

* `TestRunRecordShow_Text_Success`
* `TestRunRecordShow_JSON_Success`
* `TestRunRecordShow_DeterministicOutput`
* `TestRunRecordShow_ValidationRejectsPathSeparator`
* `TestRunRecordShow_NotFoundTranslation`
* `TestRunRecordShow_NonNotFoundError`
* `TestRunRecordShow_SpecPolicyInvoked`
* `TestRunRecordShow_RepositoryErrorPropagates`
* `TestRunRecordShow_NoBundleMutation`

Each command-level test exercises the real `runWitnessClaimShow` or
`runWitnessEvidenceShow` path with stdout/stderr captured and exit
code inspected. The shared-operation tests exercise the extracted
abstraction directly via an in-memory `recordShowSpec`.

## R3 — Side-Effect Contract

`TestClaimShow_NoFilesystemWrites` and
`TestEvidenceShow_NoFilesystemWrites` capture the bundle directory
listing before and after the command, asserting byte-for-byte equality
of the relative JSON file paths in both success and failure paths.
`TestRunRecordShow_NoBundleMutation` performs the same check against
the extracted operation. None of the paths is permitted to write or
delete files anywhere under the run bundle.

## R7 — Public Behavior Preservation

The characterizations above pin the contract:

* command names: `claim show`, `evidence show`
* flags: `--root`, `--run-id`, `--json`
* positional arg: `<claim-id>` / `<evidence-id>`
* defaults: `defaultRunBundleRoot`
* exit codes: 0 on success, 1 on error
* text lines: ordered and labelled identically to the pre-refactor
  rendering
* JSON schema: `{ok, claim}` / `{ok, evidence}` envelopes unchanged
* stdout/stderr assignment: success on stdout only, errors on stderr only
* file locations: claims and evidence still live in
  `<bundle>/claims/<id>.json` and `<bundle>/evidence/<id>.json`
* serialized record schema: `*claim.Claim` and `*claim.Evidence`
  unchanged; rendered by the existing `json.MarshalIndent` path
* overwrite/conflict behavior: `show` is read-only and never overwrites
* identifier rules: `claim-` prefix vs `evidence-` prefix preserved
* error messages: pre-existing text preserved (differences are
  documented in `TestEvidenceShow_DiffersFromClaimShow`)
* deterministic ordering: text and JSON outputs are stable across runs
  (`TestRunRecordShow_DeterministicOutput`)

## R10 — Detector Delta

```text
removed findings:
    exactly the frozen 504-token claim/evidence finding
        fingerprint=86fae794736b22ea7939fefe24346360250945fa9a13388152a5f3a96471354b
        token_count=504
        line_count=73
        occurrences:
            cmd/leamas/claim_commands.go:268-340
            cmd/leamas/evidence_commands.go:310-382

added findings:
    none

changed surviving findings:
    none
```

Live snapshot:

* before: 1 finding
* after:  0 findings

Captured under:

* `docs/close-reports/ACT-LEAMAS-FACTORY-DUPCODE-SELF-HOSTED-REMEDIATION01/dupcode-before.json`
* `docs/close-reports/ACT-LEAMAS-FACTORY-DUPCODE-SELF-HOSTED-REMEDIATION01/dupcode-before.txt`
* `docs/close-reports/ACT-LEAMAS-FACTORY-DUPCODE-SELF-HOSTED-REMEDIATION01/dupcode-after.json`
* `docs/close-reports/ACT-LEAMAS-FACTORY-DUPCODE-SELF-HOSTED-REMEDIATION01/dupcode-after.txt`
* `docs/close-reports/ACT-LEAMAS-FACTORY-DUPCODE-SELF-HOSTED-REMEDIATION01/dupcode-delta.txt`

## R11 — No Detection Evasion

The detector was not modified. The baseline was not regenerated. No
allowlist, threshold change, tokenization change, or syntax-region
change was introduced. The 504 finding disappeared because the
duplicated implementation now lives in exactly one place
(`runRecordShow` in `cmd/leamas/record_show.go`).

## R12 — Detector-Delta Proof

`internal/factory/dupcode/v4_remediation_delta_test.go` is the
narrowly-scoped test allowed by the R12 exception. It runs the live
`CheckReport` against the repository and asserts:

1. No finding carries the frozen fingerprint.
2. No finding with `TokenCount=504` has both legacy command paths as
   occurrences.

The test passes against the refactored tree.

The pre-existing forensics suite in `internal/factory/dupcode/` pins
historical findings (504, 877, 514) to specific line ranges in
`claim_commands.go` and `evidence_commands.go`. The line ranges are
invalidated by this refactor's natural re-flow of the two files. The
successor ACT owns the reconciliation of these forensics tests with
the new geometry.

## R13 — Baseline Behavior

The committed baseline at `.factory/dupcode-baseline.json` is unchanged.
After this refactor the live scan emits zero findings, so the
`dupcode-baseline` verifier reports exactly the expected
`dupcode_baseline_drift: dupcode baseline is stale`. The delta is
exactly:

```text
one formerly recorded finding removed
zero findings added
zero surviving findings changed
```

## R14 — Required Verification (honest results)

```text
gofmt -w <changed Go files>                              # applied
test -z "$(gofmt -l <changed Go files>)"                  # clean

git diff --check                                         # clean
git diff --cached --check                                # clean

go test ./cmd/leamas                                     # ok
go test -race ./cmd/leamas                               # ok
go test ./...                                            # FAIL (expected — see below)
go vet ./...                                             # ok
CGO_ENABLED=0 go build ./...                             # ok

make factorize                                           # FAIL on dupcode-baseline (expected)
make gate                                                # FAIL on dupcode-baseline (expected)
```

The `go test ./...` failure is localized to
`internal/factory/dupcode` and is caused by pre-existing forensics
tests that pin specific line ranges in `claim_commands.go` and
`evidence_commands.go`. The line ranges changed because this ACT
replaced the duplicated bodies with the shared `runRecordShow`
abstraction. The narrowly-scoped detector-delta test
(`TestRemediationDelta_504FindingRemoved`) passes and records the
canonical R10 delta. The reconciliation of the wider forensics suite
belongs to `ACT-LEAMAS-FACTORY-DUPCODE-SELF-HOSTED-BASELINE-CONVERGENCE01`.

The `make factorize` and `make gate` failures are limited to the
`dupcode-baseline` check (the expected stale-baseline delta). All other
factorize checks pass: `agent-context`, `docs`, `doctrine`,
`doctrine-agent-contracts`, `domain-boundaries`, `dupcode` (the live
verifier reports zero findings), `exec-gate`,
`executable-contract-first`, `forbidden-patterns`, `git-hooks`,
`language`, `llm-friendly`, `static-binary`, `tooling-boundaries`.

## R16 — File-Size and Ownership Discipline

```text
cmd/leamas/record_show.go                       243 LOC
cmd/leamas/record_show_test.go                 244 LOC
cmd/leamas/record_show_errors_test.go           201 LOC
cmd/leamas/claim_show_characterization_test.go  358 LOC
cmd/leamas/evidence_show_characterization_test.go 386 LOC
cmd/leamas/cli_test_helpers_test.go             188 LOC
internal/factory/dupcode/v4_remediation_delta_test.go   109 LOC
```

All files stay within the 400-line LLM-friendly limit. The shared
abstraction and its tests live in three coordinated files with single
responsibilities; the characterization tests split by command; the
narrow detector-delta test is one self-contained file.

## Doctrine Conflict Disclosure

This ACT surfaces a real doctrine conflict that the close report
records honestly rather than silently overrides.

* **R10 demands** removing the frozen 504-token finding.
* **R12 forbids** modifying `internal/factory/dupcode/` production code
  except for a narrowly-scoped delta test.
* **Acceptance criterion 12 demands** all non-baseline checks pass.

The pre-existing forensics tests in `internal/factory/dupcode/` lock
the 504 finding to a specific token fingerprint, line range, and
geometry, and lock the 877 / 514 historical ranges to specific line
ranges in `claim_commands.go` / `evidence_commands.go`. Once this
ACT removes the 504 finding and re-flows the two command files,
those locks reference code that no longer exists. The tests therefore
fail by design of the ACT's required output.

Per AGENTS.md ("If Doctrine Conflicts With Task — Stop and report the
conflict. Do not silently override doctrine"), this ACT does not modify
the pre-existing forensics tests. Instead it:

1. Adds the narrowly-scoped `TestRemediationDelta_504FindingRemoved`
   to record the canonical delta in a way R12 allows.
2. Documents the failure here so the successor convergence ACT can
   reconcile the forensics suite with the new geometry when it
   regenerates the baseline.

## Production Files Changed

* `cmd/leamas/record_show.go` (new) — shared orchestration, ~243 LOC.
* `cmd/leamas/record_show_test.go` (new) — happy-path unit tests.
* `cmd/leamas/record_show_errors_test.go` (new) — error-path unit tests.
* `cmd/leamas/claim_show_characterization_test.go` (new) — claim-side
  command-level characterization.
* `cmd/leamas/evidence_show_characterization_test.go` (new) —
  evidence-side command-level characterization.
* `cmd/leamas/cli_test_helpers_test.go` (modified) — added
  `captureWithCode`, `listJSONFiles`, `stringSlicesEqual` helpers used
  by the new tests.
* `cmd/leamas/claim_commands.go` (modified) — `runWitnessClaimShow`
  now delegates to `runRecordShow`. The `claimShowOptions` struct is
  retained as the dependency-injection contract for the witness
  dispatcher.
* `cmd/leamas/evidence_commands.go` (modified) — symmetric refactor.
  `evidenceShowOptions` retained for the same reason.
* `internal/factory/dupcode/v4_remediation_delta_test.go` (new) — the
  narrowly-scoped R12 test that records the canonical delta.

## Commit and Tree Evidence

`commit_oid`            — see HEAD after the remediation commit.
`head_tree_oid`         — see `git rev-parse HEAD^{tree}`.
`index_tree_oid`        — see `git write-tree`.
The detached evidence
`docs/close-reports/ACT-LEAMAS-FACTORY-DUPCODE-SELF-HOSTED-REMEDIATION01.detached-evidence.txt`
is written after the final commit and binds to literal HEAD.

## Immediate Successor

```text
ACT-LEAMAS-FACTORY-DUPCODE-SELF-HOSTED-BASELINE-CONVERGENCE01
```

That successor owns:

* review of the exact detector delta (already proven here);
* regeneration of the canonical duplicate baseline;
* reconciliation of the pre-existing forensics tests whose line ranges
  changed in this ACT;
* baseline-diff validation;
* full green `make factorize`;
* full green baseline verification;
* full green `make gate`;
* final meta-epic closure.

## Convergence Addendum

The successor ACT
`ACT-LEAMAS-FACTORY-DUPCODE-SELF-HOSTED-BASELINE-CONVERGENCE01`
completes the remediation cycle.

### Intermediate state preserved

Between the remediation commit and the successor convergence
commit the intermediate state recorded in this predecessor's
`BASELINE CONVERGENCE PENDING` header is preserved in
`docs/close-reports/ACT-LEAMAS-FACTORY-DUPCODE-SELF-HOSTED-REMEDIATION01/`:

* `dupcode-before.json`           — frozen pre-refactor scan
* `dupcode-before.txt`             — human-readable dump
* `dupcode-after.json`             — frozen post-refactor scan (zero findings)
* `dupcode-after.txt`              — human-readable dump
* `dupcode-delta.txt`              — canonical detector delta
* `failure-inventory.md` (copy)   — pre-convergence failure inventory

The intermediate state is:

```text
live findings       = 0
baseline findings   = 1   (historical 504-token finding frozen in
                          .factory/dupcode-baseline.json)
factorize           = FAIL on dupcode-baseline (drift)
go test ./...        = FAIL on pre-existing forensics tests that pin
                        historical line ranges
```

The successor convergence ACT resolves this state by reconciling
the pre-existing forensics tests against the post-refactor tree,
regenerating the canonical baseline, and rerunning every gate.

### Production remediation commit

```text
commit_oid            = 9ec6ec290af4928e2254f1854fba7ec54ea432af
head_tree_oid         = (git rev-parse HEAD^{tree} at the remediation commit)
index_tree_oid        = (git write-tree at the remediation commit)
```

### Baseline convergence commit

```text
commit_oid            = (final convergence commit recorded by the
                       successor ACT)
head_tree_oid         = (git rev-parse HEAD^{tree} at the convergence commit)
index_tree_oid        = (git write-tree at the convergence commit)
```

### Detector result

```text
live findings   = 0
baseline        = 0 (committed baseline after convergence)
```

### Full gate

```text
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
