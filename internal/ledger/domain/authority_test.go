package domain

import "testing"

// This file qualifies the FWL01 amendment single-acceptance-authority
// witnesses.
//
// The authority is ONE predicate: EvaluateAcceptance. The explicit
// acceptance operation and the canonical validator consult it. These
// witnesses prove that the two never disagree.

// acceptanceCase is one row of the acceptance authority matrix.
type acceptanceCase struct {
	// Name identifies the row.
	Name string
	// Ledger is the canonical state of the row.
	Ledger Ledger
	// Admissible is the REQUIRED verdict of the authority.
	Admissible bool
}

// acceptanceMatrix is the orthogonal acceptance matrix.
//
// Every row is built from a valid accepted-item shape, so the only
// discriminator is the rule under test.
func acceptanceMatrix() []acceptanceCase {
	base := func(ev EvidenceRefRecord, criteria []string, live bool) Ledger {
		return Ledger{
			SchemaVersion: LedgerSchemaVersion,
			Evidence:      []EvidenceRefRecord{ev},
			Items: []WorkItem{{
				SchemaVersion:      LedgerSchemaVersion,
				EpicID:             "EPIC-A",
				ItemID:             "ITEM-1",
				Title:              "item",
				Kind:               ItemKindTask,
				Lifecycle:          LifecycleClosed,
				Acceptance:         AcceptanceAccepted,
				Quality:            EvidenceQualityVerified,
				AcceptanceCriteria: criteria,
				RequiresLiveGate:   live,
				Evidence: []EvidenceRef{{
					EvidenceID: ev.EvidenceID,
					Quality:    EvidenceQualityVerified,
					Class:      ev.Class,
					Criterion:  ev.Criterion,
				}},
				Revision: 1,
			}},
		}
	}
	verified := EvidenceRefRecord{
		SchemaVersion: LedgerSchemaVersion,
		EvidenceID:    "EV-A",
		Kind:          "structural",
		Title:         "evidence",
		ItemID:        "EPIC-A/ITEM-1",
		Verified:      true,
		Quality:       EvidenceQualityVerified,
		Class:         EvidenceClassStructural,
		Criterion:     "criterion A",
		Revision:      1,
	}
	live := verified
	live.Class = EvidenceClassLiveQualification

	unverified := verified
	unverified.Verified = false

	unbound := verified
	unbound.ItemID = "EPIC-A/ITEM-OTHER"

	noCriterion := verified
	noCriterion.Criterion = ""

	otherCriterion := verified
	otherCriterion.Criterion = "criterion Z"

	return []acceptanceCase{
		{
			Name:       "one criterion covered by verified bound evidence",
			Ledger:     base(verified, []string{"criterion A"}, false),
			Admissible: true,
		},
		{
			Name:       "two criteria with only one covered",
			Ledger:     base(verified, []string{"criterion A", "criterion B"}, false),
			Admissible: false,
		},
		{
			Name:       "evidence not verified",
			Ledger:     base(unverified, []string{"criterion A"}, false),
			Admissible: false,
		},
		{
			Name:       "evidence bound to another item",
			Ledger:     withOtherItem(base(unbound, []string{"criterion A"}, false)),
			Admissible: false,
		},
		{
			Name:       "evidence without a criterion association",
			Ledger:     base(noCriterion, []string{"criterion A"}, false),
			Admissible: false,
		},
		{
			Name:       "evidence naming an unrelated criterion",
			Ledger:     base(otherCriterion, []string{"criterion A"}, false),
			Admissible: false,
		},
		{
			Name:       "live gate with a structural evidence class",
			Ledger:     base(verified, []string{"criterion A"}, true),
			Admissible: false,
		},
		{
			Name:       "live gate with a live qualification evidence class",
			Ledger:     base(live, []string{"criterion A"}, true),
			Admissible: true,
		},
	}
}

// withOtherItem adds a second work item to the ledger.
//
// The "bound to another item" row needs the other item to exist: a
// dangling reference would be an unrelated defect.
func withOtherItem(l Ledger) Ledger {
	l.Items = append(l.Items, WorkItem{
		SchemaVersion: LedgerSchemaVersion,
		EpicID:        "EPIC-A",
		ItemID:        "ITEM-OTHER",
		Title:         "other item",
		Kind:          ItemKindTask,
		Lifecycle:     LifecycleOpen,
		Acceptance:    AcceptancePending,
		Revision:      1,
	})
	return l
}

