package version

import (
	"strings"
	"testing"
)

// TestEffective_DevDerivesSemVerCompatible verifies that a declared
// "dev" version is replaced by a SemVer-compatible build identifier
// "<SemVerCompatible>+dev.<commit>.<timestamp>" so the binary can
// participate in the doctrine compatibility protocol.
func TestEffective_DevDerivesSemVerCompatible(t *testing.T) {
	got := EffectiveFrom("dev", "fd71cf21519f", "2026-07-11T21:07:23Z")
	want := "0.1.0+dev.fd71cf21519f.20260711T210723Z"
	if got != want {
		t.Errorf("EffectiveFrom dev = %q, want %q", got, want)
	}
}

// TestEffective_EmptyDerivesSemVerCompatible verifies that an empty
// declared version is treated like "dev" and auto-derived.
func TestEffective_EmptyDerivesSemVerCompatible(t *testing.T) {
	got := EffectiveFrom("", "fd71cf21519f", "2026-07-11T21:07:23Z")
	want := "0.1.0+dev.fd71cf21519f.20260711T210723Z"
	if got != want {
		t.Errorf("EffectiveFrom empty = %q, want %q", got, want)
	}
}

// TestEffective_UnknownDerivesSemVerCompatible verifies that the
// "unknown" placeholder is replaced by an auto-derived stamp.
func TestEffective_UnknownDerivesSemVerCompatible(t *testing.T) {
	got := EffectiveFrom("unknown", "fd71cf21519f", "2026-07-11T21:07:23Z")
	want := "0.1.0+dev.fd71cf21519f.20260711T210723Z"
	if got != want {
		t.Errorf("EffectiveFrom unknown = %q, want %q", got, want)
	}
}

// TestEffective_DeclaredUnchangedWhenAlreadySemVer verifies that a
// real SemVer declared version is preserved verbatim; release
// builds keep their declared identity.
func TestEffective_DeclaredUnchangedWhenAlreadySemVer(t *testing.T) {
	cases := []string{"0.1.0", "0.2.0", "1.5.0", "0.1.0+abc.123"}
	for _, in := range cases {
		if got := EffectiveFrom(in, "fd71cf21519f", "2026-07-11T21:07:23Z"); got != in {
			t.Errorf("EffectiveFrom(%q) = %q, want unchanged", in, got)
		}
	}
}

// TestEffective_CommitSanitized verifies the commit id is reduced
// to the first 12 hex characters when it is longer.
func TestEffective_CommitSanitized(t *testing.T) {
	got := EffectiveFrom("dev", "fd71cf21519fbeefcafe", "2026-07-11T21:07:23Z")
	if !strings.Contains(got, "fd71cf21519f") {
		t.Errorf("effective %q must contain short commit", got)
	}
	if strings.Contains(got, "beefcafe") {
		t.Errorf("effective %q must not contain long tail of commit", got)
	}
}

// TestEffective_DirtyCommitStripsMarker verifies that a "fd71cf2-dirty"
// commit (a common Git describe output) has its "-dirty" suffix
// stripped before the stamp is computed.
func TestEffective_DirtyCommitStripsMarker(t *testing.T) {
	got := EffectiveFrom("dev", "fd71cf2-dirty", "2026-07-11T21:07:23Z")
	if strings.Contains(got, "dirty") {
		t.Errorf("effective %q must not contain 'dirty' marker", got)
	}
}

// TestEffective_BuildMetadataIsSemVerSafe verifies the build-time
// stamp uses only SemVer-build-metadata-legal characters
// (alphanumerics and dots).
func TestEffective_BuildMetadataIsSemVerSafe(t *testing.T) {
	got := EffectiveFrom("dev", "fd71cf2", "2026-07-11T21:07:23Z")
	i := strings.Index(got, "+")
	if i < 0 {
		t.Fatalf("effective %q missing '+' separator", got)
	}
	meta := got[i+1:]
	for _, r := range meta {
		ok := (r >= '0' && r <= '9') ||
			(r >= 'A' && r <= 'Z') ||
			(r >= 'a' && r <= 'z') ||
			r == '.'
		if !ok {
			t.Errorf("effective %q has illegal SemVer build char %q at offset %d", got, r, i+1)
			break
		}
	}
}

// TestEffective_NoCommitNoTimestamp verifies that with neither a
// commit nor a timestamp the helper still produces a valid SemVer
// stamp (the bare base version).
func TestEffective_NoCommitNoTimestamp(t *testing.T) {
	got := EffectiveFrom("dev", "unknown", "unknown")
	if got != SemVerCompatible {
		t.Errorf("effective %q, want %q", got, SemVerCompatible)
	}
}

// TestEffective_WhitespaceWrappedDeclaredPassesVerbatim (R3.1) —
// EffectiveFrom does NOT trim whitespace; a whitespace-padded
// "SemVer" is preserved for the oracle to reject.
func TestEffective_WhitespaceWrappedDeclaredPassesVerbatim(t *testing.T) {
	got := EffectiveFrom(" 1.2.3 ", "fd71cf2", "2026-07-11T21:07:23Z")
	if got != " 1.2.3 " {
		t.Errorf("EffectiveFrom must not trim; got %q, want %q", got, " 1.2.3 ")
	}
}

// TestEffectiveVersion_AuthoritativeOverDeclared (R3.2) — when
// the linker-injected Version is already a strict SemVer, it is
// authoritative regardless of what the declared value says.
func TestEffectiveVersion_AuthoritativeOverDeclared(t *testing.T) {
	got := EffectiveVersion("9.9.9", "dev", "fd71cf2", "2026-07-11T21:07:23Z")
	want := "9.9.9"
	if got != want {
		t.Errorf("EffectiveVersion(version=9.9.9, declared=dev) = %q, want %q", got, want)
	}
}

// TestEffectiveVersion_FallsThroughToDerived (R3.2) — when Version
// is a placeholder, the canonical helper derives the stamp from
// the declared value plus the VCS provenance.
func TestEffectiveVersion_FallsThroughToDerived(t *testing.T) {
	got := EffectiveVersion("dev", "dev", "fd71cf21519f", "2026-07-11T21:07:23Z")
	want := "0.1.0+dev.fd71cf21519f.20260711T210723Z"
	if got != want {
		t.Errorf("EffectiveVersion(dev, dev, …) = %q, want %q", got, want)
	}
}

// TestEffectiveVersion_MalformedVersionPreserved (R5.1) proves
// that a malformed Version does NOT silently derive a stamp from
// the placeholder DeclaredVersion. The malformed value must reach
// the strict-SemVer oracle for rejection, not be laundered.
func TestEffectiveVersion_MalformedVersionPreserved(t *testing.T) {
	cases := []string{
		"banana",
		"1oops",
		"1.2",
		"01.2.3",
		"1.2.3+",
		"1.2.3.4",
		"1.2.3-01",
		"1.2.3-alpha..1",
		" 1.2.3 ", // whitespace-wrapped
	}
	for _, in := range cases {
		got := EffectiveVersion(in, "dev", "fd71cf21519f", "2026-07-11T21:07:23Z")
		if got != in {
			t.Errorf("EffectiveVersion(%q, \"dev\", …) = %q, want %q (malformed must not derive)", in, got, in)
		}
	}
}
