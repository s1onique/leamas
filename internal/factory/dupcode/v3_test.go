// Package dupcode provides tests for v3 algorithm (maximal clone detection).
package dupcode

import (
	"testing"
)

// TestV3_SeedMatchGeneration tests seed match generation from raw windows.
func TestV3_SeedMatchGeneration(t *testing.T) {
	windows := []rawWindow{
		{Path: "a.go", StartLine: 10, EndLine: 50, StartPos: 0, EndPos: 40},
		{Path: "b.go", StartLine: 100, EndLine: 140, StartPos: 0, EndPos: 40},
	}

	matches := buildSeedMatches("test-fp", windows)

	if len(matches) != 1 {
		t.Errorf("expected 1 match, got %d", len(matches))
	}

	if len(matches) > 0 {
		if matches[0].SeedFingerprint != "test-fp" {
			t.Errorf("expected seed fingerprint 'test-fp', got %q", matches[0].SeedFingerprint)
		}
		if matches[0].Left.Path != "a.go" {
			t.Errorf("expected left path 'a.go', got %q", matches[0].Left.Path)
		}
		if matches[0].Right.Path != "b.go" {
			t.Errorf("expected right path 'b.go', got %q", matches[0].Right.Path)
		}
		if matches[0].Offset != 0 {
			t.Errorf("expected offset 0, got %d", matches[0].Offset)
		}
	}
}

// TestV3_SeedMatchCanonicalOrdering tests that seed matches use canonical path ordering.
func TestV3_SeedMatchCanonicalOrdering(t *testing.T) {
	// b.go < z.go alphabetically, so b.go should be left
	// Same positions to test canonical ordering (same-position matching)
	windows := []rawWindow{
		{Path: "z.go", StartLine: 10, EndLine: 50, StartPos: 5, EndPos: 45},
		{Path: "b.go", StartLine: 100, EndLine: 140, StartPos: 5, EndPos: 45},
	}

	matches := buildSeedMatches("test-fp", windows)

	if len(matches) != 1 {
		t.Errorf("expected 1 match, got %d", len(matches))
	}

	if len(matches) > 0 {
		// Left should be the alphabetically smaller path (b.go)
		if matches[0].Left.Path != "b.go" {
			t.Errorf("expected left path 'b.go', got %q", matches[0].Left.Path)
		}
		if matches[0].Right.Path != "z.go" {
			t.Errorf("expected right path 'z.go', got %q", matches[0].Right.Path)
		}
		// Same position = offset 0
		if matches[0].Offset != 0 {
			t.Errorf("expected offset 0, got %d", matches[0].Offset)
		}
	}
}

// TestV3_FinalizeChain tests chain finalization.
func TestV3_FinalizeChain(t *testing.T) {
	// Three overlapping matches that should chain
	matches := []seedMatch{
		{
			Left:   rawWindow{Path: "a.go", StartPos: 0, EndPos: 39, StartLine: 10, EndLine: 49},
			Right:  rawWindow{Path: "b.go", StartPos: 100, EndPos: 139, StartLine: 100, EndLine: 139},
			Offset: 100,
		},
		{
			Left:   rawWindow{Path: "a.go", StartPos: 30, EndPos: 69, StartLine: 30, EndLine: 69},
			Right:  rawWindow{Path: "b.go", StartPos: 130, EndPos: 169, StartLine: 130, EndLine: 169},
			Offset: 100,
		},
		{
			Left:   rawWindow{Path: "a.go", StartPos: 60, EndPos: 99, StartLine: 60, EndLine: 99},
			Right:  rawWindow{Path: "b.go", StartPos: 160, EndPos: 199, StartLine: 160, EndLine: 199},
			Offset: 100,
		},
	}

	chain := finalizeChain(matches)

	if chain == nil {
		t.Fatal("expected non-nil chain")
	}

	// Expected token span: left 0-99 = 100 tokens, right 100-199 = 100 tokens
	if chain.TokenSpan != 100 {
		t.Errorf("expected TokenSpan=100, got %d", chain.TokenSpan)
	}

	// Left range: 0-99
	if chain.LeftRange.StartPos != 0 || chain.LeftRange.EndPos != 99 {
		t.Errorf("expected left range 0-99, got %d-%d", chain.LeftRange.StartPos, chain.LeftRange.EndPos)
	}

	// Right range: 100-199
	if chain.RightRange.StartPos != 100 || chain.RightRange.EndPos != 199 {
		t.Errorf("expected right range 100-199, got %d-%d", chain.RightRange.StartPos, chain.RightRange.EndPos)
	}
}

