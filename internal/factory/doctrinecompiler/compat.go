package doctrinecompiler

import (
	"fmt"
	"strings"

	"github.com/s1onique/leamas/internal/version"
)

// CheckCompilerCompatibility validates that the compiler version
// satisfies the lock's recorded compatibility constraint.
//
// The current contract accepts a comma-separated list of constraints
// (any of which may match) and supports a tiny prefix language:
//
//   - ""          — empty: any compiler is acceptable.
//   - ">=X"       — current compiler version >= X (SemVer).
//   - "MAJOR.x"   — current major matches the constraint.
//
// Precedence rules follow SemVer 2.0.0 §11:
//
//   - Build metadata after "+" is stripped before ordering; it
//     does not affect precedence.
//   - A pre-release suffix after "-" ranks strictly below the
//     same MAJOR.MINOR.PATCH without pre-release.
//
// Malformed SemVer inputs (leading zeroes, numeric prerelease
// identifiers with leading zeroes, missing patch, trailing `+`,
// empty prerelease/build identifiers, extra dot-components,
// arbitrary text) are rejected: a non-empty constraint requires
// a syntactically valid effective version. The placeholder
// strings "dev", "unknown", and "" are likewise rejected for
// non-empty constraints.
func CheckCompilerCompatibility(constraint, have string) error {
	c := strings.TrimSpace(constraint)
	if c == "" {
		return nil
	}
	// Whitespace inside the constraint string is parser syntax,
	// but the have value is the compiler identity. Inspect it
	// as-is so that whitespace-wrapped versions are rejected by
	// IsValidSemVer rather than silently coerced.
	h := have
	if h == "" || h == "dev" || h == "unknown" {
		return fmt.Errorf("compiler version unknown; refusing to verify lock with non-empty constraint %q", c)
	}
	if !version.IsValidSemVer(h) {
		return fmt.Errorf("compiler %q is not a valid SemVer 2.0.0 version; refusing to satisfy constraint %q", have, c)
	}
	for _, raw := range strings.Split(c, ",") {
		raw = strings.TrimSpace(raw)
		if raw == "" {
			continue
		}
		if strings.HasPrefix(raw, ">=") {
			want := strings.TrimSpace(strings.TrimPrefix(raw, ">="))
			if !version.IsValidSemVer(want) {
				continue
			}
			if compareSemver(h, want) >= 0 {
				return nil
			}
			continue
		}
		if strings.HasSuffix(raw, ".x") {
			wantMajor := strings.TrimSuffix(raw, ".x")
			haveMajor := majorVersion(h)
			if wantMajor == haveMajor {
				return nil
			}
			continue
		}
		if raw == h {
			return nil
		}
	}
	return fmt.Errorf("compiler %q does not satisfy compatibility %q", have, c)
}

// majorVersion extracts the major component from a dotted version.
// Used only by the legacy "MAJOR.x" constraint shape; called
// after IsValidSemVer, so the head is always a non-empty digit string.
func majorVersion(v string) string {
	i := strings.Index(v, ".")
	if i < 0 {
		return v
	}
	return v[:i]
}

// compareSemver returns -1, 0, or 1, comparing two SemVer strings
// per SemVer 2.0.0 §11. Both inputs must already satisfy
// IsValidSemVer. Numeric components and pre-release identifiers
// are compared as decimal strings (no integer arithmetic), so
// arbitrarily large versions are handled correctly.
func compareSemver(a, b string) int {
	pa, okA := version.ParseSemVer(a)
	if !okA {
		return strings.Compare(a, b)
	}
	pb, okB := version.ParseSemVer(b)
	if !okB {
		return strings.Compare(a, b)
	}
	if c := compareNumeric(pa.Major, pb.Major); c != 0 {
		return c
	}
	if c := compareNumeric(pa.Minor, pb.Minor); c != 0 {
		return c
	}
	if c := compareNumeric(pa.Patch, pb.Patch); c != 0 {
		return c
	}
	// Same major.minor.patch. Apply pre-release precedence per
	// SemVer §11: a version without pre-release ranks above the
	// same version with pre-release.
	if len(pa.Pre) == 0 && len(pb.Pre) == 0 {
		return 0
	}
	if len(pa.Pre) == 0 {
		return 1
	}
	if len(pb.Pre) == 0 {
		return -1
	}
	return comparePrerelease(pa.Pre, pb.Pre)
}

// compareNumeric compares two non-empty decimal strings without
// converting them to int. SemVer §2 forbids leading zeroes on
// numeric components other than the literal "0", so the
// following rules suffice:
//
//  1. Drop leading zeroes (then strip the trailing "0" if it
//     was originally "0").
//  2. Compare the lengths of the trimmed strings; longer
//     wins.
//  3. Tie-break with a lexicographic compare of the trimmed
//     strings.
//
// This works for any length of valid numeric SemVer identifier
// and never overflows.
func compareNumeric(a, b string) int {
	a = strings.TrimLeft(a, "0")
	b = strings.TrimLeft(b, "0")
	if a == "" {
		a = "0"
	}
	if b == "" {
		b = "0"
	}
	if len(a) != len(b) {
		if len(a) < len(b) {
			return -1
		}
		return 1
	}
	return strings.Compare(a, b)
}

// comparePrerelease compares two SemVer pre-release identifier
// lists using SemVer §11 rules. Numeric identifiers compare
// numerically (via compareNumeric); alphanumeric identifiers
// compare ASCII-lexically; numeric identifiers always rank below
// alphanumeric identifiers (a numeric never compares equal to
// any alphanumeric of the same length under ASCII). When all
// shared identifiers tie, the shorter list wins.
func comparePrerelease(a, b []string) int {
	n := len(a)
	if len(b) < n {
		n = len(b)
	}
	for i := 0; i < n; i++ {
		x, y := a[i], b[i]
		xNum := isAllDigits(x)
		yNum := isAllDigits(y)
		switch {
		case xNum && yNum:
			if c := compareNumeric(x, y); c != 0 {
				return c
			}
		case xNum && !yNum:
			return -1
		case !xNum && yNum:
			return 1
		default:
			if x < y {
				return -1
			}
			if x > y {
				return 1
			}
		}
	}
	if len(a) < len(b) {
		return -1
	}
	if len(a) > len(b) {
		return 1
	}
	return 0
}

func isAllDigits(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}
