package gatesummary

import (
	"errors"
	"fmt"
)

// Normalize runs the semantic normalization pipeline on a sealed Document.
// It returns a NormalizationResult with the normalized Summary on success,
// or diagnostics on semantic-invalid input, or an error on operational failure.
func Normalize(doc Document) NormalizationResult {
	// Stage 1: Validate sealed Document state
	if doc.Version() == 0 {
		return NormalizationResult{
			Err: fmt.Errorf("gatesummary: normalize: %w",
				errors.New("zero-value document")),
		}
	}

	// Stage 2-4: Project version-specific wire values and normalize
	var candidate Summary
	var ds diagnosticSet

	switch doc.Version() {
	case Version1:
		v1, _ := doc.V1()
		candidate = projectV1(v1)
	case Version2:
		v2, _ := doc.V2()
		candidate = projectV2(v2)
	default:
		return NormalizationResult{
			Err: fmt.Errorf("gatesummary: normalize: %w",
				errors.New("impossible schema version")),
		}
	}

	// Stage 5: Run version-specific semantic validators
	if doc.Version() == Version2 {
		// Duplicate check names
		for _, d := range validateCheckNames(candidate.Checks) {
			ds.add(d)
		}
		// Exit code relationships
		for _, d := range validateExitCodes(candidate.Checks) {
			ds.add(d)
		}
		// Test totals arithmetic
		for _, d := range validateTestTotals(candidate.Checks) {
			ds.add(d)
		}
		// Overall status derivation
		for _, d := range validateOverallStatus(candidate.Checks, candidate.Overall.Status) {
			ds.add(d)
		}
		// Cleanliness validation
		if candidate.Scope != nil {
			for _, d := range validateCleanliness(
				candidate.Scope.Status,
				candidate.Worktree.CleanBefore,
				candidate.Worktree.CleanAfter,
			) {
				ds.add(d)
			}
		}
	}

	// Stage 7: Collect and deterministically order diagnostics
	diagnostics := ds.emit()

	// Stage 8: Publish Summary only when diagnostics are empty
	if len(diagnostics) > 0 {
		return NormalizationResult{
			Diagnostics: diagnostics,
		}
	}

	return NormalizationResult{
		Summary: candidate,
	}
}

// NormalizeWithFault is package-private for fault injection testing.
// It applies operational failures at the specified stage.
func NormalizeWithFault(doc Document, faultStage int) NormalizationResult {
	switch faultStage {
	case 1:
		// Operational failure: zero document
		return NormalizationResult{
			Diagnostics: []Diagnostic{{
				Code:    CodeNormalizationFailure,
				Path:    "/",
				Message: "injected normalization failure",
			}},
			Err: errors.New("gatesummary: injected fault"),
		}
	case 2:
		// Impossible version
		return NormalizationResult{
			Diagnostics: []Diagnostic{{
				Code:    CodeInternal,
				Path:    "/",
				Message: "impossible schema version",
			}},
			Err: errors.New("gatesummary: injected internal failure"),
		}
	}
	return Normalize(doc)
}
