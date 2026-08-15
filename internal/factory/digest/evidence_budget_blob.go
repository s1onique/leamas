// SPDX-License-Identifier: Apache-2.0

// Package digest: evidence_budget_blob.go contains the
// range-mode Git blob helpers (F13 historical identity +
// F15 batched cat-file lookup). Kept in its own file so
// evidence_budget.go stays under the LLM-friendly 400-line
// limit.
package digest

import "strings"

// BlobLookupStatus describes the result of a single
// `git cat-file --batch-check` query for a "<ref>:<path>"
// expression. F16 (CORRECTION04): when a blob is missing
// or unresolved, Git returns "<ref> <status>" lines such
// as "<hash> missing" or "<hash> ambiguous". Storing the
// raw "<ref>:<path>" string into a BlobOID field is
// incorrect identity semantics, so we surface the status
// explicitly and refuse to publish a partial OID.
//
// F20 (CORRECTION05): the zero value of BlobLookupStatus
// MUST NOT be BlobPresent. A zero value is what a missing
// map key returns (RangeBlobResult{}), and what
// rangeBlobOIDsBatch stores when cat-file fails or when
// the response has fewer records than the request. If
// BlobPresent is the zero value, a missing-key lookup
// falsely publishes "RangeBaseBlobStatus: PRESENT" with
// an empty OID. The new ordering puts BlobUnknown first
// so the zero value fail-closes.
type BlobLookupStatus int

const (
	// BlobUnknown is the zero value. It is what callers
	// get for a missing map key, for a failed batch
	// lookup, and for a short/incomplete batch response.
	// A PRESENT status MUST only be set by an explicit,
	// successful parse of a "<oid> blob <size>" line.
	BlobUnknown BlobLookupStatus = iota
	// BlobPresent: cat-file returned "<oid> blob <size>".
	// OID is recorded.
	BlobPresent
	// BlobMissing: cat-file reported "<hash> missing".
	// No OID is published.
	BlobMissing
	// BlobAmbiguous: cat-file reported "<hash> ambiguous".
	// No OID is published.
	BlobAmbiguous
	// BlobOther: any other non-present status (excluded,
	// not-a-blob, etc.). No OID is published.
	BlobOther
)

func (s BlobLookupStatus) String() string {
	switch s {
	case BlobPresent:
		return "PRESENT"
	case BlobMissing:
		return "MISSING"
	case BlobAmbiguous:
		return "AMBIGUOUS"
	case BlobUnknown:
		return "UNKNOWN"
	default:
		return "OTHER"
	}
}

// RangeBlobResult is one element of the F13 historical
// identity lookup. Status is always populated; OID is
// only populated when Status == BlobPresent.
//
// The zero value (Status == BlobUnknown, OID == "") is the
// safe default for "we have no information about this
// reference". F20 (CORRECTION05) made this invariant
// load-bearing: a render that publishes PRESENT with an
// empty OID is a fail-open bug.
type RangeBlobResult struct {
	OID    string
	Status BlobLookupStatus
}

// IsPresent reports whether the lookup succeeded with a
// resolved blob OID. This is the only safe predicate for
// deciding whether to publish the OID field. F20
// (CORRECTION05): the renderer must call IsPresent()
// rather than checking Status == BlobPresent directly,
// because a zero-value RangeBlobResult has
// Status == BlobUnknown and IsPresent() == false.
func (r RangeBlobResult) IsPresent() bool {
	return r.Status == BlobPresent && r.OID != ""
}

