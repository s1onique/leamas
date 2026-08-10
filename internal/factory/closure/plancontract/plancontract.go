// SPDX-License-Identifier: Apache-2.0

// Package plancontract is the single production source of truth
// for Plan Contract v1 decoding.
//
// B2-R2 motivation: prior to B2-R2, the closure package owned
// the bounded parser and the evidence package contained a
// parallel, hand-maintained copy. Two decoders is two
// authorities for the same wire contract, which the doctrine
// rejects. The plancontract package is the leaf both consumers
// import:
//
//	closure runner  -> plancontract
//	evidence        -> plancontract
//
// plancontract imports nothing from the closure package or
// the evidence package. It owns the bounded parser, the size
// cap (MaxPlanBytes), and the syntactic invariants of the
// Plan Contract v1 document. Any change to the parsing rules
// is made here and exercised by both consumers.
//
// The package exposes three surfaces:
//
//  1. DecodeBytes returns the parsed JSON object (any) so the
//     closure runner can re-decode the canonical shape into
//     its own fully-attributed Plan struct.
//  2. DecodeBytesToResult returns the minimal DecodeResult
//     (contract_version + ordered checks) for the evidence
//     authority.
//  3. DecodeAndValidate composes the minimal surface with the
//     semantic invariants (mode must be run|exclude, IDs
//     non-empty, contract_version required).
package plancontract

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
)

// MaxPlanBytes is the canonical size cap for the Plan Contract
// v1 wire format. The closure package's closure.MaxPlanBytes
// is the same value; the constant is duplicated here to keep
// the leaf package independent.
const MaxPlanBytes = 1 << 20

// PlanCheck is the minimal projection of a Plan Contract v1
// check entry. The closure package's closure.PlanCheck has
// many more fields; this minimal shape is what both
// consumers require for the public authority.
type PlanCheck struct {
	ID   string
	Mode string
}

// DecodeResult is the canonical decoded Plan Contract v1
// document. ContractVersion is preserved verbatim from the
// wire bytes; Checks preserves the document's declared order.
type DecodeResult struct {
	ContractVersion int
	Checks          []PlanCheck
}

// DecodeError is the typed decoder error. The Message is the
// human-readable description; the Code is the canonical
// failure token callers can switch on.
type DecodeError struct {
	Code    string
	Message string
}

func (e *DecodeError) Error() string {
	return fmt.Sprintf("plancontract: %s: %s", e.Code, e.Message)
}

// DecodeBytes is the syntactic entry point. It returns the
// parsed JSON object so the closure runner can re-decode it
// into its own Plan struct via the typed-decoder path.
// Numeric tokens preserve their textual form via
// Decoder.UseNumber so contract_version and 40-char SHA-1
// OIDs round-trip without precision loss.
//
// The function enforces the MaxPlanBytes cap, requires
// exactly one JSON object, rejects duplicate keys, and
// rejects trailing content. The closure runner and the
// evidence authority both consume the same parser output.
func DecodeBytes(data []byte) (any, error) {
	if len(data) == 0 {
		return nil, &DecodeError{
			Code:    "invalid_json",
			Message: "document is empty; expected JSON object",
		}
	}
	if len(data) > MaxPlanBytes {
		return nil, &DecodeError{
			Code:    "invalid_json",
			Message: fmt.Sprintf("plan exceeds %d-byte size limit", MaxPlanBytes),
		}
	}
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.UseNumber()
	root, err := scanStrictDocument(dec, "")
	if err != nil {
		return nil, &DecodeError{
			Code:    "invalid_json",
			Message: err.Error(),
		}
	}
	// After exactly one document, the next Decode MUST return
	// io.EOF. Any other value (a second object, a trailing
	// scalar, malformed trailing garbage) is a strict failure.
	if err := requireEOF(dec); err != nil {
		return nil, err
	}
	return root, nil
}

// DecodeBytesToResult is the minimal entry point. It runs
// DecodeBytes and projects the parsed JSON object into the
// canonical DecodeResult. The semantic pass is NOT applied;
// callers that want it invoke Validate on the returned
// DecodeResult.
func DecodeBytesToResult(data []byte) (DecodeResult, error) {
	root, err := DecodeBytes(data)
	if err != nil {
		return DecodeResult{}, err
	}
	return projectToResult(root)
}

// DecodeAndValidate is the production entry point. It
// composes DecodeBytesToResult with Validate. The composite
// is the single authority the evidence package and the
// closure runner both consume.
func DecodeAndValidate(data []byte) (DecodeResult, error) {
	result, err := DecodeBytesToResult(data)
	if err != nil {
		return DecodeResult{}, err
	}
	if err := Validate(result); err != nil {
		return DecodeResult{}, err
	}
	return result, nil
}

