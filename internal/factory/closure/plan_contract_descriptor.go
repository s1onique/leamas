package closure

import (
	"sort"
	"strings"
)

// planContractV1Descriptor is the private, immutable declaration of
// the Closure Protocol v1 plan wire contract. The descriptor is the
// authoritative source consumed by:
//
//   - structural validation (PlanValidationResult);
//   - typed decoding (the schema, the Go Plan struct);
//   - semantic validation (validatePlan* helpers);
//   - future schema generation (schema/example CLI commands);
//   - future example generation (schema/example CLI commands);
//   - diagnostics (instance paths, enum authority, required sets).
//
// The descriptor is private because it is a Factory-internal
// declaration. Consumers outside the closure package see only the
// public validation result types and the public ParseExecutionMode
// API. The descriptor contains:
//
//   - one explicit contract version (ContractVersionV1);
//   - deterministic field order (lexicographic within each container);
//   - deterministic enum order (matches runtime supportedExecutionModes
//     and the ArtifactRole / RunnerAuthorityMode / CheckMode constants);
//   - per-field: JSON name, Go name, type category, required status,
//     defaulting annotation, enum authority, semantic rule identifier,
//     description, and example value.
//
// The descriptor intentionally has NO package-global mutable state.
// It is built once by planContractV1() through pure constructors.
//
// # Design rationale
//
// Closure Protocol v1 has been the only supported version for the
// entire lifetime of the package. Several ACTs (Closure Protocol v1
// adoption, plan execution-mode reconciliation, schema introspection,
// policy profile authority) have already pinned the wire shape
// through fixture tables and JSON Schema parity tests. The descriptor
// turns that implicit agreement into one reviewable declaration that
// every future validator (schema/example CLI commands, structural
// JSON-pointer diagnostics) can read directly.
type planContractV1Descriptor struct {
	// ContractVersion is the explicit numeric version the runtime
	// dispatcher accepts. Plan.ContractVersion must equal this
	// constant.
	ContractVersion int

	// Root describes the top-level plan object.
	Root planObjectDescriptor

	// TopLevelAliases are JSON names that LOOK like plan keys but
	// MUST be rejected as unknown_property diagnostics. Keeping the
	// list explicit guards against silent aliasing; aliases must
	// move to the Root.Required or Root.Optional lists to be
	// accepted.
	TopLevelAliases []string

	// AliasSubpaths names the locations where aliases have been
	// historically observed (for example `policy.mode`,
	// `policy.execution`, `policy.execution_mode`,
	// `runner_authority.tool_release_exact_v1`). The structural
	// validator reports a precise unknown_property code for each.
	AliasSubpaths []string

	// HistoricalRejectedModes is the ordered set of literal strings
	// that producers have historically tried to pass as execution
	// modes (for example `exitcode`, `gate`). The set is
	// authoritative: every value here is rejected with an
	// invalid_enum diagnostic. New values MUST be appended here and
	// only here.
	HistoricalRejectedModes []string
}

// planObjectDescriptor declares the shape of a single JSON object:
// the ordered set of accepted field names, whether each field is
// required, and the field descriptors that pin the type and enum
// authority.
type planObjectDescriptor struct {
	// Path is the canonical JSON pointer to the object (empty for
	// the root).
	Path string

	// Required is the ordered list of required field names.
	Required []string

	// Fields is the ordered map from JSON field name to descriptor.
	Fields map[string]planFieldDescriptor
}

// planFieldDescriptor declares the shape of a single field: type
// category, pointer-vs-value presence semantics, defaulting rule,
// enum authority, semantic rule identifier, description, and
// example value. The descriptor never references ClineMM-derived
// aliases.
type planFieldDescriptor struct {
	// JSONName is the canonical lowercase_underscore JSON name.
	JSONName string

	// GoName is the PascalCase Go field name.
	GoName string

	// Kind classifies the JSON type category. One of:
	//   kindObject, kindArray, kindString, kindInteger,
	//   kindBoolean, kindEnum.
	Kind planFieldKind

	// Required reports whether the field MUST appear in a valid
	// plan.
	Required bool

	// Pointer reports whether the Go field is a pointer so that
	// absent vs present-empty can be distinguished. The structural
	// validator treats Pointer fields as having three presence
	// categories (absent, present-null, present-value).
	Pointer bool

	// ConstantValue, when non-nil, names the only accepted value.
	// For example contract_version is pinned to 1; policy booleans
	// are pinned to true.
	ConstantValue any

	// EnumAuthority is the closed, ordered list of accepted string
	// values. Empty for non-string-enum kinds.
	EnumAuthority []string

	// SemanticRule names the Go function or constant that
	// authoritative semantic validation delegates to. The string is
	// for diagnostics only; the rule itself lives in the typed
	// validator.
	SemanticRule string

	// DefaultingRule describes how the field is filled when the
	// producer omits it. The descriptor never mutates a
	// package-global; the rule is documentation plus a helper
	// (DefaultValueFor).
	DefaultingRule string

	// Description is the human-readable description of the field.
	Description string

	// ExampleValue is a literal example suitable for documentation
	// and the future example CLI command.
	ExampleValue any

	// Children is the ordered object descriptor for nested objects.
	// Nil for non-object kinds.
	Children *planObjectDescriptor

	// ItemDescriptor, when non-nil, describes the items of an array.
	// Nil for non-array kinds.
	ItemDescriptor *planFieldDescriptor

	// MinItems is the inclusive minimum number of array items. Zero
	// means no minimum. Ignored for non-array kinds.
	MinItems int

	// ModeDependent lists the field names on the parent object whose
	// presence or value decides whether THIS field applies. The
	// descriptor is the single source of mode-dependent applicability
	// (for example: `argv` applies when the sibling `mode` is "run";
	// `reason` applies when the sibling `mode` is "exclude").
	// Empty for unconditional fields.
	ModeDependent []string

	// RejectedAliases are JSON names that LOOK like this field but
	// MUST be rejected. The descriptor captures the rejection
	// contract that historical ACTs already pinned.
	RejectedAliases []string
}

