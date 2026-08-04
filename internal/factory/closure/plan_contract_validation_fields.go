package closure

import "strings"

// plan_contract_validation_fields.go centralises the per-field and
// per-array walkers (validatePlanField, validatePlanArray,
// stringInSlice) used by validatePlanObject. Splitting it from
// plan_contract_validation.go keeps every file under the
// LLM-friendly 400-line threshold while preserving the single
// closure over the descriptor that
// ACT-LEAMAS-FACTORY-CLOSE-PLAN-CONTRACT-AUTHORITY01 requires.

// validatePlanField walks a single field against its descriptor.
func validatePlanField(field planFieldDescriptor, path string, value any, parent planObjectDescriptor, _ map[string]any, contract planContractV1Descriptor) []PlanValidationError {
	var diagnostics []PlanValidationError
	if value == nil {
		if field.Nullable {
			return diagnostics
		}
		diagnostics = append(diagnostics, PlanValidationError{
			InstancePath:  path,
			SchemaPath:    path,
			Code:          PlanCodeInvalidType,
			Keyword:       KeywordType,
			Message:       "property \"" + field.JSONName + "\" must not be null",
			RejectedValue: nil,
		})
		return diagnostics
	}
	switch field.Kind {
	case kindObject:
		if field.Children == nil {
			if _, ok := value.(map[string]any); !ok {
				diagnostics = append(diagnostics, PlanValidationError{
					InstancePath: path,
					SchemaPath:   path,
					Code:         PlanCodeInvalidType,
					Keyword:      KeywordType,
					Message:      "property \"" + field.JSONName + "\" must be a JSON object, got " + typeNameOf(value),
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
				InstancePath:  path,
				SchemaPath:    path,
				Code:          PlanCodeInvalidType,
				Keyword:       KeywordType,
				Message:       "property \"" + field.JSONName + "\" must be a string, got " + typeNameOf(value),
				RejectedValue: value,
			})
		}
	case kindInteger:
		intVal, ok := jsonNumberToInteger(value)
		if !ok {
			diagnostics = append(diagnostics, PlanValidationError{
				InstancePath:  path,
				SchemaPath:    path,
				Code:          PlanCodeInvalidType,
				Keyword:       KeywordType,
				Message:       "property \"" + field.JSONName + "\" must be an integer, got " + typeNameOf(value),
				RejectedValue: value,
			})
		} else if field.ConstantValue != nil {
			if cv, ok := field.ConstantValue.(int); ok && intVal != cv {
				code := PlanCodeInvalidEnum
				// Use the documented "unsupported_contract_version"
				// code when the field is /contract_version.
				if field.JSONName == "contract_version" {
					code = PlanCodeUnsupportedContractVersion
				}
				diagnostics = append(diagnostics, PlanValidationError{
					InstancePath:   path,
					SchemaPath:     path,
					Code:           code,
					Keyword:        KeywordConst,
					Message:        "property \"" + field.JSONName + "\" must equal " + itoa(cv) + ", got " + itoa(intVal),
					RejectedValue:  intVal,
					AcceptedValues: []string{itoa(cv)},
				})
			}
		}
	case kindBoolean:
		if _, ok := value.(bool); !ok {
			diagnostics = append(diagnostics, PlanValidationError{
				InstancePath:  path,
				SchemaPath:    path,
				Code:          PlanCodeInvalidType,
				Keyword:       KeywordType,
				Message:       "property \"" + field.JSONName + "\" must be a boolean, got " + typeNameOf(value),
				RejectedValue: value,
			})
		} else if field.ConstantValue != nil {
			if want, ok := field.ConstantValue.(bool); ok && value.(bool) != want {
				diagnostics = append(diagnostics, PlanValidationError{
					InstancePath:   path,
					SchemaPath:     path,
					Code:           PlanCodeInvalidEnum,
					Keyword:        KeywordConst,
					Message:        "property \"" + field.JSONName + "\" must be " + boolStr(want) + ", got " + boolStr(value.(bool)),
					RejectedValue:  value.(bool),
					AcceptedValues: []string{boolStr(want)},
				})
			}
		}
	case kindEnum:
		str, ok := value.(string)
		if !ok {
			diagnostics = append(diagnostics, PlanValidationError{
				InstancePath:  path,
				SchemaPath:    path,
				Code:          PlanCodeInvalidType,
				Keyword:       KeywordType,
				Message:       "property \"" + field.JSONName + "\" must be a string, got " + typeNameOf(value),
				RejectedValue: value,
			})
		} else if !stringInSlice(field.EnumAuthority, str) {
			acceptedCopy := append([]string(nil), field.EnumAuthority...)
			diagnostics = append(diagnostics, PlanValidationError{
				InstancePath:   path,
				SchemaPath:     path,
				Code:           PlanCodeInvalidEnum,
				Keyword:        KeywordEnum,
				Message:        "property \"" + field.JSONName + "\" value \"" + str + "\" is not in " + strings.Join(acceptedCopy, ","),
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
			InstancePath:  path,
			SchemaPath:    path,
			Code:          PlanCodeInvalidType,
			Keyword:       KeywordType,
			Message:       "property \"" + field.JSONName + "\" must be an array, got " + typeNameOf(value),
			RejectedValue: value,
		})
		return diagnostics
	}
	if field.MinItems > 0 && len(asArray) < field.MinItems {
		diagnostics = append(diagnostics, PlanValidationError{
			InstancePath: path,
			SchemaPath:   path,
			Code:         PlanCodeInvalidType,
			Keyword:      KeywordMinItems,
			Message:      "property \"" + field.JSONName + "\" must contain at least " + itoa(field.MinItems) + " item(s), got " + itoa(len(asArray)),
		})
	}
	if field.ItemDescriptor == nil {
		return diagnostics
	}
	for index, item := range asArray {
		itemPath := path + "/" + itoa(index)
		if field.ItemDescriptor.Children != nil {
			diagnostics = append(diagnostics, validatePlanObject(*field.ItemDescriptor.Children, item, contract, itemPath)...)
			continue
		}
		switch field.ItemDescriptor.Kind {
		case kindString:
			if _, ok := item.(string); !ok {
				diagnostics = append(diagnostics, PlanValidationError{
					InstancePath:  itemPath,
					SchemaPath:    path,
					Code:          PlanCodeInvalidType,
					Keyword:       KeywordType,
					Message:       "item at " + itemPath + " must be a string, got " + typeNameOf(item),
					RejectedValue: item,
				})
			}
		case kindInteger:
			if _, ok := jsonNumberToInteger(item); !ok {
				diagnostics = append(diagnostics, PlanValidationError{
					InstancePath:  itemPath,
					SchemaPath:    path,
					Code:          PlanCodeInvalidType,
					Keyword:       KeywordType,
					Message:       "item at " + itemPath + " must be an integer, got " + typeNameOf(item),
					RejectedValue: item,
				})
			}
		case kindEnum:
			if str, ok := item.(string); ok {
				if !stringInSlice(field.ItemDescriptor.EnumAuthority, str) {
					diagnostics = append(diagnostics, PlanValidationError{
						InstancePath:   itemPath,
						SchemaPath:     path,
						Code:           PlanCodeInvalidEnum,
						Keyword:        KeywordEnum,
						Message:        "item at " + itemPath + " value \"" + str + "\" is not in " + strings.Join(field.ItemDescriptor.EnumAuthority, ","),
						RejectedValue:  str,
						AcceptedValues: append([]string(nil), field.ItemDescriptor.EnumAuthority...),
					})
				}
			}
		}
	}
	return diagnostics
}

// stringInSlice reports whether list contains the literal value.
func stringInSlice(list []string, value string) bool {
	for _, candidate := range list {
		if candidate == value {
			return true
		}
	}
	return false
}
