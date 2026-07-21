// Package gate provides the ResourceSampler interface for platform-specific resource sampling.
//go:build darwin
// +build darwin

package gate

import (
	"os"
	"syscall"
	"time"
)

// PlatformSampler samples system resources on Darwin/macOS.
type PlatformSampler struct{}

// NewPlatformSampler creates a new Darwin platform sampler.
func NewPlatformSampler() ResourceSampler {
	return &PlatformSampler{}
}

// Sample captures current resource usage on Darwin.
// Note: Darwin's getrusage returns ru_maxrss in bytes, not KiB.
func (s *PlatformSampler) Sample() (ResourceSnapshot, error) {
	var rusage syscall.Rusage
	if err := syscall.Getrusage(syscall.RUSAGE_SELF, &rusage); err != nil {
		return ResourceSnapshot{}, err
	}

	// Darwin returns ru_maxrss in bytes, convert to KiB
	rssBytes := rusage.Maxrss
	rssKiB := int64(rssBytes) / 1024

	return ResourceSnapshot{
		UserCPU:   time.Duration(rusage.Utime.Sec)*time.Second + time.Duration(rusage.Utime.Usec)*time.Microsecond,
		SystemCPU: time.Duration(rusage.Stime.Sec)*time.Second + time.Duration(rusage.Stime.Usec)*time.Microsecond,
		// On Darwin, ru_maxrss is in bytes; convert to KiB
		ProcessMaxRSSKB: rssKiB,
	}, nil
}
