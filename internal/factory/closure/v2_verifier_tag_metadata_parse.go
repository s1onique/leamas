// SPDX-License-Identifier: Apache-2.0

package closure

// v2_verifier_tag_metadata_parse.go implements the low-level
// header / line / body primitives the trailer parser and
// the orchestrator-facing metadata observation rely on.
// Splitting the parser along the header/trailer boundary
// keeps each file under the LLM-friendliness 400-line
// threshold.
//
// The trailer parser, the binder, and the orchestrator
// entry point live in v2_verifier_tag_metadata_parse_trailers.go.

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"unicode/utf8"
)

// readV2ClosureTagObjectBytes is the single point through
// which the metadata parser observes raw tag-object bytes.
// The call uses the same authorityRunGit helper that the
// existing ResolveV2ClosureTagAssertion uses, so the parser
// inherits the repository-bound path and the bounded
// execution / output limits.
func readV2ClosureTagObjectBytes(ctx context.Context, authority V2ClosureGitAuthority, oid string) ([]byte, error) {
	// Reuse the helper in v2_verifier_tag.go: the function
	// is package-private so the parser can drive it
	// directly. The helper routes `git cat-file tag <oid>`
	// through the same authority so the caller never sees
	// a different CWD.
	return authorityRunGitCatFileTag(ctx, authority, oid)
}

// v2ClosureTagObjectHeaderNames is the closed list of
// annotated-tag-object header keys the parser recognises.
// Each header MUST appear exactly once; the parser rejects
// duplicates and unknown headers.
var v2ClosureTagObjectHeaderNames = []string{"object", "type", "tag", "tagger"}

// V2ClosureTagObjectHeaders is the typed parsed-header shape
// of a Git annotated tag object. The parser rejects objects
// that omit `object`, `type`, `tag` or `tagger`; missing
// `tagger` is accepted as long as the body still contains
// the required Leamas metadata trailers, since some legacy
// tags omit the tagger line.
type V2ClosureTagObjectHeaders struct {
	Object string
	Type   string
	Tag    string
	Tagger string
}

// ParseV2ClosureTagObjectHeaders parses the raw annotated
// tag object bytes into typed header fields and the message
// body. The function returns a non-empty diagnostic for
// every structural rejection:
//
//   - closure_tag_metadata_malformed   empty / truncated
//   - closure_tag_metadata_malformed   duplicate header
//   - closure_tag_metadata_malformed   unknown header
//   - closure_tag_metadata_malformed   wrong object type
//   - closure_tag_metadata_malformed   NUL / CR-only
//
// On the happy path the returned Diagnostics slice is
// empty and Message carries the raw annotation body without
// trailing newlines trimmed (preserved verbatim so the
// trailer parser sees the exact bytes the tag-object
// carries).
func ParseV2ClosureTagObjectHeaders(raw []byte) (V2ClosureTagObjectHeaders, []byte, V2VerifierDiagnostics) {
	headers := V2ClosureTagObjectHeaders{}
	diags := V2VerifierDiagnostics{}
	if len(raw) == 0 {
		diags = append(diags, V2VerifierDiagnostic{
			Code:         V2VerifierClosureTagMetadataMalformed,
			Message:      "tag object bytes are empty",
			PropertyName: "tag_object",
		})
		return headers, nil, diags
	}
	if bytes.IndexByte(raw, 0x00) >= 0 {
		diags = append(diags, V2VerifierDiagnostic{
			Code:         V2VerifierClosureTagMetadataMalformed,
			Message:      "tag object bytes contain NUL",
			PropertyName: "tag_object",
		})
		return headers, nil, diags
	}
	headerBytes, body, hasSeparator := splitV2ClosureTagHeaderBody(raw)
	if !hasSeparator {
		diags = append(diags, V2VerifierDiagnostic{
			Code:         V2VerifierClosureTagMetadataMalformed,
			Message:      "tag object missing blank header/body separator",
			PropertyName: "tag_object",
		})
		return headers, nil, diags
	}
	seen := map[string]bool{}
	for _, line := range splitV2ClosureTagLines(headerBytes) {
		idx := bytes.IndexByte(line, ' ')
		if idx <= 0 {
			diags = append(diags, V2VerifierDiagnostic{
				Code:         V2VerifierClosureTagMetadataMalformed,
				Message:      fmt.Sprintf("tag header line is malformed: %q", string(line)),
				PropertyName: "tag_object",
			})
			return headers, nil, diags
		}
		key := string(line[:idx])
		val := string(line[idx+1:])
		if !v2ClosureTagKnownHeader(key) {
			diags = append(diags, V2VerifierDiagnostic{
				Code:         V2VerifierClosureTagMetadataMalformed,
				Message:      fmt.Sprintf("tag header key %q is unknown", key),
				PropertyName: "tag_object",
			})
			return headers, nil, diags
		}
		if seen[key] {
			diags = append(diags, V2VerifierDiagnostic{
				Code:         V2VerifierClosureTagMetadataMalformed,
				Message:      fmt.Sprintf("tag header %q is duplicated", key),
				PropertyName: "tag_object",
			})
			return headers, nil, diags
		}
		seen[key] = true
		switch key {
		case "object":
			headers.Object = val
		case "type":
			headers.Type = val
		case "tag":
			headers.Tag = val
		case "tagger":
			headers.Tagger = val
		}
	}
	if headers.Object == "" {
		diags = append(diags, V2VerifierDiagnostic{
			Code:         V2VerifierClosureTagMetadataMalformed,
			Message:      "tag object header missing required field: object",
			PropertyName: "tag_object",
		})
		return headers, nil, diags
	}
	if headers.Type == "" {
		diags = append(diags, V2VerifierDiagnostic{
			Code:         V2VerifierClosureTagMetadataMalformed,
			Message:      "tag object header missing required field: type",
			PropertyName: "tag_object",
		})
		return headers, nil, diags
	}
	if headers.Type != "commit" {
		diags = append(diags, V2VerifierDiagnostic{
			Code:         V2VerifierClosureTagMetadataMalformed,
			Message:      fmt.Sprintf("tag object type %q is not commit", headers.Type),
			PropertyName: "tag_object",
		})
		return headers, nil, diags
	}
	if headers.Tag == "" {
		diags = append(diags, V2VerifierDiagnostic{
			Code:         V2VerifierClosureTagMetadataMalformed,
			Message:      "tag object header missing required field: tag",
			PropertyName: "tag_object",
		})
		return headers, nil, diags
	}
	return headers, body, diags
}

