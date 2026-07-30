package closure

// plan_contract_descriptor_types.go centralises the descriptor's
// type declarations (structs, enums, and their String methods).
// Splitting them from plan_contract_descriptor.go keeps the root
// builder and helper functions in the main descriptor file under
// the LLM-friendly 400-line threshold while every type remains
// reviewable in one screen.

// planObjectKind is the wire-level shape of a JSON object the
// descriptor recognises. Closed objects reject unknown properties;
// free-form string maps accept any property name and require each
// value to be a JSON string.
type planObjectKind int

const (
	// objectClosed is the default: every property must be declared
	// in the descriptor's Fields map and unknown_property is
	// rejected.
	objectClosed planObjectKind = iota

	// objectStringMap describes a JSON Schema
	// `additionalProperties: { "type": "string" }` object.
	objectStringMap
)

// String renders the kind as a stable lowercase label.
func (k planObjectKind) String() string {
	switch k {
	case objectClosed:
		return "closed"
	case objectStringMap:
		return "string_map"
	}
	return "unknown"
}

// planObjectDescriptor declares the shape of a single JSON object.
type planObjectDescriptor struct {
	// Path is the canonical JSON pointer to the object (empty for
	// the root).
	Path string

	// Kind is the wire-level object shape. The default (zero value)
	// is objectClosed; the descriptor never relies on map-default.
	Kind planObjectKind

	// Required is the ordered list of required field names.
	Required []string

	// Fields is the ordered map from JSON field name to descriptor.
	Fields map[string]planFieldDescriptor
}

// PresenceRule is the documented conditional-presence outcome a
// fieldApplicabilityRule applies when its sibling equals its Value.
type PresenceRule int

const (
	// PresenceOptional is the default absence-of-rule: the field
	// has no constraint beyond what the descriptor already says.
	PresenceOptional PresenceRule = iota
	// PresenceRequired means the field MUST be present.
	PresenceRequired
	// PresenceForbidden means the field MUST NOT be present.
	PresenceForbidden
)

// String renders the presence rule as a stable lowercase label.
func (p PresenceRule) String() string {
	switch p {
	case PresenceRequired:
		return "required"
	case PresenceForbidden:
		return "forbidden"
	}
	return "optional"
}

// fieldApplicabilityRule is one (sibling, value, presence) rule. A
// field may carry multiple rules so both branches of the same
// condition (for example mode=run and mode=exclude) can be encoded
// without inferring inverse behavior.
type fieldApplicabilityRule struct {
	// Sibling is the JSON name of the sibling field whose value
	// decides applicability (for example "mode" on /checks[]).
	Sibling string
	// Value is the literal sibling value that activates this rule.
	Value string
	// Presence is the documented outcome.
	Presence PresenceRule
}

// field whose presence or value depends on a sibling field's value.
// The single Applicability rule remains for backward compatibility
// with the previous descriptor surface; new code should add
// ApplicabilityRules so both branches are encoded explicitly.

// planFieldDescriptor declares the shape of a single field.
type planFieldDescriptor struct {
	JSONName string
	GoName   string

	// Kind classifies the JSON type category.
	Kind planFieldKind

	// Required reports whether the field MUST appear in a valid
	// plan when the parent applicability is neutral.
	Required bool

	// Nullable reports whether the field accepts a JSON null value.
	Nullable bool

	// Pointer reports whether the Go field is a pointer.
	Pointer bool

	// ConstantValue, when non-nil, names the only accepted value.
	ConstantValue any

	// EnumAuthority is the closed, ordered list of accepted string
	// values.
	EnumAuthority []string

	// SemanticRule names the Go function or constant authoritative
	// semantic validation delegates to.
	SemanticRule string

	// DefaultingRule describes how the field is filled when the
	// producer omits it.
	DefaultingRule string

	// Description is the human-readable description of the field.
	Description string

	// ExampleValue is a literal example suitable for documentation
	// and the future example CLI command.
	ExampleValue any

	// Children is the ordered object descriptor for nested objects.
	Children *planObjectDescriptor

	// ItemDescriptor, when non-nil, describes the items of an array.
	ItemDescriptor *planFieldDescriptor

	// MinItems is the inclusive minimum number of array items.
	MinItems int

	// Applicability, when non-nil, declares that this field applies
	// only when the named sibling equals the named value.

	// ApplicabilityRules is the authoritative, exhaustive list of
	// presence rules the applicability walker consults. A field with
	// both Applicability (legacy single rule) and ApplicabilityRules
	// (rule list) MUST have both encoded explicitly; the walker does
	// not infer inverse behavior from a single rule.
	ApplicabilityRules []fieldApplicabilityRule

	// RejectedAliases are JSON names that LOOK like this field but
	// MUST be rejected.
	RejectedAliases []string
}

// planFieldKind enumerates the JSON type categories.
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

// String renders the kind as a stable lowercase label.
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
