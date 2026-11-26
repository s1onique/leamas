package store

import (
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

// crashHelperSource is the helper program of the process-termination
// witness.
//
// It publishes the FIRST record of a two-record transaction, signals that
// it did so and then waits forever, so the parent can terminate it.
const crashHelperSource = `package main

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
`

// buildCrashHelper builds the helper program of the process-termination
// witness.
func buildCrashHelper(t *testing.T) string {
	t.Helper()
	pkgDir := filepath.Join("internal", "ledger", "store", "crashtesthelper")
	if err := os.MkdirAll(pkgDir, 0o755); err != nil {
		t.Fatalf("mkdir helper pkg: %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(pkgDir) })
	if err := os.WriteFile(filepath.Join(pkgDir, "main.go"), []byte(crashHelperSource), 0o644); err != nil {
		t.Fatalf("write helper: %v", err)
	}
	bin := filepath.Join(t.TempDir(), "crashhelper")
	build := exec.Command("go", "build", "-o", bin, "./internal/ledger/store/crashtesthelper")
	build.Dir = "."
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build helper: %v\n%s", err, out)
	}
	return bin
}

// waitForFile waits until the given file exists.
func waitForFile(t *testing.T, path string, budget time.Duration) {
	t.Helper()
	deadline := time.Now().Add(budget)
	for time.Now().Before(deadline) {
		if _, err := os.Stat(path); err == nil {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("the helper did not signal readiness within %s", budget)
}

// TestWitnessConcurrentSubprocessWritersAreMutuallyExclusive proves that
// two concurrent SUBPROCESS writers cannot both hold the exclusive lock.
//
// The witness runs the real storage code in real subprocesses.
func TestWitnessConcurrentSubprocessWritersAreMutuallyExclusive(t *testing.T) {
	if testing.Short() {
		t.Skip("the subprocess witness is skipped in short mode")
	}
	root := filepath.Join(t.TempDir(), "ledger")
	s := New(root)
	if err := s.Init(); err != nil {
		t.Fatalf("init: %v", err)
	}
	if err := WriteFileAtomic(filepath.Join(root, "items", "EPIC-A__ITEM-1.json"),
		mustEncode(t, amendmentItem("ITEM-1", 1))); err != nil {
		t.Fatalf("seed: %v", err)
	}

	bin := buildWriterHelper(t)
	defer os.RemoveAll(filepath.Dir(bin))

	// The two helpers start at the same time. Each publishes one
	// conflicting update of the same record.
	start := filepath.Join(t.TempDir(), "go")
	var wg sync.WaitGroup
	results := make([]string, 2)
	codes := make([]int, 2)
	for i, title := range []string{"writer-a", "writer-b"} {
		wg.Add(1)
		go func(i int, title string) {
			defer wg.Done()
			cmd := exec.Command(bin, root, title, start)
			out, err := cmd.CombinedOutput()
			results[i] = string(out)
			if err != nil {
				if ee, ok := err.(*exec.ExitError); ok {
					codes[i] = ee.ExitCode()
					return
				}
				codes[i] = -1
				return
			}
			codes[i] = 0
		}(i, title)
	}
	wg.Wait()

	committed := 0
	for i := range results {
		if codes[i] == 0 {
			committed++
		}
	}
	if committed != 1 {
		t.Fatalf("expected exactly one subprocess writer to publish, got %d (results: %v codes: %v)",
			committed, results, codes)
	}

	// The published state is one of the two conflicting updates, not a mix.
	l, err := s.Load()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	item, _ := l.ItemByScoped("EPIC-A/ITEM-1")
	if item.Title != "writer-a" && item.Title != "writer-b" {
		t.Fatalf("the published state is not a single writer's update: %q", item.Title)
	}
}

// writerHelperSource is the helper program of the concurrent-subprocess
// witness.
//
// It waits for the shared start marker, so both helpers contend for the
// exclusive lock at the same moment.
const writerHelperSource = `package main

import (
	"fmt"
	"os"
	"time"

	"github.com/s1onique/leamas/internal/ledger/domain"
	"github.com/s1onique/leamas/internal/ledger/store"
)

func main() {
	root := os.Args[1]
	title := os.Args[2]
	start := os.Args[3]
	for i := 0; i < 2000; i++ {
		if _, err := os.Stat(start); err == nil {
			break
		}
		time.Sleep(5 * time.Millisecond)
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
`

// buildWriterHelper builds the helper program of the concurrent-subprocess
// witness.
func buildWriterHelper(t *testing.T) string {
	t.Helper()
	pkgDir := filepath.Join("internal", "ledger", "store", "writertesthelper")
	if err := os.MkdirAll(pkgDir, 0o755); err != nil {
		t.Fatalf("mkdir helper pkg: %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(pkgDir) })
	if err := os.WriteFile(filepath.Join(pkgDir, "main.go"), []byte(writerHelperSource), 0o644); err != nil {
		t.Fatalf("write helper: %v", err)
	}
	bin := filepath.Join(t.TempDir(), "writerhelper")
	build := exec.Command("go", "build", "-o", bin, "./internal/ledger/store/writertesthelper")
	build.Dir = "."
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build helper: %v\n%s", err, out)
	}
	return bin
}
