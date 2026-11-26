// SPDX-License-Identifier: Apache-2.0

// Package digest: evidence_budget_historical_batch_parse.go holds the
// `git cat-file --batch` RESPONSE PARSER of the historical classifier.
//
// ACT-LEAMAS-RANGE-PATCH-BUDGET-AUTHORITY-01 FINAL AUTHORITY CORRECTION
// (G2)
//
// INDEPENDENTLY REPRODUCED DEFECT: the `git cat-file --batch` protocol
// emits, for every RESOLVED object, a header line followed by the
// object's FULL CONTENT and a trailing delimiter:
//
//	<oid> <type> <size>\n
//	<content bytes>\n
//
// The predecessor parser consumed the CONTENT only for "blob" records.
// When a batch mixed a tree (or any other resolved object type) with a
// blob, the tree's declared content bytes were NOT consumed, so the
// next record was read from the middle of the tree's binary content and
// EVERY subsequent record was mis-attributed.
//
// The companion file evidence_budget_historical_classifier.go holds the
// loader and the classification authority.
package digest

import (
	"strconv"
	"strings"
)

// parseHistoricalBatchRecords splits the `git cat-file --batch` output
// into the per-path classifier inputs.
//
// The wire format for a RESOLVED object is:
//
//	<oid> <type> <size>\n
//	<content bytes>\n
//
// and for an UNRESOLVED one:
//
//	<obj> missing\n
//
// The parser consumes the full declared content and the trailing
// delimiter of EVERY resolved object, whatever its type, and classifies
// only the blob records.
//
// The parser is FAIL-CLOSED on framing defects:
//
//   - a record whose declared content is not fully present is a
//     TRUNCATED record: it is NOT marked OK and no partial body is
//     retained;
//   - a header that is not a well-formed "<oid> <type> <size>" triple
//     is a MALFORMED record: the alignment can no longer be trusted,
//     so parsing stops.
//
// An unresolved record ("missing") carries no content and is simply
// skipped; the corresponding path keeps its zero value.
func parseHistoricalBatchRecords(data []byte, paths []string,
	out map[string]HistoricalClassifierData) {
	pos := 0
	for _, p := range paths {
		if pos >= len(data) {
			return
		}
		nl := indexByteFrom(data, '\n', pos)
		if nl < 0 {
			// The header itself is truncated. The alignment can no
			// longer be trusted, so no further record is read.
			return
		}
		header := string(data[pos:nl])
		pos = nl + 1
		fields := strings.Fields(header)
		if len(fields) == 2 {
			// An unresolved record ("<obj> missing"): it carries no
			// content, so the alignment is preserved by skipping it.
			continue
		}
		if len(fields) != 3 {
			// A malformed header. The alignment can no longer be
			// trusted, so parsing stops.
			return
		}
		size, err := strconv.Atoi(fields[2])
		if err != nil || size < 0 {
			// A malformed size. The alignment can no longer be
			// trusted, so parsing stops.
			return
		}
		end := pos + size
		if end > len(data) {
			// The declared content is NOT fully present. The record
			// is truncated: it is NOT marked OK and no partial body
			// is retained. The alignment cannot be recovered, so
			// parsing stops.
			return
		}
		content := data[pos:end]
		pos = end
		// Skip the trailing newline Git appends after the content.
		if pos < len(data) && data[pos] == '\n' {
			pos++
		}
		if fields[1] != "blob" {
			// A resolved object of another type (a tree, a commit, a
			// tag). Its content was consumed, so the alignment is
			// preserved, but it is NOT classified as a blob.
			continue
		}
		scan := len(content)
		if scan > classifierScanCap {
			scan = classifierScanCap
		}
		// The retained body is a COPY: a slice of the batch backing
		// array would keep the whole batch alive for as long as the
		// body is retained, and any later write into the batch array
		// would be observable through the retained body.
		body := make([]byte, scan)
		copy(body, content[:scan])
		prefixBytes := len(body)
		if prefixBytes > classifierPrefixBytes {
			prefixBytes = classifierPrefixBytes
		}
		out[p] = HistoricalClassifierData{
			Prefix: string(body[:prefixBytes]),
			Body:   body,
			OK:     true,
		}
	}
}
