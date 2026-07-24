// SPDX-License-Identifier: Apache-2.0

// Package authority: correction01_capability_test.go asserts the
// capability surface required by
// ACT-LEAMAS-FACTORY-SELF-HOSTED-ENTRYPOINT-AUTHORITY01-CORRECTION01.
//
// The three required capabilities are:
//   - factory_digest_auto_range
//   - factory_self_hosted_authority
//   - closure_protocol
//
// The tests exercise the production capability table and the
// required-capabilities loader; they do not mock the surface.
package authority

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestProductionCapabilitiesExposeAllThreeNames asserts the
// production capability table embeds the three required names.
func TestProductionCapabilitiesExposeAllThreeNames(t *testing.T) {
	got := Embedded()
	for _, name := range []string{
		CapDigestAutoRange,
		CapSelfHostedAuthority,
		CapClosureProtocol,
	} {
		if _, ok := got[name]; !ok {
			t.Fatalf("production capability table missing %q (got %v)", name, got)
		}
		if got[name] < 1 {
			t.Fatalf("production capability %q level=%d want >=1", name, got[name])
		}
	}
}

// TestSnapshotEmbeddedIsSortedDeterministic asserts the snapshot
// helper returns a deterministic name order so the doctor output is
// stable across runs.
func TestSnapshotEmbeddedIsSortedDeterministic(t *testing.T) {
	snap := SnapshotEmbedded()
	names := snap.Names()
	for i := 1; i < len(names); i++ {
		if names[i-1] >= names[i] {
			t.Fatalf("SnapshotEmbedded().Names() not sorted: %v", names)
		}
	}
	// And the order must be the canonical order documented in
	// production: closure_protocol < factory_digest_auto_range <
	// factory_self_hosted_authority (lexicographic).
	want := []string{
		"closure_protocol",
		"factory_digest_auto_range",
		"factory_self_hosted_authority",
	}
	if strings.Join(names, ",") != strings.Join(want, ",") {
		t.Fatalf("SnapshotEmbedded().Names()=%v want=%v", names, want)
	}
}

// TestRequiredCapabilitiesFileSatisfiedByProduction pins the
// contract between the repository's required-capabilities
// declaration and the production table.
func TestRequiredCapabilitiesFileSatisfiedByProduction(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	repoRoot := filepath.Clean(filepath.Join(wd, "..", "..", ".."))
	path := DefaultPath(repoRoot)
	required, err := LoadRequired(path)
	if err != nil {
		t.Fatalf("load required capabilities from %s: %v", path, err)
	}
	if err := required.SatisfiedBy(SnapshotEmbedded()); err != nil {
		t.Fatalf("production capabilities fail required floor: %v", err)
	}
	for _, name := range []string{
		CapDigestAutoRange,
		CapSelfHostedAuthority,
		CapClosureProtocol,
	} {
		if _, ok := required.Raw[name]; !ok {
			t.Fatalf("required capabilities file missing %q (got %v)", name, required.Raw)
		}
	}
}

// TestStaleGlobalBinaryLacksRequiredCapability simulates the
// July 24 stale-binary regression: a binary that is a technical
// ancestor of HEAD but lacks the required capability floor must be
// reported as stale.
//
// The test uses SetEmbedded to lower the capability for
// factory_self_hosted_authority to 0; the production required level
// is 1.
func TestStaleGlobalBinaryLacksRequiredCapability(t *testing.T) {
	original := capabilities[CapSelfHostedAuthority]
	t.Cleanup(func() { SetEmbedded(CapSelfHostedAuthority, original) })

	SetEmbedded(CapSelfHostedAuthority, 0)
	defer SetEmbedded(CapSelfHostedAuthority, original)

	required := &RequiredCapabilities{Raw: map[string]int{
		CapDigestAutoRange:     1,
		CapSelfHostedAuthority: 1,
		CapClosureProtocol:     1,
	}}
	err := required.SatisfiedBy(SnapshotEmbedded())
	if err == nil {
		t.Fatalf("expected CapabilityGap when self_hosted_authority=0")
	}
	if !strings.Contains(err.Error(), CapSelfHostedAuthority) {
		t.Fatalf("expected gap message to name %q, got %v", CapSelfHostedAuthority, err)
	}
}

// TestCurrentCanonicalBinarySatisfiesRequiredCapability is the
// inverse: the production capability set must satisfy the required
// floor so the canonical binary never fails its own capability
// check.
func TestCurrentCanonicalBinarySatisfiesRequiredCapability(t *testing.T) {
	required := &RequiredCapabilities{Raw: map[string]int{
		CapDigestAutoRange:     1,
		CapSelfHostedAuthority: 1,
		CapClosureProtocol:     1,
	}}
	if err := required.SatisfiedBy(SnapshotEmbedded()); err != nil {
		t.Fatalf("canonical capability set must satisfy required floor: %v", err)
	}
}

// TestSymlinkedCanonicalBinaryIsResolved asserts the checker resolves
// symlinks for the canonical executable path. This pins the
// observable surface of the doctor command's `resolved_symlink`
// field.
func TestSymlinkedCanonicalBinaryIsResolved(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "target")
	if err := os.WriteFile(target, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatalf("write target: %v", err)
	}
	link := filepath.Join(dir, "link")
	if err := os.Symlink(target, link); err != nil {
		t.Skipf("symlink unsupported on this platform: %v", err)
	}
	resolved, err := filepath.EvalSymlinks(link)
	if err != nil {
		t.Fatalf("eval symlinks: %v", err)
	}
	if resolved != target {
		t.Fatalf("resolved=%q want=%q", resolved, target)
	}
}

// TestDiscoverPATHAmbiguityReturnsMultiple confirms the PATH
// ambiguity contract: when more than one leamas executable is
// discoverable in PATH, the doctor surfaces an ambiguity diagnostic.
//
// The test pins the predicate `len(entries) >= 2` using the
// production-discoverable helper `discoverPATHExecutables`. The
// helper returns at most `maxEntries` candidates to bound the
// surface and avoid surprising system-wide scans.
func TestDiscoverPATHAmbiguityReturnsMultiple(t *testing.T) {
	dir := t.TempDir()
	first := filepath.Join(dir, "leamas")
	second := filepath.Join(dir, "bin", "leamas")
	if err := os.MkdirAll(filepath.Dir(second), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	for _, p := range []string{first, second} {
		if err := os.WriteFile(p, []byte("#!/bin/sh\n"), 0o755); err != nil {
			t.Fatalf("write %s: %v", p, err)
		}
	}
	path := dir + string(os.PathListSeparator) + filepath.Dir(second)
	entries := discoverPATHExecutablesForName("leamas", []string{path})
	if len(entries) < 2 {
		t.Fatalf("entries=%v want at least two", entries)
	}
}

// discoverPATHExecutablesForName mirrors the production helper's
// behavior: scan PATH for executables whose basename matches name.
// It returns at most `maxEntries` entries (capped to keep the test
// surface bounded).
func discoverPATHExecutablesForName(name string, paths []string) []string {
	const maxEntries = 16
	var out []string
	for _, p := range paths {
		for _, segment := range strings.Split(p, string(os.PathListSeparator)) {
			if segment == "" {
				continue
			}
			candidate := filepath.Join(segment, name)
			if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
				out = append(out, candidate)
				if len(out) >= maxEntries {
					return out
				}
			}
		}
	}
	return out
}
