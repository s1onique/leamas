# B2-R7 Micro-Correction Task List — Plan Contract Authority Convergence

## Phase 1: Survey current state
- [x] Map current plancontract package surface
- [x] Map closure package's ValidatePlan and helpers
- [x] Map evidence package's plan parsing
- [x] Identify all duplicated semantic validators
- [x] Identify all mirrored constants/patterns

## Phase 2: Build acceptance matrix (RED)
- [ ] Author TestPlanContractExecutionEvidenceParityR7
- [ ] Each fixture must satisfy json.Valid == true
- [ ] Cover all required rows from spec
- [ ] Confirm parity RED before refactor

## Phase 3: Establish canonical validated model
- [ ] Introduce ValidatedPlan in plancontract
- [ ] Implement DecodeAndValidateFull single-pass entry
- [ ] One call: syntax, dup-key, unknown-field, EOF, semantic, canonical projection

## Phase 4: Refactor closure execution
- [ ] Replace closure.ValidatePlan independent logic
- [ ] Reduce to: serialize → canonical call → adapt typed errors
- [ ] No duplicated rules in adapter

## Phase 5: Refactor evidence derivation
- [ ] Evidence derives ExpectedChecks from canonical ValidatedPlan
- [ ] No second parse / second validation
- [ ] EXECUTION_PLAN_AUTHORITY == EVIDENCE_PLAN_AUTHORITY by construction

## Phase 6: Adapter layer for legacy diagnostics
- [ ] plancontract.DecodeError → closure typed errors
- [ ] Code/Field/InstancePath/Message mapping
- [ ] No re-evaluation of plan in adapter

## Phase 7: Delete duplicated semantics
- [ ] validatePlanTyped, validatePlanChecks, validatePlanArtifacts, validatePlanPolicy
- [ ] ValidateRunnerAuthority, validateToolBlock
- [ ] portablePathValidate, validateRepositoryRelativePath
- [ ] Mirrored helpers/constants

## Phase 8: Single-source constants
- [ ] ContractVersionV1, MaxChecks, MaxArtifacts
- [ ] MaxArgvElements, MaxEnvironmentEntries, MaxCheckTimeoutSeconds
- [ ] Act-id, item-id, OID, env-name, SHA-256 patterns
- [ ] Placeholder sets
- [ ] Policy field list
- [ ] Runner authority mode enum

## Phase 9: Tool authority semantics
- [ ] Inventory ToolAuthority wire fields
- [ ] Migrate revision, binary_sha256 to plancontract
- [ ] Migrate tree_oid, tag_object_oid, tag_name if present

## Phase 10: Semantic authority guard
- [ ] Add TestPlanContractSingleSemanticAuthority
- [ ] AST/source inspection of closure legacy entry points

## Phase 11: Preserve strict JSON
- [ ] Duplicate top-level, nested, in-array-object rejection
- [ ] Same name in different objects → accept
- [ ] Unknown field, second object, second scalar, trailing garbage reject
- [ ] Whitespace-only suffix → accept

## Phase 12: Test migration
- [ ] Preserve typed-error contracts
- [ ] Adapter-focused tests

## Phase 13: Source hygiene
- [ ] All ACT-owned files ≤400 lines
- [ ] Split by responsibility

## Phase 14: Verification
- [ ] gofmt -w on R7 files
- [ ] go test -race ./internal/factory/closure/...
- [ ] go test ./cmd/leamas/...
- [ ] go vet ./internal/factory/closure/... ./cmd/leamas/...
- [ ] git diff --check
- [ ] llm-friendly, tooling-boundaries, long-test-policy, static-binary
- [ ] Report factorize/gate/gate-dupcode as NOT RUN

## Phase 15: Publication
- [ ] Single forward commit: factory: make closure plan semantics single-authority
- [ ] Clean-worktree go test -count=1 ./cmd/leamas/...
- [ ] No tag, no lifecycle commit, no B3, no ClineMM changes
