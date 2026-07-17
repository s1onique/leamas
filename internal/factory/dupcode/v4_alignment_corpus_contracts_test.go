// Package dupcode provides structural validation that runs before the
// production/oracle differential corpus.
package dupcode

import (
	"fmt"
	"reflect"
	"sort"
	"strings"
	"testing"
)

func TestV4Alignment_CorpusContracts(t *testing.T) {
	errs := v4CorpusContractErrors(v4BuildAlignmentCorpus())
	if len(errs) != 0 {
		t.Fatalf("corpus contract violations:\n  %s", strings.Join(errs, "\n  "))
	}
}

func v4RequireCorpusContracts(t *testing.T, corpus []v4CorpusFixture) {
	t.Helper()
	if errs := v4CorpusContractErrors(corpus); len(errs) != 0 {
		t.Fatalf("corpus contracts must pass before differential execution:\n  %s",
			strings.Join(errs, "\n  "))
	}
}

func v4CorpusContractErrors(corpus []v4CorpusFixture) []string {
	var errs []string
	dimensionCounts := make(map[v4CorpusDimension]int)
	names := make(map[string]int)
	for _, fx := range corpus {
		dimensionCounts[fx.Dimension]++
		names[fx.Name]++
		if fx.Name == "" {
			errs = append(errs, fmt.Sprintf("dimension %q has an empty fixture name", fx.Dimension))
		}
		if err := v4ValidatePrimaryPair(fx); err != nil {
			errs = append(errs, err.Error())
		}
		if err := v4ValidateDimensionStructure(fx); err != nil {
			errs = append(errs, err.Error())
		}
	}
	for _, dimension := range requiredV4CorpusDimensions {
		if dimensionCounts[dimension] != 1 {
			errs = append(errs, fmt.Sprintf("dimension %s primary fixture count=%d, want exactly 1",
				dimension, dimensionCounts[dimension]))
		}
	}
	for name, count := range names {
		if count != 1 {
			errs = append(errs, fmt.Sprintf("fixture name %q count=%d, want exactly 1", name, count))
		}
	}
	sort.Strings(errs)
	return errs
}

func v4ValidatePrimaryPair(fx v4CorpusFixture) error {
	left := v4WindowsForDeclaredRegion(fx, fx.LeftRegion)
	right := v4WindowsForDeclaredRegion(fx, fx.RightRegion)
	if len(left) == 0 || len(right) == 0 {
		return fmt.Errorf("%s: primary sides resolve to left=%d right=%d windows", fx.Name, len(left), len(right))
	}
	if fx.LeftRegion == fx.RightRegion {
		return fmt.Errorf("%s: cross-region primary sides resolve to one region %s", fx.Name, fx.LeftRegion)
	}
	if v4DimensionMustAlign(fx.Dimension) && !v4SequencesPositionallyAligned(left, right) {
		return fmt.Errorf("%s: aligned fixture is not positionally aligned", fx.Name)
	}
	if v4DimensionMustBeAsymmetric(fx.Dimension) && v4SequencesPositionallyAligned(left, right) {
		return fmt.Errorf("%s: asymmetric fixture is positionally aligned", fx.Name)
	}
	return nil
}

func v4DimensionMustAlign(d v4CorpusDimension) bool {
	return d == v4AlignedN8 || d == v4AlignedN32 || d == v4AlignedN128 || d == v4SamePathDifferentOrdinals
}

func v4DimensionMustBeAsymmetric(d v4CorpusDimension) bool {
	switch d {
	case v4LeadingExtraLeft, v4LeadingExtraRight, v4MiddleExtra,
		v4TrailingExtra, v4UnequalCardinality, v4NonUniformSpacing,
		v4OffIndexMaximalChain, v4TwoIndependentOffsetChains,
		v4ThreeRegionsAsymmetric, v4RepeatedWithinRegion,
		v4ShuffledRawInput, v4DuplicateRawWindow:
		return true
	default:
		return false
	}
}

