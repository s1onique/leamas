// SPDX-License-Identifier: Apache-2.0

// Package digest provides targeted digest generation for Git repositories.
package digest

import (
	"os"
	"path/filepath"

	"github.com/s1onique/leamas/internal/gatesummary"
)

// gateSummaryPath is the canonical source path for gate summary artifacts.
const gateSummaryPath = ".factory/gate-summary.json"

// diagnosticCodeReadFailed is a stable digest-local rendering diagnostic.
const diagnosticCodeReadFailed = "DG_GATE_SUMMARY_READ_FAILED"

// diagnosticPath is the stable repository-relative path used in diagnostics.
const diagnosticPath = "/.factory/gate-summary.json"

// buildGateSummarySection is the shared entry point for all digest modes.
// It opens and consumes the artifact once, returning the exact same string
// for rendering and evidence hashing.
func buildGateSummarySection(repoRoot string) string {
	sourceFile := filepath.Join(repoRoot, gateSummaryPath)
	return buildGateSummarySectionFromPath(sourceFile)
}

// buildGateSummarySectionFromPath builds the gate summary section from a specific path.
// This is the primary shared adapter used by all digest modes.
func buildGateSummarySectionFromPath(sourcePath string) string {
	// Stage 1: Attempt to open the file
	f, err := os.Open(sourcePath)
	if err != nil {
		if os.IsNotExist(err) {
			return renderGateSummaryMissing(sourcePath)
		}
		// Invalid read - directory, permission, etc.
		return renderGateSummaryInvalidRead(sourcePath)
	}
	defer f.Close()

	// Stage 2: Verify it's a regular file
	fi, err := f.Stat()
	if err != nil {
		return renderGateSummaryInvalidRead(sourcePath)
	}
	if !fi.Mode().IsRegular() {
		return renderGateSummaryInvalidRead(sourcePath)
	}

	// Stage 3: Decode using the authoritative pipeline
	decodeResult := gatesummary.Decode(f)
	if !decodeResult.Success() {
		return renderGateSummaryInvalidDecode(sourcePath, decodeResult.Diagnostics)
	}

	// Stage 4: Normalize - only after successful decode
	normResult := gatesummary.Normalize(decodeResult.Document)
	if !normResult.Success() {
		return renderGateSummaryInvalidNormalize(sourcePath, decodeResult.Document.Version(), normResult.Diagnostics)
	}

	// Stage 5: Render based on schema version
	summary := normResult.Summary
	switch summary.SchemaVersion {
	case gatesummary.Version1:
		return renderGateSummaryV1(sourcePath, summary)
	case gatesummary.Version2:
		return renderGateSummaryV2(sourcePath, summary)
	default:
		// Should not happen - normalization should reject invalid versions
		return renderGateSummaryInvalidRead(sourcePath)
	}
}
