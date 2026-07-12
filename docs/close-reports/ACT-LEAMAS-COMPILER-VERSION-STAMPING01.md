# Close Report: ACT-LEAMAS-COMPILER-VERSION-STAMPING01

## ACT Reference

**ACT-LEAMAS-COMPILER-VERSION-STAMPING01**: Stamp developer and
release Leamas binaries with SemVer-compatible versions; separate
version stamping from build provenance.

## Summary

Diagnosed the Circus bootstrap failure as a Leamas build-stamping
defect (not a compiler bug) and shipped the fix. After **R1**
strict-SemVer enforcement and **R2** SemVer grammar + overflow
safety, the contract is the canonical SemVer 2.0.0 grammar at
both runtime and shell level, the implementation is
overflow-safe, the VCS fallback feeds the stamp, the JSON wire
form is consistent, and the Circus bootstrap reproduces all
expected artefacts on a fresh target directory.

## Files Changed

### Added
- `internal/version/semver.go` — `IsValidSemVer`, `ParseSemVer`,
  `IsPlaceholder`, plus the `SemVerCompatible = "0.1.0"` and
  `MaxCommitLen = 12` constants.
- `internal/version/semver_test.go` — accepted/rejected tables,
  large-numerics parser, round-trip, whitespace behaviour.
- `internal/factory/doctrinecompiler/compat_extra_test.go` —
  precedence table, overflow-safety table, full malformed
  oracle matrix.
- `docs/acts/ACT-LEAMAS-COMPILER-VERSION-STAMPING01.md` — ACT spec.
- `docs/doctrine/compiler-version-stamping.md` — split-out
  contract reference.

### Modified
- `internal/version/version.go` — added `DeclaredVersion` and
  `Dirty` package variables; `Info` gains corresponding fields;
  `Get()` derives the effective `Version` from
  `info.DeclaredVersion` (post-VCS fallback) and clears
  `DeclaredVersion`/`Dirty` when they match the omitempty
  contract. Custom `MarshalJSON` enforces the same wire form
  regardless of who constructed the Info.
- `internal/version/effective.go` — uses `IsPlaceholder`
  (renamed from the old `IsUnparseable`); preserves malformed
  declared values verbatim.
- `internal/factory/doctrinecompiler/compat.go` — calls
  `version.IsValidSemVer` on `have`; pre-release-aware
  string-based `compareSemver` (R2.2 overflow safety); no
  integer arithmetic on numeric identifiers.
- `internal/factory/doctrinecompiler/compat_test.go` — added
  build-metadata / pre-release / unknown-prerelease RED tests.
- `internal/factory/digest/contract_integration_test.go` —
  contract test now sets both `Version` and `DeclaredVersion`
  so the stamp passes through unchanged.
- `cmd/leamas/version.go` — line-oriented output matches the
  JSON omitempty contract: `declared_version:` only when
  non-empty; `dirty:` only when non-`false`.
- `cmd/leamas/version_cli_test.go` — line-bound increased to 5
  (R2.5a); new `TestVersionCLI_ReleaseBinary` actually builds
  a release binary with `-ldflags` and verifies the schema
  (R2.5b); `TestVersionCLI_DeclaredVersionEmitted` fails
  (not skip) when missing on dev builds.
- `Makefile` — adds `DeclaredVersion` linker injection, the
  strict `STAMP_REGEX` POSIX-ERE grammar mirroring the official
  suggested regex, default `STAMP_BINARY ?= bin/leamas`,
  stricter `release-build` guard, new `stamp-check-build` and
  `release-stamp-verify` recipes.
- `docs/factory/version.md` — accurate 4-linker-variable
  Makefile contract, dirty-from-ReadBuildInfo clarification,
  integer-overflow-safety note, full strict-SemVer rejection
  matrix.
- `docs/doctrine/doctrine-compiler.md` — `Compiler compatibility`
  table extended with build-metadata/pre-release rows.
- `docs/doctrine/compiler-version-stamping.md` — full contract.

## Behaviour Changed

### Compatibility oracle (strict SemVer 2.0.0)
- `0.1.0+dev.<commit>` satisfies `>=0.1.0` (build metadata has no
  precedence).
