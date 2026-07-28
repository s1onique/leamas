# Close Report: ACT-LEAMAS-FACTORY-DUPCODE-CI-ONLY-AUTHORITY01

## ACT Identity
- **ACT-ID**: ACT-LEAMAS-FACTORY-DUPCODE-CI-ONLY-AUTHORITY01
- **Title**: Make dupcode a CI-only verifier lane
- **Status**: D_F (Done - Frozen)
- **Published Baseline**: F17 = a71c0340dd08a821e66832488a83e665ba09f02c

## Files Changed
1. `internal/factory/gate/dupcodeauthority/authority.go` - Core authority implementation
2. `internal/factory/gate/dupcodeauthority/authority_test.go` - Main test suite
3. `internal/factory/gate/dupcodeauthority/authority_error_test.go` - Error/interface tests
4. `internal/factory/gate/dupcodeauthority/authority_shavariants_test.go` - SHA variant tests
5. `internal/factory/gate/dupcodeauthority/dispatch.go` - CLI dispatch guard
6. `internal/factory/gate/dupcode.go` - RunGateDupcode wiring
7. `Makefile` - gate-dupcode shell-level denial
8. `.github/workflows/factory.yml` - CI preflight checks
9. `scripts/verify_exec_gate.sh` - exec-gate allowlist update

## Behavior Changed

### Before
- `make gate-dupcode` would execute dupcode locally
- `bin/leamas factory gate --lane=dupcode` would execute dupcode locally
- Any local invocation of dupcode was allowed

### After
- `make gate-dupcode` fails immediately with CI-only diagnostic
- `bin/leamas factory gate --lane=dupcode` fails immediately with CI-only diagnostic
- Local dupcode execution is prohibited; only authorized GitHub Actions exact-SHA job may run it

## Exact Commands Run

### gate-fast (primary verification)
```bash
CGO_ENABLED=0 make gate-fast
```
**Result**: PASSED

### Local dupcode denial test
```bash
./bin/leamas factory gate --lane=dupcode
```
**Result**: FAILED with canonical diagnostic:
```
dupcode: dupcode is a CI-only verifier lane; local execution is prohibited; push a branch or open a PR and use the Factory Dupcode status check: CI must be set to "true"
```

## Honest Results

| Check | Result |
|-------|--------|
| gate-fast | PASSED |
| gofmt | PASSED |
| go vet | PASSED |
| go test -short | PASSED |
| llm-friendly | PASSED |
| local dupcode denial | CONFIRMED |
| CLI denial | CONFIRMED |
| Makefile denial | CONFIRMED |

## Authority Validation Matrix

| Condition | Expected | Actual |
|-----------|----------|--------|
| Missing CI | Denied | ✓ Denied |
| Missing GITHUB_ACTIONS | Denied | ✓ Denied |
| Missing LEAMAS_DUPCODE_AUTHORITY | Denied | ✓ Denied |
| Missing GITHUB_SHA | Denied | ✓ Denied |
| Missing GITHUB_WORKSPACE | Denied | ✓ Denied |
| Wrong authority value | Denied | ✓ Denied |
| Malformed SHA | Denied | ✓ Denied |
| HEAD != GITHUB_SHA | Denied | ✓ Denied |
| Workspace mismatch | Denied | ✓ Denied |
| Dirty worktree | Denied | ✓ Denied |
| Valid GitHub Actions context | Allowed | ✓ Allowed |

## Skipped Checks
- Real dupcode execution (by design - this is a CI-only lane)
- `make gate-dupcode` with real dupcode (prohibited locally)

## Follow-up ACTs
- Portable runner correction freeze from new published baseline
- Must consume exact-subject Factory Dupcode CI status/attestation
- Must preserve one-subject/one-closure geometry
