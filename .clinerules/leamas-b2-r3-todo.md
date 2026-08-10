# B2-R3 Micro-Correction Task List

- [ ] Step 1: Move full Plan Contract semantics into plancontract
  - contract_version MUST equal 1
  - Migrate structural validation rules from closure.ValidatePlan
  - closure execution and evidence must call the same authority
  - No second semantic validator remains
- [ ] Step 2: Add differential regression test
  - For a matrix of valid/invalid plans, execution and evidence
    validation results must be identical
  - Explicit row: contract_version=2 -> reject by both
- [ ] Step 3: Bind Gate.RepositoryRoot to Runtime.RepositoryRoot
  - New predicate: gateRepositoryRootEqualsRuntimeRepositoryRoot
  - Add empty and mismatch rows
- [ ] Step 4: Reject duplicate JSON member names in evidence
  - Preserve: unknown-fields rejection, second-Decode == io.EOF
  - Add: duplicate top-level known field -> reject
  - Add: duplicate nested known field -> reject
  - Use plancontract's strict scanner
- [ ] Step 5: Update tests
  - Update mutation matrix
  - Add new rows for predicate and duplicate-field tests
- [ ] Step 6: Verify with go test
- [ ] Step 7: Single commit and fresh digest
