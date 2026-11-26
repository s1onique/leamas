package domain

import (
	"strings"
	"testing"
)

// baseLedger returns a minimal valid ledger used as the mutation target.
func baseLedger() Ledger {
	return Ledger{
		SchemaVersion: LedgerSchemaVersion,
		Epics: []Epic{{
			SchemaVersion: LedgerSchemaVersion,
			EpicID:        "EPIC-A",
			Title:         "Epic A",
			Priority:      PriorityP1,
			Revision:      1,
		}},
		Items: []WorkItem{{
			SchemaVersion: LedgerSchemaVersion,
			EpicID:        "EPIC-A",
			ItemID:        "ITEM-1",
			Title:         "First item",
			Kind:          ItemKindTask,
			Lifecycle:     LifecycleOpen,
			Acceptance:    AcceptancePending,
			Priority:      PriorityP1,
			Quality:       EvidenceQualityNone,
			Revision:      1,
		}},
	}
}

func hasCode(res ValidationResult, code string) bool {
	for _, f := range res.Findings {
		if f.Code == code {
			return true
		}
	}
	return false
}

func codes(res ValidationResult) []string {
	out := make([]string, 0, len(res.Findings))
	for _, f := range res.Findings {
		out = append(out, f.Code)
	}
	return out
}

func TestValidateLedgerAcceptsBaseState(t *testing.T) {
	res := ValidateLedger(baseLedger())
	if !res.OK() {
		t.Fatalf("expected base ledger to validate, got %v", codes(res))
	}
}

// TestValidateLedgerDuplicateIdentityRejected witnesses adversarial row 1:
// a duplicate work-item identity must be rejected.
func TestValidateLedgerDuplicateIdentityRejected(t *testing.T) {
	l := baseLedger()
	dup := l.Items[0]
	l.Items = append(l.Items, dup)

	res := ValidateLedger(l)
	if !hasCode(res, CodeDuplicateIdentity) {
		t.Fatalf("expected duplicate_identity, got %v", codes(res))
	}
}

// TestValidateLedgerDuplicateEpicIdentityRejected witnesses that a
// duplicate epic identity is rejected too.
func TestValidateLedgerDuplicateEpicIdentityRejected(t *testing.T) {
	l := baseLedger()
	l.Epics = append(l.Epics, l.Epics[0])

	res := ValidateLedger(l)
	if !hasCode(res, CodeDuplicateIdentity) {
		t.Fatalf("expected duplicate_identity, got %v", codes(res))
	}
}

// TestValidateLedgerDanglingDependencyRejected witnesses adversarial row 2:
// a dependency reference to an unknown item is rejected.
func TestValidateLedgerDanglingDependencyRejected(t *testing.T) {
	l := baseLedger()
	l.Items[0].Dependencies = []Dependency{
		{Kind: DependencyBlocks, Ref: "EPIC-A/ITEM-MISSING"},
	}

	res := ValidateLedger(l)
	if !hasCode(res, CodeUnknownReference) {
		t.Fatalf("expected unknown_reference, got %v", codes(res))
	}
}

// TestValidateLedgerBlockingCycleRejected witnesses adversarial row 3:
// a blocking dependency cycle is rejected.
func TestValidateLedgerBlockingCycleRejected(t *testing.T) {
	l := baseLedger()
	l.Items = append(l.Items, WorkItem{
		SchemaVersion: LedgerSchemaVersion,
		EpicID:        "EPIC-A",
		ItemID:        "ITEM-2",
		Title:         "Second item",
		Kind:          ItemKindTask,
		Lifecycle:     LifecycleOpen,
		Acceptance:    AcceptancePending,
		Revision:      1,
	})
	l.Items[0].Dependencies = []Dependency{
		{Kind: DependencyBlocks, Ref: "EPIC-A/ITEM-2"},
	}
	l.Items[1].Dependencies = []Dependency{
		{Kind: DependencyBlocks, Ref: "EPIC-A/ITEM-1"},
	}

	res := ValidateLedger(l)
	if !hasCode(res, CodeCycle) {
		t.Fatalf("expected dependency_cycle, got %v", codes(res))
	}
}

