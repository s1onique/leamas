// SPDX-License-Identifier: Apache-2.0

// Package evidence - gate_runner_buffer.go owns the
// bounded stdout/stderr writer OsRunner uses. Splitting
// the writer out of gate_capture.go keeps the production
// capture file under the LLM-friendly 400-line threshold
// while preserving the single closure over the descriptor
// that ACT-LEAMAS-FACTORY-CLOSURE-PROTOCOL-V1-01 requires.
package evidence

import (
	"bytes"
)

// truncatedBuffer is the authoritative writer. Its
// truncated flag is set when overflow occurs; downstream
// consumers MUST read this flag rather than re-deriving
// truncation from byte length.
type truncatedBuffer struct {
	cap       int
	buf       bytes.Buffer
	truncated bool
}

func (b *truncatedBuffer) Write(p []byte) (int, error) {
	remaining := b.cap - b.buf.Len()
	if remaining <= 0 {
		b.truncated = true
		return len(p), nil
	}
	if len(p) > remaining {
		b.buf.Write(p[:remaining])
		b.truncated = true
		return len(p), nil
	}
	b.buf.Write(p)
	return len(p), nil
}

func (b *truncatedBuffer) bytes() []byte {
	return append([]byte(nil), b.buf.Bytes()...)
}
