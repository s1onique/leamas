// Package render produces the deterministic derived views of the Factory
// Work Ledger.
//
// The rendered views are projections: identical canonical input produces
// byte-identical output. The only value that is not derived from the
// canonical state is the presentation timestamp, which is emitted in its
// own explicitly separated field.
package render

import (
	"fmt"
	"strings"

	"github.com/s1onique/leamas/internal/ledger/domain"
)

// Board renders the compact operational board.
func Board(l domain.Ledger) string {
	var b strings.Builder
	b.WriteString("# Factory Work Ledger\n\n")
	for _, epic := range domain.BuildBoard(l).Epics {
		fmt.Fprintf(&b, "## %s — %s\n\n", epic.EpicID, epic.Title)
		if len(epic.Items) == 0 {
			b.WriteString("_no work items_\n\n")
			continue
		}
		for _, item := range epic.Items {
			fmt.Fprintf(&b, "- `%s` [%s] %s — %s\n",
				item.ScopedID, item.Kind, item.Title, item.Lifecycle)
			for _, blocker := range item.Blockers {
				fmt.Fprintf(&b, "  - blocked: %s\n", blocker)
			}
		}
		b.WriteString("\n")
	}
	return b.String()
}

// EpicView renders the epic-scoped view.
func EpicView(l domain.Ledger, epic domain.EpicID) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# Epic %s\n\n", epic)
	items := []domain.WorkItem{}
	for _, it := range l.Items {
		if it.EpicID == epic {
			items = append(items, it)
		}
	}
	domain.SortItems(items)
	for _, it := range items {
		fmt.Fprintf(&b, "## `%s` %s\n\n", it.Scoped(), it.Title)
		fmt.Fprintf(&b, "- kind: %s\n", it.Kind)
		fmt.Fprintf(&b, "- lifecycle: %s\n", it.Lifecycle)
		fmt.Fprintf(&b, "- acceptance: %s\n", it.Acceptance)
		fmt.Fprintf(&b, "- priority: %s\n", it.Priority)
		fmt.Fprintf(&b, "- evidence quality: %s\n", it.Quality)
		if it.Owner != "" {
			fmt.Fprintf(&b, "- owner: %s\n", it.Owner)
		}
		for _, c := range it.AcceptanceCriteria {
			fmt.Fprintf(&b, "- criterion: %s\n", c)
		}
		for _, e := range it.ExitCriteria {
			fmt.Fprintf(&b, "- exit criterion: %s\n", e)
		}
		for _, d := range it.Dependencies {
			fmt.Fprintf(&b, "- depends (%s): %s\n", d.Kind, d.Ref)
		}
		for _, e := range it.Evidence {
			fmt.Fprintf(&b, "- evidence (%s): %s\n", e.Quality, e.EvidenceID)
		}
		for _, s := range it.Successors {
			fmt.Fprintf(&b, "- successor: %s\n", s)
		}
		b.WriteString("\n")
	}
	return b.String()
}

// ReadyView renders the ready-work view.
func ReadyView(l domain.Ledger, opts domain.ReadyOptions) string {
	var b strings.Builder
	b.WriteString("# Ready Work\n\n")
	ready := domain.SelectReady(l, opts)
	if len(ready) == 0 {
		b.WriteString("_no ready work_\n")
		return b.String()
	}
	for _, item := range ready {
		fmt.Fprintf(&b, "- `%s` [%s] %s\n", item.ScopedID, item.Kind, item.Title)
	}
	return b.String()
}

// BlockersView renders the blocker explanations.
func BlockersView(l domain.Ledger) string {
	var b strings.Builder
	b.WriteString("# Blockers\n\n")
	items := []domain.WorkItem{}
	for _, it := range l.Items {
		if it.Archived {
			continue
		}
		items = append(items, it)
	}
	domain.SortItems(items)
	for _, it := range items {
		blockers := domain.BlockersFor(l, it)
		if len(blockers) == 0 {
			continue
		}
		fmt.Fprintf(&b, "## `%s` %s\n\n", it.Scoped(), it.Title)
		for _, blocker := range blockers {
			fmt.Fprintf(&b, "- %s\n", blocker)
		}
		b.WriteString("\n")
	}
	return b.String()
}

// HistoryView renders the historical lookup of terminal ACT verdicts.
func HistoryView(l domain.Ledger) string {
	var b strings.Builder
	b.WriteString("# History\n\n")
	acts := append([]domain.ActExecution(nil), l.Acts...)
	domain.SortActs(acts)
	for _, a := range acts {
		fmt.Fprintf(&b, "- `%s` %s — %s\n", a.ActID, a.Verdict, a.TerminalDisposition)
	}
	return b.String()
}

// StatsView renders the ledger statistics.
func StatsView(l domain.Ledger) string {
	stats := domain.Stats(l)
	var b strings.Builder
	b.WriteString("# Ledger Statistics\n\n")
	fmt.Fprintf(&b, "- epics: %d\n", stats.Epics)
	fmt.Fprintf(&b, "- work items: %d\n", stats.Items)
	fmt.Fprintf(&b, "- live work items: %d\n", stats.LiveItems)
	fmt.Fprintf(&b, "- archived work items: %d\n", stats.ArchivedItems)
	fmt.Fprintf(&b, "- act records: %d\n", stats.Acts)
	fmt.Fprintf(&b, "- evidence references: %d\n", stats.Evidence)
	fmt.Fprintf(&b, "- unfinished obligations: %d\n", stats.Unfinished)
	fmt.Fprintf(&b, "- ready work items: %d\n", stats.Ready)
	fmt.Fprintf(&b, "- blocked work items: %d\n", stats.Blocked)
	fmt.Fprintf(&b, "- workstreams: %d\n", len(stats.Workstreams))
	return b.String()
}
