# ACT-LEAMAS-FACTORY-DIGEST-STAGED-STATUS-CLASSIFICATION01

## Title

Correct the targeted digest's staged and dirty-mode change
classification so `CHANGESET_MANIFEST` and `CHANGESET_STATS` agree
exactly with Git's authoritative status for each path.

## Status

Implemented (pending close report).

## Context

A self-hosting review exposed this incorrect digest output:

```text
A  internal/factory/gate/gate.go
```

although the file already existed in `HEAD` and Git reported:

```text
M  internal/factory/gate/gate.go
```

The same digest consequently reported:

```text
added_files=5
modified_files=0
```

instead of:

```text
added_files=4
modified_files=1
```

The defect existed because the current dirty/staged file collection
recorded only whether a path is present in the staged or unstaged
diff. It discarded Git's actual change kind, after which
`BuildManifest` treated every tracked path that is staged but not
unstaged as `A`.

## Goal

For staged changes, make these digest sections agree with:

```bash
git diff --cached --name-status HEAD --
```

* `CHANGESET_MANIFEST`
* `CHANGESET_STATS`
* all manifest-derived risk signals and evidence hashes

For dirty/auto mode, classify tracked paths according to their net
change relative to `HEAD`, while preserving:

* staged-present metadata;
* unstaged-present metadata;
* staged and unstaged patch rendering;
* untracked-file handling;
* deterministic ordering;
* NUL-safe path handling.

## Hard constraints

1. Do not infer `A`, `M`, `D`, `R`, or `C` from the booleans
   `Tracked`, `StagedPresent`, or `UnstagedPresent`.
2. Obtain status information from Git's structured output.
3. Use NUL-delimited Git output. Do not parse human-formatted
   `git status` output.
4. Preserve filenames containing spaces, tabs, newlines, Unicode,
   leading dashes, and other unusual characters.
5. Do not invoke a shell to execute Git.
6. Do not silently turn malformed or unknown Git status records
   into `A` or `M`.
7. Preserve range-mode behavior unless a test proves that shared
   parsing must be corrected.
8. Preserve changed-file detail metadata and diff rendering.
9. Preserve deterministic lexical output ordering.
10. Do not bump the targeted-digest contract version merely for
    correcting status semantics; the schema is unchanged.
11. Do not regenerate or modify unrelated baselines.
12. Do not fix duplicate-code performance in this ACT.
13. Do not use the targeted digest itself as the sole oracle
    proving that the targeted digest is correct.

## Approach

1. Add a shared NUL-delimited parser
   (`internal/factory/digest/git_status_parser.go`) that turns the
   `diff --name-status -z` output into `[]GitChange` records.
2. Replace the changed-file data model so `ChangedFile` carries the
   authoritative `Kind` and `OldPath` instead of inferring them
   from boolean presence flags.
3. Rewrite `GetStagedFiles` and `GetDirtyFiles` to consume the
   parser's structured output. Staged mode gets the change kind
   directly from `git diff --cached --name-status -z` with rename and
   copy detection enabled (at the digest's lowered similarity
   threshold of 30%). Dirty mode gets the net status
   relative to `HEAD` via `git diff --name-status -z HEAD --`.
4. Simplify `BuildManifest` to project `ChangedFile` directly instead
   of inferring from `Tracked`/`StagedPresent`/`UnstagedPresent`.
5. Surface the explicit kind on the diff renderer's
   "Changed files" metadata so reviewers can read the kind straight
   off the digest.
6. Migrate `GetRangeFiles` to the shared parser so its renames
   index into the correct paths (the prior hand-rolled indexing
   skipped one path on rename entries, reporting only the source).

## Detailed contract tables

### Staged-mode manifest status

| Repository state                              | Manifest status |
| --------------------------------------------- | --------------- |
| Existing tracked file modified and staged      | `M`             |
| Newly staged path                              | `A`             |
| Existing staged deletion                       | `D`             |
| Existing staged rename                         | `R old -> new`  |

### Dirty-mode manifest status

| Repository state                                              | Manifest status |
| ------------------------------------------------------------- | --------------- |
| Existing file modified and staged                             | `M`             |
| Existing file modified only in worktree                       | `M`             |
| Existing file modified both staged and unstaged               | `M`             |
| New file staged                                               | `A`             |
| New staged file modified again in worktree                    | `A`             |
| Existing file deleted and staged                              | `D`             |
| Existing file renamed and staged                              | `R old -> new`  |
| Staged rename followed by an unstaged edit of the destination | `R old -> new`  |
| Untracked file                                                | `?`             |

### Normalize Git status tokens

```text
A     → A
M     → M
D     → D
R100  → R
R087  → R
C100  → C
C075  → C
U     → U
```

## Test coverage

### Unit tests

`git_status_parser_test.go` covers all 20 cases specified in the
ACT body: ordinary A/M/D/U records, renames and copies with and
without `100` similarity, paths containing spaces, tabs, newlines,
Unicode, leading dashes, multiple adjacent records, empty input,
truncated ordinary record, truncated rename, truncated copy,
unknown status, and empty destination path. A separate
`TestParseGitStatusRecords_DoesNotPanic` exercises additional
malformed inputs and confirms the parser never panics.

### Integration tests

`digest_status_staged_test.go` reproduces the original defect with
the exact five-file fixture (`internal/factory/gate/gate.go` plus
four new files). It reconciles the digest manifest and statistics
against the literal `git diff --cached --name-status HEAD --` for
every staged-mode scenario.

