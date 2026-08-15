// SPDX-License-Identifier: Apache-2.0

// Package digest: evidence_budget.go is the bounded evidence renderer
// introduced by ACT-LEAMAS-TARGETED-DIGEST-RECURSIVE-EVIDENCE-GUARD01.
//
// The renderer enforces the recursive-evidence invariant:
//
//	size(D[n+1]) must be bounded independently of size(D[n]).
//
// Every changed file is classified by classifyFileEvidence into one
// of ClassNormal, ClassBoundedBody, ClassBoundedSelfOutput, or
// ClassBoundedRecursive. Classification is content-aware; filename
// alone is NEVER authoritative. The current Leamas digest artifact
// is recognised by its canonical contract marker
// (LEAMAS_TARGETED_DIGEST_CONTRACT_VERSION: 3) and the legacy
// `# Targeted digest` heading. Path signal participates only as a
// supporting heuristic, never alone.
//
// Budgets are deterministic and small:
//
//	MaxPerFileBytes        =  64 KiB  per-file rendered budget
//	MaxPerLineBytes        =   4 KiB  per physical line rendered budget
//	MaxTotalRenderBytes    = 256 KiB  total rendered evidence budget
//	MaxFileSizeForFull     =   1 MiB  threshold for "large file" gate
//
// A file larger than MaxFileSizeForFull, a file with a physical
// line longer than MaxPerLineBytes, or a file recognised as a
// recursive digest artifact is always rendered via
// ClassBoundedBody or ClassBoundedRecursive. The output path of
// the digest itself is excluded via ClassBoundedSelfOutput.
package digest

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// Budget constants for the bounded evidence renderer.
//
// These values are deliberate engineering choices, NOT closure
// protocol invariants. They trade review fidelity (we lose the
// body of pathological files) for boundable output size (we can
// guarantee D_n size independence from D_{n-1}).
const (
	// MaxPerFileBytes is the maximum number of bytes the bounded
	// renderer will emit per changed file. Files that exceed the
	// "large" gate get a bounded representation well below this
	// cap.
	MaxPerFileBytes = 64 * 1024

	// MaxPerLineBytes is the maximum physical line length the
	// renderer will emit. Lines longer than this are split and
	// tagged with a line-truncation marker.
	MaxPerLineBytes = 4 * 1024

	// MaxTotalRenderBytes is the cumulative ceiling for the
	// rendered file-evidence section. Once exceeded, remaining
	// files are emitted as identity-only stubs.
	MaxTotalRenderBytes = 256 * 1024

	// MaxFileSizeForFull is the file-size threshold above which
	// a file is rendered via the bounded policy even when its
	// content is ordinary text. 1 MiB matches the existing
	// LargeFileThreshold constant in this package.
	MaxFileSizeForFull int64 = 1024 * 1024
)

// EvidenceClass is the bounded-renderer classification of a
// changed file. See package doc for the full taxonomy.
type EvidenceClass string

const (
	// ClassNormal: render full content with the existing
	// untruncated semantics. Used for ordinary small source
	// changes that pass every bounded-renderer gate.
	ClassNormal EvidenceClass = "NORMAL"

	// ClassBoundedBody: render an identity + bounded preview +
	// suppression reason. Used for files that exceed
	// MaxFileSizeForFull or contain a line longer than
	// MaxPerLineBytes.
	ClassBoundedBody EvidenceClass = "BOUNDED_BODY"

	// ClassBoundedSelfOutput: the file is the digest's own
	// output path. Emit a stub that explains the exclusion so
	// reviewers can confirm the digest did not ingest itself.
	ClassBoundedSelfOutput EvidenceClass = "BOUNDED_SELF_OUTPUT"

	// ClassBoundedDerivedDigest: the file is recognised as a
	// Leamas digest artifact by content signature (current
	// contract header or legacy `# Targeted digest` heading),
	// but the body does NOT contain structural recursion.
	// Emit identity, hashes, and a DERIVED_DIGEST_BODY_BOUNDED
	// diagnostic; do not embed the body.
	ClassBoundedDerivedDigest EvidenceClass = "BOUNDED_DERIVED_DIGEST"

	// ClassBoundedRecursive: the file is recognised as a Leamas
	// digest artifact AND the body exhibits structural recursion
	// (multiple nested contract markers or a self-diff of the
	// same path). Emit identity, hashes, and a DIGEST_RECURSION
	// diagnostic; do not embed the body.
	ClassBoundedRecursive EvidenceClass = "BOUNDED_RECURSIVE"
)

