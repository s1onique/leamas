package gatesummary

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"

	jsonschema "github.com/santhosh-tekuri/jsonschema/v6"
)

// schemaValidator validates a precision-preserving generic JSON
// value against a compiled jsonschema/v6 schema.
type schemaValidator interface {
	validate(v any) error
}

// jsonschemaValidator wraps a jsonschema/v6 compiled schema.
type jsonschemaValidator struct {
	sch *jsonschema.Schema
}

func (j jsonschemaValidator) validate(v any) error {
	return j.sch.Validate(v)
}

// newJSONNumberDecoder returns a json.Decoder with UseNumber enabled.
func newJSONNumberDecoder(r io.Reader) *json.Decoder {
	dec := json.NewDecoder(r)
	dec.UseNumber()
	return dec
}

// jsonNewNumberDecoder is a package-level wrapper used during
// bootstrap. It is split out so tests can override it if needed.
var jsonNewNumberDecoder = func(data []byte) *json.Decoder {
	return newJSONNumberDecoder(bytes.NewReader(data))
}

// validateAgainstSchema runs the compiled schema against a
// precision-preserving generic JSON value derived from data. The
// returned *jsonschema.ValidationError has the structured fields the
// schema-error translator needs.
func validateAgainstSchema(sch *jsonschema.Schema, data []byte) error {
	dec := jsonNewNumberDecoder(data)
	var v any
	if err := dec.Decode(&v); err != nil {
		return fmt.Errorf("internal: schema input decode: %w", err)
	}
	return sch.Validate(v)
}
