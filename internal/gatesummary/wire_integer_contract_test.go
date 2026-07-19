package gatesummary

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

const (
	int64MaxWire  = "9223372036854775807"
	int64OverMax  = "9223372036854775808"
	int64MinWire  = "-9223372036854775808"
	int64UnderMin = "-9223372036854775809"
)

var hundredsDigitWireInteger = strings.Repeat("9", 512)

func TestDecodePreservesSchemaValidWireIntegers(t *testing.T) {
	nonNegative := []string{
		int64MaxWire,
		int64OverMax,
		hundredsDigitWireInteger,
		"1.0",
		"1e3",
	}
	allSigns := append(append([]string{}, nonNegative...),
		int64MinWire, int64UnderMin, "-1e3")

	fields := []struct {
		name     string
		values   []string
		document func(string) string
		observed func(*testing.T, Document) []string
	}{
		{
			name:     "v1 duration_ms",
			values:   allSigns,
			document: validV1IntegerDocument,
			observed: func(t *testing.T, doc Document) []string {
				v1, ok := doc.V1()
				if !ok || v1.Checks[0].DurationMs == nil {
					t.Fatal("decoded document is missing v1 duration_ms")
				}
				return []string{fmt.Sprint(*v1.Checks[0].DurationMs)}
			},
		},
		{
			name:     "v2 exit_code",
			values:   allSigns,
			document: validV2ExitCodeDocument,
			observed: func(t *testing.T, doc Document) []string {
				v2, ok := doc.V2()
				if !ok || v2.Checks[0].Extras.ExitCode == nil {
					t.Fatal("decoded document is missing v2 exit_code")
				}
				return []string{fmt.Sprint(*v2.Checks[0].Extras.ExitCode)}
			},
		},
		{
			name:     "v2 duration_ms",
			values:   nonNegative,
			document: validV2DurationDocument,
			observed: func(t *testing.T, doc Document) []string {
				v2, ok := doc.V2()
				if !ok {
					t.Fatal("decoded document is not v2")
				}
				return []string{fmt.Sprint(v2.Checks[0].Extras.DurationMs)}
			},
		},
		{
			name:     "v2 test counts",
			values:   nonNegative,
			document: validV2CountsDocument,
			observed: observedV2Counts,
		},
	}

	for _, field := range fields {
		for _, raw := range field.values {
			t.Run(field.name+"/"+integerCaseName(raw), func(t *testing.T) {
				res := Decode(strings.NewReader(field.document(raw)))
				if !res.Success() {
					t.Fatalf("schema-valid integer rejected: diagnostics=%+v err=%v",
						res.Diagnostics, res.Err)
				}
				for _, got := range field.observed(t, res.Document) {
					if got != raw {
						t.Fatalf("wire integer=%q, want exact %q", got, raw)
					}
				}
			})
		}
	}
}

func TestWireIntegerConversions(t *testing.T) {
	cases := []struct {
		raw       string
		bigWant   string
		int64Want int64
		int64OK   bool
	}{
		{raw: int64MaxWire, bigWant: int64MaxWire, int64Want: 9223372036854775807, int64OK: true},
		{raw: int64OverMax, bigWant: int64OverMax},
		{raw: hundredsDigitWireInteger, bigWant: hundredsDigitWireInteger},
		{raw: int64MinWire, bigWant: int64MinWire, int64Want: -9223372036854775808, int64OK: true},
		{raw: int64UnderMin, bigWant: int64UnderMin},
		{raw: "1.0", bigWant: "1", int64Want: 1, int64OK: true},
		{raw: "1e3", bigWant: "1000", int64Want: 1000, int64OK: true},
		{raw: "-1E+3", bigWant: "-1000", int64Want: -1000, int64OK: true},
	}

	for _, tc := range cases {
		t.Run(integerCaseName(tc.raw), func(t *testing.T) {
			var got WireInteger
			if err := json.Unmarshal([]byte(tc.raw), &got); err != nil {
				t.Fatalf("UnmarshalJSON(%q): %v", tc.raw, err)
			}
			if got.String() != tc.raw {
				t.Fatalf("String()=%q, want exact %q", got.String(), tc.raw)
			}
			bigValue, ok := got.BigInt()
			if !ok || bigValue.String() != tc.bigWant {
				t.Fatalf("BigInt()=(%v, %v), want (%s, true)", bigValue, ok, tc.bigWant)
			}
			int64Value, int64OK := got.Int64()
			if int64Value != tc.int64Want || int64OK != tc.int64OK {
				t.Fatalf("Int64()=(%d, %v), want (%d, %v)",
					int64Value, int64OK, tc.int64Want, tc.int64OK)
			}
		})
	}
}

func TestWireIntegerRejectsNonIntegerJSON(t *testing.T) {
	for _, raw := range []string{"null", `"1"`, "true", `{}`, `[]`, "1.5", "1e-1"} {
		t.Run(raw, func(t *testing.T) {
			var got WireInteger
			if err := json.Unmarshal([]byte(raw), &got); err == nil {
				t.Fatalf("UnmarshalJSON(%q) succeeded with %q", raw, got.String())
			}
		})
	}
}

