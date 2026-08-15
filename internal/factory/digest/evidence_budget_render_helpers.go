// SPDX-License-Identifier: Apache-2.0

// Package digest: evidence_budget_render_helpers.go is the third
// file of the bounded evidence renderer split.
//
// The companion files are:
//   - evidence_budget.go         (policy, classifyFileEvidence, helpers)
//   - evidence_budget_render.go  (boundedWriter, boundedFileBlock,
//     renderChangedFilesAndDiffsBounded,
//     renderRangeFileEvidenceBoundedWithRunner)
//
// This file contains the smaller helpers used by the renderer:
//   - computeRangeMaxBytes       (F3 protection: range-stat pre-check)
//   - parseNumstatInt64          (helper for --numstat parsing)
//   - truncateLongLines          (per-line cap helper)
//
// Splitting across three files keeps each file within the LLM-friendly
// 400-line limit.
package digest

import (
	"fmt"
	"strings"
)

// computeRangeMaxBytes returns a per-path map of the larger
// blob size (in bytes) observed at either endpoint of the
// range. This is the F3 protection: when the working-tree
// file is small (deleted, replaced, or down-sized) but the
// historical diff is multi-MB, the returned value is still
// large enough to trigger the bounded policy.
//
// Implementation: rather than spawning 2N Git subprocesses
// (one per (endpoint, path) pair), we use `git cat-file
// --batch-check` with stdin formatted as one `<ref>:<path>`
// per line. The output is `<sha> <type> <size>` per line so
// we look up the SHA-256 from `git rev-parse` once and the
// sizes come back in one process. F15 (CORRECTION02).
//
// Returns an empty map on any error (e.g. invalid rangeSpec).
func computeRangeMaxBytes(runner gitRunner, repoRoot string,
	files []RangeFile, rangeSpec string) map[string]int64 {
	out := make(map[string]int64)
	if rangeSpec == "" || len(files) == 0 {
		return out
	}
	// F14 (CORRECTION03): for A...B ranges, anchor the
	// left endpoint to merge-base so divergent histories
	// with a large merge-base blob still trigger the
	// bounded policy. The original rangeSpec is still
	// passed to git diff downstream.
	left, right, ok := resolveRangeEndpoints(runner, repoRoot, rangeSpec)
	if !ok {
		return out
	}

	// F23 (CORRECTION06): the batched fast path joins
	// queries with '\n'. Git paths can legitimately
	// contain '\n' or '\r' (and POSIX filenames can
	// never contain NUL), so a single newline-containing
	// path turns one intended query into two protocol
	// records and corrupts the response/refs alignment.
	// Detect the hazard and route to the per-object
	// fallback, which uses `git cat-file -s <ref>:<path>`
	// per call (path-safe). Slower than the batch path
	// but only fires for pathological inputs.
	//
	// F26 (CORRECTION07) NOTE: whitespace in the path
	// (spaces, tabs) is a separate hazard class. The
	// newline-delimited framing here is SAFE for
	// whitespace in the input (no record-splitting),
	// but the OUTPUT parser (parseCatFileLine in
	// evidence_budget_blob.go) previously inspected
	// fields[1] and would mis-classify missing /
	// ambiguous / excluded responses for paths with
	// whitespace. F26 fixed the parser; the framing
	// here is unchanged for the whitespace case.
	paths := make([]string, 0, len(files))
	for _, f := range files {
		paths = append(paths, f.Path)
	}
	if hasAnyPathProtocolHazard(paths) {
		return computeRangeMaxBytesFallback(runner,
			repoRoot, files, left, right)
	}

	// Build the batch input. Each query is "<ref>:<path>".
	type key struct{ ref, path string }
	queries := make([]string, 0, 2*len(files))
	keyOrder := make([]key, 0, 2*len(files))
	for _, f := range files {
		queries = append(queries, left+":"+f.Path)
		keyOrder = append(keyOrder, key{left, f.Path})
		queries = append(queries, right+":"+f.Path)
		keyOrder = append(keyOrder, key{right, f.Path})
	}
	input := strings.Join(queries, "\n")
	batchOut, err := runner.RunWithStdin(repoRoot, []string{
		"cat-file", "--batch-check"}, input)
	if err != nil {
		// Fall back to single-call path; better than
		// nothing.
		return computeRangeMaxBytesFallback(runner, repoRoot,
			files, left, right)
	}
	lines := strings.Split(batchOut, "\n")
	for i, line := range lines {
		if i >= len(keyOrder) {
			break
		}
		if line == "" {
			continue
		}
		// Format: "<sha> <type> <size>"
		fields := strings.Fields(line)
		if len(fields) < 3 {
			continue
		}
		size := parseNumstatInt64(fields[2])
		if size <= 0 {
			continue
		}
		k := keyOrder[i]
		path := k.path
		if cur, ok := out[path]; !ok || size > cur {
			out[path] = size
		}
	}
	return out
}

