// SPDX-License-Identifier: Apache-2.0

package closure

// v2_verifier_tag_metadata_parse_trailers.go implements
// the trailer parser, the metadata binder, and the
// orchestrator-facing metadata observation for the
// Closure Protocol v2 annotated-tag metadata contract.
//
// Splitting this from v2_verifier_tag_metadata_parse.go
// keeps each file under the LLM-friendliness 400-line
// threshold while preserving a single concern per file.
//
// Diagnostic taxonomy emitted here:
//
//   - closure_tag_metadata_missing    required key absent
//   - closure_tag_metadata_duplicate  required key repeated
//   - closure_tag_metadata_unknown    unknown Leamas-* alias
//   - closure_tag_metadata_malformed  value is empty, has
//                                     surrounding whitespace,
//                                     contains CR-only
//                                     endings, or contains
//                                     non-UTF-8 bytes
//   - closure_tag_metadata_mismatch   observed value disagrees
//                                     with externally supplied
//                                     S/F/C/P/M (or with the
//                                     protocol / contract
//                                     version)

import (
	"context"
	"fmt"
	"strconv"
	"strings"
)

// v2ClosureTagMetadataTrailer is the parsed form of a
// single trailer line.
type v2ClosureTagMetadataTrailer struct {
	Key   string
	Value string
}

// ParseV2ClosureTagMetadataTrailers parses the message body
// of a Git annotated tag object into a closed map of
// required Leamas-* trailers. The parser returns a typed
// V2ClosureTagMetadata plus the per-key diagnostics
// accumulated while validating cardinality, syntax and
// charset.
//
// The parser is fail-closed: any rejection emits a typed
// diagnostic and the returned metadata is zero-valued.
//
// Non-Leamas trailers and ordinary prose are ignored; the
// parser neither accepts nor rejects them.
func ParseV2ClosureTagMetadataTrailers(body []byte) (V2ClosureTagMetadata, V2VerifierDiagnostics) {
	metadata := V2ClosureTagMetadata{}
	diags := V2VerifierDiagnostics{}

	body, ok := SplitV2ClosureTagMessageBody(body)
	if !ok {
		diags = append(diags, V2VerifierDiagnostic{
			Code:         V2VerifierClosureTagMetadataMalformed,
			Message:      "tag message body is not valid UTF-8 or has CR-only endings",
			PropertyName: "tag_metadata",
		})
		return metadata, diags
	}

	trailers := []v2ClosureTagMetadataTrailer{}
	seenKey := map[string]int{}
	for _, line := range splitV2ClosureTagLines(body) {
		key, value, ok := parseV2ClosureTagTrailerLine(line)
		if !ok {
			// Non-trailer line (prose or trailing garbage).
			continue
		}
		if !strings.HasPrefix(key, V2TagMetadataPropertyPrefix) {
			// Non-Leamas trailers are non-authoritative; the
			// ACT contract allows them. They are ignored.
			continue
		}
		if _, exists := seenKey[key]; exists {
			seenKey[key]++
			diags = append(diags, V2VerifierDiagnostic{
				Code:         V2VerifierClosureTagMetadataDuplicate,
				Message:      fmt.Sprintf("tag metadata key %q is duplicated", key),
				PropertyName: V2TagMetadataPropertyName(V2ClosureTagMetadataKey(key)),
			})
			continue
		}
		seenKey[key]++
		trailers = append(trailers, v2ClosureTagMetadataTrailer{Key: key, Value: value})
	}

	for _, key := range V2TagMetadataAllKeys {
		canonical := string(key)
		if _, ok := seenKey[canonical]; !ok {
			diags = append(diags, V2VerifierDiagnostic{
				Code:         V2VerifierClosureTagMetadataMissing,
				Message:      fmt.Sprintf("tag metadata key %q is missing", canonical),
				PropertyName: V2TagMetadataPropertyName(key),
			})
		}
	}

	for _, trailer := range trailers {
		key := V2ClosureTagMetadataKey(trailer.Key)
		if !v2ClosureTagKeyIsKnown(key) {
			diags = append(diags, V2VerifierDiagnostic{
				Code:         V2VerifierClosureTagMetadataUnknown,
				Message:      fmt.Sprintf("tag metadata key %q is not recognised", trailer.Key),
				PropertyName: V2TagMetadataPropertyName(key),
			})
			continue
		}
		if err := assignV2ClosureTagMetadata(&metadata, key, trailer.Value); err != nil {
			diags = append(diags, V2VerifierDiagnostic{
				Code:         V2VerifierClosureTagMetadataMalformed,
				Message:      err.Error(),
				PropertyName: V2TagMetadataPropertyName(key),
			})
		}
	}
	return metadata, diags
}