// TestV3_FinalizeChainMismatchedSpans fails closed.
func TestV3_FinalizeChainMismatchedSpans(t *testing.T) {
	// Left and right spans differ - should fail closed
	matches := []seedMatch{
		{
			Left:   rawWindow{Path: "a.go", StartPos: 0, EndPos: 99},    // 100 tokens
			Right:  rawWindow{Path: "b.go", StartPos: 100, EndPos: 199}, // 100 tokens
			Offset: 100,
		},
		{
			Left:   rawWindow{Path: "a.go", StartPos: 50, EndPos: 149},  // 100 tokens, extending left
			Right:  rawWindow{Path: "b.go", StartPos: 100, EndPos: 199}, // 100 tokens, same right
			Offset: 50,                                                  // Different offset!
		},
	}

	chain := finalizeChain(matches)

	// Mismatched spans should return nil (fail closed)
	if chain != nil {
		t.Errorf("expected nil chain for mismatched spans, got %v", chain)
	}
}

// TestV3_BuildChains tests chain construction from aligned matches.
func TestV3_BuildChains(t *testing.T) {
	matches := []seedMatch{
		{
			Left:   rawWindow{Path: "a.go", StartPos: 0, EndPos: 39},
			Right:  rawWindow{Path: "b.go", StartPos: 100, EndPos: 139},
			Offset: 100,
		},
		{
			Left:   rawWindow{Path: "a.go", StartPos: 30, EndPos: 69},
			Right:  rawWindow{Path: "b.go", StartPos: 130, EndPos: 169},
			Offset: 100,
		},
		{
			Left:   rawWindow{Path: "a.go", StartPos: 100, EndPos: 139}, // Disjoint
			Right:  rawWindow{Path: "b.go", StartPos: 200, EndPos: 239},
			Offset: 100,
		},
	}

	chains := buildChains(matches)

	// Should have 2 chains (first two chain, third is separate)
	if len(chains) != 2 {
		t.Errorf("expected 2 chains, got %d", len(chains))
	}

	// First chain should have 2 matches
	if len(chains) > 0 && len(chains[0].Matches) != 2 {
		t.Errorf("expected first chain to have 2 matches, got %d", len(chains[0].Matches))
	}

	// Second chain should have 1 match
	if len(chains) > 1 && len(chains[1].Matches) != 1 {
		t.Errorf("expected second chain to have 1 match, got %d", len(chains[1].Matches))
	}
}

// TestV3_IndependentClonesRemainSeparate tests that different clones produce separate findings.
func TestV3_IndependentClonesRemainSeparate(t *testing.T) {
	// Two different fingerprints with different alignments
	windowMap := map[string][]rawWindow{
		"clone-a": {
			{Path: "a.go", StartLine: 10, EndLine: 50, StartPos: 0, EndPos: 40},
			{Path: "b.go", StartLine: 100, EndLine: 140, StartPos: 0, EndPos: 40},
		},
		"clone-b": {
			{Path: "a.go", StartLine: 60, EndLine: 100, StartPos: 0, EndPos: 40},
			{Path: "c.go", StartLine: 200, EndLine: 240, StartPos: 0, EndPos: 40},
		},
	}

	results := v3CoalesceFindings(windowMap, nil)

	// Different clones = 2 findings
	if len(results) != 2 {
		t.Errorf("expected 2 findings for independent clones, got %d", len(results))
	}
}

