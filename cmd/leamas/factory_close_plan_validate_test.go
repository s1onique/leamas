package main

import (
	"bytes"
	"errors"
	"io"
	"strings"
	"sync"
	"testing"

	"github.com/s1onique/leamas/internal/factory/closure"
)

// errReader always returns error.
type errReader struct{}

func (errReader) Read(p []byte) (n int, err error) {
	return 0, errors.New("unexpected error")
}

// partialWriter writes only the first byte.
type partialWriter struct {
	written int
}

func (p *partialWriter) Write(b []byte) (n int, err error) {
	if len(b) == 0 {
		return 0, nil
	}
	p.written = 1
	return 1, nil
}

// errorWriter always returns error.
type errorWriter struct{}

func (errorWriter) Write(b []byte) (n int, err error) {
	return 0, errors.New("closed pipe")
}

// closedReadCloser always returns error on close.
type closedReadCloser struct {
	r io.Reader
}

func (c *closedReadCloser) Read(p []byte) (n int, err error) {
	return c.r.Read(p)
}
func (c *closedReadCloser) Close() error {
	return errors.New("closed pipe")
}

// closedReadCloserOK never fails on close.
type closedReadCloserOK struct {
	r        io.Reader
	closeErr error
}

func (c *closedReadCloserOK) Read(p []byte) (n int, err error) {
	return c.r.Read(p)
}
func (c *closedReadCloserOK) Close() error {
	return c.closeErr
}

func TestValidateHelpSolo(t *testing.T) {
	var stdout, stderr bytes.Buffer
	exit := runFactoryClosePlanValidate([]string{"-h"}, &stdout, &stderr)
	if exit != 0 {
		t.Fatalf("help exit = %d, want 0", exit)
	}
	if stderr.Len() == 0 {
		t.Fatal("help wrote nothing to stderr")
	}
	if stdout.Len() != 0 {
		t.Fatalf("help wrote to stdout: %q", stdout.String())
	}
}

func TestValidateHelpLong(t *testing.T) {
	var stdout, stderr bytes.Buffer
	exit := runFactoryClosePlanValidate([]string{"--help"}, &stdout, &stderr)
	if exit != 0 {
		t.Fatalf("help exit = %d, want 0", exit)
	}
	if stderr.Len() == 0 {
		t.Fatal("help wrote nothing to stderr")
	}
	if stdout.Len() != 0 {
		t.Fatalf("help wrote to stdout: %q", stdout.String())
	}
}

func TestValidateHelpWithExtra(t *testing.T) {
	var stdout, stderr bytes.Buffer
	exit := runFactoryClosePlanValidate([]string{"-h", "extra"}, &stdout, &stderr)
	if exit != 2 {
		t.Fatalf("exit = %d, want 2", exit)
	}
}

func TestValidateHelpMidArg(t *testing.T) {
	var stdout, stderr bytes.Buffer
	exit := runFactoryClosePlanValidate([]string{"--file", "x", "-h"}, &stdout, &stderr)
	if exit != 2 {
		t.Fatalf("exit = %d, want 2", exit)
	}
}

func TestValidateRepeatedFile(t *testing.T) {
	var stdout, stderr bytes.Buffer
	exit := runFactoryClosePlanValidate([]string{"--file", "a", "--file", "b"}, &stdout, &stderr)
	if exit != 2 {
		t.Fatalf("exit = %d, want 2", exit)
	}
	if !strings.Contains(stderr.String(), "repeated --file") {
		t.Fatalf("stderr = %q, want repeated --file", stderr.String())
	}
}

func TestValidateRepeatedStdin(t *testing.T) {
	var stdout, stderr bytes.Buffer
	exit := runFactoryClosePlanValidate([]string{"--stdin", "--stdin"}, &stdout, &stderr)
	if exit != 2 {
		t.Fatalf("exit = %d, want 2", exit)
	}
	if !strings.Contains(stderr.String(), "repeated --stdin") {
		t.Fatalf("stderr = %q, want repeated --stdin", stderr.String())
	}
}

