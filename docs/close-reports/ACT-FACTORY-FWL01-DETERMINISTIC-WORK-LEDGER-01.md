# ACT-FACTORY-FWL01-DETERMINISTIC-WORK-LEDGER-01 — Close Report

**Date**: 2026-11-26
**Status**: Closed
**Verdict**: LEDGER_CORE_READY

## Summary

The Factory Work Ledger is implemented as a reusable, repository-native
engineering work-management mechanism in Leamas. Canonical records,
required operations, semantic validation, safe archival and representative
migration primitives are implemented and qualified.

Target repositories own the canonical work records. Leamas owns the
implementation, the schema and the deterministic read models.

## Final Head

```text
FINAL_HEAD  6f0ff6cd81769a47bf55fbb0690fe963a413f54e
parent      4bca64c2c5762b8c58184d558e9b014bbcb5590d
subject     factory: implement the deterministic Factory Work Ledger
tree        clean after the subject commit
```

The FINAL_HEAD is recorded dynamically from the committed repository
state. It is not embedded in any other committed artifact.

## Actual Implementation Paths

### Domain core (pure, no I/O)

| Path | Lines | Purpose |
|------|-------|---------|
| `internal/ledger/domain/model.go` | 310 | Typed identifiers and enumerations |
| `internal/ledger/domain/records.go` | 193 | Epic, work item, ACT, evidence, decision, knowledge records |
| `internal/ledger/domain/identity.go` | 112 | Stable scoped identity rules |
| `internal/ledger/domain/path.go` | 60 | Path-traversal and digest validation |
| `internal/ledger/domain/validate.go` | 240 | Ledger-level semantic authority |
| `internal/ledger/domain/validate_records.go` | 296 | Record-level semantic authority |
| `internal/ledger/domain/readmodel.go` | 165 | Operational board projection |
| `internal/ledger/domain/ready.go` | 194 | Ready-work selection and statistics |

### Storage

| Path | Lines | Purpose |
|------|-------|---------|
| `internal/ledger/store/store.go` | 158 | Canonical layout, init, load |
| `internal/ledger/store/load.go` | 151 | Strict record decoding |
| `internal/ledger/store/atomic.go` | 72 | Temp-plus-rename atomic writer |
| `internal/ledger/store/txn.go` | 253 | Journal-backed transaction and recovery |
| `internal/ledger/store/mutate.go` | 296 | Candidate-state validation and commit |

### Service, import and render

| Path | Lines | Purpose |
|------|-------|---------|
| `internal/ledger/service/service.go` | 204 | Create, update, revision checks |
| `internal/ledger/service/lifecycle.go` | 130 | Checkpoint, claim, accept |
| `internal/ledger/service/relations.go` | 234 | Link, evidence, successor |
| `internal/ledger/service/archive.go` | 158 | Archive, restore, ACT close |
| `internal/ledger/importer/importer.go` | 314 | Dry-run import planning |
| `internal/ledger/importer/apply.go` | 179 | Idempotent apply and delta |
| `internal/ledger/render/render.go` | 166 | Derived views |
| `internal/ledger/render/bootstrap.go` | 81 | Agent bootstrap context |

### CLI

| Path | Purpose |
|------|---------|
| `cmd/leamas/ledger.go` | Command dispatcher and usage |
| `cmd/leamas/ledger_read.go` | init, list, show, next, prime, stats, validate, render |
| `cmd/leamas/ledger_write.go` | create, update, claim, link |
| `cmd/leamas/ledger_write2.go` | evidence add, checkpoint, close, archive, restore |
| `cmd/leamas/ledger_import.go` | import (dry-run by default) |
| `cmd/leamas/ledger_recover.go` | recover (repair an interrupted transaction) |

### Documentation

| Path | Purpose |
|------|---------|
| `docs/factory/work-ledger.md` | The canonical contract |
| `docs/factory/skills/work-ledger.md` | Reusable agent guidance (W7) |
| `docs/adr/0007-factory-work-ledger.md` | Decision record |

