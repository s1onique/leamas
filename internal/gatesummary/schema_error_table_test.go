package gatesummary

import (
	"slices"
	"testing"

	jsonschema "github.com/santhosh-tekuri/jsonschema/v6"
	"github.com/santhosh-tekuri/jsonschema/v6/kind"
)

func TestSchemaErrorFrozenLeafTable(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		kind     jsonschema.ErrorKind
		location []string
		wantCode string
		wantPath string
	}{
		{name: "status enum root", kind: &kind.Enum{}, location: []string{"overall_status"},
			wantCode: CodeInvalidStatus, wantPath: "/overall_status"},
		{name: "status enum check", kind: &kind.Enum{}, location: []string{"checks", "12", "status"},
			wantCode: CodeInvalidStatus, wantPath: "/checks/12/status"},
		{name: "OID type", kind: &kind.Type{}, location: []string{"execution_head_oid"},
			wantCode: CodeInvalidOID, wantPath: "/execution_head_oid"},
		{name: "OID pattern", kind: &kind.Pattern{}, location: []string{"subject_tree_oid"},
			wantCode: CodeInvalidOID, wantPath: "/subject_tree_oid"},
		{name: "stdout hash", kind: &kind.Pattern{},
			location: []string{"checks", "0", "extras", "stdout_sha256"},
			wantCode: CodeInvalidOutputHash, wantPath: "/checks/0/extras/stdout_sha256"},
		{name: "stderr hash", kind: &kind.Pattern{},
			location: []string{"checks", "99", "extras", "stderr_sha256"},
			wantCode: CodeInvalidOutputHash, wantPath: "/checks/99/extras/stderr_sha256"},
		{name: "duration", kind: &kind.Minimum{},
			location: []string{"checks", "3", "extras", "duration_ms"},
			wantCode: CodeInvalidDuration, wantPath: "/checks/3/extras/duration_ms"},
		{name: "timestamp format", kind: &kind.Format{}, location: []string{"generated_at"},
			wantCode: CodeInvalidTimestamp, wantPath: "/generated_at"},
		{name: "timestamp min length", kind: &kind.MinLength{}, location: []string{"generated_at"},
			wantCode: CodeInvalidTimestamp, wantPath: "/generated_at"},
		{name: "known fallback", kind: &kind.Type{}, location: []string{"scope_id"},
			wantCode: CodeSchemaViolation, wantPath: "/scope_id"},
		{name: "unknown leaf fallback", kind: &kind.AllOf{}, location: []string{"scope_id"},
			wantCode: CodeSchemaViolation, wantPath: "/scope_id"},
		{name: "not fallback", kind: &kind.Not{}, location: []string{"scope_id"},
			wantCode: CodeSchemaViolation, wantPath: "/scope_id"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			node := validationLeaf(tc.kind, tc.location, v2SchemaID+"#/properties/example")
			got := translateValidationTree(validationRoot(node))
			assertDiagnostic(t, got, tc.wantCode, tc.wantPath)
		})
	}
}

func TestSchemaErrorCollectionLimitsUseStructuredValues(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		kind     jsonschema.ErrorKind
		location []string
		wantCode string
		wantPath string
		want     string
		got      string
	}{
		{name: "max items", kind: &kind.MaxItems{Got: 10001, Want: 10000},
			location: []string{"checks"}, wantCode: CodeCollectionLimit,
			wantPath: "/checks", want: "10000", got: "10001"},
		{name: "max length", kind: &kind.MaxLength{Got: 65537, Want: 65536},
			location: []string{"checks", "0", "extras", "argv", "0"},
			wantCode: CodeCollectionLimit, wantPath: "/checks/0/extras/argv/0",
			want: "65536", got: "65537"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ds := translateValidationTree(validationRoot(
				validationLeaf(tc.kind, tc.location, v2SchemaID+"#/properties/example"),
			))
			assertDiagnostic(t, ds, tc.wantCode, tc.wantPath)
			if ds[0].Expected != tc.want || ds[0].Observed != tc.got {
				t.Fatalf("expected/observed=(%q,%q), want (%q,%q)",
					ds[0].Expected, ds[0].Observed, tc.want, tc.got)
			}
		})
	}
}

func TestSchemaErrorRequiredAndAdditionalFanoutOrdering(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name      string
		kind      jsonschema.ErrorKind
		wantCode  string
		wantPaths []string
	}{
		{name: "required", kind: &kind.Required{Missing: []string{"z", "a/b", "a~b"}},
			wantCode:  CodeRequiredFieldMissing,
			wantPaths: []string{"/root/a~0b", "/root/a~1b", "/root/z"}},
		{name: "additional", kind: &kind.AdditionalProperties{
			Properties: []string{"z", "a/b", "a~b"},
		}, wantCode: CodeUnknownField,
			wantPaths: []string{"/root/a~0b", "/root/a~1b", "/root/z"}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ds := translateValidationTree(validationRoot(
				validationLeaf(tc.kind, []string{"root"}, v2SchemaID+"#/properties/root"),
			))
			if len(ds) != len(tc.wantPaths) {
				t.Fatalf("diagnostic count=%d, want %d: %+v", len(ds), len(tc.wantPaths), ds)
			}
			paths := make([]string, len(ds))
			for i, d := range ds {
				if d.Code != tc.wantCode {
					t.Fatalf("diagnostic[%d].Code=%s, want %s", i, d.Code, tc.wantCode)
				}
				paths[i] = d.Path
			}
			if !slices.Equal(paths, tc.wantPaths) {
				t.Fatalf("paths=%q, want %q", paths, tc.wantPaths)
			}
		})
	}
}

func TestPostDispatchSchemaVersionRowsAreInternal(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		kind     jsonschema.ErrorKind
		location []string
	}{
		{name: "required", kind: &kind.Required{Missing: []string{"other", "schema_version"}}},
		{name: "type", kind: &kind.Type{}, location: []string{"schema_version"}},
		{name: "const", kind: &kind.Const{}, location: []string{"schema_version"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ds := translateValidationTree(validationRoot(
				validationLeaf(tc.kind, tc.location, v2SchemaID+"#/properties/schema_version"),
			))
			assertDiagnostic(t, ds, CodeInternal, "/schema_version")
		})
	}
}

func validationRoot(causes ...*jsonschema.ValidationError) *jsonschema.ValidationError {
	return &jsonschema.ValidationError{
		SchemaURL: v2SchemaID + "#",
		ErrorKind: &kind.Schema{},
		Causes:    causes,
	}
}

func validationLeaf(k jsonschema.ErrorKind, loc []string, schemaURL string) *jsonschema.ValidationError {
	return &jsonschema.ValidationError{
		SchemaURL:        schemaURL,
		InstanceLocation: loc,
		ErrorKind:        k,
	}
}

func translateValidationTree(root *jsonschema.ValidationError) []Diagnostic {
	return schemaErrorTranslator{root: root}.translate()
}

func assertDiagnostic(t *testing.T, ds []Diagnostic, code, path string) {
	t.Helper()
	if len(ds) != 1 {
		t.Fatalf("diagnostic count=%d, want 1: %+v", len(ds), ds)
	}
	if ds[0].Code != code || ds[0].Path != path {
		t.Fatalf("diagnostic=(%s,%q), want (%s,%q): %+v",
			ds[0].Code, ds[0].Path, code, path, ds[0])
	}
}
