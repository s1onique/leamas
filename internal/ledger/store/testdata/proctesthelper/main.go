// Command proctesthelper is the tracked fixture of the concurrent-process
// amendment witness.
//
// It publishes one conflicting update of the same record.
package main

import (
	"fmt"
	"os"

	"github.com/s1onique/leamas/internal/ledger/domain"
	"github.com/s1onique/leamas/internal/ledger/store"
)

func main() {
	root := os.Args[1]
	title := os.Args[2]
	s := store.New(root)
	txn, err := s.BeginMutation("tx-" + title)
	if err != nil {
		fmt.Println("BEGIN-ERR", err)
		os.Exit(3)
	}
	it := domain.WorkItem{
		SchemaVersion: domain.LedgerSchemaVersion,
		EpicID:        domain.EpicID("EPIC-A"),
		ItemID:        domain.ItemID("ITEM-1"),
		Title:         title,
		Kind:          domain.ItemKindTask,
		Lifecycle:     domain.LifecycleOpen,
		Acceptance:    domain.AcceptancePending,
		Revision:      2,
	}
	if err := txn.PutItem(it); err != nil {
		fmt.Println("PUT-ERR", err)
		os.Exit(4)
	}
	if err := txn.Commit(); err != nil {
		fmt.Println("COMMIT-ERR", err)
		os.Exit(5)
	}
	fmt.Println("COMMITTED", title)
}
