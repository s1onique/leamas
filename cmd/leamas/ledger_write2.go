package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/s1onique/leamas/internal/ledger/domain"
	"github.com/s1onique/leamas/internal/ledger/service"
	"github.com/s1onique/leamas/internal/ledger/store"
)

func splitLines(value string) []string { return strings.Split(value, "\n") }

func trimSpace(value string) string { return strings.TrimSpace(value) }

func runLedgerEvidence(args []string) int {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "ERROR: evidence requires a subcommand (add)")
		return 1
	}
	switch args[0] {
	case "add":
		return runLedgerEvidenceAdd(args[1:])
	default:
		fmt.Fprintf(os.Stderr, "unknown ledger evidence subcommand: %s\n", args[0])
		return 1
	}
}

func runLedgerEvidenceAdd(args []string) int {
	fs := flag.NewFlagSet("evidence add", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	root := fs.String("root", store.DefaultRoot, "ledger root directory")
	id := fs.String("id", "", "evidence identity (required)")
	kind := fs.String("kind", "file", "evidence kind")
	title := fs.String("title", "", "evidence title (required)")
	relPath := fs.String("relative-path", "", "repository-relative artifact path")
	digest := fs.String("sha256", "", "artifact SHA-256 digest")
	item := fs.String("item", "", "work item to bind the evidence to")
	quality := fs.String("quality", string(domain.EvidenceQualityDeclared), "evidence quality")
	expectedRev := fs.Int("expect-revision", 0, "expected revision of the bound work item")
	jsonOut := fs.Bool("json", false, "output JSON")
	if err := fs.Parse(args); err != nil {
		return 1
	}
	if *id == "" || *title == "" {
		fmt.Fprintln(os.Stderr, "ERROR: evidence add requires --id and --title")
		return 1
	}
	created, err := ledgerService(*root).AddEvidence("ledger-evidence", service.AddEvidenceOptions{
		EvidenceID:   domain.EvidenceID(*id),
		Kind:         *kind,
		Title:        *title,
		RelativePath: *relPath,
		SHA256:       *digest,
		ItemID:       domain.ScopedItemID(*item),
		Quality:      domain.EvidenceQuality(*quality),
		ExpectedRev:  *expectedRev,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
		return 1
	}
	if *jsonOut {
		emitJSON(map[string]any{"ok": true, "evidence": created})
		return 0
	}
	fmt.Printf("added evidence %s\n", created.EvidenceID)
	return 0
}

func runLedgerCheckpoint(args []string) int {
	fs := flag.NewFlagSet("checkpoint", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	root := fs.String("root", store.DefaultRoot, "ledger root directory")
	id := fs.String("id", "", "work item identity (required)")
	lifecycle := fs.String("lifecycle", string(domain.LifecycleInProgress), "new lifecycle state")
	summary := fs.String("summary", "", "checkpoint summary")
	expectedRev := fs.Int("expect-revision", 0, "expected revision (optimistic concurrency)")
	jsonOut := fs.Bool("json", false, "output JSON")
	if err := fs.Parse(args); err != nil {
		return 1
	}
	if *id == "" {
		fmt.Fprintln(os.Stderr, "ERROR: checkpoint requires --id")
		return 1
	}
	updated, err := ledgerService(*root).Checkpoint("ledger-checkpoint", service.CheckpointOptions{
		ScopedID:    domain.ScopedItemID(*id),
		ExpectedRev: *expectedRev,
		Lifecycle:   domain.LifecycleState(*lifecycle),
		Summary:     *summary,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
		return 1
	}
	if *jsonOut {
		emitJSON(map[string]any{"ok": true, "item": updated})
		return 0
	}
	fmt.Printf("checkpoint recorded on %s (revision %d)\n", updated.Scoped(), updated.Revision)
	return 0
}

func runLedgerClose(args []string) int {
	fs := flag.NewFlagSet("close", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	root := fs.String("root", store.DefaultRoot, "ledger root directory")
	actID := fs.String("act", "", "ACT execution record identity (required)")
	item := fs.String("item", "", "work item the ACT is bound to")
	title := fs.String("title", "", "ACT title")
	verdict := fs.String("verdict", string(domain.ActVerdictClosed), "terminal ACT verdict")
	note := fs.String("note", "", "operator note")
	disposition := fs.String("terminal-disposition", "", "preserved terminal disposition")
	jsonOut := fs.Bool("json", false, "output JSON")
	if err := fs.Parse(args); err != nil {
		return 1
	}
	if *actID == "" {
		fmt.Fprintln(os.Stderr, "ERROR: close requires --act")
		return 1
	}
	closed, err := ledgerService(*root).CloseAct("ledger-close", service.CloseActOptions{
		ActID:               domain.ActID(*actID),
		ItemID:              domain.ScopedItemID(*item),
		Title:               *title,
		Verdict:             domain.ActVerdict(*verdict),
		Note:                *note,
		TerminalDisposition: *disposition,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
		return 1
	}
	if *jsonOut {
		emitJSON(map[string]any{"ok": true, "act": closed})
		return 0
	}
	fmt.Printf("closed %s with verdict %s\n", closed.ActID, closed.Verdict)
	return 0
}

func runLedgerArchive(args []string) int {
	fs := flag.NewFlagSet("archive", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	root := fs.String("root", store.DefaultRoot, "ledger root directory")
	id := fs.String("id", "", "work item identity (required)")
	reopen := fs.String("reopen-condition", "", "reopen condition (repeatable)")
	expectedRev := fs.Int("expect-revision", 0, "expected revision (optimistic concurrency)")
	jsonOut := fs.Bool("json", false, "output JSON")
	if err := fs.Parse(args); err != nil {
		return 1
	}
	if *id == "" {
		fmt.Fprintln(os.Stderr, "ERROR: archive requires --id")
		return 1
	}
	var conditions []domain.ReopenCondition
	for _, c := range stringList(*reopen) {
		conditions = append(conditions, domain.ReopenCondition{Kind: c})
	}
	archived, err := ledgerService(*root).Archive("ledger-archive", service.ArchiveOptions{
		ScopedID:         domain.ScopedItemID(*id),
		ExpectedRev:      *expectedRev,
		ReopenConditions: conditions,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
		return 1
	}
	if *jsonOut {
		emitJSON(map[string]any{"ok": true, "item": archived})
		return 0
	}
	fmt.Printf("archived %s\n", archived.Scoped())
	return 0
}

func runLedgerRestore(args []string) int {
	fs := flag.NewFlagSet("restore", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	root := fs.String("root", store.DefaultRoot, "ledger root directory")
	id := fs.String("id", "", "work item identity (required)")
	expectedRev := fs.Int("expect-revision", 0, "expected revision (optimistic concurrency)")
	jsonOut := fs.Bool("json", false, "output JSON")
	if err := fs.Parse(args); err != nil {
		return 1
	}
	if *id == "" {
		fmt.Fprintln(os.Stderr, "ERROR: restore requires --id")
		return 1
	}
	restored, err := ledgerService(*root).Restore("ledger-restore", service.RestoreOptions{
		ScopedID:    domain.ScopedItemID(*id),
		ExpectedRev: *expectedRev,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
		return 1
	}
	if *jsonOut {
		emitJSON(map[string]any{"ok": true, "item": restored})
		return 0
	}
	fmt.Printf("restored %s\n", restored.Scoped())
	return 0
}
