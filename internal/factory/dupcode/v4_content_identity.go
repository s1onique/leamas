package dupcode

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
)

// v4ExactContentKey is the complete identity of one normalized token body.
// Digest and TokenCount are both required; a digest match with a different
// count is an unresolved geometry conflict, never merge evidence.
type v4ExactContentKey struct {
	Digest     string
	TokenCount int
}

// v4PairCloneEvidence is one validated undirected clone edge. It is emitted
// only after both occurrence slices have independently produced the same
// exact content key.
type v4PairCloneEvidence struct {
	ContentKey v4ExactContentKey
	Left       maximalOccurrence
	Right      maximalOccurrence
	LineCount  int
}

const v4ExactSeedWidth = 400

// v4ExactContentKeyForOccurrence hashes exactly the normalized token interval
// represented by occ. Path, line geometry, region ordinal, orientation, and
// map order are not inputs to the digest.
func v4ExactContentKeyForOccurrence(file v4AnalyzedFile, occ maximalOccurrence) (v4ExactContentKey, error) {
	if err := validateV4AnalyzedFile(file); err != nil {
		return v4ExactContentKey{}, err
	}
	if file.FileTokens.path != occ.Path || occ.StartPos < 0 || occ.EndPos < occ.StartPos ||
		occ.EndPos >= len(file.NormalizedTokens) {
		return v4ExactContentKey{}, fmt.Errorf("invalid exact occurrence %s:%d-%d", occ.Path, occ.StartPos, occ.EndPos)
	}
	tokens := file.NormalizedTokens[occ.StartPos : occ.EndPos+1]
	return v4ExactContentKey{
		Digest:     v4ExactNormalizedDigest(tokens),
		TokenCount: len(tokens),
	}, nil
}

// v4ExactNormalizedDigest uses the frozen V4 content projection: every
// ordered canonical seed contributes its exact normalized token values and
// adjacent windows contribute the fixed geometry tuple. This retains the
// published V4 stable-fingerprint oracle while deriving it from the exact
// normalized slice rather than from pair metadata or map state.
func v4ExactNormalizedDigest(tokens []string) string {
	if len(tokens) == 0 {
		return ""
	}
	h := sha256.New()
	h.Write([]byte(algorithmDomain()))
	if len(tokens) < v4ExactSeedWidth {
		h.Write([]byte(strings.Join(tokens, " ")))
		h.Write([]byte("|"))
		return hex.EncodeToString(h.Sum(nil))
	}
	for start := 0; start <= len(tokens)-v4ExactSeedWidth; start++ {
		h.Write([]byte(strings.Join(tokens[start:start+v4ExactSeedWidth], " ")))
		if start > 0 {
			h.Write([]byte(":0:0:399:399"))
		}
		h.Write([]byte("|"))
	}
	return hex.EncodeToString(h.Sum(nil))
}

func v4StableFingerprintForContentKey(key v4ExactContentKey) string {
	return v4SeedFingerprint(key.Digest)
}

func compareV4ContentKeys(left, right v4ExactContentKey) int {
	if left.Digest < right.Digest {
		return -1
	}
	if left.Digest > right.Digest {
		return 1
	}
	if left.TokenCount < right.TokenCount {
		return -1
	}
	if left.TokenCount > right.TokenCount {
		return 1
	}
	return 0
}
