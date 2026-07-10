// Package execution provides a bounded execution gateway for Leamas.
package execution

import (
	"bytes"
	"io"
	"sync"
	"sync/atomic"
)

// CappedWriter wraps a writer with a byte limit.
// When the limit is exceeded, subsequent writes are blocked and
// the truncated flag is set.
type CappedWriter struct {
	w         io.Writer
	limit     int64
	written   int64
	truncated atomic.Bool
	mu        sync.Mutex
}

// NewCappedWriter creates a new CappedWriter.
func NewCappedWriter(w io.Writer, limit int64) *CappedWriter {
	return &CappedWriter{
		w:     w,
		limit: limit,
	}
}

// Write implements io.Writer with a byte limit.
func (c *CappedWriter) Write(p []byte) (n int, err error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Check if already truncated
	if c.truncated.Load() {
		return 0, io.ErrClosedPipe
	}

	// Check if this write would exceed the limit
	if c.written+int64(len(p)) > c.limit {
		// Write what we can up to the limit
		available := c.limit - c.written
		if available > 0 {
			toWrite := p[:available]
			written, err := c.w.Write(toWrite)
			c.written += int64(written)
			if err != nil {
				return written, err
			}
		}
		c.truncated.Store(true)
		return len(p), io.ErrClosedPipe
	}

	// Normal write within limit
	written, err := c.w.Write(p)
	c.written += int64(written)
	return written, err
}

// Truncated returns true if the output was truncated.
func (c *CappedWriter) Truncated() bool {
	return c.truncated.Load()
}

// BytesWritten returns the number of bytes written.
func (c *CappedWriter) BytesWritten() int64 {
	return atomic.LoadInt64(&c.written)
}

// CappedBuffer is a buffer that stores output up to a limit.
// It retains the head and tail of the output.
type CappedBuffer struct {
	limit int64
	mu    sync.Mutex
	buf   *bytes.Buffer
}

// NewCappedBuffer creates a new CappedBuffer with the specified limit.
func NewCappedBuffer(limit int64) *CappedBuffer {
	return &CappedBuffer{
		limit: limit,
		buf:   new(bytes.Buffer),
	}
}

// Write implements io.Writer with a byte limit.
func (c *CappedBuffer) Write(p []byte) (n int, err error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if int64(c.buf.Len())+int64(len(p)) > c.limit {
		// Would exceed limit - write what we can
		available := c.limit - int64(c.buf.Len())
		if available > 0 {
			toWrite := p[:available]
			_, err := c.buf.Write(toWrite)
			if err != nil {
				return len(p), err
			}
		}
		return len(p), nil
	}

	return c.buf.Write(p)
}

// Bytes returns the buffered content.
func (c *CappedBuffer) Bytes() []byte {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.buf.Bytes()
}

// Len returns the current buffer length.
func (c *CappedBuffer) Len() int64 {
	c.mu.Lock()
	defer c.mu.Unlock()
	return int64(c.buf.Len())
}

// Truncated returns true if content was truncated.
func (c *CappedBuffer) Truncated() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return int64(c.buf.Len()) >= c.limit
}

// WriteTo writes the buffered content to a writer.
func (c *CappedBuffer) WriteTo(w io.Writer) (n int64, err error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return io.Copy(w, c.buf)
}

// MultiWriter creates a writer that duplicates writes to multiple writers.
// It stops writing to all writers when any writer returns an error.
type MultiWriter struct {
	writers []io.Writer
}

// NewMultiWriter creates a MultiWriter that writes to all provided writers.
func NewMultiWriter(writers ...io.Writer) *MultiWriter {
	return &MultiWriter{writers: writers}
}

// Write writes to all writers, stopping on first error.
func (m *MultiWriter) Write(p []byte) (n int, err error) {
	for _, w := range m.writers {
		n, err = w.Write(p)
		if err != nil {
			return n, err
		}
	}
	return len(p), nil
}

// TeeWriter writes to two writers and tracks total bytes.
type TeeWriter struct {
	w1    io.Writer
	w2    io.Writer
	total int64
	mu    sync.Mutex
}

// NewTeeWriter creates a new TeeWriter.
func NewTeeWriter(w1, w2 io.Writer) *TeeWriter {
	return &TeeWriter{w1: w1, w2: w2}
}

// Write writes to both writers.
func (t *TeeWriter) Write(p []byte) (n int, err error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	n1, err1 := t.w1.Write(p)
	n2, err2 := t.w2.Write(p)

	t.total += int64(n1)

	if err1 != nil {
		return n1, err1
	}
	if err2 != nil {
		return n2, err2
	}

	if n1 != n2 {
		return n1, io.ErrShortWrite
	}

	return n1, nil
}

// Total returns the total bytes written.
func (t *TeeWriter) Total() int64 {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.total
}
