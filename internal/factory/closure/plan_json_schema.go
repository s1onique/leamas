package closure

import (
	"errors"
	"fmt"
	"slices"
)

// ErrSchemaGeneration indicates a field with an unknown or malformed descriptor.
var ErrSchemaGeneration = errors.New("schema generation failed")

// Leamas Closure Protocol v1 dialect and vocabulary URIs.
//
// CORRECTION04: the emitted schema now selects a stable Leamas
// dialect URI so a generic JSON Schema consumer can detect the
// Leamas extensions (x-applicability, x-leamas-repository-relative-path)
// without parsing prose or hardcoding field names. The dialect
// itself remains a Draft 2020-12 schema; the URI is the stable
// contract identifier.
const (
	leamasClosurePlanV1DialectURI = "https://leamas.io/closure-plan/v1/schema.json"
	leamasVocabularyURI           = "https://leamas.io/closure-plan/v1/vocab"
)

// JSONSchema generates a real JSON Schema document from the closure plan
// descriptor authority. This is the stable schema command output.
//
// The generated schema includes:
//   - $schema and $id declarations selecting the Leamas Closure Plan v1 dialect
//   - properties, required arrays
//   - const and enum constraints
//   - minItems for arrays
//   - minLength/pattern/minimum/maximum value-level constraints
//   - x-leamas-repository-relative-path for path-policy rules JSON
//     Schema cannot express portably
//   - x-applicability extensions for conditional fields
//   - additionalProperties: false for strict objects
//
// Internal implementation fields (go_name) are not exposed.
// Migration aliases are NOT included in the public schema.
func JSONSchema() (map[string]any, error) {
	contract := planContractV1()
	root := contract.Root

	schema := map[string]any{
		"$schema":                       leamasClosurePlanV1DialectURI,
		"$id":                           leamasClosurePlanV1DialectURI,
		"title":                         "Closure Protocol v1 Plan",
		"type":                          "object",
		"x-leamas-dialect":              "leamas-closure-plan-v1",
		"x-leamas-vocabulary":           leamasVocabularyURI,
		"x-leamas-validation-authority": "leamas factory close plan validate",
	}

	props, required, err := buildObjectProperties(root, "")
	if err != nil {
		return nil, err
	}
	schema["properties"] = props
	if len(required) > 0 {
		schema["required"] = required
	}
	schema["additionalProperties"] = false

	return schema, nil
}

func buildObjectProperties(obj planObjectDescriptor, path string) (map[string]any, []string, error) {
	props := make(map[string]any)
	var required []string

	for name, field := range obj.Fields {
		fieldSchema, err := buildFieldSchema(field, path+"/"+name)
		if err != nil {
			return nil, nil, err
		}
		if fieldSchema != nil {
			props[name] = fieldSchema
			if field.Required {
				required = append(required, name)
			}
		}
	}

	slices.Sort(required)

	return props, required, nil
}

func buildFieldSchema(field planFieldDescriptor, path string) (map[string]any, error) {
	var jsonType string
	switch field.Kind {
	case kindString:
		jsonType = "string"
	case kindInteger:
		jsonType = "integer"
	case kindBoolean:
		jsonType = "boolean"
	case kindObject:
		jsonType = "object"
	case kindArray:
		jsonType = "array"
	case kindEnum:
		jsonType = "string"
	default:
		return nil, fmt.Errorf("%w: unknown kind at %s", ErrSchemaGeneration, path)
	}

	schema := map[string]any{
		"description": field.Description,
	}

	if field.ConstantValue != nil {
		schema["const"] = field.ConstantValue
		return schema, nil
	}

	schema["type"] = jsonType

	if len(field.EnumAuthority) > 0 {
		if jsonType == "string" {
			schema["enum"] = field.EnumAuthority
		}
	}

	if field.Kind == kindArray {
		if field.ItemDescriptor == nil {
			return nil, fmt.Errorf("%w: array without item descriptor at %s", ErrSchemaGeneration, path)
		}
		itemSchema, err := buildFieldSchema(*field.ItemDescriptor, path+"_items")
		if err != nil {
			return nil, err
		}
		if itemSchema != nil {
			schema["items"] = itemSchema
		}
	}

	if field.Kind == kindArray && field.MinItems > 0 {
		schema["minItems"] = field.MinItems
	}

	if field.Kind == kindObject {
		if field.Children == nil {
			return nil, fmt.Errorf("%w: object missing children at %s", ErrSchemaGeneration, path)
		}
		if field.Children.Kind == objectStringMap {
			schema["type"] = "object"
			schema["additionalProperties"] = map[string]any{"type": "string"}
		} else {
			childProps, childRequired, err := buildObjectProperties(*field.Children, path)
			if err != nil {
				return nil, err
			}
			schema["type"] = "object"
			schema["properties"] = childProps
			schema["additionalProperties"] = false
			if len(childRequired) > 0 {
				schema["required"] = childRequired
			}
		}
	}

	if field.MinLength != nil {
		schema["minLength"] = *field.MinLength
	}
	if field.Pattern != "" {
		schema["pattern"] = field.Pattern
	}
	if field.Minimum != nil {
		schema["minimum"] = *field.Minimum
	}
	if field.Maximum != nil {
		schema["maximum"] = *field.Maximum
	}
	if field.PathPolicy != nil {
		schema["x-leamas-repository-relative-path"] = pathPolicyToMap(field.PathPolicy)
	}

	if field.ExampleValue != nil {
		schema["examples"] = []any{field.ExampleValue}
	}

	if len(field.ApplicabilityRules) > 0 {
		seen := make(map[string]bool)
		for _, rule := range field.ApplicabilityRules {
			key := rule.Sibling + "=" + rule.Value
			if seen[key] {
				return nil, fmt.Errorf("%w: duplicate applicability identity at %s: %s=%s", ErrSchemaGeneration, path, rule.Sibling, rule.Value)
			}
			seen[key] = true
		}
		// CORRECTION04: emit rules as []map[string]any so the
		// generic schema evaluator (which inspects []any) sees
		// the same wire shape that the public CLI serialises.
		// Returning a typed struct slice would hide the rules
		// from generic JSON Schema consumers.
		schema["x-applicability"] = toApplicabilityMaps(field.ApplicabilityRules)
	}

	return schema, nil
}

// pathPolicyToMap renders a planPathPolicy as a map[string]any
// with the canonical snake_case JSON keys so consumers see the
// documented public shape.
func pathPolicyToMap(p *planPathPolicy) map[string]any {
	return map[string]any{
		"allow_dot":               p.AllowDot,
		"allow_parent_segments":   p.AllowParentSegments,
		"require_lexically_clean": p.RequireLexicallyClean,
		"separator":               p.Separator,
	}
}

// toApplicabilityMaps converts internal applicability rules to
// the public wire shape []map[string]any. The shape mirrors the
// public JSON Schema declaration: { sibling, value, presence }.
func toApplicabilityMaps(rules []fieldApplicabilityRule) []map[string]any {
	out := make([]map[string]any, len(rules))
	for i, rule := range rules {
		out[i] = map[string]any{
			"sibling":  rule.Sibling,
			"value":    rule.Value,
			"presence": rule.Presence.String(),
		}
	}
	return out
}