`digest_status_dirty_test.go` covers the full dirty-mode contract
table and verifies that tracked paths keep their net change, that
untracked files render as `?`, and that the manifest is sorted
lexicographically by path.

`digest_status_evidence_hashes_test.go` ensures the manifest,
statistics, and aggregate digest evidence hashes change when the
status letter flips from `M` to `A` (the original defect). It also
guards against hash values being hardcoded.

`digest_status_range_test.go` exercises `leamas factory digest
--range HEAD~1..HEAD` for additions, modifications, deletions, and
renames, and a mixed commit combining all three. Range mode now uses
the shared parser too, which fixes a pre-existing indexing bug that
was reporting renames without the `old -> new` half.

## Verification

Run and record exact commands and exit statuses:

```bash
gofmt -w internal/factory/digest/*.go

go test ./internal/factory/digest -count=1
go test ./cmd/leamas -count=1

go test ./internal/factory/digest \
  -run 'Test.*(Status|Manifest|Stats|NameStatus|Staged|Dirty|Range)' \
  -count=1 -v

go vet ./...

CGO_ENABLED=0 go build \
  -trimpath \
  -o bin/leamas \
  ./cmd/leamas

./bin/leamas factory verify llm-friendly
./bin/leamas factory verify agent-context
./bin/leamas factory verify forbidden-patterns

git diff --check
```

`make factorize` and `make gate` are known to exercise slow
duplicate-code live-tree paths on the Mint host. They were not
attempted in this ACT and remain blocked on the previously
documented ACTs (`ACT-LEAMAS-FACTORY-FACTORIZE-RUNNER-FIXTURE01` and
`ACT-LEAMAS-FACTORY-DUPCODE-PERF-RATCHET01`). The original defect is
resolved and the focused scope is green; full-tree canonical
verification is intentionally out of scope.

## Self-hosting proof

After implementation and focused tests pass:

1. Build a fresh Leamas binary outside the repository:

   ```bash
   CGO_ENABLED=0 go build \
     -trimpath \
     -o /tmp/leamas-digest-status \
     ./cmd/leamas
   ```

2. Stage the ACT changes:

   ```bash
   git add -A
   ```

3. Record the authoritative Git classification:

   ```bash
   git diff --cached --name-status -z \
     --find-renames=30% --find-copies=30% HEAD --
     | tee /tmp/digest-status-git-oracle.txt
   ```

4. Generate a staged digest outside the repository:

   ```bash
   /tmp/leamas-digest-status factory digest \
     --staged \
     --output /tmp/digest-status-proof.txt
   ```

5. Compare every staged path and status in `CHANGESET_MANIFEST`
   with the Git oracle: path-for-path agreement observed.
6. Independently recompute status counts from the Git oracle and
   compare them with `CHANGESET_STATS`:
   `added_files=7, modified_files=5` matches the 12 staged files.
7. Confirm at least one modified existing file is rendered `M`,
   not `A` — observed for `internal/factory/digest/file_evidence.go`,
   `internal/factory/digest/file_operations.go`,
   `internal/factory/digest/range_types.go`,
   `internal/factory/digest/review_manifest.go`, and
   `internal/factory/digest/review_test.go`.
8. Run `git diff --cached --check` to verify whitespace hygiene
   (clean run).

The generated proof files remain outside the repository
(`//tmp/digest-status-...`) and must not be committed.

## Acceptance criteria

1. A staged modification of an existing tracked file renders `M`.
2. A newly staged path renders `A`.
3. A staged deletion renders `D`.
4. A staged rename renders `R old -> new`.
5. Staged manifest output agrees path-for-path with Git's
   authoritative index classification.
6. Dirty-mode manifest output represents net change relative to
   `HEAD`.
7. Staged and unstaged presence metadata remains accurate.
8. `CHANGESET_STATS` agrees exactly with the manifest.
9. The original four-added/one-modified reproduction reports
   `added_files=4, modified_files=1`.
10. Rename/copy scores are normalized to `R`/`C`.
11. Unusual filenames are preserved through NUL-safe parsing.
12. Malformed structured Git output fails closed.
13. Range-mode status behavior does not regress.
14. Diff rendering does not regress.
15. Evidence hashes bind the corrected manifest and statistics.
16. Focused digest and CLI tests pass.
17. `go vet ./...` passes.
18. Static Leamas build passes.
19. Self-hosting staged proof agrees with literal Git output.
20. Documentation and close report accurately distinguish scoped
    success from unrelated full-gate performance limitations.

## Closure rule

Do not close the ACT solely because the original `gate.go`
example is corrected. Closure requires parser unit coverage,
staged and dirty integration coverage, exact manifest/statistics
agreement, range-mode regression coverage, self-hosting comparison
against literal Git output, clean patch hygiene, committed
implementation and evidence, and honest full-verification accounting.

If the scoped implementation and all focused checks pass but
canonical full-tree verification remains blocked solely by the
previously documented duplicate-code runtime, mark the ACT
`PARTIAL` unless repository doctrine explicitly permits scoped
closure with retained baseline evidence.

## Out of scope

* `ACT-LEAMAS-FACTORY-FACTORIZE-RUNNER-FIXTURE01`
* `ACT-LEAMAS-FACTORY-DUPCODE-PERF-RATCHET01`
