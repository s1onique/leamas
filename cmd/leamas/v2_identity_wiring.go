// SPDX-License-Identifier: Apache-2.0

package main

// v2_identity_wiring.go wires the running-binary identity
// helpers exposed by internal/factory/closure into the
// concrete values stamped at link time via internal/version.
// The indirection avoids an import cycle between the closure
// package and the version package.

import (
	"runtime/debug"

	"github.com/s1onique/leamas/internal/factory/closure"
	"github.com/s1onique/leamas/internal/version"
)

func init() {
	closure.RunningLeamasVersion = func() string { return version.Effective() }
	closure.RunningLeamasVCSRevision = func() string { return version.Commit }
	closure.RunningLeamasVCSModified = func() bool { return runningBinaryDirty() }
}

// runningBinaryDirty returns the VCS dirty flag for the running
// binary. The build flag injection is the authoritative path;
// runtime/debug.ReadBuildInfo is the fallback when the binary
// was built with `-buildvcs=true` (the modern Go default).
func runningBinaryDirty() bool {
	if version.Dirty != "" && version.Dirty != "false" {
		return true
	}
	bi, ok := debug.ReadBuildInfo()
	if !ok {
		return false
	}
	for _, s := range bi.Settings {
		if s.Key == "vcs.modified" && s.Value == "true" {
			return true
		}
	}
	return false
}