// Validate enforces the semantic invariants of the decoded
// Plan Contract v1 document. The minimum required shape is
// the contract_version field and a checks array. The
// function returns a typed DecodeError so callers can
// distinguish structural failures from deeper semantic
// failures.
func Validate(result DecodeResult) error {
	if result.ContractVersion == 0 {
		return &DecodeError{
			Code:    "missing_field",
			Message: "contract_version is required and must be set",
		}
	}
	for i, c := range result.Checks {
		if c.ID == "" {
			return &DecodeError{
				Code:    "missing_field",
				Message: fmt.Sprintf("checks[%d].id is required", i),
			}
		}
		if c.Mode != "run" && c.Mode != "exclude" {
			return &DecodeError{
				Code:    "invalid_mode",
				Message: fmt.Sprintf("checks[%d].mode is %q (expected run|exclude)", i, c.Mode),
			}
		}
	}
	return nil
}

// requireEOF verifies that after the first Decode the
// decoder has reached the end of the input. The canonical
// Go idiom is a second Decode call that MUST return
// io.EOF; any other return value (a second object, a
// trailing scalar, malformed trailing garbage) is a
// strict failure.
func requireEOF(dec *json.Decoder) error {
	var extra any
	if err := dec.Decode(&extra); err == io.EOF {
		return nil
	} else if err != nil {
		return &DecodeError{
			Code:    "invalid_json",
			Message: err.Error(),
		}
	}
	return &DecodeError{
		Code:    "trailing_value",
		Message: "trailing JSON value after root document",
	}
}

// projectToResult converts the syntactic JSON object emitted
// by the strict parser into the canonical DecodeResult. The
// function is pure; it does not call Validate so callers
// that want the syntactic result alone can use it.
func projectToResult(root any) (DecodeResult, error) {
	obj, ok := root.(map[string]any)
	if !ok {
		return DecodeResult{}, &DecodeError{
			Code:    "invalid_json",
			Message: "root is not a JSON object",
		}
	}
	result := DecodeResult{}
	if v, ok := obj["contract_version"]; ok {
		if n, ok := v.(json.Number); ok {
			if iv, err := n.Int64(); err == nil {
				result.ContractVersion = int(iv)
			}
		}
	}
	checksAny, ok := obj["checks"]
	if !ok {
		return DecodeResult{}, &DecodeError{
			Code:    "missing_field",
			Message: "checks is required",
		}
	}
	checks, ok := checksAny.([]any)
	if !ok {
		return DecodeResult{}, &DecodeError{
			Code:    "invalid_type",
			Message: "checks is not an array",
		}
	}
	for i, item := range checks {
		entry, ok := item.(map[string]any)
		if !ok {
			return DecodeResult{}, &DecodeError{
				Code:    "invalid_type",
				Message: fmt.Sprintf("checks[%d] is not a JSON object", i),
			}
		}
		id, _ := entry["id"].(string)
		mode, _ := entry["mode"].(string)
		result.Checks = append(result.Checks, PlanCheck{ID: id, Mode: mode})
	}
	return result, nil
}

// ----------------------------------------------------------------------------
// Strict single-document scanner
// ----------------------------------------------------------------------------
//
// scanStrictDocument is the strict single-document object/array
// scanner. It mirrors the closure package's parser at the
// syntactic level (no semantic validation, no schema) so that
// the public Decode entry point does not require the closure
// package to be installed. The scanner rejects malformed
// JSON, duplicate keys, and unexpected delimiters.
//
// scanStrictObject and scanStrictArray are unexported
// helpers. The scanner is intentionally tiny: production
// semantics live in the closure package's own validator,
// and this leaf only proves the document is a single,
// well-formed JSON object.

func scanStrictDocument(dec *json.Decoder, path string) (any, error) {
	tok, err := dec.Token()
	if err != nil {
		return nil, err
	}
	return scanStrictValue(dec, tok, path)
}

func scanStrictValue(dec *json.Decoder, tok json.Token, path string) (any, error) {
	delim, ok := tok.(json.Delim)
	if !ok {
		return scalarFromToken(tok), nil
	}
	switch delim {
	case '{':
		return scanStrictObject(dec, path)
	case '[':
		return scanStrictArray(dec, path)
	default:
		return nil, fmt.Errorf("unexpected delimiter %q", delim)
	}
}

func scanStrictObject(dec *json.Decoder, path string) (any, error) {
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
		valueTok, err := dec.Token()
		if err != nil {
			return nil, fmt.Errorf("decode value for key %q: %w", key, err)
		}
		child, err := scanStrictValue(dec, valueTok, joinPath(path, key))
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

func scanStrictArray(dec *json.Decoder, path string) (any, error) {
	out := []any{}
	idx := 0
	for dec.More() {
		tok, err := dec.Token()
		if err != nil {
			return nil, fmt.Errorf("decode array value: %w", err)
		}
		child, err := scanStrictValue(dec, tok, fmt.Sprintf("%s/%d", path, idx))
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

func scalarFromToken(tok json.Token) any {
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

func joinPath(parent, child string) string {
	if parent == "" {
		return "/" + child
	}
	return parent + "/" + child
}
