package store

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// The exit codes of the tracked subprocess fixtures. They mirror the
// constants of the fixture helpers and distinguish an EXPECTED outcome from
// an UNEXPECTED failure.
const (
	// helperExitStartMissing means the helper never observed the shared
	// start marker.
	helperExitStartMissing = 7
	// helperExitConflict means the helper was rejected by the revision
	// guard, which is the EXPECTED outcome of the losing writer.
	helperExitConflict = 8
)

// buildFixture compiles a tracked helper fixture into a unique temporary
// directory.
//
// The fixture source lives under testdata/ and is compiled by an explicit
// package-relative source path, so no Go source is ever generated inside the
// repository.
func buildFixture(t *testing.T, name string) string {
	t.Helper()
	bin := filepath.Join(t.TempDir(), name)
	src := filepath.Join("testdata", name, "main.go")
	build := exec.Command("go", "build", "-o", bin, src)
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build fixture %s: %v\n%s", name, err, out)
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

	bin := buildFixture(t, "writertesthelper")

	// The two helpers contend for the exclusive lock at the same moment.
	//
	// The barrier is explicit: each helper publishes its own readiness
	// marker, the parent waits for BOTH markers and only then publishes the
	// shared start marker. A helper that never observes the start marker
	// fails explicitly instead of proceeding silently.
	dir := t.TempDir()
	start := filepath.Join(dir, "go")
	ready := []string{filepath.Join(dir, "ready-a"), filepath.Join(dir, "ready-b")}
	titles := []string{"writer-a", "writer-b"}
	cmds := make([]*exec.Cmd, len(titles))
	// The output destinations are configured BEFORE Start: a nil Stdout or
	// Stderr sends the subprocess output to the null device, and assigning
	// the destination afterwards cannot recover it.
	outputs := make([]bytes.Buffer, len(titles))
	for i := range titles {
		cmd := exec.Command(bin, root, titles[i], ready[i], start)
		cmd.Stdout = &outputs[i]
		cmd.Stderr = &outputs[i]
		if err := cmd.Start(); err != nil {
			t.Fatalf("start helper %s: %v", titles[i], err)
		}
		cmds[i] = cmd
	}
	for i := range ready {
		waitForFile(t, ready[i], 30*time.Second)
	}
	if err := os.WriteFile(start, []byte("go"), 0o644); err != nil {
		t.Fatalf("publish the start marker: %v", err)
	}

	results := make([]string, len(cmds))
	codes := make([]int, len(cmds))
	for i := range cmds {
		err := cmds[i].Wait()
		results[i] = outputs[i].String()
		switch {
		case err == nil:
			codes[i] = 0
		case errors.As(err, new(*exec.ExitError)):
			codes[i] = err.(*exec.ExitError).ExitCode()
		default:
			codes[i] = -1
		}
	}
	for i := range codes {
		if codes[i] == helperExitStartMissing {
			t.Fatalf("helper %s never observed the start marker: %q", titles[i], results[i])
		}
	}

	// The witness requires EXACTLY ONE successful publication and EXACTLY
	// ONE expected revision conflict. An arbitrary process failure must not
	// satisfy the witness: it would prove nothing about the exclusion.
	committed := 0
	conflicted := 0
	for i := range codes {
		switch codes[i] {
		case 0:
			committed++
		case helperExitConflict:
			conflicted++
			if !strings.Contains(results[i], "CONFLICT") {
				t.Fatalf("helper %s reported a conflict without the conflict receipt: %q",
					titles[i], results[i])
			}
		default:
			t.Fatalf("helper %s failed unexpectedly with code %d: %q",
				titles[i], codes[i], results[i])
		}
	}
	if committed != 1 {
		t.Fatalf("expected exactly one subprocess writer to publish, got %d (results: %v codes: %v)",
			committed, results, codes)
	}
	if conflicted != 1 {
		t.Fatalf("expected exactly one subprocess writer to be rejected by the revision guard, got %d (results: %v codes: %v)",
			conflicted, results, codes)
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
