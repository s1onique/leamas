# P0 Recon — ACT-LEAMAS-DIGEST-SUBJECT-FINGERPRINT-WORKTREE-CANONICAL01

```text
ACT_ID=ACT-LEAMAS-DIGEST-SUBJECT-FINGERPRINT-WORKTREE-CANONICAL01
PHASE=P0_RECON
STATUS=COMPLETE
```

This recon is the **only blocking input** for the implementation ACT.
The recon is read-only by construction. Pre/post invariants were
captured and compared.

---

## 1. Pre-recon invariant snapshot

```text
PRE_REAL_INDEX_SHA256=3347e0c0399f52949717e00416e8239dab7a84157b9dfcb5041b83d4a2048741
PRE_LOOSE_OBJECTS=3127
PRE_REF_LOCKS=0
```

## 2. Post-recon invariant proof

```text
POST_REAL_INDEX_SHA256=3347e0c0399f52949717e00416e8239dab7a84157b9dfcb5041b83d4a2048741
POST_LOOSE_OBJECTS=3127
POST_REF_LOCKS=0
HEAD=dc500b60a27181c71939305608da7eb17ebcbc46
WORKTREE_CLEAN=true
```

```text
INVARIANT_PROOF:
  REAL_INDEX_MUTATION_COUNT=0
  REAL_OBJECT_DB_MUTATION_COUNT=0
  REF_MUTATION_COUNT=0
  WORKTREE_MUTATION_COUNT=0
```

---

## 3. Identity

```text
BRANCH=main
HEAD=dc500b60a27181c71939305608da7eb17ebcbc46
HEAD_TREE=7251b0a65e92000df9d4c24404379ea1f758894c
ORIGIN_MAIN=dc500b60a27181c71939305608da7eb17ebcbc46
WORKTREE_STATUS=clean
```

(Host identifier under REPOSITORY_ROOT is intentionally omitted;
see AGENTS.md rule on not embedding absolute host paths in committed
documents.)

---

## 4. Git version + object format

```text
GIT_VERSION=git version 2.51.2
GIT_OBJECT_FORMAT=sha1
```

**Decision:** `GIT_OBJECT_FORMAT=sha1` — v1 protocol constraint
satisfied. sha256 is a v2 protocol concern.

---

## 5. Real object-DB layout

```text
COMMON_DIR=.git
REAL_OBJECT_DIRECTORY=.git/objects
EXISTING_ALTERNATE_OBJECT_DIRECTORIES=none
```

**Decision:** No existing alternates. The fingerprint collector must
set `GIT_ALTERNATE_OBJECT_DIRECTORIES=.git/objects` and
`GIT_OBJECT_DIRECTORY=<temp>` so new blobs/trees land in the temp store.

---

## 6. Submodules (gitlinks)

```text
HAS_GITLINKS=false
```

**Decision:** No gitlinks in this repo. Runtime
`SUBMODULE_POLICY_V1=FAIL_CLOSED_ON_ANY_GITLINK` remains for forward
compatibility; not exercised here.

---

## 7. Sparse checkout

```text
SPARSE_CHECKOUT=false
SPARSE_CHECKOUT_FILE=absent
```

**Decision:** Not sparse. Runtime fallback `UNAVAILABLE` available if
implementation cannot prove exact semantics; not exercised here.

---

## 8. Ignore sources

```text
CORE_EXCLUDES_FILE=unset
INFO_EXCLUDE_PRESENT=true
GITIGNORE_PRESENT=true
```

**Decision:**

- Neutralize `core.excludesFile` via `-c core.excludesFile=/dev/null`
  for every collector invocation (overrides any future user-global
  configuration).
- Keep `.gitignore` and `.git/info/exclude` active — they are
  worktree-local sources.
- **Adopt the corrected scope name:** `git_worktree_local_nonignored_v1`
  (replaces `git_worktree_nonignored_v1`).
- Document `FINGERPRINT_IS_CROSS_CLONE_REPRODUCIBLE=false` for v1.


---

## 9. Filter drivers + content-conversion attributes

```text
FILTER_DRIVERS_CONFIGURED=none
```

**All participating tracked-path attribute scan** (no sampling):

```text
git ls-files -z |
  git check-attr -z --stdin \
    filter text eol working-tree-encoding
```

Result:

```text
ATTRS_RECORDS=7224
ATTRS_NON_TRIVIAL=0
```