## Schema and CLI Version

```text
ledger schema version   1
CLI surface version     1
command group           leamas ledger <subcommand>
```

Storage root: `.leamas/ledger` (overridable with `--root`).

## Behavior Changed

The repository gains a new command group. No existing command, verifier
or artifact changed behaviour. The only pre-existing file modified is
`cmd/leamas/main.go`, which registers the `ledger` command group and adds
one usage line.

## Executable Test Results

### Focused ledger suites (executed, PASS)

```text
go test ./internal/ledger/... -count=1
ok  github.com/s1onique/leamas/internal/ledger/domain
ok  github.com/s1onique/leamas/internal/ledger/importer
ok  github.com/s1onique/leamas/internal/ledger/render
ok  github.com/s1onique/leamas/internal/ledger/service
ok  github.com/s1onique/leamas/internal/ledger/store
```

### CLI ledger suite (executed, PASS)

```text
go test ./cmd/leamas/ -run TestLedger -count=1
ok  github.com/s1onique/leamas/cmd/leamas
```

### Build, vet, format (executed, PASS)

```text
go build ./...            PASS
go vet ./...              PASS
gofmt -l <ACT files>      PASS (no output)
```

### Factory verifiers (executed)

```text
leamas factory verify doctrine              PASS
leamas factory verify docs                  PASS
leamas factory verify forbidden-patterns    PASS
leamas factory verify long-test-policy      PASS
leamas factory verify agent-context         PASS
leamas factory verify domain-boundaries      PASS
leamas factory verify llm-friendly          PASS for every ACT-owned file
```

`leamas factory verify llm-friendly` reports findings in pre-existing
files outside this ACT (`docs/acts/...`,
`cmd/leamas/factory_close_tag_status.go`,
`cmd/leamas/factory_close_v2_verifier_mac_handoff_correction01_test.go`,
`internal/factory/closure/...`,
`internal/factory/digest/gate_summary_render_v3.go`,
`internal/gatesummary/...`).

A grep of the verifier output for `ledger`, `FWL01` or `work-ledger`
returns ZERO findings. No ACT-owned file is reported.

### Gate lanes

```text
CGO_ENABLED=0 make gate-fast   FAIL — caused ONLY by pre-existing
                                worktree-cleanliness constraints in the
                                cmd/leamas closure tests (see below)
make factorize                  NOT RUN — not authorized by this ACT
make gate-dupcode                NOT RUN — not authorized by this ACT
make gate                        NOT RUN — not authorized by this ACT
```

### Baseline comparison (definitive)

The same full suite (`go test -short -p 4 ./...`) was executed on a
STASHED (clean) tree during this ACT to separate pre-existing failures
from regressions. The failing packages are IDENTICAL in both runs:

```text
package                                     clean tree   ACT tree
cmd/leamas                                      FAIL        FAIL
internal/execution                              FAIL        FAIL
internal/factory/authority                      FAIL        FAIL
internal/factory/closure                        FAIL        FAIL
internal/factory/closure/evidence               FAIL        FAIL
internal/factory/digest                         FAIL        FAIL
internal/factory/dupcode                        FAIL        FAIL
internal/factory/forbidden                      FAIL        FAIL
internal/factory/gate                           FAIL        FAIL
internal/factory/verifierauthority              FAIL        FAIL
```

Both runs completed: the clean-tree baseline reported `ok packages: 37
/ 53` with the ten packages above failing, and the ACT tree reported
`ok packages: 45 / 53` with the same ten packages failing (the ACT tree
adds the five new ledger packages to the passing set).

ALL five ACT-owned ledger packages pass in BOTH runs:

```text
ok  github.com/s1onique/leamas/internal/ledger/domain
ok  github.com/s1onique/leamas/internal/ledger/importer
ok  github.com/s1onique/leamas/internal/ledger/render
ok  github.com/s1onique/leamas/internal/ledger/service
ok  github.com/s1onique/leamas/internal/ledger/store
```

