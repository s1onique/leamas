// Command writertesthelper is the tracked fixture of the concurrent
// subprocess writer witness.
//
// It publishes a readiness marker, waits for the shared start marker and
// only then attempts to commit a conflicting revision of the same record.
// The start marker is mandatory: a missing marker is an explicit failure,
// never a silent continuation.
package main

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/s1onique/leamas/internal/ledger/domain"
	"github.com/s1onique/leamas/internal/ledger/store"
)

// startBudget bounds how long the helper waits for the start marker.
const startBudget = 30 * time.Second

// The exit codes of the helper. The witness distinguishes an EXPECTED
// conflict from an UNEXPECTED failure, so the conflict has its own code.
const (
	// exitOK means the helper published its update.
	exitOK = 0
	// exitReadyFailed means the readiness marker could not be written.
	exitReadyFailed = 6
	// exitStartMissing means the shared start marker never appeared.
	exitStartMissing = 7
	// exitConflict means the commit was rejected by the revision guard.
	exitConflict = 8
	// exitBeginFailed means the mutation could not be started.
	exitBeginFailed = 3
	// exitPutFailed means the staged record could not be written.
	exitPutFailed = 4
	// exitCommitFailed means the commit failed for an UNEXPECTED reason.
	exitCommitFailed = 5
)

func main() {
	root := os.Args[1]
	title := os.Args[2]
	ready := os.Args[3]
	start := os.Args[4]

	if err := os.WriteFile(ready, []byte("ready"), 0o644); err != nil {
		fmt.Println("READY-ERR", err)
		os.Exit(exitReadyFailed)
	}
	if !waitFor(start) {
		fmt.Println("START-MISSING", start)
		os.Exit(exitStartMissing)
	}

	s := store.New(root)
	txn, err := s.BeginMutation("tx-" + title)
	if err != nil {
		fmt.Println("BEGIN-ERR", err)
		os.Exit(exitBeginFailed)
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
		os.Exit(exitPutFailed)
	}
	if err := txn.Commit(); err != nil {
		// The revision conflict is the EXPECTED outcome for the writer that
		// loses the race. It has its own exit code, so the witness can tell
		// an expected conflict apart from an arbitrary failure.
		if errors.Is(err, store.ErrStaleRevision) {
			fmt.Println("CONFLICT", err)
			os.Exit(exitConflict)
		}
		fmt.Println("COMMIT-ERR", err)
		os.Exit(exitCommitFailed)
	}
	fmt.Println("COMMITTED", title)
	os.Exit(exitOK)
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
