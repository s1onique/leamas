package closure

import (
	"slices"
)

// JSONSchema generates a real JSON Schema document from the closure plan
// descriptor authority. This is the stable schema command output.
//
// The generated schema includes:
//   - $schema and $id declarations
//   - properties, required arrays
//   - const and enum constraints
//   - minItems for arrays
//   - additionalProperties: false for strict objects
//   - x-applicability extensions for conditional fields
//
// Internal implementation fields (go_name) are not exposed.
func JSONSchema() map[string]any {
	contract := planContractV1()
	root := contract.Root

	schema := map[string]any{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"$id":     "https://leamas.io/closure-plan/v1/schema.json",
		"title":   "Closure Protocol v1 Plan",
		"type":    "object",
	}

	props, required := buildObjectProperties(root)
	schema["properties"] = props
	if len(required) > 0 {
		schema["required"] = required
	}
	schema["additionalProperties"] = false

	// Add alias documentation
	if len(contract.TopLevelAliases) > 0 || len(contract.AliasSubpaths) > 0 {
		schema["$defs"] = map[string]any{
			"aliases": map[string]any{
				"description": "Migration aliases accepted during deserialization",
				"type":        "object",
				"examples":    contract.TopLevelAliases,
				"subpaths":    contract.AliasSubpaths,
			},
		}
	}

	return schema
}

func buildObjectProperties(obj planObjectDescriptor) (map[string]any, []string) {
	props := make(map[string]any)
	var required []string

	for name, field := range obj.Fields {
		fieldSchema := buildFieldSchema(field)
		if fieldSchema != nil {
			props[name] = fieldSchema
			if field.Required {
				required = append(required, name)
			}
		}
	}

	// Sort required for determinism
	slices.Sort(required)

	return props, required
}

func buildFieldSchema(field planFieldDescriptor) map[string]any {
	// Determine JSON type from Go kind
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
	default:
		// Skip unknown kinds
		return nil
	}

	schema := map[string]any{
		"description": field.Description,
	}

	// Handle constant values
	if field.ConstantValue != nil {
		schema["const"] = field.ConstantValue
		return schema
	}

	// Set type
	schema["type"] = jsonType

	// Handle enum values
	if len(field.EnumAuthority) > 0 {
		if jsonType == "string" {
			schema["enum"] = field.EnumAuthority
		}
	}

	// Handle array items
	if field.Kind == kindArray && field.ItemDescriptor != nil {
		itemSchema := buildFieldSchema(*field.ItemDescriptor)
		if itemSchema != nil {
			schema["items"] = itemSchema
		}
	}

	// Handle minItems for arrays
	if field.Kind == kindArray && field.MinItems > 0 {
		schema["minItems"] = field.MinItems
	}

	// Handle object children
	if field.Kind == kindObject && field.Children != nil {
		childProps, childRequired := buildObjectProperties(*field.Children)
		schema["type"] = "object"
		schema["properties"] = childProps
		schema["additionalProperties"] = false
		if len(childRequired) > 0 {
			schema["required"] = childRequired
		}
	}

	// Add example value if present
	if field.ExampleValue != nil {
		schema["examples"] = []any{field.ExampleValue}
	}

	// Handle applicability rules as extension
	if len(field.ApplicabilityRules) > 0 {
		schema["x-applicability"] = field.ApplicabilityRules
	}

	return schema
}
