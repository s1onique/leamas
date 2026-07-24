// SPDX-License-Identifier: Apache-2.0

package digest

import (
	"encoding/json"
	"sort"
	"strings"
)

// headIntroducedActs returns the ACT IDs that should be considered
// for HEAD. The candidate set is built from three sources, in this
// order:
//
//  1. Closure-artifact files (docs/closure-manifests/, docs/close-reports/)
//     introduced by HEAD itself (the diff between HEAD~1 and HEAD).
//  2. Annotated act/<ACT-ID> tags pointing at HEAD.
//  3. The ACT ID embedded in HEAD's commit subject.
//
// When none of the above produce a candidate, headIntroducedActs
// returns an empty slice so the caller can fall back to the
// single-commit path.
func headIntroducedActs(repoRoot, headOID string) ([]string, error) {
	acts := map[string]bool{}

	parentRaw, perr := runGitValueTrimmed(repoRoot, "rev-parse", headOID+"^")
	diffArgs := []string{"diff", "--name-only", "--diff-filter=A"}
	if perr != nil || strings.TrimSpace(parentRaw) == "" {
		diffArgs = append(diffArgs, emptyTreeBaseline, headOID)
	} else {
		diffArgs = append(diffArgs, parentRaw, headOID)
	}

	output, err := runGitValueTrimmed(repoRoot, diffArgs...)
	if err == nil {
		for _, line := range strings.Split(output, "\n") {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			if id, ok := actIDFromPath(line); ok {
				acts[id] = true
			}
		}
	}

	if len(acts) == 0 {
		ids, _ := listActsAtHeadTree(repoRoot, headOID)
		for _, id := range ids {
			acts[id] = true
		}
	}

	for _, tag := range listActTagsAtHEAD(repoRoot, headOID) {
		if id, ok := actIDFromTagName(tag); ok {
			acts[id] = true
		}
	}

	if len(acts) == 0 {
		if id, ok := actIDFromCommitSubject(repoRoot, headOID); ok {
			acts[id] = true
		}
	}

	out := make([]string, 0, len(acts))
	for id := range acts {
		out = append(out, id)
	}
	sort.Strings(out)
	return out, nil
}

// listActTagsAtHEAD returns the names of annotated act/<ACT-ID> tags
// pointing at HEAD.
func listActTagsAtHEAD(repoRoot, headOID string) []string {
	out, err := runGitOutput(repoRoot, "for-each-ref", "--format=%(refname:short)", "refs/tags/act/")
	if err != nil {
		return nil
	}
	var tags []string
	for _, line := range strings.Split(out, "\n") {
		name := strings.TrimSpace(line)
		if name == "" {
			continue
		}
		peeled, err := runGitValueTrimmed(repoRoot, "rev-parse", "--verify", "--end-of-options", name+"^{commit}")
		if err != nil {
			continue
		}
		if mustResolveOID(repoRoot, peeled) != headOID {
			continue
		}
		objType, err := runGitValueTrimmed(repoRoot, "cat-file", "-t", name)
		if err != nil {
			continue
		}
		if strings.TrimSpace(objType) != "tag" {
			continue
		}
		tags = append(tags, name)
	}
	return tags
}

// listActsAtHeadTree returns ACT IDs derived from any closure artifact
// present at HEAD's tree.
func listActsAtHeadTree(repoRoot, headOID string) ([]string, error) {
	acts := map[string]bool{}
	for _, dir := range []string{closureManifestsDir, closureReportsDir} {
		files, err := listTreeNames(repoRoot, headOID, dir)
		if err != nil {
			continue
		}
		for _, path := range files {
			if id, ok := actIDFromPath(path); ok {
				acts[id] = true
			}
		}
	}
	if len(acts) == 0 {
		return nil, nil
	}
	out := make([]string, 0, len(acts))
	for id := range acts {
		out = append(out, id)
	}
	sort.Strings(out)
	return out, nil
}

