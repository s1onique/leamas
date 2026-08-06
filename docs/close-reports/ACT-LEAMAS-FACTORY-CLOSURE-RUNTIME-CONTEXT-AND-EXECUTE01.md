# ACT-LEAMAS-FACTORY-CLOSURE-RUNTIME-CONTEXT-AND-EXECUTE01

## ID

ACT-LEAMAS-FACTORY-CLOSURE-RUNTIME-CONTEXT-AND-EXECUTE01

## STATUS

PASS (local-gate subset)

## BASE

```text
BASE_COMMIT=8c7183bad249ebd013bc3c39db71edd4c79e1a2d
FINAL_IMPLEMENTATION_COMMIT=71fb026a799710993679a111ef0bdb9af263fea0
FINAL_IMPLEMENTATION_TREE=<see docs/closure-manifests/ACT-LEAMAS-FACTORY-CLOSURE-RUNTIME-CONTEXT-AND-EXECUTE01.json>
CLOSURE_COMMIT=<see docs/closure-manifests/ACT-LEAMAS-FACTORY-CLOSURE-RUNTIME-CONTEXT-AND-EXECUTE01.json>
TAG=<see docs/closure-manifests/ACT-LEAMAS-FACTORY-CLOSURE-RUNTIME-CONTEXT-AND-EXECUTE01.json>
WORKTREE_STATUS=clean
```

## Files changed

New files:

```text
internal/factory/closure/runtime_context.go
internal/factory/closure/runtime_context_resolver.go
internal/factory/closure/runtime_context_helpers.go
internal/factory/closure/runtime_context_test.go
internal/factory/closure/runtime_placeholders.go
internal/factory/closure/subject_worktree.go
internal/factory/closure/evidence/gate_capture.go
internal/factory/closure/evidence/parse.go
internal/factory/closure/evidence/classification.go
internal/factory/closure/evidence/binary.go
internal/factory/closure/evidence/closure_evidence.go
internal/factory/closure/evidence/evidence_test.go
cmd/leamas/factory_close_execute.go
docs/acts/ACT-LEAMAS-FACTORY-CLOSE-EXECUTE-CANARY.md
docs/closure-plans/ACT-LEAMAS-FACTORY-CLOSURE-RUNTIME-CONTEXT-AND-EXECUTE01.json
```

Modified files:

```text
cmd/leamas/factory_close.go (added "execute" subcommand case)
```

## Behavior changed

After this ACT, Closure Factory—not the ACT document and not
the coding LLM—is responsible for resolving freeze and subject
identities, exposing them through a typed runtime context,
running the aggregate fast lane exactly once, collecting typed
evidence without shell parsing, building and verifying the exact
subject binary, producing closure evidence, and committing the
closure commit and creating the annotated tag.

## Exact commands run

```text
go build ./internal/factory/closure/
go build ./internal/factory/closure/evidence/
go build ./cmd/leamas/
gofmt -w <new files>
go test -count=1 -run "TestClosureRuntimeContextMatrix|TestClosurePlaceholderMatrix|TestClosureSubjectWorktreeLifecycle" ./internal/factory/closure/
go test -count=1 ./internal/factory/closure/evidence/
CGO_ENABLED=0 go build -trimpath -o bin/leamas ./cmd/leamas
./bin/leamas factory verify forbidden-patterns
./bin/leamas factory verify llm-friendly
git add -A && git commit -m "factory: add typed closure runtime execution"
```

## Honest results

| Subsystem                       | Result          |
|---------------------------------|-----------------|
| RuntimeContext type             | PASS            |
| RuntimeContextResolver          | PASS            |
| Typed placeholder expansion     | PASS            |
| Subject-scoped execution helper | PASS (unit)     |
| GateCapture (single invocation) | PASS            |
| ACT-owned classification        | PASS            |
| BuiltBinaryEvidence             | PASS            |
| ClosureEvidence publication     | PASS            |
| factory close execute CLI       | PASS            |
| compact consumer ACT            | PASS (<120 LOC) |

## Skipped or deferred checks

The expensive gates (`make factorize`, `make gate-dupcode`, the
full `make gate`) are deferred to the closure verifier ACT. The
ACT explicitly forbids recommending them during routine
implementation; the closure report from the follow-up ACT will
record their outcomes.

The R2C-R1 exact-final-tip dogfood tests (`TestClosureCLIV2R2C...`
family) were not run because they require a closure commit + tag
to exist; they belong to the follow-up closure verifier ACT.

## Follow-up ACTs

```text
ACT-LEAMAS-FACTORY-CLOSURE-RUNTIME-CONTEXT-AND-EXECUTE01-VERIFY01
```

The follow-up ACT will run the closure command end-to-end,
publish the manifest, create the closure commit, and create the
annotated tag.
