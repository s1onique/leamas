package boardmd

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/s1onique/leamas/internal/ledger/importer"
)

// Extract reads the actual historical board and epic files of a
// repository and returns the deterministic normalized source board.
//
// The extraction NEVER mutates the source repository.
func Extract(repoRoot, repository, head string) (Extraction, error) {
	boardText, err := os.ReadFile(filepath.Join(repoRoot, BoardPath))
	if err != nil {
		return Extraction{}, fmt.Errorf("read the historical board %s: %w", BoardPath, err)
	}
	epicFiles, err := listMarkdownFiles(filepath.Join(repoRoot, EpicsDir))
	if err != nil {
		return Extraction{}, err
	}

	ex := Extraction{
		Source: Source{
			Repository: repository,
			Head:       head,
			BoardPath:  BoardPath,
			EpicsDir:   EpicsDir,
		},
		Sections: []Section{},
		Unavailable: []string{
			"the historical board carries no executable dependency graph between its work items",
			"the historical board carries no artifact digests for its evidence references",
			"the historical board carries no canonical acceptance authority for its historical claims",
			"the historical board reports evidence LOCATORS only; the evidence itself stays in the source repository",
		},
	}

	board := importer.SourceBoard{
		Source:           repository,
		EpicTitles:       map[string]string{},
		EpicExitCriteria: map[string][]string{},
	}

	// The board sections are the ACT REPORT archive. Each section carries
	// the ACT identity, the verdict and the entry head of one historical
	// ACT execution. They are NOT work items: a closed ACT report is not a
	// second current obligation.
	acts := []ACTExecution{}
	for _, sec := range splitBoardSections(string(boardText)) {
		ex.Sections = append(ex.Sections, sec.section)
		ex.Inventory.Sections++
		act, ok := actOf(sec)
		if ok {
			acts = append(acts, act)
			ex.Inventory.ACTs++
		}
		// A board section is an ACT REPORT, never a work item. The
		// board is a navigation index, not an archive of obligations:
		// the WORK_ITEM field names the work item the report
		// CONCERNS. Deriving a work item from it would create one
		// obligation per historical report.
	}

	// The current frontier is the authoritative CURRENT state of the board.
	// Every lane is a current obligation: a lane that is not closed names
	// work that is still to be done.
	//
	// The epic scope of a lane is resolved from the epic FILES: the lane
	// names its epic by the epic FILE, and the epic identity is that
	// file's heading.
	epicScopes := map[string]string{}
	for _, name := range epicFiles {
		rel := EpicsDir + "/" + name
		text, err := os.ReadFile(filepath.Join(repoRoot, EpicsDir, name))
		if err != nil {
			return Extraction{}, fmt.Errorf("read the epic file %s: %w", rel, err)
		}
		epic := parseEpicFile(rel, string(text))
		if epic == nil {
			continue
		}
		epicScopes[epic.workstream] = epic.id
	}
	for _, lane := range frontierLanes(string(boardText)) {
		epic := epicScopeOf(lane.epic, epicScopes)
		item := importer.SourceItem{
			SourceID:   laneIdentity(lane),
			EpicID:     epic,
			Workstream: lane.lane,
			Title:      laneTitle(lane),
			// The lane state is conserved VERBATIM. A state the
			// importer does not understand stays unresolved and
			// non-executable.
			Status:           importer.SourceStatus(strings.ToUpper(lane.state)),
			Priority:         lane.pri,
			SourceLine:       lane.line,
			SourceRow:        lane.lane,
			SourceDetail:     laneTitle(lane),
			SourceConfidence: "imported",
		}
		board.Items = append(board.Items, item)
		ex.Inventory.WorkItems++
		if importer.IsUnfinishedSourceStatus(item.Status) {
			ex.Inventory.Unfinished++
		}
		if !importer.IsKnownSourceStatus(item.Status) {
			ex.Inventory.Unresolved++
		}
	}

	// The epic files carry the epic scopes, their work items and their
	// ACT ledgers. They are the authoritative source of the OUTSTANDING
	// engineering obligations.
	for _, name := range epicFiles {
		rel := EpicsDir + "/" + name
		text, err := os.ReadFile(filepath.Join(repoRoot, EpicsDir, name))
		if err != nil {
			return Extraction{}, fmt.Errorf("read the epic file %s: %w", rel, err)
		}
		epic := parseEpicFile(rel, string(text))
		if epic == nil {
			continue
		}
		ex.Sections = append(ex.Sections, epic.section)
		ex.Inventory.Sections++
		ex.Inventory.Epics++
		board.EpicTitles[epic.id] = epic.title
		if len(epic.exitCriteria) > 0 {
			board.EpicExitCriteria[epic.id] = epic.exitCriteria
			ex.Inventory.ExitCriteria += len(epic.exitCriteria)
		}
		for _, item := range epic.workItems {
			board.Items = append(board.Items, item)
			ex.Inventory.WorkItems++
			if importer.IsUnfinishedSourceStatus(item.Status) {
				ex.Inventory.Unfinished++
			}
			if !importer.IsKnownSourceStatus(item.Status) {
				ex.Inventory.Unresolved++
			}
		}
		for _, row := range epic.actRows {
			acts = append(acts, ACTExecution{
				ActID:    row.id,
				Verdict:  row.verdict,
				Head:     row.head,
				EpicID:   epic.id,
				Locator:  epic.section.Locator,
				SourceLn: row.line,
			})
		}
	}

	ex.Board = board
	ex.ACTs = dedupeACTs(acts)
	ex.Inventory.DuplicateSections = countDuplicateSections(board.Items)
	return ex, nil
}