// v2ClosureTagKeyIsKnown reports whether key is one of the
// seven required metadata keys. The function is intentionally
// separate from the per-key switch so the caller can emit a
// single typed diagnostic for unknown aliases.
func v2ClosureTagKeyIsKnown(key V2ClosureTagMetadataKey) bool {
	for _, k := range V2TagMetadataAllKeys {
		if k == key {
			return true
		}
	}
	return false
}

// assignV2ClosureTagMetadata populates the typed metadata
// struct field for the supplied key. The function is the
// single switch through which trailer values reach the
// typed model so all value-format checks live in one place.
func assignV2ClosureTagMetadata(meta *V2ClosureTagMetadata, key V2ClosureTagMetadataKey, value string) error {
	switch key {
	case V2TagMetadataClosureProtocolVersion:
		meta.ClosureProtocolVersion = ClosureProtocolVersion(value)
		return nil
	case V2TagMetadataPlanContractVersion:
		n, err := strconv.Atoi(value)
		if err != nil {
			return fmt.Errorf("plan contract version %q is not an integer", value)
		}
		meta.PlanContractVersion = PlanContractVersion(n)
		return nil
	case V2TagMetadataSubjectCommit,
		V2TagMetadataFreezeCommit,
		V2TagMetadataClosureCommit:
		return assignV2ClosureTagMetadataOID(meta, key, value)
	case V2TagMetadataPlanPath,
		V2TagMetadataManifestPath:
		return assignV2ClosureTagMetadataPath(meta, key, value)
	}
	return fmt.Errorf("unknown metadata key %q", string(key))
}

// assignV2ClosureTagMetadataOID validates and copies an
// OID-shaped metadata value. Abbreviated OIDs (shorter
// than 40 hex chars) are rejected.
func assignV2ClosureTagMetadataOID(meta *V2ClosureTagMetadata, key V2ClosureTagMetadataKey, value string) error {
	if len(value) != 40 {
		return fmt.Errorf("metadata %q OID must be 40-char hex, got %d chars", string(key), len(value))
	}
	for _, r := range value {
		if !((r >= '0' && r <= '9') || (r >= 'a' && r <= 'f')) {
			return fmt.Errorf("metadata %q OID %q is not lowercase hex", string(key), value)
		}
	}
	switch key {
	case V2TagMetadataSubjectCommit:
		meta.SubjectCommit = value
	case V2TagMetadataFreezeCommit:
		meta.FreezeCommit = value
	case V2TagMetadataClosureCommit:
		meta.ClosureCommit = value
	}
	return nil
}

// assignV2ClosureTagMetadataPath validates and copies a
// path-shaped metadata value. Empty values are rejected.
func assignV2ClosureTagMetadataPath(meta *V2ClosureTagMetadata, key V2ClosureTagMetadataKey, value string) error {
	if value == "" {
		return fmt.Errorf("metadata %q value is empty", string(key))
	}
	if strings.ContainsAny(value, "\r\n") {
		return fmt.Errorf("metadata %q value contains newline", string(key))
	}
	switch key {
	case V2TagMetadataPlanPath:
		meta.PlanPath = value
	case V2TagMetadataManifestPath:
		meta.ManifestPath = value
	}
	return nil
}

// BindV2ClosureTagMetadata compares the parsed metadata
// against the externally supplied S/F/C/P/M. The function
// returns one closure_tag_metadata_mismatch diagnostic per
// mismatching field plus the typed V2ClosureTagMetadata.
// The diagnostics slice is empty when every field matches.
func BindV2ClosureTagMetadata(
	metadata V2ClosureTagMetadata,
	subject, freeze, closure, planPath, manifestPath string,
	protocolVersion ClosureProtocolVersion,
	planContractVersion PlanContractVersion,
) V2VerifierDiagnostics {
	diags := V2VerifierDiagnostics{}
	checks := []struct {
		key      V2ClosureTagMetadataKey
		observed string
		expected string
	}{
		{V2TagMetadataClosureProtocolVersion, string(metadata.ClosureProtocolVersion), string(protocolVersion)},
		{V2TagMetadataPlanContractVersion, strconv.Itoa(int(metadata.PlanContractVersion)), strconv.Itoa(int(planContractVersion))},
		{V2TagMetadataSubjectCommit, metadata.SubjectCommit, subject},
		{V2TagMetadataFreezeCommit, metadata.FreezeCommit, freeze},
		{V2TagMetadataClosureCommit, metadata.ClosureCommit, closure},
		{V2TagMetadataPlanPath, metadata.PlanPath, planPath},
		{V2TagMetadataManifestPath, metadata.ManifestPath, manifestPath},
	}
	for _, c := range checks {
		if c.observed != c.expected {
			diags = append(diags, V2VerifierDiagnostic{
				Code:         V2VerifierClosureTagMetadataMismatch,
				Message:      fmt.Sprintf("tag metadata %q=%q, expected %q", c.key, c.observed, c.expected),
				PropertyName: V2TagMetadataPropertyName(c.key),
				Expected:     c.expected,
				Observed:     c.observed,
			})
		}
	}
	return diags
}