func TestWireIntegerZeroValueHasNoConversion(t *testing.T) {
	var got WireInteger
	if got.String() != "" {
		t.Fatalf("zero String()=%q, want empty", got.String())
	}
	if value, ok := got.BigInt(); value != nil || ok {
		t.Fatalf("zero BigInt()=(%v, %v), want (nil, false)", value, ok)
	}
	if value, ok := got.Int64(); value != 0 || ok {
		t.Fatalf("zero Int64()=(%d, %v), want (0, false)", value, ok)
	}
}

func TestDecodeFindsDuplicateAfterHugeWireInteger(t *testing.T) {
	data := fmt.Sprintf(`{
		"schema_version":1,
		"generated_at":"2026-07-19T08:43:26Z",
		"overall_status":"pass",
		"checks":[{
			"name":"check",
			"status":"pass",
			"duration_ms":%s,
			"status":"pass"
		}]
	}`, hundredsDigitWireInteger)

	trace := decodeTrace{}
	res := decodeWithTrace(strings.NewReader(data), &trace)
	assertOnlyCode(t, res, CodeDuplicateKey)
	if res.Diagnostics[0].Path != "/checks/0/status" {
		t.Fatalf("duplicate path=%q, want /checks/0/status", res.Diagnostics[0].Path)
	}
	if trace.Stage != stageDuplicateKeyScan || trace.SchemaInvoked || trace.WireDecoded {
		t.Fatalf("huge integer crossed duplicate-key boundary: %+v", trace)
	}
}

func observedV2Counts(t *testing.T, doc Document) []string {
	t.Helper()
	v2, ok := doc.V2()
	if !ok {
		t.Fatal("decoded document is not v2")
	}
	check := v2.Checks[0]
	if check.Total == nil || check.PassCount == nil || check.FailCount == nil ||
		check.SkipCount == nil || check.UnavailableCount == nil {
		t.Fatal("decoded document is missing v2 test counts")
	}
	return []string{
		fmt.Sprint(*check.Total),
		fmt.Sprint(*check.PassCount),
		fmt.Sprint(*check.FailCount),
		fmt.Sprint(*check.SkipCount),
		fmt.Sprint(*check.UnavailableCount),
	}
}

func validV1IntegerDocument(raw string) string {
	return fmt.Sprintf(`{
		"schema_version":1,
		"generated_at":"2026-07-19T08:43:26Z",
		"overall_status":"pass",
		"checks":[{"name":"check","status":"pass","duration_ms":%s}]
	}`, raw)
}

func validV2ExitCodeDocument(raw string) string {
	return validV2WireIntegerDocument(raw, "0", "")
}

func validV2DurationDocument(raw string) string {
	return validV2WireIntegerDocument("null", raw, "")
}

func validV2CountsDocument(raw string) string {
	counts := fmt.Sprintf(
		`,"total":%[1]s,"pass_count":%[1]s,"fail_count":%[1]s,`+
			`"skip_count":%[1]s,"unavailable_count":%[1]s`, raw)
	return validV2WireIntegerDocument("0", "0", counts)
}

func validV2WireIntegerDocument(exitCode, duration, counts string) string {
	const emptySHA256 = "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
	return fmt.Sprintf(`{
		"schema_version":2,
		"generated_at":"2026-07-19T08:43:26Z",
		"scope_id":"ACT-X",
		"scope_status":"CLOSED",
		"scope_disposition":"d",
		"parent_act":"",
		"parent_status":"CLOSED",
		"parent_disposition":"d",
		"overall_status":"pass",
		"overall_disposition":"d",
		"execution_head_oid":"0123456789abcdef0123456789abcdef01234567",
		"execution_tree_oid":"0123456789abcdef0123456789abcdef01234567",
		"subject_tree_oid":"0123456789abcdef0123456789abcdef01234567",
		"worktree_clean_before":true,
		"worktree_clean_after":true,
		"checks":[{
			"name":"check",
			"scope":"ROOT",
			"status":"pass",
			"evidence":"e",
			"detail":"d",
			"extras":{
				"argv":[],
				"exit_code":%s,
				"duration_ms":%s,
				"stdout_sha256":"%s",
				"stderr_sha256":"%s"
			}%s
		}]
	}`, exitCode, duration, emptySHA256, emptySHA256, counts)
}

func integerCaseName(raw string) string {
	switch raw {
	case int64MaxWire:
		return "int64-max"
	case int64OverMax:
		return "int64-max-plus-one"
	case int64MinWire:
		return "int64-min"
	case int64UnderMin:
		return "int64-min-minus-one"
	case "1.0":
		return "decimal-integer"
	case "1e3":
		return "positive-exponent-integer"
	case "-1e3", "-1E+3":
		return "negative-exponent-integer"
	default:
		return "hundreds-of-digits"
	}
}
