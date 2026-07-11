package doctrinecompiler

import (
	"fmt"
	"strings"
)

// CheckCompilerCompatibility validates that the compiler version
// satisfies the lock's recorded compatibility constraint.
//
// The current contract accepts a comma-separated list of constraints
// (any of which may match) and supports a tiny prefix language:
//
//   - ""          — empty: any compiler is acceptable.
//   - ">=X"       — current compiler version >= X (SemVer-ish).
//   - "MAJOR.x"   — current major matches the constraint.
//
// The have value "" or "dev" is rejected when the constraint is
// non-empty: a non-empty constraint requires a real, comparable
// version string. The check is therefore a meaningful guard rather
// than a vacuous tautology.
func CheckCompilerCompatibility(constraint, have string) error {
	c := strings.TrimSpace(constraint)
	if c == "" {
		return nil
	}
	h := strings.TrimSpace(have)
	if h == "" || h == "dev" {
		return fmt.Errorf("compiler version unknown; refusing to verify lock with non-empty constraint %q", c)
	}
	for _, raw := range strings.Split(c, ",") {
		raw = strings.TrimSpace(raw)
		if raw == "" {
			continue
		}
		if strings.HasPrefix(raw, ">=") {
			want := strings.TrimSpace(strings.TrimPrefix(raw, ">="))
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
func majorVersion(v string) string {
	i := strings.Index(v, ".")
	if i < 0 {
		return v
	}
	return v[:i]
}

// compareSemver returns -1, 0, or 1.
func compareSemver(a, b string) int {
	pa := strings.Split(a, ".")
	pb := strings.Split(b, ".")
	n := len(pa)
	if len(pb) > n {
		n = len(pb)
	}
	for i := 0; i < n; i++ {
		var xa, xb int
		if i < len(pa) {
			x := 0
			for _, r := range pa[i] {
				if r < '0' || r > '9' {
					break
				}
				x = x*10 + int(r-'0')
			}
			xa = x
		}
		if i < len(pb) {
			x := 0
			for _, r := range pb[i] {
				if r < '0' || r > '9' {
					break
				}
				x = x*10 + int(r-'0')
			}
			xb = x
		}
		if xa < xb {
			return -1
		}
		if xa > xb {
			return 1
		}
	}
	return 0
}
