package gatesummary

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// normalizationContractCorpus is the frozen, literal 41-row corpus.
// Each row is explicit. Rows are NOT generated from production
// validators, fixtures directories, or reflection. Adding, removing,
// or reordering a row is a contract change and must be reviewed.
var normalizationContractCorpus = []normalizationContractCase{
	// valid/ (7 rows) — all normalized
	{
		ID: "GS2-NORM-001", Fixture: "valid/v1-full.json",
		Stage:         stageNormalized,
		SuccessSchema: Version1,
	},
	{
		ID: "GS2-NORM-002", Fixture: "valid/v1-minimal.json",
		Stage:         stageNormalized,
		SuccessSchema: Version1,
	},
	{
		ID: "GS2-NORM-003", Fixture: "valid/v2-clinemm-microc3.json",
		Stage:         stageNormalized,
		SuccessSchema: Version2,
	},
	{
		ID: "GS2-NORM-004", Fixture: "valid/v2-full.json",
		Stage:         stageNormalized,
		SuccessSchema: Version2,
	},
	{
		ID: "GS2-NORM-005", Fixture: "valid/v2-leamas-self-hosted.json",
		Stage:         stageNormalized,
		SuccessSchema: Version2,
	},
	{
		ID: "GS2-NORM-006", Fixture: "valid/v2-minimal.json",
		Stage:         stageNormalized,
		SuccessSchema: Version2,
	},
	{
		ID: "GS2-NORM-007", Fixture: "valid/v2-root-scope.json",
		Stage:         stageNormalized,
		SuccessSchema: Version2,
	},

	// invalid/ decode-rejected (20 rows)
	{
		ID: "GS2-NORM-008", Fixture: "invalid/v1-unknown-field.json",
		Stage: stageDecodeRejected,
		WantDiagnostics: []diagnosticProjection{
			{Code: CodeUnknownField, Path: "/scope_id"},
			{Code: CodeUnknownField, Path: "/scope_status"},
		},
	},
	{
		ID: "GS2-NORM-009", Fixture: "invalid/v2-bad-status-enum.json",
		Stage: stageDecodeRejected,
		WantDiagnostics: []diagnosticProjection{
			{Code: CodeInvalidStatus, Path: "/checks/0/status"},
		},
	},
	{
		ID: "GS2-NORM-010", Fixture: "invalid/v2-empty-generated-at.json",
		Stage: stageDecodeRejected,
		WantDiagnostics: []diagnosticProjection{
			{Code: CodeInvalidTimestamp, Path: "/generated_at"},
		},
	},
	{
		ID: "GS2-NORM-011", Fixture: "invalid/v2-invalid-hash.json",
		Stage: stageDecodeRejected,
		WantDiagnostics: []diagnosticProjection{
			{Code: CodeInvalidOutputHash, Path: "/checks/0/extras/stdout_sha256"},
		},
	},
	{
		ID: "GS2-NORM-012", Fixture: "invalid/v2-invalid-timestamp.json",
		Stage: stageDecodeRejected,
		WantDiagnostics: []diagnosticProjection{
			{Code: CodeInvalidTimestamp, Path: "/generated_at"},
		},
	},
	{
		ID: "GS2-NORM-013", Fixture: "invalid/v2-lower-lifecycle.json",
		Stage: stageDecodeRejected,
		WantDiagnostics: []diagnosticProjection{
			{Code: CodeInvalidStatus, Path: "/parent_status"},
			{Code: CodeInvalidStatus, Path: "/scope_status"},
		},
	},
	{
		ID: "GS2-NORM-014", Fixture: "invalid/v2-missing-execution-head-oid.json",
		Stage: stageDecodeRejected,
		WantDiagnostics: []diagnosticProjection{
			{Code: CodeRequiredFieldMissing, Path: "/execution_head_oid"},
		},
	},
	{
		ID: "GS2-NORM-015", Fixture: "invalid/v2-missing-schema-version.json",
		Stage: stageDecodeRejected,
		WantDiagnostics: []diagnosticProjection{
			{Code: CodeVersionMissing, Path: "/schema_version"},
		},
	},
	{
		ID: "GS2-NORM-016", Fixture: "invalid/v2-negative-duration.json",
		Stage: stageDecodeRejected,
		WantDiagnostics: []diagnosticProjection{
			{Code: CodeInvalidDuration, Path: "/checks/0/extras/duration_ms"},
		},
	},
	{
		ID: "GS2-NORM-017", Fixture: "invalid/v2-null-execution-head-oid.json",
		Stage: stageDecodeRejected,
		WantDiagnostics: []diagnosticProjection{
			{Code: CodeInvalidOID, Path: "/execution_head_oid"},
		},
	},
	{
		ID: "GS2-NORM-018", Fixture: "invalid/v2-partial-test-totals.json",
		Stage: stageDecodeRejected,
		WantDiagnostics: []diagnosticProjection{
			{Code: CodePartialTestTotals, Path: "/checks/0"},
		},
	},
	{
		ID: "GS2-NORM-019", Fixture: "invalid/v2-schema-version-decimal.json",
		Stage: stageDecodeRejected,
		WantDiagnostics: []diagnosticProjection{
			{Code: CodeInvalidVersionType, Path: "/schema_version"},
		},
	},
	{
		ID: "GS2-NORM-020", Fixture: "invalid/v2-schema-version-negative.json",
		Stage: stageDecodeRejected,
		WantDiagnostics: []diagnosticProjection{
			{Code: CodeUnsupportedVersion, Path: "/schema_version"},
		},
	},
	{
		ID: "GS2-NORM-021", Fixture: "invalid/v2-schema-version-string.json",
		Stage: stageDecodeRejected,
		WantDiagnostics: []diagnosticProjection{
			{Code: CodeInvalidVersionType, Path: "/schema_version"},
		},
	},
	{
		ID: "GS2-NORM-022", Fixture: "invalid/v2-schema-version-zero.json",
		Stage: stageDecodeRejected,
		WantDiagnostics: []diagnosticProjection{
			{Code: CodeUnsupportedVersion, Path: "/schema_version"},
		},
	},
	{
		ID: "GS2-NORM-023", Fixture: "invalid/v2-trailing-second-value.json",
		Stage: stageDecodeRejected,
		WantDiagnostics: []diagnosticProjection{
			{Code: CodeTrailingJSON, Path: ""},
		},
	},
	{
		ID: "GS2-NORM-024", Fixture: "invalid/v2-truncated.json",
		Stage: stageDecodeRejected,
		WantDiagnostics: []diagnosticProjection{
			{Code: CodeMalformedJSON, Path: "/scope_id"},
		},
	},
	{
		ID: "GS2-NORM-025", Fixture: "invalid/v2-unknown-field.json",
		Stage: stageDecodeRejected,
		WantDiagnostics: []diagnosticProjection{
			{Code: CodeUnknownField, Path: "/tool"},
		},
	},
	{
		ID: "GS2-NORM-026", Fixture: "invalid/v2-unsupported-version-3.json",
		Stage: stageDecodeRejected,
		WantDiagnostics: []diagnosticProjection{
			{Code: CodeUnsupportedVersion, Path: "/schema_version"},
		},
	},
	{
		ID: "GS2-NORM-027", Fixture: "invalid/v2-uppercase-oid.json",
		Stage: stageDecodeRejected,
		WantDiagnostics: []diagnosticProjection{
			{Code: CodeInvalidOID, Path: "/execution_head_oid"},
		},
	},

	// invalid/ normalize-rejected (8 rows)
	{
		ID: "GS2-NORM-028", Fixture: "invalid/v2-duplicate-check-name.json",
		Stage: stageNormalizeRejected,
		WantDiagnostics: []diagnosticProjection{
			{Code: CodeDuplicateCheckName, Path: "/checks/1/name"},
		},
	},
	{
		ID: "GS2-NORM-029", Fixture: "invalid/v2-fail-exit-zero.json",
		Stage: stageNormalizeRejected,
		WantDiagnostics: []diagnosticProjection{
			{Code: CodeFailExitCodeMismatch, Path: "/checks/0/extras/exit_code"},
		},
	},
	{
		ID: "GS2-NORM-030", Fixture: "invalid/v2-overall-mismatch.json",
		Stage: stageNormalizeRejected,
		WantDiagnostics: []diagnosticProjection{
			{Code: CodeOverallStatusMismatch, Path: "/overall_status"},
		},
	},
	{
		ID: "GS2-NORM-031", Fixture: "invalid/v2-pass-nonzero-exit.json",
		Stage: stageNormalizeRejected,
		WantDiagnostics: []diagnosticProjection{
			{Code: CodePassExitCodeMismatch, Path: "/checks/0/extras/exit_code"},
		},
	},
	{
		ID: "GS2-NORM-032", Fixture: "invalid/v2-scope-closed-dirty-after.json",
		Stage: stageNormalizeRejected,
		WantDiagnostics: []diagnosticProjection{
			{Code: CodeOverallStatusMismatch, Path: "/overall_status"},
			{Code: CodeScopeClosedDirtyWorktree, Path: "/worktree_clean_after"},
		},
	},
	{
		ID: "GS2-NORM-033", Fixture: "invalid/v2-skip-nonnull-exit.json",
		Stage: stageNormalizeRejected,
		WantDiagnostics: []diagnosticProjection{
			{Code: CodeSkipExitCodeMismatch, Path: "/checks/0/extras/exit_code"},
		},
	},
	{
		ID: "GS2-NORM-034", Fixture: "invalid/v2-test-total-mismatch.json",
		Stage: stageNormalizeRejected,
		WantDiagnostics: []diagnosticProjection{
			{Code: CodeTestTotalMismatch, Path: "/checks/0"},
		},
	},
	{
		ID: "GS2-NORM-035", Fixture: "invalid/v2-unavailable-nonnull-exit.json",
		Stage: stageNormalizeRejected,
		WantDiagnostics: []diagnosticProjection{
			{Code: CodeUnavailExitCodeMismatch, Path: "/checks/0/extras/exit_code"},
		},
	},

	// duplicate-keys/ (3 rows) — all decode-rejected
	{
		ID: "GS2-NORM-036", Fixture: "duplicate-keys/v2-duplicate-nested-field.json",
		Stage: stageDecodeRejected,
		WantDiagnostics: []diagnosticProjection{
			{Code: CodeDuplicateKey, Path: "/checks/0/name"},
		},
	},
	{
		ID: "GS2-NORM-037", Fixture: "duplicate-keys/v2-duplicate-schema-version.json",
		Stage: stageDecodeRejected,
		WantDiagnostics: []diagnosticProjection{
			{Code: CodeDuplicateKey, Path: "/schema_version"},
		},
	},
	{
		ID: "GS2-NORM-038", Fixture: "duplicate-keys/v2-duplicate-top-level-field.json",
		Stage: stageDecodeRejected,
		WantDiagnostics: []diagnosticProjection{
			{Code: CodeDuplicateKey, Path: "/scope_id"},
		},
	},

	// limits/ (3 rows) — structural-shape fixtures
	{
		ID: "GS2-NORM-039", Fixture: "limits/v2-checks-boundary-shape.json",
		Stage:         stageNormalized,
		SuccessSchema: Version2,
	},
	{
		ID: "GS2-NORM-040", Fixture: "limits/v2-checks-over-boundary-shape.json",
		Stage:         stageNormalized,
		SuccessSchema: Version2,
	},
	{
		ID: "GS2-NORM-041", Fixture: "limits/v2-document-size-shape.json",
		Stage: stageNormalizeRejected,
		WantDiagnostics: []diagnosticProjection{
			{Code: CodeOverallStatusMismatch, Path: "/overall_status"},
		},
	},
}