Zero participating paths carry an effective `filter`, `text`, `eol`,
or `working-tree-encoding` value that would alter Git-canonical blob
content versus literal filesystem bytes.

**Decision:** No runtime `CONTENT_CONVERSION` fail-closed trigger in
this repo today. Implementation MUST still implement the check
(reason `CONTENT_CONVERSION_UNSUPPORTED_V1`) so it fires correctly
elsewhere.

---

## 10. Build path

```text
VERSION_METADATA_PACKAGE=internal/version
VERSION_LDFLAGS_PATH=Makefile:52:LDFLAGS := -X '$(MODULE_PATH)/internal/version.Version=$(VERSION)' ...
CANONICAL_BUILD_TARGET=Makefile:188:build:
```

**Decision:** `make build` (or `make leamas`) is the canonical
fingerprint-aware build path. Raw `go build ./cmd/leamas` remains
supported but yields `UNAVAILABLE` for dirty authority.

The fingerprint vars live in `internal/version` (additive over the
existing `Version`, `DeclaredVersion`, `Commit`, `BuildTime`, `Dirty`).
The Makefile MUST be extended with an **imperative** single-recipe
guard:

```
F1 = fingerprint()
build(binary, embedded=F1)
F2 = fingerprint()
if F1 != F2: refuse qualified output
```

Implementation in a single shell recipe body or a small Go build
wrapper — **not** via `$(shell ...)` variable assignments, which
evaluate at Make's parsing time and cannot enforce build-time
ordering.

---

## 11. Digest atomicity

```text
DIGEST_USES_STRINGS_BUILDER=internal/factory/digest/digest.go:1 (yes)
DIGEST_OUTPUT_VIA_GENERATE=internal/factory/digest/digest.go:34:func Generate(opts Options) (string, error)
```

**Decision:** Digest output is already buffered through
`strings.Builder` and returned as a single `string` from `Generate()`.
The CLI then writes atomically. The F2-before-publish race guard
inserts cleanly between evidence collection and the final
`return sb.String(), nil`. No renderer streams to the final
destination mid-pipeline.

---

## 12. Existing temp-index precedent

```text
PRECEDENT_FILE=internal/factory/closure/run_v2_objects.go
PRECEDENT_USES_GIT_INDEX_FILE=1
PRECEDENT_USES_GIT_OBJECT_DIRECTORY=0
PRECEDENT_USES_GIT_ALTERNATE_OBJECT_DIRECTORIES=0
SECOND_PRECEDENT=internal/factory/closure/simple_entrypoint.go:633
```

**Decision:** The precedent uses `GIT_INDEX_FILE` only. The
fingerprint collector MUST additionally set `GIT_OBJECT_DIRECTORY`
and `GIT_ALTERNATE_OBJECT_DIRECTORIES` because it invokes
`git add -A -- .`, which hashes worktree bytes. The closure runner
uses `update-index --add --cacheinfo` and does not hash worktree
content, so it does not need those vars. **The fingerprint collector
is a new helper; it does NOT modify the closure runner.**

---

## 13. Stop-condition evaluation

```text
GIT_OBJECT_FORMAT_IS_SHA1=PASS
WORKTREE_CLEAN=PASS
HEAD_EXISTS=PASS
NO_GITLINKS=PASS
NO_SPARSE_CHECKOUT=PASS
NO_FILTER_DRIVERS_CONFIGURED=PASS
NO_CONTENT_CONVERSION_ATTRIBUTES_IN_USE=PASS
CANONICAL_BUILD_PATH=PASS (make build + internal/version)
PRECEDENT_FILE_CONFIRMED=PASS
PRECEDENT_USES_GIT_INDEX_FILE=PASS
PRECEDENT_NEEDS_GIT_OBJECT_DIRECTORY_ENV=PASS (extension required)
DIGEST_OUTPUT_ATOMIC=PASS (strings.Builder)
REAL_OBJECT_DIR_RESOLVABLE=PASS

ALL_FREEZE_BLOCKERS=PASS
```

---

## 14. Reconclusions (machine-readable)

