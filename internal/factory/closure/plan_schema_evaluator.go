package closure

import (
	"fmt"
	"strings"
)

// plan_schema_evaluator.go implements a generic, schema-driven
// evaluator for Closure Protocol v1 plans. The evaluator
// consumes the bytes produced by JSONSchema() — every decision
// flows from the schema document; the evaluator does not
// hardcode any field name, value, or rule.
//
// The evaluator supports two modes:
//
//   - Standard mode: ignores any Leamas-specific extension
//     (x-leamas-*, x-applicability) so a generic JSON Schema
//     consumer sees only the portable subset
//     (type/required/properties/minLength/pattern/minimum/
//     maximum/items/additionalProperties/const/enum).
//
//   - Extension-aware mode: interprets the published Leamas
//     extensions on top of the standard subset.
//
// Both modes report the evaluation result via schemaEvaluation
// so downstream tooling can distinguish them from the runtime
// result without parsing message text.

// schemaEvaluation is the typed result of the generic schema
// evaluator. The struct is JSON-marshallable so it can flow
// through the public CLI without an additional translation
// step.
type schemaEvaluation struct {
	Mode   string   `json:"mode"` // "standard" or "extension_aware"
	Accept bool     `json:"accept"`
	Issues []string `json:"issues,omitempty"`
}

// acceptStd / acceptExt are pre-built acceptances so the
// evaluator can return them inline.
var (
	acceptStd = schemaEvaluation{Mode: "standard", Accept: true}
	acceptExt = schemaEvaluation{Mode: "extension_aware", Accept: true}
)

// evaluateWithSchemaStandard runs the standard JSON Schema
// subset (no Leamas extensions) over the supplied instance.
func evaluateWithSchemaStandard(schema map[string]any, instance map[string]any) schemaEvaluation {
	return schemaEval(schema, instance, false)
}

// evaluateWithSchemaExtensionAware runs the standard subset AND
// interprets the Leamas extensions x-applicability and
// x-leamas-repository-relative-path.
func evaluateWithSchemaExtensionAware(schema map[string]any, instance map[string]any) schemaEvaluation {
	return schemaEval(schema, instance, true)
}

// schemaEval walks a parsed schema map against an instance map.
// When leamasExtensions is false, any x-* extension members are
// ignored as far as the standard evaluator is concerned; the
// evaluator still reads them but treats them as opaque metadata.
func schemaEval(schema map[string]any, instance map[string]any, leamasExtensions bool) schemaEvaluation {
	if schema == nil {
		return schemaEvaluation{Accept: false, Issues: []string{"schema is nil"}}
	}
	if instance == nil {
		return schemaEvaluation{Accept: false, Issues: []string{"instance is nil"}}
	}
	if t, ok := schema["type"].(string); ok && t != "object" {
		return schemaEvaluation{Accept: false, Issues: []string{fmt.Sprintf("schema type %q does not match instance", t)}}
	}
	if v, ok := schema["const"]; ok {
		if !schemaValueEqual(v, instance) {
			return schemaEvaluation{Accept: false, Issues: []string{"instance does not equal const"}}
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
			return schemaEvaluation{Accept: false, Issues: []string{"instance not in enum"}}
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
				return schemaEvaluation{Accept: false, Issues: []string{"required property missing: " + name}}
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
				return schemaEvaluation{Accept: false, Issues: []string{"unknown property: " + k}}
			}
		}
	}
	return acceptAny(leamasExtensions)
}

// jsonStringifyForSchema renders a value as a deterministic
// JSON string for equality. The function uses the same encoding
// choices the JSON Schema ecosystem expects (numbers preserved,
// strings unescaped).
func jsonStringifyForSchema(v any) (string, bool) {
	switch x := v.(type) {
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
func acceptAny(leamasExtensions bool) schemaEvaluation {
	if leamasExtensions {
		return acceptExt
	}
	return acceptStd
}

// rejectAny returns the appropriate reject evaluation for the
// requested mode.
func rejectAny(leamasExtensions bool, reason string) schemaEvaluation {
	if leamasExtensions {
		return schemaEvaluation{Mode: "extension_aware", Accept: false, Issues: []string{reason}}
	}
	return schemaEvaluation{Mode: "standard", Accept: false, Issues: []string{reason}}
}
