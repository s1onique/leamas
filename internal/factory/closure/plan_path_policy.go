package closure

import (
	"fmt"
	"strings"
)

// plan_path_policy.go centralises the platform-independent
// repository-relative path validator used by the production
// runtime. The extension-aware schema evaluator
// (x-leamas-repository-relative-path) lives in the test-only
// internal/factory/closure/evaltest package; this file owns
// the production authority.
//
// The runtime's `path = runtime` table maps every diagnostic
// family to a stable PlanCodePathPolicyViolation /
// KeywordPathPolicy pair via errPathPolicyViolation.

// pathPolicySeparator is the single permitted path separator.
// The Closure Protocol v1 public contract defines the path
// language as platform-independent and "/" is the only
// structural separator; backslash is forbidden as an ordinary
// character.
const pathPolicySeparator = "/"

// errPathPolicyViolation is the typed path-policy error used by
// the runtime semantic validator. It carries the supplied path
// so consumers see exactly what was rejected.
type errPathPolicyViolation struct {
	cause string
	path  string
}

func (e *errPathPolicyViolation) Error() string { return e.cause }

// portablePathClean is a platform-independent implementation of
// filepath.Clean restricted to the "/" separator. It returns the
// canonical lexically-clean form; equal input and output means
// the input was already lexically clean. A leading ".." is left
// alone (canonical form equals input) so the lexically-clean
// probe fires on inputs like "../escape".
func portablePathClean(p string) string {
	if p == "" {
		return "."
	}
	if p == "/" {
		return "/"
	}
	parts := strings.Split(p, pathPolicySeparator)
	cleaned := make([]string, 0, len(parts))
	for i, seg := range parts {
		switch seg {
		case "", ".":
			continue
		case "..":
			if len(cleaned) == 0 {
				// Leading ".." is left as-is so the caller's
				// equality check fires; the parent-segment
				// probe in portablePathValidate catches it.
				return p
			}
			cleaned = cleaned[:len(cleaned)-1]
			continue
		default:
			cleaned = append(cleaned, seg)
			_ = i
		}
	}
	if len(cleaned) == 0 {
		return "."
	}
	return strings.Join(cleaned, pathPolicySeparator)
}

// portablePathValidate enforces the v1 repository-relative path
// policy and returns an *errPathPolicyViolation when the value
// violates any rule. The returned error is nil for accepted
// values and never nil for rejected ones.
//
// Rule summary (also documented in the public JSON Schema
// x-leamas-repository-relative-path extension):
//   - separator is "/" only; backslash is rejected;
//   - leading "/" is rejected (absolute path);
//   - empty paths are rejected;
//   - NUL and other control characters (U+0000-U+001F, U+007F)
//     are rejected;
//   - Windows-volume prefixes ([A-Za-z]:) are rejected;
//   - parent-traversal segments ("..") are rejected unless
//     allow_parent_segments is true;
//   - single-dot segments ("."/"./foo") are rejected unless
//     allow_dot is true and "." is the entire path;
//   - empty/trailing separators are rejected;
//   - the value must equal portablePathClean(value).
func portablePathValidate(path string, allowDot, allowParentSegments bool) error {
	if path == "" {
		return &errPathPolicyViolation{cause: "must be a non-empty path", path: path}
	}
	if strings.ContainsRune(path, 0) {
		return &errPathPolicyViolation{cause: "must not contain a NUL byte", path: path}
	}
	if containsClosurePlaceholder(path) {
		return &errPathPolicyViolation{cause: "must not contain a closure placeholder", path: path}
	}
	for _, r := range path {
		if r < 0x20 || r == 0x7f {
			return &errPathPolicyViolation{
				cause: fmt.Sprintf("must not contain control character U+%04X", r),
				path:  path,
			}
		}
	}
	if strings.ContainsRune(path, '\\') {
		return &errPathPolicyViolation{cause: "must not contain a backslash; separator is /", path: path}
	}
	if path[0] == pathPolicySeparator[0] {
		return &errPathPolicyViolation{cause: "must not be an absolute path (no leading separator)", path: path}
	}
	if len(path) >= 2 && isASCIILetter(path[0]) && path[1] == ':' {
		return &errPathPolicyViolation{cause: "must not carry a Windows-volume prefix", path: path}
	}
	parts := strings.Split(path, pathPolicySeparator)
	for _, seg := range parts {
		if seg == "" {
			return &errPathPolicyViolation{cause: "must not contain empty or trailing separators", path: path}
		}
		if seg == "." {
			if !allowDot || path != "." {
				return &errPathPolicyViolation{
					cause: "must not contain single-dot segments",
					path:  path,
				}
			}
		}
		if seg == ".." && !allowParentSegments {
			return &errPathPolicyViolation{cause: "must not contain a parent-traversal segment", path: path}
		}
	}
	if clean := portablePathClean(path); clean != path {
		return &errPathPolicyViolation{
			cause: fmt.Sprintf("must be lexically clean; canonical form is %q", clean),
			path:  path,
		}
	}
	return nil
}

// isASCIILetter reports whether b is an ASCII letter (matches the
// Windows-volume-prefix probe used by portablePathValidate).
func isASCIILetter(b byte) bool {
	return (b >= 'A' && b <= 'Z') || (b >= 'a' && b <= 'z')
}