// rangeBlobOIDsBatch returns a map keyed by "<ref>:<path>"
// of the historical blob lookup result for each query. It
// uses `git cat-file --batch-check` (one process, N
// lookups) so the F13 path does not regress the bounded
// renderer performance on large ranges. F15 (CORRECTION02);
// pairs naturally with computeRangeMaxBytes.
//
// F16 (CORRECTION04): the parser now correctly interprets
// cat-file's "<oid> <type> <size>" or "<oid> <status>"
// response protocol. The previous implementation copied
// `fields[0]` into the BlobOID field unconditionally,
// which made `RangeBaseBlobOID: HEAD~2:big.txt` appear in
// emitted digests when the blob was missing. We now
// record the explicit status and refuse to publish a
// non-OID value in the BlobOID field.
//
// F19 (CORRECTION04): the invocation no longer passes
// `--follow-symlinks`. With that flag, Git reports the
// resolved target instead of the symlink blob itself and
// can emit two output lines for one input query (the
// symlink blob plus its target), shifting the
// response/refs alignment. For historical repository
// object identity, the default behavior — report the
// actual blob at "<ref>:<path>" — is the correct
// semantics.
//
// F20 (CORRECTION05): on batch failure (RunWithStdin err
// != nil) or short output (fewer usable lines than
// requested), the missing keys are left at the zero value
// RangeBlobResult{} whose Status is BlobUnknown. Callers
// that read `rangeBlobOIDMap[key]` for an absent key get
// Status == BlobUnknown, NOT BlobPresent. Combined with
// IsPresent() this gives fail-closed semantics.
func rangeBlobOIDsBatch(runner gitRunner, repoRoot string,
	refs []string) map[string]RangeBlobResult {
	out := make(map[string]RangeBlobResult)
	if len(refs) == 0 {
		return out
	}
	// F23 (CORRECTION06): refs are "<ref>:<path>". When
	// ANY path contains '\n' or '\r', the batched
	// fast path would shift the response/refs
	// alignment and emit plausible but incorrectly
	// attributed identity evidence. Route to the
	// per-object fallback which uses
	// `git rev-parse <ref>:<path>` per call
	// (path-safe).
	paths := make([]string, 0, len(refs))
	for _, ref := range refs {
		if idx := strings.Index(ref, ":"); idx >= 0 {
			paths = append(paths, ref[idx+1:])
		}
	}
	if hasAnyPathProtocolHazard(paths) {
		return rangeBlobOIDsBatchFallback(runner, repoRoot, refs)
	}

	input := strings.Join(refs, "\n")
	batchOut, err := runner.RunWithStdin(repoRoot, []string{
		"cat-file", "--batch-check"}, input)
	if err != nil {
		// F20 (CORRECTION05): on error, leave the map
		// empty so callers see BlobUnknown (fail-closed).
		return out
	}
	lines := strings.Split(batchOut, "\n")
	// cat-file --batch-check emits exactly one line per
	// input request in normal usage. Without
	// --follow-symlinks (F19) the one-to-one invariant
	// holds even for unresolved objects: "<hash>
	// missing" is still a single line. We only assign
	// results for refs[i] that have a usable response
	// line. Trailing empty lines from the trailing
	// newline of stdout do not count.
	assigned := 0
	for _, line := range lines {
		if assigned >= len(refs) {
			break
		}
		if line == "" {
			continue
		}
		out[refs[assigned]] = parseCatFileLine(line)
		assigned++
	}
	return out
}

// blobOIDAt is the path-safe per-object companion of
// rangeBlobOIDsBatch. Used by the F23 (CORRECTION06)
// fallback when ANY input path contains '\n' or '\r'
// and the batched fast path would corrupt
// response/refs alignment. Uses
// `git rev-parse <ref>:<path>` to obtain the SHA
// identity of the blob; missing objects return
// BlobMissing rather than polluting the result with
// a partial OID. rev-parse exits non-zero with no
// output when the path is not present at the ref.
func blobOIDAt(runner gitRunner, repoRoot,
	ref, path string) RangeBlobResult {
	if ref == "" || path == "" {
		return RangeBlobResult{Status: BlobUnknown}
	}
	out, err := runner.Run(repoRoot, []string{
		"rev-parse", ref + ":" + path})
	if err != nil {
		return RangeBlobResult{Status: BlobMissing}
	}
	oid := strings.TrimSpace(out)
	if !isHexOID(oid) {
		return RangeBlobResult{Status: BlobUnknown}
	}
	return RangeBlobResult{OID: oid, Status: BlobPresent}
}

