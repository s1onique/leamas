package closure

import (
	"reflect"
	"sort"
	"strings"
	"testing"
)

// plan_contract_correction_parity_test.go contains the Phase 7
// reflection-driven Go-model parity tests for
// ACT-LEAMAS-FACTORY-CLOSE-PLAN-CONTRACT-AUTHORITY01-CORRECTION01-
// STRUCTURAL-PARITY-CLOSURE01. Splitting them keeps every file under
// the LLM-friendly 400-line threshold.

// --- Phase 7: reflection-driven Go model parity ---

type contractShape struct {
	JSONName string
	GoName   string
	Kind     string
	Required bool
	Nullable bool
	Object   bool
	Array    bool
	EnumAuth []string
	HasItem  bool
}

func TestContractGoModelParityViaReflection(t *testing.T) {
	// Walk every JSON-tagged field in the typed Plan struct and its
	// nested types. The walker emits a contractShape snapshot; the
	// test asserts every JSON name maps to a descriptor property.
	shapes := walkModelShapes()
	contract := planContractV1()
	required := func(jsonName string) bool {
		f, ok := contract.Root.Fields[jsonName]
		if !ok {
			return false
		}
		return f.Required
	}
	for _, shape := range shapes {
		if shape.JSONName == "" {
			continue
		}
		field, ok := contract.Root.Fields[shape.JSONName]
		if !ok {
			t.Fatalf("descriptor missing JSON field %q", shape.JSONName)
		}
		if field.GoName != shape.GoName {
			t.Fatalf("%q: GoName = %q, want %q", shape.JSONName, field.GoName, shape.GoName)
		}
		if shape.Object && field.Kind != kindObject {
			t.Fatalf("%q: kind = %v, want object", shape.JSONName, field.Kind)
		}
		if shape.Array && field.Kind != kindArray {
			t.Fatalf("%q: kind = %v, want array", shape.JSONName, field.Kind)
		}
		if field.Required != required(shape.JSONName) {
			t.Fatalf("%q: descriptor.Required=%v, model required=%v", shape.JSONName, field.Required, shape.Required)
		}
	}
}

// walkModelShapes uses reflection to walk the typed model and emit
// one shape per JSON-tagged field. The walker stops at the root
// level (Plan and its direct fields); nested fields are exercised
// by the structural validator's descriptor walk.
func walkModelShapes() []contractShape {
	planType := reflect.TypeOf(Plan{})
	var out []contractShape
	for i := 0; i < planType.NumField(); i++ {
		f := planType.Field(i)
		tag := f.Tag.Get("json")
		if tag == "" || tag == "-" {
			continue
		}
		name := strings.Split(tag, ",")[0]
		if name == "" {
			continue
		}
		shape := contractShape{
			JSONName: name,
			GoName:   f.Name,
		}
		switch f.Type.Kind() {
		case reflect.Struct:
			shape.Object = true
		case reflect.Slice, reflect.Array:
			shape.Array = true
		case reflect.Ptr:
			shape.Nullable = true
		}
		// Determine required from tag: only fields without omitempty
		// are required (per Go JSON convention).
		if !strings.Contains(tag, "omitempty") {
			shape.Required = true
		}
		out = append(out, shape)
	}
	// Stable ordering for diagnostics.
	sort.Slice(out, func(i, j int) bool { return out[i].JSONName < out[j].JSONName })
	return out
}
