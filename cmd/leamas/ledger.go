package main

import (
	"fmt"
	"io"
	"os"

	"github.com/s1onique/leamas/internal/ledger/service"
	"github.com/s1onique/leamas/internal/ledger/store"
)

// ledgerDispatcher routes the `leamas ledger` subcommands.
func runLedger(args []string) int {
	if len(args) < 1 {
		printLedgerUsage(os.Stderr)
		return 1
	}

	switch args[0] {
	case "init":
		return runLedgerInit(args[1:])
	case "prime":
		return runLedgerPrime(args[1:])
	case "list":
		return runLedgerList(args[1:])
	case "show":
		return runLedgerShow(args[1:])
	case "next":
		return runLedgerNext(args[1:])
	case "create":
		return runLedgerCreate(args[1:])
	case "update":
		return runLedgerUpdate(args[1:])
	case "claim":
		return runLedgerClaim(args[1:])
	case "link":
		return runLedgerLink(args[1:])
	case "evidence":
		return runLedgerEvidence(args[1:])
	case "checkpoint":
		return runLedgerCheckpoint(args[1:])
	case "close":
		return runLedgerClose(args[1:])
	case "accept":
		return runLedgerAccept(args[1:])
	case "transfer":
		return runLedgerTransfer(args[1:])
	case "reopen":
		return runLedgerReopen(args[1:])
	case "decision":
		return runLedgerDecision(args[1:])
	case "knowledge":
		return runLedgerKnowledge(args[1:])
	case "archive":
		return runLedgerArchive(args[1:])
	case "restore":
		return runLedgerRestore(args[1:])
	case "render":
		return runLedgerRender(args[1:])
	case "validate":
		return runLedgerValidate(args[1:])
	case "stats":
		return runLedgerStats(args[1:])
	case "import":
		return runLedgerImport(args[1:])
	case "import-original":
		return runLedgerImportOriginal(args[1:])
	case "migrate-board":
		return runLedgerMigrateBoard(args[1:])
	case "reconcile":
		return runLedgerReconcile(args[1:])
	case "recover":
		return runLedgerRecover(args[1:])
	case "--help", "-h":
		printLedgerUsage(os.Stdout)
		return 0
	default:
		fmt.Fprintf(os.Stderr, "unknown ledger subcommand: %s\n", args[0])
		printLedgerUsage(os.Stderr)
		return 1
	}
}

func printLedgerUsage(w io.Writer) {
	fmt.Fprintln(w, "Usage: leamas ledger <subcommand> [flags]")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Subcommands:")
	fmt.Fprintln(w, "  init                 Initialize the ledger in the repository")
	fmt.Fprintln(w, "  prime                Print the minimal agent bootstrap context")
	fmt.Fprintln(w, "  list                 List work items")
	fmt.Fprintln(w, "  show <item-id>       Show one work item")
	fmt.Fprintln(w, "  next                 Show the next ready work item")
	fmt.Fprintln(w, "  create               Create a work item")
	fmt.Fprintln(w, "  update               Update a work item")
	fmt.Fprintln(w, "  claim                Claim a work item")
	fmt.Fprintln(w, "  link                 Link two work items")
	fmt.Fprintln(w, "  evidence add         Add an evidence reference")
	fmt.Fprintln(w, "  checkpoint           Record a durable semantic checkpoint")
	fmt.Fprintln(w, "  close                Record a terminal ACT verdict")
	fmt.Fprintln(w, "  accept               Record an explicit engineering acceptance")
	fmt.Fprintln(w, "  transfer             Transfer the ownership of a work item")
	fmt.Fprintln(w, "  reopen               Explicitly reopen a work item")
	fmt.Fprintln(w, "  decision add|show    Register or look up a durable decision")
	fmt.Fprintln(w, "  knowledge add|show   Register or look up a knowledge reference")
	fmt.Fprintln(w, "  archive              Archive an eligible work item")
	fmt.Fprintln(w, "  restore              Restore an archived work item")
	fmt.Fprintln(w, "  render               Render a derived view")
	fmt.Fprintln(w, "  validate             Validate the canonical ledger state")
	fmt.Fprintln(w, "  stats                Print the ledger statistics")
	fmt.Fprintln(w, "  import               Import a source board (dry-run by default)")
	fmt.Fprintln(w, "  import-original      Import the original IECR board fixture (dry-run by default)")
	fmt.Fprintln(w, "  migrate-board        Migrate the actual historical board (dry-run by default)")
	fmt.Fprintln(w, "  reconcile            Report the import conservation receipt")
	fmt.Fprintln(w, "  recover              Repair an interrupted transaction")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Flags:")
	fmt.Fprintln(w, "  --root <path>    Ledger root directory (default: .leamas/ledger)")
	fmt.Fprintln(w, "  --json           Output JSON format")
	fmt.Fprintln(w, "  --help, -h       Show this help")
}

// ledgerService builds the service for the given root.
//
// The evidence artifacts are resolved against the current working
// directory: the ledger root is repository-relative, so the working
// directory is the repository root from the operator's point of view.
func ledgerService(root string) *service.Service {
	svc := service.New(store.New(root))
	if wd, err := os.Getwd(); err == nil {
		svc = svc.WithRepoRoot(wd)
	}
	return svc
}
