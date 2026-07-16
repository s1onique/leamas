package dupcode

import (
	"fmt"
	"sort"
)

// v4MaterializeComponents validates pair evidence and turns it into
// deterministic connected components of exact occurrence vertices.
func v4MaterializeComponents(chains []cloneChain, files map[string]*v4AnalyzedFile) ([]v4InternalFinding, error) {
	if len(chains) == 0 {
		return nil, nil
	}
	edges := make([]v4PairCloneEvidence, 0, len(chains))
	for _, chain := range chains {
		evidence, err := v4PairEvidenceFromChain(chain, files)
		if err != nil {
			return nil, err
		}
		edges = append(edges, evidence)
	}
	sort.Slice(edges, func(i, j int) bool {
		if order := compareV4ContentKeys(edges[i].ContentKey, edges[j].ContentKey); order != 0 {
			return order < 0
		}
		if order := compareV4MaximalOccurrences(edges[i].Left, edges[j].Left); order != 0 {
			return order < 0
		}
		return compareV4MaximalOccurrences(edges[i].Right, edges[j].Right) < 0
	})

	byDigest := make(map[string]int)
	byKey := make(map[v4ExactContentKey][]v4PairCloneEvidence)
	for _, edge := range edges {
		if count, ok := byDigest[edge.ContentKey.Digest]; ok && count != edge.ContentKey.TokenCount {
			return nil, fmt.Errorf("dupcode: exact-content geometry conflict for digest %q: token counts %d and %d",
				edge.ContentKey.Digest, count, edge.ContentKey.TokenCount)
		}
		byDigest[edge.ContentKey.Digest] = edge.ContentKey.TokenCount
		byKey[edge.ContentKey] = append(byKey[edge.ContentKey], edge)
	}

	keys := make([]v4ExactContentKey, 0, len(byKey))
	for key := range byKey {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i, j int) bool { return compareV4ContentKeys(keys[i], keys[j]) < 0 })

	var findings []v4InternalFinding
	for _, contentKey := range keys {
		group := byKey[contentKey]
		var flattened []maximalOccurrence
		vertices := make(map[maximalOccurrenceKey]maximalOccurrence)
		adjacency := make(map[maximalOccurrenceKey][]maximalOccurrenceKey)
		for _, edge := range group {
			flattened = append(flattened, edge.Left, edge.Right)
			leftKey := occurrenceKey(edge.Left)
			rightKey := occurrenceKey(edge.Right)
			vertices[leftKey] = edge.Left
			vertices[rightKey] = edge.Right
			adjacency[leftKey] = append(adjacency[leftKey], rightKey)
			adjacency[rightKey] = append(adjacency[rightKey], leftKey)
		}
		// This is intentionally one check over the complete content group,
		// before any vertex map deduplication.
		if err := validateOccurrenceIdentityInvariants(flattened); err != nil {
			return nil, err
		}

		vertexKeys := make([]maximalOccurrenceKey, 0, len(vertices))
		for key := range vertices {
			vertexKeys = append(vertexKeys, key)
		}
		sort.Slice(vertexKeys, func(i, j int) bool {
			return compareV4OccurrenceKeys(vertexKeys[i], vertexKeys[j]) < 0
		})
		for _, key := range vertexKeys {
			sort.Slice(adjacency[key], func(i, j int) bool {
				return compareV4OccurrenceKeys(adjacency[key][i], adjacency[key][j]) < 0
			})
		}

		visited := make(map[maximalOccurrenceKey]bool, len(vertices))
		for _, start := range vertexKeys {
			if visited[start] {
				continue
			}
			var component []maximalOccurrenceKey
			var visit func(maximalOccurrenceKey)
			visit = func(key maximalOccurrenceKey) {
				if visited[key] {
					return
				}
				visited[key] = true
				component = append(component, key)
				for _, next := range adjacency[key] {
					visit(next)
				}
			}
			visit(start)
			if len(component) < 2 {
				continue
			}
			sort.Slice(component, func(i, j int) bool {
				return compareV4OccurrenceKeys(component[i], component[j]) < 0
			})
			occurrences := make([]maximalOccurrence, len(component))
			lineCount := 0
			for i, key := range component {
				occurrences[i] = vertices[key]
				if span := occurrences[i].EndLine - occurrences[i].StartLine + 1; span > lineCount {
					lineCount = span
				}
			}
			findings = append(findings, v4InternalFinding{
				StableFingerprint: v4StableFingerprintForContentKey(contentKey),
				TokenCount:        contentKey.TokenCount,
				LineCount:         lineCount,
				Occurrences:       occurrences,
			})
		}
	}
	return findings, nil
}

