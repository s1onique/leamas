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
// for rendering and evidence hashing. The resolved digest mode is threaded
// into the binding classifier so the rendered section reports whether the
// discovered gate summary is authoritative for the current digest.
func buildGateSummarySection(repoRoot string, resolved *ResolvedMode) string {
	sourceFile := filepath.Join(repoRoot, gateSummaryPath)
	return buildGateSummarySectionFromPath(sourceFile, resolved)
}

// buildGateSummarySectionFromPath builds the gate summary section from a
// specific path. This is the primary shared adapter used by all digest modes.
// The resolved digest mode is required so the binding classifier can decide
// whether the discovered gate summary is authoritative for the digest.
func buildGateSummarySectionFromPath(sourcePath string, resolved *ResolvedMode) string {
	authority := digestAuthorityFromResolved(resolved)
	// Stage 1: Attempt to open the file
	f, err := os.Open(sourcePath)
	if err != nil {
		if os.IsNotExist(err) {
			return renderGateSummaryMissing(sourcePath, authority)
		}
		// Invalid read - directory, permission, etc.
		return renderGateSummaryInvalidRead(sourcePath, authority)
	}
	defer f.Close()

	// Stage 2: Verify it's a regular file
	fi, err := f.Stat()
	if err != nil {
		return renderGateSummaryInvalidRead(sourcePath, authority)
	}
	if !fi.Mode().IsRegular() {
		return renderGateSummaryInvalidRead(sourcePath, authority)
	}

	// Stage 3: Decode using the authoritative pipeline
	decodeResult := gatesummary.Decode(f)
	if !decodeResult.Success() {
		return renderGateSummaryInvalidDecode(sourcePath, decodeResult.Diagnostics, authority)
	}

	// Stage 4: Normalize - only after successful decode
	normResult := gatesummary.Normalize(decodeResult.Document)
	if !normResult.Success() {
		return renderGateSummaryInvalidNormalize(sourcePath, decodeResult.Document.Version(), normResult.Diagnostics, authority)
	}

	// Stage 5: Compute the binding classifier verdict. The
	// classifier is the single source of truth for whether
	// the discovered gate summary is authoritative for the
	// digest. By Stage 5 the file has been read, parsed,
	// and normalized successfully, so SourceValid applies.
	summary := normResult.Summary
	binding := classifyGateEvidence(summary, SourceValid, resolved)

	// Stage 6: Render based on schema version. The binding
	// block is rendered FIRST so authoritative qualification
	// is adjacent to (and above) the historical verdict.
	switch summary.SchemaVersion {
	case gatesummary.Version1:
		return renderGateSummaryV1(sourcePath, summary, binding)
	case gatesummary.Version2:
		return renderGateSummaryV2(sourcePath, summary, binding)
	case gatesummary.Version3:
		return renderGateSummaryV3(sourcePath, summary, binding)
	default:
		// Should not happen - normalization should reject invalid versions
		return renderGateSummaryInvalidRead(sourcePath, authority)
	}
}
