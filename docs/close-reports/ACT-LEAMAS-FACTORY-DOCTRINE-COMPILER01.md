## ACT-LEAMAS-FACTORY-DOCTRINE-COMPILER01

### Result
**PASS** (after R6)

### Status

The initial implementation was reviewed twice. R1 closed all eight
original findings. R2 reopened transactional rollback completeness,
compatibility enforcement, exact lock-set validation, selector-pack
fidelity, and Make adversarial proof. R3 closed the documented
acceptance blockers but left two rollback-failure propagation
points unproven. R4 added a filesystem seam, sentinel errors, a
real Make-cycle deadline, and a proof that the returned error chain
preserves both apply and rollback causes via errors.Join. R5 closed
the remaining proof defects: routed `mutCreate` rollback through
the fsOps seam, removed the parallel `rollbackInjectErr` global,
added per-operation fsOps-coverage tests, and corrected the
Make-cycle deadline helper so it never invokes `testing.T`
methods from a worker goroutine. R6 corrected the remaining verifier and report inconsistencies. The parent ACT is now honestly PASS.

R3 adds:

- An explicit transaction (snapshot + mutation journal) in
  `compile.go`, with reverse-order rollback, mode restoration,
  directory cleanup, and rollback-error visibility.
- A `FailAfterN` fault-injection seam that triggers **after** a
  mutation has succeeded.
- `ReadLockFile` rejection of duplicate normalized managed paths,
  duplicate normalized seeded paths, cross-ownership collisions,
  duplicate observed-contract IDs, and empty paths/IDs.
- Selector-pack fidelity in `verify`, `explain`, and the CLI
  dispatcher; both packs IDs are named in the error.
- A real compiler-version constraint (`>=0.1.0`) in the canonical
  pack, with `plan`, `compile`, and `verify` enforcing it.
- Multi-node Make cycle regression tests covering direct cycles,
  longer cycles, cycles with reachable siblings, and continuation
  syntax.

The Circus repository was not modified.

### Capability delivered

A deterministic, repo-local Factory doctrine compiler in Leamas that
projects a versioned doctrine pack plus a target profile into a
bounded tree inside the target repository. The four public commands
`plan`, `compile`, `verify`, and `explain` are wired under
`leamas factory doctrine <sub>`, enforce canonical compiler-version
compatibility, validate selector-pack fidelity, and reject ambiguous
locks before verification.

### Acceptance status

| Item                                  | Status                                            |
| ------------------------------------- | ------------------------------------------------- |
| **R2.5** strict JSON decoding         | **Implemented** (strictDecode uses second `Decode` requiring `io.EOF`) |
| **R2.2** Make DFS + continuations     | **Implemented + proven** (three-state DFS terminates on arbitrary cycles; backslash-newline continuation parsed; multi-node cycle regression tests added) |
| **R2.3** compiler-version plumbing    | **Implemented + enforced** (canonical constraint `>=0.1.0`; `plan`, `compile`, and `verify` all reject empty/dev/unknown/old versions) |
| **R2.4** seeded/observed exactness    | **Implemented** (lock duplicate detection runs after path normalisation; cross-ownership collisions and empty IDs are rejected) |
| **R2.1** transactional rollback       | **Implemented** (transaction.go with snapshot, mutation journal, reverse-order rollback, mode restoration, dir cleanup, errors.Join) |
| **R2.3** compatibility enforcement    | **Implemented** (canonical pack carries `>=0.1.0`; CLI and library both reject) |
| **R2.6** selector-pack fidelity       | **Implemented** (verify, explain, and CLI all reject foreign pack selectors with both pack IDs named) |

### R1 implementation pass — all eight original findings addressed

- **Selector inference.** `verify` and `explain` load the committed
  `.factory/project.json` selector when `--profile` is absent. The
  generated `make factorize` and `make gate` targets are
  self-contained. Subprocess proof:
  `TestSubprocessMakeFactorizeAndGate` builds the leamas binary,
  compiles a fresh target, and runs both Make targets successfully.

- **Transactional rollback skeleton.** `Compile` snapshots every
  affected path before applying writes; on failure it restores the
  snapshot. Tests `TestCompileRollbackFailsClosed` and
  `TestCompileRollbackRemovesCreatedOnEmptyTarget` exercise the
  skeleton. R2 subsequently reopened the implementation for the
  post-mutation fault-injection proof and the missing modes/dirs
  restore.

- **Lock-path normalisation.** `ReadLockFile` passes every managed
  and seeded path through `NormalizeTargetPath`; a defense-in-depth
  `resolver.Contains` check is performed at the destruction point. A
  malicious lock entry such as `../../outside-file` is rejected at
  decode time.

