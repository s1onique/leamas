package closure

import (
	"regexp"
	"strings"
)

// plan_contract_validation_fields.go centralises the per-field and
// per-array walkers (validatePlanField, validatePlanArray,
// stringInSlice) used by validatePlanObject. Splitting it from
// plan_contract_validation.go keeps every file under the
// LLM-friendly 400-line threshold while preserving the single
// closure over the descriptor that
// ACT-LEAMAS-FACTORY-CLOSE-PLAN-CONTRACT-AUTHORITY01 requires.

// validatePlanField walks a single field against its descriptor.
// In addition to the JSON type/const/enum checks, it enforces the
// descriptor's value-level constraints (MinLength, Pattern,
// Minimum, Maximum) when those constraints are declared. The
// diagnostics use stable, constraint-specific codes
// (value_below_min_length, pattern_mismatch, numeric_below_minimum,
// numeric_above_maximum) so consumers can classify failures
// without parsing the message or using the generic invalid_type
// bucket.
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
			PropertyName:  field.JSONName,
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
		strVal, ok := value.(string)
		if !ok {
			diagnostics = append(diagnostics, PlanValidationError{
				InstancePath:  path,
				SchemaPath:    path,
				Code:          PlanCodeInvalidType,
				Keyword:       KeywordType,
				Message:       "property \"" + field.JSONName + "\" must be a string, got " + typeNameOf(value),
				RejectedValue: value,
				PropertyName:  field.JSONName,
			})
		} else {
			diagnostics = append(diagnostics, validateStringConstraints(field, path, strVal)...)
		}
	case kindInteger:
		// CORRECTION16: route every integer-shaped value
		// through the exact-number authority so 9223372036854775808
		// (int64_max+1) and 1e1000 classify as
		// numeric_above_maximum rather than invalid_type. The
		// authority's InRange decides below/above on the
		// rational form before any int conversion.
		iv, oversize, integral, ok := jsonNumberClassify(value)
		if !ok {
			diagnostics = append(diagnostics, PlanValidationError{
				InstancePath:  path,
				SchemaPath:    path,
				Code:          PlanCodeInvalidType,
				Keyword:       KeywordType,
				Message:       "property \"" + field.JSONName + "\" must be an integer, got " + typeNameOf(value),
				RejectedValue: value,
				PropertyName:  field.JSONName,
			})
		} else if !integral {
			// Mathematically non-integral; the schema's
			// type=integer branch rejects it on type. The
			// bound check would otherwise misclassify it
			// as numeric_below_minimum when the value is
			// between 0 and 1.
			diagnostics = append(diagnostics, PlanValidationError{
				InstancePath:  path,
				SchemaPath:    path,
				Code:          PlanCodeInvalidType,
				Keyword:       KeywordType,
				Message:       "property \"" + field.JSONName + "\" must be an integer, got " + typeNameOf(value),
				RejectedValue: value,
				PropertyName:  field.JSONName,
			})
		} else if oversize {
			// Mathematical integer that does not fit in int64.
			// Decide below/above maximum from the rational
			// form so the bound check fires before any int
			// coercion.
			diagnostics = append(diagnostics, validateOversizeInteger(field, path, value)...)
		} else {
			intVal := int(iv)
			diagnostics = append(diagnostics, validateIntegerConstraints(field, path, intVal)...)
			if field.ConstantValue != nil {
				if cv, ok := field.ConstantValue.(int); ok && intVal != cv {
					code := PlanCodeInvalidEnum
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
				PropertyName:  field.JSONName,
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
				PropertyName:  field.JSONName,
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

// validateStringConstraints enforces the descriptor's string
// value-level constraints (MinLength, Pattern). Diagnostics use the
// constraint-specific codes value_below_min_length and
// pattern_mismatch so a source-free consumer can distinguish
// length from pattern failures without parsing the message.
func validateStringConstraints(field planFieldDescriptor, path, value string) []PlanValidationError {
	var diagnostics []PlanValidationError
	if field.MinLength != nil && len(value) < *field.MinLength {
		diagnostics = append(diagnostics, PlanValidationError{
			InstancePath:  path,
			SchemaPath:    path,
			Code:          PlanCodeValueBelowMinLength,
			Keyword:       KeywordMinLength,
			Message:       "property \"" + field.JSONName + "\" length " + itoa(len(value)) + " is below minLength " + itoa(*field.MinLength),
			RejectedValue: value,
			PropertyName:  field.JSONName,
			AcceptedValues: []string{
				"length >= " + itoa(*field.MinLength),
			},
		})
		// Pattern is only checked when the length precondition
		// already passes; otherwise a value that fails both checks
		// produces two stacked diagnostics for the same field.
	} else if field.Pattern != "" {
		re, err := regexp.Compile(field.Pattern)
		if err != nil {
			diagnostics = append(diagnostics, PlanValidationError{
				InstancePath:  path,
				SchemaPath:    path,
				Code:          PlanCodeValuePatternMismatch,
				Keyword:       KeywordPattern,
				Message:       "property \"" + field.JSONName + "\" descriptor pattern is unparsable",
				RejectedValue: value,
				PropertyName:  field.JSONName,
			})
		} else if !re.MatchString(value) {
			diagnostics = append(diagnostics, PlanValidationError{
				InstancePath:  path,
				SchemaPath:    path,
				Code:          PlanCodeValuePatternMismatch,
				Keyword:       KeywordPattern,
				Message:       "property \"" + field.JSONName + "\" value does not match pattern",
				RejectedValue: value,
				PropertyName:  field.JSONName,
			})
		}
	}
	return diagnostics
}

// validateIntegerConstraints enforces the descriptor's integer
// value-level constraints (Minimum, Maximum). Diagnostics use the
// constraint-specific codes numeric_below_minimum and
// numeric_above_maximum so a correctly typed integer that falls
// outside the inclusive bounds is not misclassified as
// invalid_type. The exact supplied integer is preserved in
// rejected_value; the inclusive bounds are exposed via
// accepted_values in the canonical "[min, max]" form so a
// source-free consumer can correct the input.
func validateIntegerConstraints(field planFieldDescriptor, path string, value int) []PlanValidationError {
	var diagnostics []PlanValidationError
	if field.Minimum != nil && int64(value) < *field.Minimum {
		diagnostics = append(diagnostics, PlanValidationError{
			InstancePath:   path,
			SchemaPath:     path,
			Code:           PlanCodeNumericBelowMinimum,
			Keyword:        KeywordMinimum,
			Message:        "property \"" + field.JSONName + "\" value " + itoa(value) + " is below minimum " + itoa64(*field.Minimum),
			RejectedValue:  value,
			PropertyName:   field.JSONName,
			AcceptedValues: integerBoundRange(field),
		})
		return diagnostics
	}
	if field.Maximum != nil && int64(value) > *field.Maximum {
		diagnostics = append(diagnostics, PlanValidationError{
			InstancePath:   path,
			SchemaPath:     path,
			Code:           PlanCodeNumericAboveMaximum,
			Keyword:        KeywordMaximum,
			Message:        "property \"" + field.JSONName + "\" value " + itoa(value) + " exceeds maximum " + itoa64(*field.Maximum),
			RejectedValue:  value,
			PropertyName:   field.JSONName,
			AcceptedValues: integerBoundRange(field),
		})
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
