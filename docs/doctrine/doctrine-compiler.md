# Factory Doctrine Compiler

This document describes the Factory doctrine compiler that lives in
Leamas and projects a canonical doctrine pack plus a target profile
into a target repository.

## Why Leamas owns canonical Factory doctrines

Factory doctrines are versioned, shared artefacts. Target repositories
select them but do not redefine them. This separation has two benefits:

1. **Drift is detected, not negotiated.** A target repository cannot
   silently redefine what "doctrine projection" means; the compiler
   detects divergence via the committed lock file.
2. **Upgrades are atomic.** A doctrine pack change is delivered as a
   single Leamas release. Target repositories re-compile to pick up
   the new canonical projection.

The compiler is therefore a deterministic projector: given a pack and
a profile, it produces a bounded tree inside the target repository.
No target-defined arbitrary shell commands are accepted; the only
target inputs are the project selector and a small set of bounded
settings.

## Terminology

| Term             | Definition                                                                  |
| ---------------- | --------------------------------------------------------------------------- |
| **Pack**         | A versioned, declarative JSON document owned by Leamas.                     |
| **Profile**      | A bundle of outputs, seeds, observed contracts, and checks declared in a pack. |
| **Selector**     | `.factory/project.json`; target-owned; identifies the pack and profile.   |
| **Projection**   | The bounded tree of files derived from a pack, profile, and target state. |
| **Lock**         | `.factory/doctrine.lock.json`; compiler-owned; records the projected state. |
| **Managed**      | Compiler-owned file. Never edited by hand.                                  |
| **Seeded**       | Target-owned file after first creation. Compiler may create it once.       |
| **Observed**     | Runtime invariant asserted by the verifier against existing target files.   |

## Ownership

Ownership is a closed type with three values:

- `Managed` — written by the compiler, verified by digest.
- `Seeded` — written by the compiler only when absent; subsequent
  compile runs preserve whatever the target holds.
- `Observed` — declared in the pack, asserted by the verifier; the
  compiler never writes observed contracts.

The compiler rejects target-side attempts to redefine ownership: the
selector cannot grant command injection, target-defined shell
commands, or arbitrary filesystem writes. All writes are performed by
ordinary Go code paths inside the compiler; no general-purpose
expression or templating language is involved.

## Commands

The compiler exposes four public commands:

```text
leamas factory doctrine plan     --profile <id> [--target <path>]
leamas factory doctrine compile  --profile <id> [--target <path>]
leamas factory doctrine verify   [--profile <id>] [--target <path>]
leamas factory doctrine explain  [--profile <id>] [--target <path>]
```

`--target` defaults to the current working directory.

For `verify` and `explain`, `--profile` is optional: when omitted,
the compiler reads the committed `.factory/project.json` selector and
uses its declared `pack` and `profile`. The `plan` and `compile`
subcommands require an explicit `--profile` so that a fresh target
without a selector can still be bootstrapped.

### `plan`

Performs no writes. Loads the pack, validates it, inspects the target,
and prints the projected action set classified as one of:

| Class                       | Meaning                                                |
| --------------------------- | ------------------------------------------------------ |
| `create-managed`            | Path is missing; the compiler will create a managed file. |
| `create-seeded`             | Path is missing; the compiler will create a seeded file. |
| `update-managed`            | Managed file exists but its digest has drifted.          |
| `unchanged`                 | Managed file exists and matches the canonical digest.    |
| `preserve-seeded`           | Seeded file exists; the compiler will not touch it.      |
| `remove-obsolete-managed`   | Path is recorded as managed but no longer in the pack.   |
| `reject`                    | The target shape is unsafe; the compiler refuses to plan. |

`plan` rejects an incompatible or placeholder compiler version
before producing an actionable plan. The CLI checks compatibility
via `CheckCompilerCompatibility` before invoking the planner, and
the library entry points `PlanWithOptions` consult the same check
when supplied with a non-empty compiler version.

`plan` returns a non-zero exit code when the plan contains reject
actions.

### `compile`

Performs the same planning as `plan` and then applies the
non-rejecting actions:

- Creates managed and seeded files whose action is `create-*`.
- Updates managed files whose action is `update-managed`.
- Removes files whose action is `remove-obsolete-managed`.
- Preserves seeded files.
- Writes the lock file **last**, only after every other write
  succeeds.

