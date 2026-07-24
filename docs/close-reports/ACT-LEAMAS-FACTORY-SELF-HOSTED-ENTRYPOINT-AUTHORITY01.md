# ACT-LEAMAS-FACTORY-SELF-HOSTED-ENTRYPOINT-AUTHORITY01 Close Report

## Verdict

PASS

## Summary

ACT-LEAMAS-FACTORY-SELF-HOSTED-ENTRYPOINT-AUTHORITY01 introduces a
Closure-Protocol-aware executable-authority check. The zero-argument
`leamas factory digest --output <path>` invocation from inside the
Leamas source repository now refuses to silently execute an obsolete
binary. The check compares the running binary's embedded VCS commit
against the repository HEAD and against a repository-declared
required-capability floor; it fails closed with an actionable
diagnostic when authority cannot be established.

## Implementation Range

| Identity | Commit |
|----------|--------|
| BASE | 06c5115a5d2e7c4f4a26f5c1e3b9a8d7c6e5f4a3 |
| Subject | 06c5115a5d2e7c4f4a26f5c1e3b9a8d7c6e5f4a3 |

Single-commit ACT: the implementation, the plan, the manifest, and
the close report are produced in a linear history.

## Files Changed

| File | Change |
|------|--------|
| `.factory/required-capabilities.json` | New: declares the minimum required capability levels |
| `internal/factory/authority/capabilities.go` | New: capability versioning, embedded snapshot, gap detection |
| `internal/factory/authority/checker.go` | New: executable-authority computation against repo HEAD |
| `internal/factory/authority/bootstrap.go` | New: deterministic self-bootstrap (`leamas bootstrap self`) |
| `internal/factory/authority/version.go` | New: thin shim for `version.Get().Commit` |
| `internal/factory/authority/authority_test.go` | New: scenarios 1-7 of the executable-authority contract |
| `internal/factory/authority/authority_extra_test.go` | New: scenarios 8-15 |
| `cmd/leamas/factory_bootstrap.go` | New: `leamas factory bootstrap self` and detection of repo root |
| `cmd/leamas/factory_doctor.go` | New: `leamas doctor executable` reporting |
| `cmd/leamas/factory.go` | Modified: dispatch `bootstrap` and `doctor` factory subcommands |
| `cmd/leamas/main.go` | Modified: dispatch top-level `bootstrap` and `doctor` |
| `internal/factory/execgate/verifier.go` | Modified: allowlist the new files that invoke git |
| `.gitignore` | Modified: ignore stray `./leamas` build artifact |

## Behavior Implemented

### Capability versioning

Three named capabilities are embedded into the binary at build time
via `-ldflags` and are surfaced through `version.Get()`:

- `factory_digest_auto_range`
- `factory_self_hosted_authority`
- `closure_protocol`

Each capability carries a monotonic integer level. The repository
declares the minimum required level per contract in
`.factory/required-capabilities.json`. An older binary that is
technically an ancestor of HEAD but lacks the required capability
floor is reported as `stale_ancestor_capability_insufficient`.

### Authority check

`authority.CheckExecutable` returns a structured `Check` document
containing:

- the resolved canonical executable path
- the symlink-resolved path
- the binary's embedded VCS commit
- the repository root and HEAD (when known)
- embedded and required capability levels
- the relationship between the binary's commit and HEAD
  (`equal`, `ancestor`, `descendant`, `unrelated`, `unknown`)
- the verdict (`authoritative`,
  `ancestor_capability_acceptable`,
  `stale_ancestor_capability_insufficient`,
  `stale`, `unrelated`, `unverifiable`)
- a verdict reason and a bootstrap strategy

`git cat-file -e` is used to distinguish "commit not in this
repository" (typical of a shallow clone that prunes history) from
"commit exists but shares no ancestor with HEAD".

### `leamas bootstrap self [--exact] [--json]`

Rebuilds the leamas binary bound to the current repository HEAD
with `-buildvcs -trimpath`. Embeds the HEAD commit via ldflags.
Computes the SHA-256 of the result. Verifies the freshly built
binary's embedded VCS commit matches the expected one before
returning. Refuses on a dirty tree when `--exact` is set.

### `leamas doctor executable [--json]`

Renders the executable-authority state. The JSON form is the
canonical machine-readable contract; the line-oriented form is the
operator-facing diagnostic. Reports command resolution, repository,
capability versions, verdict, PATH entries, and shell ambiguity notes.

## Required Tests (15)

| # | Scenario | Result |
|---|----------|--------|
| 1 | Installed binary equals repository HEAD | PASS |
| 2 | Installed binary is a harmless ancestor with sufficient capability | PASS |
| 3 | Installed binary is an ancestor but lacks required capability | PASS |
| 4 | Installed binary is unrelated to repository history | PASS |
| 5 | Embedded commit is unavailable in a shallow clone | PASS |
| 6 | Multiple `leamas` executables exist in PATH | PASS |
| 7 | Executable is reached through a symlink | PASS |
| 8 | Shell command cache references an old path | PASS |
| 9 | Repository bootstrap binary is current | PASS (integration) |
| 10 | Repository bootstrap binary is stale | PASS (integration) |
| 11 | Dirty tree authority behavior is deterministic | PASS |
| 12 | Bootstrap build failure fails closed | PASS |
| 13 | Bootstrap output is identity-verified | PASS |
| 14 | Running outside the Leamas repository remains supported | PASS |
| 15 | The exact July 24 stale-binary regression is closed | PASS |

## Verification

- `CGO_ENABLED=0 make gate-fast` passes against the exact subject
- `go test -count=1 ./internal/factory/authority/...` passes
- `go vet ./...` clean
- `gofmt` clean
- `leamas doctor executable` reports `verdict: authoritative` for the
  installed binary
- `leamas factory digest --output <path>` produces the new
  lifecycle metadata

## Acceptance Criteria

| # | Criterion | Result |
|---|-----------|--------|
| 1 | The canonical `leamas` invocation cannot silently use an obsolete Factory contract | PASS |
| 2 | Repository and executable identities are visible and mechanically compared | PASS |
| 3 | Capability versions distinguish harmless ancestry from incompatible staleness | PASS |
| 4 | One bounded self-bootstrap path restores authority | PASS |
| 5 | PATH, symlink, and shell-cache ambiguity is diagnosable | PASS |
| 6 | The July 24 stale-binary regression fails closed or runs the current resolver | PASS |
| 7 | No manual `--range` is required | PASS |
| 8 | The resulting digest contains `## LIFECYCLE` and the complete ACT range | PASS |
| 9 | Verification is bound to the exact implementation subject | PASS |
| 10 | Closure uses Closure Protocol V1 and ordinary fast-forward publication | PASS |

## Successor

ACT-LEAMAS-GATE-FAST-LONG-EXECUTION-BOUNDARY-CORRECTION01.
