package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/s1onique/leamas/internal/ledger/boardmd"
	"github.com/s1onique/leamas/internal/ledger/importer"
	"github.com/s1onique/leamas/internal/ledger/store"
)

func runLedgerImport(args []string) int {
	fs := flag.NewFlagSet("import", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	root := fs.String("root", store.DefaultRoot, "ledger root directory")
	source := fs.String("source", "", "path to the source board JSON file (required)")
	apply := fs.Bool("apply", false, "apply the import (default is a dry run)")
	jsonOut := fs.Bool("json", false, "output JSON")
	if err := fs.Parse(args); err != nil {
		return 1
	}
	if *source == "" {
		fmt.Fprintln(os.Stderr, "ERROR: import requires --source")
		return 1
	}
	data, err := os.ReadFile(*source)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: read source board: %v\n", err)
		return 1
	}
	var board importer.SourceBoard
	if err := json.Unmarshal(data, &board); err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: decode source board: %v\n", err)
		return 1
	}

	st := store.New(*root)
	existing, err := st.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
		return 1
	}
	plan := importer.BuildPlan(board, existing)

	if !*apply {
		if *jsonOut {
			emitJSON(map[string]any{"ok": true, "dry_run": true, "plan": plan})
			return 0
		}
		fmt.Print("dry run — no records written\n")
		fmt.Print(plan.Summary())
		for _, a := range plan.Ambiguities {
			fmt.Printf("ambiguity: %s: %s: %s\n", a.SourceID, a.Kind, a.Detail)
		}
		for _, u := range plan.Unresolved {
			fmt.Printf("unresolved: %s: %s (%s)\n", u.SourceID, u.Ref, u.Kind)
		}
		return 0
	}

	res, err := importer.Apply(st, plan, "ledger-import")
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
		return 1
	}
	updated, err := st.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
		return 1
	}
	delta := importer.ComputeDelta(existing, updated)
	if *jsonOut {
		emitJSON(map[string]any{
			"ok": true, "dry_run": false, "result": res,
			"delta": delta, "ambiguities": plan.Ambiguities,
			"unresolved": plan.Unresolved,
		})
		return 0
	}
	fmt.Print(plan.Summary())
	fmt.Printf("delta: %s\n", delta.Describe())
	return 0
}

// runLedgerImportOriginal imports the ORIGINAL IECR board intake fixture.
//
// The original schema is NOT the repository-native source board schema. The
// command uses the minimum necessary versioned adapter.
//
// The command is a dry run by default: it parses, normalizes and reports
// the conservation WITHOUT writing anything. The applied import requires
// the explicit --apply flag.
func runLedgerImportOriginal(args []string) int {
	fs := flag.NewFlagSet("import-original", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	root := fs.String("root", store.DefaultRoot, "ledger root directory")
	source := fs.String("source", "", "path to the original fixture JSON file (required)")
	apply := fs.Bool("apply", false, "apply the import (default is a dry run)")
	jsonOut := fs.Bool("json", false, "output JSON")
	if err := fs.Parse(args); err != nil {
		return 1
	}
	if *source == "" {
		fmt.Fprintln(os.Stderr, "ERROR: import-original requires --source")
		return 1
	}
	data, err := os.ReadFile(*source)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: read the original fixture: %v\n", err)
		return 1
	}

	board, inventory, err := importer.ParseOriginalFixture(data)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
		return 1
	}
	st := store.New(*root)
	existing, err := st.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
		return 1
	}
	plan := importer.BuildPlan(board, existing)
	receipt := importer.BuildReceipt(existing, plan)

	if !*apply {
		if *jsonOut {
			emitJSON(map[string]any{
				"ok": true, "dry_run": true, "inventory": inventory,
				"receipt": receipt, "plan": plan,
				"unavailable": importer.UnavailableHistoricalInformation(),
			})
			return 0
		}
		fmt.Print("dry run — no records written\n")
		printOriginalInventory(inventory)
		fmt.Print(plan.Summary())
		for _, rec := range receipt.Reconciliations {
			fmt.Printf("reconciliation: %s: %s: %s\n", rec.ScopedID, rec.Outcome, rec.Detail)
		}
		for _, u := range plan.Unresolved {
			fmt.Printf("unresolved: %s: %s (%s)\n", u.SourceID, u.Ref, u.Kind)
		}
		for _, u := range importer.UnavailableHistoricalInformation() {
			fmt.Printf("unavailable: %s\n", u)
		}
		return 0
	}

	if _, err := importer.Apply(st, plan, "ledger-import-original"); err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
		return 1
	}
	updated, err := st.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
		return 1
	}
	delta := importer.ComputeDelta(existing, updated)
	if *jsonOut {
		emitJSON(map[string]any{
			"ok": true, "dry_run": false, "inventory": inventory,
			"receipt": receipt, "delta": delta,
			"unavailable": importer.UnavailableHistoricalInformation(),
		})
		return 0
	}
	printOriginalInventory(inventory)
	fmt.Printf("delta: %s\n", delta.Describe())
	return 0
}

