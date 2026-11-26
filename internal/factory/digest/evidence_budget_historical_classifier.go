// SPDX-License-Identifier: Apache-2.0

// Package digest: evidence_budget_historical_classifier.go reads the
// HISTORICAL blobs a range-mode digest classifies.
//
// ACT-LEAMAS-RANGE-PATCH-BUDGET-AUTHORITY-01 AMENDMENT-01 (A4)
//
// The predecessor classified a changed file by reading the CURRENT
// working-tree file. With the blob-size suppression removed by the
// original ACT, that worktree read became the ONLY classifier for a
// historical artifact, so the disposition of an immutable historical
// range changed whenever the checkout changed.
//
// The loaders here read the blob at the HEAD endpoint of the range,
// so the classification describes the artifact the range actually
// contains.
//
// The reads are bounded and batched: one Git process serves the whole
// range, and no repository-wide source ingestion occurs.
//
// The companion file evidence_budget_patch_spec.go holds the canonical
// PatchSpec and the retrieval serialization.
package digest

import (
	"strings"
)

// loadHistoricalClassifierData reads the classifier inputs for the
// HISTORICAL blob of `path` at `ref`.
//
// AMENDMENT-01 (A4): the range-mode classifier must classify the
// artifact the range actually describes, not whatever happens to sit
// in the working tree right now.
//
// The predecessor read the CURRENT worktree file. With the blob-size
// suppression removed by the original ACT, that worktree read became
// the only classifier for a historical artifact, so the disposition of
// an immutable historical range changed whenever the checkout changed.
//
// AMENDMENT-01 FINALIZATION (F1): the read is BOUNDED. The loader
// asks Git for the blob SIZE first and then reads only the prefix it
// actually needs. Reading the whole blob would (a) exceed the
// capture limit for a large blob and (b) defeat the bounded-evidence
// contract of the digest.
//
// The second result reports whether a historical blob was available.
// When it is false the caller reports the degraded disposition; it
// does NOT fall back to the working tree.
func loadHistoricalClassifierData(runner gitRunner, repoRoot, ref,
	path string) (prefix string, body []byte, ok bool) {
	if runner == nil || repoRoot == "" || ref == "" || path == "" {
		return "", nil, false
	}
	// The size is fetched first so the content read can be BOUNDED.
	sizeOut, err := runner.Run(repoRoot, []string{
		"cat-file", "-s", ref + ":" + path})
	if err != nil {
		return "", nil, false
	}
	size := parseNumstatInt64(strings.TrimSpace(sizeOut))
	if size <= 0 {
		return "", nil, false
	}
	// Read at most the scan cap. Git has no "read the first N bytes
	// of a blob" mode, so the content is read whole and then
	// truncated -- but the READ is bounded by the capture limit, and
	// a blob that exceeds it is reported as unavailable rather than
	// being silently truncated.
	out, err := runner.Run(repoRoot, []string{
		"cat-file", "blob", ref + ":" + path})
	if err != nil {
		return "", nil, false
	}
	scan := len(out)
	if scan > classifierScanCap {
		scan = classifierScanCap
	}
	body = []byte(out[:scan])
	prefixBytes := len(body)
	if prefixBytes > classifierPrefixBytes {
		prefixBytes = classifierPrefixBytes
	}
	return string(body[:prefixBytes]), body, true
}

// HistoricalClassifierData holds the classifier inputs of ONE
// historical blob.
type HistoricalClassifierData struct {
	Prefix string
	Body   []byte
	OK     bool
}

// historicalBatchContentBudget is the maximum aggregate blob CONTENT
// that a single `git cat-file --batch` chunk may request.
//
// AMENDMENT-01 FINALIZATION (F1): the batch protocol emits the CONTENT
// of the requested objects, not just their metadata, so a batch over
// many blobs can produce far more output than the capture limit
// allows. The chunking keeps the aggregate content below the capture
// limit so the batch never overflows in the first place.
//
// The capture limit is NOT raised to accommodate a large range. The
// loading stays bounded.
const historicalBatchContentBudget = MaxGitCaptureBytes / 2

