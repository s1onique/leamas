package closure

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"

	jsonschema "github.com/santhosh-tekuri/jsonschema/v6"
)

// planJSONNumberDecoder builds a json.Decoder with UseNumber enabled.
// We use it so the schema validator can compare numbers exactly
// without losing precision through float64 round-tripping.
func planJSONNumberDecoder(r io.Reader) *json.Decoder {
	dec := json.NewDecoder(r)
	dec.UseNumber()
	return dec
}

// planJSONNewNumberDecoder is a package-level wrapper used during
// bootstrap. It is split out so tests can override it if needed.
var planJSONNewNumberDecoder = func(data []byte) *json.Decoder {
	return planJSONNumberDecoder(bytes.NewReader(data))
}

// validateAgainstPlanSchema runs the compiled schema against a
// precision-preserving generic JSON value derived from data. The
// returned *jsonschema.ValidationError has the structured fields the
// translator needs.
func validateAgainstPlanSchema(sch *jsonschema.Schema, data []byte) error {
	dec := planJSONNewNumberDecoder(data)
	var v any
	if err := dec.Decode(&v); err != nil {
		return fmt.Errorf("internal: schema input decode: %w", err)
	}
	return sch.Validate(v)
}

// jsonBytesToAny unmarshals a JSON byte slice into a generic value
// using UseNumber so that numbers survive without float conversion.
func jsonBytesToAny(data []byte) any {
	var v any
	dec := planJSONNewNumberDecoder(data)
	if err := dec.Decode(&v); err != nil {
		// AddResource accepts the unmarshaled value; if decoding
		// fails the schema is structurally invalid and the
		// caller's Compile step will report it.
		return nil
	}
	return v
}