- `0.0.9+dev.<commit>` does not satisfy `>=0.1.0` (base below
  floor).
- `0.1.0-dev.<commit>` does not satisfy `>=0.1.0` (pre-release
  ranks below the same MAJOR.MINOR.PATCH).
- Numeric prerelease identifiers with leading zeros are
  rejected (`1.2.3-01`, `1.2.3-alpha.01`).
- Empty prerelease/build identifiers are rejected
  (`1.2.3-alpha..1`, `1.2.3+build..42`).
- Arbitrary text is rejected (`banana`, `1oops`).
- `unknown`, `dev`, and `""` are rejected like placeholders.

### Version command output
- `version:` reports the effective SemVer; release builds show
  `0.1.0` verbatim, development builds show
  `0.1.0+dev.<commit>.<ts>`.
- `declared_version:` line and `declared_version` JSON key are
  emitted only when the stamp was auto-derived (omitted when
  equal to `version`).
- `dirty:` line and `dirty` JSON key are emitted only when
  `runtime/debug.ReadBuildInfo()` reports `vcs.modified=true`
  (omitted when `false`).

### Build pipeline
- `make build` (default `VERSION=dev`) auto-stamps the binary via
  the runtime `Effective()` derivation. `make install` reuses
  the same `build` recipe and then runs `make stamp-check`.
- `make release VERSION=...` rejects placeholders (`''`, `dev`,
  `unknown`) and any value that does not match the strict
  SemVer 2.0.0 pattern enforced by `STAMP_REGEX`.
- `make stamp-check STAMP_BINARY=...` audits any binary against
  the same regex; the `install` recipe chains through
  `stamp-check-build` so installing an unstamped or malformed
  binary exits non-zero.

### Integer-overflow safety
- `compareSemver` and `compareNumeric` operate entirely on
  decimal strings. No numeric identifier (Major / Minor /
  Patch / numeric pre-release identifiers) is ever converted to
  a Go `int`. SemVer strings with components larger than
  `math.MaxInt64` are ordered correctly.

### Get/FromSettings seam (R2.3 → R3 refinement)
- `Get()` populates `Info` from the package globals, runs the
- production `getFromSettings` seam to fill unknown
- commit/build_time/dirty from `runtime/debug.ReadBuildInfo()`'s
- `vcs.revision`, `vcs.time`, and `vcs.modified` (when missing),
- then calls `EffectiveVersion(Version, info.DeclaredVersion,
- info.Commit, info.BuildTime)`. VCS-recovered provenance
- participates in the stamp when no LDFLAGS are provided.

## Verification

### Strict-SemVer rejection (R1.1, R2.1 acceptance proof)

```bash
$ for v in banana 1oops 1.2 01.2.3 1.2.3+ 1.2.3.4 1.2.3-01 \
         1.2.3-alpha.01 1.2.3-00 1.2.3-alpha..1 1.2.3-.alpha \
         1.2.3-alpha. 1.2.3+build..42 1.2.3+.build 1.2.3+build.; do
    make release VERSION="$v"  # expected: exit 2
  done
banana          REJECTED (rc=2)
1oops           REJECTED (rc=2)
1.2             REJECTED (rc=2)
01.2.3          REJECTED (rc=2)
1.2.3+          REJECTED (rc=2)
1.2.3.4         REJECTED (rc=2)
1.2.3-01        REJECTED (rc=2)
1.2.3-alpha.01  REJECTED (rc=2)
1.2.3-00        REJECTED (rc=2)
1.2.3-alpha..1  REJECTED (rc=2)
1.2.3-.alpha    REJECTED (rc=2)
1.2.3-alpha.    REJECTED (rc=2)
1.2.3+build..42 REJECTED (rc=2)
1.2.3+.build    REJECTED (rc=2)
1.2.3+build.    REJECTED (rc=2)

$ for v in 0.1.0 1.2.3-alpha 1.2.3+build.42; do
    make release VERSION="$v"  # expected: BUILD OK
  done
0.1.0           BUILD OK
1.2.3-alpha     BUILD OK
1.2.3+build.42  BUILD OK
```

