package service

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/s1onique/leamas/internal/ledger/domain"
	"github.com/s1onique/leamas/internal/ledger/store"
)

// This file qualifies the FWL01 amendment addendum authority witnesses.
//
// Each witness drives the REAL production seam (service + store + domain)
// and asserts the REQUIRED behaviour.

// TestWitnessAcceptedItemWithoutEvidenceIsRejected proves that a canonical
// accepted live item without admissible evidence is reported.
//
// The historical acceptance is PRESERVED — the item keeps its accepted
// state — but the invalidation is REPORTED, never silent.
func TestWitnessAcceptedItemWithoutEvidenceIsRejected(t *testing.T) {
	svc := newTestService(t)
	created := createItem(t, svc, "EPIC-A", "ITEM-1", "Needs evidence")
	l, err := svc.Load()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	item, _ := l.ItemByScoped(created.Scoped())

	// Force the accepted state without admissible evidence.
	mutated := item
	mutated.Acceptance = domain.AcceptanceAccepted
	mutated.Lifecycle = domain.LifecycleClosed
	mutated.Quality = domain.EvidenceQualityVerified
	mutated.AcceptanceCriteria = []string{"criterion"}
	txn, err := svc.Store.BeginMutation("tx-force")
	if err != nil {
		t.Fatalf("begin: %v", err)
	}
	if err := txn.PutItem(mutated); err != nil {
		t.Fatalf("put: %v", err)
	}
	if err := txn.Commit(); err == nil {
		t.Fatal("the accepted item without evidence was published without a report")
	}

	// The state was NOT published: the validator rejects the candidate.
	after, err := svc.Load()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	it, _ := after.ItemByScoped(created.Scoped())
	if it.Acceptance == domain.AcceptanceAccepted {
		t.Fatal("the accepted item without evidence was published")
	}
}

