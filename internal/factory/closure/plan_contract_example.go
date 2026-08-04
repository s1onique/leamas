package closure

import "strings"

// DescriptorExample returns a canonical example closure-plan v1
// document whose every field is emitted by walking the descriptor.
// The example is the single source for the future CLI's
// `factory schema example` command and is mechanically tested by the
// parity suite.
//
// Mode-dependent rules force the example to emit a "run" check so
// argv, working_directory, timeout_seconds, and environment are all
// present; environment contains a string entry to demonstrate the
// free-form string-map shape. argv items are real strings so the
// future schema generator can emit items: { type: "string" }.
//
// Every required field is emitted, every pointer-backed policy
// boolean is set to true (matching the constant value constraint),
// and the example passes both structural and semantic validation.
//
// DescriptorExample delegates to descriptorExampleWithContract
// using the production descriptor. The descriptor identity
// validator runs before any example is generated; a malformed
// production descriptor would yield an empty map. The
// production descriptor is currently valid so the existing
// example output is unchanged.
func DescriptorExample() map[string]any {
	example, _ := descriptorExampleWithContract(planContractV1())
	return example
}

// descriptorExampleWithContract is the result-bearing internal
// example path the future CLI and tests consume. The descriptor
// applicability-identity validator runs before any example is
// generated; a malformed descriptor receives an empty map plus
// the validator's diagnostic stream and never a partial example.
//
// Returned diagnostics are sorted deterministically. The
// example returned for a valid production descriptor matches
// the original DescriptorExample output byte-for-byte.
func descriptorExampleWithContract(contract planContractV1Descriptor) (map[string]any, []PlanValidationError) {
	identityDiagnostics := validateDescriptorApplicabilityIdentity(contract)
	if hasDuplicateApplicabilityDiagnostic(identityDiagnostics) {
		return map[string]any{}, identityDiagnostics
	}
	example := buildExampleObject(contract.Root)
	out, ok := example.(map[string]any)
	if !ok {
		return map[string]any{}, identityDiagnostics
	}
	return out, identityDiagnostics
}

// hasDuplicateApplicabilityDiagnostic reports whether the
// diagnostic stream carries any duplicate_applicability_rule
// diagnostic. A malformed descriptor cannot reach the example
// generator; this is the gate that enforces the contract.
func hasDuplicateApplicabilityDiagnostic(diagnostics []PlanValidationError) bool {
	for _, diag := range diagnostics {
		if diag.Code == PlanCodeDuplicateApplicabilityRule {
			return true
		}
	}
	return false
}

// buildExampleObject walks the descriptor and emits a JSON-shaped
// value that matches every required field, constant, and enum.
func buildExampleObject(object planObjectDescriptor) any {
	switch object.Kind {
	case objectClosed:
		return buildClosedExampleObject(object)
	case objectStringMap:
		return map[string]any{"EXAMPLE_KEY": "example_value"}
	default:
		return buildClosedExampleObject(object)
	}
}

// buildClosedExampleObject emits a JSON object whose keys are the
// union of required and non-required-but-known fields. Children,
// arrays, enums, and constants are emitted so every descriptor
// constraint has a visible example.
//
// Special case: when the parent path is /checks/[] the example emits
// a coherent run-mode check (argv, working_directory, timeout_seconds,
// environment present; reason absent). The mode-dependent
// applicability of every field is honoured.
func buildClosedExampleObject(object planObjectDescriptor) map[string]any {
	out := map[string]any{}
	isCheckItem := object.Path == "/checks" || strings.HasPrefix(object.Path, "/checks/")
	// Required fields first, in descriptor order.
	for _, name := range object.Required {
		field, ok := object.Fields[name]
		if !ok {
			continue
		}
		if isCheckItem && skipForCheckItemRunMode(name, field) {
			continue
		}
		if v := buildExampleField(field); !isExampleSkip(v) {
			out[name] = v
		}
	}
	// Then optional fields, in lexicographic order.
	for _, name := range object.fieldNamesSorted() {
		if _, already := out[name]; already {
			continue
		}
		if object.Fields[name].Required {
			continue
		}
		if isCheckItem && skipForCheckItemRunMode(name, object.Fields[name]) {
			continue
		}
		if v := buildExampleField(object.Fields[name]); !isExampleSkip(v) {
			out[name] = v
		}
	}
	return out
}

