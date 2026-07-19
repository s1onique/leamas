# ACT-LEAMAS-FACTORY-DIGEST-STAGED-STATUS-CLASSIFICATION01-CORRECTION01

## Title

Repair the staged / dirty / range digest to:
1. support the full set of Git `--name-status -z` status letters
   (`T`, `X`, `B`);
2. make the 30% similarity policy explicit at every oracle
   reference;
3. lock the range regression suite on exact-equality assertions;
4. emit rendered paths through `PathEscape` so unusual filenames
   survive through to the final digest;
5. align `NormalizeGitStatusToken` and `SplitNULRecords` with the
   parser's stricter contracts;
6. record a bounded `go test ./...` attempt honestly.

## Status

Implemented (PARTIAL; full canonical verification still blocked on
the previously documented ACTs).

## Context

The parent ACT
(`ACT-LEAMAS-FACTORY-DIGEST-STAGED-STATUS-CLASSIFICATION01`)
fixed the original four-added/one-modified misclassification and
made the manifest agree path-for-path with `git diff --cached
--name-status` (at the lowered 30% similarity threshold). A
reviewer pass surfaced six contract defects that this correction
addresses:

1. Valid Git `T` (type-change), `X` (unknown), and `B` (broken
   pair) status letters would silently map to `M` and the digest
   would either report the wrong status or fail at the parser. Git
   emits `T` whenever a tracked path's file type changes
   (regular-file to symlink, regular-file to submodule, and so on).
2. The ACT and the implementation used a 30% similarity threshold
   (`--find-renames=30% --find-copies=30%`) for rename detection.
   Git's default is 50%. Plain `git diff --name-status` does not
   produce the same output, so calling the digest "in agreement
   with `git diff --name-status`" was misleading. The choice is
   defensible (the lower threshold keeps `R` consistent for the
   common "rename then small edit at destination" case), but the
   documentation and tests must say so explicitly.
3. `TestRangeMode_Addition`, `TestRangeMode_Modification`, and
   `TestRangeMode_Deletion` used a `for ... range` loop that
   silently passes on an empty render. The replacement uses
   exact-equality assertions.
4. The parser preserves filenames with embedded newlines, but the
   renderer wrote paths directly into a line-oriented Markdown
   layout, so a path with a literal newline would split one
   manifest entry across multiple lines and silently corrupt the
   digest. Git itself uses `-z` precisely so callers can render
   safely; the digest now uses `PathEscape` for every rendered
   path, with a documented round-trip via `ParseEscapedPath`.
5. `NormalizeGitStatusToken` accepted bare `R` and `C` without a
   numeric score even though `ParseGitStatusRecords` rejects them
   at the structured layer. `SplitNULRecords` documented a
   "drop trailing empty field" behaviour it did not implement.
6. The ACT body required a bounded `go test ./...` attempt; only
   `make factorize` and `make gate` were attempted before, with
   no unfiltered test-run attempt recorded.

## Goal

Eliminate each P1/P2 defect above without regressing the original
manifest/status correctness for the four-added/one-modified
reproduction.

## Hard constraints

The parent ACT's hard constraints remain in force. In particular:

1. Status classification must continue to come from `git diff
   --name-status -z`, not from boolean presence flags.
2. NUL-delimited Git output must be the source. Paths must be
   rendered escaping in the digest so unusual filenames survive.
3. No force-push, no contract version bump, no `make factorize`
   claim unless it actually finished.

## Approach

1. Extend the parser's accepted status letters to `A`/`M`/`D`/
   `T`/`U`/`X`/`B` plus rename/copy; add corresponding `Kind*`
   constants and a typed `Unknown`-style bucket for `X` and `B`.
2. Add `StatusTypeChanged`, `StatusUnknown`, `StatusBrokenPair`
   and the matching `FileStats` fields (`TypeChangedFiles`,
   `UnknownFiles`, `BrokenPairFiles`).
3. Map the new letters through `statusToHuman` (range mode) and
   `BuildRangeManifest` so range mode renders `T` instead of
   silently falling back to `modified`.
4. Add a canonical `PathEscape` helper plus a `PathEscapeRoundTrip`
   table-driven unit test and three rendering integration tests
   (staged, range, parser-level round-trip on every control byte).
5. Replace the three vacuous range-mode tests with exact-equality
   assertions; rename `TestRangeMode_StableAcrossACT` to
   `TestRangeMode_MixedAllKinds` and make all four kinds
   (add/mod/del/rename) load-bearing.
6. Tighten `NormalizeGitStatusToken` so it rejects bare `R` and
   `C`; fix `SplitNULRecords` to drop the single trailing empty
   element that arises when the input ends with NUL.

## Test coverage

### Parser unit tests

