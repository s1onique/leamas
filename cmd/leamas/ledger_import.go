package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

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
