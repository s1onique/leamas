package store

import (
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/s1onique/leamas/internal/ledger/domain"
)

// TestAmendmentConcurrentProcessesCannotBothPublish reproduces witness 1
// at the PROCESS seam: two concurrent processes that share an expected
// revision cannot both publish conflicting updates.
//
// The witness runs the real storage code in real subprocesses.
func TestAmendmentConcurrentProcessesCannotBothPublish(t *testing.T) {
	if testing.Short() {
		t.Skip("subprocess witness skipped in short mode")
	}
	root := filepath.Join(t.TempDir(), "ledger")
	seed := amendmentItem("ITEM-1", 1)
	if err := New(root).Init(); err != nil {
		t.Fatalf("init: %v", err)
	}
	if err := WriteFileAtomic(filepath.Join(root, RelPathFor("items", "EPIC-A__ITEM-1.json")), mustEncode(t, seed)); err != nil {
		t.Fatalf("seed: %v", err)
	}

	// The helper subprocess publishes one conflicting update. Both
	// processes use the SAME expected revision.
	//
	// The helper source is a TRACKED fixture under testdata/. It is
	// compiled into a temporary directory, so no Go source is generated
	// inside the repository.
	bin := buildFixture(t, "proctesthelper")

	var wg sync.WaitGroup
	results := make([]string, 2)
	codes := make([]int, 2)
	for i, title := range []string{"writer-a", "writer-b"} {
		wg.Add(1)
		go func(i int, title string) {
			defer wg.Done()
			cmd := exec.Command(bin, root, title)
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
		if strings.Contains(results[i], "COMMITTED") {
			committed++
		}
	}
	if committed != 1 {
		t.Fatalf("expected exactly one writer to publish, got %d (results: %v)", committed, results)
	}

	// The published state must be one of the two conflicting updates, not
	// a mix.
	l, err := New(root).Load()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	item, _ := l.ItemByScoped("EPIC-A/ITEM-1")
	if item.Title != "writer-a" && item.Title != "writer-b" {
		t.Fatalf("the published state is not a single writer's update: %q", item.Title)
	}
}

// TestAmendmentReaderNeverSeesPartialPublication reproduces witness 2: a
// reader cannot accept partial multi-record publication.
//
// The transaction publishes several records. A concurrent reader either
// sees the state before the transaction or the state after it, never a
// mixture.
func TestAmendmentReaderNeverSeesPartialPublication(t *testing.T) {
	root := filepath.Join(t.TempDir(), "ledger")
	s := New(root)
	if err := s.Init(); err != nil {
		t.Fatalf("init: %v", err)
	}

	const records = 6
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		txn, err := s.BeginMutation("tx-multi")
		if err != nil {
			return
		}
		for i := 0; i < records; i++ {
			id := domain.ItemID("ITEM-" + string(rune('A'+i)))
			if err := txn.PutItem(amendmentItem(string(id), 1)); err != nil {
				return
			}
		}
		_ = txn.Commit()
	}()

	// The reader observes the canonical state repeatedly. Every
	// observation must be internally consistent: the set of published
	// records is either empty or complete.
	for i := 0; i < 200; i++ {
		l, err := s.Load()
		if err != nil {
			continue
		}
		if len(l.Items) != 0 && len(l.Items) != records {
			t.Fatalf("a reader observed a partial publication: %d of %d records",
				len(l.Items), records)
		}
	}
	wg.Wait()

	l, err := s.Load()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if len(l.Items) != records {
		t.Fatalf("expected %d records after the transaction, got %d", records, len(l.Items))
	}
}
