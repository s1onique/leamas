package importer

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"

	"github.com/s1onique/leamas/internal/ledger/domain"
)

// OriginalFixtureName is the file name of the original IECR board intake
// fixture that the FWL01 amendment requires.
const OriginalFixtureName = "iecr-board-intake-fixture.json"

// originalFixtureBytes is the EXPECTED byte size of the original fixture.
const originalFixtureBytes = 21043

// originalFixtureSHA256 is the EXPECTED digest of the original fixture.
const originalFixtureSHA256 = "bab945991714438fdf66ba5b2efcd7d2a13e484f7d8804a25c7084319bba574c"

// originalFixturePath returns the path of the original fixture.
func originalFixturePath(t *testing.T) string {
	t.Helper()
	return filepath.Join("testdata", OriginalFixtureName)
}

// loadOriginalFixture reads the original fixture and verifies its identity.
//
// The identity check is part of the qualification: the original fixture is
// the operator-supplied document, and a substituted document would make
// every assertion below meaningless.
func loadOriginalFixture(t *testing.T) []byte {
	t.Helper()
	path := originalFixturePath(t)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("MISSING INPUT: the original IECR intake fixture %q is not available in the repository (%v). "+
			"The repository-native fixture testdata/iecr-intake-board.json is SYNTHETIC and cannot substitute "+
			"for the original migration gate.", path, err)
	}
	if len(data) != originalFixtureBytes {
		t.Fatalf("the original fixture has %d bytes; expected %d", len(data), originalFixtureBytes)
	}
	sum := sha256.Sum256(data)
	got := hex.EncodeToString(sum[:])
	if got != originalFixtureSHA256 {
		t.Fatalf("the original fixture digest is %s; expected %s", got, originalFixtureSHA256)
	}
	return data
}

// TestOriginalIECRFixtureIdentityAndInventory qualifies the ORIGINAL fixture:
// it is parsed, its inventory is observed and every declared count is
// verified.
//
// This is NOT a presence-only test: the document is actually parsed.
func TestOriginalIECRFixtureIdentityAndInventory(t *testing.T) {
	data := loadOriginalFixture(t)

	board, inventory, err := ParseOriginalFixture(data)
	if err != nil {
		t.Fatalf("parse the original fixture: %v", err)
	}

	if inventory.Classification != OriginalClassification {
		t.Fatalf("expected the classification %q, got %q", OriginalClassification, inventory.Classification)
	}
	if inventory.WorkItems != 44 {
		t.Fatalf("expected 44 work items, got %d", inventory.WorkItems)
	}
	if inventory.Workstreams != 8 {
		t.Fatalf("expected 8 workstreams, got %d", inventory.Workstreams)
	}
	if inventory.Epics != 1 {
		t.Fatalf("expected 1 parent epic, got %d", inventory.Epics)
	}
	if inventory.ExitCriteria != 12 {
		t.Fatalf("expected 12 exit criteria, got %d", inventory.ExitCriteria)
	}
	if inventory.DistinctStatuses != 10 {
		t.Fatalf("expected 10 distinct source statuses, got %d", inventory.DistinctStatuses)
	}
	if got := inventory.StatusCounts["CLOSED"]; got != 10 {
		t.Fatalf("expected 10 CLOSED records, got %d", got)
	}
	if got := inventory.StatusCounts["CLOSED_INSUFFICIENT"]; got != 1 {
		t.Fatalf("expected 1 CLOSED_INSUFFICIENT record, got %d", got)
	}
	if got := inventory.StatusCounts["BLOCKED"]; got != 6 {
		t.Fatalf("expected 6 BLOCKED records, got %d", got)
	}
	if inventory.Unfinished != 33 {
		t.Fatalf("expected 33 unfinished records, got %d", inventory.Unfinished)
	}
	if len(board.Items) != 44 {
		t.Fatalf("expected 44 normalized source records, got %d", len(board.Items))
	}
}

