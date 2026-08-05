package evaltest

import (
	"fmt"
	"strings"
)

// pathPolicySeparator is the single permitted path separator.
// The Closure Protocol v1 public contract defines the path
// language as platform-independent and "/" is the only
// structural separator; backslash is forbidden as an ordinary
// character. The constant mirrors the production
// closure.pathPolicySeparator so parity tests share the same
// invariant.
const pathPolicySeparator = "/"

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
// policy. Mirrors the production closure.portablePathValidate
// so the parity witness exercises the same invariant.
//
// Rule summary:
//   - separator is "/" only; backslash is rejected;
//   - leading "/" is rejected (absolute path);
//   - empty paths are rejected;
//   - NUL and other control characters are rejected;
//   - Windows-volume prefixes ([A-Za-z]:) are rejected;
//   - parent-traversal segments ("..") are rejected unless
//     allow_parent_segments is true;
//   - single-dot segments ("."/"./foo") are rejected unless
//     allow_dot is true and "." is the entire path;
//   - empty/trailing separators are rejected;
//   - the value must equal portablePathClean(value).
func portablePathValidate(path string, allowDot, allowParentSegments bool) error {
	if path == "" {
		return fmt.Errorf("must be a non-empty path")
	}
	if strings.ContainsRune(path, 0) {
		return fmt.Errorf("must not contain a NUL byte")
	}
	for _, r := range path {
		if r < 0x20 || r == 0x7f {
			return fmt.Errorf("must not contain control character U+%04X", r)
		}
	}
	if strings.ContainsRune(path, '\\') {
		return fmt.Errorf("must not contain a backslash; separator is /")
	}
	if path[0] == pathPolicySeparator[0] {
		return fmt.Errorf("must not be an absolute path (no leading separator)")
	}
	if len(path) >= 2 && isASCIILetter(path[0]) && path[1] == ':' {
		return fmt.Errorf("must not carry a Windows-volume prefix")
	}
	parts := strings.Split(path, pathPolicySeparator)
	for _, seg := range parts {
		if seg == "" {
			return fmt.Errorf("must not contain empty or trailing separators")
		}
		if seg == "." {
			if !allowDot || path != "." {
				return fmt.Errorf("must not contain single-dot segments")
			}
		}
		if seg == ".." && !allowParentSegments {
			return fmt.Errorf("must not contain a parent-traversal segment")
		}
	}
	if clean := portablePathClean(path); clean != path {
		return fmt.Errorf("must be lexically clean; canonical form is %q", clean)
	}
	return nil
}

// isASCIILetter reports whether b is an ASCII letter.
func isASCIILetter(b byte) bool {
	return (b >= 'A' && b <= 'Z') || (b >= 'a' && b <= 'z')
}

// portablePathAccepts is the extension-aware schema evaluator
// entry point for x-leamas-repository-relative-path. The function
// takes the *parsed* extension members plus the value and
// returns true when the value satisfies the policy. It does not
// hardcode any field name or value: every decision flows from
// the supplied extension members.
//
// The function fails closed on:
//   - missing required extension member;
//   - wrong extension-member type;
//   - unsupported separator value;
//   - malformed extension object.
func portablePathAccepts(value string, ext map[string]any) (bool, string) {
	if ext == nil {
		return false, "x-leamas-repository-relative-path extension missing"
	}
	allowDot, okDot := boolFromExt(ext, "allow_dot")
	if !okDot {
		return false, "x-leamas-repository-relative-path: allow_dot missing or wrong type"
	}
	allowParentSegments, okPar := boolFromExt(ext, "allow_parent_segments")
	if !okPar {
		return false, "x-leamas-repository-relative-path: allow_parent_segments missing or wrong type"
	}
	requireLexicallyClean, okLex := boolFromExt(ext, "require_lexically_clean")
	if !okLex {
		return false, "x-leamas-repository-relative-path: require_lexically_clean missing or wrong type"
	}
	separator, okSep := stringFromExt(ext, "separator")
	if !okSep {
		return false, "x-leamas-repository-relative-path: separator missing or wrong type"
	}
	if separator != pathPolicySeparator {
		return false, fmt.Sprintf("x-leamas-repository-relative-path: unsupported separator %q", separator)
	}
	// Leamas Closure Plan v1 mandates the canonical
	// invariant set; non-canonical values must be rejected
	// so an extension-aware consumer cannot silently accept
	// a malformed path policy.
	if !allowDot {
		return false, "x-leamas-repository-relative-path: noncanonical invariant allow_dot=false (must be true)"
	}
	if allowParentSegments {
		return false, "x-leamas-repository-relative-path: noncanonical invariant allow_parent_segments=true (must be false)"
	}
	if !requireLexicallyClean {
		return false, "x-leamas-repository-relative-path: noncanonical invariant require_lexically_clean=false (must be true)"
	}
	if err := portablePathValidate(value, allowDot, allowParentSegments); err != nil {
		return false, err.Error()
	}
	if requireLexicallyClean {
		clean := portablePathClean(value)
		if clean != value {
			return false, fmt.Sprintf("lexically unclean: canonical form is %q", clean)
		}
	}
	return true, ""
}

// boolFromExt returns the bool value of key in m and reports
// whether the key was present with the right type.
func boolFromExt(m map[string]any, key string) (bool, bool) {
	v, ok := m[key]
	if !ok {
		return false, false
	}
	b, ok := v.(bool)
	return b, ok
}

// stringFromExt returns the string value of key in m and reports
// whether the key was present with the right type.
func stringFromExt(m map[string]any, key string) (string, bool) {
	v, ok := m[key]
	if !ok {
		return "", false
	}
	s, ok := v.(string)
	return s, ok
}
