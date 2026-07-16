// Package dupcode provides duplicate code detection for Go source files.
package dupcode

// rawWindow represents a sliding window over a file's tokens.
// StartPos and EndPos are token offsets (0-based).
type rawWindow struct {
	Path      string
	StartLine int // 1-based line number
	EndLine   int
	StartPos  int // 0-based token position
	EndPos    int
}

// tokenRange represents a span in token coordinates.
type tokenRange struct {
	StartPos int
	EndPos   int
}

// coalescedFinding represents a coalesced clone finding.
type coalescedFinding struct {
	Fingerprint       string
	StableFingerprint string
	SeedFingerprint   string // Original seed fingerprint that generated this finding
	Occurrences       []Occurrence
	TokenCount        int
	LineCount         int
}

// windowMatch represents a matched pair of windows from two files.
// DEPRECATED: Use seedMatch from v3.go for algorithm v3.
type windowMatch struct {
	Fingerprint string
	Left        rawWindow
	Right       rawWindow
}

// alignedComponent represents a maximal aligned clone component.
// DEPRECATED: Use cloneChain from v3.go for algorithm v3.
type alignedComponent struct {
	Fingerprint   string
	Occurrences   map[string][]tokenRange
	MaxTokenCount int
	MaxLineCount  int
}

// coalesceFindings is the v4 implementation for finding maximal clones.
// It uses aligned seed-match chaining to produce one maximal finding per clone
// rather than one finding per overlapping MinTokens window.
func coalesceFindings(
	windowMap map[string][]rawWindow,
	fingerprintTokens map[string]int,
) []coalescedFinding {
	// Delegate to v4 algorithm
	return v4CoalesceFindings(windowMap, fingerprintTokens)
}
