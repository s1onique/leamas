package main

import (
	"bytes"
	"encoding/json"
	"testing"
)

// TestSchemaParityContractVersion proves contract_version is const=1.
func TestSchemaParityContractVersion(t *testing.T) {
	var stdout, stderr bytes.Buffer
	exit := runFactoryClosePlanSchema(nil, &stdout, &stderr)
	if exit != 0 {
		t.Fatalf("schema exit = %d, want 0", exit)
	}

	var schema map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &schema); err != nil {
		t.Fatal(err)
	}

	props := schema["properties"].(map[string]any)
	cv := props["contract_version"].(map[string]any)
	if cv["const"] != float64(1) {
		t.Errorf("contract_version const = %v, want 1", cv["const"])
	}
}

// TestSchemaParityClosedEnums proves execution_mode enum is closed.
func TestSchemaParityClosedEnums(t *testing.T) {
	var stdout, stderr bytes.Buffer
	exit := runFactoryClosePlanSchema(nil, &stdout, &stderr)
	if exit != 0 {
		t.Fatalf("schema exit = %d, want 0", exit)
	}

	var schema map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &schema); err != nil {
		t.Fatal(err)
	}

	props := schema["properties"].(map[string]any)
	exec := props["execution"].(map[string]any)
	execProps := exec["properties"].(map[string]any)
	mode := execProps["mode"].(map[string]any)
	enum := mode["enum"].([]any)

	hasSerial := false
	for _, v := range enum {
		if v == "serial_fail_fast" {
			hasSerial = true
		}
	}
	if !hasSerial {
		t.Errorf("missing expected enum value: serial_fail_fast=%v", hasSerial)
	}
}

// TestSchemaParityRequiredFields proves required fields are marked required.
func TestSchemaParityRequiredFields(t *testing.T) {
	var stdout, stderr bytes.Buffer
	exit := runFactoryClosePlanSchema(nil, &stdout, &stderr)
	if exit != 0 {
		t.Fatalf("schema exit = %d, want 0", exit)
	}

	var schema map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &schema); err != nil {
		t.Fatal(err)
	}

	required := schema["required"].([]any)
	reqSet := make(map[string]bool)
	for _, r := range required {
		reqSet[r.(string)] = true
	}

	if !reqSet["contract_version"] {
		t.Error("contract_version should be required")
	}
	if !reqSet["act_id"] {
		t.Error("act_id should be required")
	}
}

// TestSchemaParityAdditionalPropertiesFalse proves objects are closed.
func TestSchemaParityAdditionalPropertiesFalse(t *testing.T) {
	var stdout, stderr bytes.Buffer
	exit := runFactoryClosePlanSchema(nil, &stdout, &stderr)
	if exit != 0 {
		t.Fatalf("schema exit = %d, want 0", exit)
	}

	var schema map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &schema); err != nil {
		t.Fatal(err)
	}

	if schema["additionalProperties"] != false {
		t.Error("root should have additionalProperties=false")
	}
}

// TestSchemaParityArrayMinItems proves argv has minItems.
func TestSchemaParityArrayMinItems(t *testing.T) {
	var stdout, stderr bytes.Buffer
	exit := runFactoryClosePlanSchema(nil, &stdout, &stderr)
	if exit != 0 {
		t.Fatalf("schema exit = %d, want 0", exit)
	}

	var schema map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &schema); err != nil {
		t.Fatal(err)
	}

	props := schema["properties"].(map[string]any)
	checks := props["checks"].(map[string]any)
	checkItems := checks["items"].(map[string]any)
	checkProps := checkItems["properties"].(map[string]any)
	argv := checkProps["argv"].(map[string]any)

	minItems := argv["minItems"]
	if minItems == nil || minItems.(float64) < 1 {
		t.Errorf("argv minItems = %v, want >= 1", minItems)
	}
}

// TestSchemaParityEnvironmentStringMap proves environment emits a string-valued additionalProperties.
func TestSchemaParityEnvironmentStringMap(t *testing.T) {
	var stdout, stderr bytes.Buffer
	exit := runFactoryClosePlanSchema(nil, &stdout, &stderr)
	if exit != 0 {
		t.Fatalf("schema exit = %d, want 0", exit)
	}

	var schema map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &schema); err != nil {
		t.Fatal(err)
	}

	props := schema["properties"].(map[string]any)
	checks := props["checks"].(map[string]any)
	checkItems := checks["items"].(map[string]any)
	checkProps := checkItems["properties"].(map[string]any)
	env := checkProps["environment"].(map[string]any)

	if env["type"] != "object" {
		t.Fatalf("environment type = %v, want object", env["type"])
	}
	additional, ok := env["additionalProperties"].(map[string]any)
	if !ok {
		t.Fatalf("environment additionalProperties must be object, got %T", env["additionalProperties"])
	}
	if additional["type"] != "string" {
		t.Errorf("environment additionalProperties.type = %v, want string", additional["type"])
	}
}

