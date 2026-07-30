package closure

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
)

// plan_contract_validation_engine.go contains the recursive
// structural validator that walks a JSON document against the
// v1 descriptor and emits deterministic
// PlanValidationError diagnostics. Splitting it from
// plan_contract_validation.go keeps every file under the
// LLM-friendly 400-line threshold while preserving the single
// closure over the descriptor that
// ACT-LEAMAS-FACTORY-CLOSE-PLAN-CONTRACT-AUTHORITY01 requires.

// validatePlanObject walks a JSON value against the descriptor for
// its parent object. The instancePath parameter overrides the
// descriptor's static Path; array callers pass the
// path-to-the-item so per-item required diagnostics reference the
// indexed location. The descriptor's static Path is still used as
// SchemaPath (the canonical schema-pointer at which the contract
// declares the field set).
func validatePlanObject(object planObjectDescriptor, raw any, contract planContractV1Descriptor, instancePath string) []PlanValidationError {
	if instancePath == "" {
		instancePath = object.Path
	}
	var diagnostics []PlanValidationError
	if raw == nil {
		diagnostics = append(diagnostics, PlanValidationError{
			InstancePath: instancePath,
			SchemaPath:   object.Path,
			Code:         PlanCodeInvalidType,
			Keyword:      KeywordType,
			Message:      "value must be a JSON object",
		})
		return diagnostics
	}
	asMap, ok := raw.(map[string]any)
	if !ok {
		diagnostics = append(diagnostics, PlanValidationError{
			InstancePath: instancePath,
			SchemaPath:   object.Path,
			Code:         PlanCodeInvalidType,
			Keyword:      KeywordType,
			Message:      fmt.Sprintf("value must be a JSON object, got %T", raw),
		})
		return diagnostics
	}
	// Required-property diagnostics first: every missing required
	// sibling raises one required_property_missing diagnostic so
	// consumers can fix every gap in a single edit.
	for _, required := range object.Required {
		if _, present := asMap[required]; !present {
			diagnostics = append(diagnostics, PlanValidationError{
				InstancePath: canonicalJSONPointer(instancePath, required),
				SchemaPath:   canonicalJSONPointer(object.Path, required),
				Code:         PlanCodeRequiredPropertyMissing,
				Keyword:      KeywordRequired,
				Message:      fmt.Sprintf("missing required property %q", required),
				PropertyName: required,
			})
		}
	}
	// Iterate fields in lexicographic order so the diagnostic
	// stream is deterministic across runs.
	for _, name := range object.fieldNamesSorted() {
		field := object.Fields[name]
		value, present := asMap[name]
		if !present {
			continue
		}
		fieldPath := canonicalJSONPointer(instancePath, name)
		fieldDiags := validatePlanField(field, fieldPath, value, object, asMap, contract)
		diagnostics = append(diagnostics, fieldDiags...)
	}
	// Unknown-property diagnostics next: any JSON key not declared
	// in the descriptor (including aliased names) is rejected with
	// an unknown_property diagnostic.
	known := make(map[string]struct{}, len(object.Fields))
	for name := range object.Fields {
		known[name] = struct{}{}
	}
	unknownKeys := make([]string, 0)
	for key := range asMap {
		if _, ok := known[key]; !ok {
			unknownKeys = append(unknownKeys, key)
		}
	}
	sort.Strings(unknownKeys)
	for _, key := range unknownKeys {
		diagnostics = append(diagnostics, PlanValidationError{
			InstancePath:  canonicalJSONPointer(instancePath, key),
			SchemaPath:    canonicalJSONPointer(object.Path, key),
			Code:          PlanCodeUnknownProperty,
			Keyword:       KeywordAdditionalProp,
			Message:       fmt.Sprintf("unknown property %q", key),
			PropertyName:  key,
			RejectedValue: asMap[key],
		})
	}
	return diagnostics
}