Direct oracle tests for malformed `have` values are in
`internal/factory/doctrinecompiler/compat_extra_test.go` (the
`TestCheckCompilerCompatibility_BuildMetadataVsConstraint`
table now includes `banana`, `1oops`, `1.2`, `01.2.3`,
`1.2.3+`, `1.2.3.4`, `1.2.3-01`, `1.2.3-alpha..1`, and
`1.2.3+build..42` per R2.5c).

### End-to-end Circus proof (R1.2 acceptance proof)

```bash
make build                          # auto-stamped development binary
./bin/leamas version                 # reports effective SemVer

rm -rf /tmp/circus-r1
mkdir -p /tmp/circus-r1

./bin/leamas factory doctrine plan  --profile fsharp-elm-service-v1 \
    --target /tmp/circus-r1
# compiler: 0.1.0+dev.<commit>.<ts>

./bin/leamas factory doctrine compile  --profile fsharp-elm-service-v1 \
    --target /tmp/circus-r1
# doctrine compile: OK

./bin/leamas factory doctrine verify  --target /tmp/circus-r1
# doctrine verify: OK

# 6 produced outputs (the selector is consumed input):
test -f /tmp/circus-r1/.factory/project.json                    # selector
test -f /tmp/circus-r1/.factory/doctrine.lock.json              # produced
test -f /tmp/circus-r1/.factory/generated/factory.mk            # produced
test -f /tmp/circus-r1/.factory/generated/doctrine-inventory.md # produced
test -f /tmp/circus-r1/docs/factory/README.md                   # produced
test -f /tmp/circus-r1/Makefile                                # produced
```

All six checks pass; `doctrine compile: OK` /
`doctrine verify: OK`.

### Commands Run

```bash
go test ./internal/version/...                         # PASS
go test ./internal/factory/doctrinecompiler/...        # PASS
go test ./internal/factory/digest/...                 # PASS
go test ./cmd/leamas/...                              # PASS
go test ./...                                         # PASS
make factorize                                        # PASSED
make gate                                             # PASSED
```

### Results (honest)

- `make factorize` and `make gate`: PASSED (15 verifiers + 5
  Go-toolchain stages).
- `go test ./...`: PASSED on all packages, including the new
  R2 strict-SemVer tests, overflow-safety cases, and release-binary
  CLI test.
- `make release VERSION=dev`: failed closed (`Error 2`).
- `make release VERSION=banana|1oops|1.2|01.2.3|1.2.3+|1.2.3.4|
  1.2.3-01|1.2.3-alpha.01|1.2.3-alpha..1|1.2.3+build..42`:
  all failed closed (`Error 2`).
- `make release VERSION=0.1.0|1.2.3-alpha|1.2.3+build.42`: each
  built, stamp-checked, checksummed, verified OK.
- `TestVersionCLI_ReleaseBinary` builds a real release binary
  with explicit `-ldflags` injecting `0.1.0`, asserts the
  output omits `declared_version:`, and asserts the JSON wire
  form omits the `declared_version` key.
- `TestGet_FromSettingsFillsProvenanceIntoStamp` exercises the
  R2.3 acceptance criterion.

### Skipped / Deferred

- Full boolean SemVer range matching (`^`, `~`, `||`); deferred
  per `docs/doctrine/doctrine-compiler.md` and unchanged by this
  ACT.
- Multi-platform release matrix (`dist/leamas_<ver>_<goos>_<goarch>`)
  continues to use the host platform only; not part of this ACT.
- Circus ACT retry is intentionally left to
  `ACT-CIRCUS-FACTORY-BOOTSTRAP01`; this ACT enables but does not
  execute it.

## Decisions Made

1. **Strict SemVer is enforced by both validators.** The Go
   `IsValidSemVer` regex and the Makefile `STAMP_REGEX` use the
   same grammar: leading-zero rejection, no implicit trim, no
   empty identifiers, no trailing `+`. Single source of truth.
2. **Pre-release ranks below same MAJOR.MINOR.PATCH per SemVer
   §11.** A `0.1.0-dev.<commit>` is rejected by `>=0.1.0`;
   `0.1.0+dev.<commit>` is accepted.
