package closure

import (
	"errors"
	"strings"
	"testing"
)

// TestJSONSchema_Success proves JSONSchema works on the real descriptor.
func TestJSONSchema_Success(t *testing.T) {
	schema, err := JSONSchema()
	if err != nil {
		t.Fatalf("JSONSchema failed: %v", err)
	}
	if schema == nil {
		t.Fatal("JSONSchema returned nil schema")
	}
}

// TestBuildFieldSchema_UnknownKind proves unknown kind returns ErrSchemaGeneration.
func TestBuildFieldSchema_UnknownKind(t *testing.T) {
	field := planFieldDescriptor{
		JSONName: "bad",
		Kind:     kindUnknown,
	}
	_, err := buildFieldSchema(field, "/bad")
	if err == nil {
		t.Fatal("expected error for unknown kind")
	}
	if !errors.Is(err, ErrSchemaGeneration) {
		t.Errorf("error must wrap ErrSchemaGeneration: %v", err)
	}
}

// TestBuildFieldSchema_ArrayWithoutItems proves array without item descriptor returns ErrSchemaGeneration.
func TestBuildFieldSchema_ArrayWithoutItems(t *testing.T) {
	field := planFieldDescriptor{
		JSONName: "arr",
		Kind:     kindArray,
	}
	_, err := buildFieldSchema(field, "/arr")
	if err == nil {
		t.Fatal("expected error for array without item descriptor")
	}
	if !errors.Is(err, ErrSchemaGeneration) {
		t.Errorf("error must wrap ErrSchemaGeneration: %v", err)
	}
}

// TestBuildFieldSchema_ObjectWithoutChildren proves object without children returns ErrSchemaGeneration.
func TestBuildFieldSchema_ObjectWithoutChildren(t *testing.T) {
	field := planFieldDescriptor{
		JSONName: "obj",
		Kind:     kindObject,
	}
	_, err := buildFieldSchema(field, "/obj")
	if err == nil {
		t.Fatal("expected error for object without children")
	}
	if !errors.Is(err, ErrSchemaGeneration) {
		t.Errorf("error must wrap ErrSchemaGeneration: %v", err)
	}
}

// TestBuildObjectProperties_PropagatesError proves errors propagate up.
func TestBuildObjectProperties_PropagatesError(t *testing.T) {
	obj := planObjectDescriptor{
		Fields: map[string]planFieldDescriptor{
			"bad": {JSONName: "bad", Kind: kindUnknown},
		},
	}
	_, _, err := buildObjectProperties(obj, "/root")
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, ErrSchemaGeneration) {
		t.Errorf("error must wrap ErrSchemaGeneration: %v", err)
	}
}

// TestBuildFieldSchema_StringMapSchema proves free-form string maps emit additionalProperties:type=string.
func TestBuildFieldSchema_StringMapSchema(t *testing.T) {
	field := planFieldDescriptor{
		JSONName: "env",
		Kind:     kindObject,
		Children: &planObjectDescriptor{
			Kind: objectStringMap,
		},
	}
	schema, err := buildFieldSchema(field, "/env")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if schema["type"] != "object" {
		t.Errorf("type = %v, want object", schema["type"])
	}
	additional, ok := schema["additionalProperties"]
	if !ok {
		t.Fatal("additionalProperties missing")
	}
	additionalMap, ok := additional.(map[string]any)
	if !ok {
		t.Fatalf("additionalProperties type = %T, want map", additional)
	}
	if additionalMap["type"] != "string" {
		t.Errorf("additionalProperties.type = %v, want string", additionalMap["type"])
	}
}

// TestBuildFieldSchema_ApplicabilityRules prove x-applicability extension is emitted.
func TestBuildFieldSchema_ApplicabilityRules(t *testing.T) {
	field := planFieldDescriptor{
		JSONName: "argv",
		Kind:     kindArray,
		ApplicabilityRules: []fieldApplicabilityRule{
			{Sibling: "mode", Value: "run", Presence: PresenceRequired},
		},
		ItemDescriptor: &planFieldDescriptor{JSONName: "argv[]", Kind: kindString},
	}
	schema, err := buildFieldSchema(field, "/argv")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	appl, ok := schema["x-applicability"]
	if !ok {
		t.Fatal("x-applicability missing")
	}
	rules, ok := appl.([]map[string]any)
	if !ok {
		t.Fatalf("x-applicability type = %T", appl)
	}
	if len(rules) != 1 {
		t.Errorf("rules = %d, want 1", len(rules))
	}
}

