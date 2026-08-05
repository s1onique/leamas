// SPDX-License-Identifier: Apache-2.0

package main

// bounded_subprocess_v2_test.go provides a bounded subprocess
// harness for the installed-style v2 authority tests. The
// harness is required by
// ACT-LEAMAS-FACTORY-CLOSURE-PROTOCOL-V2-RUNNER-MAC-CANARY-READINESS01-R1
// because raw exec.Command(...).Run() does not bound the
// subprocess lifetime, output sizes, or cleanup.

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os/exec"
	"time"
)

type boundedSubprocessV2Options struct {
	Timeout   time.Duration
	MaxStdout int
	MaxStderr int
	WorkDir   string
	Env       []string
}

type boundedSubprocessV2Result struct {
	Stdout   []byte
	Stderr   []byte
	ExitCode int
	Err      error
	TimedOut bool
}

func boundedSubprocessV2Defaults() boundedSubprocessV2Options {
	return boundedSubprocessV2Options{
		Timeout:   5 * time.Minute,
		MaxStdout: 1 << 20,
		MaxStderr: 1 << 20,
	}
}

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
		Stdout: stdout.Bytes(),
		Stderr: stderr.Bytes(),
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

type boundedBuffer struct {
	bytes.Buffer
	limit int
}

func newBoundedBuffer(limit int) *boundedBuffer {
	return &boundedBuffer{limit: limit}
}

func (b *boundedBuffer) Write(p []byte) (int, error) {
	if b.Len() >= b.limit {
		return len(p), nil
	}
	remaining := b.limit - b.Len()
	if len(p) <= remaining {
		return b.Buffer.Write(p)
	}
	if _, err := b.Buffer.Write(p[:remaining]); err != nil {
		return 0, err
	}
	return len(p), nil
}

var _ = io.Discard
