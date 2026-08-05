package closure

import (
	"errors"
	"fmt"
	"slices"
)

// ErrSchemaGeneration indicates a field with an unknown or malformed descriptor.
var ErrSchemaGeneration = errors.New("schema generation failed")

// JSONSchema generates a real JSON Schema document from the closure plan
// descriptor authority. This is the stable schema command output.
//
// The generated schema includes:
//   - $schema and $id declarations
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
		"$schema":                       "https://json-schema.org/draft/2020-12/schema",
		"$id":                           "https://leamas.io/closure-plan/v1/schema.json",
		"title":                         "Closure Protocol v1 Plan",
		"type":                          "object",
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
		schema["x-applicability"] = toApplicabilityDTOs(field.ApplicabilityRules)
	}

	return schema, nil
}

// pathPolicyDTO is retained for documentation purposes; the
// schema generator emits pathPolicyToMap directly so consumers
// always see the canonical snake_case public keys.
type pathPolicyDTO struct {
	AllowDot              bool   `json:"allow_dot"`
	AllowParentSegments   bool   `json:"allow_parent_segments"`
	RequireLexicallyClean bool   `json:"require_lexically_clean"`
	Separator             string `json:"separator"`
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

// applicabilityDTO is the frozen public wire shape for the
// x-applicability extension.
type applicabilityDTO struct {
	Sibling  string `json:"sibling"`
	Value    string `json:"value"`
	Presence string `json:"presence"`
}

// toApplicabilityDTOs converts internal applicability rules to
// the public DTO.
func toApplicabilityDTOs(rules []fieldApplicabilityRule) []applicabilityDTO {
	out := make([]applicabilityDTO, len(rules))
	for i, rule := range rules {
		out[i] = applicabilityDTO{
			Sibling:  rule.Sibling,
			Value:    rule.Value,
			Presence: rule.Presence.String(),
		}
	}
	return out
}
