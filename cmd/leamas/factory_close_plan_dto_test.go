package main

import (
	"bytes"
	"encoding/json"
	"io"
	"strings"
	"testing"

	"github.com/s1onique/leamas/internal/factory/closure"
)

// TestDTO_ExactTopLevelKeys pins the public validation wire to
// exactly the documented key set. Adding an internal JSON-tagged
// field must NOT change the CLI protocol; this test catches that
// regression.
func TestDTO_ExactTopLevelKeys(t *testing.T) {
	result := closure.ComposedPlanValidationResult{
		Structural: closure.PlanValidationResult{Valid: true},
		Valid:      true,
	}
	dto := toPlanValidationDTO(result)
	data, err := json.Marshal(dto)
	if err != nil {
		t.Fatal(err)
	}
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatal(err)
	}

	wantKeys := map[string]bool{
		"structural":      false,
		"decoded":         false,
		"decode_errors":   false,
		"semantic_valid":  false,
		"semantic_errors": false,
		"valid":           false,
	}
	for k := range raw {
		if _, ok := wantKeys[k]; ok {
			wantKeys[k] = true
		} else {
			t.Errorf("unexpected top-level key: %q", k)
		}
	}
	for k, present := range wantKeys {
		if !present {
			t.Errorf("missing top-level key: %q", k)
		}
	}
}

// TestDTO_NoForbiddenFields proves no internal-cause/observer fields
// leak into the public wire.
func TestDTO_NoForbiddenFields(t *testing.T) {
	result := closure.ComposedPlanValidationResult{
		Decoded:       true,
		SemanticValid: true,
		Valid:         true,
	}
	dto := toPlanValidationDTO(result)
	data, _ := json.Marshal(dto)

	// Check that forbidden top-level field names are not present.
	// These are common internal-field names that must not leak into
	// the public wire.
	forbidden := []string{`"cause"`, `"observer"`, `"internal"`, `"impl"`}
	for _, f := range forbidden {
		if strings.Contains(string(data), f) {
			t.Errorf("DTO must not contain %q field: %s", f, string(data))
		}
	}
}

// TestDTO_NilArraysBecomeEmpty proves nil diagnostic arrays
// serialise as [] not null.
func TestDTO_NilArraysBecomeEmpty(t *testing.T) {
	result := closure.ComposedPlanValidationResult{
		Structural:     closure.PlanValidationResult{Valid: true},
		DecodeErrors:   nil,
		SemanticErrors: nil,
		Valid:          true,
	}
	dto := toPlanValidationDTO(result)
	data, _ := json.Marshal(dto)
	s := string(data)

	for _, key := range []string{`"decode_errors"`, `"semantic_errors"`, `"errors"`} {
		if strings.Contains(s, key+":null") {
			t.Errorf("DTO field %s must not be null: %s", key, s)
		}
	}
}

// TestDTO_NestedStructuralKeys pins the nested structural keys.
func TestDTO_NestedStructuralKeys(t *testing.T) {
	result := closure.ComposedPlanValidationResult{
		Structural: closure.PlanValidationResult{
			Valid:           true,
			ContractVersion: 1,
		},
		Valid: true,
	}
	dto := toPlanValidationDTO(result)
	data, _ := json.Marshal(dto)

	var raw struct {
		Structural map[string]any `json:"structural"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatal(err)
	}
	wantKeys := map[string]bool{"valid": false, "contract_version": false, "errors": false}
	for k := range raw.Structural {
		if _, ok := wantKeys[k]; ok {
			wantKeys[k] = true
		} else {
			t.Errorf("unexpected structural key: %q", k)
		}
	}
	for k, present := range wantKeys {
		if !present {
			t.Errorf("missing structural key: %q", k)
		}
	}
}

// TestDTO_DiagnosticKeys pins nested diagnostic keys to the
// canonical typed taxonomy.
func TestDTO_DiagnosticKeys(t *testing.T) {
	result := closure.ComposedPlanValidationResult{
		DecodeErrors: []closure.PlanValidationError{
			{
				InstancePath: "/x",
				SchemaPath:   "#/properties/x",
				Code:         closure.PlanValidationCode("required"),
				Keyword:      closure.PlanValidationKeyword("required"),
				Message:      "missing field",
			},
		},
		Valid: false,
	}
	dto := toPlanValidationDTO(result)
	data, _ := json.Marshal(dto)

	var raw struct {
		DecodeErrors []map[string]any `json:"decode_errors"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatal(err)
	}
	if len(raw.DecodeErrors) != 1 {
		t.Fatalf("expected 1 error, got %d", len(raw.DecodeErrors))
	}
	wantKeys := map[string]bool{
		"instance_path":   false,
		"schema_path":     false,
		"code":            false,
		"keyword":         false,
		"message":         false,
		"rejected_value":  false,
		"accepted_values": false,
		"property_name":   false,
	}
	for k := range raw.DecodeErrors[0] {
		if _, ok := wantKeys[k]; ok {
			wantKeys[k] = true
		} else {
			t.Errorf("unexpected diagnostic key: %q", k)
		}
	}
	for k, present := range wantKeys {
		if !present {
			t.Errorf("missing diagnostic key: %q", k)
		}
	}
}

