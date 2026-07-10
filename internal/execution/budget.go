// Package execution provides a bounded execution gateway for Leamas.
// All external command execution must flow through this package.
package execution

import (
	"errors"
	"time"
)

// Default execution bounds for Leamas.
const (
	DefaultMaxConcurrent    int64  = 4
	DefaultMaxStarts        uint64 = 64
	DefaultMaxTaskDepth     uint16 = 8
	DefaultMaxOutputBytes   int64  = 8 * 1024 * 1024 // 8 MiB
	DefaultTerminationGrace        = 2 * time.Second
	DefaultPostKillWait            = 1 * time.Second
	DefaultTimeout                 = 120 * time.Second
)

// Hard maximum bounds.
const (
	MaxPermittedTimeout                 = 10 * time.Minute
	MaxPermittedMaxConcurrent    int64  = 16
	MaxPermittedMaxStarts        uint64 = 1024
	MaxPermittedMaxTaskDepth     uint16 = 32
	MaxPermittedMaxOutputBytes   int64  = 64 * 1024 * 1024 // 64 MiB
	MaxPermittedTerminationGrace        = 10 * time.Second
	MaxPermittedPostKillWait            = 5 * time.Second
)

// Budget defines resource constraints for a root execution.
type Budget struct {
	Deadline         time.Time
	MaxConcurrent    int64
	MaxStarts        uint64
	MaxTaskDepth     uint16
	MaxOutputBytes   int64
	TerminationGrace time.Duration
	PostKillWait     time.Duration
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
		PostKillWait:     DefaultPostKillWait,
	}
}

// Validate checks that the budget values are within permitted bounds.
func (b *Budget) Validate(now time.Time) error {
	if b == nil {
		return errors.New("budget is nil")
	}

	// Deadline must be in the future
	if !b.Deadline.IsZero() && b.Deadline.Before(now) {
		return errors.New("deadline is in the past")
	}

	// Concurrency must be positive and within bounds
	if b.MaxConcurrent <= 0 {
		return errors.New("MaxConcurrent must be positive")
	}
	if b.MaxConcurrent > MaxPermittedMaxConcurrent {
		return errors.New("MaxConcurrent exceeds hard maximum")
	}

	// Starts must be positive and within bounds
	if b.MaxStarts == 0 {
		return errors.New("MaxStarts must be positive")
	}
	if b.MaxStarts > MaxPermittedMaxStarts {
		return errors.New("MaxStarts exceeds hard maximum")
	}

	// Task depth must be positive and within bounds
	if b.MaxTaskDepth == 0 {
		return errors.New("MaxTaskDepth must be positive")
	}
	if b.MaxTaskDepth > MaxPermittedMaxTaskDepth {
		return errors.New("MaxTaskDepth exceeds hard maximum")
	}

	// Output bytes must be positive and within bounds
	if b.MaxOutputBytes <= 0 {
		return errors.New("MaxOutputBytes must be positive")
	}
	if b.MaxOutputBytes > MaxPermittedMaxOutputBytes {
		return errors.New("MaxOutputBytes exceeds hard maximum")
	}

	// Termination grace must be positive and within bounds
	if b.TerminationGrace <= 0 {
		return errors.New("TerminationGrace must be positive")
	}
	if b.TerminationGrace > MaxPermittedTerminationGrace {
		return errors.New("TerminationGrace exceeds hard maximum")
	}

	// PostKillWait must be positive and within bounds
	if b.PostKillWait <= 0 {
		return errors.New("PostKillWait must be positive")
	}
	if b.PostKillWait > MaxPermittedPostKillWait {
		return errors.New("PostKillWait exceeds hard maximum")
	}

	return nil
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
