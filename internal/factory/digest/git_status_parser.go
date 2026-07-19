// Package digest provides targeted digest generation for Git repositories.
// This file implements a focused, NUL-delimited parser for git
// `diff --name-status -z` output. It is the single source of truth for
// translating Git's authoritative change records into typed Go values.
//
// The parser handles the full set of status letters Git emits in
// `diff --name-status -z` for an unstaged or staged diff against a
// baseline: A, M, D, T (type change), R<score>, C<score>, U (unmerged),
// X (unknown), B (broken pairing). Parsing refuses malformed records
// rather than guessing.
//
// Shared parser invariants:
//   - Fields are delimited by NUL (\x00), not by whitespace or newlines.
//   - A status token followed by N ordinary paths is treated as one record.
//   - Rename / copy records consume two NUL-delimited paths.
package digest

import (
	"fmt"
	"strings"
)

// ChangeKind is the explicit normalized change kind for a Git record.
//
// The constants intentionally mirror the single-letter codes used in the
// manifest rendering so downstream consumers (stats, risk, evidence hashes)
// can compare typed values.
type ChangeKind string

// ChangeKind values.
const (
	// KindUntracked is reported for untracked files; Git does not emit
	// any record for them in `diff --name-status` so this kind is
	// derived from `ls-files --others --exclude-standard`.
	KindUntracked ChangeKind = "?"

	// KindAdded, KindModified, KindDeleted, KindTypeChanged,
	// KindRenamed, KindCopied, KindUnmerged, KindUnknown, KindBroken
	// correspond to the matching Git status letters (see
	// `git-diff-tree --help` and `git diff --help` for the canonical
	// list). The lowercase token forms from `--name-status` -z output
	// while the uppercase letters signal the unrewritten status.
	KindAdded       ChangeKind = "A"
	KindModified    ChangeKind = "M"
	KindDeleted     ChangeKind = "D"
	KindTypeChanged ChangeKind = "T"
	KindRenamed     ChangeKind = "R"
	KindCopied      ChangeKind = "C"
	KindUnmerged    ChangeKind = "U"
	KindUnknown     ChangeKind = "X"
	KindBrokenPair  ChangeKind = "B"
)

// IsInternalDeltaKind reports whether k is one of the kinds the parser
// can derive from `git diff --name-status -z` against a working-tree
// baseline. Untracked and unknown are handled separately.
func (k ChangeKind) IsInternalDeltaKind() bool {
	switch k {
	case KindAdded, KindModified, KindDeleted, KindTypeChanged,
		KindRenamed, KindCopied, KindUnmerged,
		KindUnknown, KindBrokenPair:
		return true
	default:
		return false
	}
}

// GitChange is one parsed record from `git diff --name-status -z`.
//
// Path is always the post-change path (the destination for renames/copies).
// OldPath is non-empty only for renames and copies.
type GitChange struct {
	Kind    ChangeKind
	Path    string
	OldPath string
}

// String renders the change in Git's diff-without-`-z` textual form.
//
// This is what the digest manifest uses internally; tests and downstream
// renderers should not depend on the exact format.
func (g GitChange) String() string {
	if g.OldPath != "" {
		return fmt.Sprintf("%s %s -> %s", string(g.Kind), g.OldPath, g.Path)
	}
	return fmt.Sprintf("%s %s", string(g.Kind), g.Path)
}

