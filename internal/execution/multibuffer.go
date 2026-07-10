//go:build unix || darwin || linux

package execution

import (
	"bytes"
	"io"
	"sync"
)

// MultiBuffer captures stdout and stderr separately.
type MultiBuffer struct {
	stdout bytes.Buffer
	stderr bytes.Buffer
	mu     sync.Mutex
}

// NewMultiBuffer creates a new MultiBuffer.
func NewMultiBuffer() *MultiBuffer {
	return &MultiBuffer{}
}

// StdoutWriter returns an io.Writer for stdout.
func (b *MultiBuffer) StdoutWriter() io.Writer {
	return cappedWriter{buf: &b.stdout, mu: &b.mu}
}

// StderrWriter returns an io.Writer for stderr.
func (b *MultiBuffer) StderrWriter() io.Writer {
	return cappedWriter{buf: &b.stderr, mu: &b.mu}
}

// Stdout returns the stdout content.
func (b *MultiBuffer) Stdout() []byte {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.stdout.Bytes()
}

// Stderr returns the stderr content.
func (b *MultiBuffer) Stderr() []byte {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.stderr.Bytes()
}

type cappedWriter struct {
	buf *bytes.Buffer
	mu  *sync.Mutex
}

func (w cappedWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.buf.Write(p)
}
