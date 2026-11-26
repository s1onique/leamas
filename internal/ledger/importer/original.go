package importer

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/s1onique/leamas/internal/ledger/domain"
	"github.com/s1onique/leamas/internal/ledger/store"
)

// OriginalFixtureVersion is the version of the ORIGINAL IECR board intake
// fixture schema this adapter understands.
//
// The original schema is NOT the SourceBoard schema. It is a pasted-text
// board capture: its work items carry a local identity, a workstream
// abbreviation, an UPPERCASE source status, a source priority that may be
// an em dash, a free-form detail and the original row text.
//
// The adapter is deliberately the MINIMUM necessary versioned adapter: it
// reads the original schema, normalizes it into a SourceBoard and reports
// what could not be normalized.
const OriginalFixtureVersion = 1

// OriginalClassification is the classification of the original fixture.
//
// The original board is a CONVERSATIONAL import that has NOT been verified
// against the repository. The classification is preserved through the
// import: no imported record may acquire repository-verified provenance.
const OriginalClassification = "CONVERSATIONAL_IMPORT_UNVERIFIED"

// originalFixture is the parsed ORIGINAL fixture document.
type originalFixture struct {
	// FixtureVersion is the version of the original schema.
	FixtureVersion int `json:"fixture_version"`
	// Classification is the declared classification of the document.
	Classification string `json:"classification"`
	// Source describes the original paste the fixture was captured from.
	Source originalSource `json:"source"`
	// Counts declares the expected inventory of the document.
	Counts map[string]int `json:"counts"`
	// Items are the original work-item rows.
	Items []originalItem `json:"items"`
	// ExitCriteria are the board-scoped exit criteria.
	ExitCriteria []originalExitCriterion `json:"exit_criteria"`
	// Notes are the free-form notes of the document.
	Notes []string `json:"notes"`
}

// originalSource describes the origin of the original document.
type originalSource struct {
	// Filename is the name of the original paste.
	Filename string `json:"filename"`
	// SHA256 is the digest of the original paste.
	SHA256 string `json:"sha256"`
	// Bytes is the size of the original paste.
	Bytes int `json:"bytes"`
	// BoardRevision is the board revision the capture was taken at.
	BoardRevision string `json:"board_revision"`
}

// originalItem is one original work-item row.
type originalItem struct {
	// LocalID is the identity the row carried in the original board.
	LocalID string `json:"local_id"`
	// Epic is the parent epic identity of the row.
	Epic string `json:"epic"`
	// Workstream is the workstream abbreviation of the row.
	Workstream string `json:"workstream"`
	// Title is the title of the row.
	Title string `json:"title"`
	// SourceStatus is the EXACT source status of the row.
	SourceStatus string `json:"source_status"`
	// SourcePriority is the observed source priority. It may be a dash.
	SourcePriority string `json:"source_priority"`
	// SourceDetail is the free-form detail of the row.
	SourceDetail string `json:"source_detail"`
	// SourceLine is the original line locator of the row.
	SourceLine int `json:"source_line"`
	// SourceRow is the original row text.
	SourceRow string `json:"source_row"`
}

// originalExitCriterion is one board-scoped exit criterion.
type originalExitCriterion struct {
	// SourceLine is the original line locator.
	SourceLine int `json:"source_line"`
	// Text is the criterion text.
	Text string `json:"text"`
}

// OriginalInventory is the observed inventory of the original document.
//
// It is the EVIDENCE of the conservation: every source identity, every
// source status and every workstream is counted, not merely the number of
// imported records.
type OriginalInventory struct {
	// WorkItems is the number of original work-item rows.
	WorkItems int `json:"work_items"`
	// Workstreams is the number of distinct source workstreams.
	Workstreams int `json:"workstreams"`
	// Epics is the number of distinct parent epics.
	Epics int `json:"epics"`
	// ExitCriteria is the number of board-scoped exit criteria.
	ExitCriteria int `json:"exit_criteria"`
	// DistinctStatuses is the number of distinct source statuses.
	DistinctStatuses int `json:"distinct_statuses"`
	// StatusCounts maps every source status to its observed count.
	StatusCounts map[string]int `json:"status_counts"`
	// Unfinished is the number of rows whose status is not a completion.
	Unfinished int `json:"unfinished"`
	// Classification is the declared classification of the document.
	Classification string `json:"classification"`
}

// OriginalImport is the result of adapting and importing the original
// fixture.
type OriginalImport struct {
	// Board is the normalized source board.
	Board SourceBoard `json:"-"`
	// Inventory is the observed inventory of the original document.
	Inventory OriginalInventory `json:"inventory"`
	// Plan is the deterministic import plan.
	Plan Plan `json:"-"`
	// Receipt is the conservation receipt of the applied import.
	Receipt Receipt `json:"receipt"`
	// Reconciliations are the explicit per-identity reconciliation outcomes.
	Reconciliations []Reconciliation `json:"reconciliations"`
	// Unavailable records the historical information the original fixture
	// does NOT contain. It is reported explicitly: missing historical
	// information must remain explicitly unavailable.
	Unavailable []string `json:"unavailable"`
}