func v4SequencesPositionallyAligned(left, right []v4RawWindow) bool {
	if len(left) != len(right) {
		return false
	}
	if len(left) == 0 {
		return true
	}
	for i := 1; i < len(left); i++ {
		if left[i].StartPos-left[0].StartPos != right[i].StartPos-right[0].StartPos {
			return false
		}
		if left[i].EndPos-left[0].EndPos != right[i].EndPos-right[0].EndPos {
			return false
		}
	}
	return true
}

func v4ValidateDimensionStructure(fx v4CorpusFixture) error {
	left := v4WindowsForDeclaredRegion(fx, fx.LeftRegion)
	right := v4WindowsForDeclaredRegion(fx, fx.RightRegion)
	switch fx.Dimension {
	case v4AlignedN8:
		return v4RequireSideCounts(fx, left, right, 8, 8)
	case v4AlignedN32:
		return v4RequireSideCounts(fx, left, right, 32, 32)
	case v4AlignedN128:
		return v4RequireSideCounts(fx, left, right, 128, 128)
	case v4LeadingExtraLeft:
		return v4RequireExtraAt(fx, left, right, 0)
	case v4LeadingExtraRight:
		return v4RequireExtraAt(fx, right, left, 0)
	case v4MiddleExtra:
		return v4RequireMiddleExtra(fx, left, right)
	case v4TrailingExtra:
		return v4RequireExtraAt(fx, right, left, len(right)-1)
	case v4UnequalCardinality:
		if len(left) == len(right) {
			return fmt.Errorf("%s: unequal-cardinality fixture has equal side counts", fx.Name)
		}
	case v4NonUniformSpacing:
		if len(left) != len(right) || !v4HasNonUniformSpacing(left, right) {
			return fmt.Errorf("%s: non-uniform-spacing structure is absent", fx.Name)
		}
	case v4OffIndexMaximalChain:
		if !v4HasOffIndexMaximalChain(left, right) {
			return fmt.Errorf("%s: no maximal chain exists outside the same-index diagonal", fx.Name)
		}
	case v4TwoIndependentOffsetChains:
		if v4SeparatedRuns(left) < 2 || v4SeparatedRuns(right) < 2 || v4LongOffsetGroups(left, right) < 2 {
			return fmt.Errorf("%s: two independent offset chains are absent", fx.Name)
		}
	case v4ThreeRegionsAsymmetric:
		if !v4HasThreeAsymmetricRegions(fx) {
			return fmt.Errorf("%s: three asymmetric owned regions are absent", fx.Name)
		}
	case v4RepeatedWithinRegion:
		if !v4HasNonOverlappingPair(left) {
			return fmt.Errorf("%s: repeated non-overlapping windows within one region are absent", fx.Name)
		}
	case v4ShuffledRawInput:
		if reflect.DeepEqual(fx.RawWindows, v4CanonicalRawWindows(fx.RawWindows)) {
			return fmt.Errorf("%s: shuffled fixture is already in canonical raw order", fx.Name)
		}
	case v4UnownedWindow:
		if v4CountUnowned(fx) < 2 {
			return fmt.Errorf("%s: unowned fixture resolves all but %d windows", fx.Name, v4CountUnowned(fx))
		}
	case v4DuplicateRawWindow:
		if !v4HasDuplicateRawWindow(fx.RawWindows) {
			return fmt.Errorf("%s: duplicate raw window is absent", fx.Name)
		}
	case v4SamePathDifferentOrdinals:
		if fx.LeftRegion.Path != fx.RightRegion.Path || fx.LeftRegion.Ordinal == fx.RightRegion.Ordinal {
			return fmt.Errorf("%s: sides do not resolve to one path with different ordinals", fx.Name)
		}
	default:
		return fmt.Errorf("%s: unknown claimed dimension %q", fx.Name, fx.Dimension)
	}
	return nil
}

func v4RequireSideCounts(fx v4CorpusFixture, left, right []v4RawWindow, wantLeft, wantRight int) error {
	if len(left) != wantLeft || len(right) != wantRight {
		return fmt.Errorf("%s: side counts=(%d,%d), want=(%d,%d)", fx.Name, len(left), len(right), wantLeft, wantRight)
	}
	return nil
}

