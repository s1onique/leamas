package gatesummary

import (
	"bytes"
	"path/filepath"
	"slices"
	"testing"
)

type corpusStageCase struct {
	fixture       string
	wantSuccess   bool
	wantCode      string
	wantPaths     []string
	wantStage     stage
	wantSelected  Version
	schemaInvoked bool
	wireDecoded   bool
}

func TestCorpusStageEvidence(t *testing.T) {
	cases := corpusStageCases()
	if len(cases) != 41 {
		t.Fatalf("stage matrix has %d fixtures, want 41", len(cases))
	}
	for _, tc := range cases {
		t.Run(tc.fixture, func(t *testing.T) {
			data := readFixture(t, filepath.Join("testdata", tc.fixture))
			trace := decodeTrace{}
			res := decodeWithTrace(bytes.NewReader(data), &trace)

			if res.Err != nil {
				t.Fatalf("unexpected operational error: %v", res.Err)
			}
			if res.Success() != tc.wantSuccess {
				t.Fatalf("Success()=%v, want %v; diagnostics=%+v",
					res.Success(), tc.wantSuccess, res.Diagnostics)
			}
			if trace.Stage != tc.wantStage || trace.SchemaSelected != tc.wantSelected ||
				trace.SchemaInvoked != tc.schemaInvoked || trace.WireDecoded != tc.wireDecoded {
				t.Fatalf("trace=%+v, want stage=%s selected=%s schema=%v wire=%v",
					trace, tc.wantStage, tc.wantSelected, tc.schemaInvoked, tc.wireDecoded)
			}
			assertCorpusDiagnostics(t, res.Diagnostics, tc.wantCode, tc.wantPaths)
		})
	}
}

func corpusStageCases() []corpusStageCase {
	return []corpusStageCase{
		acceptFixture("valid/v1-minimal.json", Version1),
		acceptFixture("valid/v1-full.json", Version1),
		acceptFixture("valid/v2-minimal.json", Version2),
		acceptFixture("valid/v2-full.json", Version2),
		acceptFixture("valid/v2-root-scope.json", Version2),
		acceptFixture("valid/v2-clinemm-microc3.json", Version2),
		acceptFixture("valid/v2-leamas-self-hosted.json", Version2),

		schemaFixture("invalid/v1-unknown-field.json", Version1, CodeUnknownField,
			"/scope_id", "/scope_status"),
		versionFixture("invalid/v2-missing-schema-version.json", stageVersionProbe, CodeVersionMissing),
		versionFixture("invalid/v2-schema-version-string.json", stageVersionProbe, CodeInvalidVersionType),
		versionFixture("invalid/v2-schema-version-decimal.json", stageVersionProbe, CodeInvalidVersionType),
		versionFixture("invalid/v2-schema-version-zero.json", stageVersionDispatch, CodeUnsupportedVersion),
		versionFixture("invalid/v2-schema-version-negative.json", stageVersionDispatch, CodeUnsupportedVersion),
		versionFixture("invalid/v2-unsupported-version-3.json", stageVersionDispatch, CodeUnsupportedVersion),
		schemaFixture("invalid/v2-empty-generated-at.json", Version2, CodeInvalidTimestamp,
			"/generated_at"),
		schemaFixture("invalid/v2-invalid-timestamp.json", Version2, CodeInvalidTimestamp,
			"/generated_at"),
		schemaFixture("invalid/v2-missing-execution-head-oid.json", Version2,
			CodeRequiredFieldMissing, "/execution_head_oid"),
		schemaFixture("invalid/v2-null-execution-head-oid.json", Version2,
			CodeInvalidOID, "/execution_head_oid"),
		schemaFixture("invalid/v2-uppercase-oid.json", Version2,
			CodeInvalidOID, "/execution_head_oid"),
		schemaFixture("invalid/v2-partial-test-totals.json", Version2,
			CodePartialTestTotals, "/checks/0"),
		schemaFixture("invalid/v2-negative-duration.json", Version2,
			CodeInvalidDuration, "/checks/0/extras/duration_ms"),
		schemaFixture("invalid/v2-invalid-hash.json", Version2,
			CodeInvalidOutputHash, "/checks/0/extras/stdout_sha256"),
		syntaxFixture("invalid/v2-trailing-second-value.json", CodeTrailingJSON),
		schemaFixture("invalid/v2-unknown-field.json", Version2, CodeUnknownField, "/tool"),
		schemaFixture("invalid/v2-bad-status-enum.json", Version2,
			CodeInvalidStatus, "/checks/0/status"),
		schemaFixture("invalid/v2-lower-lifecycle.json", Version2,
			CodeInvalidStatus, "/parent_status", "/scope_status"),
		corpusStageCase{
			fixture: "invalid/v2-truncated.json", wantCode: CodeMalformedJSON,
			wantPaths: []string{"/scope_id"}, wantStage: stageSyntaxScan,
		},

		acceptFixture("invalid/v2-pass-nonzero-exit.json", Version2),
		acceptFixture("invalid/v2-fail-exit-zero.json", Version2),
		acceptFixture("invalid/v2-skip-nonnull-exit.json", Version2),
		acceptFixture("invalid/v2-unavailable-nonnull-exit.json", Version2),
		acceptFixture("invalid/v2-test-total-mismatch.json", Version2),
		acceptFixture("invalid/v2-duplicate-check-name.json", Version2),
		acceptFixture("invalid/v2-overall-mismatch.json", Version2),
		acceptFixture("invalid/v2-scope-closed-dirty-after.json", Version2),

		duplicateFixture("duplicate-keys/v2-duplicate-schema-version.json", "/schema_version"),
		duplicateFixture("duplicate-keys/v2-duplicate-top-level-field.json", "/scope_id"),
		duplicateFixture("duplicate-keys/v2-duplicate-nested-field.json", "/checks/0/name"),

		acceptFixture("limits/v2-document-size-shape.json", Version2),
		acceptFixture("limits/v2-checks-boundary-shape.json", Version2),
		acceptFixture("limits/v2-checks-over-boundary-shape.json", Version2),
	}
}

