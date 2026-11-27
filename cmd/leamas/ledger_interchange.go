package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/s1onique/leamas/internal/ledger/domain"
	"github.com/s1onique/leamas/internal/ledger/fwmi"
	"github.com/s1onique/leamas/internal/ledger/importer"
	"github.com/s1onique/leamas/internal/ledger/store"
)

// runLedgerInterchangeImport imports an FWMI v1 migration interchange
// package into the canonical ledger.
//
// The command is a dry run by default: it validates the interchange records
// strictly, adapts them, builds the deterministic import plan and reports
// the complete accounting WITHOUT writing anything. The applied import
// requires the explicit --apply flag.
//
// The import NEVER promotes a historical qualification assertion into a
// newly verified acceptance decision, and it NEVER converts an unknown
// source status into an intentional deferral.
func runLedgerInterchangeImport(args []string) int {
	fs := flag.NewFlagSet("import-interchange", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	root := fs.String("root", store.DefaultRoot, "ledger root directory")
	source := fs.String("source", "", "path to the FWMI interchange package directory (required)")
	apply := fs.Bool("apply", false, "apply the import (default is a dry run)")
	jsonOut := fs.Bool("json", false, "output JSON")
	if err := fs.Parse(args); err != nil {
		return 1
	}
	if *source == "" {
		fmt.Fprintln(os.Stderr, "ERROR: import-interchange requires --source")
		return 1
	}

	pkg, err := fwmi.Load(*source)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
		return 1
	}
	adapted, err := fwmi.Adapt(pkg)
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
	plan := importer.BuildPlan(adapted.Board, existing)

	if !*apply {
		if *jsonOut {
			emitJSON(map[string]any{
				"ok": true, "dry_run": true,
				"adapter": fwmi.AdapterVersion,
				"source": map[string]any{
					"repository":  pkg.Manifest.Repository,
					"source_head": pkg.Manifest.SourceHead,
				},
				"stats":           adapted.Stats,
				"plan":            plan,
				"reconciliations": plan.Reconciliations,
				"unresolved":      plan.Unresolved,
				"ambiguities":     plan.Ambiguities,
			})
			return 0
		}
		fmt.Print("dry run — no records written\n")
		printInterchangeInventory(pkg, adapted)
		fmt.Print(plan.Summary())
		for _, rec := range plan.Reconciliations {
			fmt.Printf("reconciliation: %s: %s: %s\n", rec.ScopedID, rec.Outcome, rec.Detail)
		}
		for _, u := range plan.Unresolved {
			fmt.Printf("unresolved: %s: %s (%s)\n", u.SourceID, u.Ref, u.Kind)
		}
		for _, a := range plan.Ambiguities {
			fmt.Printf("ambiguity: %s: %s: %s\n", a.SourceID, a.Kind, a.Detail)
		}
		return 0
	}

	if _, err := importer.ApplyIsolated(st, plan, "ledger-import-interchange"); err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
		return 1
	}
	persisted, err := st.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
		return 1
	}
	receipt := importer.BuildPersistedReceipt(existing, plan, persisted)
	delta := importer.ComputeDelta(existing, persisted)
	if *jsonOut {
		emitJSON(map[string]any{
			"ok": true, "dry_run": false,
			"adapter":         fwmi.AdapterVersion,
			"stats":           adapted.Stats,
			"receipt":         receipt,
			"delta":           delta,
			"reconciliations": plan.Reconciliations,
			"unresolved":      plan.Unresolved,
		})
		if !receipt.Conserved {
			return 1
		}
		return 0
	}
	printInterchangeInventory(pkg, adapted)
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

// runLedgerInterchangeHistory performs the historical lookup of an indexed
// historical ACT execution.
//
// The lookup covers the ARCHIVED historical index: an indexed historical
// execution attempt stays discoverable with its original disposition and its
// source provenance, without ever re-entering the live execution history.
func runLedgerInterchangeHistory(args []string) int {
	fs := flag.NewFlagSet("interchange-history", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	root := fs.String("root", store.DefaultRoot, "ledger root directory")
	id := fs.String("id", "", "historical ACT identity (required)")
	jsonOut := fs.Bool("json", false, "output JSON")
	if err := fs.Parse(args); err != nil {
		return 1
	}
	if *id == "" {
		fmt.Fprintln(os.Stderr, "ERROR: interchange-history requires --id")
		return 1
	}
	l, err := ledgerService(*root).Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
		return 1
	}
	act, ok := l.ActByID(domain.ActID(*id))
	if !ok {
		fmt.Fprintf(os.Stderr, "ERROR: historical ACT not found: %s\n", *id)
		return 1
	}
	if *jsonOut {
		emitJSON(map[string]any{"ok": true, "act": act})
		return 0
	}
	fmt.Printf("%s\tarchived=%t\tverdict=%s\toriginal_verdict=%s\n",
		act.ActID, act.Archived, act.Verdict, act.OriginalVerdict)
	for _, item := range act.ItemIDs {
		fmt.Printf("work item: %s\n", item)
	}
	for _, item := range act.UnresolvedItemIDs {
		fmt.Printf("unresolved work item association: %s\n", item)
	}
	for _, loc := range act.EvidenceLocators {
		fmt.Printf("evidence locator: %s\n", loc)
	}
	for _, ref := range act.SourceRefs {
		fmt.Printf("source: %s:%d-%d\n", ref.Path, ref.StartLine, ref.EndLine)
	}
	return 0
}

// printInterchangeInventory renders the observed inventory of the package.
func printInterchangeInventory(pkg *fwmi.Package, adapted fwmi.AdaptResult) {
	fmt.Printf("adapter version: %d\n", fwmi.AdapterVersion)
	fmt.Printf("source repository: %s\n", pkg.Manifest.Repository)
	fmt.Printf("source head: %s\n", pkg.Manifest.SourceHead)
	fmt.Printf("epic scopes: %d\n", adapted.Stats.Epics)
	fmt.Printf("work items: %d\n", adapted.Stats.Items)
	fmt.Printf("historical acts: %d\n", adapted.Stats.HistoricalActs)
	fmt.Printf("dependencies: %d\n", adapted.Stats.Dependencies)
	fmt.Printf("cross links: %d\n", adapted.Stats.CrossLinks)
	fmt.Printf("reconciliation records: %d\n", adapted.Stats.ReconciliationRecords)
	fmt.Printf("explicit blockers: %d\n", adapted.Stats.Blockers)
	fmt.Printf("historical ACT associations: %d\n", adapted.Stats.HistoricalAssociations)
}
