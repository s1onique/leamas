// Package gatesummary implements the strict, bounded, versioned gate-summary
// v1/v2 wire decoder for Leamas Factory. The decoder accepts untrusted JSON
// bytes from gate-summary producers and emits either a versioned wire
// document or a stable ordered list of GS_* diagnostics.
package gatesummary

import (
	"sort"
	"strings"
)

// Diagnostic is the stable public diagnostic shape for gate-summary
// decoding failures. The Code value is stable across releases; the
// Message is human-readable and may change.
type Diagnostic struct {
	Code     string `json:"code"`
	Path     string `json:"path,omitempty"`
	Expected string `json:"expected,omitempty"`
	Observed string `json:"observed,omitempty"`
	Message  string `json:"message"`
}

// Diagnostic code registry. Precedence ranks are frozen by
// gate-summary-compatibility-matrix.md. Do not renumber.
const (
	CodeDocumentTooLarge         = "GS_DOCUMENT_TOO_LARGE"
	CodeMalformedJSON            = "GS_MALFORMED_JSON"
	CodeTrailingJSON             = "GS_TRAILING_JSON"
	CodeDuplicateKey             = "GS_DUPLICATE_KEY"
	CodeVersionMissing           = "GS_VERSION_MISSING"
	CodeInvalidVersionType       = "GS_INVALID_VERSION_TYPE"
	CodeUnsupportedVersion       = "GS_UNSUPPORTED_VERSION"
	CodeUnknownField             = "GS_UNKNOWN_FIELD"
	CodeRequiredFieldMissing     = "GS_REQUIRED_FIELD_MISSING"
	CodeSchemaViolation          = "GS_SCHEMA_VIOLATION"
	CodeInvalidTimestamp         = "GS_INVALID_TIMESTAMP"
	CodeInvalidStatus            = "GS_INVALID_STATUS"
	CodeInvalidOID               = "GS_INVALID_OID"
	CodeCollectionLimit          = "GS_COLLECTION_LIMIT"
	CodeDuplicateCheckName       = "GS_DUPLICATE_CHECK_NAME"
	CodePassExitCodeMismatch     = "GS_PASS_EXIT_CODE_MISMATCH"
	CodeFailExitCodeMismatch     = "GS_FAIL_EXIT_CODE_MISMATCH"
	CodeSkipExitCodeMismatch     = "GS_SKIP_EXIT_CODE_MISMATCH"
	CodeUnavailExitCodeMismatch  = "GS_UNAVAILABLE_EXIT_CODE_MISMATCH"
	CodeInvalidDuration          = "GS_INVALID_DURATION"
	CodeInvalidOutputHash        = "GS_INVALID_OUTPUT_HASH"
	CodePartialTestTotals        = "GS_PARTIAL_TEST_TOTALS"
	CodeTestTotalMismatch        = "GS_TEST_TOTAL_MISMATCH"
	CodeOverallStatusMismatch    = "GS_OVERALL_STATUS_MISMATCH"
	CodeScopeClosedDirtyWorktree = "GS_SCOPE_CLOSED_DIRTY_WORKTREE"
	CodeNormalizationFailure     = "GS_NORMALIZATION_FAILURE"
	CodeInternal                 = "GS_INTERNAL"
)

// codePrecedence maps every known code to its frozen precedence rank.
// Lower rank is emitted earlier.
var codePrecedence = map[string]int{
	CodeDocumentTooLarge:         1,
	CodeMalformedJSON:            2,
	CodeTrailingJSON:             3,
	CodeDuplicateKey:             4,
	CodeVersionMissing:           5,
	CodeInvalidVersionType:       6,
	CodeUnsupportedVersion:       7,
	CodeUnknownField:             8,
	CodeRequiredFieldMissing:     9,
	CodeSchemaViolation:          10,
	CodeInvalidTimestamp:         11,
	CodeInvalidStatus:            12,
	CodeInvalidOID:               13,
	CodeCollectionLimit:          14,
	CodeDuplicateCheckName:       15,
	CodePassExitCodeMismatch:     16,
	CodeFailExitCodeMismatch:     17,
	CodeSkipExitCodeMismatch:     18,
	CodeUnavailExitCodeMismatch:  19,
	CodeInvalidDuration:          20,
	CodeInvalidOutputHash:        21,
	CodePartialTestTotals:        22,
	CodeTestTotalMismatch:        23,
	CodeOverallStatusMismatch:    24,
	CodeScopeClosedDirtyWorktree: 25,
	CodeNormalizationFailure:     26,
	CodeInternal:                 27,
}

// precedence returns the rank for code; unknown codes sort last.
func precedence(code string) int {
	if v, ok := codePrecedence[code]; ok {
		return v
	}
	return 1 << 20
}

// newDiagnostic constructs a Diagnostic with a stable message.
func newDiagnostic(code, path, message string) Diagnostic {
	return Diagnostic{
		Code:    code,
		Path:    path,
		Message: message,
	}
}

// diagnosticSet collects diagnostics while preserving encounter index,
// deduplicates by (Code, Path), and emits in deterministic order:
// precedence rank, then JSON Pointer path, then encounter index.
type diagnosticSet struct {
	items     []encounteredDiag
	encounter int
}

type encounteredDiag struct {
	index int
	d     Diagnostic
}

// add appends a diagnostic and assigns the next encounter index.
func (s *diagnosticSet) add(d Diagnostic) {
	s.encounter++
	s.items = append(s.items, encounteredDiag{index: s.encounter, d: d})
}

// emit returns the deduplicated, deterministically ordered slice.
func (s *diagnosticSet) emit() []Diagnostic {
	seen := make(map[string]bool, len(s.items))
	uniq := make([]encounteredDiag, 0, len(s.items))
	for _, it := range s.items {
		key := it.d.Code + "\x00" + it.d.Path
		if seen[key] {
			continue
		}
		seen[key] = true
		uniq = append(uniq, it)
	}
	sort.SliceStable(uniq, func(i, j int) bool {
		a, b := uniq[i], uniq[j]
		pa, pb := precedence(a.d.Code), precedence(b.d.Code)
		if pa != pb {
			return pa < pb
		}
		if a.d.Path != b.d.Path {
			return a.d.Path < b.d.Path
		}
		return a.index < b.index
	})
	out := make([]Diagnostic, len(uniq))
	for i, it := range uniq {
		out[i] = it.d
	}
	return out
}

// escapePointer escapes a path token per RFC 6901.
func escapePointer(token string) string {
	token = strings.ReplaceAll(token, "~", "~0")
	token = strings.ReplaceAll(token, "/", "~1")
	return token
}
