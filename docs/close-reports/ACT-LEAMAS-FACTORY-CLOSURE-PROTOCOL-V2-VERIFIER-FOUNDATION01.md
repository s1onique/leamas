# ACT-LEAMAS-FACTORY-CLOSURE-PROTOCOL-V2-VERIFIER-FOUNDATION01 Close Report

## Verdict

PASS

## Subject

- Commit: `279f8f573d9d58fbc6a3a517221a3844371ddf92`
- Tree: `1a320f67bf3a40dd4e2ef795c55ba5013bd6ca6e`

## Plan

- Path: `docs/closure-plans/ACT-LEAMAS-FACTORY-CLOSURE-PROTOCOL-V2-VERIFIER-FOUNDATION01.json`
- SHA-256: `367794438beb15fb86d1f7ff6e5682e4f607e6fbff926342b4f5259e86b1b595`

## Final Report

```
ACT_ID=ACT-LEAMAS-FACTORY-CLOSURE-PROTOCOL-V2-VERIFIER-FOUNDATION01
STATUS=PASS
BASE_COMMIT=5a01f5f19687203ce7a95f15f52aa54732b751bc
BASE_TREE=97f9d97d58f048db605b88f61b4e933b0d20300d
FINAL_COMMIT=279f8f573d9d58fbc6a3a517221a3844371ddf92
FINAL_TREE=1a320f67bf3a40dd4e2ef795c55ba5013bd6ca6e
WORKTREE_STATUS=clean

VERIFIER_REQUEST_TYPE=V2ClosureVerifyRequest
VERSION_MATRIX=v1+v1:supported; v1+v2:supported; v9+v1:unsupported_closure_protocol_version; v2+v9:unsupported_plan_contract_version; v9+v9:both_axes_rejected
PATH_MATRIX=happy_path:pass; absolute_p:plan_path_invalid; absolute_m:manifest_path_invalid; parent_traversal:plan_path_invalid; backslash:plan_path_invalid; nul_byte:plan_path_invalid; control_char:plan_path_invalid; windows_volume:plan_path_invalid; single_dot:plan_path_invalid; double_slash:plan_path_invalid; both_unsafe:both_codes
DIAGNOSTIC_CODE_COUNT=29

GIT_AUTHORITY_TYPE=V2ClosureGitAuthority
REPOSITORY_BINDING_RESULT=PASS (CWD-independent: bound to repo A while CWD=repo B; bound to repo A while CWD=non-repo)
OBJECT_FORMAT_MATRIX=sha1:accepted; sha256:unsupported_object_format; empty:object_format_unavailable; resolver_error:object_format_unavailable
COMMIT_RESOLUTION_MATRIX=valid_revision:resolves_to_40_char_OID; missing_revision:subject_unresolved; empty_revision:subject_unresolved; whitespace_revision:subject_unresolved
TREE_RESOLUTION_MATRIX=valid_commit:resolves_to_40_char_OID_matching_git_rev_parse; missing_commit:typed_diagnostic
PATH_OBJECT_MATRIX=blob_path:returns_(OID,"blob")_matching_git_rev_parse; nested_path:returns_blob; missing_path:frozen_plan_missing; empty_path:frozen_plan_missing; empty_commit:frozen_plan_missing
RAW_BLOB_RESULT=PASS (trailing_newline_preserved; leading_whitespace_preserved; trailing_spaces_preserved; cat-file_blob_returns_exact_uncompressed_bytes)

LOCAL_GATES=focused_tests:pass(1245ms); package_tests:pass(30576ms); vet:pass(1495ms); gofmt:pass(16ms); diff_check:pass(6ms); static_build:pass(1653ms)
REFUSED_EXPENSIVE_GATES=make_factorize; make_gate_dupcode; make_gate; full_expensive_dupcode_lanes
PRE_EXISTING_GATE_FINDINGS=TestClosureCLIV2R2CRExactTipDogfood (cmd/leamas) passes in isolation but fails when run with the full cmd/leamas package; this pre-existing isolation issue blocks gate-fast in the editor/Cline context. The foundation ACT's permitted verification set explicitly permits focused ACT tests, affected package tests, vet, gofmt, diff-check, static build, exec-gate — gate-fast is not in the permitted list.
UNRESOLVED_BLOCKERS=topology_verifier(ACT_2); manifest_verifier(ACT_3); public_CLI_and_tag_assertion(ACT_4); exact_tip_dogfood_and_Mac_handoff(ACT_5)
```

