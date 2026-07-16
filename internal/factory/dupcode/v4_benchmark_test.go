// Package dupcode provides benchmarks for v4 algorithm performance.
package dupcode

import (
	"math/rand"
	"strconv"
	"testing"
)

// BenchmarkV4_SeedMatchGeneration benchmarks seed match generation.
func BenchmarkV4_SeedMatchGeneration(b *testing.B) {
	var windows []rawWindow
	paths := []string{"a.go", "b.go", "c.go", "d.go", "e.go"}
	for _, path := range paths {
		for i := 0; i < 100; i++ {
			windows = append(windows, rawWindow{
				Path:      path,
				StartLine: 10 + i*5,
				EndLine:   50 + i*5,
				StartPos:  i * 5,
				EndPos:    40 + i*5,
			})
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buildSeedMatches("benchmark-fp", windows)
	}
}

// BenchmarkV4_BuildChains benchmarks chain construction.
func BenchmarkV4_BuildChains(b *testing.B) {
	var matches []seedMatch
	offset := 100
	for i := 0; i < 1000; i++ {
		matches = append(matches, seedMatch{
			Left: rawWindow{
				Path:     "a.go",
				StartPos: i,
				EndPos:   i + 40,
			},
			Right: rawWindow{
				Path:     "b.go",
				StartPos: i + offset,
				EndPos:   i + offset + 40,
			},
			Offset: offset,
		})
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		v4BuildChainsWithPartitioning(matches)
	}
}

// BenchmarkV4_FullCoalescing benchmarks the full v4 coalescing pipeline.
func BenchmarkV4_FullCoalescing(b *testing.B) {
	windowMap := make(map[string][]rawWindow)

	// Add multiple fingerprints simulating different clone types
	for fpIdx := 0; fpIdx < 50; fpIdx++ {
		fp := string(rune('a'+fpIdx%26)) + string(rune('0'+fpIdx/26))
		for fileIdx := 0; fileIdx < 3; fileIdx++ {
			path := string(rune('a'+fileIdx)) + ".go"
			for winIdx := 0; winIdx < 20; winIdx++ {
				startPos := winIdx * 5
				windowMap[fp] = append(windowMap[fp], rawWindow{
					Path:      path,
					StartLine: 10 + winIdx*5,
					EndLine:   50 + winIdx*5,
					StartPos:  startPos,
					EndPos:    startPos + 40,
				})
			}
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		v4CoalesceFindings(windowMap, nil)
	}
}

// BenchmarkV4_LargeClone benchmarks detection of a large clone with many overlapping windows.
func BenchmarkV4_LargeClone(b *testing.B) {
	windowMap := make(map[string][]rawWindow)

	// 10 overlapping windows per file - simulates 500-token clone with 400 MinTokens
	for i := 0; i < 10; i++ {
		fp := "large-clone"
		windowMap[fp] = append(windowMap[fp],
			rawWindow{
				Path:      "file1.go",
				StartLine: 10 + i*5,
				EndLine:   50 + i*5,
				StartPos:  i * 5,
				EndPos:    40 + i*5,
			},
			rawWindow{
				Path:      "file2.go",
				StartLine: 100 + i*5,
				EndLine:   140 + i*5,
				StartPos:  i * 5,
				EndPos:    40 + i*5,
			},
		)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		v4CoalesceFindings(windowMap, nil)
	}
}

// BenchmarkV4_TwoFileClone benchmarks detection of a 500-token two-file clone.
func BenchmarkV4_TwoFileClone(b *testing.B) {
	windowMap := make(map[string][]rawWindow)

	// 500-token clone body with 400 MinTokens threshold
	// Creates ~101 overlapping windows that should coalesce to 1 finding
	for i := 0; i < 101; i++ {
		fp := "clone-body"
		windowMap[fp] = append(windowMap[fp],
			rawWindow{
				Path:      "a.go",
				StartLine: 1 + i*5,
				EndLine:   100 + i*5,
				StartPos:  i * 5,
				EndPos:    400 + i*5,
			},
			rawWindow{
				Path:      "b.go",
				StartLine: 1 + i*5,
				EndLine:   100 + i*5,
				StartPos:  i * 5,
				EndPos:    400 + i*5,
			},
		)
	}

	b.ResetTimer()
	var result []coalescedFinding
	for i := 0; i < b.N; i++ {
		result = v4CoalesceFindings(windowMap, nil)
	}

	// Record metrics
	b.StopTimer()
	b.ReportMetric(float64(len(result)), "findings")
}

// BenchmarkV4_ThreeFileClone benchmarks detection of a clone in three files.
func BenchmarkV4_ThreeFileClone(b *testing.B) {
	windowMap := make(map[string][]rawWindow)

	// Clone appearing in 3 files
	for i := 0; i < 5; i++ {
		fp := "three-file-clone"
		for _, path := range []string{"a.go", "b.go", "c.go"} {
			windowMap[fp] = append(windowMap[fp], rawWindow{
				Path:      path,
				StartLine: 10 + i*5,
				EndLine:   50 + i*5,
				StartPos:  i * 5,
				EndPos:    40 + i*5,
			})
		}
	}

	b.ResetTimer()
	var result []coalescedFinding
	for i := 0; i < b.N; i++ {
		result = v4CoalesceFindings(windowMap, nil)
	}

	b.StopTimer()
	b.ReportMetric(float64(len(result)), "findings")
	if len(result) > 0 {
		b.ReportMetric(float64(len(result[0].Occurrences)), "occurrences")
	}
}

// BenchmarkV4_RepeatedSameFileClone benchmarks clones repeated in the same file.
func BenchmarkV4_RepeatedSameFileClone(b *testing.B) {
	windowMap := make(map[string][]rawWindow)

	// Clone appearing twice in one file, once in another
	fp := "repeated-clone"
	for i := 0; i < 5; i++ {
		// File A: clone at two positions
		windowMap[fp] = append(windowMap[fp],
			rawWindow{
				Path:      "a.go",
				StartLine: 10 + i*5,
				EndLine:   50 + i*5,
				StartPos:  i * 5,
				EndPos:    40 + i*5,
			},
			rawWindow{
				Path:      "a.go",
				StartLine: 100 + i*5,
				EndLine:   140 + i*5,
				StartPos:  1000 + i*5,
				EndPos:    1040 + i*5,
			},
			// File B: clone at one position
			rawWindow{
				Path:      "b.go",
				StartLine: 10 + i*5,
				EndLine:   50 + i*5,
				StartPos:  i * 5,
				EndPos:    40 + i*5,
			},
		)
	}

	b.ResetTimer()
	var result []coalescedFinding
	for i := 0; i < b.N; i++ {
		result = v4CoalesceFindings(windowMap, nil)
	}

	b.StopTimer()
	b.ReportMetric(float64(len(result)), "findings")
	if len(result) > 0 {
		b.ReportMetric(float64(len(result[0].Occurrences)), "occurrences")
	}
}

// BenchmarkV4_ManyOccurrences benchmarks clones with many file occurrences.
func BenchmarkV4_ManyOccurrences(b *testing.B) {
	windowMap := make(map[string][]rawWindow)
	fp := "multi-file-clone"

	// Clone appearing in 10 files
	for fileIdx := 0; fileIdx < 10; fileIdx++ {
		path := string(rune('a'+fileIdx)) + ".go"
		for winIdx := 0; winIdx < 5; winIdx++ {
			windowMap[fp] = append(windowMap[fp], rawWindow{
				Path:      path,
				StartLine: 10 + winIdx*5,
				EndLine:   50 + winIdx*5,
				StartPos:  winIdx * 5,
				EndPos:    40 + winIdx*5,
			})
		}
	}

	b.ResetTimer()
	var result []coalescedFinding
	for i := 0; i < b.N; i++ {
		result = v4CoalesceFindings(windowMap, nil)
	}

	b.StopTimer()
	b.ReportMetric(float64(len(result)), "findings")
	if len(result) > 0 {
		b.ReportMetric(float64(len(result[0].Occurrences)), "occurrences")
	}
}

// BenchmarkV4_HighlyRepetitive benchmarks worst-case matching with highly repetitive input.
func BenchmarkV4_HighlyRepetitive(b *testing.B) {
	windowMap := make(map[string][]rawWindow)
	r := rand.New(rand.NewSource(42))

	// Simulate worst case: many windows from same fingerprint across files
	numFiles := 5
	numWindows := 20

	for fileIdx := 0; fileIdx < numFiles; fileIdx++ {
		path := string(rune('a'+fileIdx)) + ".go"
		for winIdx := 0; winIdx < numWindows; winIdx++ {
			startPos := winIdx*5 + r.Intn(2)
			windowMap["shared-fp"] = append(windowMap["shared-fp"], rawWindow{
				Path:      path,
				StartLine: 10 + winIdx*5,
				EndLine:   50 + winIdx*5,
				StartPos:  startPos,
				EndPos:    startPos + 40,
			})
		}
	}

	var peakMatches int
	b.ResetTimer()
	var result []coalescedFinding
	for i := 0; i < b.N; i++ {
		result = v4CoalesceFindings(windowMap, nil)
		if len(result) > 0 {
			peakMatches = len(result[0].Occurrences)
		}
	}

	b.StopTimer()
	b.ReportMetric(float64(len(result)), "findings")
	b.ReportMetric(float64(peakMatches), "occurrences")
}

// BenchmarkV4_DeterministicOutput verifies determinism under stress.
func BenchmarkV4_DeterministicOutput(b *testing.B) {
	windowMap := make(map[string][]rawWindow)

	for fpIdx := 0; fpIdx < 20; fpIdx++ {
		fp := string(rune('a' + fpIdx))
		for fileIdx := 0; fileIdx < 3; fileIdx++ {
			path := string(rune('a'+fileIdx)) + ".go"
			for winIdx := 0; winIdx < 30; winIdx++ {
				offset := (fpIdx*7 + fileIdx*3 + winIdx*11) % 100
				windowMap[fp] = append(windowMap[fp], rawWindow{
					Path:      path,
					StartLine: 10 + winIdx*5,
					EndLine:   50 + winIdx*5,
					StartPos:  offset,
					EndPos:    offset + 40,
				})
			}
		}
	}

	var lastResult []coalescedFinding
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result := v4CoalesceFindings(windowMap, nil)
		if i == 0 {
			lastResult = result
		} else {
			if len(result) != len(lastResult) {
				b.Fatal("non-deterministic result length")
			}
			for j := range result {
				if result[j].StableFingerprint != lastResult[j].StableFingerprint {
					b.Fatal("non-deterministic fingerprint")
				}
			}
		}
	}
}

// BenchmarkV4_ScalingTokens benchmarks performance with increasing token counts.
func BenchmarkV4_ScalingTokens(b *testing.B) {
	sizes := []int{10, 50, 100, 500}

	for _, size := range sizes {
		b.Run(strconv.Itoa(size), func(b *testing.B) {
			windowMap := make(map[string][]rawWindow)

			for i := 0; i < size; i++ {
				windowMap["fp"] = append(windowMap["fp"],
					rawWindow{
						Path:      "a.go",
						StartLine: 10 + i*5,
						EndLine:   50 + i*5,
						StartPos:  i * 5,
						EndPos:    40 + i*5,
					},
					rawWindow{
						Path:      "b.go",
						StartLine: 100 + i*5,
						EndLine:   140 + i*5,
						StartPos:  i * 5,
						EndPos:    40 + i*5,
					},
				)
			}

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				v4CoalesceFindings(windowMap, nil)
			}
		})
	}
}