// frontierLanes returns the current-frontier lanes of the board.
//
// The current frontier is the AUTHORITATIVE current state of the board:
// it names every workstream scope, its state and its frontier ACT.
//
// A lane is a CURRENT OBLIGATION: a lane that is not closed names work
// that is still to be done.
func frontierLanes(boardText string) []frontierLane {
	return extractFrontierLanes(strings.Split(boardText, "\n"))
}

// ACTExecution is one extracted ACT execution record.
//
// It is the historical ACT report of the source: the identity, the
// verdict, the source head and the section locator it came from.
type ACTExecution struct {
	// ActID is the ACT identity.
	ActID string `json:"act_id"`
	// Verdict is the EXACT source verdict.
	Verdict string `json:"verdict"`
	// Head is the source head the ACT was executed against.
	Head string `json:"head,omitempty"`
	// EpicID is the epic scope the ACT belongs to.
	EpicID string `json:"epic_id,omitempty"`
	// Locator is the stable section locator of the source record.
	Locator string `json:"locator"`
	// SourceLn is the source line of the record.
	SourceLn int `json:"source_line,omitempty"`
}

// actOf converts a board section into an ACT execution record.
//
// A section without an ACT identity is not an ACT execution record: it is
// narrative. It is counted as a section and never invented as an ACT.
func actOf(sec boardSection) (ACTExecution, bool) {
	id := sec.fields["ACT_ID"]
	if id == "" {
		return ACTExecution{}, false
	}
	return ACTExecution{
		ActID:    id,
		Verdict:  sec.fields["VERDICT"],
		Head:     sec.fields["ENTRY_HEAD"],
		EpicID:   sec.fields["EPIC"],
		Locator:  sec.section.Locator,
		SourceLn: sec.section.StartLine,
	}, true
}

// boardWorkItem converts a board section into a work-item record.
//
// The board section names the work item it concerns: the WORK_ITEM field
// carries the work-item identity and the EPIC field carries the scope.
//
// A section without a work item is an ACT report about the epic scope
// itself: it is not a separate work-item obligation.
func boardWorkItem(sec boardSection) (importer.SourceItem, bool) {
	workItem := sec.fields["WORK_ITEM"]
	if workItem == "" {
		return importer.SourceItem{}, false
	}
	identity, title := splitWorkItem(workItem)
	if identity == "" {
		return importer.SourceItem{}, false
	}
	epic := sec.fields["EPIC"]
	if epic == "" {
		epic = identity
	}
	return importer.SourceItem{
		SourceID:   identity,
		EpicID:     epic,
		Workstream: epic,
		Title:      title,
		// The verdict is conserved VERBATIM: an unrecognised verdict
		// stays unresolved and non-executable.
		Status:           importer.SourceStatus(sec.fields["VERDICT"]),
		SourceLine:       sec.section.StartLine,
		SourceRow:        identity,
		SourceConfidence: "imported",
	}, true
}

// splitWorkItem splits a work-item field into its identity and title.
//
// The field carries both: "OBS-05 — Active execution and waiting-time
// accounting".
//
// The separator is the EM DASH. It is a multi-byte rune: the split must
// advance by the FULL rune width, never by one byte, or the title would
// start with a broken rune.
func splitWorkItem(workItem string) (string, string) {
	const separator = "—"
	if idx := strings.Index(workItem, separator); idx > 0 {
		return strings.TrimSpace(workItem[:idx]),
			strings.TrimSpace(workItem[idx+len(separator):])
	}
	return strings.TrimSpace(workItem), strings.TrimSpace(workItem)
}

// dedupeACTs deduplicates the extracted ACT execution records.
//
// The same ACT may be reported by the board and by an epic ledger. The
// duplicate representation is NOT a second execution: it is counted and
// reported, never imported twice.
func dedupeACTs(acts []ACTExecution) []ACTExecution {
	seen := map[string]bool{}
	var out []ACTExecution
	for _, act := range acts {
		if act.ActID == "" {
			continue
		}
		if seen[act.ActID] {
			continue
		}
		seen[act.ActID] = true
		out = append(out, act)
	}
	return out
}

// listMarkdownFiles lists the markdown files of a directory, sorted.
func listMarkdownFiles(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read the epic directory %s: %w", EpicsDir, err)
	}
	var out []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}
		out = append(out, entry.Name())
	}
	sort.Strings(out)
	return out, nil
}

// countDistinctACTs counts the distinct work-item identities.
func countDistinctACTs(items []importer.SourceItem) int {
	seen := map[string]bool{}
	for _, item := range items {
		seen[item.SourceID] = true
	}
	return len(seen)
}

// countDuplicateSections counts the source records that repeat an
// identity that was already observed.
//
// A duplicate historical representation is NOT a second current
// obligation: it is counted, reported and never imported twice.
func countDuplicateSections(items []importer.SourceItem) int {
	seen := map[string]bool{}
	dupes := 0
	for _, item := range items {
		key := item.EpicID + "/" + item.SourceID
		if seen[key] {
			dupes++
			continue
		}
		seen[key] = true
	}
	return dupes
}
