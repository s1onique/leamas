// Package evaltest provides the Closure Protocol v1 schema
// evaluator used exclusively by the closure package's parity
// tests. The package has NO production callers; every consumer
// in the repository is a *_test.go file.
//
// ACT-LEAMAS-FACTORY-CLOSE-PLAN-V1-RUN-EXECUTION-FIELDS-
// CONTRACT-PARITY01-CORRECTION16 placed the evaluator here
// because the structural validator (closure.ValidatePlanStructural)
// is the production authority. The evaluator in this package
// is a parity witness: it exercises the round-tripped JSON
// Schema document so the four-layer matrix (standard,
// extension-aware, structural, composed) can be compared
// without re-implementing JSON Schema by hand.
package evaltest

import (
	"encoding/json"
	"fmt"
	"strings"
)

// SchemaEvaluation is the typed result of the generic schema
// evaluator. The struct is JSON-marshallable so it can flow
// through the parity-test fixtures without an additional
// translation step.
type SchemaEvaluation struct {
	Mode   string   `json:"mode"` // "standard", "extension_aware", or "schema_issue"
	Accept bool     `json:"accept"`
	Issues []string `json:"issues,omitempty"`
}

// acceptStd / acceptExt are pre-built acceptances so the
// evaluator can return them inline.
var (
	acceptStd = SchemaEvaluation{Mode: "standard", Accept: true}
	acceptExt = SchemaEvaluation{Mode: "extension_aware", Accept: true}
)

// EvaluateWithSchemaStandard runs the standard JSON Schema
// subset (no Leamas extensions) over the supplied instance.
func EvaluateWithSchemaStandard(schema map[string]any, instance map[string]any) SchemaEvaluation {
	return schemaEval(schema, instance, false)
}

// EvaluateWithSchemaExtensionAware runs the standard subset
// AND interprets the Leamas extensions x-applicability and
// x-leamas-repository-relative-path after the extension-aware
// evaluator has verified the dialect URI, resolved the embedded
// Leamas meta-schema, verified $id, read $vocabulary, and
// rejected unsupported required vocabularies.
func EvaluateWithSchemaExtensionAware(schema map[string]any, instance map[string]any) SchemaEvaluation {
	if err := ensureDialectAdmitted(schema); err != nil {
		return SchemaEvaluation{Mode: "extension_aware", Accept: false, Issues: []string{err.Error()}}
	}
	return schemaEval(schema, instance, true)
}

// schemaEval walks a parsed schema map against an instance map.
// When leamasExtensions is false, any x-* extension members are
// ignored as far as the standard evaluator is concerned; the
// evaluator still reads them but treats them as opaque metadata.
func schemaEval(schema map[string]any, instance map[string]any, leamasExtensions bool) SchemaEvaluation {
	if schema == nil {
		return SchemaEvaluation{Accept: false, Issues: []string{"schema is nil"}}
	}
	if instance == nil {
		return SchemaEvaluation{Accept: false, Issues: []string{"instance is nil"}}
	}
	if t, ok := schema["type"].(string); ok && t != "object" {
		return SchemaEvaluation{Accept: false, Issues: []string{fmt.Sprintf("schema type %q does not match instance", t)}}
	}
	if v, ok := schema["const"]; ok {
		if !schemaValueEqual(v, instance) {
			return SchemaEvaluation{Accept: false, Issues: []string{"instance does not equal const"}}
		}
		return acceptAny(leamasExtensions)
	}
	if enum, ok := schema["enum"].([]any); ok {
		matched := false
		for _, e := range enum {
			if schemaValueEqual(e, instance) {
				matched = true
				break
			}
		}
		if !matched {
			return SchemaEvaluation{Accept: false, Issues: []string{"instance not in enum"}}
		}
		return acceptAny(leamasExtensions)
	}
	// Required properties.
	if req, ok := schema["required"].([]any); ok {
		for _, r := range req {
			name, ok := r.(string)
			if !ok {
				continue
			}
			if _, present := instance[name]; !present {
				return SchemaEvaluation{Accept: false, Issues: []string{"required property missing: " + name}}
			}
		}
	}
	// x-applicability walker for the top-level schema's
	// properties. The walker runs in extension-aware mode so
	// the same fail-closed shape validation that fires inside
	// evalField also fires for the root object's properties.
	// Properties without an x-applicability extension are
	// skipped: a missing extension is not a schema failure,
	// only a malformed one is.
	if leamasExtensions {
		if props, hasProps := schema["properties"].(map[string]any); hasProps {
			for propName, propSchemaAny := range props {
				psm, ok := propSchemaAny.(map[string]any)
				if !ok {
					continue
				}
				rawApp, hasApp := psm["x-applicability"]
				if !hasApp {
					continue
				}
				app, ok, appErr := normalizeApplicabilityRules(rawApp)
				if !ok {
					return rejectAny(leamasExtensions, appErr)
				}
				_, fieldPresent := instance[propName]
				ctx := schemaPropertyContext{
					PropertyName:   propName,
					PropertySchema: psm,
					ParentSchema:   schema,
					ParentInstance: instance,
					Value:          nil,
					Present:        fieldPresent,
				}
				if fieldPresent {
					ctx.Value = instance[propName]
				}
				if res := applyApplicabilityRules(ctx, app, leamasExtensions); !res.Accept {
					return res
				}
			}
		}
	}
	props, hasProps := schema["properties"].(map[string]any)
	if hasProps {
		for propName, propSchemaAny := range props {
			propSchema, ok := propSchemaAny.(map[string]any)
			if !ok {
				continue
			}
			val, present := instance[propName]

			if !present {
				continue
			}
			if isEmpty(val) && schemaFieldRequiresPresence(propSchema) {
				continue
			}
			subEval := evalField(propSchema, val, leamasExtensions)
			if !subEval.Accept {
				return subEval
			}
		}
	}
	if addl, ok := schema["additionalProperties"]; ok {
		if addlBool, ok := addl.(bool); ok && !addlBool {
			for k := range instance {
				if hasProps {
					if _, found := props[k]; found {
						continue
					}
				}
				return SchemaEvaluation{Accept: false, Issues: []string{"unknown property: " + k}}
			}
		}
	}
	return acceptAny(leamasExtensions)
}

