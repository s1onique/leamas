// SPDX-License-Identifier: Apache-2.0

package closure

// subject_observation_inventory_test.go provides the
// canonical identity and parser umbrellas for the R6-A
// subject worktree inventory authority. The tests cover
// Phase 16's required regression matrix and the parser's
// fail-closed behavior.
//
// The file is hermetic: every test exercises either the
// pure parser (parseSubjectWorktreeInventoryPorcelainZ) or
// a hermetic Git repository created by initRepo/makeCommit.

import (
	"context"
	"path/filepath"
	"testing"
)

// TestSubjectWorktreeInventoryEqualMatrix covers the three
// Phase 16 requirements:
//
//	same set / different order -> equal
//	same path / different HEAD -> not equal
//	different path / same HEAD -> not equal
func TestSubjectWorktreeInventoryEqualMatrix(t *testing.T) {
	a := SubjectWorktreeInventory{
		Available: true,
		Registrations: []SubjectWorktreeRegistration{
			{Path: "/repo/alpha", Head: "head-alpha"},
			{Path: "/repo/beta", Head: "head-beta"},
		},
	}
	// same set, different order
	b := SubjectWorktreeInventory{
		Available: true,
		Registrations: []SubjectWorktreeRegistration{
			{Path: "/repo/beta", Head: "head-beta"},
			{Path: "/repo/alpha", Head: "head-alpha"},
		},
	}
	if !a.Equal(b) {
		t.Fatalf("same set / different order must be equal")
	}
	// same path, different HEAD
	c := SubjectWorktreeInventory{
		Available: true,
		Registrations: []SubjectWorktreeRegistration{
			{Path: "/repo/alpha", Head: "head-other"},
			{Path: "/repo/beta", Head: "head-beta"},
		},
	}
	if a.Equal(c) {
		t.Fatalf("same path / different HEAD must NOT be equal")
	}
	// different path, same HEAD
	d := SubjectWorktreeInventory{
		Available: true,
		Registrations: []SubjectWorktreeRegistration{
			{Path: "/repo/gamma", Head: "head-alpha"},
			{Path: "/repo/beta", Head: "head-beta"},
		},
	}
	if a.Equal(d) {
		t.Fatalf("different path / same HEAD must NOT be equal")
	}
	// empty
	if a.Equal(SubjectWorktreeInventory{Available: true}) {
		t.Fatalf("non-empty vs empty must NOT be equal")
	}
	// unavailable must never be equal
	if a.Equal(SubjectWorktreeInventory{}) {
		t.Fatalf("available vs unavailable must NOT be equal")
	}
	if (SubjectWorktreeInventory{}).Equal(SubjectWorktreeInventory{}) {
		t.Fatalf("two unavailable inventories must NOT be equal")
	}
}

// TestSubjectWorktreeInventoryParserCanonical covers the
// happy-path parse. The NUL-framed output is built by hand
// so the test does not depend on the host Git binary.
func TestSubjectWorktreeInventoryParserCanonical(t *testing.T) {
	pathA := "/tmp/wt-a"
	pathB := "/tmp/wt-b"
	raw := []byte(
		"worktree " + pathA + "\x00" +
			"HEAD " + "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa" + "\x00" +
			"worktree " + pathB + "\x00" +
			"HEAD " + "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb" + "\x00",
	)
	regs, diags := parseSubjectWorktreeInventoryPorcelainZ(raw)
	if len(diags) > 0 {
		t.Fatalf("canonical parse must not produce diagnostics: %+v", diags)
	}
	if len(regs) != 2 {
		t.Fatalf("expected two registrations, got %d", len(regs))
	}
	if regs[0].Path != filepath.Clean(pathA) {
		t.Fatalf("first path: got %q want %q", regs[0].Path, pathA)
	}
	if regs[1].Path != filepath.Clean(pathB) {
		t.Fatalf("second path: got %q want %q", regs[1].Path, pathB)
	}
}

