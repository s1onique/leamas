// SPDX-License-Identifier: Apache-2.0

package closure

// v2_caller_state_refs.go extends the caller-state snapshot to
// capture the deterministic ref bytes (refs bytes/hash) required
// by ACT-LEAMAS-FACTORY-CLOSURE-RUNTIME-CONTEXT-AND-EXECUTE01-CORRECTION02.
//
// The implementation invokes `git for-each-ref` with an explicit
// NUL-framed machine format:
//
//	git for-each-ref --format='%(objectname)%00%(refname)%00' --sort=refname
//
// Each record is `<oid>\x00<refname>\x00\n`. The trailing NUL
// marks the end of the record and the trailing LF marks the
// end of the line. The parser splits on the `\x00\n` pair so
// two records never share a delimiter, and it enforces a
// terminal `\x00\n` byte so a half-record cannot slip through.
//
// Splitting this from v2_lifecycle_invariants.go keeps both
// files under the LLM-friendly 400-line threshold while
// preserving the single closure over the descriptor that
// ACT-LEAMAS-FACTORY-CLOSURE-PROTOCOL-V1-01 requires.

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
)

// callerRefsFormat is the explicit NUL-framed machine format
// the snapshot invokes. The pair order is deliberate:
// objectname first, refname second, NUL terminator, then LF,
// so the parser can split on the NUL+LF pair.
const callerRefsFormat = "%(objectname)%00%(refname)%00"

// snapshotCallerRefsArgs returns the args the snapshot
// passes to `git for-each-ref`. Exposed for tests so the
// production wire format is exercised end-to-end.
func snapshotCallerRefsArgs() []string {
	return []string{
		"for-each-ref",
		"--format=" + callerRefsFormat,
		"--sort=refname",
	}
}

// snapshotCallerRefs captures `git for-each-ref` output and
// produces a deterministic sorted ref listing plus its SHA-256
// digest. The snapshot is fail-closed: any observation failure
// (Git error, malformed framing, duplicate ref, invalid OID,
// empty record, etc.) is captured as a typed V2Diagnostic and
// Available=false. The runner must reject when Available=false.
func snapshotCallerRefs(ctx context.Context, git gitClient, repoRoot string) (refsBytes string, refsHash string, diags V2Diagnostics, available bool) {
	if git == nil || repoRoot == "" {
		return "", "", V2Diagnostics{{
			Code:         V2CodeCallerStateUnavailable,
			Message:      "caller refs observation failed: no Git client supplied",
			PropertyName: "caller_refs",
		}}, false
	}
	result := git.Run(ctx, repoRoot, snapshotCallerRefsArgs()...)
	if result.Err != nil || result.ExitCode != 0 {
		return "", "", V2Diagnostics{{
			Code:         V2CodeCallerStateUnavailable,
			Message:      fmt.Sprintf("caller refs observation failed: %s", strings.TrimSpace(string(result.Stderr))),
			PropertyName: "caller_refs",
		}}, false
	}
	records, parseDiags := parseNulFramedRefs(result.Stdout)
	if len(parseDiags) > 0 {
		return "", "", parseDiags, false
	}
	// Records are already sorted by Git (--sort=refname) and
	// parseNulFramedRefs preserves order. We re-sort by refname
	// defensively so a future caller-supplied format flag cannot
	// silently change canonical order.
	sort.Slice(records, func(i, j int) bool { return records[i].refname < records[j].refname })
	var b strings.Builder
	for _, r := range records {
		b.WriteString(r.objectname)
		b.WriteByte(0x00)
		b.WriteString(r.refname)
		b.WriteByte(0x00)
		b.WriteByte('\n')
	}
	bytes := b.String()
	sum := sha256.Sum256([]byte(bytes))
	return bytes, hex.EncodeToString(sum[:]), nil, true
}

// refRecord is one parsed NUL-framed ref entry.
type refRecord struct {
	objectname string
	refname    string
}

// parseNulFramedRefs parses NUL-framed `git for-each-ref`
// output into ordered records. The function is fail-closed:
// any framing violation produces a typed V2Diagnostic and an
// empty record list. The parser enforces:
//
//   - terminal boundary <NUL><LF> on every non-empty buffer
//   - records are NUL-framed and NUL+LF terminated
//   - no duplicate refnames
//   - no empty refnames or empty objectnames
//   - objectnames are 40-char hex
func parseNulFramedRefs(raw []byte) ([]refRecord, V2Diagnostics) {
	// Empty input is a valid zero-ref snapshot.
	if len(raw) == 0 {
		return nil, nil
	}
	// Terminal boundary enforcement. The parser chose the
	// NUL-newline pair as the record separator, so valid Git
	// output always ends with `\x00\n`. A truncated frame
	// such as `<oid>\x00<refname>` is malformed and MUST be
	// rejected so a silent parser cannot accept a half-record.
	if !bytes.HasSuffix(raw, []byte{0x00, 0x0a}) {
		return nil, V2Diagnostics{{
			Code:         V2CodeCallerStateUnavailable,
			Message:      "caller refs observation failed: malformed NUL framing (missing terminal boundary)",
			PropertyName: "caller_refs",
		}}
	}
	var records []refRecord
	seen := make(map[string]bool, 0)
	// Records are split on the NUL+LF pair. Each segment is
	// `<oid>\x00<refname>`. The trailing empty segment is
	// stripped by strings.Split.
	chunks := strings.Split(string(raw), "\x00\n")
	for _, chunk := range chunks {
		if chunk == "" {
			continue
		}
		parts := strings.Split(chunk, "\x00")
		if len(parts) != 2 {
			return nil, V2Diagnostics{{
				Code:         V2CodeCallerStateUnavailable,
				Message:      "caller refs observation failed: malformed NUL framing (record boundary)",
				PropertyName: "caller_refs",
			}}
		}
		object := parts[0]
		name := parts[1]
		if object == "" || name == "" {
			return nil, V2Diagnostics{{
				Code:         V2CodeCallerStateUnavailable,
				Message:      "caller refs observation failed: empty refname or objectname",
				PropertyName: "caller_refs",
			}}
		}
		// OID validation is enforced inside the parser so fake
		// codes that survive ref-framing cannot slip through.
		if !isHexOID(object) {
			return nil, V2Diagnostics{{
				Code:         V2CodeCallerStateUnavailable,
				Message:      fmt.Sprintf("caller refs observation failed: invalid OID %q for ref %q", object, name),
				PropertyName: "caller_refs",
			}}
		}
		if seen[name] {
			return nil, V2Diagnostics{{
				Code:         V2CodeCallerStateUnavailable,
				Message:      fmt.Sprintf("caller refs observation failed: duplicate ref %q", name),
				PropertyName: "caller_refs",
			}}
		}
		seen[name] = true
		records = append(records, refRecord{objectname: object, refname: name})
	}
	return records, nil
}

// isHexOID reports whether s is a 40-character lowercase or
// uppercase hex string. We do NOT enforce lowercase here so
// production Git's mixed-case output passes.
func isHexOID(s string) bool {
	if len(s) != 40 {
		return false
	}
	for _, ch := range s {
		switch {
		case ch >= '0' && ch <= '9':
		case ch >= 'a' && ch <= 'f':
		case ch >= 'A' && ch <= 'F':
		default:
			return false
		}
	}
	return true
}
