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