// computeRangeMaxBytesFallback is the previous 2N-Git-process
// implementation, kept for the rare case where batched
// cat-file fails. It is far slower than the batch path and
// logs no op on success.
func computeRangeMaxBytesFallback(runner gitRunner, repoRoot string,
	files []RangeFile, left, right string) map[string]int64 {
	out := make(map[string]int64)
	for _, f := range files {
		var max int64
		if sz := blobSizeAt(runner, repoRoot, left, f.Path); sz > max {
			max = sz
		}
		if sz := blobSizeAt(runner, repoRoot, right, f.Path); sz > max {
			max = sz
		}
		if max > 0 {
			out[f.Path] = max
		}
	}
	return out
}

// splitRangeSpec parses a range spec into (left, right)
// endpoint refs.
//
// F14 (CORRECTION02): the previous implementation used
// `strings.Index(rangeSpec, "..")` which turned `A...B` into
// `left=A, right=".B"`. We now distinguish four forms:
//
//   - "A..B"  -> left=A, right=B
//   - "A...B" -> left=A, right=B   (the extra "." is treated
//     as the symmetric-difference
//     marker, not consumed here
//     because Leamas upstream
//     normalises ranges. We
//     preserve the same shape
//     so `git diff` itself
//     interprets the three-dot
//     semantics.)
//   - "A"     -> left=A, right=A
//   - ""      -> ok=false
func splitRangeSpec(rangeSpec string) (left, right string, ok bool) {
	if rangeSpec == "" {
		return "", "", false
	}
	// Three-dot form: A...B. We preserve the form so the
	// downstream `git diff` invocation interprets it. Both
	// endpoints are A and B.
	if idx := strings.Index(rangeSpec, "..."); idx >= 0 {
		return strings.TrimSpace(rangeSpec[:idx]),
			strings.TrimSpace(rangeSpec[idx+3:]), true
	}
	// Two-dot form: A..B.
	if idx := strings.Index(rangeSpec, ".."); idx >= 0 {
		return strings.TrimSpace(rangeSpec[:idx]),
			strings.TrimSpace(rangeSpec[idx+2:]), true
	}
	// Single ref.
	return rangeSpec, rangeSpec, true
}

// hasPathProtocolHazard reports whether `path` contains
// characters that would break the newline-delimited batch
// protocol used by `git cat-file --batch-check`. Git paths
// can legitimately contain '\n' and '\r' (the staged-status
// tests in this repository already exercise this fact), so
// the batched fast path is unsafe whenever ANY input path
// has such a character. F23 (CORRECTION06): the previous
// implementation joined queries with '\n' regardless, which
// would silently shift the response/refs alignment for
// newline-containing paths and emit plausible but
// incorrectly attributed identity evidence. NUL itself
// cannot occur in a POSIX filename, so '\n' and '\r' are
// the only realistic attack vectors.
func hasPathProtocolHazard(path string) bool {
	return strings.ContainsAny(path, "\n\r")
}