// exampleSkip is the sentinel returned by buildExampleField when a
// field should be omitted from the example entirely (typically
// because including it would force the example to satisfy a
// policy-profile or runner-binding semantic rule).
type exampleSkip struct{}

func isExampleSkip(v any) bool {
	_, ok := v.(exampleSkip)
	return ok
}

// skipForCheckItemRunMode reports whether a check-item field should
// be omitted from the canonical run-mode example. The example is
// always a "run" check; any field whose applicability says it is
// only required when mode=exclude is skipped.
func skipForCheckItemRunMode(name string, field planFieldDescriptor) bool {
	for _, rule := range field.ApplicabilityRules {
		if rule.Value == CheckModeExclude && rule.Presence == PresenceRequired {
			return true
		}
	}
	return false
}

// buildExampleField returns the example value for a single field.
// Mode-dependent rules force the example to emit a runnable check so
// argv, working_directory, timeout_seconds, and environment are all
// present.
func buildExampleField(field planFieldDescriptor) any {
	// Skip optional policy-enforcing fields: they trigger additional
	// semantic checks (policy profile, runner binding) that the
	// canonical example cannot satisfy without becoming a copy of
	// a real leamas-act-v1 plan. The example builder omits these
	// keys entirely; the sentinel below lets buildClosedExampleObject
	// detect the request.
	if !field.Required && (field.JSONName == "policy_profile" || field.JSONName == "runner_binding" || field.JSONName == "runner_authority") {
		return exampleSkip{}
	}
	// Apply mode-dependent applicability first: a check item must
	// emit a runnable example so argv, environment, etc. are
	// demonstrated. Only one applicability per field is supported
	// (the v1 wire shape).
	if applicabilityPresenceForMode(field, "run") == PresenceRequired {
		return buildApplicabilityExample(field)
	}
	if field.Kind == kindObject && field.Children != nil {
		return buildExampleObject(*field.Children)
	}
	if field.Kind == kindArray {
		return buildExampleArray(field)
	}
	if field.ConstantValue != nil {
		return field.ConstantValue
	}
	if len(field.EnumAuthority) == 1 {
		return field.EnumAuthority[0]
	}
	if field.ExampleValue != nil {
		return field.ExampleValue
	}
	switch field.Kind {
	case kindString:
		return ""
	case kindInteger:
		return 0
	case kindBoolean:
		return false
	case kindEnum:
		if len(field.EnumAuthority) > 0 {
			return field.EnumAuthority[0]
		}
		return ""
	default:
		return nil
	}
}

// buildApplicabilityExample is invoked for fields whose presence is
// forced by mode-dependent applicability. The example emits a run
// check so argv, environment, and the other run-only fields are
// demonstrated. The check object is built by walking the parent
// descriptor and replacing the mode-dependent fields with their
// run-mode examples.
func buildApplicabilityExample(field planFieldDescriptor) any {
	if field.JSONName == "argv" {
		return []any{"sh", "-c", "exit 0"}
	}
	if field.JSONName == "working_directory" {
		return "."
	}
	if field.JSONName == "timeout_seconds" {
		return 60
	}
	if field.JSONName == "environment" {
		return map[string]any{"EXAMPLE_KEY": "example_value"}
	}
	if field.ExampleValue != nil {
		return field.ExampleValue
	}
	return ""
}

// buildExampleArray returns the example array for an array-typed
// field. Arrays of objects emit one item; arrays of primitives emit
// a single string example.
func buildExampleArray(field planFieldDescriptor) []any {
	if field.ItemDescriptor != nil && field.ItemDescriptor.Children != nil {
		return []any{buildExampleObject(*field.ItemDescriptor.Children)}
	}
	if field.ItemDescriptor != nil {
		switch field.ItemDescriptor.Kind {
		case kindString:
			return []any{"example"}
		case kindInteger:
			return []any{0}
		case kindEnum:
			if len(field.ItemDescriptor.EnumAuthority) > 0 {
				return []any{field.ItemDescriptor.EnumAuthority[0]}
			}
			return []any{"example"}
		}
	}
	return []any{}
}
