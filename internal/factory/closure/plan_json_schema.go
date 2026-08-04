package closure

import (
	"errors"
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
//   - additionalProperties: false for strict objects
//   - x-applicability extensions for conditional fields
//
// Internal implementation fields (go_name) are not exposed.
// Migration aliases are NOT included in the public schema.
func JSONSchema() (map[string]any, error) {
	contract := planContractV1()
	root := contract.Root

	schema := map[string]any{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"$id":     "https://leamas.io/closure-plan/v1/schema.json",
		"title":   "Closure Protocol v1 Plan",
		"type":    "object",
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

	// Sort required for determinism
	slices.Sort(required)

	return props, required, nil
}

func buildFieldSchema(field planFieldDescriptor, path string) (map[string]any, error) {
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
	case kindEnum:
		jsonType = "string"
	default:
		return nil, errors.New("unknown kind at " + path)
	}

	schema := map[string]any{
		"description": field.Description,
	}

	// Handle constant values
	if field.ConstantValue != nil {
		schema["const"] = field.ConstantValue
		return schema, nil
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
	if field.Kind == kindArray {
		if field.ItemDescriptor == nil {
			return nil, errors.New("array without item descriptor at " + path)
		}
		itemSchema, err := buildFieldSchema(*field.ItemDescriptor, path+"_items")
		if err != nil {
			return nil, err
		}
		if itemSchema != nil {
			schema["items"] = itemSchema
		}
	}

	// Handle minItems for arrays
	if field.Kind == kindArray && field.MinItems > 0 {
		schema["minItems"] = field.MinItems
	}

	// Handle object children
	if field.Kind == kindObject {
		if field.Children == nil {
			return nil, errors.New("object missing required child/value schema at " + path)
		}
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

	// Add example value if present
	if field.ExampleValue != nil {
		schema["examples"] = []any{field.ExampleValue}
	}

	// Handle applicability rules as extension
	if len(field.ApplicabilityRules) > 0 {
		schema["x-applicability"] = field.ApplicabilityRules
	}

	return schema, nil
}
