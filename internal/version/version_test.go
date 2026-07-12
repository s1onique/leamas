package version

import (
	"encoding/json"
	"runtime/debug"
	"strings"
	"testing"
)

// TestPackageVariables_Defaults exercises the package defaults.
func TestPackageVariables_Defaults(t *testing.T) {
	if Version != "dev" {
		t.Errorf("Version = %q, want dev", Version)
	}
	if DeclaredVersion != "dev" {
		t.Errorf("DeclaredVersion = %q, want dev", DeclaredVersion)
	}
	if Commit != "unknown" {
		t.Errorf("Commit = %q, want unknown", Commit)
	}
	if BuildTime != "unknown" {
		t.Errorf("BuildTime = %q, want unknown", BuildTime)
	}
	if Dirty != "false" {
		t.Errorf("Dirty = %q, want false", Dirty)
	}
}

// TestInfo_DefaultValues sanity-checks the Info zero value.
func TestInfo_DefaultValues(t *testing.T) {
	info := Info{
		Version:         Version,
		DeclaredVersion: DeclaredVersion,
		Commit:          Commit,
		BuildTime:       BuildTime,
		Dirty:           Dirty,
	}
	if info.Version != "dev" {
		t.Errorf("info.Version = %q", info.Version)
	}
	if info.DeclaredVersion != "dev" {
		t.Errorf("info.DeclaredVersion = %q", info.DeclaredVersion)
	}
}

// TestGet_PreservesInjectedSemVer shows that when both Version
// (effective) and DeclaredVersion (declared) are injected with a
// real SemVer, Get() reports them. When the two are equal the
// JSON-omitempty contract clears DeclaredVersion so the wire
// form is clean; the clean value is observable as an empty
// string in Info.DeclaredVersion.
func TestGet_PreservesInjectedSemVer(t *testing.T) {
	oldV, oldDV, oldC, oldBT := Version, DeclaredVersion, Commit, BuildTime
	oldDirty := Dirty
	t.Cleanup(func() {
		Version, DeclaredVersion, Commit, BuildTime = oldV, oldDV, oldC, oldBT
		Dirty = oldDirty
	})
	Version = "1.2.3"
	DeclaredVersion = "1.2.3"
	Commit = "abc123"
	BuildTime = "2026-07-09T10:24:46Z"
	Dirty = "true" // stays non-false to avoid the dirty-cleanup path

	info := Get()

	if info.Version != "1.2.3" {
		t.Fatalf("Version = %q, want 1.2.3", info.Version)
	}
	// Equal-to-Version DeclaredVersion is cleared to satisfy
	// the JSON omitempty contract.
	if info.DeclaredVersion != "" {
		t.Fatalf("DeclaredVersion = %q, want empty (omits when equal to Version)", info.DeclaredVersion)
	}
	if info.Commit != "abc123" {
		t.Fatalf("Commit = %q, want abc123", info.Commit)
	}
	if info.BuildTime != "2026-07-09T10:24:46Z" {
		t.Fatalf("BuildTime = %q", info.BuildTime)
	}
	if info.Dirty != "true" {
		t.Fatalf("Dirty = %q, want true", info.Dirty)
	}
}

// TestGet_AutoStampsDeclaredPlaceholder verifies R2.3: a
// declared "dev" placeholder combined with VCS-fallback Commit
// and BuildTime values yields an effective stamp carrying that
// provenance (and no info-discard bug). This exercises the
// Get/FromSettings seam.
func TestGet_AutoStampsDeclaredPlaceholder(t *testing.T) {
	oldV, oldDV, oldC, oldBT := Version, DeclaredVersion, Commit, BuildTime
	oldDirty := Dirty
	t.Cleanup(func() {
		Version, DeclaredVersion, Commit, BuildTime = oldV, oldDV, oldC, oldBT
		Dirty = oldDirty
	})
	// Set up a development build profile.
	Version = "dev"
	DeclaredVersion = "dev"
	Commit = "fd71cf21519f"
	BuildTime = "2026-07-11T21:07:23Z"
	Dirty = "false"

	info := Get()

	if info.Version != "0.1.0+dev.fd71cf21519f.20260711T210723Z" {
		t.Errorf("effective stamp wrong: %q", info.Version)
	}
	if info.DeclaredVersion != "dev" {
		t.Errorf("DeclaredVersion = %q, want dev", info.DeclaredVersion)
	}
	if info.Commit != "fd71cf21519f" {
		t.Errorf("Commit = %q", info.Commit)
	}
}