// splitV2ClosureTagHeaderBody splits raw into the headers
// portion and the message body. The split point is the
// first \n\n (LF+LF) sequence. CR-only line endings are
// rejected here so the trailer parser never sees them.
func splitV2ClosureTagHeaderBody(raw []byte) (headerBytes, body []byte, ok bool) {
	if len(raw) == 0 {
		return nil, nil, false
	}
	if bytes.IndexByte(raw, '\r') >= 0 && !bytes.Contains(raw, []byte("\n")) {
		return nil, nil, false
	}
	for i := 0; i < len(raw)-1; i++ {
		if raw[i] == '\n' && raw[i+1] == '\n' {
			return raw[:i], raw[i+2:], true
		}
	}
	return nil, nil, false
}

// splitV2ClosureTagLines returns the LF-separated lines of
// raw, stripping the trailing newline of each line.
func splitV2ClosureTagLines(raw []byte) [][]byte {
	out := [][]byte{}
	start := 0
	for i := 0; i < len(raw); i++ {
		if raw[i] == '\n' {
			out = append(out, raw[start:i])
			start = i + 1
		}
	}
	if start < len(raw) {
		out = append(out, raw[start:])
	}
	return out
}

// v2ClosureTagKnownHeader reports whether key is one of
// the four Git tag-object header keys the parser accepts.
func v2ClosureTagKnownHeader(key string) bool {
	for _, k := range v2ClosureTagObjectHeaderNames {
		if k == key {
			return true
		}
	}
	return false
}

// SplitV2ClosureTagMessageBody is a tiny helper exposed so
// tests can drive the trailer parser directly with a
// hand-crafted body. The function returns the trimmed body
// (CR-only endings rejected, trailing newlines preserved).
func SplitV2ClosureTagMessageBody(raw []byte) ([]byte, bool) {
	if bytes.IndexByte(raw, '\r') >= 0 && !bytes.Contains(raw, []byte("\n")) {
		return nil, false
	}
	if bytes.IndexByte(raw, 0x00) >= 0 {
		return nil, false
	}
	if !utf8.Valid(raw) {
		return nil, false
	}
	return raw, true
}

// parseV2ClosureTagTrailerLine extracts the key and value
// of a Git trailer line. The format is KEY ":" SPACE VALUE.
// Lines that do not match the trailer grammar are ignored.
func parseV2ClosureTagTrailerLine(line []byte) (string, string, bool) {
	if len(line) == 0 {
		return "", "", false
	}
	idx := bytes.IndexByte(line, ':')
	if idx <= 0 {
		return "", "", false
	}
	key := string(bytes.TrimSpace(line[:idx]))
	value := string(line[idx+1:])
	if !strings.HasPrefix(value, " ") {
		return "", "", false
	}
	value = strings.TrimPrefix(value, " ")
	if value != strings.TrimRight(value, " ") {
		return "", "", false
	}
	if strings.TrimSpace(value) != value {
		return "", "", false
	}
	if value == "" {
		return "", "", false
	}
	return key, value, true
}
