package doctrinecompiler

import (
	"encoding/json"
	"fmt"
	"io"
)

// strictDecode reads exactly one JSON value from r. It rejects unknown
// fields and any trailing data after the first value.
//
// The trailing-data check uses a second Decode call to detect a second
// top-level value, since json.Decoder.More() only reports whether the
// current array or object contains more elements.
func strictDecode(r io.Reader, v interface{}) error {
	dec := json.NewDecoder(r)
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		return err
	}
	// Try to decode a second value. EOF is the only acceptable outcome.
	var extra interface{}
	if err := dec.Decode(&extra); err != nil {
		if err == io.EOF {
			return nil
		}
		return fmt.Errorf("trailing data after JSON document")
	}
	return fmt.Errorf("trailing data after JSON document")
}
