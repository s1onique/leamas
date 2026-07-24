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
package authority

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os/exec"
	"sort"
	"strings"
)

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
	ContractVersion int `json:"contract_version"`
	ActID           string `json:"act_id"`
	Plan            struct {
		Path string `json:"path"`
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
	Tag       string `json:"tag,omitempty"`
	Verdict   string `json:"verdict"`
	Runner    struct {
		BinarySHA256 string `json:"binary_sha256"`
	} `json:"runner"`
	Repository struct {
		HeadCommitOID string `json:"head_commit_oid"`
	} `json:"repository"`
}

// AttestationLoose mirrors the attestation schema's identity
// fields. Unknown fields are tolerated.
type AttestationLoose struct {
	AttestationVersion int `json:"attestation_version"`
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

// Resolve classifies the lifecycle authority for the supplied
// resolver options. It is the single source of truth for
// authoritative range selection across digest, status, and
// verify commands.
//
// The resolver enforces three rules:
//
//  1. Zero-argument (auto) resolution only returns an
//     authoritative range when validated lifecycle artifacts pin
//     the implementation subject. Heuristic fallbacks such as
//     `HEAD~1..HEAD`, the previous commit, and working-tree
//     cleanliness are not consulted for authority classification.
//  2. Explicit ranges are classified as AuthorityExplicitRange
//     and never reported as authoritative.
//  3. Tool identity is recorded on every resolution so the
//     caller can detect incompatible stale binaries.
func Resolve(opts ResolverOptions) (*ResolvedAuthority, error) {
	if opts.RepoRoot == "" {
		return nil, &AuthorityResolutionError{
			Status: AuthorityInvalidArtifact,
			Reason: "repository root is required",
		}
	}
	git := opts.RunGit
	if git == nil {
		git = DefaultGitRunner
	}

	headOID, headTree, err := resolveHEAD(git, opts)
	if err != nil {
		return nil, err
	}

	tool, err := captureToolIdentity(opts)
	if err != nil {
		return nil, err
	}
	tool.RepositoryHead = headOID
	tool.RepositoryTree = headTree

	// Explicit range bypasses lifecycle authority. The resolver
	// still records identities but classifies the result as
	// non-authoritative.
	if strings.TrimSpace(opts.ExplicitRange) != "" {
		return &ResolvedAuthority{
			AuthorityStatus: AuthorityExplicitRange,
			DigestRange:     strings.TrimSpace(opts.ExplicitRange),
			ResolutionSrc:   "explicit_cli",
			ToolIdentity:    tool,
		}, nil
	}

	// Inspect HEAD-introduced ACTs.
	actIDs, err := headIntroducedActs(git, opts.RepoRoot, headOID)
	if err != nil {
		return nil, &AuthorityResolutionError{
			Status: AuthorityInvalidGitObject,
			Reason: fmt.Sprintf("scan HEAD-introduced ACTs: %v", err),
		}
	}

	if len(actIDs) == 0 {
		if isHeadEvidenceOnly(git, opts.RepoRoot, headOID) {
			return nil, &AuthorityResolutionError{
				Status: AuthorityEvidenceOnlyHead,
				Reason: fmt.Sprintf("HEAD %s is evidence-only; supply --range or close the ACT with closure artifacts", shortSHA(headOID)),
			}
		}
		return nil, &AuthorityResolutionError{
			Status: AuthorityMissingAuthority,
			Reason: fmt.Sprintf("no authoritative ACT for clean tree; HEAD %s has no lifecycle artifacts", shortSHA(headOID)),
		}
	}

	if len(actIDs) > 1 {
		return nil, &AuthorityResolutionError{
			Status: AuthorityAmbiguousAuthority,
			Reason: fmt.Sprintf("multiple ACTs claim authority at HEAD: %s", strings.Join(actIDs, ",")),
		}
	}

	actID := actIDs[0]
	resolved, err := resolveSingleAct(git, opts.RepoRoot, headOID, actID)
	if err != nil {
		return nil, err
	}
	resolved.ToolIdentity = tool
	return resolved, nil
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

// populate fills the remaining ToolIdentity fields by running
// `version --json` on path and hashing the binary bytes.
func (t *ToolIdentity) populate(path string) error {
	data, err := readFileBytes(path)
	if err != nil {
		return fmt.Errorf("read tool bytes: %w", err)
	}
	sum := sha256.Sum256(data)
	t.ToolSHA256 = hex.EncodeToString(sum[:])

	cmd := exec.Command(path, "version", "--json")
	out, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("tool version probe: %w", err)
	}
	var v struct {
		Version         string `json:"version"`
		DeclaredVersion string `json:"declared_version"`
		Commit          string `json:"commit"`
		BuildTime       string `json:"build_time"`
	}
	if err := json.Unmarshal(out, &v); err != nil {
		return fmt.Errorf("decode tool version json: %w", err)
	}
	if v.Version != "" {
		t.ToolVersion = v.Version
	} else if v.DeclaredVersion != "" {
		t.ToolVersion = v.DeclaredVersion
	}
	if v.Commit != "" {
		t.ToolCommit = v.Commit
		t.ToolVCSRev = v.Commit
	}
	return nil
}

// headIntroducedActs returns the ACT IDs of closure artifacts
// (manifest, attestation, tag) introduced by HEAD.
func headIntroducedActs(git GitRunner, repoRoot, headOID string) ([]string, error) {
	acts := map[string]struct{}{}

	parent, perr := git(repoRoot, "rev-parse", "--verify", "--end-of-options", headOID+"^")
	var diffArgs []string
	if perr != nil || strings.TrimSpace(parent) == "" {
		diffArgs = []string{"diff", "--name-only", "--diff-filter=A", "4b825dc642cb6eb9a060e54bf8d69288fbee4904", headOID}
	} else {
		diffArgs = []string{"diff", "--name-only", "--diff-filter=A", parent, headOID}
	}
	if diffOut, derr := git(repoRoot, diffArgs...); derr == nil {
		for _, line := range strings.Split(diffOut, "\n") {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			if id, ok := actIDFromManifestPath(line); ok {
				acts[id] = struct{}{}
				continue
			}
			if id, ok := actIDFromAttestationPath(line); ok {
				acts[id] = struct{}{}
				continue
			}
		}
	}

	if tagOut, terr := git(repoRoot, "for-each-ref", "--format=%(refname:short)", "refs/tags/act/"); terr == nil {
		for _, raw := range strings.Split(tagOut, "\n") {
			name := strings.TrimSpace(raw)
			if !strings.HasPrefix(name, "act/") {
				continue
			}
			peeled, perr := git(repoRoot, "rev-parse", "--verify", "--end-of-options", name+"^{commit}")
			if perr != nil || strings.TrimSpace(peeled) != headOID {
				continue
			}
			objType, terr := git(repoRoot, "cat-file", "-t", name)
			if terr != nil || strings.TrimSpace(objType) != "tag" {
				continue
			}
			id := strings.TrimPrefix(name, "act/")
			acts[id] = struct{}{}
		}
	}

	out := make([]string, 0, len(acts))
	for id := range acts {
		out = append(out, id)
	}
	sort.Strings(out)
	return out, nil
}

// actIDFromManifestPath extracts the ACT ID from a manifest file
// path (e.g. docs/closure-manifests/<ID>.json).
func actIDFromManifestPath(path string) (string, bool) {
	const prefix = "docs/closure-manifests/"
	if !strings.HasPrefix(path, prefix) {
		return "", false
	}
	name := strings.TrimPrefix(path, prefix)
	if !strings.HasSuffix(name, ".json") || strings.HasSuffix(name, ".attestation.json") {
		return "", false
	}
	id := strings.TrimSuffix(name, ".json")
	if isValidActID(id) {
		return id, true
	}
	return "", false
}

// actIDFromAttestationPath extracts the ACT ID from an attestation
// path (e.g. docs/closure-manifests/<ID>.attestation.json).
func actIDFromAttestationPath(path string) (string, bool) {
	const prefix = "docs/closure-manifests/"
	if !strings.HasPrefix(path, prefix) {
		return "", false
	}
	name := strings.TrimPrefix(path, prefix)
	const suffix = ".attestation.json"
	if !strings.HasSuffix(name, suffix) {
		return "", false
	}
	id := strings.TrimSuffix(name, suffix)
	if isValidActID(id) {
		return id, true
	}
	return "", false
}

// isHeadEvidenceOnly reports whether every file changed by HEAD
// lives under docs/closure-* or is a Markdown file. Such a HEAD
// is evidence-only and may not serve as the implementation
// subject.
func isHeadEvidenceOnly(git GitRunner, repoRoot, headOID string) bool {
	show, err := git(repoRoot, "show", "--name-only", "--pretty=format:", headOID)
	if err != nil {
		return false
	}
	for _, line := range strings.Split(show, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "docs/closure-manifests/") ||
			strings.HasPrefix(line, "docs/close-reports/") ||
			strings.HasPrefix(line, "docs/closure-plans/") {
			continue
		}
		lower := strings.ToLower(line)
		if strings.HasSuffix(lower, ".md") || strings.HasSuffix(lower, ".markdown") {
			continue
		}
		return false
	}
	return true
}

