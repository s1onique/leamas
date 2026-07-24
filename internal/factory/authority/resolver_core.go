// SPDX-License-Identifier: Apache-2.0

// Package authority: resolver_core.go declares the typed status
// set, resolver options, the ResolvedAuthority result struct, the
// loose manifest and attestation schemas, the typed error, and
// the main Resolve entry point plus its pre-resolution helpers.
package authority

import (
	"fmt"
	"strings"
)

// SPDX-License-Identifier: Apache-2.0

// Package authority: resolver.go provides the shared authority
// resolver used by `factory digest`, `factory close status`,
// `factory close verify`, and any other consumer that needs to
// classify the lifecycle authority of an ACT range.
//
// Authority classification is fail-closed: zero-argument resolution
// can only return one of the typed statuses below. The resolver
// never infers an implementation range from heuristic fallbacks
// such as `HEAD~1..HEAD`, the previous commit, the most recent
// documentation file, or current working-tree cleanliness.
//
// The resolver is intentionally narrow: it reads only validated
// lifecycle artifacts already committed to the repository
// (manifest, attestation, annotated tag) and never invents missing
// identities. Where the established closure protocol uses
// additional identities, callers can populate them via the typed
// fields on ResolvedAuthority.
//
// Implementation rule: authoritative resolution requires the
// repository HEAD to descend from the freeze commit AND the
// resolved subject to descend from the freeze AND the freeze to
// predate the subject. Any ancestry violation is fatal. Any
// attestation-hash mismatch is fatal. A lightweight tag is fatal
// when an annotated tag is required. A stale executable is fatal
// for automatic authoritative mode but remains usable for
// explicit (non-authoritative) ranges so operators can still
// diagnose drift.
// AuthorityStatus enumerates the typed classifications the resolver
// can return. Callers MUST switch on these values rather than
// parsing error prose.
type AuthorityStatus string

const (
	// AuthorityAuthoritativeClosed indicates the resolver found a
	// fully closed ACT (manifest + attestation + annotated tag)
	// that pins the implementation range to a known F..S span.
	AuthorityAuthoritativeClosed AuthorityStatus = "AuthoritativeClosed"

	// AuthorityAuthoritativeClosedLocal indicates the resolver
	// found a manifest and optionally an attestation but the
	// annotated publication tag is missing; the implementation
	// range is still authoritative against the manifest's plan
	// freeze.
	AuthorityAuthoritativeClosedLocal AuthorityStatus = "AuthoritativeClosedLocal"

	// AuthorityExplicitRange indicates the caller supplied an
	// explicit --range. This is classified as non-authoritative
	// because it bypasses lifecycle artifact validation.
	AuthorityExplicitRange AuthorityStatus = "ExplicitRange"

	// AuthorityDirtyWorktree indicates the working tree has
	// unstaged, staged, or untracked changes. The resolver
	// preserves the documented dirty-mode contract without
	// pretending the dirty tree is an authoritative closure.
	AuthorityDirtyWorktree AuthorityStatus = "DirtyWorktree"

	// AuthorityMissingAuthority indicates no lifecycle artifact
	// identifies the current ACT. The resolver refuses to
	// select an implementation range on this basis.
	AuthorityMissingAuthority AuthorityStatus = "MissingAuthority"

	// AuthorityAmbiguousAuthority indicates more than one ACT
	// claims authority at the current HEAD. Callers MUST NOT
	// silently pick one.
	AuthorityAmbiguousAuthority AuthorityStatus = "AmbiguousAuthority"

	// AuthorityInvalidArtifact indicates a lifecycle artifact
	// exists but failed structural or hash validation.
	AuthorityInvalidArtifact AuthorityStatus = "InvalidArtifact"

	// AuthorityInvalidGitObject indicates a referenced Git
	// object has the wrong type or does not exist.
	AuthorityInvalidGitObject AuthorityStatus = "InvalidGitObject"

	// AuthorityEvidenceOnlyHead indicates HEAD is itself
	// evidence-only (every file touched is documentary) and
	// therefore cannot serve as the implementation subject.
	AuthorityEvidenceOnlyHead AuthorityStatus = "EvidenceOnlyHead"

	// AuthorityToolIdentityMismatch indicates the producing
	// binary's embedded VCS revision is incompatible with the
	// repository state under the defined identity policy.
	AuthorityToolIdentityMismatch AuthorityStatus = "ToolIdentityMismatch"

	// AuthorityTagMismatch indicates the resolved tag's peeled
	// target does not match the closure commit.
	AuthorityTagMismatch AuthorityStatus = "TagMismatch"

	// AuthorityRepositoryIdentityMismatch indicates the
	// repository state (HEAD, branch) disagrees with the
	// manifest's recorded repository identity.
	AuthorityRepositoryIdentityMismatch AuthorityStatus = "RepositoryIdentityMismatch"
)

