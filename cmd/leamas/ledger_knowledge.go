package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/s1onique/leamas/internal/ledger/domain"
	"github.com/s1onique/leamas/internal/ledger/service"
	"github.com/s1onique/leamas/internal/ledger/store"
)

// runLedgerDecision registers or looks up a durable decision.
func runLedgerDecision(args []string) int {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "ERROR: decision requires a subcommand (add, show)")
		return 1
	}
	switch args[0] {
	case "add":
		return runLedgerDecisionAdd(args[1:])
	case "show":
		return runLedgerDecisionShow(args[1:])
	default:
		fmt.Fprintf(os.Stderr, "unknown ledger decision subcommand: %s\n", args[0])
		return 1
	}
}

func runLedgerDecisionAdd(args []string) int {
	fs := flag.NewFlagSet("decision add", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	root := fs.String("root", store.DefaultRoot, "ledger root directory")
	id := fs.String("id", "", "decision identity (required)")
	title := fs.String("title", "", "decision title (required)")
	rationale := fs.String("rationale", "", "decision rationale")
	item := fs.String("item", "", "work item the decision belongs to")
	jsonOut := fs.Bool("json", false, "output JSON")
	if err := fs.Parse(args); err != nil {
		return 1
	}
	if *id == "" || *title == "" {
		fmt.Fprintln(os.Stderr, "ERROR: decision add requires --id and --title")
		return 1
	}
	created, err := ledgerService(*root).RecordDecision("ledger-decision",
		service.RecordDecisionOptions{
			DecisionID: domain.DecisionID(*id),
			Title:      *title,
			Rationale:  *rationale,
			ItemID:     domain.ScopedItemID(*item),
		})
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
		return 1
	}
	if *jsonOut {
		emitJSON(map[string]any{"ok": true, "decision": created})
		return 0
	}
	fmt.Printf("recorded decision %s\n", created.DecisionID)
	return 0
}

// runLedgerDecisionShow performs the historical lookup of a decision.
func runLedgerDecisionShow(args []string) int {
	fs := flag.NewFlagSet("decision show", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	root := fs.String("root", store.DefaultRoot, "ledger root directory")
	id := fs.String("id", "", "decision identity (required)")
	jsonOut := fs.Bool("json", false, "output JSON")
	if err := fs.Parse(args); err != nil {
		return 1
	}
	if *id == "" {
		fmt.Fprintln(os.Stderr, "ERROR: decision show requires --id")
		return 1
	}
	l, err := ledgerService(*root).Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
		return 1
	}
	for _, d := range l.Decisions {
		if d.DecisionID == domain.DecisionID(*id) {
			if *jsonOut {
				emitJSON(map[string]any{"ok": true, "decision": d})
				return 0
			}
			fmt.Printf("%s %s\n", d.DecisionID, d.Title)
			return 0
		}
	}
	fmt.Fprintf(os.Stderr, "ERROR: record not found: %s\n", *id)
	return 1
}

// runLedgerKnowledge registers or looks up a knowledge reference.
func runLedgerKnowledge(args []string) int {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "ERROR: knowledge requires a subcommand (add, show)")
		return 1
	}
	switch args[0] {
	case "add":
		return runLedgerKnowledgeAdd(args[1:])
	case "show":
		return runLedgerKnowledgeShow(args[1:])
	default:
		fmt.Fprintf(os.Stderr, "unknown ledger knowledge subcommand: %s\n", args[0])
		return 1
	}
}

func runLedgerKnowledgeAdd(args []string) int {
	fs := flag.NewFlagSet("knowledge add", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	root := fs.String("root", store.DefaultRoot, "ledger root directory")
	id := fs.String("id", "", "knowledge identity (required)")
	title := fs.String("title", "", "knowledge title (required)")
	summary := fs.String("summary", "", "knowledge summary")
	relPath := fs.String("relative-path", "", "repository-relative artifact path")
	item := fs.String("item", "", "work item the knowledge belongs to")
	jsonOut := fs.Bool("json", false, "output JSON")
	if err := fs.Parse(args); err != nil {
		return 1
	}
	if *id == "" || *title == "" {
		fmt.Fprintln(os.Stderr, "ERROR: knowledge add requires --id and --title")
		return 1
	}
	created, err := ledgerService(*root).RecordKnowledge("ledger-knowledge",
		service.RecordKnowledgeOptions{
			KnowledgeID:  domain.KnowledgeID(*id),
			Title:        *title,
			Summary:      *summary,
			RelativePath: *relPath,
			ItemID:       domain.ScopedItemID(*item),
		})
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
		return 1
	}
	if *jsonOut {
		emitJSON(map[string]any{"ok": true, "knowledge": created})
		return 0
	}
	fmt.Printf("recorded knowledge %s\n", created.KnowledgeID)
	return 0
}

// runLedgerKnowledgeShow performs the historical lookup of a knowledge
// reference.
func runLedgerKnowledgeShow(args []string) int {
	fs := flag.NewFlagSet("knowledge show", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	root := fs.String("root", store.DefaultRoot, "ledger root directory")
	id := fs.String("id", "", "knowledge identity (required)")
	jsonOut := fs.Bool("json", false, "output JSON")
	if err := fs.Parse(args); err != nil {
		return 1
	}
	if *id == "" {
		fmt.Fprintln(os.Stderr, "ERROR: knowledge show requires --id")
		return 1
	}
	l, err := ledgerService(*root).Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
		return 1
	}
	for _, k := range l.Knowledge {
		if k.KnowledgeID == domain.KnowledgeID(*id) {
			if *jsonOut {
				emitJSON(map[string]any{"ok": true, "knowledge": k})
				return 0
			}
			fmt.Printf("%s %s\n", k.KnowledgeID, k.Title)
			return 0
		}
	}
	fmt.Fprintf(os.Stderr, "ERROR: record not found: %s\n", *id)
	return 1
}
