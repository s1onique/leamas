// Package dupcode provides duplicate code detection for Go source files.
package dupcode

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
)

// Algorithm v3: Deterministic maximal clone detection via aligned seed-match chaining.
//
// Key changes from v2:
// - Explicit seedMatch representation with relative token offset
// - Chain matches when both sides advance consistently
// - Derive exact maximal token/line ranges from final chains
// - Stable fingerprint = domain:seedFP:canonicalPathSet
// - Deterministic output ordering

// seedMatch represents an aligned pair of token windows from the same seed fingerprint.
type seedMatch struct {
	SeedFingerprint string    // Original seed fingerprint that generated this match
	Left            rawWindow // Left occurrence (lexicographically smaller path)
	Right           rawWindow // Right occurrence (lexicographically larger path)
	Offset          int       // right.StartPos - left.StartPos (canonical order)
}

// cloneChain represents a chain of consecutive aligned seed matches.
type cloneChain struct {
	Matches    []seedMatch
	TokenSpan  int // right.EndPos - left.StartPos + 1 (same for both sides)
	LineSpan   int // max line span across all matches
	PathSet    string
	LeftRange  tokenRange
	RightRange tokenRange
	Offset     int // canonical offset for this chain (used in fingerprint)
}

// chainKey identifies matches that can be chained together.
// Includes SeedFingerprint to ensure same-clone matches chain.
// Note: Different clones at same file+offset remain separate.
type chainKey struct {
	SeedFingerprint string
	LeftPath        string
	RightPath       string
	Offset          int
}

// v3SeedFingerprint generates the stable fingerprint for v3 algorithm.
func v3SeedFingerprint(tokenFP, pathSet string) string {
	data := fmt.Sprintf("leamas-dupcode-v%d:", AlgorithmVersion) + tokenFP + ":" + pathSet
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])
}