// actIDFromPath extracts an ACT identifier from a closure-artifact
// path. The path is expected to be a relative repo path of the form
//
//	docs/closure-manifests/<ID>.json
//	docs/closure-manifests/<ID>.attestation.json
//	docs/close-reports/<ID>.md
//
// Returns false if no ACT ID can be derived.
func actIDFromPath(path string) (string, bool) {
	path = strings.TrimSpace(path)
	if !strings.HasPrefix(path, closureManifestsDir+"/") &&
		!strings.HasPrefix(path, closureReportsDir+"/") {
		return "", false
	}
	base := path
	if idx := strings.LastIndex(base, "/"); idx >= 0 {
		base = base[idx+1:]
	}
	base = strings.TrimSuffix(base, ".json")
	base = strings.TrimSuffix(base, ".attestation")
	base = strings.TrimSuffix(base, ".md")
	if actIDPattern.MatchString(base) {
		return base, true
	}
	return "", false
}

// actIDFromCommitSubject extracts an ACT identifier from the first
// segment of HEAD's commit subject (e.g. "ACT-...-CORRECTION09:
// ..."). Returns false when no ACT identifier is found.
func actIDFromCommitSubject(repoRoot, headOID string) (string, bool) {
	output, err := runGitValueTrimmed(repoRoot, "log", "-1", "--format=%s", headOID)
	if err != nil {
		return "", false
	}
	subject := strings.TrimSpace(output)
	colonIdx := strings.Index(subject, ":")
	if colonIdx < 0 {
		colonIdx = len(subject)
	}
	head := strings.TrimSpace(subject[:colonIdx])
	if actIDPattern.MatchString(head) {
		return head, true
	}
	return "", false
}

// tryResolveFromManifest attempts to resolve a LifecycleResolution
// from docs/closure-manifests/<actID>.json at HEAD.
func tryResolveFromManifest(repoRoot, actID, headOID string) (candidateResolution, bool) {
	path := closureManifestsDir + "/" + actID + ".json"
	data, err := readHeadBlob(repoRoot, headOID, path)
	if err != nil {
		return candidateResolution{}, false
	}
	var mf manifestLoose
	if err := json.Unmarshal(data, &mf); err != nil {
		return candidateResolution{}, false
	}
	if mf.ActID != "" && mf.ActID != actID {
		return candidateResolution{}, false
	}
	freeze := mf.PlanFreeze.FreezeCommit
	if freeze == "" {
		return candidateResolution{}, false
	}
	subject := mf.Subject.CommitOID
	if subject == "" {
		return candidateResolution{}, false
	}
	freezeFull := mustResolveOID(repoRoot, freeze)
	subjectFull := mustResolveOID(repoRoot, subject)
	if freezeFull == "" || subjectFull == "" {
		return candidateResolution{}, false
	}
	base := mustResolveOID(repoRoot, freezeFull+"^")
	if base == "" {
		return candidateResolution{}, false
	}
	return candidateResolution{
		strategy: StrategyClosureManifest,
		resolution: &LifecycleResolution{
			Strategy:         StrategyClosureManifest,
			ActID:            actID,
			RangeBase:        base,
			RangeHead:        headOID,
			Reason:           "closure manifest plan_freeze.freeze_commit",
			LifecycleFreeze:  freezeFull,
			LifecycleSubject: subjectFull,
			LifecycleClosure: headOID,
		},
	}, true
}

// tryResolveFromAttestation attempts to resolve a LifecycleResolution
// from docs/closure-manifests/<actID>.attestation.json at HEAD.
func tryResolveFromAttestation(repoRoot, actID, headOID string) (candidateResolution, bool) {
	path := closureManifestsDir + "/" + actID + ".attestation.json"
	data, err := readHeadBlob(repoRoot, headOID, path)
	if err != nil {
		return candidateResolution{}, false
	}
	var att attestationLoose
	if err := json.Unmarshal(data, &att); err != nil {
		return candidateResolution{}, false
	}
	if att.ActID != "" && att.ActID != actID {
		return candidateResolution{}, false
	}
	freeze := att.FreezeReference.FreezeCommit
	closureRef := att.ClosureReference.ClosureCommit
	if freeze == "" || closureRef == "" {
		return candidateResolution{}, false
	}
	freezeFull := mustResolveOID(repoRoot, freeze)
	closureFull := mustResolveOID(repoRoot, closureRef)
	if freezeFull == "" || closureFull == "" {
		return candidateResolution{}, false
	}
	base := mustResolveOID(repoRoot, freezeFull+"^")
	if base == "" {
		return candidateResolution{}, false
	}
	return candidateResolution{
		strategy: StrategyPostClosureAttestation,
		resolution: &LifecycleResolution{
			Strategy:         StrategyPostClosureAttestation,
			ActID:            actID,
			RangeBase:        base,
			RangeHead:        closureFull,
			Reason:           "attestation freeze_reference..closure_reference",
			LifecycleFreeze:  freezeFull,
			LifecycleSubject: mustResolveOID(repoRoot, att.SubjectReference.SubjectCommit),
			LifecycleClosure: closureFull,
		},
	}, true
}

