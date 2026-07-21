package gatesummary

// contractStage is the terminal stage a corpus row is expected to reach.
type contractStage string

const (
	stageDecodeRejected    contractStage = "decode_rejected"
	stageNormalizeRejected contractStage = "normalize_rejected"
	stageNormalized        contractStage = "normalized"
)

// diagnosticProjection is the minimum contract geometry for a single
// diagnostic: the diagnostic code, the JSON Pointer path, and the
// observed ordering slot. Tests assert against these three fields
// only; the human-readable message field is implementation-detail
// and is not part of the frozen contract.
type diagnosticProjection struct {
	Code string
	Path string
}

// normalizationContractCase is one row of the literal 41-row
// normalization corpus. Fixture is the path relative to testdata/.
// Stage declares the expected terminal stage. WantDiagnostics is
// the complete ordered projection for rejected rows.
// SuccessSchema identifies the expected normalized schema version
// for normalized rows.
type normalizationContractCase struct {
	ID              string
	Fixture         string
	Stage           contractStage
	WantDiagnostics []diagnosticProjection
	SuccessSchema   Version
}

// projectDiagnostics extracts the complete ordered (Code, Path)
// projection of a diagnostic slice. Used to assert exact geometry
// in both decode and normalize result sets.
func projectDiagnostics(ds []Diagnostic) []diagnosticProjection {
	out := make([]diagnosticProjection, len(ds))
	for i, d := range ds {
		out[i] = diagnosticProjection{Code: d.Code, Path: d.Path}
	}
	return out
}
