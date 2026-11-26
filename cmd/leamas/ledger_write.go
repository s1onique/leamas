package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/s1onique/leamas/internal/ledger/domain"
	"github.com/s1onique/leamas/internal/ledger/service"
	"github.com/s1onique/leamas/internal/ledger/store"
)

func runLedgerCreate(args []string) int {
	fs := flag.NewFlagSet("create", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	root := fs.String("root", store.DefaultRoot, "ledger root directory")
	epic := fs.String("epic", "", "epic scope (required)")
	item := fs.String("item", "", "work item identity (required)")
	title := fs.String("title", "", "work item title (required)")
	summary := fs.String("summary", "", "work item summary")
	kind := fs.String("kind", string(domain.ItemKindTask), "work item kind")
	priority := fs.String("priority", string(domain.PriorityP2), "work item priority")
	owner := fs.String("owner", "", "work item owner")
	criteria := fs.String("criterion", "", "acceptance criterion (repeatable)")
	exitCriteria := fs.String("exit-criterion", "", "exit criterion (repeatable)")
	live := fs.Bool("requires-live-gate", false, "acceptance requires a live qualification gate")
	jsonOut := fs.Bool("json", false, "output JSON")
	if err := fs.Parse(args); err != nil {
		return 1
	}
	if *epic == "" || *item == "" || *title == "" {
		fmt.Fprintln(os.Stderr, "ERROR: create requires --epic, --item and --title")
		return 1
	}
	svc := ledgerService(*root)
	created, err := svc.CreateItem("ledger-create", service.CreateItemOptions{
		EpicID:       domain.EpicID(*epic),
		ItemID:       domain.ItemID(*item),
		Title:        *title,
		Summary:      *summary,
		Kind:         domain.ItemKind(*kind),
		Priority:     domain.Priority(*priority),
		Owner:        *owner,
		Criteria:     stringList(*criteria),
		ExitCriteria: stringList(*exitCriteria),
		RequiresLive: *live,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
		return 1
	}
	if *jsonOut {
		emitJSON(map[string]any{"ok": true, "item": created})
		return 0
	}
	fmt.Printf("created %s\n", created.Scoped())
	return 0
}

func runLedgerUpdate(args []string) int {
	fs := flag.NewFlagSet("update", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	root := fs.String("root", store.DefaultRoot, "ledger root directory")
	id := fs.String("id", "", "work item identity (required)")
	expectedRev := fs.Int("expect-revision", 0, "expected revision (optimistic concurrency)")
	title := fs.String("title", "", "new title")
	summary := fs.String("summary", "", "new summary")
	kind := fs.String("kind", "", "new kind")
	lifecycle := fs.String("lifecycle", "", "new lifecycle state")
	acceptance := fs.String("acceptance", "", "new acceptance state")
	priority := fs.String("priority", "", "new priority")
	quality := fs.String("evidence-quality", "", "new evidence quality")
	owner := fs.String("owner", "", "new owner")
	criteria := fs.String("criterion", "", "acceptance criterion (repeatable)")
	exitCriteria := fs.String("exit-criterion", "", "exit criterion (repeatable)")
	live := fs.Bool("requires-live-gate", false, "acceptance requires a live qualification gate")
	jsonOut := fs.Bool("json", false, "output JSON")
	if err := fs.Parse(args); err != nil {
		return 1
	}
	if *id == "" {
		fmt.Fprintln(os.Stderr, "ERROR: update requires --id")
		return 1
	}
	opts := service.UpdateItemOptions{
		ScopedID:    domain.ScopedItemID(*id),
		ExpectedRev: *expectedRev,
	}
	if *title != "" {
		opts.Title = title
	}
	if *summary != "" {
		opts.Summary = summary
	}
	if *kind != "" {
		k := domain.ItemKind(*kind)
		opts.Kind = &k
	}
	if *lifecycle != "" {
		s := domain.LifecycleState(*lifecycle)
		opts.Lifecycle = &s
	}
	if *acceptance != "" {
		s := domain.AcceptanceState(*acceptance)
		opts.Acceptance = &s
	}
	if *priority != "" {
		p := domain.Priority(*priority)
		opts.Priority = &p
	}
	if *quality != "" {
		q := domain.EvidenceQuality(*quality)
		opts.Quality = &q
	}
	if *owner != "" {
		opts.Owner = owner
	}
	if *criteria != "" {
		opts.Criteria = stringList(*criteria)
	}
	if *exitCriteria != "" {
		opts.ExitCriteria = stringList(*exitCriteria)
	}
	if *live {
		opts.RequiresLive = live
	}
	updated, err := ledgerService(*root).UpdateItem("ledger-update", opts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
		return 1
	}
	if *jsonOut {
		emitJSON(map[string]any{"ok": true, "item": updated})
		return 0
	}
	fmt.Printf("updated %s (revision %d)\n", updated.Scoped(), updated.Revision)
	return 0
}

func runLedgerClaim(args []string) int {
	fs := flag.NewFlagSet("claim", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	root := fs.String("root", store.DefaultRoot, "ledger root directory")
	id := fs.String("id", "", "work item identity (required)")
	owner := fs.String("owner", "", "new owner (required)")
	expectedRev := fs.Int("expect-revision", 0, "expected revision (optimistic concurrency)")
	jsonOut := fs.Bool("json", false, "output JSON")
	if err := fs.Parse(args); err != nil {
		return 1
	}
	if *id == "" || *owner == "" {
		fmt.Fprintln(os.Stderr, "ERROR: claim requires --id and --owner")
		return 1
	}
	updated, err := ledgerService(*root).Claim("ledger-claim", service.ClaimOptions{
		ScopedID:    domain.ScopedItemID(*id),
		ExpectedRev: *expectedRev,
		Owner:       *owner,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
		return 1
	}
	if *jsonOut {
		emitJSON(map[string]any{"ok": true, "item": updated})
		return 0
	}
	fmt.Printf("claimed %s by %s\n", updated.Scoped(), updated.Owner)
	return 0
}

func runLedgerLink(args []string) int {
	fs := flag.NewFlagSet("link", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	root := fs.String("root", store.DefaultRoot, "ledger root directory")
	from := fs.String("from", "", "source work item identity (required)")
	to := fs.String("to", "", "target work item identity (required)")
	kind := fs.String("kind", string(domain.DependencyBlocks), "dependency kind (blocks, requires, relates)")
	note := fs.String("note", "", "operator note")
	expectedRev := fs.Int("expect-revision", 0, "expected revision of the source work item")
	successor := fs.Bool("successor", false, "record the target as a successor obligation instead of a dependency")
	jsonOut := fs.Bool("json", false, "output JSON")
	if err := fs.Parse(args); err != nil {
		return 1
	}
	if *from == "" || *to == "" {
		fmt.Fprintln(os.Stderr, "ERROR: link requires --from and --to")
		return 1
	}
	svc := ledgerService(*root)
	if *successor {
		updated, err := svc.SetSuccessor("ledger-successor", service.SetSuccessorOptions{
			ScopedID:    domain.ScopedItemID(*from),
			Successor:   domain.ScopedItemID(*to),
			ExpectedRev: *expectedRev,
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
			return 1
		}
		if *jsonOut {
			emitJSON(map[string]any{"ok": true, "item": updated})
			return 0
		}
		fmt.Printf("recorded successor %s -> %s\n", *from, *to)
		return 0
	}
	updated, err := svc.Link("ledger-link", service.LinkOptions{
		From:        domain.ScopedItemID(*from),
		To:          domain.ScopedItemID(*to),
		Kind:        domain.DependencyKind(*kind),
		Note:        *note,
		ExpectedRev: *expectedRev,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
		return 1
	}
	if *jsonOut {
		emitJSON(map[string]any{"ok": true, "item": updated})
		return 0
	}
	fmt.Printf("linked %s -> %s (%s)\n", *from, *to, *kind)
	return 0
}

// stringList splits a repeatable flag value into a trimmed list.
func stringList(value string) []string {
	if value == "" {
		return nil
	}
	var out []string
	for _, part := range splitLines(value) {
		trimmed := trimSpace(part)
		if trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}