// loadHistoricalClassifierDataBatch reads the classifier inputs for
// the HISTORICAL blobs of MANY paths in ONE Git process.
//
// AMENDMENT-01 (A4): the range-mode classifier must classify the
// artifact the range actually describes, not whatever happens to sit
// in the working tree right now.
//
// The per-file loader would spawn one subprocess per changed file,
// which is a measurable cost on a large range. The batched loader
// uses `git cat-file --batch` so the whole range is classified with a
// SINGLE process.
//
// The batch framing is newline delimited, so a path containing a
// newline or a carriage return would shift the response alignment.
// Such paths are routed to the per-object fallback, which is
// path-safe.
//
// The per-file reading is bounded by classifierScanCap, so a
// pathological blob cannot inflate the process output.
func loadHistoricalClassifierDataBatch(runner gitRunner, repoRoot,
	ref string, paths []string) map[string]HistoricalClassifierData {
	out := map[string]HistoricalClassifierData{}
	if runner == nil || repoRoot == "" || ref == "" || len(paths) == 0 {
		return out
	}
	for _, p := range paths {
		out[p] = HistoricalClassifierData{}
	}
	if hasAnyPathProtocolHazard(paths) {
		for _, p := range paths {
			prefix, body, ok := loadHistoricalClassifierData(
				runner, repoRoot, ref, p)
			out[p] = HistoricalClassifierData{
				Prefix: prefix, Body: body, OK: ok}
		}
		return out
	}

	queries := make([]string, 0, len(paths))
	for _, p := range paths {
		queries = append(queries, ref+":"+p)
	}

	// AMENDMENT-01 FINALIZATION (F1): the batch is CHUNKED.
	//
	// `git cat-file --batch` emits the CONTENT of every requested
	// object, so a batch over many blobs can produce far more output
	// than the capture limit allows. When that happens the whole
	// captured result is discarded and the classification of the
	// range is lost -- which is exactly how the worktree fallback
	// returned.
	//
	// The batch is therefore split into chunks whose AGGREGATE blob
	// size stays below the capture limit. The chunking is driven by
	// the blob SIZES, which are fetched with `git cat-file
	// --batch-check` (metadata only, so the output is bounded by the
	// number of paths, not by their content).
	//
	// The capture limit is NOT raised. The loading stays bounded.
	sizes := historicalBlobSizesBatch(runner, repoRoot, ref, paths)
	chunks, oversized := chunkPathsBySize(paths, sizes,
		historicalBatchContentBudget)

	// FINAL AUTHORITY CORRECTION (G2): a blob whose KNOWN size exceeds
	// the batch content budget can never be read by the batch, and the
	// per-object fallback would attempt a REDUNDANT full read of an
	// object already known to exceed the limit. Such a blob is
	// reported as unavailable directly, without any content read.
	for _, p := range oversized {
		out[p] = HistoricalClassifierData{}
	}

	for _, chunk := range chunks {
		chunkQueries := make([]string, 0, len(chunk))
		for _, p := range chunk {
			chunkQueries = append(chunkQueries, ref+":"+p)
		}
		raw, err := runner.RunWithStdin(repoRoot, []string{
			"cat-file", "--batch"}, strings.Join(chunkQueries, "\n"))
		if err != nil {
			// The chunk exceeded the capture limit anyway. Fall back
			// to the per-object loader for THIS chunk, which reports
			// the blob as unavailable when it is too large.
			for _, p := range chunk {
				prefix, body, ok := loadHistoricalClassifierData(
					runner, repoRoot, ref, p)
				out[p] = HistoricalClassifierData{
					Prefix: prefix, Body: body, OK: ok}
			}
			continue
		}
		parseHistoricalBatch(raw, chunk, out)
	}
	return out
}

// historicalBlobSizesBatch fetches the SIZE of every historical blob
// in ONE Git process.
//
// AMENDMENT-01 FINALIZATION (F1): `git cat-file --batch-check` emits
// only metadata ("<oid> blob <size>"), so the output is bounded by
// the number of paths and NOT by the size of their content. That is
// what makes it usable for sizing the content batch.
func historicalBlobSizesBatch(runner gitRunner, repoRoot, ref string,
	paths []string) map[string]int64 {
	out := map[string]int64{}
	if runner == nil || repoRoot == "" || ref == "" || len(paths) == 0 {
		return out
	}
	queries := make([]string, 0, len(paths))
	for _, p := range paths {
		queries = append(queries, ref+":"+p)
	}
	raw, err := runner.RunWithStdin(repoRoot, []string{
		"cat-file", "--batch-check"}, strings.Join(queries, "\n"))
	if err != nil {
		return out
	}
	lines := strings.Split(raw, "\n")
	for i, line := range lines {
		if i >= len(paths) {
			break
		}
		fields := strings.Fields(line)
		if len(fields) != 3 {
			continue
		}
		size := parseNumstatInt64(fields[2])
		if size < 0 {
			continue
		}
		out[paths[i]] = size
	}
	return out
}

