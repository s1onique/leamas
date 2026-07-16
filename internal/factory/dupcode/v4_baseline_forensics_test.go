// Package dupcode provides executable baseline-forensics oracle
// tests that operate on the actual production source files
// cmd/leamas/claim_commands.go and cmd/leamas/evidence_commands.go.
//
// The historical baseline (pre-canonical-content materializer)
// reported three distinct findings:
//
//   - 877 tokens / claim_commands.go:188-340 + evidence_commands.go:230-382
//   - 514 tokens / claim_commands.go:87-178  + evidence_commands.go:132-222
//   - (canonical) 504 tokens / claim_commands.go:268-340 + evidence_commands.go:310-382
//
// CORRECTION02 classified the prior 514-token finding as "either
// non-equal normalized content OR obsolete chain geometry". That
// is an unresolved disjunction, not an independent classification.
// CORRECTION03 replaces the disjunction with one executable result
// per historical range by computing the actual normalized token
// slice, the SHA-256 digest, the inclusive token positions, and the
// owning executable region for each historical public line range.
//
// All evidence in this file is regenerated against the live tree
// (NOT a stale digest). The test-owned oracle uses the production
// scanner/parser via analyzeV4AnalyzedFile and an independent
// SHA-256 digest so a defect in any single component cannot make
// the test pass silently.
package dupcode

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
	"testing"
)

// forensicsCase names one historical public line range pair and the
// expected classification under the corrected algorithm. The
// classification labels are part of the closure contract: every
// historical range gets exactly one label.
type forensicsCase struct {
	Name           string
	LeftPath       string
	LeftStartLine  int
	LeftEndLine    int
	RightPath      string
	RightStartLine int
	RightEndLine   int

	// Classification is the single, unambiguous disposition that the
	// test proves for this case. It must be one of:
	//   - "valid canonical exact duplicate"
	//   - "invalid because geometry crosses executable-region ownership"
	//   - "invalid because left/right normalized content differs"
	Classification string
}

// forensicsDisposition is the test-owned executable evidence for one
// historical public line range pair.
type forensicsDisposition struct {
	Name string

	LeftFile         *v4AnalyzedFile
	RightFile        *v4AnalyzedFile
	LeftStartPos     int
	LeftEndPos       int
	RightStartPos    int
	RightEndPos      int
	LeftStartLine    int
	LeftEndLine      int
	RightStartLine   int
	RightEndLine     int
	LeftTokenCount   int
	RightTokenCount  int
	LeftDigest       string
	RightDigest      string
	LeftRegionOwners []v4SyntaxRegionID
	RightRegionOwner []v4SyntaxRegionID

	// Classification is the single, unambiguous disposition recorded
	// by the test for this historical range.
	Classification string
}

// forensicsOracle returns the executable disposition for one
// historical public line range pair. The function maps each public
// line range to its inclusive token positions using the production
// scanner/parser, computes normalized token slices, computes
// SHA-256 digests via an independent oracle, and walks the
// TokenOwner array to record every executable region that owns any
// token in the slice.
//
// forensicsOracle never assumes a result: every classification is
// computed from the actual token stream of the production files.
func forensicsOracle(t *testing.T, c forensicsCase) forensicsDisposition {
	t.Helper()
	root := deltaRepoRoot(t)

	leftPath := c.LeftPath
	rightPath := c.RightPath
	leftAbs := repoRel(root, leftPath)
	rightAbs := repoRel(root, rightPath)

	leftFile, err := analyzeV4AnalyzedFile(leftAbs)
	if err != nil {
		t.Fatalf("analyze %s: %v", leftPath, err)
	}
	rightFile, err := analyzeV4AnalyzedFile(rightAbs)
	if err != nil {
		t.Fatalf("analyze %s: %v", rightPath, err)
	}
	// Rebase to the requested public path so downstream maps agree.
	rebaseV4AnalyzedFilePath(&leftFile, leftPath)
	rebaseV4AnalyzedFilePath(&rightFile, rightPath)

	d := forensicsDisposition{
		Name:           c.Name,
		LeftFile:       &leftFile,
		RightFile:      &rightFile,
		LeftStartLine:  c.LeftStartLine,
		LeftEndLine:    c.LeftEndLine,
		RightStartLine: c.RightStartLine,
		RightEndLine:   c.RightEndLine,
	}

	d.LeftStartPos, d.LeftEndPos = mapLineRangeToTokenRange(
		leftFile.Analysis, c.LeftStartLine, c.LeftEndLine)
	d.RightStartPos, d.RightEndPos = mapLineRangeToTokenRange(
		rightFile.Analysis, c.RightStartLine, c.RightEndLine)

	leftTokens := leftFile.NormalizedTokens[d.LeftStartPos : d.LeftEndPos+1]
	rightTokens := rightFile.NormalizedTokens[d.RightStartPos : d.RightEndPos+1]
	d.LeftTokenCount = len(leftTokens)
	d.RightTokenCount = len(rightTokens)
	// Independent SHA-256 oracle so a defect in any production
	// digest implementation cannot make the test pass silently.
	// The "first-token|sha256" form also rules out an accidental
	// zero-prefix collision between two distinct token sequences.
	d.LeftDigest = leftTokens[0] + "|" + sha256Hex(strings.Join(leftTokens, " "))
	d.RightDigest = rightTokens[0] + "|" + sha256Hex(strings.Join(rightTokens, " "))

	d.LeftRegionOwners = collectOwnersInRange(
		leftFile.Analysis.TokenOwner, d.LeftStartPos, d.LeftEndPos)
	d.RightRegionOwner = collectOwnersInRange(
		rightFile.Analysis.TokenOwner, d.RightStartPos, d.RightEndPos)

	return d
}

