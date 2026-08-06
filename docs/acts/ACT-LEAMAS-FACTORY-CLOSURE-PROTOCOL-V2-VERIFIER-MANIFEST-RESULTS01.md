# ACT-LEAMAS-FACTORY-CLOSURE-PROTOCOL-V2-VERIFIER-MANIFEST-RESULTS01

## Status

PASS

## Mission

Parse and verify the committed manifest at `C:M` against
the independently computed Git authority and the frozen plan
at `F:P`.

This ACT owns only:

1. strict manifest parsing;
2. manifest identity and version binding;
3. binary identity structural validation;
4. frozen-plan check inventory parsing via the production
   contract;
5. strict check/result bijection enforcement;
6. aggregate run-success integrity;
7. the deterministic `V2ClosureVerification` result model.

It does not own CLI, tags, caller-state dogfood, or Mac
handoff.

## Phase 0 — architecture inventory

The Phase 0 inventory is supplied by ACT 1
(VERIFIER-FOUNDATION01) and ACT 2
(VERIFIER-TOPOLOGY-OBJECTS01). ACT 3 reuses the closed
verifier request, the bound `V2ClosureGitAuthority`, the
frozen-plan and committed-manifest authorities, and the
typed `V2VerifierCode` family.

## Phase 1 — strict manifest parser

`parseV2StrictManifest` parses the exact committed manifest
bytes using `encoding/json` with `UseNumber` and
`DisallowUnknownFields`. It rejects:

- empty bytes;
- non-JSON bytes;
- trailing non-whitespace JSON garbage after the top-level
  object;
- wrong top-level type (anything other than `{ ... }`);
- duplicate top-level keys (caught by the structural map pass);
- unknown top-level fields (`closure_manifest_contract_invalid`);
- `closure_protocol_version != "2"`
  (`manifest_protocol_version_mismatch`);
- `plan_contract_version != "1"`
  (`manifest_plan_contract_version_mismatch`).

All numeric fields parse as `json.Number` so the verifier
never silently coerces integers or floats.

## Phase 2 — manifest identity verification

`verifyV2ManifestIdentityWithAnchor` independently compares
each identity field against the externally supplied topology
anchor plus the frozen-plan authority:

- `subject_commit`, `subject_tree`, `freeze_commit`,
  `freeze_tree`, `execution_tree` from `V2ClosureTopology`;
- `plan_path`, `plan_blob`, `plan_sha256` from
  `V2FrozenPlanAuthority`.

Each mismatch emits exactly one typed diagnostic and never
trusts one manifest field as evidence for another.

Stable codes:

```text
manifest_subject_mismatch
manifest_subject_tree_mismatch
manifest_freeze_mismatch
manifest_freeze_tree_mismatch
manifest_execution_tree_mismatch
manifest_plan_path_mismatch
manifest_plan_blob_mismatch
manifest_plan_sha256_mismatch
```

## Phase 3 — binary identity

`verifyV2ManifestBinaryIdentity` validates the committed
manifest's `leamas_binary_identity` as a structural assertion:

- `path` nonempty;
- `sha256` is 64 lowercase hexadecimal characters;
- `vcs_revision` is a full 40-char lowercase Git OID;
- `vcs_modified == false`;
- `leamas_version` nonempty.

The verifier never rehashes the historical binary; the
manifest is the assertion and the binary lives only at the
time the runner constructed it.

Stable code: `manifest_binary_identity_invalid`.

## Phase 4 — frozen-plan check inventory

`verifyV2FrozenPlanInventory` parses the frozen plan bytes via
the production `DecodePlan` contract. The plan is rejected as
`frozen_plan_invalid` when it fails the production parser; the
verifier never trusts the manifest's claims about the plan.

The canonical inventory is sorted by check ID so downstream
comparison is deterministic.

## Phase 5 — result bijection

`verifyV2ManifestCheckBijection` enforces a strict bijection
between the manifest's `check_results` and the canonical plan
check inventory:

- exactly one result per plan check;
- no extra result;
- result IDs unique;
- result mode equals plan mode;
- canonical order: results are emitted in canonical plan ID
  order.

Stable codes:

```text
manifest_check_result_bijection_failed
manifest_unknown_check_id
manifest_check_results_invalid
```

Exclude-mode entries additionally require `outcome == "excluded"`,
no `exit_code`, zero `duration_ms`,
`execution_classification == "excluded_by_plan"`, and
`cleanup_status == "not_required"`.

## Phase 6 — run-success integrity

`verifyV2ManifestRunSuccessIntegrity` enforces the aggregate
success invariant:

- overall run succeeded (no run-mode check failed);
- cleanup succeeded (no run-mode check has
  `cleanup_status == "failed"`);
- no timeout / cancellation / output overflow / output
  truncated / output incomplete classified as success;
- exclude semantics preserved (exclude-mode checks do not
  contribute to run success; only run-mode checks count).

Stable codes:

```text
manifest_unsuccessful_run
manifest_check_results_invalid
```

## Phase 7 — mixed-mode mutation matrix

The hermetic test rig constructs a three-check plan
(`run-1`, `exclude-1`, `run-2`) and mutates one field at a
time to exercise every rejection path:

| Mutation                         | Stable code                            |
|----------------------------------|----------------------------------------|
| missing run-1                    | `manifest_check_result_bijection_failed` |
| duplicate run-1                  | `manifest_check_result_bijection_failed` |
| unknown result id                | `manifest_unknown_check_id`            |
| run mode changed to exclude      | `manifest_check_result_bijection_failed` |
| exclude mode changed to run      | `manifest_check_result_bijection_failed` |
| aggregate success with failed    | `manifest_unsuccessful_run`            |
| cleanup failure hidden by success| `manifest_unsuccessful_run`            |
| timeout marked success           | `manifest_unsuccessful_run`            |
| overflow marked success          | `manifest_unsuccessful_run`            |

## Phase 8 — deterministic verification result

`V2ClosureVerification` is the deterministic public result.
The struct marshals deterministically via explicit field
order, and the `Diagnostics` slice is sorted by
`(Code, PropertyName, Message)` so re-runs are byte-identical.

Canonical validity:

```text
Valid == topologyValid
      AND manifestValid
      AND resultSetValid
      AND diagnostics empty
      AND required identity fields present
```

## Publication

The ACT ships exactly one subject commit:

```text
factory: verify v2 closure manifests and results
```

The verifier is exposed only via internal packages; ACT 4
(CLI-TAG-STATE01) and ACT 5 (MAC-HANDOFF01) add the public
command and the Mac handoff on top of this foundation.

## Acceptance

Closed only when:

1. exact `C:M` bytes are parsed via `encoding/json` with
   `UseNumber` and `DisallowUnknownFields`;
2. `closure_protocol_version = 2` and
   `plan_contract_version = 1` are bound;
3. `S` / `F` / `S^{tree}` / `F^{tree}` / `execution_tree` are
   each independently compared against the externally
   supplied topology anchor;
4. `plan_blob`, `plan_path`, `plan_sha256` are each
   independently compared against the frozen-plan authority;
5. binary identity is structurally valid as a committed
   assertion (no historical-binary rehash);
6. frozen-plan checks and manifest results form a strict
   bijection;
7. failed / cleanup-failed / timeout / cancellation /
   overflow checks never hide under aggregate success;
8. the deterministic `V2ClosureVerification` is byte-stable
   across re-runs;
9. v1 verifier and topology ACT behaviour is unchanged;
10. no ClineMM files change.

## Expected blockers

```text
public CLI/tag/read-only behavior          (ACT_4)
exact-tip installed dogfood and Mac handoff (ACT_5)
```