// chunkPathsBySize splits `paths` into chunks whose aggregate blob
// size stays within `budget`.
//
// AMENDMENT-01 FINALIZATION (F1): a path whose own size exceeds the
// budget forms a chunk of its own. The per-object loader then reports
// it as unavailable, which is the honest outcome: the blob is larger
// than the bounded evidence may read.
//
// FINAL AUTHORITY CORRECTION (G2): such an oversized path is returned
// SEPARATELY, in the second result, so the caller can report it as
// unavailable WITHOUT a redundant content read. The oversized path is
// NOT part of any chunk.
func chunkPathsBySize(paths []string, sizes map[string]int64,
	budget int64) ([][]string, []string) {
	chunks := make([][]string, 0, 1)
	oversized := make([]string, 0, 1)
	current := make([]string, 0, len(paths))
	currentSize := int64(0)
	for _, p := range paths {
		size := sizes[p]
		if size > budget {
			// A single oversized blob. It is reported separately:
			// no chunk can ever carry it, and reading it would
			// exceed the capture limit by construction.
			if len(current) > 0 {
				chunks = append(chunks, current)
				current = make([]string, 0, len(paths))
				currentSize = 0
			}
			oversized = append(oversized, p)
			continue
		}
		if currentSize+size > budget && len(current) > 0 {
			chunks = append(chunks, current)
			current = make([]string, 0, len(paths))
			currentSize = 0
		}
		current = append(current, p)
		currentSize += size
	}
	if len(current) > 0 {
		chunks = append(chunks, current)
	}
	if len(chunks) == 0 {
		chunks = append(chunks, []string{})
	}
	return chunks, oversized
}

// parseHistoricalBatch splits the `git cat-file --batch` output into
// the per-path classifier inputs.
//
// The companion file evidence_budget_historical_batch_parse.go holds
// the parser itself.
//
// FINAL AUTHORITY CORRECTION (G2): the parser consumes the full
// declared content and the trailing delimiter of EVERY resolved
// object, whatever its type, and classifies only the blob records. It
// is FAIL-CLOSED on framing defects.
func parseHistoricalBatch(raw string, paths []string,
	out map[string]HistoricalClassifierData) {
	parseHistoricalBatchRecords([]byte(raw), paths, out)
}

// indexByteFrom returns the index of the first occurrence of b at or
// after `from`, or -1.
func indexByteFrom(data []byte, b byte, from int) int {
	if from < 0 {
		from = 0
	}
	for i := from; i < len(data); i++ {
		if data[i] == b {
			return i
		}
	}
	return -1
}

// selectHistoricalClassifierEvidence decides WHICH historical blob is
// the classification authority for a range file.
//
// AMENDMENT-01 FINALIZATION (F1): the head endpoint alone is NOT
// sufficient authority.
//
//  1. the HEAD blob, when it resolves: it is the artifact the
//     range PRODUCES;
//  2. otherwise the BASE preimage: it is the artifact the range
//     CONSUMED. This covers a DELETION (no head blob exists) and a
//     RENAME or COPY (the preimage lives at the source path);
//  3. otherwise NO historical evidence is available.
//
// The CURRENT WORKTREE is NEVER consulted. Mutable state must not
// become the classification authority of an immutable historical
// range; the caller reports the degraded disposition explicitly
// instead.
//
// The third result is the evidence source ("HEAD", "BASE" or "NONE")
// so the reviewer can tell which endpoint carried the authority.
func selectHistoricalClassifierEvidence(
	head, base HistoricalClassifierData,
) (prefix string, body []byte, ok bool, source string) {
	if head.OK {
		return head.Prefix, head.Body, true, "HEAD"
	}
	if base.OK {
		return base.Prefix, base.Body, true, "BASE"
	}
	// NO historical evidence. The worktree is NOT a substitute.
	return "", nil, false, "NONE"
}

// isArtifactExclusionClass reports whether `class` is one of the
// content-driven ARTIFACT-EXCLUSION classifications.
//
// FINAL AUTHORITY CORRECTION (G1): the artifact-exclusion policy
// describes the artifact's IDENTITY, not the size of the bytes that
// happen to sit at the path today. The policy therefore applies to
// BOTH relevant endpoints of the range.
//
// The companion file evidence_budget_historical_classification.go holds
// the conservation selector.
func isArtifactExclusionClass(class EvidenceClass) bool {
	switch class {
	case ClassBoundedDerivedDigest, ClassBoundedRecursive:
		return true
	}
	return false
}