// tryResolveFromAnnotatedTag attempts to resolve a LifecycleResolution
// from an annotated tag refs/tags/act/<actID> pointing at HEAD.
func tryResolveFromAnnotatedTag(repoRoot, actID, headOID string) (candidateResolution, bool) {
	tagName := "act/" + actID
	peeled, err := runGitValueTrimmed(repoRoot, "rev-parse", "--verify", "--end-of-options", "refs/tags/"+tagName+"^{commit}")
	if err != nil {
		return candidateResolution{}, false
	}
	peeledFull := mustResolveOID(repoRoot, peeled)
	if peeledFull != headOID {
		return candidateResolution{}, false
	}
	tagType, err := runGitValueTrimmed(repoRoot, "cat-file", "-t", "refs/tags/"+tagName)
	if err != nil || strings.TrimSpace(tagType) != "tag" {
		return candidateResolution{}, false
	}
	rawTag, err := runGitOutput(repoRoot, "cat-file", "tag", "refs/tags/"+tagName)
	if err != nil {
		return candidateResolution{}, false
	}
	freeze, subject, closureRef := parseTagBodyLoose(rawTag)
	if freeze == "" {
		return candidateResolution{}, false
	}
	if closureRef == "" {
		closureRef = peeledFull
	}
	freezeFull := mustResolveOID(repoRoot, freeze)
	closureFull := mustResolveOID(repoRoot, closureRef)
	if freezeFull == "" || closureFull == "" {
		return candidateResolution{}, false
	}
	base := mustResolveOID(repoRoot, freezeFull+"^")
	if base == "" {
		return candidateResolution{}, false
	}
	return candidateResolution{
		strategy: StrategyAnnotatedActTag,
		resolution: &LifecycleResolution{
			Strategy:         StrategyAnnotatedActTag,
			ActID:            actID,
			RangeBase:        base,
			RangeHead:        closureFull,
			Reason:           "annotated tag " + tagName + " freeze..peeled",
			LifecycleFreeze:  freezeFull,
			LifecycleSubject: mustResolveOID(repoRoot, subject),
			LifecycleClosure: closureFull,
		},
	}, true
}

// tryResolveFromCloseReport attempts to resolve a LifecycleResolution
// from docs/close-reports/<actID>.md at HEAD. The close report must
// contain a structured "## Implementation Range" table with a BASE
// row identifying the implementation base commit.
func tryResolveFromCloseReport(repoRoot, actID, headOID string) (candidateResolution, bool) {
	path := closureReportsDir + "/" + actID + ".md"
	data, err := readHeadBlob(repoRoot, headOID, path)
	if err != nil {
		return candidateResolution{}, false
	}
	base, subject, ok := parseImplementationRangeTable(string(data))
	if !ok {
		return candidateResolution{}, false
	}
	baseFull := mustResolveOID(repoRoot, base)
	subjectFull := ""
	if subject != "" {
		subjectFull = mustResolveOID(repoRoot, subject)
	}
	if baseFull == "" {
		return candidateResolution{}, false
	}
	return candidateResolution{
		strategy: StrategyActCommitTrailers,
		resolution: &LifecycleResolution{
			Strategy:         StrategyActCommitTrailers,
			ActID:            actID,
			RangeBase:        baseFull,
			RangeHead:        headOID,
			Reason:           "close-report structured Implementation Range table",
			LifecycleFreeze:  "",
			LifecycleSubject: subjectFull,
			LifecycleClosure: headOID,
		},
	}, true
}
