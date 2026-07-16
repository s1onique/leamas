// Package dupcode provides duplicate code detection for Go source files.
package dupcode

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
)

// Algorithm v4: Truthful maximal clone detection.
//
// Key changes from v3:
// - Chain partitioning excludes SeedFingerprint to allow chaining across
//   consecutive sliding windows with different fingerprints within the same clone.
// - Stable fingerprint uses complete normalized token sequence content hash.
// - Matches by content (normalized tokens), not absolute position.

// seedMatch represents an aligned pair of token windows from the same seed fingerprint.
type seedMatch struct {
	SeedFingerprint string    // Original seed fingerprint that generated this match
	Left            rawWindow // Left occurrence (lexicographically smaller path)
	Right           rawWindow // Right occurrence (lexicographically larger path)
	Offset          int       // right.StartPos - left.StartPos (canonical order)
}

// cloneChain represents a chain of consecutive aligned seed matches.
type cloneChain struct {
	Matches     []seedMatch
	TokenSpan   int // right.EndPos - left.StartPos + 1 (same for both sides)
	LineSpan    int // max line span across all matches
	PathSet     string
	LeftRange   tokenRange
	RightRange  tokenRange
	Offset      int    // canonical offset for this chain
	ContentHash string // SHA256 of complete normalized token sequence
}

// chainKey identifies matches that can be chained together.
// V4: Excludes SeedFingerprint to allow chaining across different fingerprints
// within the same clone body. Only (leftPath, rightPath, offset) matters.
type chainKey struct {
	LeftPath  string
	RightPath string
	Offset    int
}

// algorithmDomain returns the domain string for fingerprinting.
func algorithmDomain() string {
	return fmt.Sprintf("leamas-dupcode-v%d", AlgorithmVersion)
}

// v4SeedFingerprint generates the stable fingerprint for v4 algorithm.
// V4: Excludes pathSet from fingerprint to enable N-way merging.
// Two pairwise chains with the same content hash but different path pairs
// will have the same stable fingerprint and can be merged.
func v4SeedFingerprint(contentHash string) string {
	data := algorithmDomain() + ":" + contentHash
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])
}

// computeContentHash computes a SHA256 hash of the ordered chain.
// V4: Uses ordered list of (seedFingerprint, relative advancement) tuples.
// This preserves:
// - Order: through ordered chain traversal
// - Multiplicity: each window contributes to the hash
// - Relative advancement: gap between consecutive seeds on each side
// - Overlap: overlap length between consecutive seeds
// This ensures chains with different geometry have different hashes.
func computeContentHash(matches []seedMatch) string {
	if len(matches) == 0 {
		return ""
	}

	h := sha256.New()
	h.Write([]byte(algorithmDomain()))

	for i, m := range matches {
		h.Write([]byte(m.SeedFingerprint))

		if i > 0 {
			prev := matches[i-1]
			// Left advancement: gap between previous end and current start
			leftAdv := m.Left.StartPos - prev.Left.EndPos - 1
			if leftAdv < 0 {
				leftAdv = 0 // overlapping
			}
			// Right advancement
			rightAdv := m.Right.StartPos - prev.Right.EndPos - 1
			if rightAdv < 0 {
				rightAdv = 0
			}
			// Overlap between consecutive seeds
			overlapLeft := prev.Left.EndPos - m.Left.StartPos + 1
			if overlapLeft < 0 {
				overlapLeft = 0
			}
			overlapRight := prev.Right.EndPos - m.Right.StartPos + 1
			if overlapRight < 0 {
				overlapRight = 0
			}

			h.Write([]byte(fmt.Sprintf(":%d:%d:%d:%d", leftAdv, rightAdv, overlapLeft, overlapRight)))
		}
		h.Write([]byte("|"))
	}

	return hex.EncodeToString(h.Sum(nil))
}

// hashTokens computes a SHA256 hash of normalized tokens for content identity.
func hashTokens(tokens []byte) string {
	h := sha256.New()
	h.Write([]byte(algorithmDomain()))
	h.Write(tokens)
	return hex.EncodeToString(h.Sum(nil))
}

// v3SeedFingerprint generates the stable fingerprint for v3 algorithm.
// DEPRECATED: Kept for backward compatibility with legacy code.
// Use v4SeedFingerprint for v4 algorithm.
func v3SeedFingerprint(tokenFP, pathSet string) string {
	data := algorithmDomain() + ":" + tokenFP + ":" + pathSet
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])
}