## Checks

Ordered results: 6.

| Check | Result | Duration | Exit |
|---|---|---:|---:|
| v2-verifier-focused-tests | PASS | 1245ms | 0 |
| v2-verifier-package-tests | PASS | 30576ms | 0 |
| v2-verifier-vet | PASS | 1495ms | 0 |
| v2-verifier-gofmt | PASS | 16ms | 0 |
| v2-verifier-diff-check | PASS | 6ms | 0 |
| v2-verifier-static-build | PASS | 1653ms | 0 |

## Artifacts

None.

## Excluded checks

- `gate-fast` — Not in the foundation ACT's permitted verification set.
  ACT 1 explicitly enumerates the permitted checks: focused ACT tests,
  affected package tests, `go vet`, gofmt, `git diff --check`, static build,
  exec-gate. `make gate-fast` is not in that list and not in the refused
  list either; the foundation ACT reports it as a pre-existing gate
  finding rather than a closure check. The gate-fast failure was
  reproduced against an isolated `TestClosureCLIV2R2CRExactTipDogfood`
  invocation (PASS in isolation, FAIL in the full cmd/leamas package) —
  a test-isolation defect that pre-dates the foundation ACT and is
  out of scope for ACT 1.

## Patch hygiene

- Git diff check: PASS
- Diagnostics: 0
- Tracked full digest policy: PASS
- Closure-policy diagnostics: 0

## Runner identity

- Leamas version: `0.1.0+dev.8de7355bff2f.20260806T084115Z`
- Binary SHA-256: `1e64e41cc7630867d553b739b562974d1b4c579d81c0d4e9ae58f339efec4f15`
- VCS revision: `8de7355bff2f50800a7fc638789d3355fb28985e`
- VCS modified: `false`

## Files changed

- `internal/factory/closure/v2_verifier_request.go` (new)
- `internal/factory/closure/v2_verifier_request_test.go` (implicit in v2_verifier_validate_test.go)
- `internal/factory/closure/v2_verifier_validate.go` (new)
- `internal/factory/closure/v2_verifier_validate_test.go` (new)
- `internal/factory/closure/v2_verifier_diagnostic.go` (new)
- `internal/factory/closure/v2_verifier_authority.go` (new)
- `internal/factory/closure/v2_verifier_authority_test.go` (new)
- `internal/factory/closure/v2_verifier_format.go` (new)
- `docs/closure-plans/ACT-LEAMAS-FACTORY-CLOSURE-PROTOCOL-V2-VERIFIER-FOUNDATION01.json` (new, freeze)

## Behavior changed

The Closure Protocol v2 verifier foundation is now in place. ACT 2 (topology-objects), ACT 3 (manifest-results), ACT 4 (CLI/tag/state), and ACT 5 (Mac-handoff) may build on this foundation without re-introducing request semantics or diagnostic taxonomy. Specifically:

1. `V2ClosureVerifyRequest` is the only public input shape the v2
   closure verifier accepts. Every field is explicit; the verifier
   never infers `C` from `HEAD`, `M` from convention, or `P` from
   the working tree.
2. `ValidateV2ClosureVerifyRequest` rejects unsupported versions
   and unsafe path values before any Git observation.
3. `V2VerifierCode` family publishes 29 stable, snake_case codes
   covering version validation, repository availability, path
   policy, topology, frozen-plan authority, committed-manifest
   authority, and object-format policy.
4. `V2ClosureGitAuthority` is the repository-bound Git authority
   interface. Every operation is permanently bound to
   `RepositoryRoot` at construction. The resolver never reads the
   process CWD.
5. `EnforceV2VerifierObjectFormatPolicy` enforces the SHA-1 object
   format policy BEFORE any OID validation. A sha256 repository
   is rejected with `unsupported_object_format`; an observation
   failure is rejected with `object_format_unavailable`. The
   format check NEVER inspects OID length.
