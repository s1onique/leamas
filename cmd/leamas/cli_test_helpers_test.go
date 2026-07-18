package main

import (
	"bytes"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"testing"

	"github.com/s1onique/leamas/internal/execution"
)

// withoutLeamasEnv returns a copy of the process environment with
// the Leamas re-entry markers removed. This prevents test subprocesses
// from being rejected as nested executions.
func withoutLeamasEnv() []string {
	blocked := map[string]struct{}{
		execution.EnvRootID:     {},
		execution.EnvParentPID:  {},
		execution.EnvGeneration: {},
	}

	env := os.Environ()
	out := make([]string, 0, len(env))

	for _, entry := range env {
		key, _, ok := strings.Cut(entry, "=")
		if !ok {
			continue
		}
		if _, remove := blocked[key]; !remove {
			out = append(out, entry)
		}
	}

	return out
}

// threadSafeBuffer wraps a bytes.Buffer with a mutex so concurrent
// reads (from ReadFrom) and writes are safe. It is used by the
// captureStdoutStderr helper to drain the pipe readers in goroutines
// while the captured function may also write to it.
type threadSafeBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *threadSafeBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *threadSafeBuffer) ReadFrom(r io.Reader) (int64, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.ReadFrom(r)
}

func (b *threadSafeBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

// captureStdoutStderr swaps os.Stdout and os.Stderr with in-memory pipes for
// the duration of fn, returning whatever was written to each stream. The
// helper restores the original streams even if fn panics.
//
// Tests that touch package-level globals (os.Stdout, os.Stderr) MUST NOT be
// marked t.Parallel(); the swap is process-wide and would race with other
// tests that capture output.
func captureStdoutStderr(t *testing.T, fn func()) (stdout, stderr string) {
	t.Helper()
	stdout, stderr, _ = captureWithCode(t, func() int { fn(); return 0 })
	return stdout, stderr
}

// captureWithCode is the int-returning variant used by tests that need to
// inspect the exit code produced by the command under capture. Internally it
// shares the same pipe plumbing as captureStdoutStderr.
func captureWithCode(t *testing.T, fn func() int) (stdout, stderr string, code int) {
	t.Helper()

	origOut := os.Stdout
	origErr := os.Stderr

	outR, outW, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe stdout: %v", err)
	}
	errR, errW, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe stderr: %v", err)
	}

	os.Stdout = outW
	os.Stderr = errW

	var outBuf, errBuf threadSafeBuffer
	outDone := readPipeInto(outR, &outBuf)
	errDone := readPipeInto(errR, &errBuf)

	// Run the captured function with the swapped streams in place. The
	// inner closure captures the named `code` variable, so the captured
	// function can populate it through a closure.
	var capturedCode int
	func() {
		defer func() {
			os.Stdout = origOut
			os.Stderr = origErr
		}()
		capturedCode = fn()
	}()

	// Closing the writer ends the reader side and lets readPipeInto return.
	if err := outW.Close(); err != nil {
		t.Logf("close stdout writer: %v", err)
	}
	if err := errW.Close(); err != nil {
		t.Logf("close stderr writer: %v", err)
	}

	<-outDone
	<-errDone

	// Defensive: restore globals even if they were not yet restored above.
	os.Stdout = origOut
	os.Stderr = origErr

	return outBuf.String(), errBuf.String(), capturedCode
}

// readPipeInto drains r into buf in a goroutine and signals completion on the
// returned channel.
func readPipeInto(r io.Reader, buf *threadSafeBuffer) <-chan struct{} {
	done := make(chan struct{})
	go func() {
		defer close(done)
		_, _ = buf.ReadFrom(r)
	}()
	return done
}

// listJSONFiles returns the sorted relative paths of every .json file
// under root. Used by the no-filesystem-writes characterization tests.
func listJSONFiles(t *testing.T, root string) []string {
	t.Helper()
	var paths []string
	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d == nil || d.IsDir() {
			return nil
		}
		if strings.HasSuffix(p, ".json") {
			rel, relErr := filepath.Rel(root, p)
			if relErr == nil {
				paths = append(paths, rel)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk %s: %v", root, err)
	}
	sort.Strings(paths)
	return paths
}

// stringSlicesEqual returns true when a and b contain the same elements in
// the same order.
func stringSlicesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
