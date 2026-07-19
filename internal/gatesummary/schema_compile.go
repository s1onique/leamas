package gatesummary

import (
	"fmt"
	"sync"

	jsonschema "github.com/santhosh-tekuri/jsonschema/v6"
)

// compiledSchemas holds the immutable, race-safe compiled v1 and v2
// schemas. Compilation happens once per process; the result is reused
// for every Decode call. Bootstrap errors are cached so the decoder
// fails closed.
type compiledSchemas struct {
	v1 *jsonschema.Schema
	v2 *jsonschema.Schema
}

var (
	schemaOnce   sync.Once
	schemaSet    *compiledSchemas
	schemaBootEr error
)

// schemas returns the compiled schema set, compiling once on first
// use. Bootstrap failures (malformed embedded schema, AssertFormat
// failure, etc.) are returned as an operational error so the decoder
// never returns a partial schema set.
func schemas() (*compiledSchemas, error) {
	schemaOnce.Do(compileSchemas)
	return schemaSet, schemaBootEr
}

// compileSchemas builds the immutable schema set with AssertFormat()
// enabled and a fail-closed resource loader.
func compileSchemas() {
	c := jsonschema.NewCompiler()
	c.AssertFormat()
	// Fail-closed: refuse any URL the embedded resources do not
	// resolve. The compiler's default loader is allowed to resolve
	// file: and http: URLs; we replace it so only embedded resources
	// are reachable.
	c.UseLoader(failClosedLoader{})

	if err := c.AddResource(v1SchemaID, jsonBytesToAny(v1SchemaJSON)); err != nil {
		schemaBootEr = fmt.Errorf("add v1 schema: %w", err)
		return
	}
	if err := c.AddResource(v2SchemaID, jsonBytesToAny(v2SchemaJSON)); err != nil {
		schemaBootEr = fmt.Errorf("add v2 schema: %w", err)
		return
	}

	v1, err := c.Compile(v1SchemaID)
	if err != nil {
		schemaBootEr = fmt.Errorf("compile v1 schema: %w", err)
		return
	}
	v2, err := c.Compile(v2SchemaID)
	if err != nil {
		schemaBootEr = fmt.Errorf("compile v2 schema: %w", err)
		return
	}

	schemaSet = &compiledSchemas{v1: v1, v2: v2}
}

// failClosedLoader rejects any URL that is not one of the embedded
// $ids. The compiler is configured to never fetch remote resources.
type failClosedLoader struct{}

func (failClosedLoader) Load(url string) (any, error) {
	return nil, fmt.Errorf("loader: refused external resource %q", url)
}

// jsonBytesToAny unmarshals a JSON byte slice into a generic value
// using UseNumber so that numbers survive without float conversion.
func jsonBytesToAny(data []byte) any {
	var v any
	dec := jsonNewNumberDecoder(data)
	if err := dec.Decode(&v); err != nil {
		// AddResource accepts the unmarshaled value; if decoding
		// fails the schema is structurally invalid and the
		// caller's Compile step will report it.
		return nil
	}
	return v
}