```text
RECON_RESULTS = {
    GIT_OBJECT_FORMAT:                  sha1
    WORKTREE_STATE:                     clean
    HEAD:                               dc500b60a27181c71939305608da7eb17ebcbc46
    HEAD_TREE:                          7251b0a65e92000df9d4c24404379ea1f758894c
    HAS_GITLINKS:                       false
    SPARSE_CHECKOUT_ENABLED:            false
    FILTER_DRIVERS_CONFIGURED:          none
    FILTER_ATTRIBUTES_NON_TRIVIAL:      0
    CORE_EXCLUDES_FILE:                 unset (override via -c core.excludesFile=/dev/null)
    INFO_EXCLUDE_PRESENT:               true (in-scope per worktree_local naming)
    GITIGNORE_PRESENT:                  true
    CANONICAL_BUILD_COMMAND:            make build  (Makefile:188)
    VERSION_METADATA_PACKAGE:           internal/version
    VERSION_LDFLAGS_PATH:               Makefile:52
    DIGEST_ATOMIC:                      true (strings.Builder)
    TEMP_INDEX_PRECEDENT:               internal/factory/closure/run_v2_objects.go
    PRECEDENT_EXTENSIONS_NEEDED:        [GIT_OBJECT_DIRECTORY, GIT_ALTERNATE_OBJECT_DIRECTORIES]
    REAL_OBJECT_DIRECTORY:              .git/objects
    EXISTING_ALTERNATE_OBJECT_DIRECTORIES: none
    EXISTING_REF_LOCKS:                 0
    EXISTING_LOOSE_OBJECTS:             3127
    EXISTING_INDEX_SHA256:              3347e0c0399f52949717e00416e8239dab7a84157b9dfcb5041b83d4a2048741
}

RECON_MUTATIONS = {
    REAL_INDEX:                         UNCHANGED
    REAL_OBJECT_DB:                     UNCHANGED
    REFS:                               UNCHANGED
    WORKTREE:                           UNCHANGED
    REAL_LOOSE_OBJECTS_DELTA:           0
}
```

---

## 15. Protocol decisions resolved by recon

The recon resolves every policy choice the original 100-section ACT
left open. The implementation ACT can therefore be substantially
shorter.

```text
FINGERPRINT_PROTOCOL_VERSION          = 1
FINGERPRINT_KIND                      = git-tree
FINGERPRINT_SCOPE                     = git_worktree_local_nonignored_v1
FINGERPRINT_OBJECT_FORMAT             = sha1
FINGERPRINT_OID_LENGTH                = 40

IGNORE_POLICY_VERSION                 = 1
IGNORE_NEUTRALIZED                    = core.excludesFile (-c core.excludesFile=/dev/null)
IGNORE_INCLUDED                       = .gitignore, .git/info/exclude
IGNORE_EXCLUDED                       = $XDG_CONFIG_HOME/git/ignore, user-global

FILTER_POLICY_VERSION                 = 1
FILTER_FAIL_CLOSED_ON                 = effective filter=<driver>
CONTENT_CONVERSION_FAIL_CLOSED_ON     = effective text/eol/working-tree-encoding

SUBMODULE_POLICY_V1                   = FAIL_CLOSED_ON_ANY_GITLINK
SPARSE_POLICY_V1                      = FAIL_CLOSED (UNAVAILABLE)
UNMERGED_POLICY_V1                    = FAIL_CLOSED (UNAVAILABLE)

BUILD_RACE_GUARD                      = required (imperative recipe, not Make variables)
DIGEST_RACE_GUARD                     = required (F2 before publish)
DIGEST_ATOMICITY                      = confirmed (strings.Builder return path)

BUILD_METADATA_HOME                   = internal/version
RAW_GO_BUILD_DIRTY_AUTHORITY          = UNBOUND

BINDING_BRIDGE                        = MATCH->AUTHORITATIVE; MISMATCH->SUBJECT_MISMATCH;
                                          UNBOUND->DIRTY_SUBJECT_UNBOUND; INVALID->EVIDENCE_INVALID
PROTOCOL_MISMATCH                     = IDENTITY_UNBOUND
```

---

## 16. Verdict

```text
P0_RECON=COMPLETE
RECON_INVARIANTS=VERIFIED
FREEZE_BLOCKERS=ALL_PASS
IMPLEMENTATION_ACT=READY_TO_DRAFT

NEXT_STEP:
  Draft the implementation ACT body (~30-40 sections, not 100)
  using the recon-resolved decisions in §15 above.
  Do not reopen decisions that recon has resolved.
  Apply C1, C2, C3 hardening (protocol-version freeze,
  OID-length invariant, recon-validator hook).
```
