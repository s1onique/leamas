//go:build unix || darwin || linux

package execution

import (
	"io"
	"sync"
)

// sharedOutputBuffer captures stdout and stderr with a SHARED budget.
// Total combined output (stdout + stderr) is bounded by a single limit.
// It signals overflow when the limit is exceeded.
type sharedOutputBuffer struct {
	mu         sync.Mutex
	stdoutBuf  []byte
	stderrBuf  []byte
	cap        int64
	totalUsed  int64
	written    int64
	truncated  bool
	overflowed bool
	overflowCh chan struct{}
}

// newSharedOutputBuffer creates a buffer with a shared limit for both streams.
func newSharedOutputBuffer(cap int64) *sharedOutputBuffer {
	return &sharedOutputBuffer{
		stdoutBuf:  make([]byte, 0, cap),
		stderrBuf:  make([]byte, 0, cap),
		cap:        cap,
		overflowCh: make(chan struct{}),
	}
}

// signalOverflow signals that output has exceeded the limit.
func (b *sharedOutputBuffer) signalOverflow() {
	if !b.overflowed {
		b.overflowed = true
		close(b.overflowCh)
	}
}

// OverflowCh returns a channel that is closed when output exceeds the limit.
func (b *sharedOutputBuffer) OverflowCh() <-chan struct{} {
	return b.overflowCh
}

// stdoutWriter implements io.Writer for stdout stream.
type stdoutWriter struct {
	buf *sharedOutputBuffer
}

func (w *stdoutWriter) Write(p []byte) (n int, err error) {
	w.buf.mu.Lock()
	defer w.buf.mu.Unlock()

	w.buf.written += int64(len(p))

	// Check if combined would exceed limit
	if w.buf.totalUsed+int64(len(p)) > w.buf.cap {
		// Truncate: only add up to remaining capacity
		remaining := w.buf.cap - w.buf.totalUsed
		if remaining > 0 {
			w.buf.stdoutBuf = append(w.buf.stdoutBuf, p[:remaining]...)
			w.buf.totalUsed = w.buf.cap
		}
		w.buf.truncated = true
		if !w.buf.overflowed {
			w.buf.overflowed = true
			close(w.buf.overflowCh)
		}
		return len(p), nil
	}

	w.buf.stdoutBuf = append(w.buf.stdoutBuf, p...)
	w.buf.totalUsed += int64(len(p))
	return len(p), nil
}

// stderrWriter implements io.Writer for stderr stream.
type stderrWriter struct {
	buf *sharedOutputBuffer
}

func (w *stderrWriter) Write(p []byte) (n int, err error) {
	w.buf.mu.Lock()
	defer w.buf.mu.Unlock()

	w.buf.written += int64(len(p))

	// Check if combined would exceed limit
	if w.buf.totalUsed+int64(len(p)) > w.buf.cap {
		// Truncate: only add up to remaining capacity
		remaining := w.buf.cap - w.buf.totalUsed
		if remaining > 0 {
			w.buf.stderrBuf = append(w.buf.stderrBuf, p[:remaining]...)
			w.buf.totalUsed = w.buf.cap
		}
		w.buf.truncated = true
		if !w.buf.overflowed {
			w.buf.overflowed = true
			close(w.buf.overflowCh)
		}
		return len(p), nil
	}

	w.buf.stderrBuf = append(w.buf.stderrBuf, p...)
	w.buf.totalUsed += int64(len(p))
	return len(p), nil
}

// StdoutWriter returns the stdout writer.
func (b *sharedOutputBuffer) StdoutWriter() io.Writer {
	return &stdoutWriter{buf: b}
}

// StderrWriter returns the stderr writer.
func (b *sharedOutputBuffer) StderrWriter() io.Writer {
	return &stderrWriter{buf: b}
}

// Stdout returns captured stdout.
func (b *sharedOutputBuffer) Stdout() []byte {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.stdoutBuf
}

// Stderr returns captured stderr.
func (b *sharedOutputBuffer) Stderr() []byte {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.stderrBuf
}

// Truncated returns true if any output was truncated.
func (b *sharedOutputBuffer) Truncated() bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.truncated
}

// BytesObserved returns total bytes observed (including truncated).
func (b *sharedOutputBuffer) BytesObserved() int64 {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.written
}

// BytesRetained returns total bytes retained (stdout + stderr).
func (b *sharedOutputBuffer) BytesRetained() int64 {
	b.mu.Lock()
	defer b.mu.Unlock()
	return int64(len(b.stdoutBuf)) + int64(len(b.stderrBuf))
}