// WarningCode values rendered into the digest for bounded files.
const (
	WarningCodeLargeFileBounded     = "LARGE_FILE_EVIDENCE_BOUNDED"
	WarningCodeLineTooLong          = "PHYSICAL_LINE_BOUNDED"
	WarningCodeSelfOutput           = "SELF_OUTPUT_EXCLUDED"
	WarningCodeDerivedDigestBounded = "DERIVED_DIGEST_BODY_BOUNDED"
	WarningCodeDigestRecursion      = "DIGEST_RECURSION"
)

// classifierInput bundles the inputs needed to classify a single
// changed file.
// classifierPrefixBytes is the size of the head-byte prefix that
// classifyFileEvidence inspects to detect digest-artifact
// signatures AND pathological physical lines. The value is large
// enough to span at least one MaxPerLineBytes + 1 line so the
// pathological-line classifier can fire on production
// classifications (NOT just unit tests injecting raw prefixes).
// 16 KiB covers the largest plausible single-line content a
// production classification will encounter.
const classifierPrefixBytes = 16 * 1024

// classifierScanCap is the maximum number of bytes the classifier
// reads from a file. The bytes feed two checks: the
// classifierPrefixBytes head-byte prefix and the deeper
// structural-recursion scan. The cap is enforced BEFORE allocation
// (F8 CORRECTION02) so a 20 GiB changed file only allocates 4 MiB.
//
// The 4 MiB value is large enough to span many nested digest
// markers and several diff hunks, but small enough to keep the
// classifier itself O(1) in pathological inputs.
const classifierScanCap = 4 * 1024 * 1024

type classifierInput struct {
	repoRoot  string
	relPath   string
	fullPath  string
	size      int64
	rawPrefix string // head bytes for content checks (see classifierPrefixBytes)
	outputAbs string // absolute output path; "" if unknown
	// bodyBytes is the full file body (or empty for range-mode
	// classification where the working-tree file does not
	// necessarily reflect range evidence). When non-empty, used to
	// detect structural recursion inside the digest body.
	bodyBytes []byte
}

// classifyFileEvidence classifies `in` according to the bounded
// evidence renderer rules. The decision is deterministic and
// content-aware; filename alone never overrides content.
//
// Precedence (highest first):
//
//  1. Path == canonical output: BOUNDED_SELF_OUTPUT
//  2. Content matches digest artifact signatures AND body
//     exhibits structural recursion: BOUNDED_RECURSIVE
//  3. Content matches digest artifact signatures (no
//     structural recursion): BOUNDED_DERIVED_DIGEST
//  4. File size > MaxFileSizeForFull OR a line longer than
//     MaxPerLineBytes is observed in the prefix: BOUNDED_BODY
//  5. Otherwise: NORMAL
func classifyFileEvidence(in classifierInput) EvidenceClass {
	if in.outputAbs != "" && in.fullPath != "" {
		if pathsReferToSameFile(in.fullPath, in.outputAbs) {
			return ClassBoundedSelfOutput
		}
	}
	if isLeamasDigestArtifactContent(in.rawPrefix) {
		if hasStructuralRecursion(in.bodyBytes, in.relPath) {
			return ClassBoundedRecursive
		}
		return ClassBoundedDerivedDigest
	}
	if in.size > MaxFileSizeForFull {
		return ClassBoundedBody
	}
	if hasPathologicalLine(in.rawPrefix) {
		return ClassBoundedBody
	}
	return ClassNormal
}

// pathsReferToSameFile returns true when a and b resolve to the
// same file. Uses filepath.Clean + filepath.EvalSymlinks (when
// possible) so trivial aliases do not defeat output
// self-exclusion.
func pathsReferToSameFile(a, b string) bool {
	ca := filepath.Clean(a)
	cb := filepath.Clean(b)
	if ca == cb {
		return true
	}
	ea, errA := filepath.EvalSymlinks(ca)
	eb, errB := filepath.EvalSymlinks(cb)
	if errA == nil && errB == nil && ea == eb {
		return true
	}
	return false
}

// isLeamasDigestArtifactContent returns true when the content
// matches the canonical Leamas digest artifact signatures.
//
// We accept both the current v3 contract header and the legacy
// heading. A bare mention of "Targeted digest" deep inside a
// markdown body does NOT trigger recognition: we only fire on a
// heading-style match.
func isLeamasDigestArtifactContent(prefix string) bool {
	if prefix == "" {
		return false
	}
	for _, line := range strings.SplitN(prefix, "\n", 20) {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed,
			"LEAMAS_TARGETED_DIGEST_CONTRACT_VERSION:") {
			return true
		}
		if trimmed == "# Targeted digest" {
			return true
		}
	}
	return false
}

