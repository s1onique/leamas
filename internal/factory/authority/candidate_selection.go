// SPDX-License-Identifier: Apache-2.0

package authority

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"path"
	"sort"
	"strings"
)

// closureCandidate is the independently validated identity supplied by
// an annotated ACT tag. ClosureCommit, rather than tag order or dates,
// is the value used by the selection rule.
type closureCandidate struct {
	TagName       string
	ActID         string
	TagObject     string
	ClosureCommit string
}

// selectMaximalCandidates removes candidates whose closure commit is a
// strict ancestor of another candidate. The callback is deliberately
// injected so the rule can be tested without making ref enumeration part
// of the oracle. Equal commits represent the same closure point and are
// retained as equivalent candidates; callers may reject multiple aliases
// if their ACT identities differ.
func selectMaximalCandidates(candidates []closureCandidate, ancestor func(string, string) bool) ([]closureCandidate, error) {
	if ancestor == nil {
		return nil, fmt.Errorf("candidate ancestry function is required")
	}
	if len(candidates) == 0 {
		return nil, nil
	}
	ordered := append([]closureCandidate(nil), candidates...)
	sort.Slice(ordered, func(i, j int) bool {
		if ordered[i].ClosureCommit != ordered[j].ClosureCommit {
			return ordered[i].ClosureCommit < ordered[j].ClosureCommit
		}
		if ordered[i].ActID != ordered[j].ActID {
			return ordered[i].ActID < ordered[j].ActID
		}
		return ordered[i].TagName < ordered[j].TagName
	})
	keep := make([]bool, len(ordered))
	for i := range keep {
		keep[i] = true
	}
	for i := range ordered {
		for j := range ordered {
			if i == j || ordered[i].ClosureCommit == ordered[j].ClosureCommit {
				continue
			}
			if ancestor(ordered[i].ClosureCommit, ordered[j].ClosureCommit) {
				keep[i] = false
				break
			}
		}
	}
	result := make([]closureCandidate, 0, len(ordered))
	for i, candidate := range ordered {
		if keep[i] {
			result = append(result, candidate)
		}
	}
	return result, nil
}

type candidateRejection struct {
	ActID  string
	Status AuthorityStatus
	Reason string
}

// discoverTaggedCandidates validates every ACT tag before it can enter
// the ancestry selection set. Invalid refs are recorded rather than
// silently becoming authorities; this lets the caller return a useful
// fail-closed status when no valid candidate remains.
func discoverTaggedCandidates(git GitRunner, repoRoot, headOID string) ([]closureCandidate, []candidateRejection, error) {
	out, err := git(repoRoot, "for-each-ref", "--format=%(refname)", "refs/tags/act/")
	if err != nil {
		return nil, nil, fmt.Errorf("enumerate ACT tags: %w", err)
	}
	var candidates []closureCandidate
	var rejected []candidateRejection
	for _, rawName := range strings.Split(out, "\n") {
		ref := strings.TrimSpace(rawName)
		if ref == "" {
			continue
		}
		if !strings.HasPrefix(ref, "refs/tags/act/") {
			continue
		}
		tagName := strings.TrimPrefix(ref, "refs/tags/")
		actID := strings.TrimPrefix(tagName, "act/")
		if !isValidActID(actID) {
			continue
		}
		reject := func(status AuthorityStatus, format string, args ...interface{}) {
			rejected = append(rejected, candidateRejection{ActID: actID, Status: status, Reason: fmt.Sprintf(format, args...)})
		}
		if _, err := git(repoRoot, "cat-file", "-e", ref); err != nil {
			reject(AuthorityInvalidGitObject, "tag ref %s does not exist: %v", tagName, err)
			continue
		}
		objectType, err := git(repoRoot, "cat-file", "-t", ref)
		if err != nil {
			reject(AuthorityInvalidGitObject, "inspect tag ref %s: %v", tagName, err)
			continue
		}
		if strings.TrimSpace(objectType) != "tag" {
			reject(AuthorityTagMismatch, "tag %s is not annotated (type=%q)", tagName, strings.TrimSpace(objectType))
			continue
		}
		tagObject, err := git(repoRoot, "rev-parse", "--verify", "--end-of-options", ref)
		if err != nil || strings.TrimSpace(tagObject) == "" {
			reject(AuthorityInvalidGitObject, "resolve tag object %s: %v", tagName, err)
			continue
		}
		peeled, err := git(repoRoot, "rev-parse", "--verify", "--end-of-options", ref+"^{}")
		if err != nil || strings.TrimSpace(peeled) == "" {
			reject(AuthorityInvalidGitObject, "tag %s does not peel to an object: %v", tagName, err)
			continue
		}
		peeled = strings.TrimSpace(peeled)
		peeledType, err := git(repoRoot, "cat-file", "-t", peeled)
		if err != nil || strings.TrimSpace(peeledType) != "commit" {
			reject(AuthorityInvalidGitObject, "tag %s peels to non-commit object %s (type=%q)", tagName, peeled, strings.TrimSpace(peeledType))
			continue
		}
		if !isAncestor(git, repoRoot, peeled, headOID) {
			reject(AuthorityInvalidArtifact, "closure commit %s for tag %s is not an ancestor of query HEAD %s", shortSHA(peeled), tagName, shortSHA(headOID))
			continue
		}
		candidate := closureCandidate{TagName: tagName, ActID: actID, TagObject: strings.TrimSpace(tagObject), ClosureCommit: peeled}
		if err := validateCandidateAtClosure(git, repoRoot, headOID, candidate); err != nil {
			reject(classifyCandidateError(err), "%v", err)
			continue
		}
		candidates = append(candidates, candidate)
	}
	return candidates, rejected, nil
}