// ResolveV2ClosureTagMetadataObservation is the single
// orchestrator-facing entry point for the optional metadata
// phase. The function reads the annotated tag-object bytes
// through the bound git authority, parses the metadata
// block, and binds it to the externally supplied
// S/F/C/P/M plus the protocol/contract versions.
//
// The function is fail-closed: any structural rejection
// sets Read=false; any per-field mismatch sets Bound=false.
//
// Caller contract:
//
//   - the tag object OID is non-empty and known to be an
//     annotated-tag object (already verified by the
//     existing ResolveV2ClosureTagAssertion path)
//   - tagName is the short ref name (preserved for
//     diagnostic PropertyName display)
//   - expected values are non-empty
func ResolveV2ClosureTagMetadataObservation(
	ctx context.Context,
	authority V2ClosureGitAuthority,
	tagOID, tagName string,
	subject, freeze, closure, planPath, manifestPath string,
	protocolVersion ClosureProtocolVersion,
	planContractVersion PlanContractVersion,
) V2ClosureTagMetadataObservation {
	obs := V2ClosureTagMetadataObservation{TagObjectOID: tagOID, TagName: tagName}
	if tagOID == "" {
		obs.Diagnostics = append(obs.Diagnostics, V2VerifierDiagnostic{
			Code:         V2VerifierClosureTagUnreadable,
			Message:      "annotated tag object OID is empty",
			PropertyName: "expected_tag",
		})
		return obs
	}
	bytes, err := readV2ClosureTagObjectBytes(ctx, authority, tagOID)
	if err != nil {
		obs.Diagnostics = append(obs.Diagnostics, V2VerifierDiagnostic{
			Code:         V2VerifierClosureTagUnreadable,
			Message:      fmt.Sprintf("annotated tag object %s unreadable: %v", tagOID, err),
			PropertyName: "tag_object",
		}.withObjectOID(tagOID))
		return obs
	}
	headers, body, headerDiags := ParseV2ClosureTagObjectHeaders(bytes)
	if len(headerDiags) > 0 {
		obs.Diagnostics = append(obs.Diagnostics, headerDiags...)
		return obs
	}
	if headers.Object != closure {
		obs.Diagnostics = append(obs.Diagnostics, V2VerifierDiagnostic{
			Code:         V2VerifierClosureTagMetadataMismatch,
			Message:      fmt.Sprintf("annotated tag object points at %s, expected %s", headers.Object, closure),
			PropertyName: "tag_target",
			Expected:     closure,
			Observed:     headers.Object,
		})
		return obs
	}
	metadata, trailerDiags := ParseV2ClosureTagMetadataTrailers(body)
	if len(trailerDiags) > 0 {
		obs.Diagnostics = append(obs.Diagnostics, trailerDiags...)
		return obs
	}
	mismatchDiags := BindV2ClosureTagMetadata(metadata, subject, freeze, closure, planPath, manifestPath, protocolVersion, planContractVersion)
	if len(mismatchDiags) > 0 {
		obs.Mismatches = v2ClosureTagMetadataMismatchesFromDiags(mismatchDiags)
		obs.Diagnostics = append(obs.Diagnostics, mismatchDiags...)
		return obs
	}
	obs.Metadata = metadata
	obs.Read = true
	obs.Bound = true
	return obs
}

// v2ClosureTagMetadataMismatchesFromDiags reconstructs the
// per-key mismatch list from a slice of typed diagnostics.
// The orchestrator stores the list so future ACT revisions
// can inspect the closed set without re-walking the
// diagnostics slice.
func v2ClosureTagMetadataMismatchesFromDiags(diags V2VerifierDiagnostics) []V2ClosureTagMetadataMismatch {
	out := make([]V2ClosureTagMetadataMismatch, 0, len(diags))
	for _, d := range diags {
		for _, key := range V2TagMetadataAllKeys {
			if V2TagMetadataPropertyName(key) == d.PropertyName {
				out = append(out, V2ClosureTagMetadataMismatch{
					Key:      key,
					Expected: d.Expected,
					Observed: d.Observed,
				})
				break
			}
		}
	}
	return out
}
