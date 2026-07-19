package gatesummary

import (
	"errors"
	"fmt"
	"io"
	"strings"

	jsonschema "github.com/santhosh-tekuri/jsonschema/v6"
)

// Result is the outcome of Decode. A successful result contains only a
// Document. Ordinary invalid input contains Diagnostics with a nil Err.
// Operational integration failures set Err and may also carry GS_INTERNAL.
type Result struct {
	Document    Document
	Diagnostics []Diagnostic
	Err         error
}

// Success reports whether the decoder produced a typed document.
func (r Result) Success() bool {
	return r.Err == nil && len(r.Diagnostics) == 0 && r.Document.Version() != 0
}

// Decode runs the strict, bounded, versioned gate-summary reader pipeline.
func Decode(r io.Reader) Result {
	return decodeWithDependencies(r, nil, productionDecodeDependencies())
}

func decodeWithDependencies(r io.Reader, trace *decodeTrace, deps decodeDependencies) Result {
	resetTrace(trace)
	res := Result{}

	// Stage 1: bounded read.
	markStage(trace, stageBoundedRead)
	br := readBounded(r)
	if br.err != nil {
		if isOversize(br.err) {
			ds := &diagnosticSet{}
			ds.add(newDiagnostic(CodeDocumentTooLarge, "",
				"document exceeds 4 MiB cap"))
			res.Diagnostics = ds.emit()
			return res
		}
		res.Err = fmt.Errorf("gate-summary: bounded read: %w", br.err)
		return res
	}
	data := br.data

	// Stages 2-4: syntax/top-level + duplicate-key detection + version probe.
	markStage(trace, stageSyntaxScan)
	env := scanEnvelope(data)
	ds := &diagnosticSet{}
	if env.malformed {
		for _, d := range env.diagnostics {
			ds.add(d)
		}
		res.Diagnostics = ds.emit()
		return res
	}
	if env.trailing {
		if len(env.diagnostics) > 0 {
			ds.add(env.diagnostics[0])
		} else {
			ds.add(newDiagnostic(CodeTrailingJSON, "",
				"second JSON value follows the document"))
		}
		res.Diagnostics = ds.emit()
		return res
	}

	markStage(trace, stageDuplicateKeyScan)
	for _, hit := range detectDuplicateKeys(data) {
		ds.add(newDiagnostic(CodeDuplicateKey, hit.path,
			"duplicate object member name"))
	}
	if len(ds.items) > 0 {
		res.Diagnostics = ds.emit()
		return res
	}

	// Stages 4-5: version classification and dispatch.
	markStage(trace, stageVersionProbe)
	if !env.versionPresent {
		ds.add(newDiagnostic(CodeVersionMissing, "/schema_version",
			"schema_version field is absent"))
		res.Diagnostics = ds.emit()
		return res
	}
	decision := classifyVersion(env.versionToken)
	if decision.code != "" {
		if decision.code == CodeUnsupportedVersion {
			markStage(trace, stageVersionDispatch)
		}
		ds.add(newDiagnostic(decision.code, "/schema_version",
			"schema_version="+decision.raw))
		res.Diagnostics = ds.emit()
		return res
	}
	markStage(trace, stageVersionDispatch)
	markSchemaSelected(trace, decision.version)

	// Stages 6-7: compile, select, and invoke the selected schema.
	markStage(trace, stageSchemaValidation)
	set, bootErr := deps.schemas()
	if bootErr != nil {
		res.Err = fmt.Errorf("gate-summary: schema bootstrap: %w", bootErr)
		return res
	}
	var selected *jsonschema.Schema
	switch decision.version {
	case Version1:
		selected = set.v1
	case Version2:
		selected = set.v2
	}
	markSchemaInvoked(trace)
	validationErr := deps.validate(selected, data)
	if validationErr != nil {
		var ve *jsonschema.ValidationError
		if errors.As(validationErr, &ve) {
			res.Diagnostics = (schemaErrorTranslator{root: ve}).translate()
			if diagnosticsContain(res.Diagnostics, CodeInternal) {
				cause := errors.New("selected schema produced an impossible post-dispatch failure")
				res.Err = fmt.Errorf("gate-summary: schema/type drift: %w", cause)
			}
			return res
		}
		res.Err = fmt.Errorf("gate-summary: schema/type drift: %w", validationErr)
		ds.add(newDiagnostic(CodeInternal, "/", "selected schema validation failed operationally"))
		res.Diagnostics = ds.emit()
		return res
	}

	// Stages 9-10: strict wire decode + sealed Document construction.
	markStage(trace, stageWireDecode)
	markWireDecoded(trace)
	strictResult, wireErr := deps.decodeStrict(data, decision.version)
	if wireErr != nil {
		res.Err = fmt.Errorf("gate-summary: schema/type drift: %w", wireErr)
		ds.add(newDiagnostic(CodeInternal, "/", "strict wire decode disagreed with schema"))
		res.Diagnostics = ds.emit()
		return res
	}
	res.Document = strictResult.doc
	return res
}

func diagnosticsContain(diagnostics []Diagnostic, code string) bool {
	for _, diagnostic := range diagnostics {
		if diagnostic.Code == code {
			return true
		}
	}
	return false
}

// versionFromString is a small helper used by tests.
func versionFromString(s string) Version {
	switch strings.TrimSpace(s) {
	case "1":
		return Version1
	case "2":
		return Version2
	}
	return 0
}
