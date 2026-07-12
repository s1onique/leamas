// Package version provides build metadata for leamas.
// Metadata can be injected at build time via linker flags:
//
//	-X 'github.com/s1onique/leamas/internal/version.Version=0.1.0'
//	-X 'github.com/s1onique/leamas/internal/version.DeclaredVersion=0.1.0'
//	-X 'github.com/s1onique/leamas/internal/version.Commit=abc1234'
//	-X 'github.com/s1onique/leamas/internal/version.BuildTime=2024-01-01T00:00:00Z'
//
// Version carries the *effective* SemVer used by the doctrine
// compatibility oracle. DeclaredVersion is the literal value
// supplied to the build via VERSION=. When Version is already a
// strict SemVer, it is authoritative (the linker injection is
// the single source of truth for the effective value). When
// Version is a placeholder such as "dev", the effective version
// is derived from DeclaredVersion plus the VCS provenance.
// The CLI command surfaces both shapes.
//
// Commit and BuildTime are linker-injected by the Makefile via
// `-ldflags -X` and additionally recovered from
// runtime/debug.ReadBuildInfo()'s `vcs.revision` / `vcs.time`
// when `-buildvcs` is enabled (the modern Go default). The Dirty
// field is NOT linker-injected; the VCS information
// (`vcs.modified`) is the canonical source for the marker.
package version

import (
	"encoding/json"
	"runtime/debug"
	"time"
)

// Package-level variables for linker injection.
var (
	// Version is the authoritative effective SemVer used by the
	// doctrine compatibility check. When it is a strict SemVer,
	// it is returned verbatim by both Effective() and Get().
	// When it is a placeholder such as "dev", the effective
	// version is derived from DeclaredVersion plus the VCS
	// provenance.
	Version = "dev"

	// DeclaredVersion is the literal VERSION= value passed to
	// the build. Independent of Version so the CLI can show
	// "yes, the user passed dev; we auto-stamped to 0.1.0+..."
	// rather than reporting only the derived stamp.
	DeclaredVersion = "dev"

	// Commit is the VCS commit hash, trim-able to shortSHA.
	Commit = "unknown"

	// BuildTime is the build timestamp in RFC3339/UTC.
	BuildTime = "unknown"

	// Dirty is the VCS dirty marker. The variable exists so the
	// LDFLAGS contract is documented, but the Makefile does not
	// inject it. Instead, runtime/debug.ReadBuildInfo() supplies
	// the authoritative value when the binary was built with
	// `-buildvcs=true` (the modern Go default).
	Dirty = "false"
)

// Info holds the version information reported by `leamas version`.
//
// Version is the effective SemVer used by the doctrine
// compatibility check. DeclaredVersion is the literal value passed
// to the build via VERSION=. When the declared value was a
// placeholder (dev / unknown / empty), Version and DeclaredVersion
// differ: Version is the auto-derived SemVer (for example
// "0.1.0+dev.<commit>.<ts>") and DeclaredVersion is the original
// "dev" placeholder. Commit, BuildTime, and Dirty carry immutable
// build provenance and are deliberately separate fields.
//
// JSON serialisation uses `omitempty` on DeclaredVersion and
// Dirty so the wire form omits them when they match the version
// default or are `false`. The line-oriented CLI applies the same
// rules inline; MarshalJSON enforces the wire form for any Info
// regardless of how it was constructed.
type Info struct {
	Version         string `json:"version"`
	DeclaredVersion string `json:"declared_version,omitempty"`
	Commit          string `json:"commit"`
	BuildTime       string `json:"build_time"`
	Dirty           string `json:"dirty,omitempty"`
}

// Get returns the version metadata, applying fallback logic from
// runtime/debug if Commit, BuildTime, or Dirty are still at their
// default values. The effective SemVer is then derived from the
// post-fallback info via the canonical EffectiveVersion helper.
func Get() Info {
	info := Info{
		Version:         Version,
		DeclaredVersion: DeclaredVersion,
		Commit:          Commit,
		BuildTime:       BuildTime,
		Dirty:           Dirty,
	}

	if info.Commit == "unknown" || info.BuildTime == "unknown" || info.Dirty == "false" {
		if bi, ok := debug.ReadBuildInfo(); ok {
			info = getFromSettings(info, bi.Settings)
		}
	}

	// Compute the effective SemVer from the post-fallback provenance
	// and the linker-injected Version. The Version takes
	// precedence when it is already a strict SemVer so the build's
	// own stamping is preserved across the dev/install/release
	// pipeline.
	info.Version = EffectiveVersion(Version, info.DeclaredVersion, info.Commit, info.BuildTime)

	// JSON-contract cleanup: clear DeclaredVersion when it
	// matches the effective Version so the omitempty tag drops
	// it from the wire form; clear Dirty when it is the
	// default "false" for the same reason. The line-oriented CLI
	// applies the same rules inline.
	if info.DeclaredVersion == info.Version {
		info.DeclaredVersion = ""
	}
	if info.Dirty == "false" {
		info.Dirty = ""
	}

	return info
}

// getFromSettings applies VCS build settings to the base Info and
// returns the augmented copy. Exported as a package-private seam
// so tests can exercise the production composition: build a base
// Info from package globals, run VCS settings through
// getFromSettings, then derive via EffectiveVersion. Used by
// Get() so callers and tests run the same code path.
func getFromSettings(base Info, settings []debug.BuildSetting) Info {
	return FromSettings(base, settings)
}

// FromSettings applies VCS fallback settings to the base info struct.
// Exported for testing.
func FromSettings(base Info, settings []debug.BuildSetting) Info {
	info := base
	for _, s := range settings {
		switch s.Key {
		case "vcs.revision":
			if info.Commit == "unknown" {
				info.Commit = s.Value
			}
		case "vcs.time":
			if info.BuildTime == "unknown" {
				if t, err := time.Parse(time.RFC3339, s.Value); err == nil {
					info.BuildTime = t.UTC().Format(time.RFC3339)
				}
			}
		case "vcs.modified":
			// vcs.modified is set to "true" or "false" by the
			// Go toolchain when -buildvcs is enabled. Prefer
			// a non-default value over the linker default.
			if info.Dirty == "false" {
				info.Dirty = s.Value
			}
		}
	}
	return info
}

// jsonInfo is the wire-form shape of Info. DeclaredVersion and
// Dirty are emitted only when set, so the canonical contract is
// observable regardless of whether the consumer obtained an Info
// via Get() or constructed one directly.
type jsonInfo struct {
	Version         string `json:"version"`
	DeclaredVersion string `json:"declared_version,omitempty"`
	Commit          string `json:"commit"`
	BuildTime       string `json:"build_time"`
	Dirty           string `json:"dirty,omitempty"`
}

// MarshalJSON implements the conditional-field contract:
// DeclaredVersion is hidden when equal to Version, and Dirty
// when equal to "false". This guarantees a stable wire form
// irrespective of who constructed the Info value.
func (i Info) MarshalJSON() ([]byte, error) {
	wire := jsonInfo{
		Version:         i.Version,
		DeclaredVersion: i.DeclaredVersion,
		Commit:          i.Commit,
		BuildTime:       i.BuildTime,
		Dirty:           i.Dirty,
	}
	if wire.DeclaredVersion == wire.Version {
		wire.DeclaredVersion = ""
	}
	if wire.Dirty == "false" {
		wire.Dirty = ""
	}
	return json.Marshal(wire)
}
