package gatesummary

import (
	"io"

	jsonschema "github.com/santhosh-tekuri/jsonschema/v6"
)

// stage identifies the pipeline stage that owns a Decode result.
// It is intentionally unexported: stage evidence is an internal test
// boundary, not part of the public gate-summary API.
type stage string

const (
	stageBoundedRead      stage = "bounded-read"
	stageSyntaxScan       stage = "syntax-scan"
	stageDuplicateKeyScan stage = "duplicate-key-scan"
	stageVersionProbe     stage = "version-probe"
	stageVersionDispatch  stage = "version-dispatch"
	stageSchemaValidation stage = "schema-validation"
	stageWireDecode       stage = "wire-decode"
)

// decodeTrace records stage ownership and downstream invocation evidence.
// Tests pass it through decodeWithTrace; Decode itself allocates no trace.
type decodeTrace struct {
	Stage          stage
	SchemaSelected Version
	SchemaInvoked  bool
	WireDecoded    bool
}

// decodeDependencies are the narrow capabilities used by the orchestrator.
// Tests replace individual functions to prove operational-failure behavior.
type decodeDependencies struct {
	schemas      func() (*compiledSchemas, error)
	validate     func(*jsonschema.Schema, []byte) error
	decodeStrict func([]byte, Version) (strictDecodeResult, error)
}

func productionDecodeDependencies() decodeDependencies {
	return decodeDependencies{
		schemas:      schemas,
		validate:     validateAgainstSchema,
		decodeStrict: decodeStrict,
	}
}

func decodeWithTrace(r io.Reader, trace *decodeTrace) Result {
	return decodeWithDependencies(r, trace, productionDecodeDependencies())
}

func resetTrace(trace *decodeTrace) {
	if trace != nil {
		*trace = decodeTrace{}
	}
}

func markStage(trace *decodeTrace, owner stage) {
	if trace != nil {
		trace.Stage = owner
	}
}

func markSchemaSelected(trace *decodeTrace, version Version) {
	if trace != nil {
		trace.SchemaSelected = version
	}
}

func markSchemaInvoked(trace *decodeTrace) {
	if trace != nil {
		trace.SchemaInvoked = true
	}
}

func markWireDecoded(trace *decodeTrace) {
	if trace != nil {
		trace.WireDecoded = true
	}
}
