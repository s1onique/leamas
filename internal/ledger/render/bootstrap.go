package render

import (
	"fmt"
	"strings"

	"github.com/s1onique/leamas/internal/ledger/domain"
)

// BootstrapOptions bounds the minimal agent bootstrap context.
type BootstrapOptions struct {
	// Epic restricts the context to one epic scope.
	Epic domain.EpicID
	// MaxItems bounds the number of work items rendered. Zero means no
	// explicit bound; the caller applies the token budget instead.
	MaxItems int
	// MaxTokens is the token budget. Required obligations are never
	// silently truncated to meet it: the render reports the overflow
	// instead of hiding it.
	MaxTokens int
}

// Bootstrap renders the minimal agent bootstrap context.
//
// The bootstrap is a projection of the canonical state. It never
// truncates a required obligation: when the budget is exceeded the
// render says so explicitly and names the omitted items.
func Bootstrap(l domain.Ledger, opts BootstrapOptions) string {
	var b strings.Builder
	b.WriteString("# Ledger Bootstrap\n\n")

	items := []domain.WorkItem{}
	for _, it := range l.Items {
		if it.Archived {
			continue
		}
		if opts.Epic != "" && it.EpicID != opts.Epic {
			continue
		}
		items = append(items, it)
	}
	domain.SortItems(items)

	if opts.MaxItems > 0 && len(items) > opts.MaxItems {
		omitted := items[opts.MaxItems:]
		items = items[:opts.MaxItems]
		fmt.Fprintf(&b, "> %d work items omitted to respect the item budget.\n\n", len(omitted))
		for _, it := range omitted {
			fmt.Fprintf(&b, "> omitted: `%s` %s\n", it.Scoped(), it.Title)
		}
		b.WriteString("\n")
	}

	for _, it := range items {
		fmt.Fprintf(&b, "## `%s` %s\n\n", it.Scoped(), it.Title)
		fmt.Fprintf(&b, "- %s / %s / %s\n", it.Kind, it.Lifecycle, it.Acceptance)
		if it.Owner != "" {
			fmt.Fprintf(&b, "- owner: %s\n", it.Owner)
		}
		for _, c := range it.AcceptanceCriteria {
			fmt.Fprintf(&b, "- criterion: %s\n", c)
		}
		blockers := domain.BlockersFor(l, it)
		for _, blocker := range blockers {
			fmt.Fprintf(&b, "- blocked: %s\n", blocker)
		}
		b.WriteString("\n")
	}

	out := b.String()
	if opts.MaxTokens > 0 && EstimateTokens(out) > opts.MaxTokens {
		out += fmt.Sprintf("\n> bootstrap exceeds the %d-token budget; "+
			"no required obligation was silently truncated.\n", opts.MaxTokens)
	}
	return out
}

// EstimateTokens returns a conservative token estimate of a rendered
// document.
//
// The estimate uses the widely used approximation of four characters per
// token, rounded up. It is deliberately an over-estimate so the budget
// check fails safe.
func EstimateTokens(text string) int {
	if text == "" {
		return 0
	}
	return (len(text) + 3) / 4
}