// ParseGitStatusRecords parses NUL-delimited `git diff --name-status -z`
// output into a slice of GitChange values.
//
// The input must be the exact bytes Git wrote; the parser does not split
// on newlines or whitespace. Records are returned in the order Git wrote
// them; callers that need deterministic output should sort by path.
//
// The parser rejects:
//   - empty input (returns an empty slice);
//   - an empty status token;
//   - unsupported status letters;
//   - ordinary records missing the path;
//   - renames or copies missing either path or having a malformed score;
//   - records with an empty destination path.
//
// The parser does not panic on malformed Git output.
func ParseGitStatusRecords(input string) ([]GitChange, error) {
	if input == "" {
		return nil, nil
	}

	// Split NUL-delimited records. The leading status token is always
	// the first field; subsequent fields have known shape per token.
	fields := strings.Split(input, "\x00")
	if len(fields) == 0 {
		return nil, nil
	}

	// Git always emits a trailing NUL after the final field, which yields
	// an empty trailing element. Drop empty trailing elements only when
	// the input ends with NUL. We keep empty interior fields intact so
	// that malformation (missing path) is still detectable.
	if n := len(fields); n > 0 && fields[n-1] == "" && strings.HasSuffix(input, "\x00") {
		fields = fields[:n-1]
	}

	var out []GitChange
	i := 0
	for i < len(fields) {
		token := fields[i]
		i++
		if token == "" {
			return nil, fmt.Errorf("git status record at field %d: empty status token", i-1)
		}

		switch {
		case token == "A", token == "M", token == "D", token == "T",
			token == "U", token == "X", token == "B":
			if i >= len(fields) {
				return nil, fmt.Errorf("git status record at field %d: missing path for %s record", i-1, token)
			}
			path := fields[i]
			i++
			if path == "" {
				return nil, fmt.Errorf("git status record at field %d: empty destination path for %s record", i-1, token)
			}
			out = append(out, GitChange{Kind: ChangeKind(token), Path: path})

		case strings.HasPrefix(token, "R"), strings.HasPrefix(token, "C"):
			kind := token[:1]
			score := token[1:]
			if score == "" {
				return nil, fmt.Errorf("git status record at field %d: missing similarity score for %s record", i-1, kind)
			}
			for _, c := range score {
				if c < '0' || c > '9' {
					return nil, fmt.Errorf("git status record at field %d: malformed similarity score %q for %s record", i-1, score, kind)
				}
			}
			if i+1 >= len(fields) {
				return nil, fmt.Errorf("git status record at field %d: truncated %s record (need old and new paths)", i-1, kind)
			}
			oldPath := fields[i]
			newPath := fields[i+1]
			i += 2
			if oldPath == "" {
				return nil, fmt.Errorf("git status record at field %d: empty old path for %s record", i-1, kind)
			}
			if newPath == "" {
				return nil, fmt.Errorf("git status record at field %d: empty new path for %s record", i-1, kind)
			}
			out = append(out, GitChange{Kind: ChangeKind(kind), Path: newPath, OldPath: oldPath})

		default:
			return nil, fmt.Errorf("git status record at field %d: unsupported status token %q", i-1, token)
		}
	}

	return out, nil
}

// NormalizeGitStatusToken reduces a Git status token to its single-letter kind.
//
// Tokens like "R100" or "C075" collapse to "R" / "C"; similarity scores are
// discarded. Tokens like "R" or "C" without a numeric score are rejected
// because they would correspond to a malformed `git diff --name-status -z`
// record; this stays consistent with ParseGitStatusRecords. Unknown letters
// return ("", false). Lowercase rewrites ("a", "m", "d", "t") are rejected
// for the same reason: Git's `-z` form emits only uppercase letters.
func NormalizeGitStatusToken(token string) (ChangeKind, bool) {
	switch token {
	case "A", "M", "D", "T", "U", "X", "B":
		return ChangeKind(token), true
	case "":
		return "", false
	}
	if len(token) >= 2 && (token[0] == 'R' || token[0] == 'C') {
		rest := token[1:]
		for _, c := range rest {
			if c < '0' || c > '9' {
				return "", false
			}
		}
		return ChangeKind(token[:1]), true
	}
	return "", false
}

// SplitNULRecords splits a NUL-delimited byte stream into ordered field
// tokens, retaining empty interior fields (the parser requires that to
// detect truncated records) and dropping a single trailing empty
// element that arises when the input ends with NUL. The returned slice
// mirrors `strings.Split(s, "\x00")` with that one adjustment.
func SplitNULRecords(input string) []string {
	if input == "" {
		return nil
	}
	parts := strings.Split(input, "\x00")
	if n := len(parts); n > 0 && parts[n-1] == "" && strings.HasSuffix(input, "\x00") {
		parts = parts[:n-1]
	}
	return parts
}