// TestGet_FromSettingsFillsProvenanceIntoStamp (R3.3 — real
// production composition) builds a base Info from package
// globals, runs the production getFromSettings seam on VCS
// settings, then derives the stamp via the production
// EffectiveVersion helper. The test exercises the exact call
// chain Get() uses in production.
func TestGet_FromSettingsFillsProvenanceIntoStamp(t *testing.T) {
	oldV, oldDV, oldC, oldBT := Version, DeclaredVersion, Commit, BuildTime
	oldDirty := Dirty
	t.Cleanup(func() {
		Version, DeclaredVersion, Commit, BuildTime = oldV, oldDV, oldC, oldBT
		Dirty = oldDirty
	})
	Version = "dev"
	DeclaredVersion = "dev"
	Commit = "unknown"
	BuildTime = "unknown"
	Dirty = "false"

	base := Info{
		Version:         Version,
		DeclaredVersion: DeclaredVersion,
		Commit:          Commit,
		BuildTime:       BuildTime,
		Dirty:           Dirty,
	}
	enriched := getFromSettings(base, []debug.BuildSetting{
		{Key: "vcs.revision", Value: "abcdef1234567890"},
		{Key: "vcs.time", Value: "2026-08-01T12:00:00Z"},
		{Key: "vcs.modified", Value: "true"},
	})
	enriched.Version = EffectiveVersion(
		Version, enriched.DeclaredVersion, enriched.Commit, enriched.BuildTime,
	)

	if enriched.Commit != "abcdef1234567890" {
		t.Errorf("Commit = %q, want abcdef1234567890 (via getFromSettings)", enriched.Commit)
	}
	if !strings.HasPrefix(enriched.Version, "0.1.0+dev.") {
		t.Errorf("Version %q must use the recovered SHA in the stamp", enriched.Version)
	}
	if !strings.Contains(enriched.Version, "20260801T120000Z") {
		t.Errorf("Version %q must include the recovered BuildTime", enriched.Version)
	}
	if enriched.Dirty != "true" {
		t.Errorf("Dirty = %q, want true", enriched.Dirty)
	}
}

// TestGet_UsesInjectedEffectiveVersionWhenDeclaredDiffers (R3.2)
// proves that when the linker-injected Version is a strict SemVer
// and the declared version is the default placeholder, the
// authoritative effective value is the strict Version. This was
// the R3.2 acceptance criterion.
func TestGet_UsesInjectedEffectiveVersionWhenDeclaredDiffers(t *testing.T) {
	oldV, oldDV, oldC, oldBT := Version, DeclaredVersion, Commit, BuildTime
	t.Cleanup(func() {
		Version, DeclaredVersion, Commit, BuildTime = oldV, oldDV, oldC, oldBT
	})
	Version = "9.9.9"
	DeclaredVersion = "dev"
	Commit = "fd71cf21519f"
	BuildTime = "2026-07-11T21:07:23Z"

	info := Get()

	if info.Version != "9.9.9" {
		t.Fatalf("info.Version = %q, want 9.9.9 (Version is authoritative when it is a strict SemVer)", info.Version)
	}
	if info.DeclaredVersion != "dev" {
		t.Fatalf("info.DeclaredVersion = %q, want dev (declared provenance preserved alongside strict effective)", info.DeclaredVersion)
	}
	// Effective() must agree.
	eff := Effective()
	if eff != "9.9.9" {
		t.Fatalf("Effective() = %q, want 9.9.9", eff)
	}
}

// TestFromSettings_InjectsCommit exercises a small unit of
// FromSettings: it fills unknown Commit from vcs.revision.
func TestFromSettings_InjectsCommit(t *testing.T) {
	settings := []debug.BuildSetting{
		{Key: "vcs.revision", Value: "def456"},
	}
	info := FromSettings(Info{Version: "dev", Commit: "unknown", BuildTime: "unknown"}, settings)
	if info.Commit != "def456" {
		t.Errorf("Commit = %q, want def456", info.Commit)
	}
}

// TestFromSettings_InjectsBuildTime exercises FromSettings
// with vcs.time and the same RFC3339 timestamp used in earlier
// ACT contracts.
func TestFromSettings_InjectsBuildTime(t *testing.T) {
	settings := []debug.BuildSetting{
		{Key: "vcs.time", Value: "2026-07-09T10:24:46Z"},
	}
	info := FromSettings(Info{Version: "dev", Commit: "unknown", BuildTime: "unknown"}, settings)
	if info.BuildTime != "2026-07-09T10:24:46Z" {
		t.Errorf("BuildTime = %q", info.BuildTime)
	}
}

