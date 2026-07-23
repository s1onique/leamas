package closure

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
)

func decodeStrictBounded(data []byte, limit int, dst any) error {
	if len(data) == 0 {
		return errors.New("empty JSON document")
	}
	if len(data) > limit {
		return fmt.Errorf("JSON document exceeds %d-byte limit", limit)
	}
	if err := rejectDuplicateKeys(data); err != nil {
		return err
	}
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		return fmt.Errorf("decode JSON: %w", err)
	}
	var trailing any
	if err := dec.Decode(&trailing); !errors.Is(err, io.EOF) {
		if err == nil {
			return errors.New("trailing JSON value")
		}
		return fmt.Errorf("trailing JSON data: %w", err)
	}
	return nil
}

func rejectDuplicateKeys(data []byte) error {
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.UseNumber()
	first, err := dec.Token()
	if err != nil {
		return fmt.Errorf("decode JSON tokens: %w", err)
	}
	if err := scanJSONValue(dec, first, 0); err != nil {
		return err
	}
	if _, err := dec.Token(); !errors.Is(err, io.EOF) {
		if err == nil {
			return errors.New("trailing JSON value")
		}
		return fmt.Errorf("trailing JSON data: %w", err)
	}
	return nil
}

func scanJSONValue(dec *json.Decoder, token json.Token, depth int) error {
	if depth > MaxJSONDepth {
		return fmt.Errorf("JSON nesting exceeds %d", MaxJSONDepth)
	}
	delim, composite := token.(json.Delim)
	if !composite {
		return nil
	}
	switch delim {
	case '{':
		seen := make(map[string]struct{})
		for dec.More() {
			keyToken, err := dec.Token()
			if err != nil {
				return fmt.Errorf("decode JSON object key: %w", err)
			}
			key, ok := keyToken.(string)
			if !ok {
				return errors.New("JSON object key is not a string")
			}
			if _, exists := seen[key]; exists {
				return fmt.Errorf("duplicate JSON key %q", key)
			}
			seen[key] = struct{}{}
			value, err := dec.Token()
			if err != nil {
				return fmt.Errorf("decode JSON value for %q: %w", key, err)
			}
			if err := scanJSONValue(dec, value, depth+1); err != nil {
				return err
			}
		}
	case '[':
		for dec.More() {
			value, err := dec.Token()
			if err != nil {
				return fmt.Errorf("decode JSON array value: %w", err)
			}
			if err := scanJSONValue(dec, value, depth+1); err != nil {
				return err
			}
		}
	default:
		return fmt.Errorf("unexpected JSON delimiter %q", delim)
	}
	if _, err := dec.Token(); err != nil {
		return fmt.Errorf("decode JSON closing delimiter: %w", err)
	}
	return nil
}
