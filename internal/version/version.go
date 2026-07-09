// Package version provides build metadata for leamas.
// Metadata can be injected at build time via linker flags:
//
//	-X 'github.com/s1onique/leamas/internal/version.Version=0.1.0'
//	-X 'github.com/s1onique/leamas/internal/version.Commit=abc1234'
//	-X 'github.com/s1onique/leamas/internal/version.BuildTime=2024-01-01T00:00:00Z'
package version

import (
	"runtime/debug"
	"time"
)

// Package-level variables for linker injection.
var (
	Version   = "dev"
	Commit    = "unknown"
	BuildTime = "unknown"
)

// Info holds the version information.
type Info struct {
	Version   string `json:"version"`
	Commit    string `json:"commit"`
	BuildTime string `json:"build_time"`
}

// Get returns the version metadata, applying fallback logic from runtime/debug
// if Commit or BuildTime are still at their default "unknown" values.
func Get() Info {
	info := Info{
		Version:   Version,
		Commit:    Commit,
		BuildTime: BuildTime,
	}

	// Try to get VCS info from runtime/debug if not injected.
	if info.Commit == "unknown" || info.BuildTime == "unknown" {
		if bi, ok := debug.ReadBuildInfo(); ok {
			info = FromSettings(info, bi.Settings)
		}
	}

	return info
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
		}
	}
	return info
}
