package gatesummary

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

// TestV2TruncatedEnvelopeRejectsWithCodeMalformedJSON binds the
// pre-schema envelope rejection of v2-truncated.json at the
// parent-package authority. The malformed-JSON fixture is rejected
// by the bounded reader before the schema stage is invoked; the
// schema subpackage classifies this as "not applicable" because
// the JSON decoder fails first. This test asserts the envelope
// produces the correct diagnostic code so the schema-side
// classification is fully bound.
//
// The test deliberately does NOT use t.Skip. It calls the real
// Decode authority and asserts a failure with CodeMalformedJSON.
func TestV2TruncatedEnvelopeRejectsWithCodeMalformedJSON(t *testing.T) {
	path := filepath.Join("testdata", "invalid", "v2-truncated.json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}

	res := Decode(bytes.NewReader(data))
	if res.Success() {
		t.Fatalf("v2-truncated.json must be rejected by the decoder envelope")
	}

	if len(res.Diagnostics) == 0 {
		t.Fatalf("v2-truncated.json must produce diagnostics")
	}

	found := false
	for _, d := range res.Diagnostics {
		if d.Code == CodeMalformedJSON {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("v2-truncated.json must surface CodeMalformedJSON; got %v",
			res.Diagnostics)
	}
}