// runLedgerMigrateBoard performs the SHADOW MIGRATION of the actual
// historical board into a ledger root.
//
// The command is a dry run by default: it extracts the historical board,
// builds the deterministic migration plan, reports the complete accounting
// and persists NOTHING. The applied migration requires the explicit
// --apply flag.
//
// The command NEVER touches the legacy board: the historical board remains
// authoritative until an operator cuts over explicitly.
func runLedgerMigrateBoard(args []string) int {
	fs := flag.NewFlagSet("migrate-board", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	root := fs.String("root", store.DefaultRoot, "ledger root directory")
	sourceRoot := fs.String("source-root", ".", "root of the source repository that holds the historical board")
	repository := fs.String("repository", "", "source repository identity")
	head := fs.String("source-head", "", "source HEAD the migration is taken at")
	apply := fs.Bool("apply", false, "apply the migration (default is a dry run)")
	jsonOut := fs.Bool("json", false, "output JSON")
	if err := fs.Parse(args); err != nil {
		return 1
	}

	extraction, err := boardmd.Extract(*sourceRoot, *repository, *head)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
		return 1
	}

	st := store.New(*root)
	existing, err := st.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
		return 1
	}
	plan := importer.BuildPlan(extraction.Board, existing)

	if !*apply {
		if *jsonOut {
			emitJSON(map[string]any{
				"ok": true, "dry_run": true,
				"source":      extraction.Source,
				"inventory":   extraction.Inventory,
				"sections":    extraction.Sections,
				"acts":        extraction.ACTs,
				"plan":        plan,
				"unavailable": extraction.Unavailable,
				"adapter":     boardmd.AdapterVersion,
			})
			return 0
		}
		fmt.Print("dry run — no records written\n")
		printBoardInventory(extraction)
		fmt.Print(plan.Summary())
		for _, rec := range plan.Reconciliations {
			fmt.Printf("reconciliation: %s: %s: %s\n", rec.ScopedID, rec.Outcome, rec.Detail)
		}
		for _, u := range plan.Unresolved {
			fmt.Printf("unresolved: %s: %s (%s)\n", u.SourceID, u.Ref, u.Kind)
		}
		for _, u := range extraction.Unavailable {
			fmt.Printf("unavailable: %s\n", u)
		}
		return 0
	}

	if _, err := importer.ApplyIsolated(st, plan, "ledger-migrate-board"); err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
		return 1
	}
	persisted, err := st.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
		return 1
	}
	// The receipt is computed against the ACTUAL PERSISTED state: a plan
	// is not a conservation proof.
	receipt := importer.BuildPersistedReceipt(existing, plan, persisted)
	delta := importer.ComputeDelta(existing, persisted)
	if *jsonOut {
		emitJSON(map[string]any{
			"ok": true, "dry_run": false,
			"source":      extraction.Source,
			"inventory":   extraction.Inventory,
			"receipt":     receipt,
			"delta":       delta,
			"unavailable": extraction.Unavailable,
			"adapter":     boardmd.AdapterVersion,
		})
		if !receipt.Conserved {
			return 1
		}
		return 0
	}
	printBoardInventory(extraction)
	fmt.Printf("delta: %s\n", delta.Describe())
	fmt.Printf("conserved: %t\n", receipt.Conserved)
	for _, d := range receipt.Discrepancies {
		fmt.Printf("discrepancy: %s\n", d)
	}
	if !receipt.Conserved {
		return 1
	}
	return 0
}

// printBoardInventory renders the observed inventory of the historical
// board extraction.
func printBoardInventory(ex boardmd.Extraction) {
	fmt.Printf("adapter version: %d\n", boardmd.AdapterVersion)
	fmt.Printf("source repository: %s\n", ex.Source.Repository)
	fmt.Printf("source head: %s\n", ex.Source.Head)
	fmt.Printf("source sections: %d\n", ex.Inventory.Sections)
	fmt.Printf("epic scopes: %d\n", ex.Inventory.Epics)
	fmt.Printf("work items: %d\n", ex.Inventory.WorkItems)
	fmt.Printf("act executions: %d\n", ex.Inventory.ACTs)
	fmt.Printf("exit criteria: %d\n", ex.Inventory.ExitCriteria)
	fmt.Printf("unfinished records: %d\n", ex.Inventory.Unfinished)
	fmt.Printf("unresolved records: %d\n", ex.Inventory.Unresolved)
	fmt.Printf("duplicate representations: %d\n", ex.Inventory.DuplicateSections)
}

// printOriginalInventory renders the observed inventory of the original
// fixture.
func printOriginalInventory(inv importer.OriginalInventory) {
	fmt.Printf("classification: %s\n", inv.Classification)
	fmt.Printf("source work items: %d\n", inv.WorkItems)
	fmt.Printf("source workstreams: %d\n", inv.Workstreams)
	fmt.Printf("source epics: %d\n", inv.Epics)
	fmt.Printf("source exit criteria: %d\n", inv.ExitCriteria)
	fmt.Printf("distinct source statuses: %d\n", inv.DistinctStatuses)
	fmt.Printf("unfinished source records: %d\n", inv.Unfinished)
}