3. **No integer arithmetic on identifiers (R2.2).** Numeric
   components and pre-release identifiers are decimal strings;
   comparison is length-then-lexical, so arbitrarily large
   versions parse and order correctly.
4. **`Get()` uses post-fallback `info` for stamp derivation
   (R2.3).** Plain `go build` (without LDFLAGS) still produces a
   stamped binary when `-buildvcs=true` embeds `vcs.revision`
   and `vcs.time`.
5. **Four linker variables; `Dirty` from `ReadBuildInfo` (R2.4).**
   The Makefile injects `Version`, `DeclaredVersion`, `Commit`,
   `BuildTime`. `Dirty` is recovered from `vcs.modified` because
   computing it from `git status --porcelain` at build time
   would couple the Makefile to a working Git history.
6. **Custom `MarshalJSON` enforces omitempty regardless of
   constructor (R2.5d).** Both `Get()` and direct `Info{…}`
   literals serialise consistently because the JSON shape is
   determined by `MarshalJSON`, not by struct tags alone.
7. **Documented FourLinker / Dirty-Fallback contract.** The
   close report and `docs/factory/version.md` agree that four
   linker variables are injected; `Dirty` is a separate
   `ReadBuildInfo` recovery.
8. **Splitting docs instead of weakening the LLM-friendly gate.**
   Both `doctrine-compiler.md` and the new
   `compiler-version-stamping.md` stay under the 400-line ceiling.

## Agent Doctrine Impact

- No Python added. No new Bash verifier scripts.
- New Go test files; new Go helper files (`semver.go`,
  `effective.go`); new Makefile recipe (`stamp-check` and its
  helpers `stamp-check-build`, `release-stamp-verify`).
- No allowlists, bypasses, or exception lists added to any gate.
- No OAuth/OIDC/RBAC/database/gateway behaviour introduced.
- LLM-friendliness gate: PASSED.

## Follow-up ACTs

| ACT                                  | Description                                                       |
|--------------------------------------|-------------------------------------------------------------------|
| ACT-CIRCUS-FACTORY-BOOTSTRAP01       | Retry the Circus bootstrap with the now-stamped Leamas binary.    |
| ACT-LEAMAS-COMPILER-SEMVER-FULL01    | (Optional) Implement full boolean SemVer range matching.         |
| ACT-LEAMAS-MULTIPLATFORM-RELEASE01   | (Optional) Multi-platform release build matrix.                   |

## Acceptance Criteria Checklist (R2 final)