// hasPathologicalLine returns true when any physical line in
// `prefix` exceeds MaxPerLineBytes.
func hasPathologicalLine(prefix string) bool {
	if prefix == "" {
		return false
	}
	for _, line := range strings.Split(prefix, "\n") {
		if len(line) > MaxPerLineBytes {
			return true
		}
	}
	return false
}

// loadClassifierData returns (prefix, bodyBytes, ok) for the file
// at fullPath. The prefix is at most classifierPrefixBytes and is
// used for content-aware classification. The body is at most
// classifierScanCap bytes read with io.LimitReader so the recursion
// detector can scan deeper than the prefix WITHOUT first
// allocating the working-tree file's full size. Returns
// ok=false on any error.
//
// F8 (CORRECTION02): the previous implementation called
// `body := make([]byte, info.Size())` and only THEN sliced the
// result down to scanCap. For a 20 GiB changed file that
// allocated 20 GiB before throwing it away. We now allocate at
// most classifierScanCap and read it via io.LimitReader.
func loadClassifierData(fullPath string) (string, []byte, bool) {
	f, err := os.Open(fullPath)
	if err != nil {
		return "", nil, false
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		return "", nil, false
	}
	size := info.Size()
	if size < 0 {
		return "", nil, false
	}
	scan := int64(classifierScanCap)
	if size < scan {
		scan = size
	}
	body := make([]byte, scan)
	if scan > 0 {
		if _, err := io.ReadFull(f, body); err != nil &&
			err != io.ErrUnexpectedEOF && err != io.EOF {
			return "", nil, false
		}
	}
	prefix := body
	if len(prefix) > classifierPrefixBytes {
		prefix = prefix[:classifierPrefixBytes]
	}
	return string(prefix), body, true
}

// hasStructuralRecursion returns true when `body` exhibits the
// structural fingerprint of a recursive digest artifact. A
// recognition-only artifact (one contract header, no nested
// markers, no self-diff) is NOT structural recursion and returns
// false.
//
// Structural recursion signals (any one is sufficient):
//
//   - two or more LEAMAS_TARGETED_DIGEST_CONTRACT_VERSION markers
//     in the body (nested digest headers);
//   - one or more `diff --git a/<relPath> b/<relPath>` lines where
//     `relPath` is the path of the artifact being classified
//     (i.e. the digest body contains a self-diff of the same path).
//
// F9 (CORRECTION02): the previous implementation labelled any
// `diff --git a/` substring as recursion, which falsely flagged
// every healthy digest that reviewed ordinary source files. The
// recursion check is now path-aware and consults `relPath`.
//
// The check is bounded: it scans at most classifierScanCap bytes
// of `body` to keep the recursion detector itself O(1) in
// pathological inputs.
func hasStructuralRecursion(body []byte, relPath string) bool {
	s := string(body)
	markerCount := strings.Count(s,
		"LEAMAS_TARGETED_DIGEST_CONTRACT_VERSION:")
	if markerCount >= 2 {
		return true
	}
	if relPath == "" {
		return false
	}
	selfDiffSig := "diff --git a/" + relPath + " b/" + relPath
	return strings.Contains(s, selfDiffSig)
}

// resolveAbsoluteOutputPath canonicalises `output` against
// `repoRoot`. When output is empty, "" is returned. When output is
// relative, it is interpreted relative to repoRoot. The result is
// suitable for pathsReferToSameFile comparisons.
func resolveAbsoluteOutputPath(output, repoRoot string) string {
	if output == "" {
		return ""
	}
	if filepath.IsAbs(output) {
		return filepath.Clean(output)
	}
	if repoRoot == "" {
		return filepath.Clean(output)
	}
	return filepath.Clean(filepath.Join(repoRoot, output))
}

// sha256HexFile returns the lowercase hex SHA-256 of the file at
// fullPath, or "" if the file cannot be read.
func sha256HexFile(fullPath string) string {
	f, err := os.Open(fullPath)
	if err != nil {
		return ""
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return ""
	}
	return hex.EncodeToString(h.Sum(nil))
}

// rangeBlobOID returns the Git object ID of the blob at
// <ref>:<path>. F13 (CORRECTION02): this is the canonical Git
// identity for the suppressed historical artifact and is
// preferred over SHA-256 of arbitrary bytes.
func rangeBlobOID(runner gitRunner, repoRoot, ref, path string) string {
	if ref == "" || path == "" {
		return ""
	}
	out, err := runner.Run(repoRoot, []string{
		"rev-parse", ref + ":" + path})
	if err != nil {
		return ""
	}
	return strings.TrimSpace(out)
}