func v4PairEvidenceFromChain(chain cloneChain, files map[string]*v4AnalyzedFile) (v4PairCloneEvidence, error) {
	if len(chain.Matches) == 0 {
		return v4PairCloneEvidence{}, fmt.Errorf("dupcode: cannot materialize an empty clone chain")
	}
	leftPath := chain.Matches[0].Left.Path
	rightPath := chain.Matches[0].Right.Path
	left, err := v4OccurrenceForRange(leftPath, chain.LeftRange, files)
	if err != nil {
		return v4PairCloneEvidence{}, err
	}
	right, err := v4OccurrenceForRange(rightPath, chain.RightRange, files)
	if err != nil {
		return v4PairCloneEvidence{}, err
	}
	leftKey, err := v4ExactContentKeyForOccurrence(*files[leftPath], left)
	if err != nil {
		return v4PairCloneEvidence{}, err
	}
	rightKey, err := v4ExactContentKeyForOccurrence(*files[rightPath], right)
	if err != nil {
		return v4PairCloneEvidence{}, err
	}
	if leftKey.TokenCount != rightKey.TokenCount {
		return v4PairCloneEvidence{}, fmt.Errorf("dupcode: pair geometry conflict %s:%d-%d has %d tokens, %s:%d-%d has %d",
			left.Path, left.StartPos, left.EndPos, leftKey.TokenCount,
			right.Path, right.StartPos, right.EndPos, rightKey.TokenCount)
	}
	if leftKey.Digest != rightKey.Digest {
		return v4PairCloneEvidence{}, fmt.Errorf("dupcode: pair content conflict %s:%d-%d and %s:%d-%d have different exact digests",
			left.Path, left.StartPos, left.EndPos, right.Path, right.StartPos, right.EndPos)
	}
	if compareV4MaximalOccurrences(left, right) > 0 {
		left, right = right, left
	}
	lineCount := left.EndLine - left.StartLine + 1
	if rightSpan := right.EndLine - right.StartLine + 1; rightSpan > lineCount {
		lineCount = rightSpan
	}
	return v4PairCloneEvidence{
		ContentKey: leftKey,
		Left:       left,
		Right:      right,
		LineCount:  lineCount,
	}, nil
}

func v4OccurrenceForRange(path string, span tokenRange, files map[string]*v4AnalyzedFile) (maximalOccurrence, error) {
	file, ok := files[path]
	if !ok || file == nil {
		return maximalOccurrence{}, fmt.Errorf("dupcode: missing analyzed file %q", path)
	}
	if span.StartPos < 0 || span.EndPos < span.StartPos || span.EndPos >= len(file.Analysis.Lines) {
		return maximalOccurrence{}, fmt.Errorf("dupcode: invalid occurrence range %s:%d-%d", path, span.StartPos, span.EndPos)
	}
	return maximalOccurrence{
		Path:      path,
		StartPos:  span.StartPos,
		EndPos:    span.EndPos,
		StartLine: file.Analysis.Lines[span.StartPos],
		EndLine:   file.Analysis.Lines[span.EndPos],
	}, nil
}

func compareV4OccurrenceKeys(left, right maximalOccurrenceKey) int {
	if left.Path != right.Path {
		return compareStrings(left.Path, right.Path)
	}
	if left.StartPos != right.StartPos {
		return compareInts(left.StartPos, right.StartPos)
	}
	return compareInts(left.EndPos, right.EndPos)
}

func compareV4MaximalOccurrences(left, right maximalOccurrence) int {
	return compareV4OccurrenceKeys(occurrenceKey(left), occurrenceKey(right))
}

// v4SuppressComponentShadows removes only structurally proven sub-findings.
func v4SuppressComponentShadows(findings []v4InternalFinding, files map[string]*v4AnalyzedFile) []v4InternalFinding {
	if len(findings) < 2 || len(files) == 0 {
		return findings
	}
	shadowed := make([]bool, len(findings))
	for small := range findings {
		for large := range findings {
			if small == large || len(findings[large].Occurrences) < len(findings[small].Occurrences) {
				continue
			}
			if componentIsStructuralShadow(findings[small], findings[large], files) {
				shadowed[small] = true
				break
			}
		}
	}
	result := make([]v4InternalFinding, 0, len(findings))
	for i, finding := range findings {
		if !shadowed[i] {
			result = append(result, finding)
		}
	}
	return result
}

