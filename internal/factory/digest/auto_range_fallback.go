// SPDX-License-Identifier: Apache-2.0

package digest

import (
	"fmt"
	"strings"
)

// resolveSingleCommitFallback resolves the legacy HEAD~1..HEAD range.
// Allowed only when HEAD is not a closure-only or evidence-only commit
// AND no ACT spans multiple commits.
func resolveSingleCommitFallback(repoRoot, headOID string) (*LifecycleResolution, error) {
	parentRaw, err := runGitValueTrimmed(repoRoot, "rev-parse", "--verify", "--end-of-options", headOID+"^")
	if err != nil || strings.TrimSpace(parentRaw) == "" {
		return &LifecycleResolution{
			Strategy:         StrategyVerifiedSingleCommit,
			ActID:            "",
			RangeBase:        emptyTreeBaseline,
			RangeHead:        headOID,
			Reason:           "initial commit; empty tree baseline",
			LifecycleClosure: headOID,
		}, nil
	}

	introOutput, err := runGitOutput(repoRoot, "show", "--name-only", "--pretty=format:", headOID)
	if err == nil {
		trimmed := strings.TrimSpace(introOutput)
		if trimmed != "" && isEvidenceOnlyCommit(trimmed) {
			return nil, fmt.Errorf("%w: HEAD %s is evidence-only; supply --range or close the ACT with closure artifacts",
				ErrNoACTAuthority, shortSHA(headOID))
		}
	}

	parentFull := mustResolveOID(repoRoot, parentRaw)
	if parentFull == "" {
		return nil, fmt.Errorf("%w: cannot resolve HEAD parent", ErrNoACTAuthority)
	}

	// Use the literal "HEAD~1..HEAD" Range expression so legacy
	// callers and digests continue to display the canonical
	// single-commit form. RangeBase/RangeHead carry the full OIDs
	// for downstream tooling that needs them.
	return &LifecycleResolution{
		Strategy:         StrategyVerifiedSingleCommit,
		ActID:            "",
		RangeBase:        parentFull,
		RangeHead:        headOID,
		RangeExpression:  "HEAD~1..HEAD",
		Reason:           "no ACT authority at HEAD; HEAD~1..HEAD single-commit fallback",
		LifecycleClosure: headOID,
	}, nil
}

// isEvidenceOnlyCommit returns true when every file in the commit
// lives under one of the closure-artifact directories
// (docs/closure-manifests/, docs/close-reports/, docs/closure-plans/)
// OR is a Markdown file (which is treated as documentary evidence).
// Such commits carry only documentation/evidence, not production
// or test changes, so the single-commit fallback must not silently
// claim the implementation belongs to them.
func isEvidenceOnlyCommit(nameListBlob string) bool {
	lines := strings.Split(strings.TrimSpace(nameListBlob), "\n")
	source := false
	for _, line := range lines {
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
		source = true
	}
	return !source
}

// actIDFromTagName extracts the ACT ID from an annotated tag name of
// the form "act/<ACT-ID>". Returns false when the name does not match
// the expected pattern.
func actIDFromTagName(tagName string) (string, bool) {
	const prefix = "act/"
	if !strings.HasPrefix(tagName, prefix) {
		return "", false
	}
	candidate := strings.TrimPrefix(tagName, prefix)
	if actIDPattern.MatchString(candidate) {
		return candidate, true
	}
	return "", false
}