// mapLineRangeToTokenRange returns the inclusive [start, end] token
// positions that cover every token whose Lines[i] is in
// [startLine, endLine]. The inclusive ends are chosen so the caller
// can use the slice [StartPos:EndPos+1] directly. The function
// returns (-1, -1) when the line range covers zero tokens.
func mapLineRangeToTokenRange(analysis v4FileAnalysis, startLine, endLine int) (int, int) {
	if len(analysis.Lines) == 0 {
		return -1, -1
	}
	startPos := -1
	endPos := -1
	for i, line := range analysis.Lines {
		if line < startLine || line > endLine {
			continue
		}
		if startPos == -1 {
			startPos = i
		}
		endPos = i
	}
	return startPos, endPos
}

// collectOwnersInRange returns the sorted, deduplicated list of
// TokenOwner values for tokens in the inclusive [start, end]
// interval. The empty owner (Path=="") means "no executable region
// owner" and is preserved in the output so the caller can detect
// unowned-token leaks.
func collectOwnersInRange(owners []v4SyntaxRegionID, start, end int) []v4SyntaxRegionID {
	if start < 0 || end < start || end >= len(owners) {
		return nil
	}
	seen := make(map[v4SyntaxRegionID]bool)
	out := make([]v4SyntaxRegionID, 0, end-start+1)
	for i := start; i <= end; i++ {
		if seen[owners[i]] {
			continue
		}
		seen[owners[i]] = true
		out = append(out, owners[i])
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Path != out[j].Path {
			return out[i].Path < out[j].Path
		}
		return out[i].Ordinal < out[j].Ordinal
	})
	return out
}

// sha256Hex returns the lowercase hex SHA-256 of data.
func sha256Hex(data string) string {
	sum := sha256.Sum256([]byte(data))
	return hex.EncodeToString(sum[:])
}

// classifyFromDisposition returns the single, executable disposition
// for one historical range. The classification is computed from the
// recorded disposition so the closure report can quote it directly.
func classifyFromDisposition(d forensicsDisposition) string {
	// Any token in the slice whose TokenOwner is empty means the
	// public range crosses into unowned top-level tokens (package,
	// var, const, or inter-function gap). That is a geometry conflict
	// under the V4 region-aware architecture: the public line range
	// cannot be a valid single-region clone.
	for _, owner := range d.LeftRegionOwners {
		if owner.Path == "" {
			return "invalid because geometry crosses executable-region ownership"
		}
	}
	for _, owner := range d.RightRegionOwner {
		if owner.Path == "" {
			return "invalid because geometry crosses executable-region ownership"
		}
	}
	// If the left slice spans more than one executable region, the
	// public range is a multi-region span and cannot be a valid
	// single-region clone.
	if len(d.LeftRegionOwners) > 1 {
		return "invalid because geometry crosses executable-region ownership"
	}
	if len(d.RightRegionOwner) > 1 {
		return "invalid because geometry crosses executable-region ownership"
	}
	// Both slices lie in a single executable region but their
	// normalized content differs.
	if d.LeftDigest != d.RightDigest {
		return "invalid because left/right normalized content differs"
	}
	// Both slices lie in a single executable region and their
	// normalized content matches: the range is a valid canonical
	// exact duplicate.
	return "valid canonical exact duplicate"
}

