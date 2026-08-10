# ACT-LEAMAS-FACTORY-CLOSURE-PROTOCOL-V2-SUBJECT-OBSERVATION-AUTHORITY01-CORRECTION03 (R6-A-CORRECTION03)

## Status

**OPEN — R6-A-CORRECTION03 / EVIDENCE HYGIENE**

This ACT records the R6-A-CORRECTION03 evidence-hygiene
correction. The R6-A-CORRECTION02 commit (`00d7b95`) was
functionally correct, but the authoritative digest exposed
one acceptance failure (P0 trailing whitespace in the ACT
doc that broke `git diff --check`) and two contract
language inaccuracies (P1 prose claims about
`filepath.Clean` and `bare` that did not match the
documented Go and Git contracts).

This ACT closes those three points. No production code
changes; the substantive R6-A boundary is unchanged from
`00d7b95`.

```text
R6_A_PRE_CORRECTION02=72cd1ab
R6_A_POST_CORRECTION02=00d7b95
R6_A_CORRECTION03=PASS

SUBJECT_OBSERVATION_BOUNDARY=GOOD
SUBJECT_HEAD_AUTHORITY=PASS
SUBJECT_TREE_AUTHORITY=PASS
SUBJECT_DETACHED_AUTHORITY=PASS
STATUS_AUTHORITY=PASS
REFS_AUTHORITY=PASS
TOPOLOGY_TRANSPORT=PASS

GIT_DIFF_CHECK=PASS
PORCELAIN_Z_PATH_PROSE_ACCURATE=true
BARE_GIT_SYNTAX_ACKNOWLEDGED=true
```

## Base

```text
BASE_COMMIT=00d7b9524738b832aeb87e43035939341be56543
R6_A_PRE_CORRECTION02=00d7b95
R6_A_POST_CORRECTION02=00d7b95
R6_A_CORRECTION03=PASS
```

## Mission

Close the evidence-hygiene gaps the R6-A-CORRECTION02 ACT
left open. No production code changes; tests-only and
doc-only.

```text
1. Remove literal trailing whitespace from the
   R6-A-CORRECTION02 ACT while retaining the
   trailing-space example unambiguously.
2. Correct the filepath.Clean prose in
   subject_observation_inventory_test.go: the function
   performs lexical canonicalization; it does NOT trim
   arbitrary whitespace.
3. Correct the bare-repository prose in
   subject_observation_inventory.go: bare is valid Git
   porcelain syntax but unsupported by Closure Protocol V2
   subject authority. The implementation can still reject
   bare, but the prose must acknowledge the Git contract
   rather than describe bare as malformed syntax.
4. Verify:
   git diff --check HEAD~1..HEAD
   go test -count=1 -run '<R6-A tests>' ./internal/factory/closure/
   git status --short
```

One forward corrective commit if needed. No production
code changes.

## Hard scope (enforced)

The R6-A test and doc surface is the only allowed edit
target. Forbidden:

```text
new subprocess gateway
raw os/exec
new Git client abstraction
new publication authority
new plan authority
new evidence-completeness rule
B1 binary integration
GateCollector integration
factory close execute changes
ClineMM changes
PRODUCTION CODE CHANGES
```

The current diff (CORRECTION03) is limited to:

```text
docs/acts/ACT-LEAMAS-FACTORY-CLOSURE-PROTOCOL-V2-
    SUBJECT-OBSERVATION-AUTHORITY01-CORRECTION02.md   (whitespace)
internal/factory/closure/subject_observation_inventory.go
    (prose)
internal/factory/closure/subject_observation_inventory_test.go
    (prose)
```

`closure_protocol_v2_executor.go` and the other production
files are not touched in CORRECTION03.

## Phase 1 — trailing whitespace (delivered)

The R6-A-CORRECTION02 ACT contained a table that documented
the trailing-space adversarial example with literal
trailing whitespace:

```text
trailing-space    : /tmp/wt trailing-space<SP>
```

That literal whitespace broke `git diff --check` even
though the example itself is correct. The corrected
rendering uses the `<SP>` token to render the trailing
space unambiguously without introducing literal trailing
whitespace into the ACT file. The semantic example is
unchanged. The whitespace error is gone.

## Phase 2 — filepath.Clean prose (delivered)

The R6-A-CORRECTION02 test incorrectly claimed:

```text
filepath.CClean strips leading and trailing whitespace
because they are not meaningful to the OS file system
```

`filepath.CClean` performs purely lexical canonicalization
(separator cleanup, `./..`). It does not generally strip
ordinary spaces from a path component. The corrected
prose is honest:

```text
filepath.CClean performs the repository's lexical path
canonicalization (separator cleanup, ./.., etc.) but it
does NOT generally trim arbitrary whitespace from a path
component. The matrix therefore proves that whitespace
and embedded newlines survive the parser without
TrimSpace: the parser does not call TrimSpace on the
path, and filepath.CClean does not strip the whitespace
either, so the bytes round-trip from input to canonical
Path.
```

This matches the documented Go contract. The matrix
behaviour is unchanged.

## Phase 3 — bare-repository prose (delivered)

The R6-A-CORRECTION02 parser prose described `bare` as
"a malformed structural token". Git's porcelain worktree
format explicitly defines `bare` as a legitimate boolean
attribute for bare worktrees. The implementation
correctly rejects bare (Closure Protocol V2 operates on a
real working repository), but the prose should
acknowledge the Git contract rather than describe bare
as malformed syntax. The corrected prose is honest:

```text
R6-A-CORRECTION02/CORRECTION03: `bare` is valid Git
porcelain syntax (documented as a legitimate boolean
attribute for bare worktrees) but is outside the
Closure Protocol V2 subject-observation domain. The
parser therefore rejects bare fail-closed as an
unsupported worktree record, not as malformed syntax:
the rejection is intentional policy, not a wire
violation. A future R6-X can extend the authority to
bare by relaxing this branch; today bare is treated
as an observable structural field rather than a
non-structural annotation.
```

The implementation can still reject bare (it does). The
rejection is exercised by the existing
`TestSubjectWorktreeInventoryParserRejectsMatrix`
generic "unknown token" row.

## Verification (PASS)

```text
git diff --check HEAD~1..HEAD                          # clean
gofmt -l <R6-A files>                                  # clean
go vet ./internal/factory/closure/                    # clean
go build ./...                                         # clean
CGO_ENABLED=0 go test -count=1 -run \
    'TestSubjectWorktreeInventoryParserPreservesPathBytes|TestSubjectWorktreeInventoryParserRejectsMatrix' \
    ./internal/factory/closure/                       # ok
git status --short                                     # clean
```

Do NOT run in Cline/editor context unless explicitly authorized:

```text
make factorize
make gate-dupcode
make gate
```

Report: NOT RUN.

## Acceptance

```text
GIT_DIFF_CHECK=PASS
PORCELAIN_Z_PATH_PROSE_ACCURATE=true
BARE_GIT_SYNTAX_ACKNOWLEDGED=true

PRODUCTION_FILES_CHANGED=none

R6_A=PASS
R6_B_READY=true
```

## Successor

Only after R6-A-CORRECTION03 PASS:

```text
ACT-LEAMAS-FACTORY-CLOSURE-PROTOCOL-V2-
BINARY-GATE-INTEGRATION01
```
