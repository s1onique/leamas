package importer

import (
	"testing"

	"github.com/s1onique/leamas/internal/ledger/domain"
)

// TestSyntheticFixtureIsNotRepositoryVerified proves that the synthetic
// fixture never produces repository-verified provenance.
//
// The repository-native fixture (iecr-intake-board.json) is a SYNTHETIC
// constructed fixture: it is NOT the original and must never be claimed to
// be.
func TestSyntheticFixtureIsNotRepositoryVerified(t *testing.T) {
	board := iecrSourceBoard()
	plan := BuildPlan(board, domain.Ledger{SchemaVersion: domain.LedgerSchemaVersion})
	if len(plan.Items) == 0 {
		t.Fatal("expected the synthetic fixture to produce imported records")
	}
	for _, item := range plan.Items {
		if item.Provenance == nil {
			t.Fatalf("record %s lost its provenance", item.Scoped().String())
		}
		if item.Provenance.Confidence == "repository_verified" {
			t.Fatalf("record %s was promoted to repository_verified", item.Scoped().String())
		}
	}
}

// TestOriginalFixtureRejectsUnknownVersion proves the versioned adapter
// refuses a document it does not understand.
func TestOriginalFixtureRejectsUnknownVersion(t *testing.T) {
	data := loadOriginalFixture(t)
	changed := []byte(replaceFirst(t, string(data), `"fixture_version": 1`, `"fixture_version": 2`))
	if _, _, err := ParseOriginalFixture(changed); err == nil {
		t.Fatal("an unsupported original fixture version was accepted")
	}
}

// changeOriginalFixtureTitle rewrites the title of one source row.
//
// It replaces the TITLE VALUE of the row that carries the local identity.
// The surrounding rows are untouched, so the change is exactly one source
// record.
func changeOriginalFixtureTitle(t *testing.T, data []byte, localID, title string) []byte {
	t.Helper()
	text := string(data)
	marker := `"local_id": "` + localID + `"`
	idx := indexOf(t, text, marker)
	// The title follows the local identity in the original document.
	titleMarker := `"title": "`
	titleIdx := indexOf(t, text[idx:], titleMarker)
	absTitleIdx := idx + titleIdx + len(titleMarker)
	endIdx := indexOf(t, text[absTitleIdx:], `"`)
	absEnd := absTitleIdx + endIdx
	return []byte(text[:absTitleIdx] + title + text[absEnd:])
}

// changeOriginalFixtureStatus rewrites the source status of one source row.
//
// It replaces the SOURCE_STATUS VALUE of the row that carries the local
// identity.
func changeOriginalFixtureStatus(t *testing.T, data []byte, localID, status string) []byte {
	t.Helper()
	text := string(data)
	idx := indexOf(t, text, `"local_id": "`+localID+`"`)
	marker := `"source_status": "`
	absMarker := idx + indexOf(t, text[idx:], marker)
	start := absMarker + len(marker)
	end := start + indexOf(t, text[start:], `"`)
	return []byte(text[:start] + status + text[end:])
}

// indexOf returns the index of the first occurrence of sub in s.
func indexOf(t *testing.T, s, sub string) int {
	t.Helper()
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	t.Fatalf("the fixture does not contain %q", sub)
	return -1
}

// replaceFirst replaces the first occurrence of old in s.
func replaceFirst(t *testing.T, s, old, new string) string {
	t.Helper()
	idx := -1
	for i := 0; i+len(old) <= len(s); i++ {
		if s[i:i+len(old)] == old {
			idx = i
			break
		}
	}
	if idx < 0 {
		t.Fatalf("the fixture does not contain %q", old)
	}
	return s[:idx] + new + s[idx+len(old):]
}

// contains reports whether s contains sub.
func contains(s, sub string) bool {
	return len(s) >= len(sub) && (func() bool {
		for i := 0; i+len(sub) <= len(s); i++ {
			if s[i:i+len(sub)] == sub {
				return true
			}
		}
		return false
	})()
}

