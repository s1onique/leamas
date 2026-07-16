# ACT-LEAMAS-FACTORY-DUPCODE-V4-CANONICAL-MAXIMAL-COMPONENT-MERGE01-CORRECTION05

## Status: COMPLETE

`ACT-LEAMAS-FACTORY-DUPCODE-V4-CANONICAL-MAXIMAL-COMPONENT-MERGE01-CORRECTION05`
is **COMPLETE**. The CORRECTION04 review verdict was
**REOPEN — unchanged** because the uploaded digest was a
**dirty-mode** digest against a working tree that contained
uncommitted mechanical formatting and final-newline drift on
five `*_test.go` files introduced when the lifecycle-banner
edits ran without an explicit `gofmt`. The committed tree was
not in question; the gap between the committed tree and the
working tree was. The reviewer explicitly directed **not** to
create CORRECTION06 and to finish CORRECTION05 by running
`gofmt -w` against the five files, committing them, and
re-attaching clean, freshly-time-stamped evidence plus an
explicit range digest.

CORRECTION05 executes exactly that. The five Go test files are
now committed, the working tree is clean, the live gate is
20/20 pass against the new HEAD, the post-commit evidence is
regenerated and bound to `HEAD^{tree}` of the formatting
commit, and an explicit range digest covers the
CORRECTION04-to-CORRECTION05 interval. No algorithmic change
was introduced by this corrective pass; the maximality proof,
the one-token extension audit, the structural-shadow survival,
the public acceptance test, the 877/514 owner-count and
unowned-token facts, the production pipeline, and the public
surface are byte-identical to the CORRECTION04 committed tree.

`ACT-LEAMAS-FACTORY-DUPCODE-V4-ALL-PAIRS-MATERIALIZATION-PERFORMANCE01`
is now unblocked.

## Defect closed

The reviewer reported:

```text
* v4_baseline_forensics_504_maximality_test.go
* v4_baseline_forensics_504_trace_test.go
* v4_baseline_forensics_facts_test.go
* v4_baseline_forensics_helpers_test.go
* v4_pipeline_trace_test.go
```

were unstaged in the working tree after the CORRECTION04
commit landed. The blob diffs are pure:

  - removal of a single blank line inside the test-only file
    header (`import (...)` group spacing);
  - restoration of a final newline that had been truncated to
    `\ No newline at end of file`.

Neither change affects semantics — both files are pure test
artefacts and the only externally visible behaviour is the
test result itself.

The corrective recipe applied was the one provided by the
reviewer:

```bash
gofmt -w \
  internal/factory/dupcode/v4_baseline_forensics_504_maximality_test.go \
  internal/factory/dupcode/v4_baseline_forensics_504_trace_test.go \
  internal/factory/dupcode/v4_baseline_forensics_facts_test.go \
  internal/factory/dupcode/v4_baseline_forensics_helpers_test.go \
  internal/factory/dupcode/v4_pipeline_trace_test.go

git add internal/factory/dupcode/
git commit -m "ACT-LEAMAS-FACTORY-DUPCODE-V4-COMPONENT-MERGE-CORRECTION05 finalize test formatting"
```

The resulting commit object is the canonical reference for
this correction; the commit message is verbatim from the
verdict so the historical record is preserved.

## Verification

All gates were re-run against the new HEAD. Honest results:

```text
make factorize              PASS  (15/15 verifiers, no toolchain)
make gate                   PASS  (20/20 checks total, 0 fail)
                            toolchain re-included:
                              go mod tidy    OK
                              gofmt          OK
                              go vet ./...   OK
                              go test ./...  OK
                              static build   OK
```

Exact suite (one-token extension, sorted-fingerprint,
no-live-larger, structural-shadow survival, public
classification) and package suite were exercised as part of
`go test ./...`:

```text
go test ./internal/factory/dupcode/...
  ok  github.com/s1onique/leamas/internal/factory/dupcode  215.423s
```

`go test -run 'Exact21|Correction|Range|Region|Pipeline|V4' ./internal/factory/dupcode/...`

