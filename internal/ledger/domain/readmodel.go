package domain

import (
	"fmt"
	"sort"
)

// Board is the compact operational projection of the ledger.
//
// The board is a projection, never a second source of truth.
type Board struct {
	SchemaVersion int         `json:"schema_version"`
	Epics         []BoardEpic `json:"epics"`
	GeneratedNote string      `json:"generated_note,omitempty"`
}

// BoardEpic is the per-epic section of the board.
type BoardEpic struct {
	EpicID   EpicID      `json:"epic_id"`
	Title    string      `json:"title"`
	Priority Priority    `json:"priority,omitempty"`
	Archived bool        `json:"archived,omitempty"`
	Items    []BoardItem `json:"items"`
	Stats    BoardStats  `json:"stats"`
}

// BoardItem is the per-item row of the board.
type BoardItem struct {
	ScopedID   ScopedItemID    `json:"id"`
	Title      string          `json:"title"`
	Kind       ItemKind        `json:"kind"`
	Lifecycle  LifecycleState  `json:"lifecycle"`
	Acceptance AcceptanceState `json:"acceptance"`
	Priority   Priority        `json:"priority,omitempty"`
	Quality    EvidenceQuality `json:"evidence_quality,omitempty"`
	Owner      string          `json:"owner,omitempty"`
	Blockers   []string        `json:"blockers,omitempty"`
	Evidence   int             `json:"evidence_count"`
	// SourceLifecycle is the NORMALIZED lifecycle the source asserted for
	// the record.
	//
	// It keeps an UNRESOLVED source lifecycle distinguishable from an
	// intentional DEFERRED one in the operational projection: the
	// acceptance state of an unknown source status is DEFERRED, and the
	// source lifecycle is what says the deferral was not a decision.
	SourceLifecycle string `json:"source_lifecycle,omitempty"`
}

// BoardStats counts the lifecycle distribution of a scope.
type BoardStats struct {
	Total      int `json:"total"`
	Open       int `json:"open"`
	InProgress int `json:"in_progress"`
	Blocked    int `json:"blocked"`
	Closed     int `json:"closed"`
	Archived   int `json:"archived"`
	Accepted   int `json:"accepted"`
}

// BuildBoard renders the compact operational board.
//
// Archived records are excluded from the operational board but remain
// discoverable through the list/show commands.
func BuildBoard(l Ledger) Board {
	board := Board{SchemaVersion: LedgerSchemaVersion}

	live := make([]WorkItem, 0, len(l.Items))
	for _, it := range l.Items {
		if !it.Archived {
			live = append(live, it)
		}
	}
	SortItems(live)

	byEpic := map[EpicID][]WorkItem{}
	epicOrder := []EpicID{}
	epicTitles := map[EpicID]string{}
	epicPriorities := map[EpicID]Priority{}
	epicArchived := map[EpicID]bool{}

	for _, e := range l.Epics {
		epicTitles[e.EpicID] = e.Title
		epicPriorities[e.EpicID] = e.Priority
		epicArchived[e.EpicID] = e.Archived
	}
	for _, it := range live {
		if _, ok := byEpic[it.EpicID]; !ok {
			epicOrder = append(epicOrder, it.EpicID)
		}
		byEpic[it.EpicID] = append(byEpic[it.EpicID], it)
	}
	sort.Slice(epicOrder, func(i, j int) bool { return epicOrder[i] < epicOrder[j] })

	for _, epicID := range epicOrder {
		items := byEpic[epicID]
		section := BoardEpic{
			EpicID:   epicID,
			Title:    epicTitles[epicID],
			Priority: epicPriorities[epicID],
			Archived: epicArchived[epicID],
		}
		for _, it := range items {
			section.Items = append(section.Items, BoardItem{
				ScopedID:   it.Scoped(),
				Title:      it.Title,
				Kind:       it.Kind,
				Lifecycle:  it.Lifecycle,
				Acceptance: it.Acceptance,
				Priority:   it.Priority,
				Quality:    it.Quality,
				Owner:      it.Owner,
				Blockers:   BlockersFor(l, it),
				Evidence:   len(it.Evidence),
				// The NORMALIZED source lifecycle keeps an unresolved
				// source status distinguishable from an intentional
				// deferral in the operational projection.
				SourceLifecycle: it.SourceLifecycle,
			})
		}
		section.Stats = statsFor(items)
		board.Epics = append(board.Epics, section)
	}

	return board
}

func statsFor(items []WorkItem) BoardStats {
	var s BoardStats
	for _, it := range items {
		s.Total++
		switch it.Lifecycle {
		case LifecycleOpen:
			s.Open++
		case LifecycleInProgress:
			s.InProgress++
		case LifecycleBlocked:
			s.Blocked++
		case LifecycleClosed:
			s.Closed++
		case LifecycleArchived:
			s.Archived++
		}
		if it.Acceptance == AcceptanceAccepted {
			s.Accepted++
		}
	}
	return s
}

// BlockersFor returns the human-readable blocking reasons of a work item.
//
// A blocker is a blocking dependency that is not yet satisfied, or a
// declared acceptance criterion that has no supporting evidence.
//
// The dependency check consults the SAME acceptance authority as the
// canonical validator and the ready-work projection (CurrentlyAuthorized).
// An accepted dependency whose evidence was invalidated therefore no longer
// satisfies the prerequisite: the blocker report and the readiness
// projection can never disagree.
func BlockersFor(l Ledger, it WorkItem) []string {
	var blockers []string
	// The EXPLICIT blockers the source declared are conserved verbatim and
	// reported here: a blocker the source stated is visible in the
	// operational projection, exactly like an unresolved dependency.
	for _, blocker := range it.SourceBlockers {
		blockers = append(blockers, blocker)
	}
	for _, dep := range it.Dependencies {
		if !dep.Kind.IsBlocking() {
			continue
		}
		other, ok := l.ItemByScoped(dep.Ref)
		if !ok {
			blockers = append(blockers, fmt.Sprintf("unresolved dependency %s", dep.Ref.String()))
			continue
		}
		if !CurrentlyAuthorized(l, other) {
			blockers = append(blockers, fmt.Sprintf(
				"dependency %s is %s", other.Scoped().String(), string(other.Acceptance)))
		}
	}
	// Unmet acceptance criteria with no evidence is NORMAL work, not a
	// blocker: the item must be able to acquire its own evidence. It is
	// therefore not reported as a blocker.
	//
	// A live qualification obligation is likewise executable before the
	// live qualification evidence exists. It is reported as a blocker
	// only when the item has no evidence at all AND requires a live
	// gate, because in that case the qualification has not been run.
	if it.RequiresLiveGate && it.Quality == EvidenceQualityNone {
		blockers = append(blockers, "live qualification has not produced evidence yet")
	}
	sort.Strings(blockers)
	return blockers
}
