// SPDX-License-Identifier: Apache-2.0

// Package authority: path_discovery.go provides the bounded,
// testable PATH scanning seam used by the doctor surface and by
// any other consumer that needs to enumerate executable
// candidates without mutating process-global state.
//
// The seam deliberately separates the deterministic scanner
// (discoverPATHExecutablesFrom) from the production wrapper
// (discoverPATHExecutables) so that tests can drive the same
// code path with explicit inputs. The scanner never walks
// arbitrary directory trees and never caches results.
package authority

import (
	"os"
	"path/filepath"
	"strings"
)

// statFn is the indirection over os.Stat used by the discovery
// seam. Production code always passes os.Stat; tests inject a
// controlled fake or os.Stat itself.
type statFn func(string) (os.FileInfo, error)

// discoverPATHExecutablesFrom scans pathValue for regular
// executable files whose basename equals name. It preserves PATH
// order, deduplicates identical candidate paths, ignores missing
// or non-regular entries, refuses non-executable files, and never
// performs an unbounded directory walk.
//
// stat may be nil, in which case os.Stat is used. The function is
// safe to call with an empty pathValue (returns nil).
func discoverPATHExecutablesFrom(name, pathValue string, stat statFn) []string {
	if name == "" || pathValue == "" {
		return nil
	}
	if stat == nil {
		stat = os.Stat
	}
	var out []string
	seen := make(map[string]bool)
	for _, dir := range strings.Split(pathValue, string(os.PathListSeparator)) {
		if dir == "" {
			continue
		}
		candidate := filepath.Join(dir, name)
		if seen[candidate] {
			continue
		}
		info, err := stat(candidate)
		if err != nil || info == nil {
			continue
		}
		if !info.Mode().IsRegular() {
			continue
		}
		// Reject files with no executable bit set on any class.
		// On Windows this is a no-op because perm bits are not
		// the host executable semantic; a follow-up seam can
		// use os/exec.LookPath when host behaviour diverges.
		if info.Mode().Perm()&0o111 == 0 {
			continue
		}
		seen[candidate] = true
		out = append(out, candidate)
	}
	return out
}

// discoverPATHExecutables is the production wrapper that drives
// the scanner with the real process PATH and os.Stat. Tests MUST
// not invoke this helper to bypass the production seam; they
// should call discoverPATHExecutablesFrom with explicit inputs.
func discoverPATHExecutables(name string) []string {
	return discoverPATHExecutablesFrom(name, os.Getenv("PATH"), os.Stat)
}
