package domain

import (
	"fmt"
	"sort"
)

// ReadyItem is one ready-work candidate.
type ReadyItem struct {
	ScopedID         ScopedItemID `json:"id"`
	Title            string       `json:"title"`
	Kind             ItemKind     `json:"kind"`
	Priority         Priority     `json:"priority,omitempty"`
	EpicID           EpicID       `json:"epic_id"`
	Owner            string       `json:"owner,omitempty"`
	RequiresLiveGate bool         `json:"requires_live_gate,omitempty"`
}

// ReadyOptions bounds the ready-work selection.
type ReadyOptions struct {
	// Epic restricts the selection to a single epic scope. It is a
	// selection control, never an authorization.
	Epic EpicID
	// Owner restricts the selection to one owner. It is a selection
	// control, never an authorization.
	Owner string
	// IncludeLiveGate includes items that still need a live qualification
	// run. It is off by default: an item whose acceptance requires a live
	// gate is not ready until that gate produced verified evidence.
	IncludeLiveGate bool
	// Limit bounds the number of returned items. Zero means no bound.
	Limit int
}

// SelectReady returns the deterministic ready-work selection.
//
// Readiness is an EXECUTION readiness gate. It answers "may this work be
// started?", not "is this work accepted?" and never "is this work
// authorized to execute?".
//
// The rules are:
//
//   - the item is not archived and its lifecycle is not blocked or closed;
//   - the item's acceptance is not already terminal;
//   - the item is not explicitly deferred;
//   - every blocking dependency is resolved AND accepted.
//
// A work item with unmet acceptance criteria IS ready: it is precisely the
// work that still has to be done, and it must be able to acquire its own
// evidence. Excluding it would make the obligation unreachable.
//
// A live qualification obligation is executable before the live
// qualification evidence exists: running the qualification is the work.
//
// The epic and owner filters are selection controls. They narrow the result
// set; they never authorize execution.
func SelectReady(l Ledger, opts ReadyOptions) []ReadyItem {
	candidates := make([]WorkItem, 0, len(l.Items))
	for _, it := range l.Items {
		if it.Archived {
			continue
		}
		if opts.Epic != "" && it.EpicID != opts.Epic {
			continue
		}
		if opts.Owner != "" && it.Owner != opts.Owner {
			continue
		}
		if !IsReady(l, it, opts.IncludeLiveGate) {
			continue
		}
		candidates = append(candidates, it)
	}

	SortItems(candidates)
	out := make([]ReadyItem, 0, len(candidates))
	for _, it := range candidates {
		out = append(out, ReadyItem{
			ScopedID:         it.Scoped(),
			Title:            it.Title,
			Kind:             it.Kind,
			Priority:         it.Priority,
			EpicID:           it.EpicID,
			Owner:            it.Owner,
			RequiresLiveGate: it.RequiresLiveGate,
		})
	}
	if opts.Limit > 0 && len(out) > opts.Limit {
		out = out[:opts.Limit]
	}
	return out
}

// CurrentlyAuthorized reports whether the item's acceptance is a CURRENTLY
// VALID authority that may satisfy a downstream prerequisite.
//
// It is the acceptance authority (EvaluateAcceptance) applied to the
// downstream-prerequisite question. An accepted item whose evidence was
// invalidated keeps its historical acceptance record, but it no longer
// satisfies a current prerequisite: the invalidation is therefore never
// silent.
//
// An item that is not accepted never authorizes a downstream prerequisite.
func CurrentlyAuthorized(l Ledger, it WorkItem) bool {
	if it.Acceptance != AcceptanceAccepted {
		return false
	}
	return EvaluateAcceptance(l, it).Admissible
}

// IsReady reports whether a work item is ready to be worked on.
//
// See SelectReady for the full contract.
func IsReady(l Ledger, it WorkItem, includeLiveGate bool) bool {
	if it.Archived {
		return false
	}
	if it.Lifecycle == LifecycleBlocked || it.Lifecycle == LifecycleClosed {
		return false
	}
	if IsTerminalAcceptance(it.Acceptance) {
		return false
	}
	// Deferred work is explicitly not selected: the deferral decision is
	// an exclusion, not a priority.
	if it.Acceptance == AcceptanceDeferred {
		return false
	}
	for _, dep := range it.Dependencies {
		if !dep.Kind.IsBlocking() {
			continue
		}
		other, ok := l.ItemByScoped(dep.Ref)
		if !ok {
			// An unresolved blocking dependency keeps the item
			// non-executable. It is a durable blocker, not a satisfied
			// precondition.
			return false
		}
		if !CurrentlyAuthorized(l, other) {
			return false
		}
	}
	return true
}

// LedgerStats is the deterministic statistics projection.
type LedgerStats struct {
	SchemaVersion int            `json:"schema_version"`
	Epics         int            `json:"epics"`
	Items         int            `json:"items"`
	LiveItems     int            `json:"live_items"`
	ArchivedItems int            `json:"archived_items"`
	Acts          int            `json:"acts"`
	Evidence      int            `json:"evidence"`
	Decisions     int            `json:"decisions"`
	Knowledge     int            `json:"knowledge"`
	ByLifecycle   map[string]int `json:"by_lifecycle"`
	ByAcceptance  map[string]int `json:"by_acceptance"`
	ByQuality     map[string]int `json:"by_evidence_quality"`
	Workstreams   []string       `json:"workstreams"`
	Ready         int            `json:"ready"`
	Blocked       int            `json:"blocked"`
	Unfinished    int            `json:"unfinished"`
}

// Stats computes the deterministic ledger statistics.
func Stats(l Ledger) LedgerStats {
	s := LedgerStats{
		SchemaVersion: LedgerSchemaVersion,
		Epics:         len(l.Epics),
		Items:         len(l.Items),
		Acts:          len(l.Acts),
		Evidence:      len(l.Evidence),
		Decisions:     len(l.Decisions),
		Knowledge:     len(l.Knowledge),
		ByLifecycle:   map[string]int{},
		ByAcceptance:  map[string]int{},
		ByQuality:     map[string]int{},
	}

	workstreams := map[EpicID]bool{}
	for _, it := range l.Items {
		workstreams[it.EpicID] = true
		if it.Archived {
			s.ArchivedItems++
		} else {
			s.LiveItems++
		}
		s.ByLifecycle[string(it.Lifecycle)]++
		s.ByAcceptance[string(it.Acceptance)]++
		if it.Quality != "" {
			s.ByQuality[string(it.Quality)]++
		}
		if !IsTerminalAcceptance(it.Acceptance) {
			s.Unfinished++
		}
		if len(BlockersFor(l, it)) > 0 {
			s.Blocked++
		}
	}
	for e := range workstreams {
		s.Workstreams = append(s.Workstreams, string(e))
	}
	sort.Strings(s.Workstreams)
	s.Ready = len(SelectReady(l, ReadyOptions{}))

	return s
}

// ExitCriteriaSummary counts the epic-scoped exit criteria per epic.
func ExitCriteriaSummary(l Ledger) map[string]int {
	out := map[string]int{}
	for _, it := range l.Items {
		if it.Archived {
			continue
		}
		out[string(it.EpicID)] += len(it.ExitCriteria)
	}
	return out
}

// Describe renders a one-line summary of a work item.
func Describe(it WorkItem) string {
	return fmt.Sprintf("%s %s [%s/%s] %s",
		it.Scoped().String(), it.Kind, it.Lifecycle, it.Acceptance, it.Title)
}
