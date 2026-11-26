# ACT-LEAMAS-RANGE-PATCH-BUDGET-AUTHORITY-01-AMENDMENT-01 — EVIDENCE ANNEX

This annex holds the VERIFICATION EVIDENCE for
ACT-LEAMAS-RANGE-PATCH-BUDGET-AUTHORITY-01-AMENDMENT-01.

The close report itself is at
`docs/close-reports/ACT-LEAMAS-RANGE-PATCH-BUDGET-AUTHORITY-01-AMENDMENT-01.md`.

## BASELINE GATE DISPOSITION

`CGO_ENABLED=0 make gate-fast` was run and reports GATE FAILED. The
failure is PRE-EXISTING and unrelated to this amendment:

- `go test -short` fails in `internal/execution` on
  `TestValidateRetainedPipeTopology` (a macOS pipe-topology test).
  The SAME test fails identically at the baseline `480dcad` in a clean
  worktree. This amendment changed ZERO files in
  `internal/execution`.
- The `language` and `tooling-boundaries` verifiers report a shell
  script at `.factory/modcache/golang.org/x/tools@v0.48.0/internal/diff/lcs/git.sh`.
  That path is inside the git-ignored local Go module cache
  (`.gitignore:13:.factory/*`) and is untracked. It is an upstream
  dependency artifact, not a repository file.

No baseline gate was repaired. No full-suite campaign was authorised.

### Closure witnesses that require a COMMITTED caller source

Eight closure witnesses (`TestClosureCLIV2VerifierMacHandoff*`,
`TestClosureCLIV2R2CRExactTipDogfood`,
`TestClosureExactSubjectBinary*`) require the CALLER's worktree to be
clean and its source COMMITTED before they build their fixture:

```text
caller worktree not clean:
real Leamas umbrella requires a committed caller source; dirty status:
```

They therefore fail on a working tree that carries uncommitted
changes, regardless of whether those changes are correct. This is a
precondition of the witnesses, not a defect in the change under test.

Established by three controlled measurements:

1. With the amendment applied but UNCOMMITTED: all 8 fail with the
   precondition message.
2. With the amendment stashed (pristine `480dcad`, clean tree): all 8
   PASS.
3. With the amendment applied as a COMMIT (clean tree): all 8 PASS.

Measurement 3 is the decisive one: it shows the amendment satisfies
the witnesses once the source is committed.

The failure-set comparison over `./cmd/leamas/...` and
`./internal/factory/closure/...` at `--short`:

| Tree state | Failure count |
| --- | --- |
| Baseline `480dcad` (clean) | 35 |
| Amendment, UNCOMMITTED | 41 |
| Amendment, COMMITTED | 34 |

The committed-tree failure set is a strict SUBSET of the baseline's:
ZERO new failures, and the baseline's
`TestClosureCLIEndToEndSubprocess` passes in the committed amendment
tree (it is flaky).

### Full-suite regression proof

The same comparison over the ENTIRE repository at `--short`
(`go test -short ./...`), comparing the exact sorted sets of failing
test names:

| Tree state | Failing tests |
| --- | --- |
| Baseline `480dcad` (clean worktree) | 44 |
| Amendment, COMMITTED (clean worktree) | 36 |

Set difference:

```text
only in the amendment tree (regressions):  <none>
only in the baseline tree (pass with the amendment): 8 tests
```

The 8 are `TestApprovalCensusExplicitOracleHashStable`,
`TestApprovalCensusMatchesResolvedDeclarations`,
`TestApprovalCensusRunSharedDupcodeBaselineIsPackageFunction`,
`TestCanonicalProductionPolicyAndApprovalTruth`,
`TestClosureCLIEndToEndSubprocess`,
`TestExplicitApprovalInventoryPreservesOracleHash`,
`TestExplicitApprovalRecordEquivalence` and
`TestRootResolver_RepoRoot` — all environment- or state-dependent and
flaky at the baseline.

CONCLUSION: the amendment introduces ZERO test regressions. Every
failure that remains is present at the baseline too.

### Observed flake: `TestEvidence_NoFlag_M11_Compatibility`

One full-suite run reported `TestEvidence_NoFlag_M11_Compatibility`
failing. It passes in isolation and is a PRE-EXISTING timing flake in
`ModeDirty`, unrelated to this amendment.

The M11 witness compares two digest strings generated microseconds
apart. The dirty-mode digest embeds a second-resolution wall-clock
timestamp (`digest.go`: `createdAt := time.Now().UTC().Format(time.RFC3339)`,
rendered as `DIGEST_CREATED_AT` and `Generated at`). When the two
generations straddle a second boundary the strings differ.

Reproduced with a purpose-built probe: the ONLY differing lines are

```text
DIGEST_CREATED_AT: 2026-11-26T01:46:02Z  vs  ...T01:46:03Z
Generated at:      2026-11-26T01:46:02Z  vs  ...T01:46:03Z
```

No evidence content differs. The dirty-mode path was not modified by
this amendment; the affected code is `internal/factory/digest/digest.go`
and `internal/factory/digest/evidence_overlay_protection_test.go`,
neither of which is in the changed set. The follow-up belongs to the
overlay/evidence-overlay ACT line, not to this one.
