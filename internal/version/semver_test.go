package version

import (
	"strings"
	"testing"
)

// TestIsValidSemVer_Accepted confirms the strict SemVer 2.0.0
// grammar accepts canonical versions.
func TestIsValidSemVer_Accepted(t *testing.T) {
	cases := []string{
		"0.0.0",
		"0.1.0",
		"1.2.3",
		"10.20.30",
		"1.2.3-alpha",
		"1.2.3+build.42",
		"1.2.3-alpha.1",
		"1.2.3-0.3.7",
		"1.2.3-x.7.z.92",
		"1.2.3+20130313144700",
		"1.2.3-alpha+build.7",
		"1.2.3-x-y-z",
		"1.2.3-0",
	}
	for _, in := range cases {
		if !IsValidSemVer(in) {
			t.Errorf("IsValidSemVer(%q) = false, want true", in)
		}
	}
}

// TestIsValidSemVer_Rejected covers all R2 strict-mode rejection
// cases: malformed numerics, missing patch, leading zeros,
// disallowed prerelease forms, disallowed build forms, empty
// identifiers, and arbitrary text.
func TestIsValidSemVer_Rejected(t *testing.T) {
	cases := []string{
		"banana",          // arbitrary text
		"1oops",           // numeric prefix followed by text
		"1.2",             // missing patch
		"01.2.3",          // leading zero on major
		"1.02.3",          // leading zero on minor
		"1.2.03",          // leading zero on patch
		"1.2.3+",          // empty build metadata
		"1.2.3-",          // empty pre-release
		"1.2.3.4",         // extra dot-component
		"v1.2.3",          // v-prefix
		"-1.2.3",          // leading dash
		"1.2.3-+build",    // pre then build with no pre id
		"1.2.3+build+",    // trailing + after build metadata
		"1.2.3-01",        // numeric prerelease with leading zero (R2.1a)
		"1.2.3-alpha.01",  // numeric prerelease id with leading zero (R2.1a)
		"1.2.3-00",        // double-zero numeric prerelease
		"1.2.3-alpha..1",  // empty identifier between dots (R2.1b)
		"1.2.3-.alpha",    // empty identifier at start
		"1.2.3-alpha.",    // empty identifier at end
		"1.2.3+build..42", // empty build identifier (R2.1b)
		"1.2.3+.build",    // empty build identifier at start
		"1.2.3+build.",    // empty build identifier at end
		" 1.2.3",          // leading whitespace (R2.1c — no trim)
		"1.2.3 ",          // trailing whitespace
	}
	for _, in := range cases {
		if IsValidSemVer(in) {
			t.Errorf("IsValidSemVer(%q) = true, want false", in)
		}
	}
}

// TestIsValidSemVer_RejectsTrimmedSpacing locks in the
// non-forgiving whitespace behaviour (R2.1c): the validator
// mirrors the Makefile guard, which likewise does not trim.
func TestIsValidSemVer_RejectsTrimmedSpacing(t *testing.T) {
	for _, in := range []string{"  1.2.3  ", " 1.2.3", "1.2.3 "} {
		if IsValidSemVer(in) {
			t.Errorf("IsValidSemVer(%q) = true, want false (no whitespace tolerance)", in)
		}
	}
}

// TestParseSemVer_LargeNumericsOnly checks that the parser
// never touches integer arithmetic for the numeric components,
// so SemVer parts with oversized numerics parse successfully
// without overflow.
func TestParseSemVer_LargeNumericsOnly(t *testing.T) {
	cases := []string{
		"9223372036854775808.0.0",  // MaxInt64 + 1
		"18446744073709551616.0.0", // 2^64
		"0.9223372036854775808.0",
		"0.0.9223372036854775808",
		"1.2.3-9223372036854775808",
	}
	for _, in := range cases {
		parts, ok := ParseSemVer(in)
		if !ok {
			t.Errorf("ParseSemVer(%q) ok=false, want true (no overflow path)", in)
		}
		if len(parts.Major) == 0 {
			t.Errorf("ParseSemVer(%q) returned empty Major", in)
		}
	}
}