func componentIsStructuralShadow(small, large v4InternalFinding, files map[string]*v4AnalyzedFile) bool {
	if large.TokenCount <= small.TokenCount {
		return false
	}
	strict := false
	expectedOffset := 0
	haveOffset := false
	for _, smallOcc := range small.Occurrences {
		var candidate *maximalOccurrence
		for i := range large.Occurrences {
			largeOcc := large.Occurrences[i]
			if largeOcc.Path != smallOcc.Path || smallOcc.StartPos < largeOcc.StartPos || smallOcc.EndPos > largeOcc.EndPos {
				continue
			}
			if candidate != nil {
				return false
			}
			candidate = &large.Occurrences[i]
		}
		if candidate == nil {
			return false
		}
		offset := smallOcc.StartPos - candidate.StartPos
		if haveOffset && offset != expectedOffset {
			return false
		}
		expectedOffset = offset
		haveOffset = true
		if smallOcc.StartPos != candidate.StartPos || smallOcc.EndPos != candidate.EndPos {
			strict = true
		}
		file := files[smallOcc.Path]
		if file == nil || !equalNormalizedSubslice(*file, smallOcc, *candidate) {
			return false
		}
	}
	return strict
}

func equalNormalizedSubslice(file v4AnalyzedFile, small, large maximalOccurrence) bool {
	relativeStart := small.StartPos - large.StartPos
	relativeEnd := relativeStart + small.EndPos - small.StartPos
	largeTokens := large.EndPos - large.StartPos + 1
	if relativeStart < 0 || relativeEnd >= largeTokens {
		return false
	}
	actual := file.NormalizedTokens[small.StartPos : small.EndPos+1]
	wanted := file.NormalizedTokens[large.StartPos+relativeStart : large.StartPos+relativeEnd+1]
	if len(actual) != len(wanted) {
		return false
	}
	for i := range actual {
		if actual[i] != wanted[i] {
			return false
		}
	}
	return true
}

// v4SuppressContainedSameFileShadows handles a detector-specific artifact:
// repeated threshold windows entirely inside the same physical occurrence of
// a larger cross-file component. It is intentionally narrower than ordinary
// component shadowing and never applies to a multi-file partial clone or to
// equal-sized repeated multiplicity.
func v4SuppressContainedSameFileShadows(findings []v4InternalFinding, files map[string]*v4AnalyzedFile) []v4InternalFinding {
	shadowed := make([]bool, len(findings))
	for small := range findings {
		if len(findings[small].Occurrences) < 2 || !allOccurrencesSharePath(findings[small].Occurrences) {
			continue
		}
		for large := range findings {
			if small == large || len(findings[large].Occurrences) <= 1 ||
				findings[large].TokenCount <= findings[small].TokenCount ||
				!containsMultiplePaths(findings[large].Occurrences) {
				continue
			}
			if containedInOneCrossFileOccurrence(findings[small], findings[large], files) {
				shadowed[small] = true
				break
			}
		}
	}
	result := make([]v4InternalFinding, 0, len(findings))
	for i, finding := range findings {
		if !shadowed[i] {
			result = append(result, finding)
		}
	}
	return result
}

func allOccurrencesSharePath(occurrences []maximalOccurrence) bool {
	if len(occurrences) == 0 {
		return false
	}
	for _, occurrence := range occurrences[1:] {
		if occurrence.Path != occurrences[0].Path {
			return false
		}
	}
	return true
}

func containsMultiplePaths(occurrences []maximalOccurrence) bool {
	if len(occurrences) < 2 {
		return false
	}
	return occurrences[0].Path != occurrences[len(occurrences)-1].Path
}

func containedInOneCrossFileOccurrence(small, large v4InternalFinding, files map[string]*v4AnalyzedFile) bool {
	path := small.Occurrences[0].Path
	for _, occurrence := range small.Occurrences {
		var outer *maximalOccurrence
		for i := range large.Occurrences {
			candidate := &large.Occurrences[i]
			if candidate.Path != path || occurrence.StartPos <= candidate.StartPos || occurrence.EndPos >= candidate.EndPos {
				continue
			}
			if outer != nil {
				return false
			}
			outer = candidate
		}
		if outer == nil {
			return false
		}
		file := files[path]
		if file == nil || !equalNormalizedSubslice(*file, occurrence, *outer) {
			return false
		}
	}
	return true
}
