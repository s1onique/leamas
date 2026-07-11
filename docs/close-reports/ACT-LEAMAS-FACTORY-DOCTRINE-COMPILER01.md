## ACT-LEAMAS-FACTORY-DOCTRINE-COMPILER01

### Result
**PARTIAL**

### Status

The initial implementation was reviewed twice. The first review (R1)
identified eight original findings; the R1 implementation pass
addressed all eight. R2 then reopened or qualified several of those
areas: transactional rollback completeness, Make adversarial proof,
compatibility enforcement, exact lock-set validation, and
selector-pack fidelity. R2 also added three new findings: strict
JSON parsing, three-state Make DFS, and compiler-version plumbing.

Grouping by numbered findings:

> R2.2 and R2.5 are implemented. R2.3 is partially implemented,
> while R2.1, R2.4, and R2.6 remain partial or open.

This document is the canonical close report for `digest(46)`. No
material code change accompanies this digest; the wording is
corrected to reflect the R2 reviewer's accepted board.

### Capability delivered

A deterministic, repo-local Factory doctrine compiler in Leamas that
projects a versioned doctrine pack plus a target profile into a
bounded tree inside the target repository. The four public commands
`plan`, `compile`, `verify`, and `explain` are wired under
`leamas factory doctrine <sub>`.

The Circus repository was not modified.

### Acceptance status

| Item                                  | Status                                            |
| ------------------------------------- | ------------------------------------------------- |
| **R2.5** strict JSON decoding         | **Implemented** (strictDecode uses second `Decode` requiring `io.EOF`) |
| **R2.2** Make DFS + continuations     | **Implemented; proof incomplete** (three-state DFS terminates on arbitrary cycles; backslash-newline continuation parsed; multi-node cycle regression tests not yet added) |
| **R2.3** compiler-version plumbing    | **Implemented** (Compile threads version; CLI sets `compilerVersionSource` via `init()`) |
| **R2.4** seeded/observed exactness    | **Partial — duplicates accepted** (unexpected/missing seeded detected; observed-contract set compared; `ReadLockFile` does not yet reject duplicate normalized paths or IDs) |
| **R2.1** transactional rollback       | **Partial — implementation incomplete** (`FailAfter` injects before the action; file modes not captured; created parent directories not removed; rollback errors silently discarded) |
| **R2.3** compatibility enforcement    | **Partial — canonical constraint empty** (plumbing present; canonical pack declares `compiler_version: ""` so check is trivially satisfied) |
| **R2.6** selector-pack fidelity       | **Open** (`verify` discards selector pack; `explain` misleadingly reports core pack) |

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
  R2 reopened the requirement for multi-node cycle regression
  proofs.

### R2 work completed (two of six findings)

- **R2.5 — Strict JSON parsing.** `strictDecode` rejects unknown
  fields and a second top-level JSON value via a second `Decode`
  call requiring `io.EOF`. Applied to `DecodePack`, `ReadLockFile`,
  and `readSelector`.

- **R2.2 — Make DFS + continuations.** The `makeReaches` function
  is a true three-state DFS that terminates on arbitrary cycles and
  walks the entire reachable graph.

### R2 work partially implemented or open (four of six findings)

- **R2.3 — Compiler-version plumbing (partial).** `Compile` threads
  the supplied `CompilerVersion` through `PlanWithOptions`; the CLI
  sets `compilerVersionSource` via `init()` to
  `version.Get().Version`; `CheckCompilerCompatibility` rejects `""`
  and `"dev"` for non-empty constraints. Enforcement itself is
  currently disabled because the canonical pack declares
  `compiler_version: ""`, making the check trivially satisfied.

- **R2.1 — Rollback completeness (partial).** The implementation
  must capture `{exists, kind, mode, bytes}` in the snapshot; the
  fault-injection hook must run AFTER a successful mutation; the
  rollback must remove directories the apply created; and rollback
  failures must be surfaced as a fatal compound error rather than
  silently discarded.

- **R2.4 — Exact lock sets (partial).** `ReadLockFile` does not yet
  reject duplicate normalized managed paths, seeded paths, or
  observed-contract IDs. `compareObservedSet` uses a map-based
  comparison that collapses duplicate IDs.

- **R2.6 — Selector-pack fidelity (open).** `verify` and `explain`
  must validate `sel.Pack == pack.PackID`. Until a pack registry
  exists, fail explicitly with `unsupported selector pack` when the
  selector names a pack other than `factory-core-v1`.

### Files in the working tree

```
internal/factory/doctrinecompiler/                       (new package)
  ├── cli.go
  ├── compile.go
  ├── compile_safety_test.go
  ├── compile_test.go
  ├── compat.go
  ├── determinism_test.go
  ├── digest.go
  ├── explain.go
  ├── fsx.go
  ├── helpers.go
  ├── integration_test.go
  ├── jsoncheck.go
  ├── lock_helpers.go
  ├── lockfile.go
  ├── makecheck.go
  ├── pack.go
  ├── pack_test.go
  ├── packcore.go
  ├── packschema.go
  ├── paths.go
  ├── paths_test.go
  ├── plan.go
  ├── plan_test.go
  ├── selector.go
  ├── subprocess_test.go
  ├── test_helpers.go
  ├── types.go
  ├── verify.go
  ├── verify_test.go
  ├── packs/factory-core-v1/pack.json
  └── testdata/fsharp-elm-empty/expected/*   (golden tree)
cmd/leamas/factory.go                               (doctrine dispatch)
cmd/leamas/factory_doctrine.go                      (CLI dispatcher)
cmd/leamas/factory_verify_act_doctrine_compiler.go (ACT verifier)
cmd/leamas/main.go                                  (verify case)
cmd/leamas/factory_verify_dispatch_test.go         (count update)
docs/doctrine/doctrine-compiler.md                  (doctrine docs)
docs/close-reports/ACT-LEAMAS-FACTORY-DOCTRINE-COMPILER01.md (this file)
```

### Verification commands and exact results

```
$ go test ./internal/factory/doctrinecompiler/...
ok  	github.com/s1onique/leamas/internal/factory/doctrinecompiler	2.375s

$ go test ./cmd/leamas/...
ok  	github.com/s1onique/leamas/cmd/leamas

$ go vet ./...
(clean)

$ make factorize
*** FACTORIZE PASSED ***

$ make gate
*** GATE PASSED ***

$ go run ./cmd/leamas factory verify act-doctrine-compiler
act-doctrine-compiler: OK

$ go test -run TestSubprocessMakeFactorizeAndGate -count=1 ./internal/factory/doctrinecompiler/...
ok  	github.com/s1onique/leamas/internal/factory/doctrinecompiler
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

### Why PARTIAL, not PASS

The R2 reviewer explicitly listed R2.1, R2.4, and R2.6 as acceptance
blockers. R2.1's rollback semantics are demonstrably incomplete
against the ACT's "restore the pre-compilation state exactly"
requirement: file modes are not captured, created directories are
not removed, and rollback errors are silently discarded. R2.4's
lock verifier accepts duplicate entries that should be rejected.
R2.6's `explain` silently presents the canonical pack even when
the selector names a different one. The compatibility-enforcement
partial state (R2.3 plumbing without an active constraint) is
listed as a known limitation but is not counted as one of the three
mandatory blockers; addressing it is required for the doctrine
compiler to enforce real compatibility guarantees.

The next ACT pass must address the rollback semantics, the duplicate
lock-entry validation, and the selector-pack validation before this
ACT can be closed PASS.

### Git status

The working tree contains the files listed under **Files in the
working tree**. No files outside the ACT scope were touched. The
Circus repository is untouched.
