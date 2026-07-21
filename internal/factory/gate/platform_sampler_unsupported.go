// Package gate provides the ResourceSampler interface for platform-specific resource sampling.
// This file handles platforms other than Linux and Darwin.
//go:build !linux && !darwin
// +build !linux,!darwin

package gate

import (
	"fmt"
	"runtime"
)

// PlatformSampler samples system resources. On unsupported platforms it returns an error.
type PlatformSampler struct{}

// NewPlatformSampler creates a new platform sampler.
func NewPlatformSampler() ResourceSampler {
	return &PlatformSampler{}
}

// Sample returns an error on unsupported platforms.
func (s *PlatformSampler) Sample() (ResourceSnapshot, error) {
	return ResourceSnapshot{}, fmt.Errorf(
		"resource sampling not supported on %s/%s; supported: linux, darwin",
		runtime.GOOS, runtime.GOARCH)
}