```text
  ok  github.com/s1onique/leamas/internal/factory/dupcode  292.171s
```

Baseline verification (`leamas factory verify dupcode-baseline`)
re-confirms that the V4 algorithm's only stable ≥40-line /
≥400-token clone is the canonical 504-token claim_commands ↔
evidence_commands pair; the file is identical to the
CORRECTION04 baseline.

## Files touched

The corrective patch touches **exactly** the five files
identified by the reviewer. Total change footprint:

```text
 .../factory/dupcode/v4_baseline_forensics_504_maximality_test.go  | 3 +--
 internal/factory/dupcode/v4_baseline_forensics_504_trace_test.go   | 3 +--
 internal/factory/dupcode/v4_baseline_forensics_facts_test.go       | 4 +---
 internal/factory/dupcode/v4_baseline_forensics_helpers_test.go     | 3 +--
 internal/factory/dupcode/v4_pipeline_trace_test.go                 | 5 +++--
 5 files changed, 7 insertions(+), 11 deletions(-)
```

No production code (the `internal/factory/dupcode` *runtime*
files) was touched. No documentation, doctrine, factory, or
verifier source was touched.

## Detached evidence and gate summary

After the corrective commit landed, the evidence and
gate-summary artefacts under `.factory/` were regenerated
**from a clean tree** and bound to the new `HEAD` and
`HEAD^{tree}`:

  - `.factory/gate-summary.json` — fresh `generated_at`
    (2026-07-16T22:18:05Z, set when `make gate` finished),
    `tool: "leamas factory gate"`, `overall_status: "pass"`,
    `checks_total: 20`, `checks_passed: 20`,
    `checks_failed: 0`, `checks_unavailable: 0`. All 20 checks
    enumerated in the artefact body. `toolchain: gofmt` is
    included and reports OK.
  - `.factory/dupcode-baseline.json` — unchanged from
    CORRECTION04 (still the canonical 504-token pair).
  - `.factory/ACT-LEAMAS-FACTORY-DUPCODE-V4-CANONICAL-MAXIMAL-COMPONENT-MERGE01-CORRECTION05-gate.log`
    — fresh detached gate log bound to the new commit and
    tree OIDs.
  - `.factory/ACT-LEAMAS-FACTORY-DUPCODE-V4-CANONICAL-MAXIMAL-COMPONENT-MERGE01-CORRECTION05-evidence.json`
    — detached post-commit evidence file with `commit_oid`
    and `staged_tree_oid` taken from the new HEAD and
    `HEAD^{tree}` respectively, artefacts re-hashed fresh.

The tracked close report continues **not** to embed its own
tree OID; the binding lives entirely in the detached `.factory/`
artefacts.

## Range digest

The verdict required an **explicit range digest** covering
CORRECTION04 through the formatting commit. The digest is
generated with:

```bash
go run ./cmd/leamas factory digest \
  --range b82002c..HEAD \
  --output \
    .factory/ACT-LEAMAS-FACTORY-DUPCODE-V4-CANONICAL-MAXIMAL-COMPONENT-MERGE01-CORRECTION05-range-digest.txt
```

The artefact begins with `DIGEST_MODE: range` and enumerates
the single commit, the file deltas, and the policy-verifier
re-confirmation, in the same shape as prior range digests in
the audit trail.

## Skipped or deferred checks

None. The reviewer explicitly forbade CORRECTION06. All gates
listed in the CORRECTION04 verification matrix plus the
freshness of the gate summary and the explicit range digest
have been re-checked.

## Follow-ups

  1. Author the performance ACT
     `ACT-LEAMAS-FACTORY-DUPCODE-V4-ALL-PAIRS-MATERIALIZATION-PERFORMANCE01`.
     The reviewer states it is **algorithmically ready** but
     was procedurally blocked on the verification state
     repaired by CORRECTION05. No further algorithmic
     correction is indicated before that work proceeds.
  2. Keep running `gofmt -w` immediately after lifecycle-banner
     edits so that mechanical formatting drift does not recur.
     The repo gate already includes `gofmt`; the drift here
     was introduced between commit and upload, not at the
     `go test` boundary.
