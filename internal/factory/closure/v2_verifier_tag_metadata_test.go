// SPDX-License-Identifier: Apache-2.0

package closure

// v2_verifier_tag_metadata_test.go covers Phase 5 of
// ACT-LEAMAS-FACTORY-CLOSURE-PROTOCOL-V2-VERIFIER-MAC-HANDOFF01-CORRECTION02C:
// the authoritative parser for the annotated-tag metadata
// contract. The tests drive ParseV2ClosureTagObjectHeaders,
// ParseV2ClosureTagMetadataTrailers, and BindV2ClosureTagMetadata
// directly so the structural rejections are pinned at the
// parser boundary rather than at the orchestrator surface.

import (
	"strings"
	"testing"
)

// validTagObjectBody returns a hand-crafted annotated-tag
// object whose message body satisfies the contract v1.
func validTagObjectBody() string {
	return "" +
		"object 1111111111111111111111111111111111111111\n" +
		"type commit\n" +
		"tag v2-anno-ok\n" +
		"tagger Tagger <tagger@example.invalid> 1700000000 +0000\n" +
		"\n" +
		"factory: close ACT-FOO\n" +
		"\n" +
		"Leamas-Closure-Protocol-Version: 2\n" +
		"Leamas-Plan-Contract-Version: 1\n" +
		"Leamas-Subject-Commit: 1111111111111111111111111111111111111111\n" +
		"Leamas-Freeze-Commit: 2222222222222222222222222222222222222222\n" +
		"Leamas-Closure-Commit: 3333333333333333333333333333333333333333\n" +
		"Leamas-Plan-Path: plan/plan.json\n" +
		"Leamas-Manifest-Path: manifest/manifest.json\n"
}

// TestParseV2ClosureTagObjectHeadersHappyPath pins the
// happy path: all four headers parsed, body preserved.
func TestParseV2ClosureTagObjectHeadersHappyPath(t *testing.T) {
	headers, body, diags := ParseV2ClosureTagObjectHeaders([]byte(validTagObjectBody()))
	if len(diags) != 0 {
		t.Fatalf("happy path must produce no diagnostics, got %v", diags)
	}
	if headers.Object != "1111111111111111111111111111111111111111" {
		t.Fatalf("object header = %q", headers.Object)
	}
	if headers.Type != "commit" {
		t.Fatalf("type header = %q", headers.Type)
	}
	if headers.Tag != "v2-anno-ok" {
		t.Fatalf("tag header = %q", headers.Tag)
	}
	if !strings.Contains(string(body), "Leamas-Closure-Protocol-Version: 2") {
		t.Fatalf("body must contain trailer bytes, got %q", string(body))
	}
}

// TestParseV2ClosureTagObjectHeadersRejectsEmpty ensures
// the parser surfaces a malformed diagnostic for an empty
// raw object rather than silently succeeding.
func TestParseV2ClosureTagObjectHeadersRejectsEmpty(t *testing.T) {
	_, _, diags := ParseV2ClosureTagObjectHeaders(nil)
	if len(diags) != 1 {
		t.Fatalf("expected exactly one diagnostic, got %v", diags)
	}
	if diags[0].Code != V2VerifierClosureTagMetadataMalformed {
		t.Fatalf("code = %v, want malformed", diags[0].Code)
	}
}

// TestParseV2ClosureTagObjectHeadersRejectsMissingSeparator
// pins the LF+LF separator requirement.
func TestParseV2ClosureTagObjectHeadersRejectsMissingSeparator(t *testing.T) {
	raw := []byte("object 1111111111111111111111111111111111111111\ntype commit\ntag t\ntagger x\nthis-is-not-separated")
	_, _, diags := ParseV2ClosureTagObjectHeaders(raw)
	if len(diags) != 1 || diags[0].Code != V2VerifierClosureTagMetadataMalformed {
		t.Fatalf("expected one malformed diagnostic, got %v", diags)
	}
}

