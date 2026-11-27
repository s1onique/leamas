// Command writertesthelper is the tracked fixture of the concurrent
// subprocess writer witness.
//
// It publishes a readiness marker, waits for the shared start marker and
// only then attempts to commit a conflicting revision of the same record.
// The start marker is mandatory: a missing marker is an explicit failure,
// never a silent continuation.
package main

import (
	"fmt"
	"os"
	"time"

	"github.com/s1onique/leamas/internal/ledger/domain"
	"github.com/s1onique/leamas/internal/ledger/store"
)

// startBudget bounds how long the helper waits for the start marker.
const startBudget = 30 * time.Second

func main() {
	root := os.Args[1]
	title := os.Args[2]
	ready := os.Args[3]
	start := os.Args[4]

	if err := os.WriteFile(ready, []byte("ready"), 0o644); err != nil {
		fmt.Println("READY-ERR", err)
		os.Exit(6)
	}
	if !waitFor(start) {
		fmt.Println("START-MISSING", start)
		os.Exit(7)
	}

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

// waitFor reports whether the start marker appeared within the budget.
func waitFor(path string) bool {
	deadline := time.Now().Add(startBudget)
	for time.Now().Before(deadline) {
		if _, err := os.Stat(path); err == nil {
			return true
		}
		time.Sleep(5 * time.Millisecond)
	}
	return false
}