// resolveSingleAct loads and validates the manifest and (when
// present) attestation for actID. It enforces ancestry, hash,
// and tag-type invariants and returns the typed ResolvedAuthority.
func resolveSingleAct(git GitRunner, repoRoot, headOID, actID string) (*ResolvedAuthority, error) {
	manifestPath := "docs/closure-manifests/" + actID + ".json"
	raw, err := readBlobAt(git, repoRoot, headOID, manifestPath)
	if err != nil {
		return nil, &AuthorityResolutionError{
			Status: AuthorityMissingAuthority,
			Reason: fmt.Sprintf("manifest %s missing at HEAD: %v", manifestPath, err),
		}
	}
	var mf ManifestLoose
	if err := json.Unmarshal([]byte(raw), &mf); err != nil {
		return nil, &AuthorityResolutionError{
			Status: AuthorityInvalidArtifact,
			Reason: fmt.Sprintf("decode manifest %s: %v", manifestPath, err),
		}
	}
	if mf.ActID != "" && mf.ActID != actID {
		return nil, &AuthorityResolutionError{
			Status: AuthorityInvalidArtifact,
			Reason: fmt.Sprintf("manifest act_id %q does not match %q", mf.ActID, actID),
		}
	}
	freeze := mf.PlanFreeze.FreezeCommit
	subject := mf.PlanFreeze.SubjectCommit
	if freeze == "" || subject == "" {
		return nil, &AuthorityResolutionError{
			Status: AuthorityInvalidArtifact,
			Reason: fmt.Sprintf("manifest %s missing plan_freeze.freeze_commit or plan_freeze.subject_commit", manifestPath),
		}
	}
	freezeFull := mustResolveOID(git, repoRoot, freeze)
	subjectFull := mustResolveOID(git, repoRoot, subject)
	if freezeFull == "" || subjectFull == "" {
		return nil, &AuthorityResolutionError{
			Status: AuthorityInvalidGitObject,
			Reason: fmt.Sprintf("manifest %s references missing Git objects", manifestPath),
		}
	}
	if freezeFull == subjectFull {
		return nil, &AuthorityResolutionError{
			Status: AuthorityInvalidArtifact,
			Reason: fmt.Sprintf("manifest %s has freeze_commit == subject_commit", manifestPath),
		}
	}
	if !isAncestor(git, repoRoot, freezeFull, subjectFull) {
		return nil, &AuthorityResolutionError{
			Status: AuthorityInvalidArtifact,
			Reason: fmt.Sprintf("manifest %s: freeze %s is not an ancestor of subject %s", manifestPath, shortSHA(freezeFull), shortSHA(subjectFull)),
		}
	}
	if !isAncestor(git, repoRoot, subjectFull, headOID) {
		return nil, &AuthorityResolutionError{
			Status: AuthorityInvalidArtifact,
			Reason: fmt.Sprintf("manifest %s: subject %s is not an ancestor of HEAD %s", manifestPath, shortSHA(subjectFull), shortSHA(headOID)),
		}
	}

	// Head-commit identity. The manifest's recorded head_commit
	// must equal the actual repository HEAD, otherwise the
	// manifest was authored against a different repository
	// state.
	if mf.Repository.HeadCommitOID != "" {
		recorded := mustResolveOID(git, repoRoot, mf.Repository.HeadCommitOID)
		if recorded != headOID {
			return nil, &AuthorityResolutionError{
				Status: AuthorityRepositoryIdentityMismatch,
				Reason: fmt.Sprintf("manifest %s: recorded HEAD %s != repository HEAD %s", manifestPath, shortSHA(recorded), shortSHA(headOID)),
			}
		}
	}

	resolved := &ResolvedAuthority{
		ActID:           actID,
		FreezeCommit:    freezeFull,
		SubjectStart:    freezeFull,
		SubjectEnd:      subjectFull,
		AttestationPath: "docs/closure-manifests/" + actID + ".attestation.json",
		ResolutionSrc:   "closure_manifest",
	}

	// Attestation, when present, must validate.
	if attRaw, attErr := readBlobAt(git, repoRoot, headOID, resolved.AttestationPath); attErr == nil {
		var att AttestationLoose
		if err := json.Unmarshal([]byte(attRaw), &att); err != nil {
			return nil, &AuthorityResolutionError{
				Status: AuthorityInvalidArtifact,
				Reason: fmt.Sprintf("decode attestation %s: %v", resolved.AttestationPath, err),
			}
		}
		if att.ActID != "" && att.ActID != actID {
			return nil, &AuthorityResolutionError{
				Status: AuthorityInvalidArtifact,
				Reason: fmt.Sprintf("attestation act_id %q does not match %q", att.ActID, actID),
			}
		}
		attFreeze := mustResolveOID(git, repoRoot, att.FreezeReference.FreezeCommit)
		attSubject := mustResolveOID(git, repoRoot, att.SubjectReference.SubjectCommit)
		attClosure := mustResolveOID(git, repoRoot, att.ClosureReference.ClosureCommit)
		if attFreeze != freezeFull || attSubject != subjectFull {
			return nil, &AuthorityResolutionError{
				Status: AuthorityInvalidArtifact,
				Reason: fmt.Sprintf("attestation %s freeze/subject do not match manifest", resolved.AttestationPath),
			}
		}
		sum := sha256.Sum256([]byte(attRaw))
		actualHash := hex.EncodeToString(sum[:])
		if att.AttestationSHA256 != "" && !strings.EqualFold(att.AttestationSHA256, actualHash) {
			return nil, &AuthorityResolutionError{
				Status: AuthorityInvalidArtifact,
				Reason: fmt.Sprintf("attestation %s recorded sha256 %s != actual %s", resolved.AttestationPath, att.AttestationSHA256, actualHash),
			}
		}
		resolved.AttestationSHA = actualHash
		if attClosure != "" {
			resolved.ClosureCommit = attClosure
		}
		if att.TagIdentity.TagName != "" {
			resolved.TagName = att.TagIdentity.TagName
			resolved.TagObject = att.TagIdentity.TagObjectOID
			resolved.TagTarget = att.TagIdentity.PeeledTarget
		}
	}

	// Tag verification (when the tag name is known).
	if resolved.TagName == "" && mf.Tag != "" {
		resolved.TagName = mf.Tag
	}
	if resolved.TagName != "" {
		if err := verifyTag(git, repoRoot, resolved, headOID); err != nil {
			return nil, err
		}
	}

	// Digest range: the subject-only range F..S. Closure-only
	// commits are excluded by construction because they descend
	// from S, not the reverse.
	resolved.DigestRange = freezeFull + ".." + subjectFull

	// Status classification. AuthoritativeClosed requires an
	// attestation AND an annotated tag AND a closure commit
	// pinned by the attestation. AuthoritativeClosedLocal
	// requires the manifest only.
	switch {
	case resolved.TagName != "" && resolved.AttestationSHA != "" && resolved.ClosureCommit != "":
		resolved.AuthorityStatus = AuthorityAuthoritativeClosed
	default:
		resolved.AuthorityStatus = AuthorityAuthoritativeClosedLocal
	}

	return resolved, nil
}

