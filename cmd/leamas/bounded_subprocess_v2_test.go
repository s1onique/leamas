// SPDX-License-Identifier: Apache-2.0

package main

// bounded_subprocess_v2_test.go provides a bounded subprocess
// harness for the installed-style v2 authority tests. The
// harness is required by
// ACT-LEAMAS-FACTORY-CLOSURE-PROTOCOL-V2-RUNNER-MAC-CANARY-READINESS01-R1
// because raw exec.Command(...).Run() does not bound the
// subprocess lifetime, output sizes, or cleanup.
//
// R2B (MAC-CANARY-READINESS01-R2B): the harness now exposes
// explicit StdoutTruncated / StderrTruncated flags so callers
// (notably installed-style authority dogfood) can refuse to
// claim success when subprocess output was discarded. The
// underlying writer continues to return the requested write
// length so the subprocess is never silently blocked by an
// unread pipe; the truncation flag is the only signal a
// caller may inspect.
//
// The bounded buffer uses a plain []byte field rather than
// embedding bytes.Buffer. Embedding caused the embedded
// Write method to mask the override in some Go runtime /
// os/exec dispatch paths, leaving truncation undetected. A
// plain field keeps the override authoritative.

import (
	"context"
	"errors"
	"io"
	"os/exec"
	"sync"
	"sync/atomic"
	"time"
)

type boundedSubprocessV2Options struct {
	Timeout   time.Duration
	MaxStdout int
	MaxStderr int
	WorkDir   string
	Env       []string
}

// boundedSubprocessV2Result is the bounded subprocess result.
// R2B adds StdoutTruncated and StderrTruncated: each flag is
// true when the underlying buffer discarded at least one byte
// because the configured limit was reached. The flags MUST be
// inspected by callers that promise bounded output.
type boundedSubprocessV2Result struct {
	Stdout          []byte
	Stderr          []byte
	StdoutTruncated bool
	StderrTruncated bool
	ExitCode        int
	Err             error
	TimedOut        bool
}

func boundedSubprocessV2Defaults() boundedSubprocessV2Options {
	return boundedSubprocessV2Options{
		Timeout:   5 * time.Minute,
		MaxStdout: 1 << 20,
		MaxStderr: 1 << 20,
	}
}

// boundedSubprocessV2 runs the supplied binary with the
// supplied argv, capturing stdout / stderr into bounded
// buffers. The function blocks until the process exits or the
// timeout fires. The result carries truncation flags for any
// caller that requires explicit bounded-output semantics.
func boundedSubprocessV2(binary string, argv []string, opts boundedSubprocessV2Options) boundedSubprocessV2Result {
	if opts.Timeout == 0 {
		opts.Timeout = boundedSubprocessV2Defaults().Timeout
	}
	if opts.MaxStdout == 0 {
		opts.MaxStdout = boundedSubprocessV2Defaults().MaxStdout
	}
	if opts.MaxStderr == 0 {
		opts.MaxStderr = boundedSubprocessV2Defaults().MaxStderr
	}
	ctx, cancel := context.WithTimeout(context.Background(), opts.Timeout)
	defer cancel()
	all := append([]string{binary}, argv...)
	cmd := exec.CommandContext(ctx, all[0], all[1:]...)
	if opts.WorkDir != "" {
		cmd.Dir = opts.WorkDir
	}
	if opts.Env != nil {
		cmd.Env = opts.Env
	}
	cmd.WaitDelay = 5 * time.Second
	cmd.Cancel = func() error { return cmd.Process.Kill() }

	stdout := newBoundedBuffer(opts.MaxStdout)
	stderr := newBoundedBuffer(opts.MaxStderr)
	cmd.Stdout = stdout
	cmd.Stderr = stderr

	runErr := cmd.Run()
	result := boundedSubprocessV2Result{
		Stdout:          stdout.Bytes(),
		Stderr:          stderr.Bytes(),
		StdoutTruncated: stdout.Truncated(),
		StderrTruncated: stderr.Truncated(),
	}
	if runErr != nil {
		result.Err = runErr
		var ee *exec.ExitError
		if errors.As(runErr, &ee) {
			result.ExitCode = ee.ExitCode()
		} else {
			result.ExitCode = -1
		}
	}
	if ctx.Err() != nil && errors.Is(ctx.Err(), context.DeadlineExceeded) {
		result.TimedOut = true
	}
	return result
}

// boundedBuffer is a fixed-capacity byte buffer. Write
// returns the full len(p) so the subprocess is never blocked
// by an unread pipe. When the limit is reached, further bytes
// are silently discarded and Truncated() begins returning
// true. Concurrency-safe: Write may be invoked from a
// background goroutine spawned by os/exec.
type boundedBuffer struct {
	mu        sync.Mutex
	limit     int
	buf       []byte
	truncated atomic.Bool
}

func newBoundedBuffer(limit int) *boundedBuffer {
	return &boundedBuffer{limit: limit}
}

// Write appends p up to the configured limit. Once the limit
// is reached, every subsequent Write increments the truncated
// flag and returns len(p) without growing the buffer.
func (b *boundedBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.limit > 0 && len(b.buf) >= b.limit {
		b.truncated.Store(true)
		return len(p), nil
	}
	if b.limit > 0 && len(b.buf)+len(p) > b.limit {
		remaining := b.limit - len(b.buf)
		if remaining > 0 {
			b.buf = append(b.buf, p[:remaining]...)
		}
		b.truncated.Store(true)
		return len(p), nil
	}
	b.buf = append(b.buf, p...)
	return len(p), nil
}

// Bytes returns a snapshot of the captured bytes.
func (b *boundedBuffer) Bytes() []byte {
	b.mu.Lock()
	defer b.mu.Unlock()
	out := make([]byte, len(b.buf))
	copy(out, b.buf)
	return out
}

// Truncated reports whether at least one byte was discarded.
func (b *boundedBuffer) Truncated() bool {
	return b.truncated.Load()
}

var _ = io.Discard