// hasAnyPathProtocolHazard reports whether any path in the
// given slice contains '\n' or '\r'.
func hasAnyPathProtocolHazard(paths []string) bool {
	for _, p := range paths {
		if hasPathProtocolHazard(p) {
			return true
		}
	}
	return false
}

// blobSizeAt returns the size in bytes of the blob at
// `<ref>:<path>` or 0 if the path does not exist at that ref.
// Errors are silently treated as "no blob".
func blobSizeAt(runner gitRunner, repoRoot, ref, path string) int64 {
	if ref == "" || path == "" {
		return 0
	}
	out, err := runner.Run(repoRoot, []string{
		"cat-file", "-s", ref + ":" + path})
	if err != nil {
		return 0
	}
	var sz int64
	seenDigit := false
	for _, c := range out {
		if c >= '0' && c <= '9' {
			sz = sz*10 + int64(c-'0')
			seenDigit = true
			continue
		}
		if !seenDigit && (c == ' ' || c == '\t' || c == '\n' || c == '\r') {
			continue
		}
		break
	}
	if !seenDigit {
		return 0
	}
	return sz
}

// parseNumstatInt64 parses a `--numstat` (or `--batch-check`
// size) integer field, returning -1 for "-" (binary).
func parseNumstatInt64(s string) int64 {
	if s == "-" {
		return -1
	}
	var n int64
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0
		}
		n = n*10 + int64(c-'0')
	}
	return n
}

// truncateLongLines enforces MaxPerLineBytes on every physical
// line of `s`. Lines that exceed the cap are truncated and
// tagged with a deterministic marker. This is the second-line
// defense (F11).
func truncateLongLines(s string) string {
	if s == "" {
		return s
	}
	lines := strings.Split(s, "\n")
	var sb strings.Builder
	sb.Grow(len(s))
	for i, line := range lines {
		if len(line) > MaxPerLineBytes {
			fmt.Fprintf(&sb,
				"%s\n[truncated: line=%d bytes, "+
					"max=%d bytes]\n",
				line[:MaxPerLineBytes], len(line),
				MaxPerLineBytes)
			continue
		}
		sb.WriteString(line)
		if i < len(lines)-1 {
			sb.WriteByte('\n')
		}
	}
	return sb.String()
}

// resolveRangeEndpoints normalizes a range spec into
// (classificationLeft, classificationRight) where those
// refs are interpreted by Git as ordinary endpoints.
//
// F14 (CORRECTION03): for the two-dot form "A..B", both
// endpoints are returned as-is. For the three-dot form
// "A...B", `git diff A...B` actually compares the merge
// base of A and B against B, NOT A against B. The
// previous implementation classified against A and B;
// for a divergent history with a large merge-base blob
// and small A and B endpoints, the large merge-base blob
// then reaches the diff execution path and the bounded
// policy fails to fire. We therefore resolve "A...B"
// to `merge-base(A,B)..B` for classification/identity
// lookups while still passing the original spec to
// `git diff` for the actual diff text.
//
// For single-rev or empty inputs, both endpoints are
// the same ref. Returns ok=false on a Git merge-base
// failure (the caller can fall back to the literal
// endpoints).
func resolveRangeEndpoints(runner gitRunner, repoRoot,
	rangeSpec string) (left, right string, ok bool) {
	left, right, ok = splitRangeSpec(rangeSpec)
	if !ok {
		return "", "", false
	}
	// Three-dot form: re-anchor left to the merge base.
	if strings.Contains(rangeSpec, "...") {
		out, err := runner.Run(repoRoot, []string{
			"merge-base", left, right})
		if err != nil {
			return left, right, true
		}
		mb := strings.TrimSpace(out)
		if mb == "" {
			return left, right, true
		}
		return mb, right, true
	}
	return left, right, true
}