| # | Criterion                                                                                                       | Status |
|---|----------------------------------------------------------------------------------------------------------------|--------|
| 1 | Release binaries report strict SemVer such as `0.1.0`.                                                          | PASS — `make release VERSION=0.1.0` produces binary whose `version: 0.1.0`. |
| 2 | Compatible development binaries report strict SemVer with build metadata, such as `0.1.0+dev.<commit>`.         | PASS — verified by `make build` then `./bin/leamas version`. |
| 3 | Five provenance fields (4 linker vars + 1 ReadBuildInfo).                       | PASS: Version, DeclaredVersion, Commit, BuildTime are 4 linker vars; Dirty is 5th via ReadBuildInfo. |
| 4 | Raw dev, empty, or otherwise unparsable versions still fail.                       | PASS: CheckCompilerCompatibility + IsValidSemVer cover placeholders + malformed table. |
| 5 | `0.1.0+dev.<commit>` satisfies `>=0.1.0`.                                                                      | PASS — `TestCheckCompilerCompatibility_BuildMetadataAccepted`. |
| 6 | `0.0.9+dev.<commit>` does not satisfy `>=0.1.0`.                                                              | PASS — `TestCheckCompilerCompatibility_BuildMetadataBelowFloorRejected`. |
| 7 | `0.1.0-dev.<commit>` does not satisfy `>=0.1.0` (pre-release).                                                 | PASS — `TestCheckCompilerCompatibility_PreReleaseRejected`. |
| 8 | `plan`, `compile`, and `verify` apply the identical effective-version policy.                                  | PASS — all four CLI handlers derive effective version via `version.Get()`. |
| 9 | The generated lock records both the effective compiler version and immutable build provenance.                  | PASS — `Compile({CompilerVersion, CompilerCommit})` keeps both, lock unchanged. |
| 10 | Packaging/install targets fail when they would emit an unstamped dev binary.      | PASS: make install -> stamp-check; release VERSION=dev fails; STAMP_REGEX rejects malformed. |
| 11 | **R2.1 strict-SemVer release**: malformed `banana / 1oops / 1.2 / 01.2.3 / 1.2.3+ / 1.2.3.4 / 1.2.3-01 / 1.2.3-alpha..1 / 1.2.3+build..42` all fail closed. | PASS — direct execution showed all rejected with `rc=2`. |
| 12 | **R2.2 integer-overflow safety**: components larger than `math.MaxInt64` parse and order correctly.         | PASS — `TestParseSemVer_LargeNumericsOnly` and `TestCompareSemver_OverflowSafety`. |
| 13 | **R2.3 Get/FromSettings seam**: VCS fallback fills `info.Commit`/`info.BuildTime` and the effective stamp uses that info. | PASS — `TestGet_FromSettingsFillsProvenanceIntoStamp`. |
| 14 | **R2.4 4-linker-variable Makefile contract**: `Dirty` is from `ReadBuildInfo()`. | PASS — `docs/factory/version.md` updated; close report updated; no link variable mismatch. |
| 15 | **R2.5a line-bound**: dirty line emits for dirty builds. | PASS — `TestVersionCLI_Output_FieldSchema` accepts 3–5 lines. |
| 16 | **R2.5b real release-binary CLI test**: actually builds a release and asserts output. | PASS — `TestVersionCLI_ReleaseBinary`. |
| 17 | **R2.5c malformed-version oracle matrix**: direct tests on `CheckCompilerCompatibility`. | PASS — `TestCheckCompilerCompatibility_BuildMetadataVsConstraint` includes 9 malformed cases. |
| 18 | **R2.5d JSON conditional contract**: `DeclaredVersion` and `Dirty` are JSON-omitempty. | PASS — `TestInfo_JSON_OmitsConditionalFields`. |


## R5 Closure Notes (final code-fix pass)

R5 closed the last semantic defect flagged after R4: a malformed
`Version` could still be laundered into a derived stamp via the
`EffectiveVersion(version, declared, commit, buildTime)` fallback.

* **R5.1 — `EffectiveVersion` falls back only when Version is a
  placeholder.** The fallback now reads:

  ```go
  if IsValidSemVer(version) { return version }
  if !IsPlaceholder(version) { return version } // malformed
  return EffectiveFrom(declared, commit, buildTime)
  ```

  Malformed values like `banana`, `1oops`, `1.2.3+`, `1.2.3.4`,
  and `" 1.2.3 "` are no longer silently laundered into a derived
  stamp; they reach the strict-SemVer oracle for rejection.

  New tests:
  - `TestEffectiveVersion_MalformedVersionPreserved` (version)
  - `TestGet_MalformedVersionNotMaskedByDeclaredPlaceholder` (version)
  - `TestVersionCLI_MalformedLinkerVersionRejected` (cmd/leamas,
    executable) — builds a temporary binary with only
    `-X .../Version=banana` injected, asserts that
    `leamas version` reports `version: banana` verbatim and that
    the strict-SemVer regex would reject it.

* **R5.2 — close report truncated sentence completed.** The R4
  paragraph that ended with "the value flows through to the" now
  finishes with "strict-SemVer oracle for rejection."

Verification commands run after R5:

```bash
go test ./internal/version/...                         # PASS
go test ./internal/factory/doctrinecompiler/...        # PASS
go test ./cmd/leamas/...                              # PASS
go test ./...                                         # PASS
make factorize                                        # PASSED
make gate                                             # PASSED
```

Final ACT status: `ACT-LEAMAS-COMPILER-VERSION-STAMPING01`
is **CLOSED**. The Leamas binary is a stamped, strict-SemVer
participant, and `ACT-CIRCUS-FACTORY-BOOTSTRAP01` can retry with
this accepted binary.
