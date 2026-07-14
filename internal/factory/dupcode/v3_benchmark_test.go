// Package dupcode provides benchmarks for v3 algorithm performance.
package dupcode

import (
	"math/rand"
	"testing"
)

// BenchmarkV3_SeedMatchGeneration benchmarks seed match generation.
func BenchmarkV3_SeedMatchGeneration(b *testing.B) {
	// Create windows across multiple files
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

// BenchmarkV3_BuildSeedMatches benchmarks seed match generation with many files.
func BenchmarkV3_BuildSeedMatches(b *testing.B) {
	// Create many windows across many files
	var windows []rawWindow
	paths := []string{"a.go", "b.go", "c.go", "d.go", "e.go", "f.go"}

	for _, path := range paths {
		for i := 0; i < 50; i++ {
			windows = append(windows, rawWindow{
				Path:      path,
				StartPos:  i * 5,
				EndPos:    i*5 + 40,
				StartLine: 10 + i*5,
				EndLine:   50 + i*5,
			})
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buildSeedMatches("benchmark-fp", windows)
	}
}

// BenchmarkV3_BuildChains benchmarks chain construction.
func BenchmarkV3_BuildChains(b *testing.B) {
	// Create many aligned matches
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
		buildChains(matches)
	}
}

// BenchmarkV3_FullCoalescing benchmarks the full v3 coalescing pipeline.
func BenchmarkV3_FullCoalescing(b *testing.B) {
	// Create a realistic window map with many fingerprints and windows
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
		v3CoalesceFindings(windowMap, nil)
	}
}

// BenchmarkV3_LargeClone benchmarks detection of a large clone.
func BenchmarkV3_LargeClone(b *testing.B) {
	// Simulate a large duplicated block with many overlapping windows
	windowMap := make(map[string][]rawWindow)

	// 10 overlapping windows per file
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
		v3CoalesceFindings(windowMap, nil)
	}
}

// BenchmarkV3_ManySmallClones benchmarks detection of many small clones.
func BenchmarkV3_ManySmallClones(b *testing.B) {
	// 100 different small clones
	windowMap := make(map[string][]rawWindow)

	for i := 0; i < 100; i++ {
		fp := string(rune('a' + i%26))
		windowMap[fp] = append(windowMap[fp],
			rawWindow{
				Path:      "a.go",
				StartLine: 10 + i*10,
				EndLine:   50 + i*10,
				StartPos:  i * 10,
				EndPos:    40 + i*10,
			},
			rawWindow{
				Path:      "b.go",
				StartLine: 100 + i*10,
				EndLine:   140 + i*10,
				StartPos:  i * 10,
				EndPos:    40 + i*10,
			},
		)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		v3CoalesceFindings(windowMap, nil)
	}
}

// BenchmarkV3_MultipleOccurrences benchmarks clones with many occurrences.
func BenchmarkV3_MultipleOccurrences(b *testing.B) {
	// Clone appearing in 10 files
	windowMap := make(map[string][]rawWindow)
	fp := "multi-file-clone"

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
	for i := 0; i < b.N; i++ {
		v3CoalesceFindings(windowMap, nil)
	}
}

// BenchmarkV3_DeterministicOutput verifies determinism under stress.
func BenchmarkV3_DeterministicOutput(b *testing.B) {
	// Create window map with random-ish data
	windowMap := make(map[string][]rawWindow)

	for fpIdx := 0; fpIdx < 20; fpIdx++ {
		fp := string(rune('a' + fpIdx))
		for fileIdx := 0; fileIdx < 3; fileIdx++ {
			path := string(rune('a'+fileIdx)) + ".go"
			for winIdx := 0; winIdx < 30; winIdx++ {
				// Slight randomization to test determinism
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
		result := v3CoalesceFindings(windowMap, nil)
		if i == 0 {
			lastResult = result
		} else {
			// Verify determinism
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

// BenchmarkV3_ScalingTokens benchmarks performance with increasing token counts.
func BenchmarkV3_ScalingTokens(b *testing.B) {
	// Test with increasing window counts
	sizes := []int{10, 50, 100, 500}

	for _, size := range sizes {
		b.Run(string(rune('0'+size/100)), func(b *testing.B) {
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
				v3CoalesceFindings(windowMap, nil)
			}
		})
	}
}

// BenchmarkV3_DisjointClones benchmarks handling of disjoint clone regions.
func BenchmarkV3_DisjointClones(b *testing.B) {
	// Create 10 disjoint clone regions
	windowMap := make(map[string][]rawWindow)

	for region := 0; region < 10; region++ {
		fp := string(rune('a' + region))
		offset := region * 1000

		windowMap[fp] = append(windowMap[fp],
			rawWindow{
				Path:      "a.go",
				StartLine: 10 + offset,
				EndLine:   50 + offset,
				StartPos:  offset,
				EndPos:    offset + 40,
			},
			rawWindow{
				Path:      "b.go",
				StartLine: 100 + offset,
				EndLine:   140 + offset,
				StartPos:  offset,
				EndPos:    offset + 40,
			},
		)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		v3CoalesceFindings(windowMap, nil)
	}
}

// BenchmarkV3_RandomizedInput benchmarks with randomized input patterns.
func BenchmarkV3_RandomizedInput(b *testing.B) {
	// Simulate realistic scenario with randomized patterns
	windowMap := make(map[string][]rawWindow)
	r := rand.New(rand.NewSource(42)) // Deterministic seed

	for fpIdx := 0; fpIdx < 30; fpIdx++ {
		fp := string(rune('a' + fpIdx))
		numFiles := 2 + r.Intn(3) // 2-4 files
		for fileIdx := 0; fileIdx < numFiles; fileIdx++ {
			path := string(rune('a'+fileIdx)) + ".go"
			numWins := 10 + r.Intn(40) // 10-50 windows
			for winIdx := 0; winIdx < numWins; winIdx++ {
				startPos := r.Intn(1000)
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
		v3CoalesceFindings(windowMap, nil)
	}
}
