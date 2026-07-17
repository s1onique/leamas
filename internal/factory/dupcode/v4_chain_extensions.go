// Package dupcode provides V4 region-bounded chain extension.
//
// Once chain-pair partitions are sorted, the chain constructor walks
// each partition in deterministic order and emits maximal contiguous
// chains. The chain adjacency rule (next.StartPos <= prev.EndPos+1 on
// both sides) is the same rule used by the legacy non-region chain
// constructor, but bounded by the structured partition so chains
// cannot cross region or offset boundaries.
package dupcode

func v4RegionBoundedChains(windowMap map[string][]rawWindow, analysisByPath map[string]*v4FileAnalysis) []cloneChain {
	if len(windowMap) == 0 {
		return nil
	}
	_, partitions := v4BuildRegionBoundedChainInputs(windowMap, analysisByPath)
	if len(partitions) == 0 {
		return nil
	}

	var keys []v4ChainPairKey
	for k := range partitions {
		keys = append(keys, k)
	}
	sortChainPairKeys(keys)

	var allChains []cloneChain
	for _, key := range keys {
		group := partitions[key]
		for _, chain := range extendRegionBoundedChain(group) {
			allChains = append(allChains, chain)
		}
	}
	return allChains
}

func extendRegionBoundedChain(group []v4RegionSeedMatch) []cloneChain {
	if len(group) == 0 {
		return nil
	}

	var chains []cloneChain
	var current []seedMatch

	flush := func() {
		if len(current) > 0 {
			if c := v4FinalizeChain(current); c != nil {
				chains = append(chains, *c)
			}
			current = nil
		}
	}

	for _, m := range group {
		if len(current) == 0 {
			current = append(current, m.Match)
			continue
		}
		prev := current[len(current)-1]
		canChain := m.Match.Left.StartPos <= prev.Left.EndPos+1 &&
			m.Match.Right.StartPos <= prev.Right.EndPos+1
		if canChain {
			current = append(current, m.Match)
		} else {
			flush()
			current = append(current, m.Match)
		}
	}
	flush()
	return chains
}