// jsonStringifyForSchema renders a value as a deterministic
// JSON string for equality. The function uses the same encoding
// choices the JSON Schema ecosystem expects (numbers preserved,
// strings unescaped).
//
// CORRECTION05: json.Number (the type emitted by encoding/json
// when decoder.UseNumber is set) is rendered with its literal
// decimal form so equality with float64-derived numeric values
// remains stable across round trips.
func jsonStringifyForSchema(v any) (string, bool) {
	switch x := v.(type) {
	case json.Number:
		return x.String(), true
	case string:
		return x, true
	case bool:
		if x {
			return "true", true
		}
		return "false", true
	case float64:
		return fmt.Sprintf("%g", x), true
	case int:
		return fmt.Sprintf("%d", x), true
	case int64:
		return fmt.Sprintf("%d", x), true
	case nil:
		return "null", true
	case []any:
		parts := make([]string, 0, len(x))
		for _, e := range x {
			s, ok := jsonStringifyForSchema(e)
			if !ok {
				return "", false
			}
			parts = append(parts, s)
		}
		return "[" + strings.Join(parts, ",") + "]", true
	case map[string]any:
		keys := make([]string, 0, len(x))
		for k := range x {
			keys = append(keys, k)
		}
		sortStrings(keys)
		parts := make([]string, 0, len(x))
		for _, k := range keys {
			s, ok := jsonStringifyForSchema(x[k])
			if !ok {
				return "", false
			}
			parts = append(parts, fmt.Sprintf("%q:%s", k, s))
		}
		return "{" + strings.Join(parts, ",") + "}", true
	default:
		return "", false
	}
}

// sortStrings sorts a string slice in place. Small helper to
// avoid pulling in sort for this single use.
func sortStrings(s []string) {
	for i := 1; i < len(s); i++ {
		for j := i; j > 0 && s[j-1] > s[j]; j-- {
			s[j-1], s[j] = s[j], s[j-1]
		}
	}
}

// schemaValueEqual compares two values for schema const/enum
// equality. It uses Go's reflect-style equality through
// map[string]any / []any / primitives.
func schemaValueEqual(a, b any) bool {
	as, okA := jsonStringifyForSchema(a)
	bs, okB := jsonStringifyForSchema(b)
	if okA && okB {
		return as == bs
	}
	return false
}

// acceptAny returns the appropriate accept evaluation for the
// requested mode.
func acceptAny(leamasExtensions bool) SchemaEvaluation {
	if leamasExtensions {
		return acceptExt
	}
	return acceptStd
}

// rejectAny returns the appropriate reject evaluation for the
// requested mode.
func rejectAny(leamasExtensions bool, reason string) SchemaEvaluation {
	if leamasExtensions {
		return SchemaEvaluation{Mode: "extension_aware", Accept: false, Issues: []string{reason}}
	}
	return SchemaEvaluation{Mode: "standard", Accept: false, Issues: []string{reason}}
}