// ResolverOptions configures the shared authority resolver.
//
// The resolver uses the package-wide GitRunner function type
// declared in checker.go. Tests inject a stub by setting RunGit.
type ResolverOptions struct {
	// RepoRoot is the repository root. Required.
	RepoRoot string

	// HeadOverride, when non-empty, replaces `git rev-parse HEAD`.
	// Used by tests.
	HeadOverride string

	// TreeOverride, when non-empty, replaces `git rev-parse
	// HEAD^{tree}`. Used by tests.
	TreeOverride string

	// ToolBinaryPath, when non-empty, is hashed to derive the
	// tool SHA-256. When empty, the resolver fails closed with
	// AuthorityToolIdentityMismatch.
	ToolBinaryPath string

	// ExplicitRange, when non-empty, marks the resolution as
	// explicit and non-authoritative. The resolver still records
	// the tool identity and repository HEAD, but never searches
	// lifecycle artifacts.
	ExplicitRange string

	// RunGit exposes the Git runner. When nil, DefaultGitRunner
	// from checker.go is used. Tests inject a stub.
	RunGit GitRunner
}

// ResolvedAuthority is the typed output of the resolver. All
// callers MUST consume the AuthorityStatus and never parse the
// Reason string.
type ResolvedAuthority struct {
	ActID           string
	AuthorityStatus AuthorityStatus
	FreezeCommit    string
	SubjectStart    string
	SubjectEnd      string
	ClosureCommit   string
	AttestationPath string
	AttestationSHA  string
	TagName         string
	TagObject       string
	TagTarget       string
	DigestRange     string
	ResolutionSrc   string
	ToolIdentity    ToolIdentity
}

// ToolIdentity captures the exact executable that produced a
// digest. Equality of the binary bytes, declared commit, and VCS
// revision are recorded separately so callers can compare each
// dimension independently.
type ToolIdentity struct {
	ToolPath       string
	ToolSHA256     string
	ToolVersion    string
	ToolCommit     string
	ToolVCSRev     string
	ToolVCSModif   string
	RepositoryHead string
	RepositoryTree string
}

// AuthorityResolutionError carries a typed status plus a
// human-readable reason. Callers should switch on Status.
type AuthorityResolutionError struct {
	Status AuthorityStatus
	Reason string
}

func (e *AuthorityResolutionError) Error() string {
	return fmt.Sprintf("authority resolution: %s: %s", e.Status, e.Reason)
}

// ManifestLoose matches the subset of the closure manifest the
// resolver inspects. Unknown fields are tolerated.
type ManifestLoose struct {
	ContractVersion int    `json:"contract_version"`
	ActID           string `json:"act_id"`
	Plan            struct {
		Path   string `json:"path"`
		SHA256 string `json:"sha256"`
	} `json:"plan"`
	PlanFreeze struct {
		FreezeCommit  string `json:"freeze_commit"`
		PlanPath      string `json:"plan_path"`
		PlanBlobOID   string `json:"plan_blob_oid"`
		PlanSHA256    string `json:"plan_sha256"`
		SubjectCommit string `json:"subject_commit"`
	} `json:"plan_freeze"`
	Subject struct {
		CommitOID string `json:"commit_oid"`
		TreeOID   string `json:"tree_oid"`
	} `json:"subject"`
	Tag     string `json:"tag,omitempty"`
	Verdict string `json:"verdict"`
	Runner  struct {
		BinarySHA256 string `json:"binary_sha256"`
	} `json:"runner"`
	Repository struct {
		HeadCommitOID string `json:"head_commit_oid"`
	} `json:"repository"`
}