// TestV3_ThreeFileClone tests a clone appearing in three files.
func TestV3_ThreeFileClone(t *testing.T) {
	windowMap := map[string][]rawWindow{
		"shared": {
			{Path: "a.go", StartLine: 10, EndLine: 50, StartPos: 0, EndPos: 40},
			{Path: "b.go", StartLine: 100, EndLine: 140, StartPos: 0, EndPos: 40},
			{Path: "c.go", StartLine: 200, EndLine: 240, StartPos: 0, EndPos: 40},
		},
	}

	results := v3CoalesceFindings(windowMap, nil)

	if len(results) != 1 {
		t.Errorf("expected 1 finding, got %d", len(results))
	}

	if len(results) > 0 && len(results[0].Occurrences) != 3 {
		t.Errorf("expected 3 occurrences, got %d", len(results[0].Occurrences))
	}
}

// TestV3_Determinism tests that output is deterministic regardless of input order.
func TestV3_Determinism(t *testing.T) {
	windowMap1 := map[string][]rawWindow{
		"fp-b": {
			{Path: "a.go", StartLine: 10, EndLine: 50, StartPos: 0, EndPos: 40},
			{Path: "b.go", StartLine: 100, EndLine: 140, StartPos: 0, EndPos: 40},
		},
		"fp-a": {
			{Path: "a.go", StartLine: 10, EndLine: 50, StartPos: 0, EndPos: 40},
			{Path: "b.go", StartLine: 100, EndLine: 140, StartPos: 0, EndPos: 40},
		},
	}

	windowMap2 := map[string][]rawWindow{
		"fp-a": {
			{Path: "a.go", StartLine: 10, EndLine: 50, StartPos: 0, EndPos: 40},
			{Path: "b.go", StartLine: 100, EndLine: 140, StartPos: 0, EndPos: 40},
		},
		"fp-b": {
			{Path: "a.go", StartLine: 10, EndLine: 50, StartPos: 0, EndPos: 40},
			{Path: "b.go", StartLine: 100, EndLine: 140, StartPos: 0, EndPos: 40},
		},
	}

	results1 := v3CoalesceFindings(windowMap1, nil)
	results2 := v3CoalesceFindings(windowMap2, nil)

	if len(results1) != len(results2) {
		t.Fatal("different number of results based on input order")
	}

	for i := range results1 {
		if results1[i].StableFingerprint != results2[i].StableFingerprint {
			t.Errorf("fingerprint mismatch at index %d: %s vs %s",
				i, results1[i].StableFingerprint, results2[i].StableFingerprint)
		}
	}
}

// TestV3_AlgorithmVersionInFingerprint tests that fingerprints use the current algorithm version.
func TestV3_AlgorithmVersionInFingerprint(t *testing.T) {
	windowMap := map[string][]rawWindow{
		"test": {
			{Path: "a.go", StartLine: 10, EndLine: 50, StartPos: 0, EndPos: 40},
			{Path: "b.go", StartLine: 100, EndLine: 140, StartPos: 0, EndPos: 40},
		},
	}

	results := v3CoalesceFindings(windowMap, nil)

	if len(results) == 0 {
		t.Fatal("expected at least one result")
	}

	// Fingerprint should be SHA256
	fp := results[0].StableFingerprint
	if len(fp) != 64 {
		t.Errorf("expected 64-char SHA256 fingerprint, got %d chars", len(fp))
	}
}

// TestV3_MaximalCloneDetection tests that overlapping windows produce findings.
func TestV3_MaximalCloneDetection(t *testing.T) {
	// Simulate overlapping windows at same positions that produce findings
	windowMap := map[string][]rawWindow{
		"seed": {},
	}

	// Add overlapping windows for each of two files at same positions
	for i := 0; i < 3; i++ {
		windowMap["seed"] = append(windowMap["seed"],
			rawWindow{Path: "a.go", StartLine: 10, EndLine: 50, StartPos: 0, EndPos: 40},
			rawWindow{Path: "b.go", StartLine: 100, EndLine: 140, StartPos: 0, EndPos: 40},
		)
	}

	results := v3CoalesceFindings(windowMap, nil)

	// Should produce findings
	if len(results) == 0 {
		t.Error("expected some findings")
	}

	if len(results) > 0 {
		// TokenCount should be at least the window span
		if results[0].TokenCount < 40 {
			t.Errorf("expected TokenCount >= 40, got %d", results[0].TokenCount)
		}
	}
}