// validatePlanField walks a single field against its descriptor.
// It is the workhorse for type/enum/required diagnostics.
func validatePlanField(field planFieldDescriptor, path string, value any, parent planObjectDescriptor, _ map[string]any, contract planContractV1Descriptor) []PlanValidationError {
	var diagnostics []PlanValidationError
	// Null-handling: pointer fields distinguish absent from null
	// from value. Non-pointer fields treat null as invalid.
	if value == nil {
		if field.Pointer {
			return diagnostics
		}
		diagnostics = append(diagnostics, PlanValidationError{
			InstancePath: path,
			SchemaPath:   path,
			Code:         PlanCodeInvalidType,
			Keyword:      KeywordType,
			Message:      fmt.Sprintf("property %q must not be null", field.JSONName),
		})
		return diagnostics
	}
	switch field.Kind {
	case kindObject:
		if field.Children == nil {
			// Free-form object: validate type only, do not recurse.
			if _, ok := value.(map[string]any); !ok {
				diagnostics = append(diagnostics, PlanValidationError{
					InstancePath: path,
					SchemaPath:   path,
					Code:         PlanCodeInvalidType,
					Keyword:      KeywordType,
					Message:      fmt.Sprintf("property %q must be a JSON object, got %T", field.JSONName, value),
				})
			}
		} else {
			diagnostics = append(diagnostics, validatePlanObject(*field.Children, value, contract, path)...)
		}
	case kindArray:
		diagnostics = append(diagnostics, validatePlanArray(field, path, value, contract)...)
	case kindString:
		if _, ok := value.(string); !ok {
			diagnostics = append(diagnostics, PlanValidationError{
				InstancePath: path,
				SchemaPath:   path,
				Code:         PlanCodeInvalidType,
				Keyword:      KeywordType,
				Message:      fmt.Sprintf("property %q must be a string, got %T", field.JSONName, value),
			})
		}
	case kindInteger:
		switch v := value.(type) {
		case json.Number:
			if _, err := strconv.ParseInt(string(v), 10, 64); err != nil {
				diagnostics = append(diagnostics, PlanValidationError{
					InstancePath: path,
					SchemaPath:   path,
					Code:         PlanCodeInvalidType,
					Keyword:      KeywordType,
					Message:      fmt.Sprintf("property %q must be an integer, got %q", field.JSONName, v.String()),
				})
			}
		case float64:
			// Tolerated by json.Number-less decoders (e.g. when
			// callers re-walk a map[string]any produced by the
			// standard library). The number must still be integral.
			if v != float64(int64(v)) {
				diagnostics = append(diagnostics, PlanValidationError{
					InstancePath: path,
					SchemaPath:   path,
					Code:         PlanCodeInvalidType,
					Keyword:      KeywordType,
					Message:      fmt.Sprintf("property %q must be an integer, got %v", field.JSONName, v),
				})
			}
		default:
			diagnostics = append(diagnostics, PlanValidationError{
				InstancePath: path,
				SchemaPath:   path,
				Code:         PlanCodeInvalidType,
				Keyword:      KeywordType,
				Message:      fmt.Sprintf("property %q must be an integer, got %T", field.JSONName, value),
			})
		}
		if field.ConstantValue != nil {
			if cv, ok := field.ConstantValue.(int); ok {
				actual, ok := integerValue(value)
				if ok && actual != cv {
					diagnostics = append(diagnostics, PlanValidationError{
						InstancePath:   path,
						SchemaPath:     path,
						Code:           PlanCodeInvalidEnum,
						Keyword:        KeywordConst,
						Message:        fmt.Sprintf("property %q must equal %d, got %d", field.JSONName, cv, actual),
						RejectedValue:  actual,
						AcceptedValues: []string{strconv.Itoa(cv)},
					})
				}
			}
		}
	case kindBoolean:
		if _, ok := value.(bool); !ok {
			diagnostics = append(diagnostics, PlanValidationError{
				InstancePath: path,
				SchemaPath:   path,
				Code:         PlanCodeInvalidType,
				Keyword:      KeywordType,
				Message:      fmt.Sprintf("property %q must be a boolean, got %T", field.JSONName, value),
			})
		} else {
			if field.ConstantValue != nil {
				if want, ok := field.ConstantValue.(bool); ok {
					got := value.(bool)
					if got != want {
						diagnostics = append(diagnostics, PlanValidationError{
							InstancePath:   path,
							SchemaPath:     path,
							Code:           PlanCodeInvalidEnum,
							Keyword:        KeywordConst,
							Message:        fmt.Sprintf("property %q must be %t, got %t", field.JSONName, want, got),
							RejectedValue:  got,
							AcceptedValues: []string{strconv.FormatBool(want)},
						})
					}
				}
			}
		}
	case kindEnum:
		str, ok := value.(string)
		if !ok {
			diagnostics = append(diagnostics, PlanValidationError{
				InstancePath: path,
				SchemaPath:   path,
				Code:         PlanCodeInvalidType,
				Keyword:      KeywordType,
				Message:      fmt.Sprintf("property %q must be a string, got %T", field.JSONName, value),
			})
		} else if !stringInSlice(field.EnumAuthority, str) {
			acceptedCopy := append([]string(nil), field.EnumAuthority...)
			diagnostics = append(diagnostics, PlanValidationError{
				InstancePath:   path,
				SchemaPath:     path,
				Code:           PlanCodeInvalidEnum,
				Keyword:        KeywordEnum,
				Message:        fmt.Sprintf("property %q value %q is not in %v", field.JSONName, str, acceptedCopy),
				RejectedValue:  str,
				AcceptedValues: acceptedCopy,
				PropertyName:   field.JSONName,
			})
		}
	}
	return diagnostics
}