// TestNormalizationCorpusCardinality asserts the corpus contains
// exactly 41 explicit rows. This test is the cardinality guard.
func TestNormalizationCorpusCardinality(t *testing.T) {
	const want = 41
	if got := len(normalizationContractCorpus); got != want {
		t.Fatalf("normalization contract corpus has %d rows, want %d", got, want)
	}
	seen := make(map[string]bool, want)
	for i, c := range normalizationContractCorpus {
		if c.ID == "" {
			t.Fatalf("corpus[%d]: empty ID", i)
		}
		if !strings.HasPrefix(c.ID, "GS2-NORM-") {
			t.Fatalf("corpus[%d]: ID %q does not start with GS2-NORM-", i, c.ID)
		}
		if seen[c.ID] {
			t.Fatalf("corpus[%d]: duplicate ID %q", i, c.ID)
		}
		seen[c.ID] = true
		if c.Stage == "" {
			t.Fatalf("corpus[%d] (%s): empty stage", i, c.ID)
		}
		switch c.Stage {
		case stageDecodeRejected, stageNormalizeRejected, stageNormalized:
		default:
			t.Fatalf("corpus[%d] (%s): unknown stage %q", i, c.ID, c.Stage)
		}
	}
}

// TestNormalizationContractCorpus walks every corpus row, runs the
// canonical Decode → Normalize pipeline, and asserts exact terminal
// stage plus complete ordered diagnostic geometry for rejected rows.
// Decode-rejected rows must NOT invoke Normalize.
func TestNormalizationContractCorpus(t *testing.T) {
	TestNormalizationCorpusCardinality(t)

	for _, c := range normalizationContractCorpus {
		c := c
		t.Run(c.ID, func(t *testing.T) {
			path := filepath.Join("testdata", c.Fixture)
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("read %s: %v", path, err)
			}

			dec := Decode(strings.NewReader(string(data)))

			switch c.Stage {
			case stageDecodeRejected:
				if dec.Success() {
					t.Fatalf("%s: decode unexpectedly succeeded", c.ID)
				}
				if len(dec.Diagnostics) == 0 {
					t.Fatalf("%s: decode rejection produced zero diagnostics", c.ID)
				}
				got := projectDiagnostics(dec.Diagnostics)
				if !reflect.DeepEqual(got, c.WantDiagnostics) {
					t.Fatalf("%s: decode diagnostics = %#v, want %#v",
						c.ID, got, c.WantDiagnostics)
				}
				if dec.Document.Version() != 0 {
					t.Fatalf("%s: decode-rejected Document.Version() = %d, want 0",
						c.ID, dec.Document.Version())
				}
			case stageNormalizeRejected:
				if !dec.Success() {
					t.Fatalf("%s: decode failed before normalize stage: %v",
						c.ID, dec.Diagnostics)
				}
				norm := Normalize(dec.Document)
				if norm.Success() {
					t.Fatalf("%s: normalize unexpectedly succeeded", c.ID)
				}
				if len(norm.Diagnostics) == 0 {
					t.Fatalf("%s: normalize rejection produced zero diagnostics", c.ID)
				}
				got := projectDiagnostics(norm.Diagnostics)
				if !reflect.DeepEqual(got, c.WantDiagnostics) {
					t.Fatalf("%s: normalize diagnostics = %#v, want %#v",
						c.ID, got, c.WantDiagnostics)
				}
			case stageNormalized:
				if !dec.Success() {
					t.Fatalf("%s: decode failed before normalize stage: %v",
						c.ID, dec.Diagnostics)
				}
				norm := Normalize(dec.Document)
				if !norm.Success() {
					t.Fatalf("%s: normalize failed: %v", c.ID, norm.Diagnostics)
				}
				if got := norm.Summary.SchemaVersion; got != c.SuccessSchema {
					t.Fatalf("%s: normalized schema_version = %d, want %d",
						c.ID, got, c.SuccessSchema)
				}
				if !norm.Summary.Valid() {
					t.Fatalf("%s: normalized summary is not Valid", c.ID)
				}
			}
		})
	}
}
