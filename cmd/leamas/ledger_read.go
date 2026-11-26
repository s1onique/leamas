package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/s1onique/leamas/internal/ledger/domain"
	"github.com/s1onique/leamas/internal/ledger/render"
	"github.com/s1onique/leamas/internal/ledger/store"
)

func runLedgerInit(args []string) int {
	fs := flag.NewFlagSet("init", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	root := fs.String("root", store.DefaultRoot, "ledger root directory")
	jsonOut := fs.Bool("json", false, "output JSON")
	if err := fs.Parse(args); err != nil {
		return 1
	}
	svc := ledgerService(*root)
	if err := svc.Init(); err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
		return 1
	}
	if *jsonOut {
		emitJSON(map[string]any{"ok": true, "root": *root, "schema_version": domain.LedgerSchemaVersion})
		return 0
	}
	fmt.Printf("initialized ledger: %s\n", *root)
	return 0
}

func runLedgerList(args []string) int {
	fs := flag.NewFlagSet("list", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	root := fs.String("root", store.DefaultRoot, "ledger root directory")
	epic := fs.String("epic", "", "restrict to one epic scope")
	owner := fs.String("owner", "", "restrict to one owner")
	archived := fs.Bool("archived", false, "include archived records")
	jsonOut := fs.Bool("json", false, "output JSON")
	if err := fs.Parse(args); err != nil {
		return 1
	}
	l, err := ledgerService(*root).Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
		return 1
	}
	items := []domain.WorkItem{}
	for _, it := range l.Items {
		if it.Archived && !*archived {
			continue
		}
		if *epic != "" && string(it.EpicID) != *epic {
			continue
		}
		if *owner != "" && it.Owner != *owner {
			continue
		}
		items = append(items, it)
	}
	domain.SortItems(items)
	if *jsonOut {
		emitJSON(map[string]any{"ok": true, "items": items, "count": len(items)})
		return 0
	}
	for _, it := range items {
		fmt.Printf("%s\t%s\t%s\t%s\t%s\n",
			it.Scoped(), it.Kind, it.Lifecycle, it.Acceptance, it.Title)
	}
	return 0
}

func runLedgerShow(args []string) int {
	fs := flag.NewFlagSet("show", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	root := fs.String("root", store.DefaultRoot, "ledger root directory")
	jsonOut := fs.Bool("json", false, "output JSON")
	if err := fs.Parse(args); err != nil {
		return 1
	}
	if fs.NArg() != 1 {
		fmt.Fprintln(os.Stderr, "ERROR: show requires exactly one work item id")
		return 1
	}
	scoped := domain.ScopedItemID(fs.Arg(0))
	if err := domain.ValidateScopedItemID(scoped); err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
		return 1
	}
	l, err := ledgerService(*root).Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
		return 1
	}
	it, ok := l.ItemByScoped(scoped)
	if !ok {
		fmt.Fprintf(os.Stderr, "ERROR: work item not found: %s\n", scoped.String())
		return 1
	}
	if *jsonOut {
		emitJSON(map[string]any{"ok": true, "item": it, "blockers": domain.BlockersFor(l, it)})
		return 0
	}
	printWorkItem(l, it)
	return 0
}

func printWorkItem(l domain.Ledger, it domain.WorkItem) {
	fmt.Printf("id: %s\n", it.Scoped())
	fmt.Printf("title: %s\n", it.Title)
	fmt.Printf("kind: %s\n", it.Kind)
	fmt.Printf("lifecycle: %s\n", it.Lifecycle)
	fmt.Printf("acceptance: %s\n", it.Acceptance)
	fmt.Printf("priority: %s\n", it.Priority)
	fmt.Printf("evidence_quality: %s\n", it.Quality)
	if it.Owner != "" {
		fmt.Printf("owner: %s\n", it.Owner)
	}
	fmt.Printf("revision: %d\n", it.Revision)
	if it.Archived {
		fmt.Printf("archived: true\n")
	}
	if it.TerminalDisposition != "" {
		fmt.Printf("terminal_disposition: %s\n", it.TerminalDisposition)
	}
	for _, c := range it.AcceptanceCriteria {
		fmt.Printf("criterion: %s\n", c)
	}
	for _, c := range it.ExitCriteria {
		fmt.Printf("exit_criterion: %s\n", c)
	}
	for _, d := range it.Dependencies {
		fmt.Printf("depends_on(%s): %s\n", d.Kind, d.Ref)
	}
	for _, e := range it.Evidence {
		fmt.Printf("evidence(%s): %s\n", e.Quality, e.EvidenceID)
	}
	for _, s := range it.Successors {
		fmt.Printf("successor: %s\n", s)
	}
	for _, r := range it.Reopen {
		fmt.Printf("reopen_condition: %s\n", r.Kind)
	}
	for _, b := range domain.BlockersFor(l, it) {
		fmt.Printf("blocker: %s\n", b)
	}
}