func TestValidateFileRequiresValue(t *testing.T) {
	var stdout, stderr bytes.Buffer
	exit := runFactoryClosePlanValidate([]string{"--file"}, &stdout, &stderr)
	if exit != 2 {
		t.Fatalf("exit = %d, want 2", exit)
	}
	if !strings.Contains(stderr.String(), "--file requires a value") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestValidateFileWithHelpValue(t *testing.T) {
	var stdout, stderr bytes.Buffer
	exit := runFactoryClosePlanValidate([]string{"--file", "-h"}, &stdout, &stderr)
	if exit != 2 {
		t.Fatalf("exit = %d, want 2", exit)
	}
	if !strings.Contains(stderr.String(), "--file requires a value") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestValidateUnknownFlag(t *testing.T) {
	var stdout, stderr bytes.Buffer
	exit := runFactoryClosePlanValidate([]string{"--unknown"}, &stdout, &stderr)
	if exit != 2 {
		t.Fatalf("exit = %d, want 2", exit)
	}
	if !strings.Contains(stderr.String(), "unknown flag") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestValidateMutualExclusivity(t *testing.T) {
	var stdout, stderr bytes.Buffer
	exit := runFactoryClosePlanValidate([]string{"--file", "a", "--stdin"}, &stdout, &stderr)
	if exit != 2 {
		t.Fatalf("exit = %d, want 2", exit)
	}
	if !strings.Contains(stderr.String(), "mutually exclusive") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestValidateRequiresInput(t *testing.T) {
	var stdout, stderr bytes.Buffer
	exit := runFactoryClosePlanValidate(nil, &stdout, &stderr)
	if exit != 2 {
		t.Fatalf("exit = %d, want 2", exit)
	}
	if !strings.Contains(stderr.String(), "one of --file or --stdin is required") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestValidateFileReadError(t *testing.T) {
	deps := planValidateDeps{
		openFile: func(path string) (io.ReadCloser, error) {
			return nil, errors.New("file not found")
		},
		readBounded: func(r io.Reader, max int64) ([]byte, error) {
			return nil, nil
		},
	}
	var stdout, stderr bytes.Buffer
	exit := runFactoryClosePlanValidateWith([]string{"--file", "nonexistent"}, nil, &stdout, &stderr, deps)
	if exit != 2 {
		t.Fatalf("exit = %d, want 2", exit)
	}
	if !strings.Contains(stderr.String(), "file read failed") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestValidateStdinBoundedRead(t *testing.T) {
	deps := planValidateDeps{
		openFile: func(path string) (io.ReadCloser, error) {
			return nil, nil
		},
		readBounded: func(r io.Reader, max int64) ([]byte, error) {
			if max != closure.MaxPlanBytes+1 {
				t.Errorf("max = %d, want %d", max, closure.MaxPlanBytes+1)
			}
			return []byte(`{"contract_version": 1, "act_id": "TEST"}`), nil
		},
	}
	var stdout, stderr bytes.Buffer
	exit := runFactoryClosePlanValidateWith([]string{"--stdin"}, &bytes.Buffer{}, &stdout, &stderr, deps)
	if exit != 1 {
		t.Fatalf("exit = %d, want 1 (invalid plan)", exit)
	}
}

func TestValidateMaxSizeExceeded(t *testing.T) {
	deps := planValidateDeps{
		openFile: func(path string) (io.ReadCloser, error) {
			return nil, nil
		},
		readBounded: func(r io.Reader, max int64) ([]byte, error) {
			// Return MaxPlanBytes + 2 bytes
			return make([]byte, closure.MaxPlanBytes+2), nil
		},
	}
	var stdout, stderr bytes.Buffer
	exit := runFactoryClosePlanValidateWith([]string{"--stdin"}, &bytes.Buffer{}, &stdout, &stderr, deps)
	if exit != 2 {
		t.Fatalf("exit = %d, want 2", exit)
	}
	if !strings.Contains(stderr.String(), "exceeds max size") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestValidateExactMaxSizeAccepted(t *testing.T) {
	deps := planValidateDeps{
		openFile: func(path string) (io.ReadCloser, error) {
			return nil, nil
		},
		readBounded: func(r io.Reader, max int64) ([]byte, error) {
			// Return exactly MaxPlanBytes
			return make([]byte, closure.MaxPlanBytes), nil
		},
	}
	var stdout, stderr bytes.Buffer
	exit := runFactoryClosePlanValidateWith([]string{"--stdin"}, &bytes.Buffer{}, &stdout, &stderr, deps)
	// Should not be rejected for size; may fail validation but not for size
	if exit == 2 && strings.Contains(stderr.String(), "exceeds max size") {
		t.Fatalf("MaxPlanBytes rejected: %s", stderr.String())
	}
}

func TestValidateFileCloseError(t *testing.T) {
	deps := planValidateDeps{
		openFile: func(path string) (io.ReadCloser, error) {
			return &closedReadCloserOK{
				r:        strings.NewReader(`{}`),
				closeErr: errors.New("closed pipe"),
			}, nil
		},
		readBounded: func(r io.Reader, max int64) ([]byte, error) {
			return io.ReadAll(r)
		},
	}
	var stdout, stderr bytes.Buffer
	exit := runFactoryClosePlanValidateWith([]string{"--file", "x"}, &bytes.Buffer{}, &stdout, &stderr, deps)
	if exit != 2 {
		t.Fatalf("exit = %d, want 2", exit)
	}
	if !strings.Contains(stderr.String(), "close failed") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestValidateAtomicWriteError(t *testing.T) {
	deps := planValidateDeps{
		openFile: func(path string) (io.ReadCloser, error) {
			return &closedReadCloserOK{r: strings.NewReader(`{}`)}, nil
		},
		readBounded: func(r io.Reader, max int64) ([]byte, error) {
			return []byte(`{"contract_version": 1}`), nil
		},
	}
	var stderr bytes.Buffer
	exit := runFactoryClosePlanValidateWith([]string{"--file", "x"}, &bytes.Buffer{}, &errorWriter{}, &stderr, deps)
	if exit != 2 {
		t.Fatalf("exit = %d, want 2", exit)
	}
}

func TestValidateAtomicWritePartial(t *testing.T) {
	deps := planValidateDeps{
		openFile: func(path string) (io.ReadCloser, error) {
			return &closedReadCloserOK{r: strings.NewReader(`{}`)}, nil
		},
		readBounded: func(r io.Reader, max int64) ([]byte, error) {
			return []byte(`{"contract_version": 1}`), nil
		},
	}
	var stderr bytes.Buffer
	exit := runFactoryClosePlanValidateWith([]string{"--file", "x"}, &bytes.Buffer{}, &partialWriter{}, &stderr, deps)
	if exit != 2 {
		t.Fatalf("exit = %d, want 2", exit)
	}
}

func TestValidateAtomicWriteZero(t *testing.T) {
	deps := planValidateDeps{
		openFile: func(path string) (io.ReadCloser, error) {
			return &closedReadCloserOK{r: strings.NewReader(`{}`)}, nil
		},
		readBounded: func(r io.Reader, max int64) ([]byte, error) {
			return []byte(`{"contract_version": 1}`), nil
		},
	}
	var stderr bytes.Buffer
	exit := runFactoryClosePlanValidateWith([]string{"--file", "x"}, &bytes.Buffer{}, &zeroWriter{}, &stderr, deps)
	if exit != 2 {
		t.Fatalf("exit = %d, want 2", exit)
	}
}

func TestValidateDeterminism20Sequential(t *testing.T) {
	deps := planValidateDeps{
		openFile: func(path string) (io.ReadCloser, error) {
			return &closedReadCloserOK{r: strings.NewReader(`{"contract_version": 1}`)}, nil
		},
		readBounded: func(r io.Reader, max int64) ([]byte, error) {
			return []byte(`{"contract_version": 1}`), nil
		},
	}
	results := make([]string, 20)
	for i := 0; i < 20; i++ {
		var stdout, stderr bytes.Buffer
		runFactoryClosePlanValidateWith([]string{"--file", "x"}, &bytes.Buffer{}, &stdout, &stderr, deps)
		results[i] = stdout.String()
	}
	for i := 1; i < 20; i++ {
		if results[i] != results[0] {
			t.Fatalf("run %d differs from run 0", i)
		}
	}
}

func TestValidateDeterminism8Concurrent(t *testing.T) {
	deps := planValidateDeps{
		openFile: func(path string) (io.ReadCloser, error) {
			return &closedReadCloserOK{r: strings.NewReader(`{"contract_version": 1}`)}, nil
		},
		readBounded: func(r io.Reader, max int64) ([]byte, error) {
			return []byte(`{"contract_version": 1}`), nil
		},
	}
	results := make([]string, 8)
	var wg sync.WaitGroup
	var mu sync.Mutex
	start := make(chan struct{})
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			<-start
			var stdout, stderr bytes.Buffer
			runFactoryClosePlanValidateWith([]string{"--file", "x"}, &bytes.Buffer{}, &stdout, &stderr, deps)
			mu.Lock()
			results[idx] = stdout.String()
			mu.Unlock()
		}(i)
	}
	close(start)
	wg.Wait()
	for i := 1; i < 8; i++ {
		if results[i] != results[0] {
			t.Fatalf("concurrent run %d differs from run 0", i)
		}
	}
}

// zeroWriter writes zero bytes.
type zeroWriter struct{}

func (z zeroWriter) Write(b []byte) (n int, err error) {
	return 0, nil
}
