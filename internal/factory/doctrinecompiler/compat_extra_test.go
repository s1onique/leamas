package doctrinecompiler

import (
	"testing"

	"github.com/s1onique/leamas/internal/version"
)

// TestCompareSemver_TableDriven covers the SemVer precedence rules
// required by the canonical pack: build metadata after "+" is
// ignored for ordering; a pre-release after "-" ranks below the
// same version without pre-release. Per SemVer §11, both inputs
// MUST already satisfy IsValidSemVer; malformed inputs are not
// handled here.
func TestCompareSemver_TableDriven(t *testing.T) {
	cases := []struct {
		name string
		a, b string
		want int
	}{
		{"equal minor patch", "0.1.0", "0.1.0", 0},
		{"build metadata equal ignored", "0.1.0+dev.abc", "0.1.0", 0},
		{"build metadata differs but base equal", "0.1.0+dev.abc", "0.1.0+dev.def", 0},
		{"a lower major", "0.0.9", "0.1.0", -1},
		{"a higher minor", "0.2.0", "0.1.0", 1},
		{"a patch lower", "0.1.0", "0.1.1", -1},
		{"pre-release lower than no pre-release (a has pre)", "0.1.0-dev", "0.1.0", -1},
		{"pre-release lower than no pre-release (b has pre)", "0.1.0", "0.1.0-dev", 1},
		{"pre-release lex compare lower", "0.1.0-alpha", "0.1.0-beta", -1},
		{"pre-release numeric lower", "0.1.0-1", "0.1.0-2", -1},
		{"pre-release numeric vs alpha (numeric lower)", "0.1.0-1", "0.1.0-alpha", -1},
		{"pre-release shorter list ranks lower than longer tied", "0.1.0-alpha", "0.1.0-alpha.1", -1},
		{"build metadata ignored when pre-release differs", "0.1.0-pre+meta", "0.1.0", -1},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := compareSemver(tc.a, tc.b)
			if signum(got) != signum(tc.want) {
				t.Errorf("compareSemver(%q, %q) = %d, want sign %d", tc.a, tc.b, got, tc.want)
			}
		})
	}
}

// TestCompareSemver_OverflowSafety validates the R2.2 contract:
// components longer than math.MaxInt64 (9223372036854775807) parse
// and order correctly via the string-based compareNumeric helper,
// not int arithmetic.
func TestCompareSemver_OverflowSafety(t *testing.T) {
	cases := []struct {
		name     string
		a, b     string
		expected int
	}{
		{"max-int64 vs 2^64", "9223372036854775808.0.0", "18446744073709551616.0.0", -1},
		{"2^64 vs max-int64", "18446744073709551616.0.0", "9223372036854775808.0.0", 1},
		{"equal overflow", "9223372036854775808.0.0", "9223372036854775808.0.0", 0},
		{"small vs larger", "1.2.3", "9223372036854775808.0.0", -1},
		{"larger vs small", "9223372036854775808.0.0", "1.2.3", 1},
		{"length wins (10 vs 9)", "10000000000.0.0", "9999999999.0.0", 1},
		{"length wins (1 vs 0)", "1.0.0", "0.0.0", 1},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := compareSemver(tc.a, tc.b)
			if signum(got) != tc.expected {
				t.Errorf("compareSemver(%q, %q) = %d, want sign %d", tc.a, tc.b, got, tc.expected)
			}
		})
	}
}

// TestCheckCompilerCompatibility_WhitespaceWrappedHaveRejected
// (R3.1) proves that the runtime oracle does NOT silently trim
// whitespace on the have value. A whitespace-padded "1.2.3" is a
// malformed SemVer and must be rejected just like "banana".
func TestCheckCompilerCompatibility_WhitespaceWrappedHaveRejected(t *testing.T) {
	for _, in := range []string{" 1.2.3", "1.2.3 ", "\t1.2.3\t", "\n1.2.3\n"} {
		err := CheckCompilerCompatibility(">=0.1.0", in)
		if err == nil {
			t.Errorf("CheckCompilerCompatibility must reject whitespace-padded %q", in)
		}
	}
}

// TestEffectiveFrom_WhitespaceWrappedSemVerPreservedForRejection
// (R3.1) proves that EffectiveFrom passes declared whitespace-
// wrapped SemVer verbatim so the oracle (not the helper) is the
// one that rejects it.
func TestEffectiveFrom_WhitespaceWrappedSemVerPreservedForRejection(t *testing.T) {
	in := " 1.2.3 "
	got := version.EffectiveFrom(in, "fd71cf2", "2026-07-11T21:07:23Z")
	if got != in {
		t.Errorf("EffectiveFrom must pass %q verbatim; got %q", in, got)
	}
	// And the oracle round-trip should reject.
	if err := CheckCompilerCompatibility(">=0.1.0", got); err == nil {
		t.Errorf("oracle must reject %q after EffectiveFrom", got)
	}
}

// TestCheckCompilerCompatibility_BuildMetadataVsConstraint table
// exercises the constraint oracle with the realistic stamps the
// build pipeline produces, plus the full malformed matrix
// required by R2.5c (banana / 1oops / 1.2 / 01.2.3 / 1.2.3+ /
// 1.2.3.4 must all be rejected).
func TestCheckCompilerCompatibility_BuildMetadataVsConstraint(t *testing.T) {
	cases := []struct {
		name       string
		constraint string
		have       string
		wantErr    bool
	}{
		// Realistic build-stamp shapes.
		{"build-metadata satisfies floor", ">=0.1.0", "0.1.0+dev.fd71cf2", false},
		{"build-metadata above floor", ">=0.1.0", "0.2.0+dev.fd71cf2", false},
		{"pre-release fails floor", ">=0.1.0", "0.1.0-dev.fd71cf2", true},
		{"build-metadata below floor", ">=0.1.0", "0.0.9+dev.fd71cf2", true},
		{"strict floor satisfied by exact", ">=0.1.0", "0.1.0", false},
		{"unknown rejected by non-empty constraint", ">=0.1.0", "unknown", true},
		{"empty rejected by non-empty constraint", ">=0.1.0", "", true},
		{"1.x accepted with metadata", ">=1.0.0", "1.5.0+dev.abc", false},

		// R2.5c: malformed `have` values must be rejected. The
		// constraint oracle is the single authority; the release
		// pipeline rejects the same values via the Makefile regex.
		{"malformed arbitrary text rejected", ">=0.1.0", "banana", true},
		{"malformed numeric prefix rejected", ">=0.1.0", "1oops", true},
		{"malformed missing patch rejected", ">=0.1.0", "1.2", true},
		{"malformed leading zero rejected", ">=0.1.0", "01.2.3", true},
		{"malformed trailing plus rejected", ">=0.1.0", "1.2.3+", true},
		{"malformed extra dot component rejected", ">=0.1.0", "1.2.3.4", true},
		{"malformed numeric prerelease id leading zero", ">=0.1.0", "1.2.3-01", true},
		{"malformed empty prerelease between dots", ">=0.1.0", "1.2.3-alpha..1", true},
		{"malformed empty build between dots", ">=0.1.0", "1.2.3+build..42", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := CheckCompilerCompatibility(tc.constraint, tc.have)
			gotErr := err != nil
			if gotErr != tc.wantErr {
				t.Errorf("CheckCompilerCompatibility(%q, %q) err = %v, wantErr %v",
					tc.constraint, tc.have, err, tc.wantErr)
			}
		})
	}
}

func signum(x int) int {
	switch {
	case x < 0:
		return -1
	case x > 0:
		return 1
	default:
		return 0
	}
}
