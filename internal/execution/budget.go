// Package execution provides a bounded execution gateway for Leamas.
// All external command execution must flow through this package.
package execution

import (
	"time"
)

// Default execution bounds for Leamas.
const (
	DefaultMaxConcurrent    int64  = 4
	DefaultMaxStarts        uint64 = 64
	DefaultMaxTaskDepth     uint16 = 8
	DefaultMaxOutputBytes   int64  = 8 * 1024 * 1024 // 8 MiB
	DefaultTerminationGrace        = 2 * time.Second
	DefaultTimeout                 = 120 * time.Second
	MaxPermittedTimeout            = 10 * time.Minute
)

// Budget defines resource constraints for a root execution.
type Budget struct {
	Deadline         time.Time
	MaxConcurrent    int64
	MaxStarts        uint64
	MaxTaskDepth     uint16
	MaxOutputBytes   int64
	TerminationGrace time.Duration
}

// DefaultBudget returns a Budget with safe defaults.
func DefaultBudget() *Budget {
	return &Budget{
		Deadline:         time.Now().Add(DefaultTimeout),
		MaxConcurrent:    DefaultMaxConcurrent,
		MaxStarts:        DefaultMaxStarts,
		MaxTaskDepth:     DefaultMaxTaskDepth,
		MaxOutputBytes:   DefaultMaxOutputBytes,
		TerminationGrace: DefaultTerminationGrace,
	}
}

// WithDeadline returns a copy with the specified deadline.
func (b *Budget) WithDeadline(d time.Time) *Budget {
	c := *b
	c.Deadline = d
	return &c
}

// WithTimeout returns a copy with the specified timeout.
func (b *Budget) WithTimeout(d time.Duration) *Budget {
	c := *b
	c.Deadline = time.Now().Add(d)
	return &c
}

// WithMaxConcurrent returns a copy with the specified concurrency limit.
func (b *Budget) WithMaxConcurrent(n int64) *Budget {
	c := *b
	c.MaxConcurrent = n
	return &c
}

// WithMaxStarts returns a copy with the specified starts limit.
func (b *Budget) WithMaxStarts(n uint64) *Budget {
	c := *b
	c.MaxStarts = n
	return &c
}

// WithMaxTaskDepth returns a copy with the specified task depth.
func (b *Budget) WithMaxTaskDepth(d uint16) *Budget {
	c := *b
	c.MaxTaskDepth = d
	return &c
}

// WithMaxOutputBytes returns a copy with the specified output limit.
func (b *Budget) WithMaxOutputBytes(n int64) *Budget {
	c := *b
	c.MaxOutputBytes = n
	return &c
}

// WithTerminationGrace returns a copy with the specified grace period.
func (b *Budget) WithTerminationGrace(d time.Duration) *Budget {
	c := *b
	c.TerminationGrace = d
	return &c
}
