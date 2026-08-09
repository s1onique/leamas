// SPDX-License-Identifier: Apache-2.0

// Package evidence - plan_decode.go is the production Plan
// Contract decoder entry point used by the B2-R1 completeness
// predicates.
//
// The previous B2 implementation only checked PlanBytes length
// and SHA-256; arbitrary bytes with a matching SHA-256 could
// satisfy that predicate. The decoder here implements the
// production Plan Contract v1 syntactic authority using the
// same contract as the closure package's parseBoundedClosurePlanDocument
// so the predicate proves the bytes are actually a parseable
// Plan Contract document, not just any payload with a known hash.
//
// The decoder is inlined here (not imported from the closure
// package) to avoid a test-only import cycle: the closure
// package's test files import the evidence package, so the
// evidence package cannot import the closure package. The
// production decoder logic is small enough that maintaining a
// parallel copy is cheaper than breaking the cycle.
//
// The decoder is also used by the candidate builder to derive
// the expected check set from the canonical F:P bytes. The
// caller cannot supply an alternative expected set; the
// document bound to the freeze commit is the only source of
// truth.
package evidence

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
)

// productionPlanBytesLimit is the maximum input size the
// production decoder will accept. The closure package uses
// MaxPlanBytes (1 << 20); the value is duplicated here to
// keep the evidence package independent.
const productionPlanBytesLimit = 1 << 20

// productionDecodeClosurePlan routes the supplied bytes through
// the production Plan Contract v1 syntactic authority. The
// function returns a list of PlanCheckSpec extracted from the
// decoded document on success and a typed error on failure.
// Zero-length input, oversize input, malformed JSON, trailing
// values, and any other syntactic failure are all surfaced as
// a non-nil error.
//
// The function is the only Plan decoding entry point used by
// the B2-R1 completeness matrix. Tests that need to drive the// decoder route through this helper so the production code path
// is the only path exercised.
func productionDecodeClosurePlan(bytes []byte) ([]PlanCheckSpec, error) {
	if len(bytes) == 0 {
		return nil, fmt.Errorf("evidence: plan bytes are empty")
	}
	if len(bytes) > productionPlanBytesLimit {
		return nil, fmt.Errorf("evidence: plan exceeds %d-byte size limit", productionPlanBytesLimit)
	}
	root, diags := productionParseClosurePlanDocument(bytes)
	if len(diags) > 0 {
		return nil, fmt.Errorf("evidence: production plan decode failed: %s", diags[0].Message)
	}
	checks, err := extractPlanCheckSpecs(root)
	if err != nil {
		return nil, fmt.Errorf("evidence: production plan decode failed: %w", err)
	}
	return checks, nil
}

// deriveExpectedChecksFromPlanBytes is the candidate builder
// derivation helper. It decodes the supplied F:P bytes via
// the production Plan Contract decoder and returns the
// expected check set. When the bytes are empty or fail to
// decode the function returns an empty slice; the
// runtimeExpectedChecksDerivedFromPlanBytes predicate will
// reject the candidate in that case.
func deriveExpectedChecksFromPlanBytes(planBytes []byte) []PlanCheckSpec {
	if len(planBytes) == 0 {
		return nil
	}
	out, err := productionDecodeClosurePlan(planBytes)
	if err != nil {
		return nil
	}
	return out
}

// productionPlanDiag is the small error shape the inlined
// decoder returns. The closure package's PlanValidationError
// is too heavyweight to duplicate here; the evidence package
// only needs the message text.
type productionPlanDiag struct {
	Keyword string
	Message string
}

// productionParseClosurePlanDocument mirrors the closure
// package's parseClosurePlanDocument. It enforces the
// size limit, requires exactly one JSON object, rejects
// duplicates, and preserves number textual representation.
// The function is the production syntactic authority for the
// evidence package.
func productionParseClosurePlanDocument(data []byte) (any, []productionPlanDiag) {
	if len(data) == 0 {
		return nil, []productionPlanDiag{{
			Keyword: "type",
			Message: "document is empty; expected JSON object",
		}}
	}
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.UseNumber()
	root, err := productionScanStrictDocument(dec, "")
	if err != nil {
		return nil, []productionPlanDiag{{
			Keyword: "type",
			Message: err.Error(),
		}}
	}
	// After exactly one document, there must be no trailing value.
	if _, trailErr := dec.Token(); trailErr == nil {
		return nil, []productionPlanDiag{{
			Keyword: "type",
			Message: "trailing JSON value after root document",
		}}
	} else if trailErr != io.EOF {
		return nil, []productionPlanDiag{{
			Keyword: "type",
			Message: trailErr.Error(),
		}}
	}
	return root, nil
}