// forensicsCases returns the canonical historical public-line-range
// table that the closure report quotes. Each case is a separate
// historical finding; each MUST end up classified to exactly one
// label.
func forensicsCases() []forensicsCase {
	return []forensicsCase{
		{
			Name:           "877-historical",
			LeftPath:       "cmd/leamas/claim_commands.go",
			LeftStartLine:  188,
			LeftEndLine:    340,
			RightPath:      "cmd/leamas/evidence_commands.go",
			RightStartLine: 230,
			RightEndLine:   382,
			Classification: "invalid because geometry crosses executable-region ownership",
		},
		{
			Name:           "514-historical",
			LeftPath:       "cmd/leamas/claim_commands.go",
			LeftStartLine:  87,
			LeftEndLine:    178,
			RightPath:      "cmd/leamas/evidence_commands.go",
			RightStartLine: 132,
			RightEndLine:   222,
			Classification: "invalid because geometry crosses executable-region ownership",
		},
		{
			Name:           "504-current-canonical",
			LeftPath:       "cmd/leamas/claim_commands.go",
			LeftStartLine:  268,
			LeftEndLine:    340,
			RightPath:      "cmd/leamas/evidence_commands.go",
			RightStartLine: 310,
			RightEndLine:   382,
			Classification: "valid canonical exact duplicate",
		},
	}
}

// TestV4BaselineForensics_877_IsCrossRegionGeometry executes the
// forensics oracle against the historical 877-token public line
// range. The recorded classification proves the prior 877-token
// finding cannot be a valid single-region clone in the current
// production source.
func TestV4BaselineForensics_877_IsCrossRegionGeometry(t *testing.T) {
	c := forensicsCases()[0]
	d := forensicsOracle(t, c)
	got := classifyFromDisposition(d)
	if got != c.Classification {
		t.Fatalf("877 historical classification mismatch:\n  recorded=%q\n  expected=%q\n  disposition=%s",
			got, c.Classification, formatDisposition(d))
	}
	t.Logf("877 forensic disposition: %s", formatDisposition(d))
}

// TestV4BaselineForensics_514_IsCrossRegionGeometry executes the
// forensics oracle against the historical 514-token public line
// range. The recorded classification proves the prior 514-token
// finding cannot be a valid single-region clone in the current
// production source.
func TestV4BaselineForensics_514_IsCrossRegionGeometry(t *testing.T) {
	c := forensicsCases()[1]
	d := forensicsOracle(t, c)
	got := classifyFromDisposition(d)
	if got != c.Classification {
		t.Fatalf("514 historical classification mismatch:\n  recorded=%q\n  expected=%q\n  disposition=%s",
			got, c.Classification, formatDisposition(d))
	}
	t.Logf("514 forensic disposition: %s", formatDisposition(d))
}

// formatDisposition returns a deterministic, audit-friendly summary
// of a forensicsDisposition for test logs and close reports.
func formatDisposition(d forensicsDisposition) string {
	leftDigest := d.LeftDigest
	if len(leftDigest) > 16 {
		leftDigest = leftDigest[:16]
	}
	rightDigest := d.RightDigest
	if len(rightDigest) > 16 {
		rightDigest = rightDigest[:16]
	}
	return fmt.Sprintf(
		"name=%s left=%s:%d-%d -> [%d,%d](tokens=%d,digest=%s) right=%s:%d-%d -> [%d,%d](tokens=%d,digest=%s) left_owners=%v right_owners=%v",
		d.Name,
		d.LeftFile.FileTokens.path, d.LeftStartLine, d.LeftEndLine,
		d.LeftStartPos, d.LeftEndPos, d.LeftTokenCount, leftDigest,
		d.RightFile.FileTokens.path, d.RightStartLine, d.RightEndLine,
		d.RightStartPos, d.RightEndPos, d.RightTokenCount, rightDigest,
		d.LeftRegionOwners, d.RightRegionOwner,
	)
}

// repoRel joins root with rel. Both inputs are already absolute or
// already-relative paths; the join is left to filepath so callers
// can pass either form.
func repoRel(root, rel string) string {
	if strings.HasPrefix(rel, "/") {
		return rel
	}
	return joinPath(root, rel)
}

func joinPath(a, b string) string {
	if a == "" {
		return b
	}
	if b == "" {
		return a
	}
	if a[len(a)-1] == '/' {
		return a + b
	}
	return a + "/" + b
}
