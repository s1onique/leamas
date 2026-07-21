package exectest

import (
	"bytes"
	"io"
	"sync"
)

// DefaultOutputLimit is the default output limit per stream (1 MiB).
const DefaultOutputLimit int64 = 1 << 20

// runState tracks persistent execution state for race-free classification.
type runState struct {
	mu             sync.Mutex
	stdout         *bytes.Buffer
	stderr         *bytes.Buffer
	stdoutWritten  int64
	stderrWritten  int64
	stdoutOverflow bool
	stderrOverflow bool
	stdoutLimit    int64
	stderrLimit    int64
	wasTimeout     bool
	wasCancelled   bool
}

func newRunState(stdoutLimit, stderrLimit int64) *runState {
	return &runState{
		stdout:      &bytes.Buffer{},
		stderr:      &bytes.Buffer{},
		stdoutLimit: stdoutLimit,
		stderrLimit: stderrLimit,
	}
}

func (rs *runState) markTimeout() {
	rs.mu.Lock()
	defer rs.mu.Unlock()
	rs.wasTimeout = true
}

func (rs *runState) markCancelled() {
	rs.mu.Lock()
	defer rs.mu.Unlock()
	rs.wasCancelled = true
}

func (rs *runState) isTimeout() bool {
	rs.mu.Lock()
	defer rs.mu.Unlock()
	return rs.wasTimeout
}

func (rs *runState) isCancelled() bool {
	rs.mu.Lock()
	defer rs.mu.Unlock()
	return rs.wasCancelled
}

func (rs *runState) hadOverflow() bool {
	rs.mu.Lock()
	defer rs.mu.Unlock()
	return rs.stdoutOverflow || rs.stderrOverflow
}

func (rs *runState) stdoutBytes() []byte {
	rs.mu.Lock()
	defer rs.mu.Unlock()
	return rs.stdout.Bytes()
}

func (rs *runState) stderrBytes() []byte {
	rs.mu.Lock()
	defer rs.mu.Unlock()
	return rs.stderr.Bytes()
}

func (rs *runState) observedBytes() int64 {
	rs.mu.Lock()
	defer rs.mu.Unlock()
	observed := rs.stdoutWritten
	if rs.stderrWritten > observed {
		observed = rs.stderrWritten
	}
	return observed
}

// boundedWriter captures output with byte counting.
type boundedWriter struct {
	rs       *runState
	isStderr bool
}

var _ io.Writer = (*boundedWriter)(nil)

func (bw *boundedWriter) Write(p []byte) (int, error) {
	if bw.isStderr {
		bw.rs.mu.Lock()
		bw.rs.stderrWritten += int64(len(p))
		if bw.rs.stderrWritten > bw.rs.stderrLimit && !bw.rs.stderrOverflow {
			bw.rs.stderrOverflow = true
		}
		if bw.rs.stderr.Len() < int(bw.rs.stderrLimit) {
			remaining := bw.rs.stderrLimit - int64(bw.rs.stderr.Len())
			if int64(len(p)) > remaining {
				bw.rs.stderr.Write(p[:remaining])
			} else {
				bw.rs.stderr.Write(p)
			}
		}
		bw.rs.mu.Unlock()
	} else {
		bw.rs.mu.Lock()
		bw.rs.stdoutWritten += int64(len(p))
		if bw.rs.stdoutWritten > bw.rs.stdoutLimit && !bw.rs.stdoutOverflow {
			bw.rs.stdoutOverflow = true
		}
		if bw.rs.stdout.Len() < int(bw.rs.stdoutLimit) {
			remaining := bw.rs.stdoutLimit - int64(bw.rs.stdout.Len())
			if int64(len(p)) > remaining {
				bw.rs.stdout.Write(p[:remaining])
			} else {
				bw.rs.stdout.Write(p)
			}
		}
		bw.rs.mu.Unlock()
	}
	return len(p), nil
}