// TestValidateLedgerRelatesCycleAccepted proves the cycle guard is scoped
// to blocking dependency kinds: a "relates" edge cannot create a cycle.
func TestValidateLedgerRelatesCycleAccepted(t *testing.T) {
	l := baseLedger()
	l.Items = append(l.Items, WorkItem{
		SchemaVersion: LedgerSchemaVersion,
		EpicID:        "EPIC-A",
		ItemID:        "ITEM-2",
		Title:         "Second item",
		Kind:          ItemKindTask,
		Lifecycle:     LifecycleOpen,
		Acceptance:    AcceptancePending,
		Revision:      1,
	})
	l.Items[0].Dependencies = []Dependency{
		{Kind: DependencyRelates, Ref: "EPIC-A/ITEM-2"},
	}
	l.Items[1].Dependencies = []Dependency{
		{Kind: DependencyRelates, Ref: "EPIC-A/ITEM-1"},
	}

	res := ValidateLedger(l)
	if !res.OK() {
		t.Fatalf("expected relates cycle to be accepted, got %v", codes(res))
	}
}

// TestValidateLedgerSelfDependencyRejected witnesses that a self
// dependency is rejected.
func TestValidateLedgerSelfDependencyRejected(t *testing.T) {
	l := baseLedger()
	l.Items[0].Dependencies = []Dependency{
		{Kind: DependencyBlocks, Ref: "EPIC-A/ITEM-1"},
	}

	res := ValidateLedger(l)
	if !hasCode(res, CodeSelfDependency) {
		t.Fatalf("expected self_dependency, got %v", codes(res))
	}
}

// TestValidateLedgerInsufficientEvidenceCannotAccept witnesses adversarial
// row 4: an accepted item with declared acceptance criteria must carry
// referenced or verified evidence.
func TestValidateLedgerInsufficientEvidenceCannotAccept(t *testing.T) {
	l := baseLedger()
	l.Items[0].Acceptance = AcceptanceAccepted
	l.Items[0].AcceptanceCriteria = []string{"the test passes"}
	l.Items[0].Quality = EvidenceQualityDeclared

	res := ValidateLedger(l)
	if !hasCode(res, CodeInsufficientEvidence) {
		t.Fatalf("expected insufficient_evidence, got %v", codes(res))
	}
}

// TestValidateLedgerStructuralPassCannotCloseLiveGate witnesses adversarial
// row 5: a structural pass never satisfies a live qualification gate.
func TestValidateLedgerStructuralPassCannotCloseLiveGate(t *testing.T) {
	l := baseLedger()
	l.Items[0].Acceptance = AcceptanceAccepted
	l.Items[0].RequiresLiveGate = true
	l.Items[0].Quality = EvidenceQualityReferenced

	res := ValidateLedger(l)
	if !hasCode(res, CodeLiveGateUnsatisfied) {
		t.Fatalf("expected live_gate_unsatisfied, got %v", codes(res))
	}
}

// TestValidateLedgerLiveGateSatisfiedByVerifiedEvidence proves the live
// gate rule is not vacuous: verified evidence satisfies it.
func TestValidateLedgerLiveGateSatisfiedByVerifiedEvidence(t *testing.T) {
	l := baseLedger()
	l.Items[0].Acceptance = AcceptanceAccepted
	l.Items[0].RequiresLiveGate = true
	l.Items[0].Quality = EvidenceQualityVerified

	res := ValidateLedger(l)
	if !res.OK() {
		t.Fatalf("expected verified evidence to satisfy live gate, got %v", codes(res))
	}
}

// TestValidateLedgerTerminalActRetainsUnfinishedObligation witnesses
// adversarial row 6: a terminal ACT verdict does not close an unfinished
// engineering obligation.
func TestValidateLedgerTerminalActRetainsUnfinishedObligation(t *testing.T) {
	l := baseLedger()
	l.Items[0].Acceptance = AcceptancePending
	l.Acts = []ActExecution{{
		SchemaVersion:       LedgerSchemaVersion,
		ActID:               "ACT-A",
		ItemID:              "EPIC-A/ITEM-1",
		Verdict:             ActVerdictClosed,
		TerminalDisposition: "closed",
		Revision:            1,
	}}

	res := ValidateLedger(l)
	if !hasCode(res, CodeInsufficientEvidence) {
		t.Fatalf("expected insufficient_evidence, got %v", codes(res))
	}
	if !strings.Contains(res.Error(), "EPIC-A/ITEM-1") {
		t.Fatalf("expected finding to name the retained obligation, got %q", res.Error())
	}
}