// TestOriginalIECRFixtureImportPersistsAndReloads qualifies the ORIGINAL
// fixture end to end: the import is applied, the state is persisted, the
// state is reloaded and the persisted state passes the authoritative
// validator.
func TestOriginalIECRFixtureImportPersistsAndReloads(t *testing.T) {
	data := loadOriginalFixture(t)
	s := newTestStore(t)

	result, err := ImportOriginalFixture(s, data, "tx-original-import")
	if err != nil {
		t.Fatalf("import the original fixture: %v", err)
	}

	// The imported records are persisted.
	if got := len(result.Plan.Items); got != 44 {
		t.Fatalf("expected 44 imported records, got %d", got)
	}
	if got := len(result.Plan.Epics); got != 1 {
		t.Fatalf("expected 1 imported epic, got %d", got)
	}

	// The state is reloaded from disk.
	reloaded, err := s.Load()
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	if len(reloaded.Items) != 44 {
		t.Fatalf("expected 44 persisted records, got %d", len(reloaded.Items))
	}
	if len(reloaded.Epics) != 1 {
		t.Fatalf("expected 1 persisted epic, got %d", len(reloaded.Epics))
	}

	// The persisted state passes the AUTHORITATIVE validator.
	res := domain.ValidateLedger(reloaded)
	if !res.OK() {
		t.Fatalf("the persisted state does not pass the authoritative validator: %v", res.Errors())
	}

	// The parent epic identity is preserved: the workstreams did NOT
	// become eight independent epics.
	epic, ok := reloaded.Epics[0].EpicID, true
	if !ok || epic != "EPIC-CLINEMM-IECR" {
		t.Fatalf("expected the parent epic EPIC-CLINEMM-IECR, got %q", epic)
	}
	workstreams := map[string]bool{}
	for _, it := range reloaded.Items {
		workstreams[string(it.EpicID)] = true
	}
	if len(workstreams) != 1 {
		t.Fatalf("expected 1 preserved epic scope, got %d: %v", len(workstreams), workstreams)
	}
}

// TestOriginalIECRFixtureConservesEverySourceIdentity proves that every
// source identity of the original fixture is accounted for and that every
// source status is conserved.
func TestOriginalIECRFixtureConservesEverySourceIdentity(t *testing.T) {
	data := loadOriginalFixture(t)
	s := newTestStore(t)

	result, err := ImportOriginalFixture(s, data, "tx-original-import")
	if err != nil {
		t.Fatalf("import the original fixture: %v", err)
	}

	reloaded, err := s.Load()
	if err != nil {
		t.Fatalf("reload: %v", err)
	}

	// Every source identity is accounted for: the persisted records carry
	// exactly the source identities of the original rows.
	persisted := map[string]domain.WorkItem{}
	for _, it := range reloaded.Items {
		if it.Provenance == nil {
			t.Fatalf("the record %q lost its provenance", it.Scoped().String())
		}
		persisted[it.Provenance.SourceID] = it
	}
	sourceIDs := map[string]bool{}
	for _, src := range result.Board.Items {
		sourceIDs[src.SourceID] = true
		if _, ok := persisted[src.SourceID]; !ok {
			t.Fatalf("the source identity %q is not accounted for in the persisted state", src.SourceID)
		}
	}
	if len(sourceIDs) != 44 {
		t.Fatalf("expected 44 distinct source identities, got %d", len(sourceIDs))
	}

	// Every source status is conserved VERBATIM.
	observed := map[string]int{}
	for _, it := range reloaded.Items {
		observed[it.SourceStatus]++
	}
	for status, want := range result.Inventory.StatusCounts {
		if observed[status] != want {
			t.Fatalf("the source status %q was not conserved: expected %d records, got %d",
				status, want, observed[status])
		}
	}
	if len(observed) != 10 {
		t.Fatalf("expected 10 conserved source statuses, got %d", len(observed))
	}

	// The original row text and the source locator are preserved.
	for _, src := range result.Board.Items {
		it := persisted[src.SourceID]
		if it.SourceRow != src.SourceRow {
			t.Fatalf("the original row text of %q was not preserved: %q", src.SourceID, it.SourceRow)
		}
		if it.SourceLine != src.SourceLine {
			t.Fatalf("the source locator of %q was not preserved: %d", src.SourceID, it.SourceLine)
		}
		if it.SourceDetail != src.SourceDetail {
			t.Fatalf("the source detail of %q was not preserved: %q", src.SourceID, it.SourceDetail)
		}
	}
}

