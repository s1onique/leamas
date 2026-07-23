# Closure Protocol v1

Closure Protocol v1 is Leamas's first-class protocol for closing a verified
ACT. It replaces the recursive "write the close report, then re-verify, then
edit, then re-tag" loop with a deterministic, machine-checked contract.

## Why the legacy protocol loops

The legacy report-and-correction loop breaks because five objects are
authored independently:

* the closure plan,
* the subject commit and tree,
* the manifest,
* the close report,
* the closure tag.

Each correction opens a new identity that the next correction must
reference. As the report is amended the tag is re-anchored, and as the
tag is re-anchored the report is re-amended. Closure Protocol v1
collapses the loop by making a single compact manifest the only
authoritative record; the report is rendered from the manifest, the
tag is verified against the manifest, and a correction is a new
immutable tag with the same authoritative manifest.

## Subject, plan, manifest, report, tag

The protocol separates the five roles below.

| Role | Mutable? | File |
|------|----------|------|
| Subject | Immutable | `HEAD^{commit}` of the frozen subject branch |
| Plan | Frozen before subject | `docs/closure-plans/<ACT-ID>.json` |
| Manifest | Compact, authoritative | `docs/closure-manifests/<ACT-ID>.json` |
| Report | Deterministic projection of manifest | `docs/close-reports/<ACT-ID>.md` |
| Tag | Immutable, annotated | `act/<normalized-act-id>` |

The manifest binds the subject commit, the subject tree, the runner
identity, the committed plan, the committed artifacts, the detached
check output, and the mechanically derived verdict. The Markdown
report is a deterministic projection and is not authoritative.

## Lifecycle states

The protocol reports the following states, derived from Git and the
manifest:

* `IMPLEMENTED` — a subject commit exists.
* `VERIFIED` — the manifest is `pass`.
* `CLOSED_LOCAL` — the immutable annotated tag points at the closure
  commit.
* `PUBLISHED` — the configured remote advertises the same tag object
  and the closure commit is reachable from the remote branch.
* `DOWNSTREAM_ACCEPTED` — out of scope for v1.

`leamas factory close status` derives the local and remote states from
Git and the manifest. Web UI inspection is never used as evidence.

## Plan contract

The plan is a strict, bounded JSON document with `contract_version = 1`.
The decoder rejects unknown fields, duplicate JSON keys, trailing
JSON, unsupported contract versions, empty act IDs, duplicate check
IDs, duplicate artifact IDs, shell command strings, escaping
working directories, missing exclusion reasons, and closure
placeholders. The full shape is in the dedicated plan contract and
is exercised by the focused unit tests.

## Manifest contract

The manifest is the authoritative verification record. It records:

* the bound subject commit and tree,
* the runner identity (Leamas version, binary SHA-256, VCS revision,
  VCS modified flag),
* the repository state (branch, remote URL, head identity, worktree
  cleanliness before and after),
* the executed checks, in plan order, with stdout/stderr hashes and
  byte counts,
* the artifact hashes for committed artifacts,
* the detached evidence record list,
* the patch-hygiene status,
* the closure-policy status for tracked full digests,
* the excluded checks with reasons.

The manifest never embeds raw logs, absolute host paths, secret
environment values, or future closure/tag identities. The verdict is
derived mechanically from the manifest contents.

## Detached-evidence policy

Raw command output, full targeted digests, traces, profiles, and
performance logs are **detached**. The committed manifest references
their SHA-256, byte count, and media type but does not embed the
payload. The CLI rejects an evidence directory that resolves inside
the Git worktree.

Tracked full digests that begin with
`LEAMAS_TARGETED_DIGEST_CONTRACT_VERSION:` are forbidden in the new
subject range unless they are grandfathered legacy digests that are
unchanged. Closure plans, manifests, and close reports may not embed a
full targeted digest body; they may include a compact digest hash and
metadata summary only.

## Tag protocol

`leamas factory close tag create` produces a new annotated tag after
verifying that:

* the working tree is clean,
* the target is `HEAD`,
* the closure commit contains the exact manifest and the exact report,
* the report is the deterministic rendering of the manifest,
* the manifest verdict is `pass`,
* the tag name does not already exist,
* the subject is an ancestor of the closure commit,
* the tag name matches the frozen regular expression.

The tag message is bound byte-for-byte and never contains the
annotated-tag object OID. No `--force` option exists. A correction
creates a new tag (e.g.
`act/leamas-factory-closure-protocol-v1-01-correction01`).