// TestDTO_DiagnosticRuntimeProperty proves runner-authority
// diagnostics with empty InstancePath and nonempty PropertyName
// preserve PropertyName in the public wire.
func TestDTO_DiagnosticRuntimeProperty(t *testing.T) {
	result := closure.ComposedPlanValidationResult{
		SemanticErrors: []closure.PlanValidationError{
			{
				InstancePath: "",
				PropertyName: "vcs.revision",
				Code:         closure.PlanValidationCode("semantic"),
				Keyword:      closure.PlanValidationKeyword("required"),
				Message:      "missing field",
			},
		},
		Valid: false,
	}
	dto := toPlanValidationDTO(result)
	data, _ := json.Marshal(dto)

	var raw struct {
		SemanticErrors []map[string]any `json:"semantic_errors"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatal(err)
	}
	if len(raw.SemanticErrors) != 1 {
		t.Fatalf("expected 1 error, got %d", len(raw.SemanticErrors))
	}
	if raw.SemanticErrors[0]["instance_path"] != "" {
		t.Errorf("instance_path = %v, want empty", raw.SemanticErrors[0]["instance_path"])
	}
	if raw.SemanticErrors[0]["property_name"] != "vcs.revision" {
		t.Errorf("property_name = %v, want vcs.revision", raw.SemanticErrors[0]["property_name"])
	}
}

// TestDTO_DiagnosticAcceptedValuesNotNull proves AcceptedValues
// is [] not null when source is nil.
func TestDTO_DiagnosticAcceptedValuesNotNull(t *testing.T) {
	result := closure.ComposedPlanValidationResult{
		DecodeErrors: []closure.PlanValidationError{
			{
				InstancePath: "/x",
				Code:         closure.PlanValidationCode("required"),
			},
		},
		Valid: false,
	}
	dto := toPlanValidationDTO(result)
	data, _ := json.Marshal(dto)
	s := string(data)
	if strings.Contains(s, `"accepted_values":null`) {
		t.Errorf("accepted_values must not be null: %s", s)
	}
}

// TestDTO_DiagnosticDeepCopy proves deep-copy conversion does not
// share underlying memory.
func TestDTO_DiagnosticDeepCopy(t *testing.T) {
	src := closure.PlanValidationError{
		InstancePath: "/x",
		RejectedValue: map[string]any{
			"key": "value",
		},
		AcceptedValues: []string{"a", "b"},
	}
	dto := toPlanValidationDTO(closure.ComposedPlanValidationResult{
		DecodeErrors: []closure.PlanValidationError{src},
	})
	if len(dto.DecodeErrors) != 1 {
		t.Fatal("expected 1 error")
	}
	if dto.DecodeErrors[0].RejectedValue == nil {
		t.Fatal("rejected_value not preserved")
	}
	// Deep copy: mutating source must not affect DTO
	srcMap := src.RejectedValue.(map[string]any)
	srcMap["key"] = "mutated"
	dtoMap := dto.DecodeErrors[0].RejectedValue.(map[string]any)
	if dtoMap["key"] != "value" {
		t.Errorf("DTO shares underlying memory with source: %v", dtoMap)
	}
}

// countingReader returns bytes from a buffer and counts how many
// bytes were requested in total.
type countingReader struct {
	data  []byte
	pos   int
	reads int
	bytes int
}

func (r *countingReader) Read(p []byte) (n int, err error) {
	r.reads++
	if r.pos >= len(r.data) {
		return 0, io.EOF
	}
	n = copy(p, r.data[r.pos:])
	r.pos += n
	r.bytes += n
	return n, nil
}

// TestProductionBoundedRead_LimitsAtMax proves the production
// helper consumes at most MaxPlanBytes+1 bytes.
func TestProductionBoundedRead_LimitsAtMax(t *testing.T) {
	max := int64(closure.MaxPlanBytes + 1)
	// Provide data larger than max so the reader stops early
	data := bytes.Repeat([]byte("x"), int(max*2))
	r := &countingReader{data: data}
	got, err := productionBoundedRead(r, max)
	if err != nil && err != io.EOF {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) > int(max) {
		t.Errorf("got %d bytes, want <= %d", len(got), max)
	}
}

// TestProductionBoundedRead_InfiniteReader proves the production
// helper consumes at most MaxPlanBytes+1 bytes from an infinite stream.
func TestProductionBoundedRead_InfiniteReader(t *testing.T) {
	max := int64(closure.MaxPlanBytes + 1)
	r := &infiniteByteReader{}
	got, err := productionBoundedRead(r, max)
	if err != nil && err != io.EOF {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) > int(max) {
		t.Errorf("got %d bytes, want <= %d", len(got), max)
	}
}

// infiniteByteReader yields an infinite stream of 'x' bytes.
type infiniteByteReader struct{}

func (r *infiniteByteReader) Read(p []byte) (n int, err error) {
	for i := range p {
		p[i] = 'x'
	}
	return len(p), nil
}
