package main

import (
	"bytes"
	"encoding/json"
	"io"
	"strings"
	"sync"
	"testing"

	"github.com/s1onique/leamas/internal/factory/closure"
)

// validateInvocationResult captures the full output of one validate
// handler invocation for non-vacuous determinism checks.
type validateInvocationResult struct {
	Exit   int
	Stdout []byte
	Stderr []byte
}

func makeValidateDepsWithInput(input string) planValidateDeps {
	return planValidateDeps{
		openFile: func(path string) (io.ReadCloser, error) {
			return &closedReadCloserOK{r: strings.NewReader(input)}, nil
		},
		readBounded: func(r io.Reader, max int64) ([]byte, error) {
			return []byte(input), nil
		},
	}
}

// TestValidateDeterminismValid20Sequential proves deterministic
// valid input produces exit 0 and identical stdout.
func TestValidateDeterminismValid20Sequential(t *testing.T) {
	example := closure.DescriptorExample()
	data, _ := json.Marshal(example)
	deps := makeValidateDepsWithInput(string(data))
	results := make([]validateInvocationResult, 20)
	for i := 0; i < 20; i++ {
		var stdout, stderr bytes.Buffer
		exit := runFactoryClosePlanValidateWith([]string{"--file", "x"}, &bytes.Buffer{}, &stdout, &stderr, deps)
		results[i] = validateInvocationResult{
			Exit:   exit,
			Stdout: bytes.Clone(stdout.Bytes()),
			Stderr: bytes.Clone(stderr.Bytes()),
		}
	}
	for i, r := range results {
		if r.Exit != 0 {
			t.Fatalf("run %d exit = %d, want 0", i, r.Exit)
		}
		if len(r.Stderr) != 0 {
			t.Fatalf("run %d stderr not empty: %q", i, r.Stderr)
		}
		if len(r.Stdout) == 0 {
			t.Fatalf("run %d wrote nothing to stdout", i)
		}
	}
	for i := 1; i < 20; i++ {
		if !bytes.Equal(results[i].Stdout, results[0].Stdout) {
			t.Fatalf("run %d stdout differs from run 0", i)
		}
	}
}

// TestValidateDeterminismInvalid20Sequential proves deterministic
// invalid input produces exit 1 and identical stdout.
func TestValidateDeterminismInvalid20Sequential(t *testing.T) {
	deps := makeValidateDepsWithInput(`{"invalid": true}`)
	results := make([]validateInvocationResult, 20)
	for i := 0; i < 20; i++ {
		var stdout, stderr bytes.Buffer
		exit := runFactoryClosePlanValidateWith([]string{"--file", "x"}, &bytes.Buffer{}, &stdout, &stderr, deps)
		results[i] = validateInvocationResult{
			Exit:   exit,
			Stdout: bytes.Clone(stdout.Bytes()),
			Stderr: bytes.Clone(stderr.Bytes()),
		}
	}
	for i, r := range results {
		if r.Exit != 1 {
			t.Fatalf("invalid run %d exit = %d, want 1", i, r.Exit)
		}
		if len(r.Stdout) == 0 {
			t.Fatalf("invalid run %d wrote nothing to stdout", i)
		}
	}
	for i := 1; i < 20; i++ {
		if !bytes.Equal(results[i].Stdout, results[0].Stdout) {
			t.Fatalf("invalid run %d stdout differs from run 0", i)
		}
	}
}

// TestValidateDeterminism8Concurrent proves concurrent determinism.
func TestValidateDeterminism8Concurrent(t *testing.T) {
	example := closure.DescriptorExample()
	data, _ := json.Marshal(example)
	deps := makeValidateDepsWithInput(string(data))
	results := make([]validateInvocationResult, 8)
	var wg sync.WaitGroup
	var mu sync.Mutex
	start := make(chan struct{})
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			<-start
			var stdout, stderr bytes.Buffer
			exit := runFactoryClosePlanValidateWith([]string{"--file", "x"}, &bytes.Buffer{}, &stdout, &stderr, deps)
			mu.Lock()
			results[idx] = validateInvocationResult{
				Exit:   exit,
				Stdout: bytes.Clone(stdout.Bytes()),
				Stderr: bytes.Clone(stderr.Bytes()),
			}
			mu.Unlock()
		}(i)
	}
	close(start)
	wg.Wait()
	for i, r := range results {
		if r.Exit != 0 {
			t.Fatalf("concurrent run %d exit = %d, want 0", i, r.Exit)
		}
		if len(r.Stderr) != 0 {
			t.Fatalf("concurrent run %d stderr not empty: %q", i, r.Stderr)
		}
		if len(r.Stdout) == 0 {
			t.Fatalf("concurrent run %d wrote nothing to stdout", i)
		}
	}
	for i := 1; i < 8; i++ {
		if !bytes.Equal(results[i].Stdout, results[0].Stdout) {
			t.Fatalf("concurrent run %d stdout differs from run 0", i)
		}
	}
}