`git_status_parser_test.go` now covers 24 cases including the new
`T` (regular file to symlink / submodule), `X`, and `B` records.
`TestParseGitStatusRecords_DoesNotPanic` covers lowercase rewrites
which `ParseGitStatusRecords` rejects but never panics on.

### Path-escape unit tests

`digest_status_path_escape_test.go` covers plain ASCII, leading
dashes, spaces, tabs, newlines, CRLF, backslash, NUL, DEL byte,
Unicode, and a combined NUL+tab+newline+backslash path through the
round-trip `PathEscape` / `ParseEscapedPath`. Every test asserts the
canonical escaped form contains no unescaped LF, CR, NUL, or tab.

### Path-escape rendering integration tests

* `TestStagedStatus_NewlinePathInManifest`: a staged file whose
  name contains an embedded newline must render the manifest
  line on a single visual line in escaped form, with the literal
  newline absent from the rendered output.
* `TestRangeStatus_NewlinePathInManifest`: same contract for
  range mode.

### Range mode tests

`digest_status_range_test.go` uses exact-equality
(`assertManifestLinesExact`) for add/mod/del/rename, plus
`TestRangeMode_MixedAllKinds` (all four) and
`TestRangeMode_TypeChange` (regular file to symlink). Skips
gracefully when the host filesystem rejects `os.Symlink`.

## Verification

Run and record exact commands and exit statuses:

```bash
gofmt -w internal/factory/digest/*.go

go test ./internal/factory/digest -count=1
go test ./cmd/leamas -count=1

go test ./internal/factory/digest \
  -run 'Test.*(ParseGit|NormalizeGit|SplitNUL|BuildManifest|StagedStatus|DirtyStatus|RangeMode|EvidenceHash|PathEscape)' \
  -count=1 -v

go vet ./...

CGO_ENABLED=0 go build -trimpath \
  -o bin/leamas ./cmd/leamas

./bin/leamas factory verify llm-friendly
./bin/leamas factory verify agent-context
./bin/leamas factory verify forbidden-patterns

# Bounded tree-wide attempt, recorded honestly.
timeout 480 go test ./...
```

`make factorize` and `make gate` remain blocked on the previously
documented ACTs and are explicitly out of scope for this
correction.

## Self-hosting proof

After implementation and focused tests pass, build a fresh Leamas
binary outside the repository and confirm the rendered digest
matches the literal oracle at the lowered threshold:

```bash
CGO_ENABLED=0 go build -trimpath \
  -o /tmp/leamas-digest-status ./cmd/leamas

# Stage the corrected ACT changes.
git add -A

git diff --cached --name-status -z \
  --find-renames=30% --find-copies=30% HEAD -- \
  | tee /tmp/digest-status-git-oracle.txt

/tmp/leamas-digest-status factory digest \
  --staged --output /tmp/digest-status-proof.txt
```

Every line of `CHANGESET_MANIFEST` must match the oracle byte-for-byte.
A symlink-creation test in `/tmp` must render `T  linked.go` (not
`M  linked.go`) end to end through range mode.

## Acceptance criteria

1. Valid Git `T` records parse and render as `T`, end to end.
2. Valid Git `X` and `B` records parse and render as `X` / `B`.
3. `CHANGESET_STATS` carries `type_changed_files`,
   `unknown_files`, `broken_pair_files` fields.
4. The 30% threshold is named explicitly in the oracle command
   everywhere it appears; tests never claim a literal-Git-default
   equivalence.
5. Vacuous range tests are replaced with exact-equality
   assertions; `TestRangeMode_MixedAllKinds` is load-bearing.
6. Paths with embedded tab / newline / backslash / control bytes
   render on a single line; the escaped form round-trips through
   `PathEscape` / `ParseEscapedPath`.
7. `NormalizeGitStatusToken` rejects bare `R` / `C`; lowercase
   rewrites are rejected.
8. `SplitNULRecords` matches its documented contract.
9. `go test ./internal/factory/digest -count=1` is green.
10. `go vet ./...` is clean.
11. `./bin/leamas factory verify ...` are all PASSED.
12. Self-hosting proof agrees with the literal Git oracle at the
    lowered threshold.
13. The bounded `go test ./...` attempt is recorded honestly with
    timeout budget, exit status, and last completed phase.

## Closure rule

Closure of this CORRECTION01 ACT requires satisfying all the
acceptance criteria above. The parent ACT remains the primary scope;
the parent ACT's close report should be updated to reflect the
PARTIAL → CORRECTION01 path once the criteria are green and the
bounded `go test ./...` has been recorded with honest exit status.

If `make factorize` / `make gate` remain blocked on the documented
duplicate-code performance issue and the criteria above are
otherwise green, mark this CORRECTION01 ACT as **CLOSED** and
update the parent ACT's close report to a single PARTIAL note that
points forward to the still-out-of-scope canonical verification.
