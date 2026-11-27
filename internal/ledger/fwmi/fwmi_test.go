package fwmi

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writePackage writes an interchange package from the given file contents.
func writePackage(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	for name, body := range files {
		path := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("create %s: %v", name, err)
		}
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}
	return dir
}

// validManifest is a valid source manifest document.
const validManifest = `{
  "schema_version": 1,
  "repository": "clinemm",
  "source_head": "29774ba05851f9dd5a46a33a90d963ff3ef286d5",
  "files": [
    {"path": ".factory/epic-board.md", "bytes": 3571144, "lines": 44730,
     "sha256": "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
     "git_blob": "0123456789abcdef0123456789abcdef01234567", "disposition": "indexed"}
  ]
}`

// validEpic is a valid epic document with one obligation.
const validEpic = `{
  "schema_version": 1,
  "epic_id": "EPIC-ARCHITECTURE",
  "title": "Architecture",
  "original_status": null,
  "lifecycle": "OPEN",
  "priority": null,
  "exit_criteria": ["the architecture is documented"],
  "items": [
    {
      "schema_version": 1,
      "item_id": "E8",
      "title": "Recover the meaning of E8",
      "description": "The E8 obligation must be recovered from the source board.",
      "workstream": null,
      "original_status": "OPEN",
      "lifecycle": "OPEN",
      "priority": null,
      "qualification": "UNKNOWN",
      "criteria": [],
      "criteria_coverage": "UNKNOWN",
      "dependencies": [],
      "dependency_coverage": "UNKNOWN",
      "blockers": [],
      "current_act": null,
      "related_acts": ["ACT-EXAMPLE-01"],
      "evidence_locators": [],
      "reopen_conditions": [],
      "source_refs": [
        {"path": ".factory/epics/architecture.md", "start_line": 120, "end_line": 137,
         "sha256": "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"}
      ]
    }
  ],
  "source_refs": []
}`

// validHistorical is a valid historical index document.
const validHistorical = `{
  "schema_version": 1,
  "records": [
    {
      "schema_version": 1,
      "act_id": "ACT-EXAMPLE-01",
      "original_verdict": "PASS_STRUCTURAL_ONLY",
      "epic_id": "EPIC-ARCHITECTURE",
      "work_item_ids": ["EPIC-ARCHITECTURE/E8"],
      "evidence_locators": [".factory/evidence/act-example-01.md"],
      "source_refs": [],
      "reconciliation": "UNAMBIGUOUS"
    }
  ]
}`

// validReconciliation is a valid reconciliation manifest.
const validReconciliation = `{
  "schema_version": 1,
  "records": [
    {
      "schema_version": 1,
      "kind": "POSSIBLE_CROSS_LINK",
      "identities": ["EPIC-A/X", "EPIC-B/X"],
      "resolution": "UNRESOLVED",
      "reason": "Determine whether these are one obligation or two.",
      "source_refs": []
    }
  ]
}`

// validPackage returns the file set of a valid interchange package.
func validPackage() map[string]string {
	return map[string]string{
		ManifestFile:                   validManifest,
		"epics/EPIC-ARCHITECTURE.json": validEpic,
		HistoricalFile:                 validHistorical,
		ReconciliationFileName:         validReconciliation,
	}
}

// TestLoadValidPackage proves the valid package loads and validates.
func TestLoadValidPackage(t *testing.T) {
	pkg, err := Load(writePackage(t, validPackage()))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(pkg.Epics) != 1 {
		t.Fatalf("epics = %d, want 1", len(pkg.Epics))
	}
	if pkg.Manifest.SourceHead != "29774ba05851f9dd5a46a33a90d963ff3ef286d5" {
		t.Errorf("source head = %q", pkg.Manifest.SourceHead)
	}
}

// TestLoadRejectsUnknownField proves the strict decode rejects an unknown
// field in an interchange record.
func TestLoadRejectsUnknownField(t *testing.T) {
	files := validPackage()
	files["epics/EPIC-ARCHITECTURE.json"] = strings.Replace(validEpic,
		`"title": "Architecture"`, `"title": "Architecture", "unknown_field": 1`, 1)
	_, err := Load(writePackage(t, files))
	if err == nil {
		t.Fatal("expected the unknown field to be rejected")
	}
	if !strings.Contains(err.Error(), "unknown_field") {
		t.Errorf("error = %v, want it to name the unknown field", err)
	}
}

