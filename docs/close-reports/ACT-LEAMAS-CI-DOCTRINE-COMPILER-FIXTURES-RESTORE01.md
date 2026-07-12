# ACT-LEAMAS-CI-DOCTRINE-COMPILER-FIXTURES-RESTORE01

## Summary

Restored the canonical doctrine-compiler golden fixture tree that had been
silently excluded from version control by the repository-wide `testdata/`
ignore rule, and removed the misleading error-swallowing in
`TestCompileRepairsManagedFiles` that turned fixture-missing into a false
"managed file not repaired" failure.

No production doctrine-compiler behaviour changed. The CI failure was a
**fixture-tracking bug**, not a compiler regression.

## Diagnosis

- `.gitignore` line 19 contained the rule `testdata/`, which matches any
  directory named `testdata` at any depth.
- `git ls-files internal/factory/doctrinecompiler/testdata/` returned no
  files.
- `git check-ignore -v --no-index
  internal/factory/doctrinecompiler/testdata/fsharp-elm-empty/expected/Makefile`
  confirmed `.gitignore:19:testdata/` was the excluding rule.
- The four failing tests (`TestCompileEmptyTargetProducesGoldenTree`,
  `TestCompileRepairsManagedFiles`, `TestIntegrationGoldenProjection`,
  `TestIntegrationFixturesPresent`) all read paths under
  `testdata/fsharp-elm-empty/expected/`. On a fresh checkout (CI), none
  of those files exist.
- `TestCompileRepairsManagedFiles` swallowed both `os.ReadFile` errors
  (`got, _ := ...; golden, _ := ...`). When the golden was missing,
  `golden` was `nil`, the comparison looked like a repair failure, and
  the test printed the misleading `"managed file not repaired"`.

## Files Changed

### `.gitignore`

Re-included the doctrine-compiler fixture subtree after the broad
`testdata/` rule, with a comment explaining why.

```gitignore
# Exception: keep the adversarial test helper source
# (must come BEFORE testdata/ rule below)
!internal/execution/testdata/testhelper/main.go
testdata/

# Canonical doctrine compiler golden fixtures are source-controlled
# and must be visible to `git add` despite the broad testdata/ rule above.
!internal/factory/doctrinecompiler/testdata/
!internal/factory/doctrinecompiler/testdata/**
```

### `internal/factory/doctrinecompiler/compile_test.go`

Replaced the two discarded `os.ReadFile` errors in
`TestCompileRepairsManagedFiles` with the existing `mustRead` helper, so a
missing golden fixture fails the test with a truthful diagnostic instead
of a misleading "managed file not repaired".

```diff
-    got, _ := os.ReadFile(mk)
-    golden, _ := os.ReadFile(filepath.Join(expectedFixtureDir(t), ".factory/generated/factory.mk"))
+    got := mustRead(t, mk)
+    golden := mustRead(t, filepath.Join(expectedFixtureDir(t), ".factory/generated/factory.mk"))
     if string(got) != string(golden) {
         t.Errorf("managed file not repaired")
     }
```

### Newly tracked fixture files (6 golden + 1 documentation)

Golden fixtures (required by `TestIntegrationFixturesPresent`):

```
internal/factory/doctrinecompiler/testdata/fsharp-elm-empty/expected/.factory/doctrine.lock.json
internal/factory/doctrinecompiler/testdata/fsharp-elm-empty/expected/.factory/project.json
internal/factory/doctrinecompiler/testdata/fsharp-elm-empty/expected/.factory/generated/factory.mk
internal/factory/doctrinecompiler/testdata/fsharp-elm-empty/expected/.factory/generated/doctrine-inventory.md
internal/factory/doctrinecompiler/testdata/fsharp-elm-empty/expected/Makefile
internal/factory/doctrinecompiler/testdata/fsharp-elm-empty/expected/docs/factory/README.md
```

Fixture documentation (purpose, layout, regeneration procedure,
determinism and secret/path constraints):

```
internal/factory/doctrinecompiler/testdata/fsharp-elm-empty/README.md
```

Verified via:

```bash
test "$(git ls-files internal/factory/doctrinecompiler/testdata/fsharp-elm-empty/expected | wc -l | tr -d ' ')" -eq 6
# OK: 6 golden files tracked
```

## Behaviour Changed

