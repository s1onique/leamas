package gatesummary

import (
	"errors"
	"strings"
	"testing"

	jsonschema "github.com/santhosh-tekuri/jsonschema/v6"
	"github.com/santhosh-tekuri/jsonschema/v6/kind"
)

func TestDecodeTraceBoundedReadOwnership(t *testing.T) {
	t.Parallel()

	trace := decodeTrace{}
	res := decodeWithTrace(strings.NewReader(strings.Repeat("x", MaxDocumentBytes+1)), &trace)
	assertOnlyCode(t, res, CodeDocumentTooLarge)
	if trace.Stage != stageBoundedRead || trace.SchemaSelected != 0 ||
		trace.SchemaInvoked || trace.WireDecoded {
		t.Fatalf("unexpected bounded-read trace: %+v", trace)
	}
}

func TestResultSuccessRequiresCleanExclusiveDocument(t *testing.T) {
	t.Parallel()

	doc := newDocumentV1(V1Summary{SchemaVersion: 1})
	cases := []struct {
		name string
		res  Result
		want bool
	}{
		{name: "document only", res: Result{Document: doc}, want: true},
		{name: "zero document", res: Result{}},
		{name: "document and diagnostic", res: Result{
			Document:    doc,
			Diagnostics: []Diagnostic{newDiagnostic(CodeInternal, "/", "contradiction")},
		}},
		{name: "document and error", res: Result{Document: doc, Err: errors.New("boom")}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := tc.res.Success(); got != tc.want {
				t.Fatalf("Success()=%v, want %v", got, tc.want)
			}
		})
	}
}

func TestOperationalSchemaFailureSetsResultError(t *testing.T) {
	t.Parallel()

	deps := productionDecodeDependencies()
	cause := errors.New("injected schema failure")
	deps.validate = func(_ *jsonschema.Schema, _ []byte) error { return cause }
	trace := decodeTrace{}
	res := decodeWithDependencies(strings.NewReader(validV1JSON()), &trace, deps)

	assertOperationalInternal(t, res, cause)
	if trace.Stage != stageSchemaValidation || !trace.SchemaInvoked || trace.WireDecoded {
		t.Fatalf("unexpected schema-failure trace: %+v", trace)
	}
}

func TestOperationalWireDriftSetsResultError(t *testing.T) {
	t.Parallel()

	deps := productionDecodeDependencies()
	cause := errors.New("injected wire drift")
	deps.decodeStrict = func(_ []byte, _ Version) (strictDecodeResult, error) {
		return strictDecodeResult{}, cause
	}
	trace := decodeTrace{}
	res := decodeWithDependencies(strings.NewReader(validV1JSON()), &trace, deps)

	assertOperationalInternal(t, res, cause)
	if trace.Stage != stageWireDecode || !trace.SchemaInvoked || !trace.WireDecoded {
		t.Fatalf("unexpected wire-drift trace: %+v", trace)
	}
}

func TestImpossiblePostDispatchVersionFailureIsOperational(t *testing.T) {
	t.Parallel()

	deps := productionDecodeDependencies()
	deps.validate = func(_ *jsonschema.Schema, _ []byte) error {
		return &jsonschema.ValidationError{
			SchemaURL: v2SchemaID + "#",
			ErrorKind: &kind.Schema{},
			Causes: []*jsonschema.ValidationError{{
				SchemaURL:        v2SchemaID + "#",
				InstanceLocation: nil,
				ErrorKind:        &kind.Required{Missing: []string{"schema_version"}},
			}},
		}
	}
	trace := decodeTrace{}
	res := decodeWithDependencies(strings.NewReader(validV2JSON()), &trace, deps)

	if res.Err == nil {
		t.Fatal("impossible post-dispatch version failure must be operational")
	}
	if len(res.Diagnostics) != 1 || res.Diagnostics[0].Code != CodeInternal ||
		res.Diagnostics[0].Path != "/schema_version" {
		t.Fatalf("unexpected diagnostics: %+v", res.Diagnostics)
	}
	if res.Success() {
		t.Fatal("operational internal result cannot be successful")
	}
	if trace.Stage != stageSchemaValidation || !trace.SchemaInvoked || trace.WireDecoded {
		t.Fatalf("unexpected impossible-version trace: %+v", trace)
	}
}

func TestSchemaBootstrapFailureSetsResultError(t *testing.T) {
	t.Parallel()

	deps := productionDecodeDependencies()
	cause := errors.New("injected bootstrap failure")
	deps.schemas = func() (*compiledSchemas, error) { return nil, cause }
	trace := decodeTrace{}
	res := decodeWithDependencies(strings.NewReader(validV1JSON()), &trace, deps)

	if !errors.Is(res.Err, cause) {
		t.Fatalf("Err=%v, want wrapped %v", res.Err, cause)
	}
	if len(res.Diagnostics) != 0 || res.Success() {
		t.Fatalf("bootstrap failure result is contradictory: %+v", res)
	}
	if trace.Stage != stageSchemaValidation || trace.SchemaInvoked || trace.WireDecoded {
		t.Fatalf("unexpected bootstrap-failure trace: %+v", trace)
	}
}

func assertOperationalInternal(t *testing.T, res Result, cause error) {
	t.Helper()
	if !errors.Is(res.Err, cause) {
		t.Fatalf("Err=%v, want wrapped %v", res.Err, cause)
	}
	if res.Success() {
		t.Fatal("operational failure cannot be successful")
	}
	if len(res.Diagnostics) != 1 || res.Diagnostics[0].Code != CodeInternal {
		t.Fatalf("operational failure diagnostics=%+v, want one %s",
			res.Diagnostics, CodeInternal)
	}
}

func validV1JSON() string {
	return `{"schema_version":1,"generated_at":"2026-07-19T08:43:26Z",` +
		`"overall_status":"pass","checks":[]}`
}

func validV2JSON() string {
	return strings.Replace(v2Template, "%s", "2", 1)
}