// TestOriginalIECRFixtureDeferredAndUnknownWorkStaysNonExecutable proves
// that deferred work and unknown statuses remain excluded from execution.
func TestOriginalIECRFixtureDeferredAndUnknownWorkStaysNonExecutable(t *testing.T) {
	data := loadOriginalFixture(t)
	s := newTestStore(t)

	result, err := ImportOriginalFixture(s, data, "tx-original-import")
	if err != nil {
		t.Fatalf("import the original fixture: %v", err)
	}
	if result.Inventory.WorkItems != 44 {
		t.Fatalf("expected 44 observed source records, got %d", result.Inventory.WorkItems)
	}
	reloaded, err := s.Load()
	if err != nil {
		t.Fatalf("reload: %v", err)
	}

	// Every DEFERRED record is excluded from execution.
	for _, it := range reloaded.Items {
		if it.SourceStatus == "DEFERRED" && it.Acceptance != domain.AcceptanceDeferred {
			t.Fatalf("the deferred record %q was not deferred: %q", it.Scoped().String(), it.Acceptance)
		}
	}

	// The original fixture carries ten source statuses. Every one of them
	// must be understood by the importer: an unknown status would be a
	// defect in the ORIGINAL board, not a normal state.
	for _, it := range reloaded.Items {
		if !IsKnownSourceStatus(SourceStatus(it.SourceStatus)) {
			t.Fatalf("the source status %q of %q is not understood by the importer",
				it.SourceStatus, it.Scoped().String())
		}
	}

	// An UNKNOWN source status stays durably non-executable. The original
	// fixture carries no unknown status, so the path is qualified with a
	// mutated copy of the original document.
	mutated := changeOriginalFixtureStatus(t, data, "RC-01", "TOTALLY_UNKNOWN")
	mutatedStore := newTestStore(t)
	if _, err := ImportOriginalFixture(mutatedStore, mutated, "tx-mutated-import"); err != nil {
		t.Fatalf("import the mutated fixture: %v", err)
	}
	mutatedState, err := mutatedStore.Load()
	if err != nil {
		t.Fatalf("reload the mutated fixture: %v", err)
	}
	unknown := 0
	for _, it := range mutatedState.Items {
		if IsKnownSourceStatus(SourceStatus(it.SourceStatus)) {
			continue
		}
		unknown++
		if it.Acceptance != domain.AcceptanceDeferred {
			t.Fatalf("the unknown-status record %q is not durably non-executable: %q",
				it.Scoped().String(), it.Acceptance)
		}
		if it.SourceStatus != "TOTALLY_UNKNOWN" {
			t.Fatalf("the unknown source status was not conserved verbatim: %q", it.SourceStatus)
		}
	}
	if unknown != 1 {
		t.Fatalf("expected exactly 1 unknown-status record, got %d", unknown)
	}

	// No record of the imported board is accepted: a conversational import
	// never establishes acceptance.
	for _, it := range reloaded.Items {
		if it.Acceptance == domain.AcceptanceAccepted {
			t.Fatalf("the imported record %q was accepted by the import", it.Scoped().String())
		}
	}
	ready := domain.SelectReady(reloaded, domain.ReadyOptions{})
	for _, r := range ready {
		item, _ := reloaded.ItemByScoped(r.ScopedID)
		if item.SourceStatus == "DEFERRED" {
			t.Fatalf("the deferred record %q was selected for execution", r.ScopedID.String())
		}
	}
}

// TestOriginalIECRFixtureInsufficientInvestigationDoesNotAccept proves that
// a CLOSED_INSUFFICIENT investigation does not establish acceptance and that
// a partial result does not become complete.
func TestOriginalIECRFixtureInsufficientInvestigationDoesNotAccept(t *testing.T) {
	data := loadOriginalFixture(t)
	s := newTestStore(t)

	if _, err := ImportOriginalFixture(s, data, "tx-original-import"); err != nil {
		t.Fatalf("import the original fixture: %v", err)
	}
	reloaded, err := s.Load()
	if err != nil {
		t.Fatalf("reload: %v", err)
	}

	insufficient := 0
	partial := 0
	for _, it := range reloaded.Items {
		switch it.SourceStatus {
		case "CLOSED_INSUFFICIENT":
			insufficient++
			if it.Acceptance == domain.AcceptanceAccepted {
				t.Fatalf("the insufficient investigation %q established acceptance", it.Scoped().String())
			}
			if it.TerminalDisposition != "CLOSED_INSUFFICIENT" {
				t.Fatalf("the insufficient disposition of %q was not conserved: %q",
					it.Scoped().String(), it.TerminalDisposition)
			}
		case "PARTIAL":
			partial++
			if it.Acceptance == domain.AcceptanceAccepted {
				t.Fatalf("the partial result %q became complete", it.Scoped().String())
			}
		}
	}
	if insufficient != 1 {
		t.Fatalf("expected 1 insufficient investigation, got %d", insufficient)
	}
	if partial != 6 {
		t.Fatalf("expected 6 partial results, got %d", partial)
	}
}

// TestOriginalIECRFixtureNeverAcquiresVerifiedProvenance proves that a
// conversational claim does not acquire repository-verified provenance.
func TestOriginalIECRFixtureNeverAcquiresVerifiedProvenance(t *testing.T) {
	data := loadOriginalFixture(t)
	s := newTestStore(t)

	if _, err := ImportOriginalFixture(s, data, "tx-original-import"); err != nil {
		t.Fatalf("import the original fixture: %v", err)
	}
	reloaded, err := s.Load()
	if err != nil {
		t.Fatalf("reload: %v", err)
	}

	for _, it := range reloaded.Items {
		if it.Provenance == nil {
			t.Fatalf("the record %q lost its provenance", it.Scoped().String())
		}
		if it.Provenance.Confidence == string(domain.ProvenanceRepository) {
			t.Fatalf("the record %q was promoted to repository_verified", it.Scoped().String())
		}
		if it.Quality != domain.EvidenceQualityNone {
			t.Fatalf("the record %q acquired evidence quality %q without any evidence",
				it.Scoped().String(), it.Quality)
		}
	}
	if len(reloaded.Evidence) != 0 {
		t.Fatalf("the conversational import acquired %d evidence records", len(reloaded.Evidence))
	}
}
