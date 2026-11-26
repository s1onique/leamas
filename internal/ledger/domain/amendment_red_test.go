package domain

import "testing"

// This file records the RED reproduction of the readiness counterexamples
// against the real domain projection.

// TestAmendmentFreshWorkCanAcquireOwnEvidence reproduces counterexample 6:
// fresh work with acceptance criteria being incorrectly excluded from
// readiness.
//
// The contract: a new task with unmet acceptance criteria is eligible to
// acquire its own evidence when its prerequisites are satisfied. Execution
// readiness is not the same thing as acceptance readiness.
func TestAmendmentFreshWorkCanAcquireOwnEvidence(t *testing.T) {
	l := Ledger{
		SchemaVersion: LedgerSchemaVersion,
		Items: []WorkItem{{
			SchemaVersion:      LedgerSchemaVersion,
			EpicID:             EpicID("EPIC-A"),
			ItemID:             ItemID("ITEM-1"),
			Title:              "Fresh work",
			Kind:               ItemKindTask,
			Lifecycle:          LifecycleOpen,
			Acceptance:         AcceptancePending,
			Quality:            EvidenceQualityNone,
			AcceptanceCriteria: []string{"criterion not yet met"},
			Revision:           1,
		}},
	}

	ready := SelectReady(l, ReadyOptions{})
	if len(ready) != 1 {
		t.Fatalf("fresh work with unmet acceptance criteria was excluded from readiness: got %d items", len(ready))
	}
}

// TestAmendmentDeferredAndBlockedWorkIsNotSelected reproduces the W4
// requirement that deferred and explicitly blocked work must not be
// selected.
func TestAmendmentDeferredAndBlockedWorkIsNotSelected(t *testing.T) {
	l := Ledger{
		SchemaVersion: LedgerSchemaVersion,
		Items: []WorkItem{
			{
				SchemaVersion: LedgerSchemaVersion,
				EpicID:        EpicID("EPIC-A"),
				ItemID:        ItemID("ITEM-1"),
				Title:         "Deferred work",
				Kind:          ItemKindTask,
				Lifecycle:     LifecycleOpen,
				Acceptance:    AcceptanceDeferred,
				Revision:      1,
			},
			{
				SchemaVersion: LedgerSchemaVersion,
				EpicID:        EpicID("EPIC-A"),
				ItemID:        ItemID("ITEM-2"),
				Title:         "Blocked work",
				Kind:          ItemKindTask,
				Lifecycle:     LifecycleBlocked,
				Acceptance:    AcceptancePending,
				Revision:      1,
			},
		},
	}

	ready := SelectReady(l, ReadyOptions{})
	if len(ready) != 0 {
		t.Fatalf("deferred or blocked work was selected: got %d items", len(ready))
	}
}

// TestAmendmentLiveQualificationIsExecutableBeforeEvidence reproduces the
// W4 requirement that a live qualification obligation must be executable
// before live qualification evidence exists.
func TestAmendmentLiveQualificationIsExecutableBeforeEvidence(t *testing.T) {
	l := Ledger{
		SchemaVersion: LedgerSchemaVersion,
		Items: []WorkItem{{
			SchemaVersion:      LedgerSchemaVersion,
			EpicID:             EpicID("EPIC-A"),
			ItemID:             ItemID("ITEM-1"),
			Title:              "Needs live qualification",
			Kind:               ItemKindTask,
			Lifecycle:          LifecycleOpen,
			Acceptance:         AcceptancePending,
			AcceptanceCriteria: []string{"live qualification must pass"},
			RequiresLiveGate:   true,
			Revision:           1,
		}},
	}

	ready := SelectReady(l, ReadyOptions{})
	if len(ready) != 1 {
		t.Fatalf("live qualification obligation was not executable before evidence existed: got %d items", len(ready))
	}
}