// TestValidateLedgerTerminalActWithAcceptedObligationAccepted proves the
// terminal-ACT rule is not vacuous.
func TestValidateLedgerTerminalActWithAcceptedObligationAccepted(t *testing.T) {
	l := baseLedger()
	l.Items[0].Acceptance = AcceptanceAccepted
	l.Items[0].Quality = EvidenceQualityVerified
	l.Acts = []ActExecution{{
		SchemaVersion:       LedgerSchemaVersion,
		ActID:               "ACT-A",
		ItemID:              "EPIC-A/ITEM-1",
		Verdict:             ActVerdictClosed,
		TerminalDisposition: "closed",
		Revision:            1,
	}}

	res := ValidateLedger(l)
	if !res.OK() {
		t.Fatalf("expected accepted obligation to permit terminal act, got %v", codes(res))
	}
}

// TestValidateLedgerTerminalActRequiresPreservedDisposition witnesses that
// a terminal verdict always preserves its original disposition.
func TestValidateLedgerTerminalActRequiresPreservedDisposition(t *testing.T) {
	l := baseLedger()
	l.Items[0].Acceptance = AcceptanceAccepted
	l.Items[0].Quality = EvidenceQualityVerified
	l.Acts = []ActExecution{{
		SchemaVersion: LedgerSchemaVersion,
		ActID:         "ACT-A",
		ItemID:        "EPIC-A/ITEM-1",
		Verdict:       ActVerdictClosed,
		Revision:      1,
	}}

	res := ValidateLedger(l)
	if !hasCode(res, CodeMissingField) {
		t.Fatalf("expected missing_field, got %v", codes(res))
	}
}

// TestValidateLedgerPathTraversalRejected witnesses adversarial row 15:
// an evidence path that escapes the repository is rejected.
func TestValidateLedgerPathTraversalRejected(t *testing.T) {
	l := baseLedger()
	l.Evidence = []EvidenceRefRecord{{
		SchemaVersion: LedgerSchemaVersion,
		EvidenceID:    "EV-A",
		Kind:          "file",
		Title:         "Escape",
		RelativePath:  "../outside/secret.txt",
		Revision:      1,
	}}

	res := ValidateLedger(l)
	if !hasCode(res, CodePathTraversal) {
		t.Fatalf("expected path_traversal, got %v", codes(res))
	}
}

// TestValidateLedgerInvalidDigestRejected witnesses that a malformed
// artifact digest is rejected.
func TestValidateLedgerInvalidDigestRejected(t *testing.T) {
	l := baseLedger()
	l.Evidence = []EvidenceRefRecord{{
		SchemaVersion: LedgerSchemaVersion,
		EvidenceID:    "EV-A",
		Kind:          "file",
		Title:         "Bad digest",
		SHA256:        "not-a-digest",
		Revision:      1,
	}}

	res := ValidateLedger(l)
	if !hasCode(res, CodeInvalidDigest) {
		t.Fatalf("expected invalid_digest, got %v", codes(res))
	}
}

// TestValidateLedgerUnknownEvidenceReferenceRejected witnesses that an
// evidence reference to an unknown record is rejected.
func TestValidateLedgerUnknownEvidenceReferenceRejected(t *testing.T) {
	l := baseLedger()
	l.Items[0].Evidence = []EvidenceRef{
		{EvidenceID: "EV-MISSING", Quality: EvidenceQualityVerified},
	}

	res := ValidateLedger(l)
	if !hasCode(res, CodeUnknownReference) {
		t.Fatalf("expected unknown_reference, got %v", codes(res))
	}
}

// TestValidateLedgerDuplicateEvidenceReferenceRejected witnesses that the
// same evidence cannot be attached twice.
func TestValidateLedgerDuplicateEvidenceReferenceRejected(t *testing.T) {
	l := baseLedger()
	l.Evidence = []EvidenceRefRecord{{
		SchemaVersion: LedgerSchemaVersion,
		EvidenceID:    "EV-A",
		Kind:          "file",
		Title:         "Evidence",
		Revision:      1,
	}}
	l.Items[0].Evidence = []EvidenceRef{
		{EvidenceID: "EV-A", Quality: EvidenceQualityVerified},
		{EvidenceID: "EV-A", Quality: EvidenceQualityVerified},
	}

	res := ValidateLedger(l)
	if !hasCode(res, CodeDuplicateEvidence) {
		t.Fatalf("expected duplicate_evidence, got %v", codes(res))
	}
}

// TestValidateLedgerArchivedItemRequiresArchivedLifecycle witnesses that
// the archived flag and the lifecycle state stay consistent.
func TestValidateLedgerArchivedItemRequiresArchivedLifecycle(t *testing.T) {
	l := baseLedger()
	l.Items[0].Archived = true

	res := ValidateLedger(l)
	if !hasCode(res, CodeInvalidEnum) {
		t.Fatalf("expected invalid_enum, got %v", codes(res))
	}
}
