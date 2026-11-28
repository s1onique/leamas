package importer

import (
	"github.com/s1onique/leamas/internal/ledger/domain"
)

// This file holds the conservation comparison of the CANONICAL RECORD
// FAMILIES that the import does not modify.
//
// The evidence records, the decision records and the knowledge references are
// NOT modified by an import: the import creates epic scopes, work items,
// historical ACT executions and durable migration records. The expected
// post-import state therefore preserves those families EXACTLY as they were
// before the import.
//
// The comparison reuses the COMPLETE canonical-record comparison of
// receipt_conservation.go: no semantic field of a record is excluded, and an
// unauthorized addition, deletion or change is reported.

// comparePersistedEvidence compares the expected evidence records with the
// persisted ones.
func comparePersistedEvidence(before domain.Ledger, persisted domain.Ledger) []string {
	expected := map[domain.EvidenceID]domain.EvidenceRefRecord{}
	for _, rec := range before.Evidence {
		expected[rec.EvidenceID] = rec
	}
	persistedByID := map[domain.EvidenceID]domain.EvidenceRefRecord{}
	for _, rec := range persisted.Evidence {
		persistedByID[rec.EvidenceID] = rec
	}
	var out []string
	for id, want := range expected {
		stored, ok := persistedByID[id]
		if !ok {
			out = append(out, "the persisted state lost the evidence "+string(id))
			continue
		}
		out = append(out, compareCanonicalRecords("evidence", string(id), want, stored)...)
	}
	for _, rec := range persisted.Evidence {
		if _, ok := expected[rec.EvidenceID]; !ok {
			out = append(out, "the persisted state contains an unexpected evidence "+string(rec.EvidenceID))
		}
	}
	return out
}

// comparePersistedDecisions compares the expected decision records with the
// persisted ones.
func comparePersistedDecisions(before domain.Ledger, persisted domain.Ledger) []string {
	expected := map[domain.DecisionID]domain.Decision{}
	for _, rec := range before.Decisions {
		expected[rec.DecisionID] = rec
	}
	persistedByID := map[domain.DecisionID]domain.Decision{}
	for _, rec := range persisted.Decisions {
		persistedByID[rec.DecisionID] = rec
	}
	var out []string
	for id, want := range expected {
		stored, ok := persistedByID[id]
		if !ok {
			out = append(out, "the persisted state lost the decision "+string(id))
			continue
		}
		out = append(out, compareCanonicalRecords("decision", string(id), want, stored)...)
	}
	for _, rec := range persisted.Decisions {
		if _, ok := expected[rec.DecisionID]; !ok {
			out = append(out, "the persisted state contains an unexpected decision "+string(rec.DecisionID))
		}
	}
	return out
}

// comparePersistedKnowledge compares the expected knowledge references with
// the persisted ones.
func comparePersistedKnowledge(before domain.Ledger, persisted domain.Ledger) []string {
	expected := map[domain.KnowledgeID]domain.Knowledge{}
	for _, rec := range before.Knowledge {
		expected[rec.KnowledgeID] = rec
	}
	persistedByID := map[domain.KnowledgeID]domain.Knowledge{}
	for _, rec := range persisted.Knowledge {
		persistedByID[rec.KnowledgeID] = rec
	}
	var out []string
	for id, want := range expected {
		stored, ok := persistedByID[id]
		if !ok {
			out = append(out, "the persisted state lost the knowledge "+string(id))
			continue
		}
		out = append(out, compareCanonicalRecords("knowledge", string(id), want, stored)...)
	}
	for _, rec := range persisted.Knowledge {
		if _, ok := expected[rec.KnowledgeID]; !ok {
			out = append(out, "the persisted state contains an unexpected knowledge "+string(rec.KnowledgeID))
		}
	}
	return out
}
