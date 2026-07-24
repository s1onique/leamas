// SPDX-License-Identifier: Apache-2.0

// Package authority: resolver_act.go contains the manifest-driven
// resolution path. It owns the populate / headIntroducedActs /
// isHeadEvidenceOnly / resolveSingleAct functions that read
// validated lifecycle artifacts.
package authority

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strings"
)

func (t *ToolIdentity) populate(path string) error {
	data, err := readFileBytes(path)
	if err != nil {
		return fmt.Errorf("read tool bytes: %w", err)
	}
	sum := sha256.Sum256(data)
	t.ToolSHA256 = hex.EncodeToString(sum[:])

	cmd := exec.Command(path, "version", "--json")
	if wd, _ := os.Getwd(); wd != "" {
		cmd.Dir = wd
	}
	// Strip LEAMAS_EXEC_* re-entry fuse variables so the
	// version probe can run as a fresh root invocation.
	cmd.Env = filterReentryEnv(os.Environ())
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
			if perr != nil || strings.TrimSpace(peeled) == "" {
				continue
			}
			if _, err := git(repoRoot, "merge-base", "--is-ancestor", strings.TrimSpace(peeled), headOID); err != nil {
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

	// repository.head_commit_oid is the historical execution snapshot,
	// not a query-time HEAD identity. A valid closure may be followed by
	// descendants, so current HEAD is checked through subject/closure
	// ancestry rather than equality with the recorded snapshot.

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
