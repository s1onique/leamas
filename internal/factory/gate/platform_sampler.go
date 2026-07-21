// Package gate provides platform-specific resource sampling.
package gate

import (
	"syscall"
	"time"
)

// ResourceSampler samples resource usage for the current process.
type ResourceSampler interface {
	Sample() (ResourceSnapshot, error)
}

// ResourceSnapshot represents a point-in-time resource observation.
type ResourceSnapshot struct {
	UserCPU         time.Duration
	SystemCPU       time.Duration
	ProcessMaxRSSKB int64
}

// PlatformSampler samples resources using getrusage(2).
type PlatformSampler struct{}

// Sample collects resource usage for the current process.
func (s *PlatformSampler) Sample() (ResourceSnapshot, error) {
	var rusage syscall.Rusage
	if err := syscall.Getrusage(syscall.RUSAGE_SELF, &rusage); err != nil {
		return ResourceSnapshot{}, err
	}
	return ResourceSnapshot{
		UserCPU:         time.Duration(rusage.Utime.Nano()) * time.Nanosecond,
		SystemCPU:       time.Duration(rusage.Stime.Nano()) * time.Nanosecond,
		ProcessMaxRSSKB: int64(rusage.Maxrss), // On Linux, Maxrss is already in KiB
	}, nil
}

// NewPlatformSampler creates a new platform sampler.
func NewPlatformSampler() *PlatformSampler {
	return &PlatformSampler{}
}
