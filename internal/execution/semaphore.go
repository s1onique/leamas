package execution

import (
	"context"
	"fmt"
	"sync"
)

// contextSemaphore provides context-cancellable semaphore acquisition.
type contextSemaphore struct {
	sem   chan struct{}
	limit int64
	mu    sync.Mutex
}

func newContextSemaphore(limit int64) *contextSemaphore {
	return &contextSemaphore{
		sem:   make(chan struct{}, limit),
		limit: limit,
	}
}

func (s *contextSemaphore) Acquire(ctx context.Context, n int64) (bool, error) {
	if n <= 0 || n > s.limit {
		return false, fmt.Errorf("invalid weight: %d", n)
	}

	for i := int64(0); i < n; i++ {
		select {
		case <-ctx.Done():
			// Release any permits we acquired
			for j := int64(0); j < i; j++ {
				<-s.sem
			}
			return false, ctx.Err()
		case s.sem <- struct{}{}:
			// Acquired
		}
	}
	return true, nil
}

func (s *contextSemaphore) Release(n int64) {
	for i := int64(0); i < n; i++ {
		select {
		case <-s.sem:
			// Released
		default:
			// Already empty
		}
	}
}

func (s *contextSemaphore) Count() int64 {
	return int64(len(s.sem))
}

func (s *contextSemaphore) Limit() int64 {
	return s.limit
}
