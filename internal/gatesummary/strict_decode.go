package gatesummary

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
)

// strictDecodeResult is the Stage 9 outcome.
type strictDecodeResult struct {
	doc     Document
	wireErr error
}

// decodeStrict decodes data into the version-specific wire Document.
// It enforces:
//   - DisallowUnknownFields (defense in depth after schema success)
//   - UseNumber for any remaining numeric values
//   - exactly one top-level value
//   - EOF after the document
//
// WireInteger fields preserve exact numeric spellings while the selected
// schema remains authoritative for integer and range constraints.
func decodeStrict(data []byte, version Version) (strictDecodeResult, error) {
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.UseNumber()
	dec.DisallowUnknownFields()
	switch version {
	case Version1:
		var s V1Summary
		if err := dec.Decode(&s); err != nil {
			return strictDecodeResult{}, fmt.Errorf("strict v1 decode: %w", err)
		}
		if err := requireEOF(dec); err != nil {
			return strictDecodeResult{}, fmt.Errorf("strict v1 decode: %w", err)
		}
		return strictDecodeResult{doc: newDocumentV1(s)}, nil
	case Version2:
		var s V2Summary
		if err := dec.Decode(&s); err != nil {
			return strictDecodeResult{}, fmt.Errorf("strict v2 decode: %w", err)
		}
		if err := requireEOF(dec); err != nil {
			return strictDecodeResult{}, fmt.Errorf("strict v2 decode: %w", err)
		}
		return strictDecodeResult{doc: newDocumentV2(s)}, nil
	default:
		return strictDecodeResult{}, fmt.Errorf("internal: unsupported version %d", version)
	}
}

// requireEOF confirms that the decoder has consumed exactly one value.
func requireEOF(dec *json.Decoder) error {
	_, err := dec.Token()
	switch {
	case errors.Is(err, io.EOF):
		return nil
	case err != nil:
		return err
	default:
		return errExtraJSON{}
	}
}

// errExtraJSON signals that the decoder observed extra JSON after the
// first value during strict decoding.
type errExtraJSON struct{}

// Error implements the error interface.
func (errExtraJSON) Error() string {
	return "gate-summary: extra JSON after document"
}
