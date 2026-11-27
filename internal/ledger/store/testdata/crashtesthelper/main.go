// Command crashtesthelper is the tracked fixture of the process-termination
// witness.
//
// It publishes the FIRST record of a two-record transaction, signals that
// it did so and then waits forever, so the parent can terminate it.
package main

import (
	"os"
	"time"

	"github.com/s1onique/leamas/internal/ledger/store"
)

func main() {
	root := os.Args[1]
	ready := os.Args[2]
	s := store.New(root)
	txn, err := s.Begin("tx-crash")
	if err != nil {
		os.Exit(3)
	}
	txn.Write("items/EPIC-A__ITEM-A.json", []byte("{\"partial\":true}\n"))
	txn.Write("items/EPIC-A__ITEM-B.json", []byte("{\"created\":true}\n"))
	if err := txn.PublishPartial(); err != nil {
		os.Exit(4)
	}
	if err := os.WriteFile(ready, []byte("ready"), 0o644); err != nil {
		os.Exit(5)
	}
	for {
		time.Sleep(time.Second)
	}
}