// TestOriginalIECRFixtureReportsUnavailableHistory proves that the missing
// historical information of the original fixture remains EXPLICITLY
// unavailable.
func TestOriginalIECRFixtureReportsUnavailableHistory(t *testing.T) {
	data := loadOriginalFixture(t)
	s := newTestStore(t)

	result, err := ImportOriginalFixture(s, data, "tx-original-import")
	if err != nil {
		t.Fatalf("import the original fixture: %v", err)
	}
	if len(result.Unavailable) == 0 {
		t.Fatal("the unavailable historical information was not reported")
	}
	joined := ""
	for _, u := range result.Unavailable {
		joined += u + "\n"
	}
	if !contains(joined, "not the full historical repository board") {
		t.Fatalf("the unavailability report does not state the board scope: %s", joined)
	}

	// The report is stable and deterministic.
	second := UnavailableHistoricalInformation()
	if len(second) != len(result.Unavailable) {
		t.Fatal("the unavailability report is not deterministic")
	}
}

// TestOriginalIECRFixtureChangedReplayProducesExplicitReconciliation proves
// that a changed replay produces an explicit reconciliation and does NOT
// overwrite the canonical work.
func TestOriginalIECRFixtureChangedReplayProducesExplicitReconciliation(t *testing.T) {
	data := loadOriginalFixture(t)
	s := newTestStore(t)

	if _, err := ImportOriginalFixture(s, data, "tx-original-import"); err != nil {
		t.Fatalf("first import: %v", err)
	}
	before, err := s.Load()
	if err != nil {
		t.Fatalf("reload: %v", err)
	}

	// Change ONE source row and replay.
	changed := changeOriginalFixtureTitle(t, data, "RC-01", "CHANGED title")
	second, err := ImportOriginalFixture(s, changed, "tx-original-changed")
	if err != nil {
		t.Fatalf("changed replay: %v", err)
	}
	after, err := s.Load()
	if err != nil {
		t.Fatalf("reload: %v", err)
	}

	// The canonical work is NOT overwritten.
	prior, _ := before.ItemByScoped("EPIC-CLINEMM-IECR/RC-01")
	now, _ := after.ItemByScoped("EPIC-CLINEMM-IECR/RC-01")
	if now.Title != prior.Title {
		t.Fatalf("the canonical record was overwritten by the changed source: %q -> %q",
			prior.Title, now.Title)
	}

	// The change is an EXPLICIT reconciliation.
	found := false
	for _, rec := range second.Reconciliations {
		if rec.ScopedID == "EPIC-CLINEMM-IECR/RC-01" {
			found = true
			if rec.Outcome != ReconciliationSourceChanged {
				t.Fatalf("expected a source_changed reconciliation, got %q", rec.Outcome)
			}
		}
	}
	if !found {
		t.Fatal("the changed source record produced no explicit reconciliation")
	}
}

// TestOriginalIECRFixtureExactReplayProducesNoDelta proves that an exact
// replay produces no canonical state delta.
func TestOriginalIECRFixtureExactReplayProducesNoDelta(t *testing.T) {
	data := loadOriginalFixture(t)
	s := newTestStore(t)

	if _, err := ImportOriginalFixture(s, data, "tx-original-import"); err != nil {
		t.Fatalf("first import: %v", err)
	}
	before, err := s.Load()
	if err != nil {
		t.Fatalf("reload: %v", err)
	}

	second, err := ImportOriginalFixture(s, data, "tx-original-replay")
	if err != nil {
		t.Fatalf("replay: %v", err)
	}
	after, err := s.Load()
	if err != nil {
		t.Fatalf("reload: %v", err)
	}

	delta := ComputeDelta(before, after)
	if !delta.Empty() {
		t.Fatalf("the exact replay produced a canonical delta: %s", delta.Describe())
	}
	if len(second.Plan.Items) != 0 {
		t.Fatalf("the exact replay planned %d records", len(second.Plan.Items))
	}
	unchanged := 0
	for _, rec := range second.Reconciliations {
		if rec.Outcome == ReconciliationUnchanged {
			unchanged++
		}
	}
	if unchanged != 44 {
		t.Fatalf("expected 44 unchanged reconciliations, got %d", unchanged)
	}
}