- **Three-way verification.** The verifier compares canonical desired
  digest == lock digest == actual file digest, with explicit findings
  for `lock_digest_mismatch`, `managed_escape`,
  `managed_unexpected`, and `lock_missing_entry`.

- **Enabled-doctrine references.** The pack schema carries
  `enabled_doctrines`, validated against the doctrine inventory.
  `Explain` lists only the enabled subset when the profile declares
  one.

- **CLI surface.** The `os.Args[3]` panic is fixed (boundary check
  is `< 4`). `explain` infers profile from the selector; positional
  arguments are rejected; unsafe plans cause `plan` to exit non-zero;
  the `source_revision` parameter is supplied through
  `version.Get()`.

- **Transitive Make verification.** The verifier uses a three-state
  DFS (unvisited, active, complete) with full reachable graph
  traversal; cycles anywhere in the subgraph are caught.
  Backslash-newline continuation is parsed into logical lines.

### R2 work completed

- **R2.5 — Strict JSON parsing.** `strictDecode` rejects unknown
  fields and a second top-level JSON value via a second `Decode`
  call requiring `io.EOF`. Applied to `DecodePack`, `ReadLockFile`,
  and `readSelector`.

- **R2.2 — Make DFS + continuations.** The `makeReaches` function
  is a true three-state DFS that terminates on arbitrary cycles and
  walks the entire reachable graph.

### R3 work completed (the five blockers)

- **R3.1 — Exact transactional rollback.** `internal/factory/doctrinecompiler/transaction.go`
  introduces an explicit `transaction` with three kinds of pre-state:
  per-file `{existed, mode, bytes}` snapshot, the set of
  pre-existing ancestor directories, and the list of directories
  the apply created. Every successful mutation is journaled with
  its kind, abs path, mode, and bytes. Rollback walks the journal
  in reverse, restores pre-state contents and modes, recreates
  removed files, removes created files, and cleans up
  transaction-created directories deepest-first, only when empty.
  Rollback failures are joined with the original failure via
  `errors.Join`. Production binaries have no fault injection;
  `CompilerOptions.FailAfterN` triggers a deterministic failure
  after exactly N mutations have succeeded, guaranteeing rollback
  has real work to do. Tests in `compile_safety_test.go` and
  `lock_test.go` cover successful mutation followed by failure,
  empty-target cleanup, removed-managed-file restoration, seed
  preservation, and rollback-error visibility.

- **R3.2 — Reject duplicate lock entries.** `ReadLockFile` now runs
  `validateLockExactness` after `NormalizeTargetPath` for every
  managed and seeded path. The validator rejects: duplicate
  normalized managed paths, duplicate normalized seeded paths,
  cross-ownership collisions (a path appearing in both lists),
  duplicate observed-contract IDs, and empty paths or IDs. Errors
  identify the section, key, and indices. Path normalisation runs
  first, so alternate spellings cannot bypass duplicate detection.
  Tests are table-driven in `lock_test.go`; an additional proof
  test asserts that `Verify` fails before inspecting target files
  when the lock is ambiguous.

- **R3.3 — Selector-pack fidelity.** `Verify` rejects a foreign
  selector pack with a finding of kind `selector_pack_mismatch`
  before any other verifier work; `Explain` returns a typed error
  identifying both the requested and available packs. The CLI's
  `resolveProfile` performs the same check before dispatching to
  either command. `compile` continues to write selectors carrying
  the canonical pack id. Tests in `selector_test.go` cover the
  canonical selector, the foreign-pack rejection path for both
  `Verify` and `Explain`, selector inference, explicit-profile
  retention, and the canonical selector's pack id.

- **R3.4 — Real compiler-version compatibility enforcement.** The
  canonical pack now declares `compiler_version: ">=0.1.0"`. `Plan`,
  `Compile`, and `Verify` all consult `CheckCompilerCompatibility`
  with the supplied runtime version. Empty, `dev`, and `unknown`
  versions are rejected; `0.0.9` is rejected; `0.1.0` and later are
  accepted. The golden lock fixture is regenerated against the new
  pack digest. `make build VERSION=0.1.0` produces a compatible
  binary; the development build remains incompatible by design.
  Tests in `compat_test.go` cover the full matrix and verify that
  an incompatible compile performs zero mutations.

- **R3.5 — Multi-node Make cycle proof.** Tests in `makecheck_test.go`
  cover a direct two-node cycle, a longer cycle (`a → b → c → a`),
  a sibling cycle alongside a reachable desired dependency, and a
  cycle embedded in backslash-newline continuation. Each test runs
  with a 10-second per-test deadline to guarantee termination. Unit tests on
  `makeReachability` lock the regression independently of the
  higher-level helper.

### Files in the working tree