// isHexOID is a tiny guard for the per-object fallback.
// A valid SHA-1 OID is 40 hex chars; SHA-256 is 64.
// Accept either.
func isHexOID(s string) bool {
	n := len(s)
	if n != 40 && n != 64 {
		return false
	}
	for _, c := range s {
		if !((c >= '0' && c <= '9') ||
			(c >= 'a' && c <= 'f') ||
			(c >= 'A' && c <= 'F')) {
			return false
		}
	}
	return true
}

// rangeBlobOIDsBatchFallback is the path-safe
// per-object companion of rangeBlobOIDsBatch. Used
// by F23 (CORRECTION06) when ANY input path
// contains '\n' or '\r'. Far slower than the
// batched fast path but only fires for pathological
// inputs.
func rangeBlobOIDsBatchFallback(runner gitRunner,
	repoRoot string, refs []string) map[string]RangeBlobResult {
	out := make(map[string]RangeBlobResult)
	for _, ref := range refs {
		idx := strings.Index(ref, ":")
		if idx < 0 {
			continue
		}
		left := ref[:idx]
		path := ref[idx+1:]
		out[ref] = blobOIDAt(runner, repoRoot, left, path)
	}
	return out
}

// parseCatFileLine interprets one line of `git cat-file
// --batch-check` output. The wire format is documented in
// git-cat-file(1): "<sha> SP <type> SP <size>\n" for
// resolved objects and "<object> SP <status>\n" where
// status is one of missing, ambiguous, excluded. The
// leading "<object>" is the original object expression
// (with a placeholder hash for non-present objects), so
// for paths containing spaces or tabs the line has MORE
// than two whitespace-separated fields.
//
// F20 (CORRECTION05): an empty line or a line with only
// one whitespace-separated field is now parsed as
// BlobUnknown, NOT as a missing-key default. This
// guarantees that any path that reaches the renderer with
// Status == BlobPresent has actually been confirmed by
// cat-file.
//
// F26 (CORRECTION07): the previous implementation
// inspected `fields[1]`, which is the second
// whitespace-separated field. For a path containing
// a space (e.g. "HEAD:docs/my file.txt"), the missing
// response becomes "HEAD:docs/my file.txt missing"
// and `fields[1]` is "docs/my", not "missing". The
// BlobStatus contract was therefore broken: ordinary
// valid Git filenames with whitespace caused the
// renderer to report OTHER instead of MISSING. The fix
// matches the status SUFFIX on the raw line, which is
// invariant under whitespace in the object expression.
//
// Branch coverage:
//
//	"<sha> blob <size>"    -> PRESENT
//	"<obj> missing"        -> MISSING
//	"<obj> ambiguous"      -> AMBIGUOUS
//	"<obj> excluded"       -> OTHER (Git excludes
//	                                   from pack)
//	"<obj> <other>"        -> OTHER
//	"" or whitespace-only  -> UNKNOWN (F20)
func parseCatFileLine(line string) RangeBlobResult {
	// PRESENT path: requires exactly 3 fields
	// ("<sha> blob <size>") so the size field cannot
	// be mistaken for a status.
	if fields := strings.Fields(line); len(fields) == 3 &&
		fields[1] == "blob" && isHexOID(fields[0]) {
		return RangeBlobResult{
			OID:    fields[0],
			Status: BlobPresent,
		}
	}
	// Status-only path: match the trailing status
	// keyword on the raw line so whitespace in the
	// object expression cannot split the field we
	// inspect.
	switch {
	case strings.HasSuffix(line, " missing"):
		return RangeBlobResult{Status: BlobMissing}
	case strings.HasSuffix(line, " ambiguous"):
		return RangeBlobResult{Status: BlobAmbiguous}
	case strings.HasSuffix(line, " excluded"):
		return RangeBlobResult{Status: BlobOther}
	case strings.TrimSpace(line) == "":
		return RangeBlobResult{Status: BlobUnknown}
	default:
		return RangeBlobResult{Status: BlobOther}
	}
}
