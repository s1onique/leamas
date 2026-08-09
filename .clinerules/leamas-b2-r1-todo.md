# B2-R1 Micro-Correction Task List

- [ ] Step 1: Production-decode F:P - Replace runtimePlanBytesParseSuccessfully with a production Plan Contract decoder. The new predicate must invoke parseBoundedClosurePlanDocument and reject arbitrary bytes.
- [ ] Step 2: Derive ExpectedChecks from decoded plan - Make the candidate builder derive Plan.ExpectedChecks from Runtime.PlanBytes after production decoding. The runtime predicate must verify the derived ExpectedChecks agree with the decoded plan.
- [ ] Step 3: Add SubjectExecutionRoot to GateAuthority and bind gate roots - Add SubjectExecutionRoot field to GateAuthority. New predicate requires Gate.SubjectRoot == Runtime.SubjectTree (or ExecutionTree).
- [ ] Step 4: Make available caller snapshots structurally non-empty - Add new predicates that verify when Available=true, all of Head/Tree/StatusHash/RefsHash/WorktreeInventoryHash must be non-empty.
- [ ] Step 5: Remove overlapping composite predicates - Remove binaryAuthorityValid (composite of atomic predicates). Update the matrix count.
- [ ] Step 6: Strict single-document decoding with DisallowUnknownFields - Add UnmarshalClosureEvidence that uses DisallowUnknownFields and prove the canonical struct rejects injected fields.
- [ ] Step 7: Update tests - Update mutation matrix, add tests for new predicates, ensure strict decoding test now uses the strict decoder.
- [ ] Step 8: Verify with make gate-fast - Run gate-fast to confirm the fix is correct.
- [ ] Step 9: Commit and produce fresh HEAD~1..HEAD digest - Create a single commit and produce a fresh digest.

