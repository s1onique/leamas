package closure

import (
	"sort"
	"strings"
)

// plan_contract_descriptor_validation.go centralises the
// descriptor-inventory validators the walker and example
// generator consume before processing applicability rules. The
// validators run at consumer entry so a malformed descriptor
// cannot silently produce ambiguous walker behaviour.

// descriptorValidationObserver receives invocation-local
// events from the descriptor applicability-identity validator.
// Production callers pass a noop; tests pass a counting
// observer to assert the validator runs exactly once per
// applicability invocation.
type descriptorValidationObserver interface {
	DescriptorIdentityValidated(diagnostics []PlanValidationError)
}

// noopDescriptorValidationObserver satisfies
// descriptorValidationObserver without side effects.
type noopDescriptorValidationObserver struct{}

func (noopDescriptorValidationObserver) DescriptorIdentityValidated(diagnostics []PlanValidationError) {
}

// validateDescriptorApplicabilityIdentity walks every
// descriptor object and emits a deterministic duplicate
// diagnostic for each (Sibling, Value) pair that appears more
// than once in a single field's ApplicabilityRules slice. Two
// rules sharing (Sibling, Value) are ambiguous: a presence
// Required and a presence Forbidden for the same condition would
// yield mutually-exclusive diagnostics. The validator pins the
// "one rule per (field, Sibling, Value)" invariant so the walker
// and the example generator agree on the contract.
//
// Iteration order: object.Fields is a Go map; iteration is
// non-deterministic. The validator walks the sorted field-name
// list so duplicate diagnostics are produced in the same
// order on every invocation. The collected diagnostics are
// finally sorted by (InstancePath, Code, PropertyName,
// Message) so the walker's output is reproducible.
func validateDescriptorApplicabilityIdentity(contract planContractV1Descriptor) []PlanValidationError {
	var diagnostics []PlanValidationError
	diagnostics = append(diagnostics, validateObjectApplicabilityIdentity(contract.Root, "")...)
	sortDescriptorDiagnostics(diagnostics)
	return diagnostics
}

// sortDescriptorDiagnostics orders the diagnostics by
// (InstancePath, Code, PropertyName, Message) so consumers see
// the same order on every invocation.
func sortDescriptorDiagnostics(diagnostics []PlanValidationError) {
	sort.SliceStable(diagnostics, func(i, j int) bool {
		if diagnostics[i].InstancePath != diagnostics[j].InstancePath {
			return diagnostics[i].InstancePath < diagnostics[j].InstancePath
		}
		if diagnostics[i].Code != diagnostics[j].Code {
			return diagnostics[i].Code < diagnostics[j].Code
		}
		if diagnostics[i].PropertyName != diagnostics[j].PropertyName {
			return diagnostics[i].PropertyName < diagnostics[j].PropertyName
		}
		return diagnostics[i].Message < diagnostics[j].Message
	})
}

// validateObjectApplicabilityIdentity recurses into nested
// object descriptors and array item descriptors. Each field is
// validated exactly once against its declared rules. The
// iteration order follows object.fieldNamesSorted() so the
// traversal is deterministic.
func validateObjectApplicabilityIdentity(object planObjectDescriptor, parentPath string) []PlanValidationError {
	var diagnostics []PlanValidationError
	for _, name := range object.fieldNamesSorted() {
		field := object.Fields[name]
		path := canonicalJSONPointer(parentPath, name)
		diagnostics = append(diagnostics, validateFieldApplicabilityIdentity(field, path)...)
		if field.Children != nil {
			diagnostics = append(diagnostics, validateObjectApplicabilityIdentity(*field.Children, path)...)
		}
		if field.ItemDescriptor != nil && field.ItemDescriptor.Children != nil {
			childPath := path
			if childPath == "" {
				childPath = object.Path
			}
			diagnostics = append(diagnostics, validateObjectApplicabilityIdentity(*field.ItemDescriptor.Children, childPath)...)
		}
	}
	return diagnostics
}

// validateFieldApplicabilityIdentity reports every (Sibling,
// Value) pair that appears more than once in the field's
// ApplicabilityRules slice. Identical and conflicting duplicates
// are both rejected; the validator does not care which.
func validateFieldApplicabilityIdentity(field planFieldDescriptor, fieldPath string) []PlanValidationError {
	if len(field.ApplicabilityRules) <= 1 {
		return nil
	}
	seen := make(map[string]bool, len(field.ApplicabilityRules))
	var diagnostics []PlanValidationError
	for _, rule := range field.ApplicabilityRules {
		key := rule.Sibling + "=" + rule.Value
		if seen[key] {
			diagnostics = append(diagnostics, PlanValidationError{
				InstancePath: fieldPath,
				SchemaPath:   fieldPath,
				Code:         PlanCodeDuplicateApplicabilityRule,
				Keyword:      KeywordIfThenElse,
				Message:      duplicateApplicabilityRuleMessage(field.JSONName, rule.Sibling, rule.Value),
				PropertyName: field.JSONName,
			})
			continue
		}
		seen[key] = true
	}
	return diagnostics
}

// duplicateApplicabilityRuleMessage builds the stable diagnostic
// message. The triple (field, Sibling, Value) is the unique key
// that identifies the duplicate so consumers can locate it in
// the descriptor inventory without parsing the message.
func duplicateApplicabilityRuleMessage(fieldName, sibling, value string) string {
	var b strings.Builder
	b.WriteString("duplicate applicability rule for field \"")
	b.WriteString(fieldName)
	b.WriteString("\": Sibling=\"")
	b.WriteString(sibling)
	b.WriteString("\", Value=\"")
	b.WriteString(value)
	b.WriteString("\"")
	return b.String()
}

// duplicateApplicabilityFieldPaths returns the set of field
// paths whose applicability rules carry duplicate (Sibling,
// Value) entries. The keys are complete descriptor paths so
// the walker can suppress a single subtree without confusing
// paths in other subtrees.
//
// The function consumes the result of
// validateDescriptorApplicabilityIdentity so the validator runs
// once per caller; the same diagnostic stream drives the
// walker's reported diagnostics and the suppress set.
func duplicateApplicabilityFieldPaths(identityDiagnostics []PlanValidationError) map[string]struct{} {
	paths := make(map[string]struct{})
	for _, diag := range identityDiagnostics {
		if diag.Code != PlanCodeDuplicateApplicabilityRule {
			continue
		}
		if diag.InstancePath == "" {
			continue
		}
		paths[diag.InstancePath] = struct{}{}
	}
	return paths
}