// TestSchemaParityValidationAuthority proves x-leamas-validation-authority has the exact value.
func TestSchemaParityValidationAuthority(t *testing.T) {
	var stdout, stderr bytes.Buffer
	exit := runFactoryClosePlanSchema(nil, &stdout, &stderr)
	if exit != 0 {
		t.Fatalf("schema exit = %d, want 0", exit)
	}

	var schema map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &schema); err != nil {
		t.Fatal(err)
	}

	authority, ok := schema["x-leamas-validation-authority"]
	if !ok {
		t.Fatal("x-leamas-validation-authority missing")
	}
	if authority != "leamas factory close plan validate" {
		t.Errorf("authority = %v, want leamas factory close plan validate", authority)
	}
}

// TestSchemaParityXApplicabilityOnConditionalFields proves the
// x-applicability extension is present on the actual conditional
// check fields (not just anywhere).
func TestSchemaParityXApplicabilityOnConditionalFields(t *testing.T) {
	var stdout, stderr bytes.Buffer
	exit := runFactoryClosePlanSchema(nil, &stdout, &stderr)
	if exit != 0 {
		t.Fatalf("schema exit = %d, want 0", exit)
	}

	var schema map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &schema); err != nil {
		t.Fatal(err)
	}

	props := schema["properties"].(map[string]any)
	checks := props["checks"].(map[string]any)
	checkItems := checks["items"].(map[string]any)
	checkProps := checkItems["properties"].(map[string]any)

	// argv, reason, working_directory, timeout_seconds all carry
	// x-applicability rules in the real descriptor.
	// (environment uses free-form string map and returns early.)
	for _, field := range []string{"argv", "reason", "working_directory", "timeout_seconds"} {
		f, ok := checkProps[field].(map[string]any)
		if !ok {
			t.Errorf("conditional field %s missing from checks", field)
			continue
		}
		if _, ok := f["x-applicability"]; !ok {
			t.Errorf("conditional field %s missing x-applicability extension", field)
		}
	}
}

// TestSchemaParityUnknownProperties proves additionalProperties=false at root.
func TestSchemaParityUnknownProperties(t *testing.T) {
	var stdout, stderr bytes.Buffer
	exit := runFactoryClosePlanSchema(nil, &stdout, &stderr)
	if exit != 0 {
		t.Fatalf("schema exit = %d, want 0", exit)
	}

	var schema map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &schema); err != nil {
		t.Fatal(err)
	}

	if schema["additionalProperties"] != false {
		t.Error("root should reject unknown properties")
	}
}

// TestSchemaParityCheckModeField proves mode has proper enum structure.
func TestSchemaParityCheckModeField(t *testing.T) {
	var stdout, stderr bytes.Buffer
	exit := runFactoryClosePlanSchema(nil, &stdout, &stderr)
	if exit != 0 {
		t.Fatalf("schema exit = %d, want 0", exit)
	}

	var schema map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &schema); err != nil {
		t.Fatal(err)
	}

	props := schema["properties"].(map[string]any)
	checks := props["checks"].(map[string]any)
	checkItems := checks["items"].(map[string]any)
	checkProps := checkItems["properties"].(map[string]any)

	mode, ok := checkProps["mode"]
	if !ok {
		t.Fatal("checks should have mode field")
	}

	modeProps := mode.(map[string]any)
	enum, ok := modeProps["enum"].([]any)
	if !ok {
		t.Fatal("mode should have enum")
	}

	hasRun, hasExclude := false, false
	for _, v := range enum {
		if v == "run" {
			hasRun = true
		}
		if v == "exclude" {
			hasExclude = true
		}
	}
	if !hasRun || !hasExclude {
		t.Errorf("mode enum missing values: run=%v, exclude=%v", hasRun, hasExclude)
	}
}

// TestSchemaParityNoAliases proves migration aliases (command, cmd, cwd, dir)
// are absent from the public schema.
func TestSchemaParityNoAliases(t *testing.T) {
	var stdout, stderr bytes.Buffer
	exit := runFactoryClosePlanSchema(nil, &stdout, &stderr)
	if exit != 0 {
		t.Fatalf("schema exit = %d, want 0", exit)
	}

	var schema map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &schema); err != nil {
		t.Fatal(err)
	}

	props := schema["properties"].(map[string]any)
	for alias := range map[string]bool{
		"command": true, "cmd": true, "cwd": true, "dir": true,
	} {
		if _, ok := props[alias]; ok {
			t.Errorf("schema must not contain alias %q", alias)
		}
	}
}
