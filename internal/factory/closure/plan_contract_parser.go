package closure

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
)

// parseClosurePlanDocument is the single syntactic parsing authority
// for the closure-plan v1 wire contract. It proves:
//
//   - exactly one JSON value is present;
//   - the root value is fully consumed (no trailing non-whitespace);
//   - no object property is duplicated at any depth;
//   - numbers preserve their original textual representation (no
//     float round-trip) so contract_version, sha OIDs, and byte
//     counts round-trip through the validator unchanged;
//   - maximum input size is preserved (caller enforces MaxPlanBytes
//     before calling).
//
// The function is the shared syntactic authority between the
// structural validator and the typed decoder: both call it, so the
// structural and typed paths cannot disagree on whether a document
// is well-formed.
func parseClosurePlanDocument(data []byte) (any, []PlanValidationError) {
	planParserCalls++
	var diagnostics []PlanValidationError
	if len(data) == 0 {
		diagnostics = append(diagnostics, PlanValidationError{
			InstancePath: "",
			SchemaPath:   "",
			Code:         PlanCodeInvalidJSON,
			Keyword:      KeywordType,
			Message:      "document is empty; expected JSON object",
		})
		return nil, diagnostics
	}
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.UseNumber()
	root, err := scanStrictDocument(dec, "")
	if err != nil {
		diagnostics = append(diagnostics, parseDiagnosticFromErr("", err))
		return nil, diagnostics
	}
	// After exactly one document, there must be no trailing value.
	if _, trailErr := dec.Token(); trailErr == nil {
		diagnostics = append(diagnostics, PlanValidationError{
			InstancePath: "",
			SchemaPath:   "",
			Code:         PlanCodeInvalidJSON,
			Keyword:      KeywordType,
			Message:      "trailing JSON value after root document",
		})
		return nil, diagnostics
	} else if !errors.Is(trailErr, io.EOF) {
		diagnostics = append(diagnostics, parseDiagnosticFromErr("", trailErr))
		return nil, diagnostics
	}
	return root, diagnostics
}

func scanStrictDocument(dec *json.Decoder, instancePath string) (any, error) {
	token, err := dec.Token()
	if err != nil {
		return nil, err
	}
	return scanStrictValue(dec, token, instancePath)
}

func scanStrictValue(dec *json.Decoder, token json.Token, instancePath string) (any, error) {
	delim, composite := token.(json.Delim)
	if !composite {
		return scalarFromToken(token), nil
	}
	switch delim {
	case '{':
		return scanStrictObject(dec, instancePath)
	case '[':
		return scanStrictArray(dec, instancePath)
	default:
		return nil, fmt.Errorf("unexpected delimiter %q", delim)
	}
}

func scanStrictObject(dec *json.Decoder, instancePath string) (any, error) {
	out := map[string]any{}
	seen := make(map[string]struct{}, 4)
	for dec.More() {
		keyTok, err := dec.Token()
		if err != nil {
			return nil, fmt.Errorf("decode object key: %w", err)
		}
		key, ok := keyTok.(string)
		if !ok {
			return nil, errors.New("object key is not a string")
		}
		if _, dup := seen[key]; dup {
			// Return the parent path so the diagnostic names the
			// duplicate key as the property.
			return nil, &duplicateKeyError{Path: instancePath, Key: key}
		}
		seen[key] = struct{}{}
		valueTok, err := dec.Token()
		if err != nil {
			return nil, fmt.Errorf("decode value for key %q: %w", key, err)
		}
		childPath := canonicalJSONPointer(instancePath, key)
		value, err := scanStrictValue(dec, valueTok, childPath)
		if err != nil {
			return nil, err
		}
		out[key] = value
	}
	if _, err := dec.Token(); err != nil {
		return nil, fmt.Errorf("decode object close: %w", err)
	}
	return out, nil
}

func scanStrictArray(dec *json.Decoder, instancePath string) (any, error) {
	out := []any{}
	index := 0
	for dec.More() {
		valueTok, err := dec.Token()
		if err != nil {
			return nil, fmt.Errorf("decode array value: %w", err)
		}
		childPath := canonicalJSONPointer(instancePath, strconv.Itoa(index))
		value, err := scanStrictValue(dec, valueTok, childPath)
		if err != nil {
			return nil, err
		}
		out = append(out, value)
		index++
	}
	if _, err := dec.Token(); err != nil {
		return nil, fmt.Errorf("decode array close: %w", err)
	}
	return out, nil
}

func scalarFromToken(token json.Token) any {
	switch v := token.(type) {
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

// duplicateKeyError is the typed sentinel the syntactic parser
// raises when an object property appears more than once. The path
// is the parent object's instance path so the diagnostic names the
// duplicate key as the property.
type duplicateKeyError struct {
	Path string
	Key  string
}

func (e *duplicateKeyError) Error() string {
	return "duplicate JSON key \"" + e.Key + "\""
}

// parseDiagnosticFromErr converts a syntactic parser error into a
// stable PlanValidationError. Duplicate keys use the
// unknown_property code at the parent object's path; every other
// error uses invalid_json.
func parseDiagnosticFromErr(path string, err error) PlanValidationError {
	var dup *duplicateKeyError
	if errors.As(err, &dup) {
		return PlanValidationError{
			InstancePath: canonicalJSONPointer(dup.Path, dup.Key),
			SchemaPath:   dup.Path,
			Code:         PlanCodeDuplicateProperty,
			Keyword:      KeywordAdditionalProp,
			Message:      dup.Error(),
			PropertyName: dup.Key,
		}
	}
	msg := err.Error()
	if strings.Contains(msg, "duplicate") {
		return PlanValidationError{
			InstancePath: path,
			SchemaPath:   path,
			Code:         PlanCodeDuplicateProperty,
			Keyword:      KeywordAdditionalProp,
			Message:      msg,
		}
	}
	return PlanValidationError{
		InstancePath: path,
		SchemaPath:   path,
		Code:         PlanCodeInvalidJSON,
		Keyword:      KeywordType,
		Message:      msg,
	}
}

// jsonNumberToInteger parses a JSON-number-shaped value into an int.
// The contract follows the typed Go decoder: only JSON number tokens
// lexically valid for Go int fields are accepted. Forms like
// "1.0", "1e0", or "1.5" are rejected with invalid_type. Strings are
// also rejected. Returns (0, false) when the value is not a
// lexically-valid integer.
func jsonNumberToInteger(value any) (int, bool) {
	switch v := value.(type) {
	case json.Number:
		s := string(v)
		if i, err := strconv.ParseInt(s, 10, 64); err == nil {
			return int(i), true
		}
	}
	return 0, false
}
