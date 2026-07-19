package gatesummary

import (
	"errors"
	"io"
)

// MaxDocumentBytes is the maximum gate-summary wire document size
// accepted by the decoder, in bytes. Documents exceeding this cap
// are rejected before any tokenization.
const MaxDocumentBytes = 4 * 1024 * 1024

// boundedReadResult is the outcome of the Stage 1 bounded read.
type boundedReadResult struct {
	data []byte
	err  error
}

// readBounded reads up to MaxDocumentBytes+1 bytes from r. A read of
// more than MaxDocumentBytes bytes produces (nil, oversized) without
// copying the oversize suffix. An underlying reader error produces
// (nil, wrapped error). Otherwise the data slice is returned.
func readBounded(r io.Reader) boundedReadResult {
	if r == nil {
		return boundedReadResult{err: errors.New("nil reader")}
	}
	data, err := io.ReadAll(io.LimitReader(r, int64(MaxDocumentBytes)+1))
	if err != nil {
		return boundedReadResult{err: err}
	}
	if len(data) > MaxDocumentBytes {
		return boundedReadResult{err: errDocumentOversize{}}
	}
	return boundedReadResult{data: data}
}

// errDocumentOversize signals that the bounded reader observed the
// MaxDocumentBytes+1 sentinel byte.
type errDocumentOversize struct{}

// Error implements the error interface for the oversize sentinel.
func (errDocumentOversize) Error() string {
	return "gate-summary: document exceeds 4 MiB cap"
}

// isOversize reports whether err is the bounded-reader oversize signal.
func isOversize(err error) bool {
	var o errDocumentOversize
	return errors.As(err, &o)
}