// planFieldKind enumerates the JSON type categories the structural
// validator distinguishes. Adding a new kind is a breaking change
// and must append here and only here.
type planFieldKind int

const (
	kindUnknown planFieldKind = iota
	kindObject
	kindArray
	kindString
	kindInteger
	kindBoolean
	kindEnum
)

// String renders the kind as a stable lowercase label suitable for
// diagnostics and parity assertions.
func (k planFieldKind) String() string {
	switch k {
	case kindObject:
		return "object"
	case kindArray:
		return "array"
	case kindString:
		return "string"
	case kindInteger:
		return "integer"
	case kindBoolean:
		return "boolean"
	case kindEnum:
		return "enum"
	}
	return "unknown"
}

// planContractV1 is the single source of truth for Closure Protocol
// v1. planContractV1() returns a fresh, fully resolved copy on every
// call so the descriptor cannot be mutated by callers. The
// underlying constructors are pure functions.
func planContractV1() planContractV1Descriptor {
	descriptor := planContractV1Descriptor{
		ContractVersion: ContractVersionV1,
		TopLevelAliases: []string{
			"mode",
			"contractVersion",
			"closure_plan",
		},
		AliasSubpaths: []string{
			"/policy/mode",
			"/policy/execution",
			"/policy/execution_mode",
			"/runner_authority/tool_release_exact_v1",
		},
		HistoricalRejectedModes: []string{
			"exitcode",
			"gate",
			"parallel",
			"serial",
			"fail_fast",
			"exit_code",
		},
	}
	descriptor.Root = buildPlanRootV1()
	return descriptor
}

// fieldNamesSorted returns the JSON field names of an object in
// lexicographic order. The structural validator iterates in this
// order so the diagnostic stream is deterministic.
func (o planObjectDescriptor) fieldNamesSorted() []string {
	names := make([]string, 0, len(o.Fields))
	for name := range o.Fields {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// hasField reports whether the object declares the given field by
// JSON name. Used by the structural validator to decide between
// unknown_property and required_property_missing.
func (o planObjectDescriptor) hasField(jsonName string) bool {
	_, ok := o.Fields[jsonName]
	return ok
}

// canonicalJSONPointer is a tiny helper that returns the
// well-formed JSON pointer to a field. It refuses empty names so the
// pointer for the root object is the empty string. The pointer
// syntax matches RFC 6901: a leading slash, segments separated by
// slash, with no escaping required for the v1 wire names.
func canonicalJSONPointer(parent, name string) string {
	if name == "" {
		return parent
	}
	if parent == "" {
		return "/" + name
	}
	return parent + "/" + name
}

// isClineMMDerived reports whether the supplied identifier has the
// shape of a ClineMM-derived alias. The descriptor policy is "no
// ClineMM-derived aliases": future descriptor additions that match
// this heuristic must be approved by a separate ACT. Today no
// descriptor field name matches the heuristic, but the helper is
// available for future audits.
func isClineMMDerived(identifier string) bool {
	lower := strings.ToLower(identifier)
	return strings.HasPrefix(lower, "clinemm_") ||
		strings.Contains(lower, "_clinemm") ||
		strings.HasPrefix(lower, "cline_mm_") ||
		strings.Contains(lower, "_cline_mm_")
}

// descriptorIsClean reports whether the descriptor contains no
// ClineMM-derived JSON aliases. The check walks every JSONName and
// RejectedAliases entry recursively.
func descriptorIsClean(d planContractV1Descriptor) bool {
	for _, alias := range d.TopLevelAliases {
		if isClineMMDerived(alias) {
			return false
		}
	}
	for _, name := range d.AliasSubpaths {
		if isClineMMDerived(name) {
			return false
		}
	}
	for _, mode := range d.HistoricalRejectedModes {
		if isClineMMDerived(mode) {
			return false
		}
	}
	return walkDescriptorClean(d.Root)
}

func walkDescriptorClean(o planObjectDescriptor) bool {
	for name, field := range o.Fields {
		if isClineMMDerived(name) || isClineMMDerived(field.GoName) {
			return false
		}
		for _, alias := range field.RejectedAliases {
			if isClineMMDerived(alias) {
				return false
			}
		}
		if field.Children != nil && !walkDescriptorClean(*field.Children) {
			return false
		}
		if field.ItemDescriptor != nil && field.ItemDescriptor.Children != nil &&
			!walkDescriptorClean(*field.ItemDescriptor.Children) {
			return false
		}
	}
	return true
}