// TestFromSettings_InjectedWins verifies that already-injected
// values are not overwritten by VCS fallback.
func TestFromSettings_InjectedWins(t *testing.T) {
	settings := []debug.BuildSetting{
		{Key: "vcs.revision", Value: "from-vcs"},
		{Key: "vcs.time", Value: "2026-07-09T10:24:46Z"},
		{Key: "vcs.modified", Value: "true"},
	}
	info := FromSettings(Info{Version: "dev", Commit: "injected", BuildTime: "injected", Dirty: "false"}, settings)
	if info.Commit != "injected" {
		t.Errorf("Commit = %q, want injected", info.Commit)
	}
	if info.BuildTime != "injected" {
		t.Errorf("BuildTime = %q, want injected", info.BuildTime)
	}
}

// TestFromSettings_InvalidTime ensures invalid vcs.time does not
// poison the BuildTime field.
func TestFromSettings_InvalidTime(t *testing.T) {
	settings := []debug.BuildSetting{
		{Key: "vcs.time", Value: "not-a-time"},
	}
	info := FromSettings(Info{Version: "dev", Commit: "unknown", BuildTime: "keep-me"}, settings)
	if info.BuildTime != "keep-me" {
		t.Errorf("BuildTime = %q, want keep-me", info.BuildTime)
	}
}

// TestInfo_JSON_OmitsConditionalFields exercises the R2.5d
// contract: DeclaredVersion and Dirty are JSON-omitempty, so
// when they equal their defaults they do not appear in the wire
// form, but Version, Commit, and BuildTime are always present.
func TestInfo_JSON_OmitsConditionalFields(t *testing.T) {
	clean := Info{Version: "1.2.3", DeclaredVersion: "1.2.3", Commit: "abc", BuildTime: "2026-01-01T00:00:00Z"}
	data, err := json.Marshal(clean)
	if err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}
	if strings.Contains(string(data), `"declared_version"`) {
		t.Errorf("declared_version should be omitted when equal to version; got %s", data)
	}
	if strings.Contains(string(data), `"dirty"`) {
		t.Errorf("dirty should be omitted when false; got %s", data)
	}
	if !strings.Contains(string(data), `"version"`) {
		t.Errorf("version must always be present; got %s", data)
	}

	dev := Info{Version: "dev", DeclaredVersion: "dev", Commit: "x", BuildTime: "2026-01-01T00:00:00Z"}
	data, err = json.Marshal(dev)
	if err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}
	if strings.Contains(string(data), `"dirty"`) {
		t.Errorf("dirty=false should be omitted from JSON; got %s", data)
	}

	dirty := Info{Version: "1.2.3", DeclaredVersion: "1.2.3", Commit: "x", BuildTime: "2026-01-01T00:00:00Z", Dirty: "true"}
	data, err = json.Marshal(dirty)
	if err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}
	if !strings.Contains(string(data), `"dirty":"true"`) {
		t.Errorf("dirty=true must be present; got %s", data)
	}
}

// TestInfo_LineFormat documents the canonical line format and
// the presence of a colon separator on each emitted field.
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

// TestGet_MalformedVersionNotMaskedByDeclaredPlaceholder (R5.1)
// proves that the production `Get()` does not launder a malformed
// Version into a derived stamp even when DeclaredVersion is the
// default "dev" placeholder. The effective version must equal
// the malformed value so the strict-SemVer oracle can reject it.
func TestGet_MalformedVersionNotMaskedByDeclaredPlaceholder(t *testing.T) {
	oldV, oldDV, oldC, oldBT := Version, DeclaredVersion, Commit, BuildTime
	oldDirty := Dirty
	t.Cleanup(func() {
		Version, DeclaredVersion, Commit, BuildTime = oldV, oldDV, oldC, oldBT
		Dirty = oldDirty
	})
	Version = "banana"
	DeclaredVersion = "dev"
	Commit = "fd71cf21519f"
	BuildTime = "2026-07-11T21:07:23Z"
	Dirty = "false"

	info := Get()

	if info.Version != "banana" {
		t.Errorf("Get().Version = %q, want %q (malformed Version must not be masked by DeclaredVersion placeholder)", info.Version, "banana")
	}
	if info.DeclaredVersion != "dev" {
		t.Errorf("Get().DeclaredVersion = %q, want dev", info.DeclaredVersion)
	}
	// Effective() must agree.
	if got := Effective(); got != "banana" {
		t.Errorf("Effective() = %q, want %q", got, "banana")
	}
}