func runLedgerNext(args []string) int {
	fs := flag.NewFlagSet("next", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	root := fs.String("root", store.DefaultRoot, "ledger root directory")
	epic := fs.String("epic", "", "restrict to one epic scope")
	owner := fs.String("owner", "", "restrict to one owner")
	limit := fs.Int("limit", 1, "maximum number of items")
	includeLive := fs.Bool("include-live-gate", false, "include items awaiting a live qualification gate")
	jsonOut := fs.Bool("json", false, "output JSON")
	if err := fs.Parse(args); err != nil {
		return 1
	}
	l, err := ledgerService(*root).Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
		return 1
	}
	ready := domain.SelectReady(l, domain.ReadyOptions{
		Epic:            domain.EpicID(*epic),
		Owner:           *owner,
		Limit:           *limit,
		IncludeLiveGate: *includeLive,
	})
	if *jsonOut {
		emitJSON(map[string]any{"ok": true, "ready": ready, "count": len(ready)})
		return 0
	}
	for _, item := range ready {
		fmt.Printf("%s\t%s\t%s\t%s\n", item.ScopedID, item.Kind, item.Priority, item.Title)
	}
	return 0
}

func runLedgerPrime(args []string) int {
	fs := flag.NewFlagSet("prime", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	root := fs.String("root", store.DefaultRoot, "ledger root directory")
	epic := fs.String("epic", "", "restrict to one epic scope")
	maxItems := fs.Int("max-items", 0, "maximum number of work items")
	maxTokens := fs.Int("max-tokens", 0, "token budget for the bootstrap context")
	jsonOut := fs.Bool("json", false, "output JSON")
	if err := fs.Parse(args); err != nil {
		return 1
	}
	l, err := ledgerService(*root).Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
		return 1
	}
	text := render.Bootstrap(l, render.BootstrapOptions{
		Epic:      domain.EpicID(*epic),
		MaxItems:  *maxItems,
		MaxTokens: *maxTokens,
	})
	if *jsonOut {
		emitJSON(map[string]any{
			"ok":               true,
			"context":          text,
			"estimated_tokens": render.EstimateTokens(text),
		})
		return 0
	}
	fmt.Print(text)
	return 0
}

func runLedgerStats(args []string) int {
	fs := flag.NewFlagSet("stats", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	root := fs.String("root", store.DefaultRoot, "ledger root directory")
	jsonOut := fs.Bool("json", false, "output JSON")
	if err := fs.Parse(args); err != nil {
		return 1
	}
	l, err := ledgerService(*root).Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
		return 1
	}
	stats := domain.Stats(l)
	if *jsonOut {
		emitJSON(map[string]any{"ok": true, "stats": stats})
		return 0
	}
	fmt.Print(render.StatsView(l))
	return 0
}

func runLedgerValidate(args []string) int {
	fs := flag.NewFlagSet("validate", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	root := fs.String("root", store.DefaultRoot, "ledger root directory")
	jsonOut := fs.Bool("json", false, "output JSON")
	if err := fs.Parse(args); err != nil {
		return 1
	}
	l, err := ledgerService(*root).Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
		return 1
	}
	res := domain.ValidateLedger(l)
	if *jsonOut {
		emitJSON(map[string]any{"ok": res.OK(), "findings": res.Findings})
		if !res.OK() {
			return 1
		}
		return 0
	}
	if res.OK() {
		fmt.Println("ledger validation PASSED")
		return 0
	}
	fmt.Println("ledger validation FAILED")
	for _, f := range res.Findings {
		fmt.Printf("  %s: %s: %s\n", f.Path, f.Code, f.Message)
	}
	return 1
}

func runLedgerRender(args []string) int {
	fs := flag.NewFlagSet("render", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	root := fs.String("root", store.DefaultRoot, "ledger root directory")
	view := fs.String("view", "board", "view to render (board, epic, ready, blockers, history, stats)")
	epic := fs.String("epic", "", "epic scope for the 'epic' view")
	jsonOut := fs.Bool("json", false, "output JSON")
	if err := fs.Parse(args); err != nil {
		return 1
	}
	l, err := ledgerService(*root).Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
		return 1
	}
	var text string
	switch *view {
	case "board":
		text = render.Board(l)
	case "epic":
		text = render.EpicView(l, domain.EpicID(*epic))
	case "ready":
		text = render.ReadyView(l, domain.ReadyOptions{})
	case "blockers":
		text = render.BlockersView(l)
	case "history":
		text = render.HistoryView(l)
	case "stats":
		text = render.StatsView(l)
	default:
		fmt.Fprintf(os.Stderr, "ERROR: unknown view %q\n", *view)
		return 1
	}
	if *jsonOut {
		emitJSON(map[string]any{"ok": true, "view": *view, "content": text})
		return 0
	}
	fmt.Print(text)
	return 0
}

// emitJSON writes an indented JSON document to stdout.
func emitJSON(v any) {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: encode JSON: %v\n", err)
		return
	}
	fmt.Println(string(data))
}