## Remote verification

`leamas factory close status --remote origin` uses `git ls-remote` to
verify that the remote branch contains the closure commit, that the
remote tag exists, that the advertised tag object matches the local
tag object, and that the peeled remote target equals the local
closure commit. Web UI inspection is not used as evidence; only
Git-ref output from the configured remote is.

## Correction policy

A report-only correction may:

* regenerate the report from the same manifest,
* fix board or documentation references,
* create a new closure commit,
* create a new correction tag.

It must not change the subject, the check outcomes, the artifact
hashes, the tested tree, or move an earlier tag. A report-only
correction requires only `close verify`, render equality,
`git diff --check`, and closure-policy verification; it does not
require `make factorize`, `gate-dupcode`, full focused tests, or
race tests. If the manifest or subject changes, verification must be
re-run.

## Security and secret-handling limits

The manifest does not record secret environment values. The
runner-record of the environment contains only the names of the
overrides, not the values. The protocol scrubs absolute host paths
from artifact diagnostics, applies the same `pathInside` test used
for evidence directories, and refuses to publish a manifest with
user info embedded in the remote URL.

## Serial fail-fast execution

The protocol runs checks in plan order, serially, and stops at the
first required failure. Remaining checks are recorded as
`not_run_due_to_prior_failure`. The execution path reuses the existing
bounded execution gateway (`internal/execution`) and never bypasses
its timeout, process-group cleanup, retained-pipe handling,
truncation, incompleteness, or stdout/stderr separation.

## Examples

The plan directory layout, manifest, report, and tag are produced by
the CLI:

```bash
leamas factory close plan validate \
  --file docs/closure-plans/ACT-LEAMAS-FACTORY-CLOSURE-PROTOCOL-V1-01.json

leamas factory close run \
  --plan docs/closure-plans/ACT-LEAMAS-FACTORY-CLOSURE-PROTOCOL-V1-01.json \
  --subject $SUBJECT \
  --evidence-dir /tmp/leamas-closure/ACT-LEAMAS-FACTORY-CLOSURE-PROTOCOL-V1-01 \
  --manifest-out /tmp/ACT-LEAMAS-FACTORY-CLOSURE-PROTOCOL-V1-01.json

cp /tmp/ACT-LEAMAS-FACTORY-CLOSURE-PROTOCOL-V1-01.json \
   docs/closure-manifests/ACT-LEAMAS-FACTORY-CLOSURE-PROTOCOL-V1-01.json

leamas factory close render \
  --manifest docs/closure-manifests/ACT-LEAMAS-FACTORY-CLOSURE-PROTOCOL-V1-01.json \
  --output docs/close-reports/ACT-LEAMAS-FACTORY-CLOSURE-PROTOCOL-V1-01.md

git add docs/closure-manifests docs/close-reports
git commit -m "close act"

leamas factory close tag create \
  --manifest docs/closure-manifests/ACT-LEAMAS-FACTORY-CLOSURE-PROTOCOL-V1-01.json \
  --report docs/close-reports/ACT-LEAMAS-FACTORY-CLOSURE-PROTOCOL-V1-01.md \
  --tag act/leamas-factory-closure-protocol-v1-01 \
  --target HEAD

leamas factory close status \
  --manifest docs/closure-manifests/ACT-LEAMAS-FACTORY-CLOSURE-PROTOCOL-V1-01.json \
  --report docs/close-reports/ACT-LEAMAS-FACTORY-CLOSURE-PROTOCOL-V1-01.md \
  --tag act/leamas-factory-closure-protocol-v1-01
```

## Migration guidance

Legacy close reports that embed raw digests, host paths, or
self-referential identities are not migrated. Future ACTs MUST use
the v1 protocol. Historical ACT tags are not moved. Historical close
reports that are not v1 are tolerated as legacy evidence but must
not embed full digests in new subject ranges; grandfathered legacy
digests that are unchanged in the new subject range are tolerated.

## Non-goals

Closure Protocol v1 explicitly does not implement:

* cryptographic signatures,
* SLSA or in-toto compliance claims,
* transparency logs or Rekor,
* Git notes storage,
* remote evidence storage,
* automatic GitHub Release creation,
* automatic force pushes,
* automatic retries,
* parallel check execution,
* cross-repository adoption,
* deletion or rewriting of historical tags,
* Gate Summary v3,
* Targeted Digest v3,
* ClineMM `DOGFOOD01`,
* Leamas `v0.2.0` publication.

Those are separate successor scopes.