// TestLoadRejectsSecondDocument proves the strict decode rejects a second
// JSON document in the same file.
func TestLoadRejectsSecondDocument(t *testing.T) {
	files := validPackage()
	files[HistoricalFile] = validHistorical + "\n" + validHistorical
	_, err := Load(writePackage(t, files))
	if err == nil {
		t.Fatal("expected the second document to be rejected")
	}
}

// TestLoadRejectsDuplicateEpic proves the duplicate epic identity is
// rejected.
func TestLoadRejectsDuplicateEpic(t *testing.T) {
	files := validPackage()
	files["epics/EPIC-ARCHITECTURE-COPY.json"] = validEpic
	_, err := Load(writePackage(t, files))
	if err == nil {
		t.Fatal("expected the duplicate epic identity to be rejected")
	}
	if !strings.Contains(err.Error(), "duplicate epic identity") {
		t.Errorf("error = %v, want it to name the duplicate identity", err)
	}
}

// TestLoadRejectsInvalidLifecycle proves an unknown vocabulary value is
// rejected.
func TestLoadRejectsInvalidLifecycle(t *testing.T) {
	files := validPackage()
	files["epics/EPIC-ARCHITECTURE.json"] = strings.Replace(validEpic,
		`"lifecycle": "OPEN"`, `"lifecycle": "MAYBE"`, 1)
	_, err := Load(writePackage(t, files))
	if err == nil {
		t.Fatal("expected the invalid lifecycle to be rejected")
	}
}

// TestLoadRejectsInvalidSourceRange proves a source reference without a
// valid byte range is rejected.
func TestLoadRejectsInvalidSourceRange(t *testing.T) {
	files := validPackage()
	files["epics/EPIC-ARCHITECTURE.json"] = strings.Replace(validEpic,
		`"start_line": 120, "end_line": 137`, `"start_line": 137, "end_line": 120`, 1)
	_, err := Load(writePackage(t, files))
	if err == nil {
		t.Fatal("expected the invalid source range to be rejected")
	}
}

// TestLoadAcceptsCompleteCoverageOverEmptyCollection proves the coverage
// semantics of the interchange contract.
//
// A COMPLETE coverage over an EMPTY collection means "the collection is known
// to be empty": there is nothing to recover and nothing was lost. It is a
// fact about the obligation, not a contradiction, and it is accepted.
func TestLoadAcceptsCompleteCoverageOverEmptyCollection(t *testing.T) {
	files := validPackage()
	files["epics/EPIC-ARCHITECTURE.json"] = strings.Replace(validEpic,
		`"criteria_coverage": "UNKNOWN"`, `"criteria_coverage": "COMPLETE"`, 1)
	files["epics/EPIC-ARCHITECTURE.json"] = strings.Replace(files["epics/EPIC-ARCHITECTURE.json"],
		`"dependency_coverage": "UNKNOWN"`, `"dependency_coverage": "COMPLETE"`, 1)
	pkg, err := Load(writePackage(t, files))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	item := pkg.Epics[0].Items[0]
	if item.CriteriaCoverage != CoverageComplete {
		t.Errorf("criteria coverage = %q, want COMPLETE", item.CriteriaCoverage)
	}
	if item.DependencyCoverage != CoverageComplete {
		t.Errorf("dependency coverage = %q, want COMPLETE", item.DependencyCoverage)
	}
}

// TestLoadAcceptsUnknownCoverageOverEmptyCollection proves the DISTINCTION
// the coverage vocabulary encodes.
//
// An UNKNOWN coverage means the collection was NOT recovered. It is never an
// assertion that the collection is empty, and it is accepted as such: the
// importer reports the missing recovery, it never invents the missing
// criteria or dependencies.
func TestLoadAcceptsUnknownCoverageOverEmptyCollection(t *testing.T) {
	files := validPackage()
	pkg, err := Load(writePackage(t, files))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	item := pkg.Epics[0].Items[0]
	if item.CriteriaCoverage != CoverageUnknown {
		t.Errorf("criteria coverage = %q, want UNKNOWN", item.CriteriaCoverage)
	}
	if item.DependencyCoverage != CoverageUnknown {
		t.Errorf("dependency coverage = %q, want UNKNOWN", item.DependencyCoverage)
	}
}