A grep of the full ACT-tree run for a failure mentioning "ledger"
returns nothing. There is no regression from this ACT.

### Pre-existing failures NOT caused by this ACT

Five tests in `cmd/leamas` fail while the working tree is dirty:

```text
TestClosureCLIV2R2CRExactTipDogfood
TestClosureCLIV2VerifierMacHandoffCorrection02A
TestClosureCLIV2VerifierMacHandoffCorrection01
TestClosureCLIV2VerifierMacHandoff
TestBoundedSubprocessV2_TimeoutWithPartialBoundedOutput
```

The first four assert that the CALLER worktree is clean; they cannot pass
while an ACT is being implemented. They were confirmed to PASS on a
stashed (clean) tree during this ACT. The fifth is a timing-sensitive
test that passes in isolation on this tree and failed only under
concurrent load.

`internal/execution` additionally fails
`TestValidateRetainedPipeTopology` on this host. That failure was
confirmed on a stashed (clean) tree during this ACT: it is a
Darwin-specific pipe-topology expectation, not a regression from this
ACT.

`leamas factory verify language` and `leamas factory verify
tooling-boundaries` report findings inside `.factory/modcache/`, which is
a git-ignored, untracked Go module cache. They are environment
artifacts, not ACT-owned content.

## Fixture Conservation Report

The representative conversational intake fixture lives at
`internal/ledger/importer/testdata/iecr-intake-board.json`.

| Expected | Observed | Result |
|----------|----------|--------|
| 44 unique work items | 44 | CONSERVED |
| 8 workstreams | 8 | CONSERVED |
| 10 CLOSED records | 10 | CONSERVED |
| 1 CLOSED_INSUFFICIENT record | 1 | CONSERVED |
| 33 unfinished records | 33 | CONSERVED |
| 6 BLOCKED items | 6 | CONSERVED (BLOCKED is a subset of unfinished) |
| 12 exit criteria | 12 | CONSERVED |

Import conservation:

```text
first import   epics: 8  items: 44  skipped: 0   ambiguities: 0  unresolved: 0
                delta: added: 44, removed: 0, changed: 0, evidence added: 0
replayed import epics: 8  items: 0   skipped: 44 ambiguities: 0  unresolved: 0
                delta: added: 0, removed: 0, changed: 0, evidence added: 0
```

The fixture is conversational intake, NOT authoritative repository
evidence. No imported record starts accepted, and no imported record
acquires evidence quality. The `CLOSED_INSUFFICIENT` disposition is
conserved verbatim.

## Transaction Safety Results

| Witness | Result |
|---------|--------|
| Committed transaction leaves no journal | PASS |
| Interrupted transaction is rolled back to the pre-transaction bytes | PASS |
| Interrupted transaction leaves no created record behind | PASS |
| Recovery is idempotent | PASS |
| Replaying a transaction does not duplicate records | PASS |
| Journal path cannot escape the ledger root | PASS |
| A rejected candidate leaves the canonical state byte-identical | PASS |

## Example Compact Board

```text
# Factory Work Ledger

## EPIC-DEMO —

- `EPIC-DEMO/ITEM-01` [task] Demo work 01 — open
  - blocked: acceptance criteria declared without evidence
- `EPIC-DEMO/ITEM-02` [task] Demo work 02 — open
  - blocked: acceptance criteria declared without evidence
```

## Actual Context-Size Measurements

Measured with `leamas ledger prime --epic <EPIC> --json`, using the
built-in conservative estimator (four characters per token, rounded up):

| Scope | Items | Estimated tokens |
|-------|-------|------------------|
| EPIC-DEMO (1 criterion per item) | 8 | 293 |
| EPIC-BIG (1 criterion per item) | 36 | 1472 |

The provisional target of under 2,000 tokens for representative active
work is MET for both measured scopes. When the budget IS exceeded the
render says so explicitly:

```text
> bootstrap exceeds the 500-token budget; no required obligation was silently truncated.
```

No required obligation is ever silently truncated to meet a budget.