func productionScanStrictDocument(dec *json.Decoder, path string) (any, error) {
	tok, err := dec.Token()
	if err != nil {
		return nil, err
	}
	return productionScanStrictValue(dec, tok, path)
}

func productionScanStrictValue(dec *json.Decoder, tok json.Token, path string) (any, error) {
	delim, ok := tok.(json.Delim)
	if !ok {
		return productionScalarFromToken(tok), nil
	}
	switch delim {
	case '{':
		return productionScanStrictObject(dec, path)
	case '[':
		return productionScanStrictArray(dec, path)
	default:
		return nil, fmt.Errorf("unexpected delimiter %q", delim)
	}
}

func productionScanStrictObject(dec *json.Decoder, path string) (any, error) {
	out := map[string]any{}
	seen := map[string]struct{}{}
	for dec.More() {
		keyTok, err := dec.Token()
		if err != nil {
			return nil, fmt.Errorf("decode object key: %w", err)
		}
		key, ok := keyTok.(string)
		if !ok {
			return nil, fmt.Errorf("object key is not a string")
		}
		if _, dup := seen[key]; dup {
			return nil, fmt.Errorf("duplicate JSON key %q", key)
		}
		seen[key] = struct{}{}
		valTok, err := dec.Token()
		if err != nil {
			return nil, fmt.Errorf("decode value for key %q: %w", key, err)
		}
		child, err := productionScanStrictValue(dec, valTok, productionJoinPath(path, key))
		if err != nil {
			return nil, err
		}
		out[key] = child
	}
	if _, err := dec.Token(); err != nil {
		return nil, fmt.Errorf("decode object close: %w", err)
	}
	return out, nil
}

func productionScanStrictArray(dec *json.Decoder, path string) (any, error) {
	out := []any{}
	idx := 0
	for dec.More() {
		tok, err := dec.Token()
		if err != nil {
			return nil, fmt.Errorf("decode array value: %w", err)
		}
		child, err := productionScanStrictValue(dec, tok, fmt.Sprintf("%s/%d", path, idx))
		if err != nil {
			return nil, err
		}
		out = append(out, child)
		idx++
	}
	if _, err := dec.Token(); err != nil {
		return nil, fmt.Errorf("decode array close: %w", err)
	}
	return out, nil
}

func productionScalarFromToken(tok json.Token) any {
	switch v := tok.(type) {
	case json.Number:
		return v
	case string:
		return v
	case bool:
		return v
	case nil:
		return nil
	default:
		return fmt.Sprintf("%v", v)
	}
}

func productionJoinPath(parent, child string) string {
	if parent == "" {
		return "/" + child
	}
	return parent + "/" + child
}

// extractPlanCheckSpecs projects the production-decoded
// document into the canonical PlanCheckSpec list the
// completeness predicate consumes. The function is pure:
// it never reads from the filesystem and never mutates the
// supplied root. The order matches the document's declared
// check order because the predicate compares
// position-by-position.
func extractPlanCheckSpecs(root any) ([]PlanCheckSpec, error) {
	obj, ok := root.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("root is not a JSON object")
	}
	checksAny, ok := obj["checks"]
	if !ok {
		return nil, fmt.Errorf("root.checks is missing")
	}
	checks, ok := checksAny.([]any)
	if !ok {
		return nil, fmt.Errorf("root.checks is not an array")
	}
	out := make([]PlanCheckSpec, 0, len(checks))
	for i, item := range checks {
		entry, ok := item.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("checks[%d] is not a JSON object", i)
		}
		id, ok := entry["id"].(string)
		if !ok {
			return nil, fmt.Errorf("checks[%d].id is not a string", i)
		}
		mode, ok := entry["mode"].(string)
		if !ok {
			return nil, fmt.Errorf("checks[%d].mode is not a string", i)
		}
		out = append(out, PlanCheckSpec{ID: id, Mode: mode})
	}
	return out, nil
}