// TestLoadRejectsUnknownCoverageVocabulary proves the NEGATIVE control of
// the coverage rules: a coverage value outside the vocabulary is rejected.
func TestLoadRejectsUnknownCoverageVocabulary(t *testing.T) {
	files := validPackage()
	files["epics/EPIC-ARCHITECTURE.json"] = strings.Replace(validEpic,
		`"criteria_coverage": "UNKNOWN"`, `"criteria_coverage": "MOSTLY"`, 1)
	_, err := Load(writePackage(t, files))
	if err == nil {
		t.Fatal("expected the unknown coverage value to be rejected")
	}
	if !strings.Contains(err.Error(), "criteria coverage") {
		t.Errorf("error = %v, want it to name the coverage field", err)
	}
}

// TestLoadRejectsDuplicateTopLevelMember proves the strict decode rejects an
// AMBIGUOUS document: an object that names the same member twice.
func TestLoadRejectsDuplicateTopLevelMember(t *testing.T) {
	files := validPackage()
	files[HistoricalFile] = strings.Replace(validHistorical,
		`"reconciliation": "UNAMBIGUOUS"`,
		`"reconciliation": "UNAMBIGUOUS", "reconciliation": "AMBIGUOUS"`, 1)
	_, err := Load(writePackage(t, files))
	if err == nil {
		t.Fatal("expected the duplicate member to be rejected")
	}
	if !strings.Contains(err.Error(), "duplicate object member") {
		t.Errorf("error = %v, want it to name the duplicate member", err)
	}
}

// TestLoadRejectsDuplicateNestedMember proves the duplicate-member rejection
// reaches the NESTED objects as well.
func TestLoadRejectsDuplicateNestedMember(t *testing.T) {
	files := validPackage()
	files["epics/EPIC-ARCHITECTURE.json"] = strings.Replace(validEpic,
		`"title": "Recover the meaning of E8"`,
		`"title": "Recover the meaning of E8", "title": "another title"`, 1)
	_, err := Load(writePackage(t, files))
	if err == nil {
		t.Fatal("expected the duplicate nested member to be rejected")
	}
	if !strings.Contains(err.Error(), "duplicate object member") {
		t.Errorf("error = %v, want it to name the duplicate member", err)
	}
}

// TestLoadAcceptsTheSameMemberNameInDifferentObjects proves the duplicate
// rejection is NARROW: the same member name in different objects is
// legitimate.
func TestLoadAcceptsTheSameMemberNameInDifferentObjects(t *testing.T) {
	files := validPackage()
	// Two records that both carry a "title" member are legitimate.
	files["epics/EPIC-OTHER.json"] = strings.Replace(validEpic,
		`"epic_id": "EPIC-ARCHITECTURE"`, `"epic_id": "EPIC-OTHER"`, 1)
	if _, err := Load(writePackage(t, files)); err != nil {
		t.Fatalf("Load: %v", err)
	}
}

// TestLoadRejectsNestedSchemaVersion proves the NESTED schema versions are
// validated: a record that declares an unsupported version is rejected.
func TestLoadRejectsNestedSchemaVersion(t *testing.T) {
	files := validPackage()
	files["epics/EPIC-ARCHITECTURE.json"] = strings.Replace(validEpic,
		`"schema_version": 1,
      "item_id": "E8"`, `"schema_version": 2,
      "item_id": "E8"`, 1)
	_, err := Load(writePackage(t, files))
	if err == nil {
		t.Fatal("expected the nested schema version to be rejected")
	}
	if !strings.Contains(err.Error(), "schema version") {
		t.Errorf("error = %v, want it to name the schema version", err)
	}
}

// TestLoadRejectsMalformedDigest proves the source-reference digest is
// validated as an actual hexadecimal SHA-256 and not merely by its length.
func TestLoadRejectsMalformedDigest(t *testing.T) {
	files := validPackage()
	files["epics/EPIC-ARCHITECTURE.json"] = strings.Replace(validEpic,
		"bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
		"zzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzz", 1)
	_, err := Load(writePackage(t, files))
	if err == nil {
		t.Fatal("expected the malformed digest to be rejected")
	}
	if !strings.Contains(err.Error(), "SHA-256") {
		t.Errorf("error = %v, want it to name the digest", err)
	}
}
