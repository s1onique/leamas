# B2-R7-R1 Micro-Correction Task List — Concrete Leftovers

## Phase 1: Single-source regex patterns
- [ ] Export ActIDPattern/ItemIDPattern/OIDPattern/EnvironmentNamePattern from the SAME package-private canonical vars (no second MustCompile)
- [ ] Verify validators use the exported canonical patterns

## Phase 2: Single-source placeholder set
- [ ] Export ExactClosurePlaceholders from plancontract as canonical
- [ ] Remove closure/plan.go duplicate literal map
- [ ] Tests must read from canonical probe

## Phase 3: Runner authority type parity
- [ ] Add version/tag_name type check to plancontract.validateToolBlock
- [ ] Add parity rows: version number → reject both, tag_name number → reject both, valid strings → accept both

## Phase 4: Strengthen semantic authority guard
- [ ] Detect semantic comparisons in validate* functions
- [ ] Detect regexp matching / numeric-bound logic in production code
- [ ] Detect newly introduced validate*Plan / *Authority production functions

## Phase 5: Make execution consume ValidatedPlan
- [ ] Add LoadPlanFromBytesValidated (or consume ValidatedPlan in LoadPlanFromBytes)
- [ ] Document CANONICAL_VALIDATED_PLAN_EXECUTION_MODEL

## Phase 6: Verify
- [ ] gofmt, go vet, git diff --check
- [ ] go test ./cmd/leamas/...
- [ ] Run R7 tests

## Phase 7: Clean committed-range digest
- [ ] Clean worktree (no working-tree changes)
- [ ] Single forward commit
- [ ] Produce explicit 366ede2..FINAL digest with AUTHORITY_STATUS=CleanCommitted

No B3.
No tag.
No lifecycle commit.
No ClineMM changes.
