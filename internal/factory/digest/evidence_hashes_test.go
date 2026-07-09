// Package digest provides targeted digest generation for Git repositories.
package digest

import (
	"strings"
	"testing"
)

func TestSHA256Hex_ReturnsLowercase64Chars(t *testing.T) {
	hash := SHA256Hex("test input")
	if len(hash) != 64 {
		t.Errorf("expected 64 chars, got %d", len(hash))
	}
	// Check all lowercase hex
	for _, c := range hash {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
			t.Errorf("expected lowercase hex, got %c", c)
		}
	}
}

func TestNormalizeHashInput_ConvertsCRLFToLF(t *testing.T) {
	input := "line1\r\nline2\r\nline3\r\n"
	result := NormalizeHashInput(input)
	expected := "line1\nline2\nline3\n"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestNormalizeHashInput_EnsuresSingleFinalNewline(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"no trailing newline", "line1\nline2", "line1\nline2\n"},
		{"single trailing newline", "line1\nline2\n", "line1\nline2\n"},
		{"multiple trailing newlines", "line1\nline2\n\n\n", "line1\nline2\n"},
		{"only newlines", "\n\n\n", "\n"},
		{"empty string", "", "\n"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := NormalizeHashInput(tc.input)
			if result != tc.expected {
				t.Errorf("expected %q, got %q", tc.expected, result)
			}
		})
	}
}

func TestRenderEvidenceHashes_StableKeyOrder(t *testing.T) {
	eh := EvidenceHashes{
		HashAlgorithm:           "sha256",
		HashScope:               "normalized_digest_v2_sections",
		ChangesetManifestSHA256: "aaaa",
		ChangesetStatsSHA256:    "bbbb",
		ReviewMapSHA256:         "cccc",
		RiskSignalsSHA256:       "dddd",
		PatchHygieneSHA256:      "eeee",
		GateSummarySHA256:       "0000",
		FileEvidenceSHA256:      "ffff",
		DigestEvidenceSHA256:    "gggg",
	}

	result := RenderEvidenceHashes(eh)
	lines := strings.Split(strings.TrimSpace(result), "\n")

	// Check expected line count
	if len(lines) != 11 {
		t.Errorf("expected 11 lines, got %d", len(lines))
	}

	// Check key order
	expectedKeys := []string{
		"## EVIDENCE_HASHES",
		"hash_algorithm=",
		"hash_scope=",
		"changeset_manifest_sha256=",
		"changeset_stats_sha256=",
		"review_map_sha256=",
		"risk_signals_sha256=",
		"patch_hygiene_sha256=",
		"gate_summary_sha256=",
		"file_evidence_sha256=",
		"digest_evidence_sha256=",
	}

	for i, expected := range expectedKeys {
		if !strings.HasPrefix(lines[i], expected) {
			t.Errorf("line %d: expected prefix %q, got %q", i, expected, lines[i])
		}
	}
}

func TestEvidenceHashes_DoNotIncludeOwnSection(t *testing.T) {
	// The EVIDENCE_HASHES section should not include itself in digest_evidence_sha256
	eh := ComputeEvidenceHashes(
		"## CHANGESET_MANIFEST\nA  file.go\n",
		"## CHANGESET_STATS\nfiles_changed=1\n",
		"## REVIEW_MAP\nproduction:\n  - file.go\n",
		"## RISK_SIGNALS\nproduction_without_tests=true\n",
		"## PATCH_HYGIENE\ngit_diff_check=pass\n",
		"## GATE_SUMMARY\nsource=.factory/gate-summary.json\n",
		"## Changed files\nfile.go\n",
	)

	// Hash of the rendered EVIDENCE_HASHES section
	selfHash := SHA256Hex(RenderEvidenceHashes(eh))

	// Verify digest_evidence does not contain the EVIDENCE_HASHES section
	if eh.DigestEvidenceSHA256 == selfHash {
		t.Error("digest_evidence_sha256 should not include EVIDENCE_HASHES section")
	}
}

func TestDigestEvidenceHash_StableAcrossRepeatedRender(t *testing.T) {
	manifest := "## CHANGESET_MANIFEST\nA  file.go\n"
	stats := "## CHANGESET_STATS\nfiles_changed=1\n"
	reviewMap := "## REVIEW_MAP\nproduction:\n  - file.go\n"
	risk := "## RISK_SIGNALS\nproduction_without_tests=true\n"
	patch := "## PATCH_HYGIENE\ngit_diff_check=pass\n"
	gateSummary := "## GATE_SUMMARY\nsource=.factory/gate-summary.json\n"
	fileEv := "## Changed files\nfile.go\n"

	// Compute hashes twice
	eh1 := ComputeEvidenceHashes(manifest, stats, reviewMap, risk, patch, gateSummary, fileEv)
	eh2 := ComputeEvidenceHashes(manifest, stats, reviewMap, risk, patch, gateSummary, fileEv)

	if eh1.DigestEvidenceSHA256 != eh2.DigestEvidenceSHA256 {
		t.Error("digest_evidence_sha256 should be stable across renders")
	}

	if eh1.ChangesetManifestSHA256 != eh2.ChangesetManifestSHA256 {
		t.Error("changeset_manifest_sha256 should be stable across renders")
	}
}

func TestFileEvidenceHash_ChangesWhenDiffChanges(t *testing.T) {
	hash1 := ComputeFileEvidence("## Changed files\nfile1.go\n")
	hash2 := ComputeFileEvidence("## Changed files\nfile2.go\n")

	if hash1 == hash2 {
		t.Error("file_evidence_sha256 should change when content changes")
	}
}

func TestSectionHash_ChangesWhenRiskSignalChanges(t *testing.T) {
	hash1 := ComputeSectionHash("## RISK_SIGNALS\nproduction_without_tests=true\n")
	hash2 := ComputeSectionHash("## RISK_SIGNALS\nproduction_without_tests=false\n")

	if hash1 == hash2 {
		t.Error("section hash should change when content changes")
	}
}