// TestWitnessUnrelatedEvidenceDoesNotSatisfyAnObligation proves that
// unrelated evidence never satisfies an acceptance obligation.
//
// The item declares TWO criteria. The evidence covers only ONE of them.
func TestWitnessUnrelatedEvidenceDoesNotSatisfyAnObligation(t *testing.T) {
	dir := t.TempDir()
	svc := New(store.New(filepath.Join(dir, "ledger"))).WithRepoRoot(dir)
	if err := svc.Init(); err != nil {
		t.Fatalf("init: %v", err)
	}
	artifact := filepath.Join(dir, "a.txt")
	if err := os.WriteFile(artifact, []byte("evidence\n"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	created, err := svc.CreateItem("tx", CreateItemOptions{
		EpicID: "EPIC-A", ItemID: "ITEM-1", Title: "two criteria",
		Criteria: []string{"criterion A", "criterion B"},
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	sum := sha256.Sum256([]byte("evidence\n"))
	digest := hex.EncodeToString(sum[:])

	if _, err := svc.AddEvidence("tx-ev", AddEvidenceOptions{
		EvidenceID: "EV-A", Kind: "structural", Title: "covers criterion A",
		RelativePath: "a.txt", SHA256: digest, ItemID: created.Scoped(),
		Quality: domain.EvidenceQualityVerified, Class: domain.EvidenceClassStructural,
		Criterion:   "criterion A",
		ExpectedRev: created.Revision,
	}); err != nil {
		t.Fatalf("add evidence: %v", err)
	}
	l, err := svc.Load()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	item, _ := l.ItemByScoped(created.Scoped())
	_, err = svc.Accept("tx-accept", AcceptOptions{
		ScopedID: item.Scoped(), ExpectedRev: item.Revision, AcceptedBy: "acceptor",
	})
	if err == nil {
		t.Fatal("the unrelated evidence satisfied the second obligation")
	}
	if !errors.Is(err, ErrAcceptanceRejected) {
		t.Fatalf("expected ErrAcceptanceRejected, got %v", err)
	}
}

// TestWitnessReopenedItemIsActionableAgain proves that an explicit reopening
// creates actionable work again without erasing the historical acceptance
// record.
func TestWitnessReopenedItemIsActionableAgain(t *testing.T) {
	dir := t.TempDir()
	svc := New(store.New(filepath.Join(dir, "ledger"))).WithRepoRoot(dir)
	if err := svc.Init(); err != nil {
		t.Fatalf("init: %v", err)
	}
	artifact := filepath.Join(dir, "a.txt")
	if err := os.WriteFile(artifact, []byte("evidence\n"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	created, err := svc.CreateItem("tx", CreateItemOptions{
		EpicID: "EPIC-A", ItemID: "ITEM-1", Title: "work",
		Criteria: []string{"criterion"},
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	sum := sha256.Sum256([]byte("evidence\n"))
	digest := hex.EncodeToString(sum[:])
	if _, err := svc.AddEvidence("tx-ev", AddEvidenceOptions{
		EvidenceID: "EV-A", Kind: "structural", Title: "evidence",
		RelativePath: "a.txt", SHA256: digest, ItemID: created.Scoped(),
		Quality: domain.EvidenceQualityVerified, Class: domain.EvidenceClassStructural,
		Criterion:   "criterion",
		ExpectedRev: created.Revision,
	}); err != nil {
		t.Fatalf("add evidence: %v", err)
	}
	l, err := svc.Load()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	item, _ := l.ItemByScoped(created.Scoped())
	accepted, err := svc.Accept("tx-accept", AcceptOptions{
		ScopedID: item.Scoped(), ExpectedRev: item.Revision,
		AcceptedBy: "acceptor", Basis: "the qualification passed",
	})
	if err != nil {
		t.Fatalf("accept: %v", err)
	}
	if accepted.AcceptedBy != "acceptor" {
		t.Fatalf("the acceptance decision was not recorded: %q", accepted.AcceptedBy)
	}

	// Reopen the accepted item.
	reopened, err := svc.Reopen("tx-reopen", ReopenOptions{
		ScopedID: accepted.Scoped(), ExpectedRev: accepted.Revision,
	})
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	if reopened.Lifecycle != domain.LifecycleOpen {
		t.Fatalf("the reopened item is not actionable: lifecycle %q", reopened.Lifecycle)
	}
	if reopened.Acceptance == domain.AcceptanceAccepted {
		t.Fatal("the reopened item is still a satisfied obligation")
	}
	// The historical acceptance record is preserved.
	if reopened.AcceptedBy != "acceptor" || reopened.AcceptanceBasis != "the qualification passed" {
		t.Fatalf("the historical acceptance record was erased: %+v", reopened)
	}
	// The reopened item is selectable again.
	state, err := svc.Load()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	ready := domain.SelectReady(state, domain.ReadyOptions{})
	found := false
	for _, r := range ready {
		if r.ScopedID == reopened.Scoped() {
			found = true
		}
	}
	if !found {
		t.Fatal("the reopened item is not executable again")
	}
}

// TestWitnessClaimCannotReplaceAnOwner proves that a claim never replaces
// an existing owner.
func TestWitnessClaimCannotReplaceAnOwner(t *testing.T) {
	svc := newTestService(t)
	created := createItem(t, svc, "EPIC-A", "ITEM-1", "owned")
	if _, err := svc.Claim("tx-c1", ClaimOptions{
		ScopedID: created.Scoped(), ExpectedRev: created.Revision, Owner: "owner-a",
	}); err != nil {
		t.Fatalf("claim a: %v", err)
	}
	l, err := svc.Load()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	item, _ := l.ItemByScoped(created.Scoped())
	if _, err := svc.Claim("tx-c2", ClaimOptions{
		ScopedID: item.Scoped(), ExpectedRev: item.Revision, Owner: "owner-b",
	}); err == nil {
		t.Fatal("the claim replaced the existing owner")
	} else if !errors.Is(err, ErrOwnershipTransferRequired) {
		t.Fatalf("expected ErrOwnershipTransferRequired, got %v", err)
	}
}

// TestWitnessMutationRequiresPositiveExpectedRevision proves that an
// externally initiated mutation of an existing record requires a positive
// expected revision.
func TestWitnessMutationRequiresPositiveExpectedRevision(t *testing.T) {
	svc := newTestService(t)
	created := createItem(t, svc, "EPIC-A", "ITEM-1", "work")
	title := "changed"
	if _, err := svc.UpdateItem("tx", UpdateItemOptions{
		ScopedID: created.Scoped(), ExpectedRev: 0, Title: &title,
	}); err == nil {
		t.Fatal("the mutation without an expected revision was accepted")
	} else if !errors.Is(err, ErrStaleRevision) {
		t.Fatalf("expected ErrStaleRevision, got %v", err)
	}
}

// TestWitnessGenericUpdateCannotHideAnObligation proves that a generic
// lifecycle update may not hide an unresolved obligation.
func TestWitnessGenericUpdateCannotHideAnObligation(t *testing.T) {
	svc := newTestService(t)
	created := createItem(t, svc, "EPIC-A", "ITEM-1", "work")
	closed := domain.LifecycleClosed
	if _, err := svc.UpdateItem("tx", UpdateItemOptions{
		ScopedID: created.Scoped(), ExpectedRev: created.Revision, Lifecycle: &closed,
	}); err == nil {
		t.Fatal("the generic update hid the unresolved obligation")
	} else if !errors.Is(err, ErrUnfinishedObligation) {
		t.Fatalf("expected ErrUnfinishedObligation, got %v", err)
	}
}

// TestWitnessInvalidatedEvidenceIsPersistedAndReported proves that the
// evidence invalidation transaction is accepted by the candidate validator
// and that the invalidation is reported.
func TestWitnessInvalidatedEvidenceIsPersistedAndReported(t *testing.T) {
	dir := t.TempDir()
	svc := New(store.New(filepath.Join(dir, "ledger"))).WithRepoRoot(dir)
	if err := svc.Init(); err != nil {
		t.Fatalf("init: %v", err)
	}
	artifact := filepath.Join(dir, "a.txt")
	payload := []byte("evidence\n")
	if err := os.WriteFile(artifact, payload, 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	created, err := svc.CreateItem("tx", CreateItemOptions{
		EpicID: "EPIC-A", ItemID: "ITEM-1", Title: "work",
		Criteria: []string{"criterion"},
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	sum := sha256.Sum256(payload)
	digest := hex.EncodeToString(sum[:])
	if _, err := svc.AddEvidence("tx-ev", AddEvidenceOptions{
		EvidenceID: "EV-A", Kind: "structural", Title: "evidence",
		RelativePath: "a.txt", SHA256: digest, ItemID: created.Scoped(),
		Quality: domain.EvidenceQualityVerified, Class: domain.EvidenceClassStructural,
		Criterion:   "criterion",
		ExpectedRev: created.Revision,
	}); err != nil {
		t.Fatalf("add evidence: %v", err)
	}
	l, err := svc.Load()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	item, _ := l.ItemByScoped(created.Scoped())
	accepted, err := svc.Accept("tx-accept", AcceptOptions{
		ScopedID: item.Scoped(), ExpectedRev: item.Revision, AcceptedBy: "acceptor",
	})
	if err != nil {
		t.Fatalf("accept: %v", err)
	}

	// The artifact changes: the evidence is invalidated.
	if err := os.WriteFile(artifact, []byte("changed\n"), 0o644); err != nil {
		t.Fatalf("rewrite: %v", err)
	}
	out, err := svc.RevalidateEvidence("tx-reval", RevalidateEvidenceOptions{})
	if err != nil {
		t.Fatalf("the invalidation transaction was rejected: %v", err)
	}
	if len(out) != 1 || out[0].Valid {
		t.Fatalf("the invalidation was not reported: %+v", out)
	}

	// The historical acceptance is PRESERVED.
	state, err := svc.Load()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	after, _ := state.ItemByScoped(created.Scoped())
	if after.Acceptance != domain.AcceptanceAccepted {
		t.Fatalf("the historical acceptance was erased: %q", after.Acceptance)
	}
	// The invalidation is REPORTED as an advisory, not as a rejection.
	res := domain.ValidateLedger(state)
	if !res.OK() {
		t.Fatalf("the invalidation state does not pass the validator: %v", res.Errors())
	}
	reported := false
	for _, a := range res.Advisories {
		if a.Code == domain.CodeAcceptanceAuthorityInvalid {
			reported = true
		}
	}
	if !reported {
		t.Fatalf("the invalidation was not reported: %+v", res.Advisories)
	}
	// The invalidated authority does not SILENTLY satisfy a downstream
	// prerequisite.
	if domain.CurrentlyAuthorized(state, after) {
		t.Fatal("the invalidated authority still authorizes a downstream prerequisite")
	}
	_ = accepted
}
