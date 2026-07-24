package closure

import (
	"fmt"
	"sync"

	jsonschema "github.com/santhosh-tekuri/jsonschema/v6"

	"github.com/s1onique/leamas/internal/factory/closure/schema"
)

// compiledSchemas holds the immutable, race-safe compiled Closure
// Protocol v1 schema. Compilation happens once per process; the
// result is reused for every validation. Bootstrap errors are cached
// so the validator fails closed.
type compiledSchemas struct {
	v1 *jsonschema.Schema
}

var (
	planSchemaOnce   sync.Once
	planSchemaSet    *compiledSchemas
	planSchemaBootEr error
)

// planSchema returns the compiled closure-plan schema, compiling once
// on first use. Bootstrap failures (malformed embedded schema,
// AssertFormat failure, etc.) are returned as an operational error so
// the validator never returns a partial schema set.
func planSchema() (*compiledSchemas, error) {
	planSchemaOnce.Do(compilePlanSchemas)
	return planSchemaSet, planSchemaBootEr
}

// compilePlanSchemas builds the immutable schema set with
// AssertFormat() enabled and a fail-closed resource loader.
func compilePlanSchemas() {
	c := jsonschema.NewCompiler()
	c.AssertFormat()
	c.UseLoader(failClosedPlanLoader{})

	if err := c.AddResource(schema.SchemaIDV1, jsonBytesToAny(schema.MustBytes(schema.VersionV1))); err != nil {
		planSchemaBootEr = fmt.Errorf("add closure plan schema: %w", err)
		return
	}

	v1, err := c.Compile(schema.SchemaIDV1)
	if err != nil {
		planSchemaBootEr = fmt.Errorf("compile closure plan schema: %w", err)
		return
	}

	planSchemaSet = &compiledSchemas{v1: v1}
}

// failClosedPlanLoader rejects any URL that is not one of the
// embedded $ids. The compiler is configured to never fetch remote
// resources.
type failClosedPlanLoader struct{}

func (failClosedPlanLoader) Load(url string) (any, error) {
	return nil, fmt.Errorf("loader: refused external resource %q", url)
}

// ValidatePlanJSON runs the embedded schema against the supplied raw
// JSON. It performs NO semantic validation — that is the strict Go
// decoder's responsibility. Schema-level rejection must agree with
// runtime-level rejection on every fixture; the parity test in
// schema_parity_test.go pins this contract.
//
// The function is exposed so external callers (CLI help, third-party
// tooling) can validate a candidate plan against the embedded
// schema without invoking DecodePlan.
func ValidatePlanJSON(data []byte) error {
	compiled, err := planSchema()
	if err != nil {
		return err
	}
	return validateAgainstPlanSchema(compiled.v1, data)
}
