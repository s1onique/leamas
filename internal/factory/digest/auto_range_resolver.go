// SPDX-License-Identifier: Apache-2.0

package digest

import (
	"fmt"
	"github.com/s1onique/leamas/internal/version"
)

// candidateResolution wraps a LifecycleResolution with the strategy
// label used to break ties on priority.
type candidateResolution struct {
	strategy   string
	resolution *LifecycleResolution
}

// strategyPriority ranks strategies from most authoritative to least.
var strategyPriority = map[string]int{
	StrategyClosureManifest:        5,
	StrategyPostClosureAttestation: 4,
	StrategyAnnotatedActTag:        3,
	StrategyActCommitTrailers:      2,
	StrategyVerifiedSingleCommit:   1,
}

// resolveLifecycleAtHEAD performs the full selection hierarchy on a
// clean working tree. The caller is responsible for short-circuiting
// when the working tree is dirty; this entry point always assumes the
// tree is clean.
//
// The returned LifecycleResolution includes Generator* metadata
// describing whether the running Leamas binary is up-to-date with
// respect to the repository HEAD. When the generator is stale, the
// returned error is ErrStaleGenerator so the CLI can render a precise
// diagnostic.
func resolveLifecycleAtHEAD(repoRoot string) (*LifecycleResolution, error) {
	headRaw, err := runGitValueTrimmed(repoRoot, "rev-parse", "HEAD")
	if err != nil {
		return nil, fmt.Errorf("resolve HEAD: %w", err)
	}
	headOID := mustResolveOID(repoRoot, headRaw)
	if headOID == "" {
		return nil, fmt.Errorf("invalid HEAD OID %q", headRaw)
	}

	resolution := &LifecycleResolution{
		RepositoryHead:      headOID,
		RangeHead:           headOID,
		LifecycleClosure:    headOID,
		GeneratorCommit:     version.Get().Commit,
		GeneratorIsAncestor: false,
		GeneratorStale:      false,
	}

	introducedActs, err := headIntroducedActs(repoRoot, headOID)
	if err != nil {
		return nil, fmt.Errorf("scan HEAD-introduced ACTs: %w", err)
	}

	candidates := []candidateResolution{}
	if len(introducedActs) == 0 {
		if id, ok := actIDFromCommitSubject(repoRoot, headOID); ok {
			introducedActs = []string{id}
		}
	}

	for _, actID := range introducedActs {
		if c, ok := tryResolveFromManifest(repoRoot, actID, headOID); ok {
			candidates = append(candidates, c)
			continue
		}
		if c, ok := tryResolveFromAttestation(repoRoot, actID, headOID); ok {
			candidates = append(candidates, c)
			continue
		}
		if c, ok := tryResolveFromAnnotatedTag(repoRoot, actID, headOID); ok {
			candidates = append(candidates, c)
			continue
		}
		if c, ok := tryResolveFromCloseReport(repoRoot, actID, headOID); ok {
			candidates = append(candidates, c)
			continue
		}
	}

	switch len(candidates) {
	case 0:
		single, err := resolveSingleCommitFallback(repoRoot, headOID)
		if err != nil {
			return nil, err
		}
		*resolution = *single
	case 1:
		*resolution = *candidates[0].resolution
	default:
		uniqActID := candidates[0].resolution.ActID
		uniqRange := candidates[0].resolution.Range()
		sameAct := true
		sameRange := true
		for _, c := range candidates[1:] {
			if c.resolution.ActID != uniqActID {
				sameAct = false
			}
			if c.resolution.Range() != uniqRange {
				sameRange = false
			}
		}
		if sameAct && sameRange {
			*resolution = *highestPriority(candidates).resolution
		} else {
			return nil, ambiguousRangeError(candidates, headOID)
		}
	}

	populateGeneratorMetadata(repoRoot, resolution)

	if err := ensureRangeBaseReachable(repoRoot, resolution); err != nil {
		return nil, err
	}

	if err := populateIncludedCommits(repoRoot, resolution); err != nil {
		return nil, err
	}

	return resolution, nil
}