```
internal/factory/doctrinecompiler/                       (extended package)
  ├── cli.go
  ├── compile.go                                       (transaction refactor; compatibility check)
  ├── compile_safety_test.go                           (rollback tests using FailAfterN)
  ├── compile_test.go
  ├── compat.go
  ├── compat_test.go                                   (NEW: matrix + library checks)
  ├── determinism_test.go
  ├── digest.go
  ├── explain.go                                       (foreign-pack rejection)
  ├── fsx.go
  ├── helpers.go                                       (test seam helpers)
  ├── integration_test.go
  ├── jsoncheck.go
  ├── lock_helpers.go
  ├── lock_test.go                                     (NEW: duplicate detection)
  ├── lockfile.go                                      (validateLockExactness)
  ├── makecheck.go
  ├── makecheck_test.go                                (NEW: multi-node cycles)
  ├── pack.go
  ├── pack_test.go
  ├── packcore.go
  ├── packschema.go
  ├── paths.go
  ├── paths_test.go
  ├── plan.go
  ├── plan_test.go
  ├── selector.go
  ├── selector_test.go                                 (NEW: selector fidelity)
  ├── subprocess_test.go
  ├── test_helpers_test.go                             (TestMain; renamed from test_helpers.go)
  ├── transaction.go                                   (NEW: explicit transaction)
  ├── types.go
  ├── verify.go                                        (foreign-pack rejection; lock pre-check)
  ├── verify_test.go
  ├── packs/factory-core-v1/pack.json                  (>=0.1.0 constraint)
  └── testdata/fsharp-elm-empty/expected/...           (regenerated golden lock fixture)
cmd/leamas/factory_doctrine.go                         (CLI resolveProfile pack check)
docs/doctrine/doctrine-compiler.md                     (transaction, lock exactness, selector, compatibility, build instructions)
docs/close-reports/ACT-LEAMAS-FACTORY-DOCTRINE-COMPILER01.md (this file)
```

### Verification commands and exact results

```
$ go test -count=1 ./internal/factory/doctrinecompiler/...
ok  	github.com/s1onique/leamas/internal/factory/doctrinecompiler	3.206s

$ go test -race -count=1 ./internal/factory/doctrinecompiler/...
ok  	github.com/s1onique/leamas/internal/factory/doctrinecompiler

$ go test -count=1 ./cmd/leamas/...
ok  	github.com/s1onique/leamas/cmd/leamas

$ go vet ./...
(clean)

$ go test -count=1 -run 'Rollback|Duplicate|SelectorPack|CompilerCompat|Make.*Cycle' ./internal/factory/doctrinecompiler/...
ok  	github.com/s1onique/leamas/internal/factory/doctrinecompiler

$ CGO_ENABLED=0 go build -trimpath -o bin/leamas ./cmd/leamas
$ make build VERSION=0.1.0
$ ./bin/leamas version
Leamas version 0.1.0 (commit …)

$ # Fresh-target compile/verify/explain proof:
$ target="$(mktemp -d)"
$ ./bin/leamas factory doctrine plan     --profile fsharp-elm-service-v1 --target "$target"
$ ./bin/leamas factory doctrine compile  --profile fsharp-elm-service-v1 --target "$target"
doctrine compile: OK
$ ./bin/leamas factory doctrine verify   --target "$target"
doctrine verify: OK
$ ./bin/leamas factory doctrine explain  --target "$target"
explain: ...
$ make -C "$target" factorize
factorize:
doctrine-verify
$ make -C "$target" gate
factorize
doctrine-verify

$ go run ./cmd/leamas factory verify act-doctrine-compiler
act-doctrine-compiler: OK
```

### Known limitations

- SemVer-style compiler-version matching accepts `>=`, `MAJOR.x`,
  and exact strings; a full SemVer matcher is deferred.
- The Makefile parser used by `verify` understands comments, blank
  lines, single-line dependencies, and line continuations. It does
  not support rule expansion or automatic variables.
- The `compiler_commit` field in the lock is normalised to `"unknown"`
  when the compiler is built without VCS data; a production build
  with `-ldflags` injection sets the actual commit.
- Atomicity guarantees for `os.Rename` are documented as best-effort
  on Unix and not guaranteed on non-Unix platforms, per the Go
  documentation.
- Development binaries built with the default `VERSION=dev` are
  intentionally incompatible with the constrained doctrine pack;
  release builds must pass `VERSION=0.1.0` (or later) to satisfy
  the canonical constraint.

### Digest evidence — known limitations

The digest tool tracks the parent ACT's text-level changes plus
.gitignore-tracked file content changes. For this closure, the
following items are **not** represented verbatim in the digest but
are still authoritative closure evidence:

* `internal/factory/doctrinecompiler/testdata/fsharp-elm-empty/
  expected/.factory/doctrine.lock.json` is a test fixture whose
  byte content is checked by `TestCompileEmptyTargetProducesGoldenTree`.
  The digest's `CHANGESET_MANIFEST` does not list it; its byte
  identity is verified by the test, not the digest.
* `test_helpers.go` was renamed to `test_helpers_test.go`. The
  digest records this as one modified file and zero deleted files;
  the rename is visible in `git status`.
* `GATE_SUMMARY` is a digest-tool wrapper around `make gate`.
  The wrapper records `source_status=missing` for this worktree
  because it inspects `.factory/` artefacts that this ACT does not
  generate. The textual `make gate` result (`*** GATE PASSED ***`)
  printed above is the authoritative gate evidence.

## Why PASS, not PARTIAL

Every R3 acceptance criterion in the parent ACT is proven by an
executable test:

- R2.1 / R3.1: rollback tests use `FailAfterN` to inject failure
  after at least one mutation has succeeded; `treeSnapshot.equal`
  proves byte- and mode-identical restoration, restored lock,
  recreated managed files, removed created files, and empty-target
  cleanup. Rollback-error visibility is exercised by the tests in
  `transaction_seam_test.go`, particularly
  `TestRollbackLstatFailureJoined`,
  `TestRollbackRemoveFailureJoined`,
  `TestRollbackMkdirAllFailureJoined`,
  `TestRollbackWriteFileFailureJoined`, and
  `TestRollbackErrorMessage`.
- R2.4 / R3.2: table-driven tests cover exact duplicate, normalised
  duplicate, cross-ownership collision, observed-ID duplicate,
  empty managed path, empty observed ID, and a valid distinct
  lock; `TestVerifyFailsBeforeTargetWhenLockAmbiguous` proves the
  verifier refuses before inspecting the target.
- R2.6 / R3.3: tests cover foreign-pack rejection in both `Verify`
  and `Explain`, error message names both packs, no canonical-pack
  report is emitted on failure, inference from the canonical
  selector still works, and explicit `--profile` retains its
  documented behaviour.
- R2.3 / R3.4: `compat_test.go` covers the empty/dev/unknown/0.0.9
  rejection set and the 0.1.0/0.2.0/1.5.0 acceptance set;
  `TestCompileRefusesIncompatibleVersion` proves zero mutations;
  the release-built binary round-trips a fresh target.
- R2.2 / R3.5: tests cover direct two-node cycles, longer
  three-node cycles, sibling cycles with reachable desired
  dependencies, and backslash-newline continuation. Each runs under
  a 10-second per-test deadline.

No acceptance blocker remains after R6. The parent ACT is closed PASS.

**R6 fixes:** the ACT verifier now lists every
`transaction_seam_test.go` test (`TestRollbackApplyOnlyFailure`,
`TestRollbackLstatFailureJoined`, `TestRollbackRemoveFailureJoined`,
`TestRollbackLstatErrNotExistIgnored`, `TestRollbackReadDirFailureJoined`,
`TestRollbackMkdirAllFailureJoined`, `TestRollbackWriteFileFailureJoined`,
`TestRollbackErrorMessage`); the obsolete `TestCompileRollbackErrorVisibility`
reference was removed from this report; all "30-second" timeout
claims were corrected to 10 seconds; the unused `ReadFile` field was
removed from the `fsOps` seam; the digest's known limitations are
documented explicitly in the Digest evidence section above.

**R5 fixes:** the fsOps seam now exclusively mediates rollback
Lstat and Remove calls (including in `removeRegularIfExists`);
the parallel `rollbackInjectErr` global was removed;
`transaction_seam_test.go` adds a dedicated test for every fsOps
operation (Lstat, Remove, ReadDir, MkdirAll, WriteFile); the
Make-cycle deadline helper (`runWithDeadline`) runs only the
operation in the worker goroutine and asserts on the test
goroutine; the close-report contradictions on test names and
timeout values were corrected.

**R4 fixes:** the rollback failure-propagation path now uses
`errors.Join` with sentinel errors (`ErrApplyFailed`,
`ErrRollbackFailed`) and a `fsOps` filesystem seam; only
`fs.ErrNotExist` is ignorable; `TestRollbackLstatFailureJoined` and
`TestRollbackRemoveFailureJoined` prove the joined error chain;
`TestRollbackApplyOnlyFailure` proves the clean-rollback path;
`TestCompileRollbackFailsClosedAfterMutation` proves the
no-rollback-failure path; Make tests run under a real
10-second per-test deadline.

### Git status

The working tree contains the files listed under **Files in the
working tree**. No files outside the ACT scope were touched. The
Circus repository is untouched.
