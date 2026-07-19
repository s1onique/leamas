package gatesummary

import (
	"strings"
	"testing"
)

func TestDecodeV1Minimal(t *testing.T) {
	data := readFixture(t, "testdata/valid/v1-minimal.json")
	res := Decode(strings.NewReader(string(data)))
	if !res.Success() {
		t.Fatalf("expected success, got diags=%v err=%v", res.Diagnostics, res.Err)
	}
	if res.Document.Version() != Version1 {
		t.Fatalf("expected v1, got %s", res.Document.Version())
	}
	v1, ok := res.Document.V1()
	if !ok {
		t.Fatal("expected V1() to succeed")
	}
	if v1.SchemaVersion != 1 {
		t.Errorf("schema_version=%d, want 1", v1.SchemaVersion)
	}
	if v1.Tool != nil {
		t.Errorf("expected Tool absent, got %v", *v1.Tool)
	}
}

func TestDecodeV1Full(t *testing.T) {
	data := readFixture(t, "testdata/valid/v1-full.json")
	res := Decode(strings.NewReader(string(data)))
	if !res.Success() {
		t.Fatalf("expected success, got diags=%v", res.Diagnostics)
	}
	v1, ok := res.Document.V1()
	if !ok {
		t.Fatal("expected V1()")
	}
	if v1.Tool == nil {
		t.Fatal("expected Tool to be present in v1-full")
	}
	if *v1.Tool != "leamas factory gate" {
		t.Errorf("tool=%q", *v1.Tool)
	}
	if len(v1.Checks) != 4 {
		t.Errorf("expected 4 checks, got %d", len(v1.Checks))
	}
}

func TestDecodeV1RejectsUnknownField(t *testing.T) {
	data := readFixture(t, "testdata/invalid/v1-unknown-field.json")
	res := Decode(strings.NewReader(string(data)))
	if res.Success() {
		t.Fatal("expected reject")
	}
	var found bool
	for _, d := range res.Diagnostics {
		if d.Code == CodeUnknownField {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected %s, got %+v", CodeUnknownField, res.Diagnostics)
	}
}

func TestDecodeDispatchesV2Document(t *testing.T) {
	data := readFixture(t, "testdata/valid/v2-minimal.json")
	res := Decode(strings.NewReader(string(data)))
	if !res.Success() {
		t.Fatalf("expected success for v2 fixture, got %v", res.Diagnostics)
	}
	if res.Document.Version() != Version2 {
		t.Fatalf("expected v2, got %s", res.Document.Version())
	}
}