// TestParseV2ClosureTagObjectHeadersRejectsNUL pins the
// NUL-byte rejection path.
func TestParseV2ClosureTagObjectHeadersRejectsNUL(t *testing.T) {
	raw := []byte("object 1111111111111111111111111111111111111111\ntype commit\ntag t\n\n\x00body")
	_, _, diags := ParseV2ClosureTagObjectHeaders(raw)
	if len(diags) != 1 || diags[0].Code != V2VerifierClosureTagMetadataMalformed {
		t.Fatalf("expected one malformed diagnostic, got %v", diags)
	}
}

// TestParseV2ClosureTagObjectHeadersRejectsDuplicate pins
// the duplicate-header rejection path.
func TestParseV2ClosureTagObjectHeadersRejectsDuplicate(t *testing.T) {
	raw := []byte("object 1111111111111111111111111111111111111111\n" +
		"object 2222222222222222222222222222222222222222\n" +
		"type commit\n" +
		"tag t\n" +
		"\n" +
		"body")
	_, _, diags := ParseV2ClosureTagObjectHeaders(raw)
	if len(diags) != 1 || diags[0].Code != V2VerifierClosureTagMetadataMalformed {
		t.Fatalf("expected one malformed diagnostic, got %v", diags)
	}
}

// TestParseV2ClosureTagObjectHeadersRejectsUnknown pins the
// unknown-header rejection path.
func TestParseV2ClosureTagObjectHeadersRejectsUnknown(t *testing.T) {
	raw := []byte("object 1111111111111111111111111111111111111111\n" +
		"type commit\n" +
		"tag t\n" +
		"extra yes\n" +
		"\n" +
		"body")
	_, _, diags := ParseV2ClosureTagObjectHeaders(raw)
	if len(diags) != 1 || diags[0].Code != V2VerifierClosureTagMetadataMalformed {
		t.Fatalf("expected one malformed diagnostic, got %v", diags)
	}
}

// TestParseV2ClosureTagObjectHeadersRejectsWrongType pins
// the wrong-target-type rejection path.
func TestParseV2ClosureTagObjectHeadersRejectsWrongType(t *testing.T) {
	raw := []byte("object 1111111111111111111111111111111111111111\n" +
		"type blob\n" +
		"tag t\n" +
		"\n" +
		"body")
	_, _, diags := ParseV2ClosureTagObjectHeaders(raw)
	if len(diags) != 1 || diags[0].Code != V2VerifierClosureTagMetadataMalformed {
		t.Fatalf("expected one malformed diagnostic, got %v", diags)
	}
}

// TestParseV2ClosureTagMetadataTrailersHappyPath pins the
// happy path: all seven trailers populate the typed
// metadata struct with no diagnostics.
func TestParseV2ClosureTagMetadataTrailersHappyPath(t *testing.T) {
	body := []byte("factory: close ACT-FOO\n\n" +
		"Leamas-Closure-Protocol-Version: 2\n" +
		"Leamas-Plan-Contract-Version: 1\n" +
		"Leamas-Subject-Commit: 1111111111111111111111111111111111111111\n" +
		"Leamas-Freeze-Commit: 2222222222222222222222222222222222222222\n" +
		"Leamas-Closure-Commit: 3333333333333333333333333333333333333333\n" +
		"Leamas-Plan-Path: plan/plan.json\n" +
		"Leamas-Manifest-Path: manifest/manifest.json\n")
	meta, diags := ParseV2ClosureTagMetadataTrailers(body)
	if len(diags) != 0 {
		t.Fatalf("happy path diagnostics = %v", diags)
	}
	if meta.ClosureProtocolVersion != ClosureProtocolV2 {
		t.Fatalf("protocol version = %q", meta.ClosureProtocolVersion)
	}
	if int(meta.PlanContractVersion) != 1 {
		t.Fatalf("plan contract version = %d", meta.PlanContractVersion)
	}
	if meta.SubjectCommit != "1111111111111111111111111111111111111111" {
		t.Fatalf("subject commit = %q", meta.SubjectCommit)
	}
	if meta.PlanPath != "plan/plan.json" {
		t.Fatalf("plan path = %q", meta.PlanPath)
	}
}