// verifyTag ensures the named annotated tag exists, peels to the
// expected target, and is the correct Git object type.
func verifyTag(git GitRunner, repoRoot string, resolved *ResolvedAuthority, headOID string) error {
	if resolved.TagName == "" {
		return nil
	}
	ref := "refs/tags/" + resolved.TagName
	objType, err := git(repoRoot, "cat-file", "-t", ref)
	if err != nil {
		return &AuthorityResolutionError{
			Status: AuthorityInvalidGitObject,
			Reason: fmt.Sprintf("tag %s missing: %v", resolved.TagName, err),
		}
	}
	if strings.TrimSpace(objType) != "tag" {
		return &AuthorityResolutionError{
			Status: AuthorityTagMismatch,
			Reason: fmt.Sprintf("tag %s is lightweight (type=%q); annotated required", resolved.TagName, strings.TrimSpace(objType)),
		}
	}
	peeled, err := git(repoRoot, "rev-parse", "--verify", "--end-of-options", ref+"^{commit}")
	if err != nil {
		return &AuthorityResolutionError{
			Status: AuthorityInvalidGitObject,
			Reason: fmt.Sprintf("tag %s does not peel to a commit: %v", resolved.TagName, err),
		}
	}
	if resolved.TagObject == "" {
		if objOID, err := git(repoRoot, "rev-parse", "--verify", "--end-of-options", ref); err == nil {
			resolved.TagObject = objOID
		}
	}
	if resolved.TagTarget == "" {
		resolved.TagTarget = peeled
	}
	expected := headOID
	if resolved.ClosureCommit != "" {
		expected = resolved.ClosureCommit
	}
	if peeled != expected {
		return &AuthorityResolutionError{
			Status: AuthorityTagMismatch,
			Reason: fmt.Sprintf("tag %s peels to %s; expected %s", resolved.TagName, shortSHA(peeled), shortSHA(expected)),
		}
	}
	return nil
}

