package version

import (
	"strings"
)

// Effective is the build-time entry point: it returns the
// authoritative effective SemVer via the same derivation policy
// used by Get(). The companion-declared field defaults to the
// linker-injected DeclaredVersion; the helper dispatches to the
// canonical EffectiveVersion function below.
func Effective() string {
	return EffectiveVersion(Version, DeclaredVersion, Commit, BuildTime)
}

// EffectiveVersion is the authoritative derivation rule used by
// both the runtime Effective() and the Get() CLI handler.
//
// Policy:
//  1. If the linker-injected Version is a strict SemVer, it is
//     authoritative (so a build pipeline that stamps Version
//     directly without touching DeclaredVersion still gets the
//     stamped value).
//  2. If Version is a recognised placeholder ("", "dev",
//     "unknown", exact case-insensitive), fall through to
//     EffectiveFrom using DeclaredVersion and the VCS provenance.
//  3. If Version is neither a strict SemVer nor a recognised
//     placeholder, the value is malformed and is preserved
//     verbatim so the strict-SemVer oracle can reject it. A
//     malformed Version is never silently laundered into a
//     derived stamp.
//
// No TrimSpace: declared and version values are inspected as-is.
// Whitespace-wrapped SemVer is therefore rejected by the oracle
// rather than silently passed through.
func EffectiveVersion(version, declared, commit, buildTime string) string {
	if IsValidSemVer(version) {
		return version
	}
	if !IsPlaceholder(version) {
		// Malformed: do not derive. The strict-SemVer oracle is
		// the single authority on validity.
		return version
	}
	return EffectiveFrom(declared, commit, buildTime)
}

// EffectiveFrom implements the placeholder-derived fallback. It
// returns declared verbatim when declared is not a known
// placeholder; otherwise it constructs a SemVer-compatible stamp
// from the VCS-derived provenance.
//
// No TrimSpace: whitespace-wrapped placeholders fall through as
// unknown shapes and are rejected by the oracle rather than
// silently coerced.
func EffectiveFrom(declaredVersion, commit, buildTime string) string {
	if !IsPlaceholder(declaredVersion) {
		// Strict SemVer or malformed: pass verbatim so the
		// compatibility oracle is the single authority.
		return declaredVersion
	}
	meta := buildMetadata(commit, buildTime)
	if len(meta) == 0 {
		return SemVerCompatible
	}
	return SemVerCompatible + "+" + strings.Join(meta, ".")
}

// buildMetadata returns the SemVer build-metadata fragments that
// tag a development stamp with provenance. The order matters: the
// commit is emitted first so the result is stable across rebuilds
// that share a commit but move the timestamp.
func buildMetadata(commit, buildTime string) []string {
	var meta []string
	short := sanitizeCommit(commit)
	if short != "" {
		meta = append(meta, "dev."+short)
	}
	ts := sanitizeBuildTime(buildTime)
	if ts != "" {
		meta = append(meta, ts)
	}
	return meta
}

// sanitizeCommit reduces a commit identifier to a SemVer-build-metadata-legal
// short form. It strips any "-dirty" suffix that Git describe
// appends, drops the "unknown" placeholder, and truncates to
// MaxCommitLen characters.
//
// The result is empty when the input is empty, "unknown", or
// strips down to nothing after sanitisation.
func sanitizeCommit(commit string) string {
	c := strings.TrimSpace(commit)
	if c == "" {
		return ""
	}
	if strings.EqualFold(c, "unknown") {
		return ""
	}
	// Strip "-dirty" suffix when present.
	if i := strings.Index(c, "-"); i >= 0 {
		c = c[:i]
	}
	// Keep only [0-9A-Za-z] characters.
	var b strings.Builder
	for _, r := range c {
		switch {
		case r >= '0' && r <= '9',
			r >= 'A' && r <= 'Z',
			r >= 'a' && r <= 'z':
			b.WriteRune(r)
		}
	}
	if b.Len() == 0 {
		return ""
	}
	if b.Len() > MaxCommitLen {
		return b.String()[:MaxCommitLen]
	}
	return b.String()
}

// sanitizeBuildTime converts an RFC3339/UTC ISO timestamp (for
// example "2026-07-11T21:07:23Z") into a SemVer-build-metadata-legal
// string ("20260711T210723Z"). The original characters '-', ':'
// must be removed because SemVer build metadata only allows
// alphanumerics and dots.
func sanitizeBuildTime(ts string) string {
	ts = strings.TrimSpace(ts)
	if ts == "" {
		return ""
	}
	if strings.EqualFold(ts, "unknown") {
		return ""
	}
	// Trim a trailing "Z" so it can be re-appended after stripping
	// separators, keeping the literal "Z" suffix visible.
	tail := ""
	if strings.HasSuffix(ts, "Z") {
		tail = "Z"
		ts = strings.TrimSuffix(ts, "Z")
	}
	replacer := strings.NewReplacer("-", "", ":", "", "T", "T")
	cleaned := replacer.Replace(ts)
	if cleaned == "" {
		return tail
	}
	return cleaned + tail
}