// TestParseSemVer_AcceptanceAndStructure spot-checks the parsed
// structure for a single canonical version so regressions in the
// regex are caught.
func TestParseSemVer_AcceptanceAndStructure(t *testing.T) {
	parts, ok := ParseSemVer("1.2.3-alpha.1+build.42")
	if !ok {
		t.Fatal("ParseSemVer rejected a valid version")
	}
	if parts.Major != "1" || parts.Minor != "2" || parts.Patch != "3" {
		t.Errorf("got %s.%s.%s, want 1.2.3", parts.Major, parts.Minor, parts.Patch)
	}
	if strings.Join(parts.Pre, ".") != "alpha.1" {
		t.Errorf("pre = %v, want alpha.1", parts.Pre)
	}
	if parts.Build != "build.42" {
		t.Errorf("build = %q, want build.42", parts.Build)
	}
}

// TestIsPlaceholder_CoversKnownPlaceholders confirms the
// placeholder detector covers dev/unknown/empty (case-insensitive,
// exact equality, no trim) and rejects strict SemVer. Whitespace-
// wrapped placeholders are NOT placeholders — they are passed
// through to the strict-SemVer oracle for rejection.
func TestIsPlaceholder_CoversKnownPlaceholders(t *testing.T) {
	for _, in := range []string{"", "dev", "DEV", "Unknown", "UNKNOWN"} {
		if !IsPlaceholder(in) {
			t.Errorf("IsPlaceholder(%q) = false, want true", in)
		}
	}
	for _, in := range []string{"0.1.0", "0.1.0+dev.abc", "0.1.0-dev.abc", "1.2.3", "banana"} {
		if IsPlaceholder(in) {
			t.Errorf("IsPlaceholder(%q) = true, want false", in)
		}
	}
}

// TestIsPlaceholder_RejectsWhitespaceWrappedPlaceholders (R4.2)
// proves the R4 contract: whitespace around the placeholder
// literals is not a placeholder, so the value flows through to
// the strict-SemVer oracle for validation.
func TestIsPlaceholder_RejectsWhitespaceWrappedPlaceholders(t *testing.T) {
	for _, in := range []string{" dev ", "\tunknown\t", "   ", " DEV ", " unknown "} {
		if IsPlaceholder(in) {
			t.Errorf("IsPlaceholder(%q) = true, want false (whitespace-wrapped must reach the strict-SemVer oracle)", in)
		}
		got := EffectiveFrom(in, "fd71cf2", "2026-07-11T21:07:23Z")
		if got != in {
			t.Errorf("EffectiveFrom must pass %q verbatim; got %q", in, got)
		}
	}
}

// TestIsPlaceholder_PreservesMalformed confirms that malformed
// declared values (which the release pipeline rejects outright)
// are NOT silently replaced by an auto-stamp at runtime; the
// compatibility oracle is the single source of truth.
func TestIsPlaceholder_PreservesMalformed(t *testing.T) {
	for _, in := range []string{"banana", "1oops", "1.2", "01.2.3", "1.2.3+", "1.2.3.4"} {
		if IsPlaceholder(in) {
			t.Errorf("IsPlaceholder(%q) = true, want false (malformed must pass through to oracle)", in)
		}
		got := EffectiveFrom(in, "fd71cf2", "2026-07-11T21:07:23Z")
		if got != in {
			t.Errorf("EffectiveFrom(%q) = %q, want verbatim pass-through", in, got)
		}
	}
}

// TestIsValidSemVer_BuildMetadataIgnoresForOrdering documents
// the canonical SemVer §10 contract: two versions that differ
// only in build metadata are considered equal by the
// compatibility oracle (verified downstream in the
// doctrinecompiler package).
func TestIsValidSemVer_BuildMetadataIgnoresForOrdering(t *testing.T) {
	for _, in := range []string{
		"1.2.3",
		"1.2.3+build.42",
		"1.2.3+meta",
	} {
		if !IsValidSemVer(in) {
			t.Errorf("IsValidSemVer(%q) = false, want true", in)
		}
	}
}