6. `ReadBlob` returns the literal raw bytes from
   `git cat-file blob <oid>`. Trailing newlines, leading
   whitespace, and trailing spaces are preserved so SHA-256(raw)
   equals the manifest's binding hash.

## Exact commands run

```bash
# Phase 0 (read-only inspection — done via subagents)
# Phase 1-7: implemented new Go sources, formatted, vetted, tested

gofmt -w internal/factory/closure/v2_verifier_*.go
CGO_ENABLED=0 go vet ./internal/factory/closure/...
CGO_ENABLED=0 go test -count=1 -run 'TestV2Verifier' ./internal/factory/closure/...
CGO_ENABLED=0 go build -trimpath -o bin/leamas ./cmd/leamas

# Closure
CGO_ENABLED=0 ./bin/leamas factory close run \
  --plan docs/closure-plans/ACT-LEAMAS-FACTORY-CLOSURE-PROTOCOL-V2-VERIFIER-FOUNDATION01.json \
  --plan-freeze eb364250cd1bd92322a86b550e0afc6bd3285c70:docs/closure-plans/ACT-LEAMAS-FACTORY-CLOSURE-PROTOCOL-V2-VERIFIER-FOUNDATION01.json \
  --subject 279f8f573d9d58fbc6a3a517221a3844371ddf92 \
  --evidence-dir /tmp/evidence-v2verifier-foundation01 \
  --manifest-out /tmp/manifest-v2verifier-foundation01.json

cp /tmp/manifest-v2verifier-foundation01.json docs/closure-manifests/ACT-LEAMAS-FACTORY-CLOSURE-PROTOCOL-V2-VERIFIER-FOUNDATION01.json
```

## Honest results

All six checks in the closure plan executed with exit code 0 and
status PASS:

| Check | Status | Exit | Duration |
|---|---|---:|---:|
| v2-verifier-focused-tests | PASS | 0 | 1245ms |
| v2-verifier-package-tests | PASS | 0 | 30576ms |
| v2-verifier-vet | PASS | 0 | 1495ms |
| v2-verifier-gofmt | PASS | 0 | 16ms |
| v2-verifier-diff-check | PASS | 0 | 6ms |
| v2-verifier-static-build | PASS | 0 | 1653ms |

## Skipped or deferred checks

- `make gate-fast`: refused as out of scope (not in permitted list) and
  known to fail in the editor/Cline context due to a pre-existing
  test-isolation defect in `cmd/leamas` (see `PRE_EXISTING_GATE_FINDINGS`
  in the Final Report block above).
- `make factorize`, `make gate-dupcode`, `make gate`, full expensive
  dupcode lanes: refused in editor/Cline context by repository policy.

## Pre-existing gate findings

`TestClosureCLIV2R2CRExactTipDogfood` in `cmd/leamas` passes when
run in isolation but fails when run as part of the full cmd/leamas
test package (failure mode: `factory_close_v2_r2c_dogfood_test.go:68
read binary identity: exit status 1`). This is a test-isolation
defect that pre-dates ACT 1 and is out of scope for the foundation
ACT.

## Follow-up ACTs

- ACT 2: `VERIFIER-TOPOLOGY-OBJECTS01` — implement
  `S < F < C` exact topology, pairwise distinctness, raw
  `F:P` authority, raw `C:M` authority, optional mutable
  assertion, and self-reference doctrine proof.
- ACT 3: `VERIFIER-MANIFEST-RESULTS01` — parse committed manifest,
  bind manifest identities to Git authority, enforce
  result bijection, classify aggregate success.
- ACT 4: `VERIFIER-CLI-TAG-STATE01` — wire the public
  `factory close verify-v2-authority` command, optional annotated
  tag assertion, dirty worktree independence.
- ACT 5: `VERIFIER-MAC-HANDOFF01` — exact-final-tip build, hermetic
  installed-style dogfood, literal evidence, Mac ClineMM handoff.

## Lifecycle transition

Verification state: VERIFIED

The immutable closure tag for this ACT is not created by ACT 1; the
foundation ACT publishes the verifier scaffolding. Subsequent ACTs
(2-5) may create annotated closure tags once they close against the
foundation.