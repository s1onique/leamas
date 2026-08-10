# B2-R7 Micro-Correction Task List — Plan Contract Authority Convergence

## Phase 1: Survey current state
- [x] Map current plancontract package surface
- [x] Map closure package's ValidatePlan and helpers
- [x] Map evidence package's plan parsing
- [x] Identify all duplicated semantic validators
- [x] Identify all mirrored constants/patterns

## Phase 2: Build acceptance matrix (RED)
- [x] Author TestPlanContractExecutionEvidenceParityR7
- [x] Each fixture satisfies json.Valid == true
- [x] Cover all required rows from spec
- [x] Confirm parity before refactor (passes after refactor)

## Phase 3: Establish canonical validated model
- [x] Introduce ValidatedPlan in plancontract
- [x] Implement DecodeAndValidateFull single-pass entry
- [x] One call: syntax, dup-key, unknown-field, EOF, semantic, canonical projection

## Phase 4: Refactor closure execution
- [x] Replace closure.ValidatePlan independent logic
- [x] Reduce to: serialize → canonical call → adapt typed errors
- [x] No duplicated rules in adapter

## Phase 5: Refactor evidence derivation
- [x] Evidence derives ExpectedChecks from canonical ValidatedPlan
- [x] No second parse / second validation
- [x] EXECUTION_PLAN_AUTHORITY == EVIDENCE_PLAN_AUTHORITY by construction

## Phase 6: Adapter layer for legacy diagnostics
- [x] plancontract.DecodeError → closure typed errors
- [x] Code/Field/InstancePath/Message mapping
- [x] No re-evaluation of plan in adapter

## Phase 7: Delete duplicated semantics
- [x] validatePlanTyped, validatePlanChecks, validatePlanArtifacts, validatePlanPolicy
- [x] ValidateRunnerAuthority, validateToolBlock (made adapter-only)
- [x] portablePathValidate, validateRepositoryRelativePath
- [x] Mirrored helpers/constants

## Phase 8: Single-source constants
- [x] ContractVersionV1, MaxChecks, MaxArtifacts
- [x] MaxArgvElements, MaxEnvironmentEntries, MaxCheckTimeoutSeconds
- [x] Act-id, item-id, OID, env-name, SHA-256 patterns
- [x] Placeholder sets
- [x] Policy field list
- [x] Runner authority mode enum

## Phase 9: Tool authority semantics
- [x] Inventory ToolAuthority wire fields
- [x] Migrate revision, binary_sha256 to plancontract
- [x] Migrate tree_oid, tag_object_oid, tag_name if present

## Phase 10: Semantic authority guard
- [x] Add TestPlanContractSingleSemanticAuthority
- [x] AST/source inspection of closure legacy entry points

## Phase 11: Preserve strict JSON
- [x] Duplicate top-level, nested, in-array-object rejection
- [x] Same name in different objects → accept
- [x] Unknown field, second object, second scalar, trailing garbage reject
- [x] Whitespace-only suffix → accept

## Phase 12: Test migration
- [x] Preserve typed-error contracts
- [x] Adapter-focused tests

## Phase 13: Source hygiene
- [x] All ACT-owned files ≤400 lines

## Phase 14: Verification
- [x] gofmt -w on R7 files
- [x] go vet ./...
- [x] git diff --check (clean)
- [x] llm-friendly, tooling-boundaries, long-test-policy, static-binary
- [x] Report factorize/gate/gate-dupcode as NOT RUN

## Phase 15: Publication
- [x] Single forward commit: factory: make closure plan semantics single-authority
- [x] Clean-worktree go test -count=1 ./cmd/leamas/...
- [x] No tag, no lifecycle commit, no B3, no ClineMM changes
