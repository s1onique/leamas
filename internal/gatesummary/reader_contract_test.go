package gatesummary

import "testing"

func TestReadBoundedStopsAtSentinelByte(t *testing.T) {
	t.Parallel()

	probe := &sentinelProbeReader{remaining: MaxDocumentBytes + 2}
	res := readBounded(probe)
	if !isOversize(res.err) {
		t.Fatalf("error=%v, want oversize", res.err)
	}
	if probe.read != MaxDocumentBytes+1 {
		t.Fatalf("reader consumed %d bytes, want exactly %d", probe.read, MaxDocumentBytes+1)
	}
	if probe.remaining != 1 {
		t.Fatalf("reader crossed sentinel: %d source bytes remain, want 1", probe.remaining)
	}
}

// sentinelProbeReader panics if the bounded reader asks for bytes after
// consuming the 4 MiB + 1 sentinel. One extra source byte remains to prove
// that io.LimitReader, not source EOF, stopped the read.
type sentinelProbeReader struct {
	remaining int
	read      int
}

func (r *sentinelProbeReader) Read(p []byte) (int, error) {
	if r.read >= MaxDocumentBytes+1 {
		panic("read beyond 4 MiB + 1 sentinel")
	}
	n := len(p)
	if n > r.remaining {
		n = r.remaining
	}
	if limit := MaxDocumentBytes + 1 - r.read; n > limit {
		n = limit
	}
	for i := 0; i < n; i++ {
		p[i] = 'x'
	}
	r.read += n
	r.remaining -= n
	return n, nil
}