// TestParseV2ClosureTagMetadataTrailersMissingKeys pins the
// missing-key rejection path: every absent required key
// emits its own typed diagnostic.
func TestParseV2ClosureTagMetadataTrailersMissingKeys(t *testing.T) {
	meta, diags := ParseV2ClosureTagMetadataTrailers([]byte("only prose, no trailers\n"))
	if len(diags) != len(V2TagMetadataAllKeys) {
		t.Fatalf("expected %d diagnostics, got %d", len(V2TagMetadataAllKeys), len(diags))
	}
	for _, d := range diags {
		if d.Code != V2VerifierClosureTagMetadataMissing {
			t.Fatalf("expected missing code, got %v", d.Code)
		}
	}
	if meta.SubjectCommit != "" || meta.FreezeCommit != "" || meta.ClosureCommit != "" {
		t.Fatalf("metadata must be zero-valued, got %+v", meta)
	}
}

// TestParseV2ClosureTagMetadataTrailersDuplicateKeys pins
// the duplicate-key rejection path.
func TestParseV2ClosureTagMetadataTrailersDuplicateKeys(t *testing.T) {
	body := []byte("Leamas-Closure-Protocol-Version: 2\n" +
		"Leamas-Closure-Protocol-Version: 2\n")
	_, diags := ParseV2ClosureTagMetadataTrailers(body)
	found := false
	for _, d := range diags {
		if d.Code == V2VerifierClosureTagMetadataDuplicate {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected duplicate diagnostic, got %v", diags)
	}
}

// TestParseV2ClosureTagMetadataTrailersUnknownKey pins
// the unknown-Leamas-key rejection path.
func TestParseV2ClosureTagMetadataTrailersUnknownKey(t *testing.T) {
	body := []byte("Leamas-Unknown-Key: oops\n")
	_, diags := ParseV2ClosureTagMetadataTrailers(body)
	found := false
	for _, d := range diags {
		if d.Code == V2VerifierClosureTagMetadataUnknown {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected unknown diagnostic, got %v", diags)
	}
}

// TestParseV2ClosureTagMetadataTrailersAbbreviatedOID pins
// the abbreviated-OID rejection path.
func TestParseV2ClosureTagMetadataTrailersAbbreviatedOID(t *testing.T) {
	body := []byte("Leamas-Closure-Protocol-Version: 2\n" +
		"Leamas-Plan-Contract-Version: 1\n" +
		"Leamas-Subject-Commit: deadbeef\n")
	_, diags := ParseV2ClosureTagMetadataTrailers(body)
	found := false
	for _, d := range diags {
		if d.Code == V2VerifierClosureTagMetadataMalformed {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected malformed diagnostic, got %v", diags)
	}
}

// TestBindV2ClosureTagMetadataMatchesAndMismatches pins the
// binder: every mismatching field emits a typed diagnostic.
func TestBindV2ClosureTagMetadataMatchesAndMismatches(t *testing.T) {
	observed := V2ClosureTagMetadata{
		ClosureProtocolVersion: ClosureProtocolV2,
		PlanContractVersion:    PlanContractV1,
		SubjectCommit:          "1111111111111111111111111111111111111111",
		FreezeCommit:           "2222222222222222222222222222222222222222",
		ClosureCommit:          "9999999999999999999999999999999999999999",
		PlanPath:               "plan/plan.json",
		ManifestPath:           "manifest/wrong.json",
	}
	diags := BindV2ClosureTagMetadata(
		observed,
		"1111111111111111111111111111111111111111",
		"2222222222222222222222222222222222222222",
		"3333333333333333333333333333333333333333",
		"plan/plan.json",
		"manifest/manifest.json",
		ClosureProtocolV2,
		PlanContractV1,
	)
	if len(diags) != 2 {
		t.Fatalf("expected 2 mismatch diagnostics, got %d (%v)", len(diags), diags)
	}
	for _, d := range diags {
		if d.Code != V2VerifierClosureTagMetadataMismatch {
			t.Fatalf("code = %v, want mismatch", d.Code)
		}
	}
}
