// Package digest provides targeted digest generation for Git repositories.
package digest

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
)

// Evidence hash constants.
const (
	// EvidenceHashAlgorithm is the hash algorithm used for evidence hashes.
	EvidenceHashAlgorithm = "sha256"
	// EvidenceHashScope is the scope of evidence hashing.
	EvidenceHashScope = "normalized_digest_v2_sections"
)

// EvidenceHashes contains SHA-256 hashes for digest evidence sections.
type EvidenceHashes struct {
	HashAlgorithm           string
	HashScope               string
	ChangesetManifestSHA256 string
	ChangesetStatsSHA256    string
	ReviewMapSHA256         string
	RiskSignalsSHA256       string
	PatchHygieneSHA256      string
	GateSummarySHA256       string
	FileEvidenceSHA256      string
	DigestEvidenceSHA256    string
}

// SHA256Hex computes the SHA-256 hash of input and returns lowercase hex.
func SHA256Hex(input string) string {
	hash := sha256.Sum256([]byte(input))
	return hex.EncodeToString(hash[:])
}

// NormalizeHashInput normalizes text for stable hashing.
// - Converts CRLF to LF
// - Ensures exactly one trailing newline
func NormalizeHashInput(text string) string {
	// Convert CRLF to LF
	text = strings.ReplaceAll(text, "\r\n", "\n")

	// Remove more than one trailing newline
	text = strings.TrimRight(text, "\n")
	text += "\n"

	return text
}

// ComputeSectionHash computes the SHA-256 hash of a normalized section.
func ComputeSectionHash(section string) string {
	normalized := NormalizeHashInput(section)
	return SHA256Hex(normalized)
}

// ComputeFileEvidence computes the SHA-256 hash of file evidence content.
func ComputeFileEvidence(diffsContent string) string {
	return ComputeSectionHash(diffsContent)
}

// ComputeEvidenceHashes computes all evidence hashes from rendered sections.
func ComputeEvidenceHashes(manifestSection, statsSection, reviewMapSection,
	riskSignalsSection, patchHygieneSection, gateSummarySection, fileEvidenceSection string) EvidenceHashes {

	var eh EvidenceHashes
	eh.HashAlgorithm = EvidenceHashAlgorithm
	eh.HashScope = EvidenceHashScope

	eh.ChangesetManifestSHA256 = ComputeSectionHash(manifestSection)
	eh.ChangesetStatsSHA256 = ComputeSectionHash(statsSection)
	eh.ReviewMapSHA256 = ComputeSectionHash(reviewMapSection)
	eh.RiskSignalsSHA256 = ComputeSectionHash(riskSignalsSection)
	eh.PatchHygieneSHA256 = ComputeSectionHash(patchHygieneSection)
	eh.GateSummarySHA256 = ComputeSectionHash(gateSummarySection)
	eh.FileEvidenceSHA256 = ComputeSectionHash(fileEvidenceSection)

	// Digest evidence is the hash of all evidence sections concatenated
	digestEvidenceContent := manifestSection + "\n" +
		statsSection + "\n" +
		reviewMapSection + "\n" +
		riskSignalsSection + "\n" +
		patchHygieneSection + "\n" +
		gateSummarySection + "\n" +
		fileEvidenceSection

	eh.DigestEvidenceSHA256 = ComputeSectionHash(digestEvidenceContent)

	return eh
}

// RenderEvidenceHashes renders the EVIDENCE_HASHES section.
func RenderEvidenceHashes(eh EvidenceHashes) string {
	var sb strings.Builder
	sb.WriteString("## EVIDENCE_HASHES\n")
	sb.WriteString("hash_algorithm=")
	sb.WriteString(eh.HashAlgorithm)
	sb.WriteString("\nhash_scope=")
	sb.WriteString(eh.HashScope)
	sb.WriteString("\nchangeset_manifest_sha256=")
	sb.WriteString(eh.ChangesetManifestSHA256)
	sb.WriteString("\nchangeset_stats_sha256=")
	sb.WriteString(eh.ChangesetStatsSHA256)
	sb.WriteString("\nreview_map_sha256=")
	sb.WriteString(eh.ReviewMapSHA256)
	sb.WriteString("\nrisk_signals_sha256=")
	sb.WriteString(eh.RiskSignalsSHA256)
	sb.WriteString("\npatch_hygiene_sha256=")
	sb.WriteString(eh.PatchHygieneSHA256)
	sb.WriteString("\ngate_summary_sha256=")
	sb.WriteString(eh.GateSummarySHA256)
	sb.WriteString("\nfile_evidence_sha256=")
	sb.WriteString(eh.FileEvidenceSHA256)
	sb.WriteString("\ndigest_evidence_sha256=")
	sb.WriteString(eh.DigestEvidenceSHA256)
	sb.WriteString("\n")
	return sb.String()
}