`compile` performs the following transactional discipline:

1. Finish planning and all validation before any write.
2. Identify every path that may be created, updated, removed, or
   replaced; snapshot its pre-state (existence, mode, bytes).
3. Identify every parent directory that does not yet exist on disk;
   these are tracked so rollback can clean them up.
4. Apply mutations in deterministic order, journaling each success.
5. On any failure, walk the journal in reverse and restore the
   pre-state:
   - file contents (byte-identical),
   - permission bits (from `os.FileMode.Perm()`),
   - the previous lock file if it existed,
   - recreated files that the apply removed,
   - newly created files are removed,
   - transaction-created directories are removed deepest-first, only
     when empty,
   - pre-existing directories are never removed or altered,
   - the target root is never removed,
   - temporary files produced by interrupted writes are removed.

Rollback errors are surfaced via `errors.Join`; the returned
`CompileError` always reports whether the target was safely restored.
Production binaries have no fault injection; tests inject failures
**after** a successful mutation via `CompilerOptions.FailAfterN`,
which guarantees rollback has real work to do.

The second `compile` against an unchanged target is byte-for-byte
idempotent.

### `verify`

Performs no writes. Recomputes the expected projection, compares it
against the committed lock and the target files, and asserts each
declared observed contract.

The verifier enforces three-way consistency:

```
canonical desired digest == lock digest == actual file digest
```

It also detects:

- Missing or corrupted selector.
- Selector naming an unsupported pack (the available pack is named
  in the finding).
- Pack / profile identity mismatch.
- Pack digest mismatch.
- Managed file missing or modified.
- Unexpected managed files (recorded as managed but absent from
  the desired projection).
- Lock entries whose paths escape the target root.
- Lock exactness: duplicate normalized paths in `managed_files` or
  `seeded_files`, cross-ownership collisions, and duplicate
  observed-contract IDs are all rejected by `ReadLockFile` before
  verification proceeds.
- Compiler-version incompatibility against the pack's constraint.
- Observed-contract violations (Makefile include missing, `gate`
  lacks a transitive dependency path to `factorize`, cycles in
  the reachable subgraph).

`verify` never mutates the target. To repair drift, run `compile`.

### `explain`

Reports the selected pack, profile, compiler identity, source
revision when available, the observed managed and seeded files, the
observed contracts, the enabled doctrines, and the named extension
points. Output is text-only and deterministic.

`explain` refuses to emit a report when the committed selector names
a pack other than the loaded pack. The error identifies both the
requested and the available pack.

## Lock exactness

`ReadLockFile` rejects ambiguous locks before any other verifier
work. The lock must satisfy all of:

- No duplicate normalized paths in `managed_files`.
- No duplicate normalized paths in `seeded_files`.
- No normalized path may appear in both `managed_files` and
  `seeded_files` (cross-ownership collision).
- No duplicate IDs in `observed_contracts`.
- No empty managed/seeded paths.
- No empty observed-contract IDs.

Path normalisation runs before duplicate detection, so alternate
spellings (`./a/b`, `a/./b`) cannot bypass it. The first conflict
encountered is reported deterministically with its section, key,
and indices.

## Selector-pack fidelity

Until a pack registry exists, every selector consumed by the
compiler must name the loaded pack. The supported pack is
`factory-core-v1`. Both `verify` and `explain` reject any selector
whose `pack` field does not equal the loaded pack; the CLI surfaces
the same constraint before either command dispatches to the library.

The error message always includes both pack IDs:

```text
selector requests unsupported pack "<requested>"; available pack is "<available>"
```

`compile` continues writing selectors containing the canonical pack
ID. `verify` and `explain` never silently fall through to the
loaded pack's profile when the selector names a foreign one.

## Compiler compatibility

The canonical pack declares `compiler_version: ">=0.1.0"`. The
compiler enforces this constraint in three places:

- `plan` checks the supplied compiler version against the constraint
  before producing an actionable plan.
- `compile` checks before mutating the target.
- `verify` checks the runtime compiler version before accepting the
  lock.

For the canonical constraint:

| Compiler version | Verdict                                  |
| ---------------- | ---------------------------------------- |
| empty            | rejected                                 |
| `dev`            | rejected                                 |
| `unknown`        | rejected                                 |
| `0.0.9`          | rejected (below floor)                   |
| `0.1.0`          | accepted (at floor)                      |
| `0.2.0`          | accepted (above floor)                   |
| `1.5.0`          | accepted                                 |

