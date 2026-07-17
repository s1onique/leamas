// Package dupcode provides the V4 materialization performance
// benchmarks for ACT-LEAMAS-FACTORY-DUPCODE-V4-ALL-PAIRS-MATERIALIZATION-PERFORMANCE01.
//
// The fixtures consumed by these benchmarks live in
// v4_materialization_perf_fixtures_test.go. Both files share
// package-level helpers because they belong to the same Go package.
package dupcode

import (
	"os"
	"path/filepath"
	"sort"
	"testing"
)

// firstBucketWindows returns the windows of the first fingerprint
// bucket in deterministic (sorted-fingerprint) order. It is used to
// keep the inner benchmark loop allocation-free.
func firstBucketWindows(wm map[string][]rawWindow) []rawWindow {
	fps := make([]string, 0, len(wm))
	for fp := range wm {
		fps = append(fps, fp)
	}
	sort.Strings(fps)
	if len(fps) == 0 {
		return nil
	}
	return wm[fps[0]]
}

// perfRepoRoot returns the absolute path to the repository root
// without depending on a *testing.T helper so benchmark entry points
// can call it.
func perfRepoRoot() string {
	wd, err := os.Getwd()
	if err != nil {
		panic("os.Getwd: " + err.Error())
	}
	return filepath.Clean(filepath.Join(wd, "..", "..", ".."))
}

// BenchmarkV4Perf_SlidingNWay runs the full V4 chain-input build on
// the N-way fixture for the requested size.
func BenchmarkV4Perf_SlidingNWay(b *testing.B) {
	for _, fixture := range fixtureSizes {
		size := fixture.size
		label := fixture.descriptiveID
		b.Run(label, func(b *testing.B) {
			wm := makeSlidingWindowMap(size)
			analyses := collectAnalysesForWindowMap(wm)
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, partitions := v4BuildRegionBoundedChainInputs(wm, analyses)
				if len(partitions) == 0 {
					_, _ = v4BuildRegionBoundedChainInputs(wm, analyses)
				}
			}
		})
	}
}

// BenchmarkV4Perf_GenerateRegionAnnotatedMatches exercises the
// inner candidate-generation helper directly so the benchstat
// comparison isolates the materialisation phase.
func BenchmarkV4Perf_GenerateRegionAnnotatedMatches(b *testing.B) {
	for _, fixture := range fixtureSizes {
		size := fixture.size
		label := fixture.descriptiveID
		b.Run(label, func(b *testing.B) {
			wm := makeSlidingWindowMap(size)
			windows := firstBucketWindows(wm)
			analyses := collectAnalysesForWindowMap(wm)
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = generateRegionAnnotatedMatches("perf-fp", windows, analyses)
			}
		})
	}
}

// BenchmarkV4Perf_TwoIndependentBodies runs the chain-input build
// on a fixture containing a shared sliding bucket plus two disjoint
// duplicate bodies.
func BenchmarkV4Perf_TwoIndependentBodies(b *testing.B) {
	for _, fixture := range fixtureSizes {
		size := fixture.size
		label := fixture.descriptiveID
		b.Run(label, func(b *testing.B) {
			wm := makeTwoIndependentBodies(size)
			analyses := collectAnalysesForWindowMap(wm)
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, partitions := v4BuildRegionBoundedChainInputs(wm, analyses)
				if len(partitions) == 0 {
					_, _ = v4BuildRegionBoundedChainInputs(wm, analyses)
				}
			}
		})
	}
}

// BenchmarkV4Perf_ShadowFixture exercises shadow-suppression with a
// three-position overlapping window fixture.
func BenchmarkV4Perf_ShadowFixture(b *testing.B) {
	wm := makeShadowFixture()
	analyses := collectAnalysesForWindowMap(wm)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, partitions := v4BuildRegionBoundedChainInputs(wm, analyses)
		if len(partitions) == 0 {
			_, _ = v4BuildRegionBoundedChainInputs(wm, analyses)
		}
	}
}

// BenchmarkV4Perf_RepeatedMultiplicity runs the repeated-
// multiplicity fixture.
func BenchmarkV4Perf_RepeatedMultiplicity(b *testing.B) {
	wm := repeatedMultiplicityFixture()
	analyses := collectAnalysesForWindowMap(wm)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, partitions := v4BuildRegionBoundedChainInputs(wm, analyses)
		if len(partitions) == 0 {
			_, _ = v4BuildRegionBoundedChainInputs(wm, analyses)
		}
	}
}

// BenchmarkV4Perf_EmptyCorpus runs the pipeline on an empty window
// map to establish the no-finding detection cost.
func BenchmarkV4Perf_EmptyCorpus(b *testing.B) {
	wm := emptyWindowMap()
	analyses := collectAnalysesForWindowMap(wm)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, partitions := v4BuildRegionBoundedChainInputs(wm, analyses)
		if len(partitions) == 0 {
			_, _ = v4BuildRegionBoundedChainInputs(wm, analyses)
		}
	}
}

// BenchmarkV4Perf_LiveTreeClaimEvidence runs the full V4 internal
// finding pipeline against the live claim/evidence duplicate that
// the ACT requires remain detectable at its reviewed geometry.
func BenchmarkV4Perf_LiveTreeClaimEvidence(b *testing.B) {
	root := perfRepoRoot()
	leftAbs := repoRel(root, "cmd/leamas/claim_commands.go")
	rightAbs := repoRel(root, "cmd/leamas/evidence_commands.go")

	leftVal, err := analyzeV4AnalyzedFile(leftAbs)
	if err != nil {
		b.Skipf("analyze %s: %v", leftAbs, err)
	}
	rightVal, err := analyzeV4AnalyzedFile(rightAbs)
	if err != nil {
		b.Skipf("analyze %s: %v", rightAbs, err)
	}
	rebaseV4AnalyzedFilePath(&leftVal, "cmd/leamas/claim_commands.go")
	rebaseV4AnalyzedFilePath(&rightVal, "cmd/leamas/evidence_commands.go")

	filesMap := map[string]*v4AnalyzedFile{
		"cmd/leamas/claim_commands.go":    &leftVal,
		"cmd/leamas/evidence_commands.go": &rightVal,
	}
	analysesMap := map[string]*v4FileAnalysis{
		"cmd/leamas/claim_commands.go":    &leftVal.Analysis,
		"cmd/leamas/evidence_commands.go": &rightVal.Analysis,
	}

	windowMap := make(map[string][]rawWindow)
	fingerprintTokens := make(map[string]int)
	for i, ft1 := range []fileTokens{leftVal.FileTokens, rightVal.FileTokens} {
		if len(ft1.tokens) < DefaultConfig().MinTokens {
			continue
		}
		for j := i + 1; j < 2; j++ {
			ft2 := []fileTokens{leftVal.FileTokens, rightVal.FileTokens}[j]
			if len(ft2.tokens) < DefaultConfig().MinTokens {
				continue
			}
			findCommonWindows(ft1, ft2, DefaultConfig(), windowMap, fingerprintTokens)
		}
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, _ = v4BuildInternalFindingsTrace(windowMap, analysesMap, filesMap)
	}
}
