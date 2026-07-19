package gatesummary

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestDiagnosticDeterminismV2Minimal(t *testing.T) {
	data := readFixture(t, "testdata/valid/v2-minimal.json")
	a := Decode(strings.NewReader(string(data)))
	b := Decode(strings.NewReader(string(data)))
	if !a.Success() || !b.Success() {
		t.Fatalf("expected both to succeed")
	}
	if a.Document.Version() != b.Document.Version() {
		t.Fatalf("dispatch mismatch")
	}
}

func TestDiagnosticDeterminismInvalidOrder(t *testing.T) {
	data := readFixture(t, "testdata/invalid/v2-bad-status-enum.json")
	a := Decode(strings.NewReader(string(data)))
	b := Decode(strings.NewReader(string(data)))
	aj, _ := json.Marshal(a.Diagnostics)
	bj, _ := json.Marshal(b.Diagnostics)
	if string(aj) != string(bj) {
		t.Fatalf("diagnostics are not byte-identical:\nA=%s\nB=%s", aj, bj)
	}
}

func TestDiagnosticDeterminismDuplicateKeys(t *testing.T) {
	data := readFixture(t, "testdata/duplicate-keys/v2-duplicate-top-level-field.json")
	a := Decode(strings.NewReader(string(data)))
	b := Decode(strings.NewReader(string(data)))
	aj, _ := json.Marshal(a.Diagnostics)
	bj, _ := json.Marshal(b.Diagnostics)
	if string(aj) != string(bj) {
		t.Fatalf("duplicate-key diagnostics are not byte-identical:\nA=%s\nB=%s", aj, bj)
	}
}

func TestDiagnosticSetOrdering(t *testing.T) {
	ds := &diagnosticSet{}
	ds.add(newDiagnostic(CodeSchemaViolation, "/b", "x"))
	ds.add(newDiagnostic(CodeDocumentTooLarge, "", "x"))
	ds.add(newDiagnostic(CodeSchemaViolation, "/a", "x"))
	out := ds.emit()
	if out[0].Code != CodeDocumentTooLarge {
		t.Errorf("first diagnostic must be highest precedence, got %s", out[0].Code)
	}
	if out[1].Code != CodeSchemaViolation || out[1].Path != "/a" {
		t.Errorf("second diagnostic must be /a, got %+v", out[1])
	}
}

func TestDiagnosticSetDedup(t *testing.T) {
	ds := &diagnosticSet{}
	ds.add(newDiagnostic(CodeDuplicateKey, "/a", "x"))
	ds.add(newDiagnostic(CodeDuplicateKey, "/a", "x"))
	ds.add(newDiagnostic(CodeDuplicateKey, "/b", "x"))
	out := ds.emit()
	if len(out) != 2 {
		t.Errorf("expected dedup to 2 diagnostics, got %d", len(out))
	}
}

func TestDiagnosticSetEncounterIndexStable(t *testing.T) {
	ds := &diagnosticSet{}
	ds.add(newDiagnostic(CodeSchemaViolation, "/b", "x"))
	ds.add(newDiagnostic(CodeSchemaViolation, "/b", "x"))
	ds.add(newDiagnostic(CodeSchemaViolation, "/a", "x"))
	out := ds.emit()
	if out[0].Path != "/a" {
		t.Errorf("path /a must precede /b when precedence equal, got %+v", out)
	}
}
