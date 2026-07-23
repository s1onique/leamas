package main

import (
	"bytes"
	"errors"
	"io"
	"testing"
)

// shortWriter accepts a prefix of the slice and returns 1 byte plus
// io.ErrShortWrite. It is used to verify that the checked write
// helper detects partial-success from a destination that returns
// non-zero bytes and a non-nil error from the same Write call.
type shortWriter struct {
	err error
}

func (w *shortWriter) Write(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}
	// Pretend we accepted the first byte and refused the rest.
	return 1, io.ErrShortWrite
}

// TestCLIListWriteFailureFailsClosed asserts that `schema list`
// propagates a writer failure with a non-zero exit code and a
// diagnostic on stderr. The destination may have observed a prefix
// of the table before reporting failure; the contract documents
// that fact.
func TestCLIListWriteFailureFailsClosed(t *testing.T) {
	out := &failingWriter{err: errors.New("disk full")}
	errBuf := &bytes.Buffer{}
	code := newGateSummarySchemaCLI(out, errBuf).Run([]string{"list"})
	if code == 0 {
		t.Fatalf("list writer failure must exit non-zero")
	}
	if out.n == 0 {
		t.Fatalf("list writer was never invoked")
	}
	if errBuf.String() == "" {
		t.Fatalf("list writer failure diagnostic must reach stderr")
	}
}

// TestCLIListShortWriteFailsClosed asserts that `schema list`
// detects a destination that reports a short write (accepts a
// prefix and returns an error) and returns a non-zero exit code.
func TestCLIListShortWriteFailsClosed(t *testing.T) {
	out := &shortWriter{err: io.ErrShortWrite}
	errBuf := &bytes.Buffer{}
	code := newGateSummarySchemaCLI(out, errBuf).Run([]string{"list"})
	if code == 0 {
		t.Fatalf("list short write must exit non-zero")
	}
	if errBuf.String() == "" {
		t.Fatalf("list short-write diagnostic must reach stderr")
	}
}

// TestCLIShowShortWriteFails asserts that `schema show <version>`
// detects a short write and returns a non-zero exit code. The
// destination may have observed a prefix of the schema bytes before
// reporting failure; the contract documents that fact.
func TestCLIShowShortWriteFails(t *testing.T) {
	out := &shortWriter{err: io.ErrShortWrite}
	errBuf := &bytes.Buffer{}
	code := newGateSummarySchemaCLI(out, errBuf).Run([]string{"show", "v1"})
	if code == 0 {
		t.Fatalf("show short write must exit non-zero")
	}
	if errBuf.String() == "" {
		t.Fatalf("show short-write diagnostic must reach stderr")
	}
}
