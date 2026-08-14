// SPDX-License-Identifier: Apache-2.0

package authority

import (
	"fmt"
	"sort"
	"strings"
)

// Resolve classifies lifecycle authority for the supplied resolver
// options. Candidate discovery and ancestry selection are kept at this
// entry boundary so the data model remains small and reviewable.
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

	if strings.TrimSpace(opts.ExplicitRange) != "" {
		// CORRECTION01: resolve the right endpoint of the explicit
		// range so downstream renderers can bind the digest
		// subject against the resolved endpoint rather than
		// falling back to ambient HEAD. Only A..B and B-style
		// forms are resolved here; rev-parse failure (which is
		// possible for malformed input) yields an empty
		// RangeSubjectEnd and the renderer falls back to its
		// documented empty-subject behavior.
		right := explicitRangeRightEndpoint(git, opts.RepoRoot, strings.TrimSpace(opts.ExplicitRange))
		return &ResolvedAuthority{
			AuthorityStatus: AuthorityExplicitRange,
			DigestRange:     strings.TrimSpace(opts.ExplicitRange),
			RangeSubjectEnd: right,
			ResolutionSrc:   "explicit_cli",
			ToolIdentity:    tool,
		}, nil
	}

	// A tag is only a candidate. Validate all candidates before applying
	// the ancestry-maximal selection rule; no ref order or timestamp is
	// consulted.
	tagCandidates, tagRejections, err := discoverTaggedCandidates(git, opts.RepoRoot, headOID)
	if err != nil {
		return nil, &AuthorityResolutionError{
			Status: AuthorityInvalidGitObject,
			Reason: fmt.Sprintf("scan ACT tags: %v", err),
		}
	}
	maximal, err := selectMaximalCandidates(tagCandidates, func(ancestor, descendant string) bool {
		return isAncestor(git, opts.RepoRoot, ancestor, descendant)
	})
	if err != nil {
		return nil, &AuthorityResolutionError{Status: AuthorityInvalidArtifact, Reason: err.Error()}
	}
	if len(maximal) > 1 {
		ids := make([]string, 0, len(maximal))
		for _, candidate := range maximal {
			ids = append(ids, candidate.ActID+"@"+shortSHA(candidate.ClosureCommit))
		}
		sort.Strings(ids)
		return nil, &AuthorityResolutionError{
			Status: AuthorityAmbiguousAuthority,
			Reason: fmt.Sprintf("incomparable closure candidates: %s", strings.Join(ids, ",")),
		}
	}
	if len(maximal) == 1 {
		candidate := maximal[0]
		resolved, err := resolveSingleActAt(git, opts.RepoRoot, headOID, candidate.ClosureCommit, candidate.ActID, candidate)
		if err != nil {
			return nil, err
		}
		resolved.ToolIdentity = tool
		return resolved, nil
	}

	// Preserve the local-manifest path for an untagged closure. A
	// malformed tag for the same ACT must not fall through to it.
	actIDs, err := headIntroducedActs(git, opts.RepoRoot, headOID)
	if err != nil {
		return nil, &AuthorityResolutionError{
			Status: AuthorityInvalidGitObject,
			Reason: fmt.Sprintf("scan HEAD-introduced ACTs: %v", err),
		}
	}
	for _, actID := range actIDs {
		for _, rejection := range tagRejections {
			if rejection.ActID == actID {
				return nil, &AuthorityResolutionError{Status: rejection.Status, Reason: rejection.Reason}
			}
		}
	}
	if len(actIDs) == 0 {
		if len(tagRejections) > 0 {
			rejection := tagRejections[0]
			return nil, &AuthorityResolutionError{Status: rejection.Status, Reason: rejection.Reason}
		}
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

	resolved, err := resolveSingleAct(git, opts.RepoRoot, headOID, actIDs[0])
	if err != nil {
		return nil, err
	}
	resolved.ToolIdentity = tool
	return resolved, nil
}

// explicitRangeRightEndpoint (CORRECTION01) resolves the right
// endpoint of an explicit --range expression to a full OID.
//
// Supports the documented forms:
//
//	<rev>                  (right endpoint = rev)
//	<base>..<rev>          (right endpoint = rev)
//	<rev1>..<rev2>..<rev3> (treat as base..rev = the trailing token)
//
// Returns the empty string when rev-parse fails (malformed
// input, missing object). The resolver never panics or fails
// closed solely on right-endpoint resolution: the explicit
// range still classifies as AuthorityExplicitRange with the
// original DigestRange verbatim, and downstream renderers
// fall back to their documented empty-subject behavior.
func explicitRangeRightEndpoint(git GitRunner, repoRoot, expr string) string {
	expr = strings.TrimSpace(expr)
	if expr == "" || git == nil {
		return ""
	}
	// Split on ".." and take the rightmost non-empty token.
	// This mirrors git's own symmetric difference syntax.
	parts := strings.Split(expr, "..")
	right := ""
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			right = p
		}
	}
	if right == "" {
		return ""
	}
	resolved, err := git(repoRoot, "rev-parse", "--verify", right+"^{commit}")
	if err != nil {
		return ""
	}
	resolved = strings.TrimSpace(resolved)
	if resolved == "" {
		return ""
	}
	return resolved
}