// TestJSONSchema_HasValidationAuthority proves x-leamas-validation-authority is set.
func TestJSONSchema_HasValidationAuthority(t *testing.T) {
	schema, err := JSONSchema()
	if err != nil {
		t.Fatalf("JSONSchema failed: %v", err)
	}
	authority, ok := schema["x-leamas-validation-authority"]
	if !ok {
		t.Fatal("x-leamas-validation-authority missing")
	}
	if authority != "leamas factory close plan validate" {
		t.Errorf("authority = %v, want leamas factory close plan validate", authority)
	}
}

// TestBuildFieldSchema_EnumField proves enum fields produce type=string with enum values.
func TestBuildFieldSchema_EnumField(t *testing.T) {
	field := planFieldDescriptor{
		JSONName:      "mode",
		Kind:          kindEnum,
		EnumAuthority: []string{"run", "exclude"},
	}
	schema, err := buildFieldSchema(field, "/mode")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if schema["type"] != "string" {
		t.Errorf("type = %v, want string", schema["type"])
	}
	enum, ok := schema["enum"].([]string)
	if !ok {
		t.Fatalf("enum type = %T", schema["enum"])
	}
	if len(enum) != 2 {
		t.Errorf("enum length = %d, want 2", len(enum))
	}
}

// TestBuildFieldSchema_ConstantValue proves const fields emit const without type.
func TestBuildFieldSchema_ConstantValue(t *testing.T) {
	field := planFieldDescriptor{
		JSONName:      "contract_version",
		Kind:          kindInteger,
		ConstantValue: 1,
	}
	schema, err := buildFieldSchema(field, "/contract_version")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if schema["const"] != 1 {
		t.Errorf("const = %v, want 1", schema["const"])
	}
}

// TestBuildFieldSchema_MinItems proves arrays emit minItems.
func TestBuildFieldSchema_MinItems(t *testing.T) {
	field := planFieldDescriptor{
		JSONName:       "argv",
		Kind:           kindArray,
		MinItems:       1,
		ItemDescriptor: &planFieldDescriptor{JSONName: "argv[]", Kind: kindString},
	}
	schema, err := buildFieldSchema(field, "/argv")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if schema["minItems"] != 1 {
		t.Errorf("minItems = %v, want 1", schema["minItems"])
	}
}

// TestBuildFieldSchema_ErrorMessageFormat proves error messages identify the path.
func TestBuildFieldSchema_ErrorMessageFormat(t *testing.T) {
	field := planFieldDescriptor{JSONName: "x", Kind: kindUnknown}
	_, err := buildFieldSchema(field, "/test/path")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "/test/path") {
		t.Errorf("error must contain path: %v", err)
	}
}

// TestBuildFieldSchema_DuplicateApplicability proves duplicate (sibling, value) pairs are rejected.
func TestBuildFieldSchema_DuplicateApplicability(t *testing.T) {
	field := planFieldDescriptor{
		JSONName: "argv",
		Kind:     kindArray,
		ApplicabilityRules: []fieldApplicabilityRule{
			{Sibling: "mode", Value: "run", Presence: PresenceRequired},
			{Sibling: "mode", Value: "run", Presence: PresenceForbidden},
		},
		ItemDescriptor: &planFieldDescriptor{JSONName: "argv[]", Kind: kindString},
	}
	_, err := buildFieldSchema(field, "/argv")
	if err == nil {
		t.Fatal("expected error for duplicate applicability identity")
	}
	if !errors.Is(err, ErrSchemaGeneration) {
		t.Errorf("error must wrap ErrSchemaGeneration: %v", err)
	}
}

// TestBuildFieldSchema_UniqueApplicability proves distinct (sibling, value) pairs are accepted.
func TestBuildFieldSchema_UniqueApplicability(t *testing.T) {
	field := planFieldDescriptor{
		JSONName: "argv",
		Kind:     kindArray,
		ApplicabilityRules: []fieldApplicabilityRule{
			{Sibling: "mode", Value: "run", Presence: PresenceRequired},
			{Sibling: "mode", Value: "exclude", Presence: PresenceForbidden},
		},
		ItemDescriptor: &planFieldDescriptor{JSONName: "argv[]", Kind: kindString},
	}
	schema, err := buildFieldSchema(field, "/argv")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	appl, ok := schema["x-applicability"]
	if !ok {
		t.Fatal("x-applicability missing")
	}
	rules := appl.([]map[string]any)
	if len(rules) != 2 {
		t.Errorf("rules = %d, want 2", len(rules))
	}
}
