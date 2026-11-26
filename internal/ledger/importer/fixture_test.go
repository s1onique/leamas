package importer

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// FixtureInventory describes the expected conservation counts of the
// representative IECR intake fixture.
type FixtureInventory struct {
	WorkItems          int
	Workstreams        int
	Closed             int
	ClosedInsufficient int
	Unfinished         int
	Blocked            int
	ExitCriteria       int
}

// ExpectedIECRInventory is the declared fixture inventory.
var ExpectedIECRInventory = FixtureInventory{
	WorkItems:          44,
	Workstreams:        8,
	Closed:             10,
	ClosedInsufficient: 1,
	Unfinished:         33,
	Blocked:            6,
	ExitCriteria:       12,
}

// iecrSourceBoard builds the representative conversational intake fixture.
//
// The fixture is conversational intake, NOT authoritative repository
// evidence. Its distribution matches the declared inventory:
//
//	44 unique work items across 8 workstreams
//	10 CLOSED records
//	1 CLOSED_INSUFFICIENT record
//	33 unfinished records
//	6 BLOCKED records
//	12 exit criteria
func iecrSourceBoard() SourceBoard {
	board := SourceBoard{
		Source: "iecr-intake",
		EpicTitles: map[string]string{
			"WS-ALPHA": "Alpha workstream",
			"WS-BRAVO": "Bravo workstream",
			"WS-CHARL": "Charlie workstream",
			"WS-DELTA": "Delta workstream",
			"WS-ECHO":  "Echo workstream",
			"WS-FOX":   "Fox workstream",
			"WS-GOLF":  "Golf workstream",
			"WS-HOTEL": "Hotel workstream",
		},
	}

	workstreams := []string{
		"WS-ALPHA", "WS-BRAVO", "WS-CHARL", "WS-DELTA",
		"WS-ECHO", "WS-FOX", "WS-GOLF", "WS-HOTEL",
	}

	// 44 work items distributed round-robin over the 8 workstreams.
	for i := 1; i <= 44; i++ {
		ws := workstreams[(i-1)%len(workstreams)]
		item := SourceItem{
			SourceID:         fmt.Sprintf("W-%02d", i),
			EpicID:           ws,
			Title:            fmt.Sprintf("Intake item %02d", i),
			Status:           SourceStatusUnfinished,
			Priority:         "P2",
			SourceConfidence: "conversational",
		}
		board.Items = append(board.Items, item)
	}

	// 10 CLOSED records.
	for i := 1; i <= 10; i++ {
		board.Items[i-1].Status = SourceStatusClosed
	}
	// 1 CLOSED_INSUFFICIENT record.
	board.Items[10].Status = SourceStatusClosedInsufficient
	// 6 BLOCKED records. A blocked record is still unfinished: it is a
	// subset of the 33 unfinished records, not an additional class.
	for i := 11; i <= 16; i++ {
		board.Items[i].Status = SourceStatusBlocked
	}

	// 12 exit criteria spread over the first twelve records.
	for i := 0; i < 12; i++ {
		board.Items[i].ExitCriteria = []string{
			fmt.Sprintf("exit criterion %02d", i+1),
		}
	}

	return board
}

// TestIECRFixtureInventoryConservation proves that the fixture carries the
// declared inventory before any import happens.
func TestIECRFixtureInventoryConservation(t *testing.T) {
	board := iecrSourceBoard()

	if got := len(board.Items); got != ExpectedIECRInventory.WorkItems {
		t.Fatalf("expected %d work items, got %d", ExpectedIECRInventory.WorkItems, got)
	}

	workstreams := map[string]bool{}
	closed, closedInsuf, unfinished, blocked, exits := 0, 0, 0, 0, 0
	ids := map[string]bool{}
	for _, item := range board.Items {
		if ids[item.SourceID] {
			t.Fatalf("duplicate source identity %q", item.SourceID)
		}
		ids[item.SourceID] = true
		workstreams[item.EpicID] = true
		switch item.Status {
		case SourceStatusClosed:
			closed++
		case SourceStatusClosedInsufficient:
			closedInsuf++
		case SourceStatusBlocked:
			blocked++
			unfinished++
		default:
			unfinished++
		}
		exits += len(item.ExitCriteria)
	}

	if len(workstreams) != ExpectedIECRInventory.Workstreams {
		t.Fatalf("expected %d workstreams, got %d", ExpectedIECRInventory.Workstreams, len(workstreams))
	}
	if closed != ExpectedIECRInventory.Closed {
		t.Fatalf("expected %d closed, got %d", ExpectedIECRInventory.Closed, closed)
	}
	if closedInsuf != ExpectedIECRInventory.ClosedInsufficient {
		t.Fatalf("expected %d closed_insufficient, got %d",
			ExpectedIECRInventory.ClosedInsufficient, closedInsuf)
	}
	if blocked != ExpectedIECRInventory.Blocked {
		t.Fatalf("expected %d blocked, got %d", ExpectedIECRInventory.Blocked, blocked)
	}
	if unfinished != ExpectedIECRInventory.Unfinished {
		t.Fatalf("expected %d unfinished, got %d", ExpectedIECRInventory.Unfinished, unfinished)
	}
	if exits != ExpectedIECRInventory.ExitCriteria {
		t.Fatalf("expected %d exit criteria, got %d", ExpectedIECRInventory.ExitCriteria, exits)
	}
}

// TestIECRFixtureIsWrittenToTestdata proves the fixture is committed as
// repository-owned test data, not only generated in memory.
//
// The fixture is written with one record per line so the committed test
// data stays inside the LLM-friendliness line and length budgets.
func TestIECRFixtureIsWrittenToTestdata(t *testing.T) {
	board := iecrSourceBoard()

	var b strings.Builder
	b.WriteString("{\n")
	source, err := json.Marshal(board.Source)
	if err != nil {
		t.Fatalf("marshal source: %v", err)
	}
	fmt.Fprintf(&b, "  \"source\": %s,\n", source)
	b.WriteString("  \"epic_titles\": {\n")
	names := make([]string, 0, len(board.EpicTitles))
	for name := range board.EpicTitles {
		names = append(names, name)
	}
	sort.Strings(names)
	for i, name := range names {
		title, err := json.Marshal(board.EpicTitles[name])
		if err != nil {
			t.Fatalf("marshal title: %v", err)
		}
		sep := ","
		if i == len(names)-1 {
			sep = ""
		}
		fmt.Fprintf(&b, "    %q: %s%s\n", name, title, sep)
	}
	b.WriteString("  },\n")
	b.WriteString("  \"items\": [\n")
	for i, item := range board.Items {
		line, err := json.Marshal(item)
		if err != nil {
			t.Fatalf("marshal item: %v", err)
		}
		sep := ","
		if i == len(board.Items)-1 {
			sep = ""
		}
		fmt.Fprintf(&b, "    %s%s\n", line, sep)
	}
	b.WriteString("  ]\n")
	b.WriteString("}\n")

	path := filepath.Join("testdata", "iecr-intake-board.json")
	if err := os.WriteFile(path, []byte(b.String()), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("fixture missing: %v", err)
	}
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	lines := strings.Count(string(content), "\n")
	if lines > 400 {
		t.Fatalf("committed fixture exceeds the 400-line budget: %d", lines)
	}
	for i, line := range strings.Split(string(content), "\n") {
		if len(line) > 240 {
			t.Fatalf("committed fixture line %d exceeds the 240-char budget: %d", i+1, len(line))
		}
	}
}