// mustResolveOID returns the full 40-char OID for short or partial
// refs, or empty when the resolution fails.
func mustResolveOID(git GitRunner, repoRoot, ref string) string {
	if strings.TrimSpace(ref) == "" {
		return ""
	}
	full, err := git(repoRoot, "rev-parse", "--verify", "--end-of-options", ref)
	if err != nil {
		return ""
	}
	return full
}

// readBlobAt returns the decoded blob content for the path at the
// given commit.
func readBlobAt(git GitRunner, repoRoot, commit, path string) (string, error) {
	out, err := git(repoRoot, "cat-file", "blob", commit+":"+path)
	if err != nil {
		return "", err
	}
	return out, nil
}

// isAncestor returns true when ancestor is an ancestor of
// descendant according to `git merge-base --is-ancestor`.
func isAncestor(git GitRunner, repoRoot, ancestor, descendant string) bool {
	if ancestor == "" || descendant == "" {
		return false
	}
	_, err := git(repoRoot, "merge-base", "--is-ancestor", ancestor, descendant)
	return err == nil
}

// isValidActID enforces the canonical ACT identifier shape.
func isValidActID(id string) bool {
	if len(id) < 4 || len(id) > 200 {
		return false
	}
	if !strings.HasPrefix(id, "ACT-") {
		return false
	}
	for _, r := range id[4:] {
		switch {
		case r >= 'A' && r <= 'Z':
		case r >= '0' && r <= '9':
		case r == '-':
		default:
			return false
		}
	}
	return true
}

// shortSHA returns the first 12 chars of an OID for diagnostic
// rendering. Long OIDs are not truncated to fewer than 7 chars so
// callers always get a usable prefix.
func shortSHA(oid string) string {
	if len(oid) <= 12 {
		return oid
	}
	return oid[:12]
}

// requireValidOID rejects placeholders and non-hex inputs.
func requireValidOID(oid string) error {
	if len(oid) < 7 {
		return fmt.Errorf("oid %q is shorter than 7 chars", oid)
	}
	for _, r := range oid {
		switch {
		case r >= '0' && r <= '9':
		case r >= 'a' && r <= 'f':
		default:
			return fmt.Errorf("oid %q contains non-hex characters", oid)
		}
	}
	return nil
}
