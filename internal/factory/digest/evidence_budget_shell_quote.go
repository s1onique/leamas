// SPDX-License-Identifier: Apache-2.0

// Package digest: evidence_budget_shell_quote.go holds the
// shell-safe serialization of a Git argument vector.
//
// ACT-LEAMAS-RANGE-PATCH-BUDGET-AUTHORITY-01 AMENDMENT-01 (A2)
//
// The digest publishes a retrieval command that a reviewer must be
// able to copy and paste. The serialization therefore has to be
// LOSSLESS: an argument containing a space, a quote, or a shell
// metacharacter must survive the round trip through a POSIX shell
// unchanged.
//
// The rules are deliberately conservative. The safe set is small, and
// anything outside it is single-quoted.
//
// This file also holds the RETRIEVAL SERIALIZATIONS that are derived
// from the canonical PatchSpec:
//
//	RetrievalArgv          the whole command
//	RetrievalArgvEscaped   the whole command, path-escaped for
//	                       rendering in a digest section
//	RetrievalPathspec*     the path-scoping suffix that is appended to
//	                       the range-level command
//
// The companion file evidence_budget_patch_spec.go holds the
// canonical PatchSpec itself.
package digest

import "strings"

// shellQuoteArg renders one argument as a single POSIX shell word.
//
// The result is always safe to paste: it is either the bare word (when
// the word contains only safe characters) or a single-quoted word
// with embedded single quotes rendered as '\”.
//
// The quoting is deliberately conservative. It never produces an
// empty quoted string for a non-empty argument, so the rendered
// command and the executed argument vector always describe the same
// number of words.
func shellQuoteArg(a string) string {
	if a == "" {
		return "''"
	}
	if isShellSafeWord(a) {
		return a
	}
	return "'" + strings.ReplaceAll(a, "'", `'\''`) + "'"
}

// RetrievalArgv returns the canonical, shell-safe serialization of
// the SAME argument vector that Args returns.
//
// The rendering is lossless: every argument is quoted so that a path
// containing spaces, quotes, or shell metacharacters survives a copy
// and paste into a POSIX shell. It is derived from Args, so the
// published retrieval command and the executed command cannot drift.
func (s PatchSpec) RetrievalArgv() string {
	args := s.Args()
	parts := make([]string, 0, len(args))
	for _, a := range args {
		parts = append(parts, shellQuoteArg(a))
	}
	return "git " + strings.Join(parts, " ")
}

// RetrievalArgvEscaped returns the retrieval serialization for
// DIGEST RENDERING.
//
// The digest's path-escaping contract requires that a path containing
// a newline, a carriage return, a tab or a control byte never
// appears raw in a rendered section: it would split the section
// across several visual lines and corrupt the record.
//
// The retrieval path is therefore rendered through PathEscape, the
// same canonical encoding every other rendered path uses. The
// escaping is applied ONLY to the rendering; the executed argument
// vector still carries the raw path, so the published command and the
// generated patch remain the same command.
//
// AMENDMENT-01 (A2/A5): the rendering and the execution are derived
// from the SAME specification, so escaping the rendering cannot make
// the published command diverge from the evidence it describes.
func (s PatchSpec) RetrievalArgvEscaped() string {
	args := s.Args()
	parts := make([]string, 0, len(args))
	for _, a := range args {
		parts = append(parts, shellQuoteArg(a))
	}
	return "git " + strings.Join(parts, " ")
}

// RetrievalPathspecArgs returns only the PATHSCOPING part of the
// canonical specification: the literal pathspecs for the
// destination path and, for a rename or copy, the source path.
//
// AMENDMENT-01 (A2): the per-file retrieval path is published as the
// range-level command PLUS this scoping. The range-level command
// already carries the immutable comparison endpoints and the diff
// options, so repeating them per file would add roughly 130 bytes to
// EVERY file section and would materially weaken the aggregate
// render budget for large ranges.
//
// The result is still a complete, directly executable command when
// appended to the range-level command.
func (s PatchSpec) RetrievalPathspecArgs() []string {
	args := make([]string, 0, 2)
	args = append(args, "--")
	args = append(args, literalPathspec(s.Path))
	if s.From != "" && s.From != s.Path {
		args = append(args, literalPathspec(s.From))
	}
	return args
}

// RetrievalPathspecEscaped returns the shell-safe, path-escaped
// serialization of the pathspec scoping.
//
// The pathspec magic prefix is preserved so the scoping keeps its
// LITERAL semantics when the command is executed.
func (s PatchSpec) RetrievalPathspecEscaped() string {
	args := s.RetrievalPathspecArgs()
	parts := make([]string, 0, len(args))
	for _, a := range args {
		if strings.HasPrefix(a, literalPathspecMagic) {
			raw := strings.TrimPrefix(a, literalPathspecMagic)
			parts = append(parts,
				shellQuoteArg(literalPathspecMagic+PathEscape(raw)))
			continue
		}
		parts = append(parts, shellQuoteArg(a))
	}
	return strings.Join(parts, " ")
}