// AttestationLoose mirrors the attestation schema's identity
// fields. Unknown fields are tolerated.
type AttestationLoose struct {
	AttestationVersion int    `json:"attestation_version"`
	ActID              string `json:"act_id"`
	FreezeReference    struct {
		FreezeCommit string `json:"freeze_commit"`
	} `json:"freeze_reference"`
	SubjectReference struct {
		SubjectCommit string `json:"subject_commit"`
	} `json:"subject_reference"`
	ClosureReference struct {
		ClosureCommit string `json:"closure_commit"`
	} `json:"closure_reference"`
	TagIdentity struct {
		TagName      string `json:"tag_name"`
		TagObjectOID string `json:"tag_object_oid"`
		TagType      string `json:"tag_type"`
		PeeledTarget string `json:"peeled_target"`
	} `json:"tag_identity"`
	AttestationSHA256 string `json:"attestation_sha256,omitempty"`
}

// resolveHEAD returns the HEAD commit OID and tree OID, honoring
// the test overrides when supplied.
func resolveHEAD(git GitRunner, opts ResolverOptions) (string, string, error) {
	if opts.HeadOverride != "" && opts.TreeOverride != "" {
		if err := requireValidOID(opts.HeadOverride); err != nil {
			return "", "", &AuthorityResolutionError{Status: AuthorityInvalidGitObject, Reason: err.Error()}
		}
		return opts.HeadOverride, opts.TreeOverride, nil
	}
	head, err := git(opts.RepoRoot, "rev-parse", "--verify", "--end-of-options", "HEAD")
	if err != nil {
		return "", "", &AuthorityResolutionError{Status: AuthorityInvalidGitObject, Reason: "resolve HEAD: " + err.Error()}
	}
	tree, err := git(opts.RepoRoot, "rev-parse", "--verify", "--end-of-options", "HEAD^{tree}")
	if err != nil {
		return "", "", &AuthorityResolutionError{Status: AuthorityInvalidGitObject, Reason: "resolve HEAD tree: " + err.Error()}
	}
	return head, tree, nil
}

// captureToolIdentity records the path, SHA-256, declared
// version, and embedded VCS revision of the producing binary. It
// never panics on a missing binary: any error returns an
// AuthorityResolutionError with status AuthorityToolIdentityMismatch.
func captureToolIdentity(opts ResolverOptions) (ToolIdentity, error) {
	identity := ToolIdentity{}
	path := opts.ToolBinaryPath
	if path == "" {
		// No tool path was supplied. This is acceptable
		// for unit tests; production CLI commands MUST
		// populate ResolverOptions.ToolBinaryPath so the
		// digest header carries an executable identity.
		return identity, nil
	}
	identity.ToolPath = path
	if err := identity.populate(path); err != nil {
		return identity, &AuthorityResolutionError{Status: AuthorityToolIdentityMismatch, Reason: err.Error()}
	}
	return identity, nil
}

// filterReentryEnv returns env with every LEAMAS_EXEC_* variable
// stripped. The version probe must run as a fresh root invocation.
func filterReentryEnv(env []string) []string {
	out := make([]string, 0, len(env))
	for _, line := range env {
		if strings.HasPrefix(line, "LEAMAS_EXEC_") {
			continue
		}
		out = append(out, line)
	}
	return out
}

// populate fills the remaining ToolIdentity fields by running
// `version --json` on path and hashing the binary bytes. The
// working directory is set to the calling process's current
// directory so relative paths like "./bin/leamas" resolve.