func classifyCandidateError(err error) AuthorityStatus {
	var resolutionErr *AuthorityResolutionError
	if err != nil && errorAs(err, &resolutionErr) {
		return resolutionErr.Status
	}
	return AuthorityInvalidArtifact
}

// errorAs is kept local to avoid making candidate validation depend on
// callers' error-wrapping conventions.
func errorAs(err error, target **AuthorityResolutionError) bool {
	if err == nil {
		return false
	}
	if value, ok := err.(*AuthorityResolutionError); ok {
		*target = value
		return true
	}
	return false
}

func validateCandidateAtClosure(git GitRunner, repoRoot, headOID string, candidate closureCandidate) error {
	manifestPath := "docs/closure-manifests/" + candidate.ActID + ".json"
	raw, err := readBlobAt(git, repoRoot, candidate.ClosureCommit, manifestPath)
	if err != nil {
		return &AuthorityResolutionError{Status: AuthorityInvalidArtifact, Reason: fmt.Sprintf("candidate manifest %s is missing at closure commit: %v", manifestPath, err)}
	}
	var manifest ManifestLoose
	if err := json.Unmarshal([]byte(raw), &manifest); err != nil {
		return &AuthorityResolutionError{Status: AuthorityInvalidArtifact, Reason: fmt.Sprintf("decode candidate manifest %s: %v", manifestPath, err)}
	}
	if manifest.ActID != candidate.ActID {
		return &AuthorityResolutionError{Status: AuthorityInvalidArtifact, Reason: fmt.Sprintf("candidate tag %s does not match manifest act_id %q", candidate.TagName, manifest.ActID)}
	}
	if manifest.Tag != "" && manifest.Tag != candidate.TagName {
		return &AuthorityResolutionError{Status: AuthorityInvalidArtifact, Reason: fmt.Sprintf("manifest tag %q does not match candidate tag %q", manifest.Tag, candidate.TagName)}
	}
	if err := validateCandidateArtifactBlobs(git, repoRoot, candidate.ClosureCommit, candidate.ActID, manifest, []byte(raw)); err != nil {
		return &AuthorityResolutionError{Status: AuthorityInvalidArtifact, Reason: err.Error()}
	}
	// Resolve the identity from the exact tagged commit. This repeats the
	// ancestry and attestation checks at the same boundary used by the
	// selected authority, rather than selecting on a tag alone.
	if _, err := resolveSingleActAt(git, repoRoot, headOID, candidate.ClosureCommit, candidate.ActID, candidate); err != nil {
		return err
	}
	return nil
}

type looseBlobReference struct {
	Path      string `json:"path"`
	SHA256    string `json:"sha256"`
	ByteCount int64  `json:"byte_count"`
	Status    string `json:"status"`
}

// validateCandidateArtifactBlobs validates the canonical closure blobs
// and every indexed blob. It intentionally accepts both the original
// verbose manifest and the bounded index form used by the correction.
func validateCandidateArtifactBlobs(git GitRunner, repoRoot, commit, actID string, manifest ManifestLoose, manifestBytes []byte) error {
	refs := []looseBlobReference{
		{Path: "docs/closure-manifests/" + actID + ".json"},
		{Path: "docs/close-reports/" + actID + ".md"},
	}
	if manifest.Plan.Path != "" {
		refs = append(refs, looseBlobReference{Path: manifest.Plan.Path, SHA256: manifest.Plan.SHA256})
	}
	var envelope map[string]json.RawMessage
	if err := json.Unmarshal(manifestBytes, &envelope); err != nil {
		return err
	}
	for _, key := range []string{"artifact_index", "evidence_index", "detached_evidence_index", "artifacts"} {
		raw, ok := envelope[key]
		if !ok {
			continue
		}
		var entries []looseBlobReference
		if err := json.Unmarshal(raw, &entries); err != nil {
			return fmt.Errorf("decode %s: %w", key, err)
		}
		for _, entry := range entries {
			if entry.Status == "missing" || entry.Status == "fail" {
				continue
			}
			if entry.Path != "" {
				refs = append(refs, entry)
			}
		}
	}
	seen := make(map[string]struct{}, len(refs))
	for _, ref := range refs {
		if ref.Path == "" {
			continue
		}
		if _, exists := seen[ref.Path]; exists {
			continue
		}
		seen[ref.Path] = struct{}{}
		if err := validateRepositoryBlobPath(ref.Path); err != nil {
			return err
		}
		objectType, err := git(repoRoot, "cat-file", "-t", commit+":"+ref.Path)
		if err != nil || strings.TrimSpace(objectType) != "blob" {
			return fmt.Errorf("candidate artifact %s is not a blob at %s", ref.Path, shortSHA(commit))
		}
		if ref.SHA256 == "" && ref.ByteCount == 0 {
			continue
		}
		data, err := git(repoRoot, "cat-file", "blob", commit+":"+ref.Path)
		if err != nil {
			return fmt.Errorf("read candidate artifact %s: %w", ref.Path, err)
		}
		if ref.ByteCount != 0 && int64(len([]byte(data))) != ref.ByteCount {
			return fmt.Errorf("candidate artifact %s byte count mismatch", ref.Path)
		}
		if ref.SHA256 != "" {
			sum := sha256.Sum256([]byte(data))
			if !strings.EqualFold(hex.EncodeToString(sum[:]), ref.SHA256) {
				return fmt.Errorf("candidate artifact %s sha256 mismatch", ref.Path)
			}
		}
	}
	return nil
}

func validateRepositoryBlobPath(value string) error {
	if value == "" || strings.HasPrefix(value, "/") || path.Clean(value) != value || value == "." || strings.HasPrefix(value, "../") || strings.Contains(value, "/../") {
		return fmt.Errorf("candidate artifact path %q is not repository-relative", value)
	}
	return nil
}
