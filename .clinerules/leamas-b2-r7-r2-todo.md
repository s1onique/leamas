# B2-R7-R2 Micro-Correction Task List — Final Wiring

## Phase 1: Bind closure execution to canonical plan model
- [ ] LoadPlanFromBytes calls plancontract.DecodeAndValidateFull (canonical ValidatedPlan path)
- [ ] Add LoadPlanFromBytesWithValidated returning both typed Plan and ValidatedPlan
- [ ] Add umbrella test that the production runner path consumes the canonical loader
- [ ] Update validatePlanTyped to consume the ValidatedPlan from the same decode pass

## Phase 2: Strengthen AST guard or honestly mark as heuristic
- [ ] Improve selectorNameFromExpr to follow nested selectors (plan.Baseline.CommitOID)
- [ ] Broaden new-validator function shape detection
- [ ] Honest SEMANTIC_AUTHORITY_GUARD naming: AST passes; HEURISTIC qualifier if weak

## Phase 3: Immutable placeholder exposure
- [ ] Unexport ExactClosurePlaceholders map (or make read-only copy)
- [ ] Unexport EmbeddedClosurePlaceholders slice (or make read-only copy)
- [ ] Add ExactClosurePlaceholdersCopy() for tests that need a snapshot
- [ ] Validate via parity matrix that mutation does not affect authority

## Phase 4: Verify
- [ ] gofmt, go vet, git diff --check
- [ ] go test ./cmd/leamas/... on clean worktree
- [ ] Run R7 + R7-R1 + R7-R2 tests

## Phase 5: Clean committed-range digest
- [ ] Clean worktree
- [ ] Single forward commit (factory: bind closure execution to canonical plan model)
- [ ] Produce explicit 366ede2..FINAL digest

No B3. No lifecycle commit. No tag. No ClineMM changes.