func v4RequireExtraAt(fx v4CorpusFixture, larger, smaller []v4RawWindow, want int) error {
	if len(larger) != len(smaller)+1 || !v4RemovalAligns(larger, smaller, want) {
		return fmt.Errorf("%s: required extra occurrence at index %d is absent", fx.Name, want)
	}
	return nil
}

func v4RequireMiddleExtra(fx v4CorpusFixture, left, right []v4RawWindow) error {
	larger, smaller := left, right
	if len(right) > len(left) {
		larger, smaller = right, left
	}
	if len(larger) != len(smaller)+1 {
		return fmt.Errorf("%s: middle-extra side counts do not differ by one", fx.Name)
	}
	for i := 1; i < len(larger)-1; i++ {
		if v4RemovalAligns(larger, smaller, i) {
			return nil
		}
	}
	return fmt.Errorf("%s: no removable middle occurrence restores alignment", fx.Name)
}

func v4RemovalAligns(larger, smaller []v4RawWindow, remove int) bool {
	if remove < 0 || remove >= len(larger) {
		return false
	}
	candidate := append([]v4RawWindow(nil), larger[:remove]...)
	candidate = append(candidate, larger[remove+1:]...)
	return v4SequencesPositionallyAligned(candidate, smaller)
}

func v4HasNonUniformSpacing(left, right []v4RawWindow) bool {
	for _, side := range [][]v4RawWindow{left, right} {
		if len(side) < 3 {
			continue
		}
		step := side[1].StartPos - side[0].StartPos
		for i := 2; i < len(side); i++ {
			if side[i].StartPos-side[i-1].StartPos != step {
				return true
			}
		}
	}
	return false
}

func v4OffsetGroupSizes(left, right []v4RawWindow) map[int]int {
	groups := make(map[int]int)
	for _, l := range left {
		for _, r := range right {
			groups[r.StartPos-l.StartPos]++
		}
	}
	return groups
}

func v4HasOffIndexMaximalChain(left, right []v4RawWindow) bool {
	groups := v4OffsetGroupSizes(left, right)
	max := 0
	for _, count := range groups {
		if count > max {
			max = count
		}
	}
	if max < 3 {
		return false
	}
	for i, l := range left {
		for j, r := range right {
			if i != j && groups[r.StartPos-l.StartPos] == max {
				return true
			}
		}
	}
	return false
}

func v4LongOffsetGroups(left, right []v4RawWindow) int {
	count := 0
	for _, size := range v4OffsetGroupSizes(left, right) {
		if size >= 2 {
			count++
		}
	}
	return count
}

func v4SeparatedRuns(windows []v4RawWindow) int {
	if len(windows) == 0 {
		return 0
	}
	runs := 1
	for i := 1; i < len(windows); i++ {
		if windows[i].StartPos > windows[i-1].EndPos+1 {
			runs++
		}
	}
	return runs
}

func v4HasThreeAsymmetricRegions(fx v4CorpusFixture) bool {
	counts := make(map[v4SyntaxRegionID]int)
	for _, w := range fx.RawWindows {
		if owner, ok := v4DeclaredWindowOwner(fx, w); ok {
			counts[owner]++
		}
	}
	if len(counts) < 3 {
		return false
	}
	first := -1
	for _, count := range counts {
		if first < 0 {
			first = count
		} else if count != first {
			return true
		}
	}
	return false
}

func v4HasNonOverlappingPair(windows []v4RawWindow) bool {
	for i := range windows {
		for j := i + 1; j < len(windows); j++ {
			if windows[i].EndPos < windows[j].StartPos || windows[j].EndPos < windows[i].StartPos {
				return true
			}
		}
	}
	return false
}

func v4CountUnowned(fx v4CorpusFixture) int {
	count := 0
	for _, w := range fx.RawWindows {
		if _, ok := v4DeclaredWindowOwner(fx, w); !ok {
			count++
		}
	}
	return count
}

func v4HasDuplicateRawWindow(windows []v4RawWindow) bool {
	seen := make(map[v4RawWindow]bool)
	for _, w := range windows {
		if seen[w] {
			return true
		}
		seen[w] = true
	}
	return false
}
