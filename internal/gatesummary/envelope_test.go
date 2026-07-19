package gatesummary

import (
	"testing"
)

func TestScanEnvelopeEmptyObject(t *testing.T) {
	res := scanEnvelope([]byte("{}"))
	if res.malformed {
		t.Fatalf("empty object must not be malformed: %+v", res.diagnostics)
	}
	if res.trailing {
		t.Fatalf("empty object must not be trailing: %+v", res.diagnostics)
	}
}

func TestScanEnvelopeValidObject(t *testing.T) {
	data := []byte(`{"schema_version": 1, "x": 2}`)
	res := scanEnvelope(data)
	if res.malformed {
		t.Fatalf("valid object must not be malformed: %+v", res.diagnostics)
	}
	if res.trailing {
		t.Fatalf("valid object must not be trailing")
	}
	if res.versionToken == nil {
		t.Fatal("expected schema_version token")
	}
}

func TestScanEnvelopeTruncated(t *testing.T) {
	res := scanEnvelope([]byte(`{"schema_version": 1,`))
	if !res.malformed {
		t.Fatalf("truncated object must be malformed: %+v", res.diagnostics)
	}
	if len(res.diagnostics) == 0 {
		t.Fatal("expected at least one diagnostic")
	}
	if res.diagnostics[0].Code != CodeMalformedJSON {
		t.Fatalf("expected %s, got %s", CodeMalformedJSON, res.diagnostics[0].Code)
	}
}

func TestScanEnvelopeTopLevelArray(t *testing.T) {
	res := scanEnvelope([]byte(`[1, 2, 3]`))
	if !res.malformed {
		t.Fatalf("top-level array must be malformed")
	}
	if res.diagnostics[0].Code != CodeMalformedJSON {
		t.Fatalf("expected %s, got %s", CodeMalformedJSON, res.diagnostics[0].Code)
	}
}

func TestScanEnvelopeTopLevelScalar(t *testing.T) {
	res := scanEnvelope([]byte(`42`))
	if !res.malformed {
		t.Fatalf("top-level scalar must be malformed")
	}
}

func TestScanEnvelopeTrailing(t *testing.T) {
	res := scanEnvelope([]byte(`{} {"x": 1}`))
	if !res.trailing {
		t.Fatalf("expected trailing=true")
	}
	if len(res.diagnostics) == 0 || res.diagnostics[0].Code != CodeTrailingJSON {
		t.Fatalf("expected %s diagnostic, got %+v", CodeTrailingJSON, res.diagnostics)
	}
}

func TestScanEnvelopeMalformedWinsOverDuplicates(t *testing.T) {
	data := []byte(`{"schema_version": 1, "schema_version":`)
	res := scanEnvelope(data)
	if !res.malformed {
		t.Fatalf("malformed must win")
	}
	if res.diagnostics[0].Code != CodeMalformedJSON {
		t.Fatalf("expected malformed, got %s", res.diagnostics[0].Code)
	}
}

func TestScanEnvelopeNoTopLevel(t *testing.T) {
	res := scanEnvelope([]byte(``))
	if !res.malformed {
		t.Fatal("empty input must be malformed")
	}
}
