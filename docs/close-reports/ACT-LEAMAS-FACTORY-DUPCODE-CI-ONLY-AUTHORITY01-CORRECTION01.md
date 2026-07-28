# Close Report: ACT-LEAMAS-FACTORY-DUPCODE-CI-ONLY-AUTHORITY01-CORRECTION01

## ACT Identity
- **ACT-ID**: ACT-LEAMAS-FACTORY-DUPCODE-CI-ONLY-AUTHORITY01-CORRECTION01
- **Title**: Route all dupcode paths through central authority
- **Status**: D_F (Done - Frozen)
- **Baseline**: 7e95325b8fdf14400fe03c6e638820cd831d6afd (ACT-LEAMAS-FACTORY-DUPCODE-CI-ONLY-AUTHORITY01)

## Files Changed
1. `internal/factory/gate/dupcodeauthority/authority.go` - Fail-closed Git observation
2. `internal/factory/gate/dupcodeauthority/authority_test.go` - Refactored tests
3. `internal/factory/gate/dupcodeauthority/authority_guard_test.go` - Guard dispatch tests
4. `internal/factory/gate/gate.go` - RunGate/RunFactorize wired through authority
5. `Makefile` - dupcode-baseline target denial
6. `docs/closure-plans/ACT-LEAMAS-FACTORY-DUPCODE-CI-ONLY-AUTHORITY01-CORRECTION01.json`

## Behavior Changed

### Before CORRECTION01
- `RunGate` ran all verifiers including dupcode without authority check
- `RunFactorize` ran dupcode verifiers without authority check
- `make dupcode-baseline` directly invoked the baseline update command
- Git observation errors failed open (empty string treated as clean worktree)
- Guard test was vacuous: fake verifier never called but not asserted through Guard()

### After CORRECTION01
- `RunGate` checks authority for all dupcode-lane verifiers before execution
- `RunFactorize` denies immediately if not in CI context
- `make dupcode-baseline` exits with canonical denial message
- Git observation errors fail-closed via HeadErr/StatusErr fields
- Guard test proves fake verifier count == 0 after Guard() dispatch

## Commands Run
- `CGO_ENABLED=0 make gate-fast` → PASSED
- `./bin/leamas factory gate --lane=dupcode` → Denied with canonical error
- `make dupcode-baseline` → Denied with canonical error

## Verification Results
- gate-fast: PASSED
- Local dupcode lane: Properly denied
- Local dupcode-baseline: Properly denied
- gofmt: OK
- go vet: OK
- go test -short: OK

## Skipped/Deferred
- Full gate (requires CI context)
- make factorize (requires CI context)
- make gate-dupcode (requires CI context)

## Follow-up ACTs
None required. All local execution paths are now routed through the central authority.