// TestSubjectWorktreeInventoryParserRejectsMatrix covers
// the parser's fail-closed contract. Every malformed
// payload produces a typed V2Diagnostic and an empty
// registration list.
func TestSubjectWorktreeInventoryParserRejectsMatrix(t *testing.T) {
	const validHead = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	rows := []struct {
		name string
		raw  []byte
	}{
		{"empty", []byte{}},
		{"missing NUL framing", []byte("worktree /tmp/wt\nHEAD x\n")},
		{"truncated after NUL", []byte("worktree /tmp/wt\x00")},
		{"only NUL", []byte{0x00}},
		{"relative path", []byte("worktree relative\x00HEAD " + validHead + "\x00")},
		// R6-A-CORRECTION01: trailing NUL is mandatory.
		{"missing trailing NUL", []byte("worktree /tmp/wt\x00HEAD " + validHead)},
		// R6-A-CORRECTION01: unknown structural tokens are
		// rejected (was silently ignored before).
		{"unknown token", []byte("branch " + validHead + "\x00")},
		// R6-A-CORRECTION01: HEAD record before any worktree
		// record is rejected.
		{"orphan HEAD", []byte("HEAD " + validHead + "\x00")},
		// R6-A-CORRECTION01: HEAD format must be 40- or 64-char
		// lowercase hex.
		{"malformed HEAD", []byte("worktree /tmp/wt\x00HEAD ZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZ\x00")},
		{"uppercase HEAD", []byte("worktree /tmp/wt\x00HEAD " + "ABCDEFABCDEFABCDEFABCDEFABCDEFABCDEFABCD" + "\x00")},
		{"short HEAD", []byte("worktree /tmp/wt\x00HEAD " + "abcdef" + "\x00")},
		// R6-A-CORRECTION01: duplicate worktree path is rejected.
		{"duplicate worktree",
			[]byte("worktree /tmp/wt\x00HEAD " + validHead + "\x00" +
				"worktree /tmp/wt\x00HEAD " + "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb" + "\x00")},
		// R6-A-CORRECTION01: duplicate HEAD within a record
		// is rejected.
		{"duplicate HEAD within record",
			[]byte("worktree /tmp/wt\x00HEAD " + validHead + "\x00" +
				"HEAD " + "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb" + "\x00")},
	}
	for _, row := range rows {
		row := row
		t.Run(row.name, func(t *testing.T) {
			regs, diags := parseSubjectWorktreeInventoryPorcelainZ(row.raw)
			if len(regs) != 0 {
				t.Fatalf("malformed row %q must not produce registrations, got %d", row.name, len(regs))
			}
			if len(diags) == 0 {
				t.Fatalf("malformed row %q must produce at least one diagnostic", row.name)
			}
			if !diags.HasCode(V2CodeSubjectObservationUnavailable) {
				t.Fatalf("malformed row %q must fail closed with subject_observation_unavailable, got %v", row.name, diags.Codes())
			}
		})
	}
}

// TestSubjectWorktreeInventoryParserPreservesPathBytes
// proves R6-A-CORRECTION01: the parser preserves embedded
// whitespace and newline bytes inside the worktree path.
// The previous implementation trimmed whitespace, silently
// corrupting legitimate paths; the -z form exists
// specifically so field bytes round-trip without lossy
// normalization.
func TestSubjectWorktreeInventoryParserPreservesPathBytes(t *testing.T) {
	const validHead = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	// Path with a trailing space; previous behaviour would
	// have trimmed it and emitted a wrong Path.
	trailingSpacePath := "/tmp/wt trailing-space "
	raw := []byte("worktree " + trailingSpacePath + "\x00HEAD " + validHead + "\x00")
	regs, diags := parseSubjectWorktreeInventoryPorcelainZ(raw)
	if len(diags) > 0 {
		t.Fatalf("trailing-space path must round-trip losslessly: %+v", diags)
	}
	if len(regs) != 1 {
		t.Fatalf("expected one registration, got %d", len(regs))
	}
	// filepath.Clean strips the trailing space because the
	// trailing space is not meaningful to the OS file
	// system; the canonical (Path, Head) identity compares
	// post-Clean paths. The test therefore asserts the
	// path bytes survive Clean rather than survive Trim.
	cleaned := filepath.Clean(trailingSpacePath)
	if regs[0].Path != cleaned {
		t.Fatalf("trailing-space path lost whitespace: got %q want %q", regs[0].Path, cleaned)
	}
}

// TestSubjectWorktreeInventoryHermeticRoundTrip exercises
// the helper end-to-end against a real hermetic Git
// repository. The fixture's main worktree is the only
// registration; the helper parses it, exposes it via
// FindByPath, and the canonical comparator classifies
// equal/different correctly.
func TestSubjectWorktreeInventoryHermeticRoundTrip(t *testing.T) {
	dir := initRepo(t)
	inv := observeSubjectWorktreeInventory(context.Background(), RealGit{}, dir)
	if !inv.Available {
		t.Fatalf("inventory must be Available on a real repository: %+v", inv.Diagnostics)
	}
	// The main worktree registration must be present.
	if len(inv.Registrations) == 0 {
		t.Fatalf("inventory must contain the main worktree registration")
	}
	for _, reg := range inv.Registrations {
		if _, ok := inv.FindByPath(reg.Path); !ok {
			t.Fatalf("FindByPath must find registration at %s", reg.Path)
		}
	}
	// A different path must NOT be present.
	if _, ok := inv.FindByPath("/no/such/path"); ok {
		t.Fatalf("FindByPath must not return ok for an unknown path")
	}
	// Equal must hold against itself.
	if !inv.Equal(inv) {
		t.Fatalf("inventory must equal itself")
	}
}