- The six golden fixture files now travel with the repository, so CI
  checkouts contain them and the four previously-failing tests pass.
- `TestCompileRepairsManagedFiles` no longer hides fixture-read errors.
  A missing golden fixture now fails with the truthful
  `read .../.factory/generated/factory.mk: no such file or directory`
  instead of the misleading `managed file not repaired`.
- No production code, no pack content, no public API changed.

## Verification

### RED (CI-equivalent state, before the patch)

With the fixture tree temporarily removed to simulate a fresh checkout:

```
=== RUN   TestCompileEmptyTargetProducesGoldenTree
    compile_test.go:44: missing golden .factory/doctrine.lock.json: open ...: no such file or directory
--- FAIL: TestCompileEmptyTargetProducesGoldenTree
=== RUN   TestCompileRepairsManagedFiles
    compile_test.go:164: managed file not repaired
--- FAIL: TestCompileRepairsManagedFiles
=== RUN   TestIntegrationGoldenProjection
    integration_test.go:80: read ...: no such file or directory
--- FAIL: TestIntegrationGoldenProjection
=== RUN   TestIntegrationFixturesPresent
    integration_test.go:101: missing fixture .../doctrine.lock.json: stat ...: no such file or directory
    integration_test.go:101: missing fixture .../project.json: stat ...: no such file or directory
    integration_test.go:101: missing fixture .../factory.mk: stat ...: no such file or directory
    integration_test.go:101: missing fixture .../doctrine-inventory.md: stat ...: no such file or directory
    integration_test.go:101: missing fixture .../Makefile: stat ...: no such file or directory
    integration_test.go:101: missing fixture .../README.md: stat ...: no such file or directory
--- FAIL: TestIntegrationFixturesPresent
```

This matches the four CI failures in the task report exactly. The
misleading "managed file not repaired" diagnostic comes from the
discarded `os.ReadFile` errors.

### GREEN (after the patch)

```
go test ./internal/factory/doctrinecompiler \
    -run 'TestCompileEmptyTargetProducesGoldenTree|TestCompileRepairsManagedFiles|TestIntegrationGoldenProjection|TestIntegrationFixturesPresent' \
    -count=1 -v

=== RUN   TestCompileEmptyTargetProducesGoldenTree
--- PASS: TestCompileEmptyTargetProducesGoldenTree (0.04s)
=== RUN   TestCompileRepairsManagedFiles
--- PASS: TestCompileRepairsManagedFiles (0.05s)
=== RUN   TestIntegrationGoldenProjection
--- PASS: TestIntegrationGoldenProjection (0.04s)
=== RUN   TestIntegrationFixturesPresent
--- PASS: TestIntegrationFixturesPresent (0.00s)
PASS
ok  	github.com/s1onique/leamas/internal/factory/doctrinecompiler
```

### Repository gate

```
make factorize     # *** FACTORIZE PASSED ***
make gate          # *** GATE PASSED ***
                    #   go mod tidy... OK
                    #   gofmt... OK
                    #   go vet ./... OK
                    #   go test ./... OK
                    #   static build... OK

go test ./...                              # all packages OK
go vet ./...                               # clean
CGO_ENABLED=0 go build -trimpath \
    -o bin/leamas ./cmd/leamas             # build OK (11,651,282 bytes)
```

## Skipped / Deferred

- `internal/factory/doctrinecompiler/testdata/fsharp-elm-empty/mutations/`
  and `.../source/` are empty directories. Git does not track empty
  directories, so they remain absent from the index by design.
- The fixture tree's `README.md` (purpose / layout / regeneration
  procedure) is included as fixture documentation alongside the six
  golden files. It is not required by any test but documents the
  fixture contract.

## Follow-up ACTs

- None required for this fix. If anyone later adds user-facing test
  fixtures elsewhere under `testdata/`, they should either justify a
  new explicit re-include clause or move the fixtures out of `testdata/`.

## Verdict

- The doctrine compiler was never broken. CI failures were caused by
  uncommitted golden fixtures.
- `.gitignore` now re-includes the canonical doctrine-compiler fixture
  subtree explicitly, with a documenting comment.
- `TestCompileRepairsManagedFiles` no longer swallows fixture-read
  errors and reports honest diagnostics.
- All four previously failing tests pass; `make factorize`, `make gate`,
  `go test ./...`, `go vet ./...`, and the static build all pass.