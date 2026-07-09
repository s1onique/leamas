package version

import (
	"encoding/json"
	"runtime/debug"
	"strings"
	"testing"
)

func TestPackageVariables_Defaults(t *testing.T) {
	// Test that package variables have expected defaults
	if Version != "dev" {
		t.Errorf("Version = %q, want dev", Version)
	}
}

func TestInfo_DefaultValues(t *testing.T) {
	// Only test Version since Commit/BuildTime may legitimately
	// come from runtime/buildinfo fallback in module-built test binary.
	info := Info{Version: Version, Commit: Commit, BuildTime: BuildTime}
	if info.Version != "dev" {
		t.Errorf("info.Version = %q", info.Version)
	}
}

func TestGet_UsesInjectedValues(t *testing.T) {
	// Save original values
	oldVersion, oldCommit, oldBuildTime := Version, Commit, BuildTime
	t.Cleanup(func() {
		Version = oldVersion
		Commit = oldCommit
		BuildTime = oldBuildTime
	})

	// Simulate injected values
	Version = "1.2.3"
	Commit = "abc123"
	BuildTime = "2026-07-09T10:24:46Z"

	info := Get()

	if info.Version != "1.2.3" {
		t.Fatalf("Version = %q, want 1.2.3", info.Version)
	}
	if info.Commit != "abc123" {
		t.Fatalf("Commit = %q, want abc123", info.Commit)
	}
	if info.BuildTime != "2026-07-09T10:24:46Z" {
		t.Fatalf("BuildTime = %q, want 2026-07-09T10:24:46Z", info.BuildTime)
	}
}

func TestFromSettings_InjectsCommit(t *testing.T) {
	settings := []debug.BuildSetting{
		{Key: "vcs.revision", Value: "def456"},
	}
	info := FromSettings(Info{Version: "dev", Commit: "unknown", BuildTime: "unknown"}, settings)

	if info.Commit != "def456" {
		t.Errorf("Commit = %q, want def456", info.Commit)
	}
}

func TestFromSettings_InjectsBuildTime(t *testing.T) {
	settings := []debug.BuildSetting{
		{Key: "vcs.time", Value: "2026-07-09T10:24:46Z"},
	}
	info := FromSettings(Info{Version: "dev", Commit: "unknown", BuildTime: "unknown"}, settings)

	if info.BuildTime != "2026-07-09T10:24:46Z" {
		t.Errorf("BuildTime = %q, want 2026-07-09T10:24:46Z", info.BuildTime)
	}
}

func TestFromSettings_InjectedWins(t *testing.T) {
	// Injected Commit/BuildTime should win over fallback settings
	settings := []debug.BuildSetting{
		{Key: "vcs.revision", Value: "from-vcs"},
		{Key: "vcs.time", Value: "2026-07-09T10:24:46Z"},
	}
	info := FromSettings(Info{Version: "dev", Commit: "injected", BuildTime: "injected"}, settings)

	if info.Commit != "injected" {
		t.Errorf("Commit = %q, want injected (injected wins)", info.Commit)
	}
	if info.BuildTime != "injected" {
		t.Errorf("BuildTime = %q, want injected (injected wins)", info.BuildTime)
	}
}

func TestFromSettings_InvalidTime(t *testing.T) {
	// Invalid vcs.time should leave BuildTime unchanged
	settings := []debug.BuildSetting{
		{Key: "vcs.time", Value: "not-a-time"},
	}
	info := FromSettings(Info{Version: "dev", Commit: "unknown", BuildTime: "keep-me"}, settings)

	if info.BuildTime != "keep-me" {
		t.Errorf("BuildTime = %q, want keep-me", info.BuildTime)
	}
}

func TestInfo_JSON(t *testing.T) {
	info := Info{Version: "dev", Commit: "test", BuildTime: "2026-01-01T00:00:00Z"}
	data, err := json.Marshal(info)
	if err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}

	var parsed map[string]string
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("json.Unmarshal failed: %v", err)
	}

	if parsed["version"] == "" {
		t.Error("json missing 'version' field")
	}
	if parsed["commit"] == "" {
		t.Error("json missing 'commit' field")
	}
	if parsed["build_time"] == "" {
		t.Error("json missing 'build_time' field")
	}
}

func TestInfo_LineFormat(t *testing.T) {
	info := Info{Version: "dev", Commit: "abc", BuildTime: "2026-01-01T00:00:00Z"}
	lines := []string{
		"version: " + info.Version,
		"commit: " + info.Commit,
		"build_time: " + info.BuildTime,
	}

	for _, line := range lines {
		if !strings.Contains(line, ":") {
			t.Errorf("line %q missing colon separator", line)
		}
	}
}