func acceptFixture(path string, version Version) corpusStageCase {
	return corpusStageCase{
		fixture: path, wantSuccess: true, wantStage: stageWireDecode,
		wantSelected: version, schemaInvoked: true, wireDecoded: true,
	}
}

func schemaFixture(path string, version Version, code string, paths ...string) corpusStageCase {
	return corpusStageCase{
		fixture: path, wantCode: code, wantPaths: paths, wantStage: stageSchemaValidation,
		wantSelected: version, schemaInvoked: true,
	}
}

func versionFixture(path string, owner stage, code string) corpusStageCase {
	return corpusStageCase{
		fixture: path, wantCode: code, wantPaths: []string{"/schema_version"}, wantStage: owner,
	}
}

func syntaxFixture(path, code string) corpusStageCase {
	return corpusStageCase{fixture: path, wantCode: code, wantPaths: []string{""}, wantStage: stageSyntaxScan}
}

func duplicateFixture(path, diagnosticPath string) corpusStageCase {
	return corpusStageCase{
		fixture: path, wantCode: CodeDuplicateKey, wantPaths: []string{diagnosticPath},
		wantStage: stageDuplicateKeyScan,
	}
}

func assertCorpusDiagnostics(t *testing.T, got []Diagnostic, code string, paths []string) {
	t.Helper()
	if code == "" {
		if len(got) != 0 {
			t.Fatalf("unexpected diagnostics: %+v", got)
		}
		return
	}
	if len(got) != len(paths) {
		t.Fatalf("diagnostic count=%d, want %d: %+v", len(got), len(paths), got)
	}
	gotPaths := make([]string, len(got))
	for i, d := range got {
		if d.Code != code {
			t.Fatalf("diagnostic[%d].Code=%s, want %s", i, d.Code, code)
		}
		gotPaths[i] = d.Path
	}
	if !slices.Equal(gotPaths, paths) {
		t.Fatalf("diagnostic paths=%q, want %q", gotPaths, paths)
	}
}