// TestWitnessSingleAcceptanceAuthorityMatrix proves the acceptance
// authority verdict of every matrix row.
func TestWitnessSingleAcceptanceAuthorityMatrix(t *testing.T) {
	for _, tc := range acceptanceMatrix() {
		t.Run(tc.Name, func(t *testing.T) {
			item := tc.Ledger.Items[0]
			got := EvaluateAcceptance(tc.Ledger, item)
			if got.Admissible != tc.Admissible {
				t.Fatalf("admissible = %t, want %t (reason: %s)",
					got.Admissible, tc.Admissible, got.Reason)
			}
			if !got.Admissible && got.Reason == "" {
				t.Fatal("the rejection carries no reason")
			}
		})
	}
}

// TestWitnessCanonicalValidatorUsesTheSameAuthority proves that the
// canonical validator and the explicit acceptance operation agree.
//
// The validator reports the invalidation of an accepted item whose
// evidence is not admissible, using the SAME authority the acceptance
// operation consults. There is no second, competing acceptance rule.
func TestWitnessCanonicalValidatorUsesTheSameAuthority(t *testing.T) {
	for _, tc := range acceptanceMatrix() {
		t.Run(tc.Name, func(t *testing.T) {
			l := tc.Ledger
			res := ValidateLedger(l)
			// The authority verdict and the validator verdict agree.
			authority := EvaluateAcceptance(l, l.Items[0])
			if authority.Admissible != tc.Admissible {
				t.Fatalf("the authority verdict drifted: %t", authority.Admissible)
			}
			if tc.Admissible {
				if !res.OK() {
					t.Fatalf("the admissible row was rejected: %v", res.Errors())
				}
				return
			}
			// The inadmissible row is REPORTED, never silent.
			reported := false
			for _, a := range res.Advisories {
				if a.Code == CodeAcceptanceAuthorityInvalid {
					reported = true
				}
			}
			if !reported {
				t.Fatalf("the inadmissible accepted item was not reported: %+v", res.Advisories)
			}
			// The historical acceptance is PRESERVED: the item keeps its
			// accepted state and the state is not rejected.
			if !res.OK() {
				t.Fatalf("the preserved historical acceptance was rejected: %v", res.Errors())
			}
			if l.Items[0].Acceptance != AcceptanceAccepted {
				t.Fatal("the historical acceptance was not preserved")
			}
			// The invalidated authority does not silently satisfy a
			// downstream prerequisite.
			if CurrentlyAuthorized(l, l.Items[0]) {
				t.Fatal("the invalidated authority still authorizes a downstream prerequisite")
			}
		})
	}
}

// TestWitnessAcceptedItemWithoutEvidenceIsReported proves that an accepted
// item with NO evidence at all is reported.
func TestWitnessAcceptedItemWithoutEvidenceIsReported(t *testing.T) {
	l := Ledger{
		SchemaVersion: LedgerSchemaVersion,
		Items: []WorkItem{{
			SchemaVersion:      LedgerSchemaVersion,
			EpicID:             "EPIC-A",
			ItemID:             "ITEM-1",
			Title:              "item",
			Kind:               ItemKindTask,
			Lifecycle:          LifecycleClosed,
			Acceptance:         AcceptanceAccepted,
			Quality:            EvidenceQualityVerified,
			AcceptanceCriteria: []string{"criterion"},
			Revision:           1,
		}},
	}
	res := ValidateLedger(l)
	if !res.OK() {
		t.Fatalf("the preserved historical acceptance was rejected: %v", res.Errors())
	}
	reported := false
	for _, a := range res.Advisories {
		if a.Code == CodeAcceptanceAuthorityInvalid {
			reported = true
		}
	}
	if !reported {
		t.Fatalf("the accepted item without evidence was not reported: %+v", res.Advisories)
	}
	if CurrentlyAuthorized(l, l.Items[0]) {
		t.Fatal("the accepted item without evidence still authorizes a downstream prerequisite")
	}
}
