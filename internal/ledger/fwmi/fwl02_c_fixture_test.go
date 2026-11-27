package fwmi

// This file holds the bounded SYNTHETIC multi-epic fixture of the FWL02-C
// qualification.
//
// The fixture exercises the complete interchange contract: an empty
// historical epic, multiple reconciliation records with the same kind and
// resolution, explicit blockers, a known-empty dependency coverage, a
// required dependency outcome, historical ACT associations, evidence locators
// and reopen conditions.
//
// The fixture is SYNTHETIC. It proves the behaviour of the adapter against
// the interchange contract; it does NOT prove a migration of any actual
// source.

// multiEpicManifest is the source manifest of the multi-epic fixture.
const multiEpicManifest = `{
  "schema_version": 1,
  "repository": "clinemm",
  "source_head": "29774ba05851f9dd5a46a33a90d963ff3ef286d5",
  "files": []
}`

// multiEpicArchitecture is an epic with one obligation, an explicit blocker,
// a required dependency outcome and the historical associations.
const multiEpicArchitecture = `{
  "schema_version": 1,
  "epic_id": "EPIC-ARCHITECTURE",
  "title": "Architecture",
  "original_status": "OPEN",
  "lifecycle": "OPEN",
  "priority": "P1",
  "exit_criteria": ["the architecture is documented"],
  "items": [
    {
      "schema_version": 1,
      "item_id": "E8",
      "title": "Recover the meaning of E8",
      "description": "The E8 obligation must be recovered from the source board.",
      "workstream": "EPIC-ARCHITECTURE/ARCH",
      "original_status": "OPEN",
      "lifecycle": "OPEN",
      "priority": "P1",
      "qualification": "UNKNOWN",
      "criteria": ["the meaning of E8 is documented"],
      "criteria_coverage": "COMPLETE",
      "dependencies": [
        {"kind": "DEPENDS_ON", "target": "EPIC-ARCHITECTURE/E9",
         "required_outcome": "E9 is accepted", "source_refs": []}
      ],
      "dependency_coverage": "COMPLETE",
      "blockers": ["the source states that the upstream decision is pending"],
      "current_act": "ACT-CURRENT-01",
      "related_acts": ["ACT-EXAMPLE-01"],
      "evidence_locators": [".factory/evidence/act-example-01.md"],
      "reopen_conditions": ["the upstream decision is taken"],
      "source_refs": [
        {"path": ".factory/epics/architecture.md", "start_line": 120, "end_line": 137,
         "sha256": "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"}
      ]
    }
  ],
  "source_refs": []
}`

// multiEpicCompleted is a COMPLETED epic with no outstanding work item.
const multiEpicCompleted = `{
  "schema_version": 1,
  "epic_id": "EPIC-COMPLETED",
  "title": "The completed scope",
  "original_status": "TERMINAL",
  "lifecycle": "TERMINAL",
  "priority": "P2",
  "exit_criteria": ["the scope is finished"],
  "items": [],
  "source_refs": []
}`

// multiEpicHistorical is the historical index of the multi-epic fixture.
const multiEpicHistorical = `{
  "schema_version": 1,
  "records": [
    {
      "schema_version": 1,
      "act_id": "ACT-EXAMPLE-01",
      "original_verdict": "PASS_STRUCTURAL_ONLY",
      "epic_id": "EPIC-ARCHITECTURE",
      "work_item_ids": ["EPIC-ARCHITECTURE/E8", "EPIC-ARCHITECTURE/E-MISSING"],
      "evidence_locators": [".factory/evidence/act-example-01.md"],
      "source_refs": [],
      "reconciliation": "UNAMBIGUOUS"
    }
  ]
}`

// multiEpicReconciliation carries TWO reconciliation records with the SAME
// kind and the SAME resolution. They concern DIFFERENT identities, so they
// are distinct records and they must not overwrite one another.
const multiEpicReconciliation = `{
  "schema_version": 1,
  "records": [
    {
      "schema_version": 1,
      "kind": "POSSIBLE_CROSS_LINK",
      "identities": ["EPIC-A/X", "EPIC-B/X"],
      "resolution": "UNRESOLVED",
      "reason": "Determine whether these are one obligation or two.",
      "source_refs": []
    },
    {
      "schema_version": 1,
      "kind": "POSSIBLE_CROSS_LINK",
      "identities": ["EPIC-C/Y", "EPIC-D/Y"],
      "resolution": "UNRESOLVED",
      "reason": "A second, independent cross-link ambiguity.",
      "source_refs": []
    }
  ]
}`

// multiEpicPackage returns the file set of the bounded multi-epic fixture.
func multiEpicPackage() map[string]string {
	return map[string]string{
		ManifestFile:                   multiEpicManifest,
		"epics/EPIC-ARCHITECTURE.json": multiEpicArchitecture,
		"epics/EPIC-COMPLETED.json":    multiEpicCompleted,
		HistoricalFile:                 multiEpicHistorical,
		ReconciliationFileName:         multiEpicReconciliation,
	}
}
