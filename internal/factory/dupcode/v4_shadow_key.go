package dupcode

// v4ShadowGroupKey is a structured shadow group identity. Paths are stored as
// complete values rather than serialized with a delimiter.
type v4ShadowGroupKey struct {
	LeftPath    string
	LeftRegion  int
	RightPath   string
	RightRegion int
}

func compareV4ShadowGroupKeys(left, right v4ShadowGroupKey) int {
	if left.LeftPath != right.LeftPath {
		return compareStrings(left.LeftPath, right.LeftPath)
	}
	if left.LeftRegion != right.LeftRegion {
		return compareInts(left.LeftRegion, right.LeftRegion)
	}
	if left.RightPath != right.RightPath {
		return compareStrings(left.RightPath, right.RightPath)
	}
	return compareInts(left.RightRegion, right.RightRegion)
}

func chainPairKeyForChain(c cloneChain, analysesByPath map[string]*v4FileAnalysis) v4ShadowGroupKey {
	leftPath, rightPath := chainPaths(c)
	leftStart := c.LeftRange.StartPos
	rightStart := c.RightRange.StartPos
	leftRegion := -1
	rightRegion := -1
	if analysesByPath != nil {
		if a, ok := analysesByPath[leftPath]; ok {
			if rid, ok := a.windowFitsRegion(c.LeftRange.StartPos, c.LeftRange.EndPos); ok {
				leftRegion = rid.Ordinal
			}
		}
		if a, ok := analysesByPath[rightPath]; ok {
			if rid, ok := a.windowFitsRegion(c.RightRange.StartPos, c.RightRange.EndPos); ok {
				rightRegion = rid.Ordinal
			}
		}
	}
	if leftPath > rightPath || (leftPath == rightPath && leftStart > rightStart) {
		leftPath, rightPath = rightPath, leftPath
		leftStart, rightStart = rightStart, leftStart
		leftRegion, rightRegion = rightRegion, leftRegion
	}
	return v4ShadowGroupKey{
		LeftPath: leftPath, LeftRegion: leftRegion,
		RightPath: rightPath, RightRegion: rightRegion,
	}
}

func compareStrings(left, right string) int {
	if left < right {
		return -1
	}
	if left > right {
		return 1
	}
	return 0
}

func compareInts(left, right int) int {
	if left < right {
		return -1
	}
	if left > right {
		return 1
	}
	return 0
}
