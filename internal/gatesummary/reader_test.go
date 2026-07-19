package gatesummary

import (
	"errors"
	"io"
	"strings"
	"testing"
)

func TestReadBoundedBelowLimit(t *testing.T) {
	data := strings.Repeat("a", 1024)
	res := readBounded(strings.NewReader(data))
	if res.err != nil {
		t.Fatalf("unexpected error: %v", res.err)
	}
	if len(res.data) != 1024 {
		t.Fatalf("data length=%d, want 1024", len(res.data))
	}
}

func TestReadBoundedAtLimit(t *testing.T) {
	data := strings.Repeat("a", MaxDocumentBytes)
	res := readBounded(strings.NewReader(data))
	if res.err != nil {
		t.Fatalf("unexpected error: %v", res.err)
	}
	if len(res.data) != MaxDocumentBytes {
		t.Fatalf("data length=%d, want %d", len(res.data), MaxDocumentBytes)
	}
}

func TestReadBoundedOverLimit(t *testing.T) {
	data := strings.Repeat("a", MaxDocumentBytes+1)
	res := readBounded(strings.NewReader(data))
	if res.err == nil {
		t.Fatalf("expected oversize error, got nil (data len=%d)", len(res.data))
	}
	if !isOversize(res.err) {
		t.Fatalf("expected errDocumentOversize, got %T: %v", res.err, res.err)
	}
	if len(res.data) != 0 {
		t.Fatalf("oversize path must not return data; got %d bytes", len(res.data))
	}
}

func TestReadBoundedReaderError(t *testing.T) {
	res := readBounded(&errReader{err: io.ErrUnexpectedEOF})
	if res.err == nil {
		t.Fatal("expected error")
	}
	if isOversize(res.err) {
		t.Fatalf("reader error must not be classified as oversize")
	}
}

func TestReadBoundedNilReader(t *testing.T) {
	res := readBounded(nil)
	if res.err == nil {
		t.Fatal("expected error for nil reader")
	}
}

// errReader is an io.Reader that always returns err.
type errReader struct{ err error }

func (e *errReader) Read(p []byte) (int, error) {
	return 0, e.err
}

// errReaderReturningBytes wraps an io.Reader but errors after a fixed
// number of bytes have been returned.
type errReaderReturningBytes struct {
	limit int
	err   error
	read  int
}

func (e *errReaderReturningBytes) Read(p []byte) (int, error) {
	if e.read >= e.limit {
		return 0, e.err
	}
	remain := e.limit - e.read
	n := len(p)
	if n > remain {
		n = remain
	}
	for i := 0; i < n; i++ {
		p[i] = 'a'
	}
	e.read += n
	return n, nil
}

// errAlways returns errors unconditionally.
type errAlways struct{}

func (errAlways) Read(p []byte) (int, error) { return 0, errors.New("boom") }
