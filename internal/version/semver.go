package version

import (
	"regexp"
	"strings"
)

// SemVerCompatible is the SemVer base used to derive development
// stamps. It must always satisfy the canonical pack's compiler
// compatibility constraint (>= 0.1.0). Bump in lockstep with the
// canonical release floor.
const SemVerCompatible = "0.1.0"

// MaxCommitLen bounds the commit suffix embedded in the derived
// stamp. Twelve hex characters are the standard short-SHA width.
const MaxCommitLen = 12

// semverStrict is the canonical SemVer 2.0.0 grammar (the official
// suggested regex, verbatim) with two adjustments for Go's RE2
// engine:
//
//  1. \d is spelled out as [0-9].
//  2. Non-capturing groups (?:...) are spelled out as capturing
//     groups; capturing is harmless here because we never sub-match.
//
// The grammar enforces §2 (no leading zeroes on numeric identifiers),
// §10 (build metadata after "+" exists, dot-separated identifiers,
// each non-empty), and §11 (prerelease after "-", dot-separated
// identifiers, each non-empty, numeric identifiers without leading
// zeroes).
var semverStrict = regexp.MustCompile(
	`^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)` +
		`(?:-((0|[1-9][0-9]*|[0-9]*[a-zA-Z-][0-9a-zA-Z-]*)` +
		`(\.(0|[1-9][0-9]*|[0-9]*[a-zA-Z-][0-9a-zA-Z-]*))*))?` +
		`(?:\+([0-9a-zA-Z-]+(\.[0-9a-zA-Z-]+)*))?$`,
)

// SemVerParts is the parsed form of a strict SemVer 2.0.0 string.
//
// The numeric components and pre-release identifiers are kept as
// decimal strings so the precedence logic can avoid integer
// overflow for arbitrarily long valid numerics. Build metadata is
// preserved verbatim and ignored for ordering per SemVer 2.0.0
// §10.
type SemVerParts struct {
	Major  string
	Minor  string
	Patch  string
	Pre    []string
	Build  string
	Source string // original input, for diagnostics
}

// IsValidSemVer reports whether v is a syntactically valid
// SemVer 2.0.0 string. The validator is deliberately non-forgiving:
// it does not trim whitespace, does not accept leading zeroes on
// numeric identifiers, and rejects both empty prerelease and
// empty build identifiers. Callers that need relaxed input should
// trim explicitly.
func IsValidSemVer(v string) bool {
	return semverStrict.MatchString(v)
}

// ParseSemVer splits a strict SemVer into structured parts and a
// numeric-component string so that precedence comparisons remain
// correct for arbitrarily long values.
//
// ok is false when v is not a syntactically valid SemVer.
// semverGroupBuild is the regex group index of the build
// metadata for ParseSemVer. The full regex has the following
// numbered groups (counting opening parentheses from left to
// right):
//
//	1: major
//	2: minor
//	3: patch
//	4: pre-release joined
//	5: first pre-release identifier
//	6: trailing pre-release (\.ident)+ capture wrapper
//	7: build joined
//	8: trailing build (\.ident)+ capture wrapper
//
// Group index of the build-joined capture. The full regex has
// nine numbered groups:
//
//	1: major
//	2: minor
//	3: patch
//	4: pre-release joined
//	5: first pre-release identifier
//	6: trailing pre-release outer wrapper for (\.ident)*
//	7: trailing pre-release inner identifier
//	8: build-joined
//	9: build trailing (\.ident)* inner identifier
const semverGroupBuild = 8

func ParseSemVer(v string) (SemVerParts, bool) {
	m := semverStrict.FindStringSubmatch(v)
	if m == nil {
		return SemVerParts{}, false
	}
	pre := []string(nil)
	if m[4] != "" {
		pre = strings.Split(m[4], ".")
	}
	return SemVerParts{
		Major:  m[1],
		Minor:  m[2],
		Patch:  m[3],
		Pre:    pre,
		Build:  m[semverGroupBuild],
		Source: v,
	}, true
}

// IsPlaceholder reports whether v is exactly one of the recognised
// build-time placeholders: the empty string, "dev", or "unknown"
// (case-insensitive). No whitespace trimming: a placeholder is
// identified by exact equality, and " dev " or "\tunknown\t" is
// treated as a non-placeholder that must reach the strict-SemVer
// oracle for validation (which will reject it).
//
// The historical lenient-trim behaviour is intentionally removed
// so whitespace-wrapped values cannot silently auto-derive into
// a stamp the user did not ask for. The strict-SemVer oracle is
// the single authority on validity.
func IsPlaceholder(v string) bool {
	if v == "" {
		return true
	}
	switch strings.ToLower(v) {
	case "dev", "unknown":
		return true
	}
	return false
}
