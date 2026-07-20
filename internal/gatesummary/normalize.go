package gatesummary

import (
	"errors"
	"fmt"
)

// errInvalidSealedDocument is returned when a Document has invalid pointer state.
var errInvalidSealedDocument = errors.New("gatesummary: invalid sealed document")

// validateSealed checks that exactly one version pointer is populated.
func (d Document) validateSealed() error {
	switch {
	case d.v1 != nil && d.v2 == nil:
		return nil
	case d.v1 == nil && d.v2 != nil:
		return nil
	case d.v1 == nil && d.v2 == nil:
		return fmt.Errorf("gatesummary: normalize: %w", errInvalidSealedDocument)
	default:
		return fmt.Errorf("gatesummary: normalize: %w", errInvalidSealedDocument)
	}
}

// Normalize runs the semantic normalization pipeline on a sealed Document.
// It returns a NormalizationResult with the normalized Summary on success,
// or diagnostics on semantic-invalid input, or an error on operational failure.
func Normalize(doc Document) NormalizationResult {
	// Stage 1: Validate sealed Document state
	if err := doc.validateSealed(); err != nil {
		return NormalizationResult{
			Err: err,
		}
	}

	// Stage 2-4: Project version-specific wire values and normalize
	var candidate Summary

	switch doc.Version() {
	case Version1:
		v1, _ := doc.V1()
		var err error
		candidate, err = projectV1(v1)
		if err != nil {
			return NormalizationResult{
				Err: fmt.Errorf("gatesummary: normalize: %w", err),
			}
		}
	case Version2:
		v2, _ := doc.V2()
		var err error
		candidate, err = projectV2(v2)
		if err != nil {
			return NormalizationResult{
				Err: fmt.Errorf("gatesummary: normalize: %w", err),
			}
		}
	default:
		return NormalizationResult{
			Err: fmt.Errorf("gatesummary: normalize: %w",
				errors.New("impossible schema version")),
		}
	}

	// Stage 5: Run version-specific semantic validators
	if doc.Version() == Version2 {
		var ds diagnosticSet

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

		// Stage 7: Collect and deterministically order diagnostics
		diagnostics := ds.emit()

		// Stage 8: Publish Summary only when diagnostics are empty
		if len(diagnostics) > 0 {
			return NormalizationResult{
				Diagnostics: diagnostics,
			}
		}
	}

	return NormalizationResult{
		Summary: candidate,
	}
}

// normalizeWithFault is unexported for fault injection testing.
// It applies operational failures at the specified stage.
func normalizeWithFault(doc Document, faultStage int) NormalizationResult {
	switch faultStage {
	case 1:
		// Operational failure: invalid sealed document
		return NormalizationResult{
			Diagnostics: []Diagnostic{{
				Code:    CodeNormalizationFailure,
				Path:    "/",
				Message: "injected normalization failure",
			}},
			Err: errors.New("gatesummary: injected fault"),
		}
	case 2:
		// Internal failure: impossible version
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
