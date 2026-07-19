package gatesummary

import (
	"bytes"
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

func TestSchemaVersionPresenceAndContainerConsumption(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		value string
		want  json.Token
	}{
		{name: "null", value: "null", want: nil},
		{name: "object", value: `{}`, want: json.Delim('{')},
		{name: "nested discriminator", value: `{"schema_version": 2}`, want: json.Delim('{')},
		{name: "array", value: `[]`, want: json.Delim('[')},
		{name: "scalar array", value: `[2]`, want: json.Delim('[')},
		{name: "object array", value: `[{"schema_version": 2}]`, want: json.Delim('[')},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			data := []byte(`{"schema_version":` + tc.value + `,"after":true}`)
			env := scanEnvelope(data)
			if env.malformed || env.trailing {
				t.Fatalf("valid envelope rejected: malformed=%v trailing=%v diagnostics=%+v",
					env.malformed, env.trailing, env.diagnostics)
			}
			if !env.versionPresent {
				t.Fatal("schema_version must be marked present")
			}
			if env.versionToken != tc.want {
				t.Fatalf("version token=%#v, want %#v", env.versionToken, tc.want)
			}

			trace := decodeTrace{}
			res := decodeWithTrace(bytes.NewReader(data), &trace)
			assertOnlyCode(t, res, CodeInvalidVersionType)
			if trace.Stage != stageVersionProbe {
				t.Fatalf("owner=%s, want %s", trace.Stage, stageVersionProbe)
			}
			if trace.SchemaSelected != 0 || trace.SchemaInvoked || trace.WireDecoded {
				t.Fatalf("downstream work crossed version boundary: %+v", trace)
			}
		})
	}
}

func TestScanEnvelopeUsesTokenEOFContract(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name          string
		data          string
		wantMalformed bool
		wantTrailing  bool
	}{
		{name: "exact EOF", data: `{"schema_version":2}`},
		{name: "whitespace then EOF", data: "{\"schema_version\":2}\n\t "},
		{name: "second object", data: `{"schema_version":2}{}`, wantTrailing: true},
		{name: "second scalar", data: `{"schema_version":2} false`, wantTrailing: true},
		{name: "malformed trailing input", data: `{"schema_version":2} !`, wantMalformed: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := scanEnvelope([]byte(tc.data))
			if got.malformed != tc.wantMalformed || got.trailing != tc.wantTrailing {
				t.Fatalf("malformed=%v trailing=%v, want malformed=%v trailing=%v; diagnostics=%+v",
					got.malformed, got.trailing, tc.wantMalformed, tc.wantTrailing, got.diagnostics)
			}
		})
	}
}

func TestRequireEOFUsesNextToken(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name      string
		data      string
		wantExtra bool
		wantErr   bool
	}{
		{name: "exact EOF", data: `{}`},
		{name: "whitespace then EOF", data: "{}\n\t "},
		{name: "second value", data: `{} []`, wantExtra: true, wantErr: true},
		{name: "malformed suffix", data: `{} !`, wantErr: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			dec := json.NewDecoder(strings.NewReader(tc.data))
			var first any
			if err := dec.Decode(&first); err != nil {
				t.Fatalf("decode first value: %v", err)
			}
			err := requireEOF(dec)
			if (err != nil) != tc.wantErr {
				t.Fatalf("requireEOF error=%v, wantErr=%v", err, tc.wantErr)
			}
			var extra errExtraJSON
			if errors.As(err, &extra) != tc.wantExtra {
				t.Fatalf("extra-value classification=%v, want %v (err=%v)",
					errors.As(err, &extra), tc.wantExtra, err)
			}
		})
	}
}

func TestDecodeMalformedSuffixStopsBeforeSchema(t *testing.T) {
	t.Parallel()

	trace := decodeTrace{}
	res := decodeWithTrace(strings.NewReader(`{"schema_version":2} !`), &trace)
	assertOnlyCode(t, res, CodeMalformedJSON)
	if trace.Stage != stageSyntaxScan || trace.SchemaSelected != 0 ||
		trace.SchemaInvoked || trace.WireDecoded {
		t.Fatalf("malformed suffix crossed syntax boundary: %+v", trace)
	}
	if res.Diagnostics[0].Path != "" {
		t.Fatalf("malformed suffix path=%q, want root", res.Diagnostics[0].Path)
	}
}

func TestDecodeHostileVersionNestingDoesNotPanic(t *testing.T) {
	t.Parallel()

	const depth = 2048
	value := strings.Repeat("[", depth) + "2" + strings.Repeat("]", depth)
	data := []byte(`{"schema_version":` + value + `}`)
	defer func() {
		if recovered := recover(); recovered != nil {
			t.Fatalf("Decode panicked on %d nested containers: %v", depth, recovered)
		}
	}()

	trace := decodeTrace{}
	res := decodeWithTrace(bytes.NewReader(data), &trace)
	assertOnlyCode(t, res, CodeInvalidVersionType)
	if trace.Stage != stageVersionProbe || trace.SchemaInvoked || trace.WireDecoded {
		t.Fatalf("unexpected hostile-nesting trace: %+v", trace)
	}
}

func assertOnlyCode(t *testing.T, res Result, want string) {
	t.Helper()
	if res.Err != nil {
		t.Fatalf("unexpected operational error: %v", res.Err)
	}
	if res.Success() {
		t.Fatal("unexpected successful decode")
	}
	if len(res.Diagnostics) != 1 || res.Diagnostics[0].Code != want {
		t.Fatalf("diagnostics=%+v, want only %s", res.Diagnostics, want)
	}
}
