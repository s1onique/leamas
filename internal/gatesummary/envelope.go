package gatesummary

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
)

// envelopeResult is the syntax-scan and version-probe outcome.
type envelopeResult struct {
	versionPresent bool
	versionToken   json.Token
	diagnostics    []Diagnostic
	malformed      bool
	trailing       bool
}

// scanEnvelope validates one top-level object and preserves the complete
// schema_version member's first token. Container values are fully consumed
// so nested keys cannot corrupt top-level scanner state.
func scanEnvelope(data []byte) envelopeResult {
	res := envelopeResult{}
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.UseNumber()

	tok, err := dec.Token()
	if err != nil {
		return malformedEnvelope(res, "", err)
	}
	if delim, ok := tok.(json.Delim); !ok || delim != '{' {
		res.malformed = true
		res.diagnostics = append(res.diagnostics,
			newDiagnostic(CodeMalformedJSON, "", "top-level JSON value is not an object"))
		return res
	}

	for dec.More() {
		keyToken, keyErr := dec.Token()
		if keyErr != nil {
			return malformedEnvelope(res, "", keyErr)
		}
		key, ok := keyToken.(string)
		if !ok {
			res.malformed = true
			res.diagnostics = append(res.diagnostics,
				newDiagnostic(CodeMalformedJSON, "/", "object member name is not a string"))
			return res
		}

		valueToken, valueErr := consumeValue(dec)
		if valueErr != nil {
			return malformedEnvelope(res, "/"+escapePointer(key), valueErr)
		}
		if key == "schema_version" {
			res.versionPresent = true
			res.versionToken = valueToken
		}
	}

	closing, closeErr := dec.Token()
	if closeErr != nil {
		return malformedEnvelope(res, "", closeErr)
	}
	if delim, ok := closing.(json.Delim); !ok || delim != '}' {
		return malformedEnvelope(res, "", errors.New("top-level object did not close"))
	}

	_, nextErr := dec.Token()
	switch {
	case errors.Is(nextErr, io.EOF):
		return res
	case nextErr != nil:
		return malformedEnvelope(res, "", nextErr)
	default:
		res.trailing = true
		res.diagnostics = append(res.diagnostics,
			newDiagnostic(CodeTrailingJSON, "", "second JSON value follows the document"))
		return res
	}
}

// consumeValue consumes exactly one complete JSON value and returns its
// first token. For arrays and objects it walks the complete token stream,
// preserving only the top-level delimiter needed for type classification.
func consumeValue(dec *json.Decoder) (json.Token, error) {
	first, err := dec.Token()
	if err != nil {
		return nil, err
	}
	delim, isContainer := first.(json.Delim)
	if !isContainer || (delim != '{' && delim != '[') {
		return first, nil
	}

	depth := 1
	for depth > 0 {
		tok, tokenErr := dec.Token()
		if tokenErr != nil {
			return nil, tokenErr
		}
		d, ok := tok.(json.Delim)
		if !ok {
			continue
		}
		switch d {
		case '{', '[':
			depth++
		case '}', ']':
			depth--
		}
	}
	return first, nil
}

// skipValue consumes any single JSON value from the decoder.
func skipValue(dec *json.Decoder) error {
	_, err := consumeValue(dec)
	return err
}

func malformedEnvelope(res envelopeResult, path string, err error) envelopeResult {
	res.malformed = true
	res.diagnostics = append(res.diagnostics,
		newDiagnostic(CodeMalformedJSON, path, jsonErrorMessage(err)))
	return res
}

// jsonErrorMessage returns short deterministic text without source bytes.
func jsonErrorMessage(err error) string {
	if err == nil {
		return ""
	}
	var syntaxError *json.SyntaxError
	if errors.As(err, &syntaxError) {
		return "malformed JSON"
	}
	return "malformed JSON"
}