// validatePlanArray walks an array against its descriptor.
func validatePlanArray(field planFieldDescriptor, path string, value any, contract planContractV1Descriptor) []PlanValidationError {
	var diagnostics []PlanValidationError
	asArray, ok := value.([]any)
	if !ok {
		diagnostics = append(diagnostics, PlanValidationError{
			InstancePath: path,
			SchemaPath:   path,
			Code:         PlanCodeInvalidType,
			Keyword:      KeywordType,
			Message:      fmt.Sprintf("property %q must be an array, got %T", field.JSONName, value),
		})
		return diagnostics
	}
	if field.MinItems > 0 && len(asArray) < field.MinItems {
		diagnostics = append(diagnostics, PlanValidationError{
			InstancePath: path,
			SchemaPath:   path,
			Code:         PlanCodeInvalidType,
			Keyword:      KeywordMinItems,
			Message:      fmt.Sprintf("property %q must contain at least %d item(s), got %d", field.JSONName, field.MinItems, len(asArray)),
		})
	}
	if field.ItemDescriptor == nil {
		return diagnostics
	}
	for index, item := range asArray {
		itemPath := path + "/" + strconv.Itoa(index)
		if field.ItemDescriptor.Children != nil {
			diagnostics = append(diagnostics, validatePlanObject(*field.ItemDescriptor.Children, item, contract, itemPath)...)
			continue
		}
		switch field.ItemDescriptor.Kind {
		case kindString:
			if _, ok := item.(string); !ok {
				diagnostics = append(diagnostics, PlanValidationError{
					InstancePath: itemPath,
					SchemaPath:   path,
					Code:         PlanCodeInvalidType,
					Keyword:      KeywordType,
					Message:      fmt.Sprintf("item at %s must be a string, got %T", itemPath, item),
				})
			}
		case kindInteger:
			if _, ok := item.(json.Number); !ok {
				if f, okF := item.(float64); !okF || f != float64(int64(f)) {
					diagnostics = append(diagnostics, PlanValidationError{
						InstancePath: itemPath,
						SchemaPath:   path,
						Code:         PlanCodeInvalidType,
						Keyword:      KeywordType,
						Message:      fmt.Sprintf("item at %s must be an integer, got %T", itemPath, item),
					})
				}
			}
		case kindEnum:
			if str, ok := item.(string); ok {
				if !stringInSlice(field.ItemDescriptor.EnumAuthority, str) {
					diagnostics = append(diagnostics, PlanValidationError{
						InstancePath:   itemPath,
						SchemaPath:     path,
						Code:           PlanCodeInvalidEnum,
						Keyword:        KeywordEnum,
						Message:        fmt.Sprintf("item at %s value %q is not in %v", itemPath, str, field.ItemDescriptor.EnumAuthority),
						RejectedValue:  str,
						AcceptedValues: append([]string(nil), field.ItemDescriptor.EnumAuthority...),
					})
				}
			}
		}
	}
	return diagnostics
}
