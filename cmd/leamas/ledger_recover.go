package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/s1onique/leamas/internal/ledger/store"
)

// runLedgerRecover repairs an interrupted multi-record transaction.
//
// The recovery is deterministic and idempotent: it restores every
// snapshot recorded in the transaction journal and removes the journal.
// A partial transaction is therefore never exposed as an accepted state.
func runLedgerRecover(args []string) int {
	fs := flag.NewFlagSet("recover", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	root := fs.String("root", store.DefaultRoot, "ledger root directory")
	jsonOut := fs.Bool("json", false, "output JSON")
	if err := fs.Parse(args); err != nil {
		return 1
	}
	status, err := ledgerService(*root).Recover()
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
		return 1
	}
	if *jsonOut {
		emitJSON(map[string]any{
			"ok":             true,
			"interrupted":    status.Interrupted,
			"rolled_back":    status.RolledBack,
			"tx_id":          status.TxID,
			"restored_files": status.RestoredFiles,
		})
		return 0
	}
	if !status.Interrupted {
		fmt.Println("no interrupted transaction found")
		return 0
	}
	fmt.Printf("rolled back interrupted transaction: %s\n", status.TxID)
	for _, f := range status.RestoredFiles {
		fmt.Printf("restored: %s\n", f)
	}
	return 0
}