// ParseOriginalFixture parses the ORIGINAL fixture document.
//
// It is a strict single-document parse: the document must be one JSON
// object, and the fixture version must be the version this adapter
// understands.
func ParseOriginalFixture(data []byte) (SourceBoard, OriginalInventory, error) {
	var doc originalFixture
	dec := json.NewDecoder(strings.NewReader(string(data)))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&doc); err != nil {
		return SourceBoard{}, OriginalInventory{}, fmt.Errorf("decode the original fixture: %w", err)
	}
	if doc.FixtureVersion != OriginalFixtureVersion {
		return SourceBoard{}, OriginalInventory{}, fmt.Errorf(
			"unsupported original fixture version %d; this adapter understands version %d",
			doc.FixtureVersion, OriginalFixtureVersion)
	}
	board, inventory := adaptOriginal(doc)
	return board, inventory, nil
}

// adaptOriginal normalizes the original document into a SourceBoard and
// observes its inventory.
//
// The normalization is CONSERVATIVE:
//
//   - the exact source status is conserved verbatim, including the
//     UPPERCASE spelling of the original board;
//   - the original row text, the line locator and the free-form detail are
//     preserved;
//   - a priority that is not a canonical priority (the em dash of the
//     original board) is preserved verbatim in the source record and is
//     NOT coerced into a canonical priority;
//   - the parent epic identity is preserved: the workstreams do NOT become
//     eight independent epics;
//   - no dependency, no qualification outcome and no evidence authority is
//     inferred.
func adaptOriginal(doc originalFixture) (SourceBoard, OriginalInventory) {
	sourceName := doc.Source.Filename
	if sourceName == "" {
		sourceName = "original-iecr-board"
	}

	workstreams := map[string]bool{}
	epics := map[string]bool{}
	statusCounts := map[string]int{}
	unfinished := 0

	board := SourceBoard{
		Source: sourceName,
		Items:  make([]SourceItem, 0, len(doc.Items)),
	}
	for _, it := range doc.Items {
		workstreams[it.Workstream] = true
		epics[it.Epic] = true
		statusCounts[it.SourceStatus]++
		if IsUnfinishedSourceStatus(SourceStatus(it.SourceStatus)) {
			unfinished++
		}
		board.Items = append(board.Items, SourceItem{
			SourceID:         it.LocalID,
			EpicID:           it.Epic,
			Title:            it.Title,
			Status:           SourceStatus(it.SourceStatus),
			Priority:         it.SourcePriority,
			SourceRow:        it.SourceRow,
			SourceLine:       it.SourceLine,
			SourceDetail:     it.SourceDetail,
			SourceConfidence: domain.ProvenanceConversational,
		})
	}
	for _, ec := range doc.ExitCriteria {
		board.ExitCriteria = append(board.ExitCriteria, ec.Text)
	}

	inventory := OriginalInventory{
		WorkItems:        len(doc.Items),
		Workstreams:      len(workstreams),
		Epics:            len(epics),
		ExitCriteria:     len(doc.ExitCriteria),
		DistinctStatuses: len(statusCounts),
		StatusCounts:     statusCounts,
		Unfinished:       unfinished,
		Classification:   doc.Classification,
	}
	return board, inventory
}

// UnavailableHistoricalInformation lists the historical information the
// original fixture does NOT contain.
//
// It is reported explicitly so the operator never mistakes the imported
// board for the full historical repository board.
func UnavailableHistoricalInformation() []string {
	return []string{
		"the original fixture is not the full historical repository board",
		"no source dependency edges are present in the original fixture",
		"no evidence records or artifact identities are present in the original fixture",
		"no qualification outcomes are present in the original fixture",
		"the source board revision is a snapshot locator, not a verified repository state",
		"the source paste digest is recorded by the fixture, not verified by the importer",
	}
}

// ImportOriginalFixture adapts the ORIGINAL fixture, builds the plan and
// applies it to the store.
//
// The import FAILS CLOSED on unresolved load-bearing ambiguity: a
// quarantined identity group prevents the authoritative import.
func ImportOriginalFixture(s store.Store, data []byte, txID string) (OriginalImport, error) {
	board, inventory, err := ParseOriginalFixture(data)
	if err != nil {
		return OriginalImport{}, err
	}
	before, err := s.Load()
	if err != nil {
		return OriginalImport{}, err
	}
	plan := BuildPlan(board, before)
	if len(plan.Quarantined) > 0 {
		return OriginalImport{}, fmt.Errorf("%w: %d conflicting identity group(s) are quarantined; the import fails closed",
			ErrAmbiguousImport, len(plan.Quarantined))
	}
	if _, err := Apply(s, plan, txID); err != nil {
		return OriginalImport{}, err
	}
	return OriginalImport{
		Board:           board,
		Inventory:       inventory,
		Plan:            plan,
		Receipt:         BuildReceipt(before, plan),
		Reconciliations: plan.Reconciliations,
		Unavailable:     UnavailableHistoricalInformation(),
	}, nil
}

// SortInventoryCounts renders the status counts deterministically.
func (inv OriginalInventory) SortInventoryCounts() []string {
	out := make([]string, 0, len(inv.StatusCounts))
	for status := range inv.StatusCounts {
		out = append(out, status)
	}
	sort.Strings(out)
	return out
}