This is the existing bounded compatibility language; full SemVer
matching remains deferred.

### Building a compatible binary

A development build that uses the default `VERSION=dev` is
**incompatible** with the constrained doctrine pack by design. To
build a binary that satisfies the constraint:

```bash
make build VERSION=0.1.0
./bin/leamas version
```

The version is injected via the standard `-ldflags` mechanism in
`internal/version`. Tests that exercise doctrine commands in a
subprocess inject the same concrete version. The compiler's
`init()` wires `version.Get().Version` into the compatibility
check so production binaries always run with the correct identity.

## Why `factorize` and `gate` never compile

The generated `.factory/generated/factory.mk` defines a `factorize`
target that runs read-only verification. It is intentionally narrow:

- It never invokes `leamas factory doctrine compile`.
- It never rewrites the doctrine lock.
- It never formats or modifies source files.
- It propagates non-zero exit codes.
- It preserves failure output for human review.

The seed Makefile declares `gate` as a target that depends on
`factorize`. The dependency chain is documented and verifier-checked;
the verifier rejects Makefiles that lose the include, that drop the
`factorize` dependency, or that introduce a recursive cycle.

## How a new repository consumes the pack

```bash
cd path/to/new-repo
leamas factory doctrine plan     --profile fsharp-elm-service-v1 --target .
leamas factory doctrine compile  --profile fsharp-elm-service-v1 --target .
leamas factory doctrine verify                            --target .
leamas factory doctrine explain                           --target .
make factorize
make gate
```

The compile step writes the bounded projection into the target. The
target then owns:

- The committed `.factory/doctrine.lock.json` (must not be edited).
- The committed `.factory/project.json` (may be edited to change
  pack/profile selection in a future ACT).
- The committed `Makefile` (target-owned after creation; can be
  extended freely).

The target does **not** own:

- The canonical pack (defined inside Leamas).
- Any `.factory/generated/` file (compiler-managed).
- Any future lock file with a different pack digest.

## How a repository extends its native gate

The target-owned `Makefile` may add additional dependencies to
`gate`:

```makefile
include .factory/generated/factory.mk

.PHONY: gate my-checks
gate: factorize my-checks
my-checks:
	@echo "running extra checks"
```

The verifier checks the structural invariants:

- The generated fragment remains included.
- `gate` retains a dependency path to `factorize`.
- No recursive cycle is introduced.

Anything that violates these invariants is rejected by `verify`.

## Pack versioning

Packs are versioned by:

- `schema_version` — wire-format dialect. The compiler refuses
  schemas it does not understand.
- `pack_version` — semantic version of the doctrine inventory.
- `pack_digest` — SHA-256 digest of the canonical pack bytes. Used
  for drift detection.
- `compiler_version` — bounded compatibility constraint; the
  canonical constraint is `>=0.1.0`.

When a pack version changes, target repositories must run `compile`
again to update the lock. Until they do so, `verify` reports a
`pack_digest_mismatch` and the projection is treated as stale.

## Recovery from projection drift

`verify` reports drift but does not repair it. To recover:

1. Inspect the verify output to identify which finding applies.
2. For managed-file drift, run `compile` to repair.
3. For lock corruption, run `compile` to regenerate.
4. For unexpected managed files, run `compile`; obsolete files are
   removed automatically.
5. For observed-contract violations, fix the target file manually
   and re-run `verify`.

`compile` is transactional with respect to its own actions. A
mid-apply failure rolls back files the apply created or modified in
this run, restores the prior lock, removes transaction-created
directories that are empty, and reports the original failure with
the rollback outcome (successful or with residual errors).

## Known limitations

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

## Example: Circus bootstrap

The following sequence bootstraps an empty future Circus repository.
This is an example only; The Circus is not modified by this ACT.

```bash
cd path/to/new-circus
leamas factory doctrine plan     --profile fsharp-elm-service-v1 --target .
leamas factory doctrine compile  --profile fsharp-elm-service-v1 --target .
leamas factory doctrine verify                            --target .
make factorize
make gate
```

The projection is bounded: six files, no application scaffolding, no
network access, no remote doctrine distribution. The Circus
repository selects its doctrine and Leamas projects it.
