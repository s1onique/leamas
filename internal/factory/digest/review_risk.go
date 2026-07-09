// Package digest provides targeted digest generation for Git repositories.
package digest

import (
	"os"
	"path/filepath"
	"strings"
)

// ComputeRiskSignals computes deterministic risk signals from stats and manifest.
func ComputeRiskSignals(stats FileStats, manifest []ReviewChangedFile, repoRoot string) RiskSignals {
	var rs RiskSignals
	rs.LargeFileThresholdBytes = LargeFileThreshold

	hasProd := stats.SourceFiles > 0 || hasNonTestDocConfigFiles(manifest, repoRoot)
	hasTest := stats.TestFiles > 0
	hasCode := hasProd || hasTest

	rs.ProductionWithoutTests = hasProd && !hasTest
	rs.TestsWithoutProduction = hasTest && !hasProd
	rs.DocsWithoutCode = stats.DocFiles > 0 && !hasCode
	rs.GeneratedFilesChanged = stats.GeneratedFiles > 0
	rs.ConfigFilesChanged = stats.ConfigFiles > 0
	rs.DeletedFilesChanged = stats.DeletedFiles > 0
	rs.UnmergedFilesPresent = stats.UnmergedFiles > 0
	rs.LargeFileChanged = hasLargeFile(manifest, repoRoot)

	return rs
}

// RenderRiskSignals renders the RISK_SIGNALS section.
func RenderRiskSignals(rs RiskSignals) string {
	var sb strings.Builder
	sb.WriteString("## RISK_SIGNALS\n")
	sb.WriteString("production_without_tests=")
	sb.WriteString(boolToString(rs.ProductionWithoutTests))
	sb.WriteString("\ntests_without_production=")
	sb.WriteString(boolToString(rs.TestsWithoutProduction))
	sb.WriteString("\ndocs_without_code=")
	sb.WriteString(boolToString(rs.DocsWithoutCode))
	sb.WriteString("\ngenerated_files_changed=")
	sb.WriteString(boolToString(rs.GeneratedFilesChanged))
	sb.WriteString("\nconfig_files_changed=")
	sb.WriteString(boolToString(rs.ConfigFilesChanged))
	sb.WriteString("\ndeleted_files_changed=")
	sb.WriteString(boolToString(rs.DeletedFilesChanged))
	sb.WriteString("\nunmerged_files_present=")
	sb.WriteString(boolToString(rs.UnmergedFilesPresent))
	sb.WriteString("\nlarge_file_changed=")
	sb.WriteString(boolToString(rs.LargeFileChanged))
	sb.WriteString("\nlarge_file_threshold_bytes=")
	sb.WriteString(int64ToString(rs.LargeFileThresholdBytes))
	sb.WriteString("\n")
	return sb.String()
}

// hasNonTestDocConfigFiles checks if manifest has source files.
func hasNonTestDocConfigFiles(manifest []ReviewChangedFile, repoRoot string) bool {
	for _, f := range manifest {
		if classifyFile(f.Path) == "source" {
			return true
		}
	}
	return false
}

// hasLargeFile checks if any changed file exceeds the size threshold.
func hasLargeFile(manifest []ReviewChangedFile, repoRoot string) bool {
	for _, f := range manifest {
		if f.Status == StatusDeleted {
			continue
		}
		info, err := os.Stat(filepath.Join(repoRoot, f.Path))
		if err != nil {
			continue
		}
		if info.Size() > LargeFileThreshold {
			return true
		}
	}
	return false
}

// Helper functions

func intToString(i int) string {
	if i == 0 {
		return "0"
	}
	var sb strings.Builder
	negative := i < 0
	if negative {
		i = -i
	}
	digits := make([]byte, 0, 20)
	for i > 0 {
		digits = append(digits, byte('0'+i%10))
		i /= 10
	}
	if negative {
		sb.WriteByte('-')
	}
	for j := len(digits) - 1; j >= 0; j-- {
		sb.WriteByte(digits[j])
	}
	return sb.String()
}

func int64ToString(i int64) string {
	if i == 0 {
		return "0"
	}
	var sb strings.Builder
	negative := i < 0
	if negative {
		i = -i
	}
	digits := make([]byte, 0, 20)
	for i > 0 {
		digits = append(digits, byte('0'+i%10))
		i /= 10
	}
	if negative {
		sb.WriteByte('-')
	}
	for j := len(digits) - 1; j >= 0; j-- {
		sb.WriteByte(digits[j])
	}
	return sb.String()
}

func boolToString(b bool) string {
	if b {
		return "true"
	}
	return "false"
}
