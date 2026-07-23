# ACT-LEAMAS-FACTORY-CLOSURE-PROTOCOL-V1-01

## Status

**OPEN — P0 developer-velocity and evidence-integrity infrastructure**

## Intent

Implement Closure Protocol v1 as a first-class Leamas capability.

The protocol separates subject, plan, manifest, report, tag, and
publication state. It removes the recursive report-and-correction
loop by making a single compact manifest the authoritative record
and by deriving the lifecycle state from Git rather than copying
identities into several documents.

## Acceptance criteria

The ACT may close only when all acceptance criteria pass.

### Model separation

1. Subject, plan, manifest, report, and tag are distinct objects.
2. The manifest binds one exact subject commit and tree.
3. The report is generated from the manifest.
4. The report is not an independent evidence authority.
5. Tag-object OIDs are external evidence only.
6. No committed object contains its own future identity.

### Plan

7. Plan decoding is strict and bounded.
8. Duplicate keys are rejected.
9. Unknown fields are rejected.
10. Trailing JSON is rejected.
11. Check IDs are unique.
12. Artifact IDs are unique.
13. Commands are argv arrays.
14. Shell strings are rejected.
15. Excluded checks require reasons.
16. Placeholders are rejected.

### Execution

17. Checks execute serially.
18. Checks execute in plan order.
19. Execution is fail-fast.
20. Failed checks are not retried.
21. Remaining checks are classified explicitly.
22. Existing bounded process execution is reused.
23. Stdout and stderr are distinct.
24. Truncation and incompleteness are distinct.
25. Cleanup failure cannot be hidden.
26. The subject worktree is clean before and after.

### Manifest

27. Every plan check appears exactly once.
28. Check order matches the plan.
29. Verdict is computed.
30. Required artifacts are hash-bound.
31. Raw logs are absent.
32. Absolute host paths are absent.
33. Secret environment values are absent.
34. Future closure/tag identities are absent.
35. Manifest verification is deterministic.

### Report

36. Report bytes are deterministic.
37. Report ends with exactly one LF.
38. Report is under 200 lines.
39. Report is under 32 KiB.
40. Report contains no raw logs.
41. Report contains no full digest.
42. Report contains no placeholder.
43. Report contains no tag-object OID.
44. Manually edited reports are rejected by tag creation.

### Evidence policy

45. Raw evidence is detached.
46. Evidence directory must be outside the worktree.
47. New tracked full digests are rejected.
48. Existing unchanged legacy digests are not reopened.
49. Detached artifacts have hashes and byte counts.
50. Artifact path and symlink safety are fail-closed.

### Tags and state

51. Tag creation is refused for a failed manifest.
52. Tag creation is refused for a dirty tree.
53. Tag creation is refused if the name exists.
54. No force option exists.
55. Tag target is the closure commit.
56. Tag message binds subject, closure, manifest, and report.
57. `status` derives `CLOSED_LOCAL`.
58. Remote verification derives `PUBLISHED`.
59. Remote verification checks both tag object and peeled target.
60. Web UI visibility is not used as publication authority.

### Speed and maintainability

61. No factorize command is invoked.
62. No dupcode scan is introduced.
63. No check runs more than once.
64. Verification/rendering of a 1,000-check manifest completes
    within the generous regression bound.
65. All changed Go files remain within LLM-friendly limits.
66. `git diff --check` passes.
67. `gate-fast` passes.
68. The ACT closes itself using the new protocol.
69. No evidence-only correction is needed to record the final
    tag-object OID.
70. The final tag is never moved.

## Self-hosting acceptance

This ACT closes itself using Closure Protocol v1.

The bootstrap sequence is:

1. Freeze the closure plan at
   `docs/closure-plans/ACT-LEAMAS-FACTORY-CLOSURE-PROTOCOL-V1-01.json`
   before the final subject commit.
2. Create the subject commit that contains the closure protocol
   implementation, tests, documentation, frozen plan, and policy
   updates. The subject commit does not contain the final
   manifest or final close report.
3. Build a clean proof binary from the subject and record the
   binary SHA-256 and VCS revision.
4. Run the proof binary against the subject to produce the compact
   manifest in a detached directory.
5. Verify the manifest, render the deterministic report, copy the
   manifest and report into the canonical repository paths, commit
   them, and create the immutable annotated tag.
6. Verify local state with `leamas factory close status` and expect
   `CLOSED_LOCAL`.
7. Publish the branch and tag and verify remote state with
   `leamas factory close status --remote origin`; expect
   `PUBLISHED`. If remote publication is unavailable the ACT is
   classified as `CLOSED_LOCAL — remote publication pending`.

## Final board transition

On successful local closure:

```text
ACT-LEAMAS-FACTORY-CLOSURE-PROTOCOL-V1-01:
  CLOSED_LOCAL

Closure Protocol v1:
  AVAILABLE
  SELF-HOSTED
  SERIAL
  FAIL-FAST

Legacy report-only closure:
  DEPRECATED for new Leamas ACTs

Full tracked digests:
  FORBIDDEN for new closure evidence

Tag-object identities:
  EXTERNAL EVIDENCE ONLY

ClineMM DOGFOOD01:
  unaffected

ACT-LEAMAS-FACTORY-CLOSURE-PROTOCOL-V1-CROSS-REPO-ADOPTION01:
  READY
```

After remote verification:

```text
ACT-LEAMAS-FACTORY-CLOSURE-PROTOCOL-V1-01:
  PUBLISHED
```

## Non-goals

The ACT does not implement cryptographic signatures, SLSA or
in-toto compliance claims, transparency logs or Rekor, Git notes
storage, remote evidence storage, automatic GitHub Release creation,
automatic force pushes, automatic retries, parallel check execution,
cross-repository adoption, deletion or rewriting of historical tags,
Gate Summary v3, Targeted Digest v3, ClineMM DOGFOOD01, or Leamas
`v0.2.0` publication. Those are separate successor scopes.

## Required verification plan

The frozen plan must include at least:

```bash
go test -count=1 \
  ./internal/factory/closure/... \
  ./cmd/leamas/...

go test -count=20 \
  ./internal/factory/closure/... \
  ./cmd/leamas/...

go test -race -count=5 \
  ./internal/factory/closure/... \
  ./cmd/leamas/...

go vet \
  ./internal/factory/closure/... \
  ./cmd/leamas/...

CGO_ENABLED=0 go build \
  -buildvcs=true \
  -trimpath \
  -o /tmp/leamas-closure-protocol-v1 \
  ./cmd/leamas

CGO_ENABLED=0 make gate-fast

git diff --check
```

Also run:

```bash
go test -count=1 \
  ./internal/factory/closure/... \
  -run 'TestClosure'
```

The exact command is frozen in the plan rather than reconstructed
in the close report.

Not required unless owned files change:

```text
make gate-dupcode
```

Not required:

```text
make factorize
```
