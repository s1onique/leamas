package closure

import (
	"encoding/json"
	"testing"
)

// executionModeFixture is the mutable, in-memory plan builder used by
// the canonical contract tests. The builder is small but principled:
//
//   - the base fixture passes strict validation (canonical mode);
//   - callers mutate fields through With to derive negative fixtures;
//   - Bytes() is the single source of the JSON the test feeds to the
//     strict decoder and the embedded JSON Schema.
//
// Sharing the builder prevents the unit, schema, and CLI subprocess
// tests from drifting: every test derives its input from the same
// canonical starting point.
type executionModeFixture struct {
	name string
	body map[string]any
}

func executionModeFixtureBuilder() map[string]any {
	return map[string]any{
		"contract_version": 1,
		"act_id":           "ACT-LEAMAS-EXECUTION-MODE-FIXTURE",
		"baseline": map[string]any{
			"commit_oid": "1111111111111111111111111111111111111111",
			"tree_oid":   "2222222222222222222222222222222222222222",
		},
		"execution": map[string]any{
			"mode": string(ExecutionModeSerialFailFast),
		},
		"checks": []any{
			map[string]any{
				"id":                "noop",
				"mode":              "run",
				"argv":              []any{"true"},
				"working_directory": ".",
				"timeout_seconds":   60,
				"environment":       map[string]any{},
			},
		},
		"artifacts": []any{},
		"policy": map[string]any{
			"require_clean_before":        true,
			"require_clean_after":         true,
			"forbid_tracked_full_digests": true,
			"require_diff_check":          true,
		},
	}
}

// newExecutionModeFixture returns a fixture whose name is used only
// for test diagnostics.
func newExecutionModeFixture(name string) *executionModeFixture {
	return &executionModeFixture{
		name: name,
		body: executionModeFixtureBuilder(),
	}
}

// With replaces the named top-level key with the supplied value.
// Tests use it to construct negative fixtures.
func (f *executionModeFixture) With(key string, value any) *executionModeFixture {
	f.body[key] = value
	return f
}

// Bytes renders the fixture as canonical JSON. Two-space indentation
// matches the rest of the closure-package test corpus.
func (f *executionModeFixture) Bytes() []byte {
	out, err := json.MarshalIndent(f.body, "", "  ")
	if err != nil {
		panic("executionModeFixture marshal: " + err.Error())
	}
	return out
}

// executionModeParityFixture describes one entry in the schema/runtime
// parity table. WantRuntimeAccept and WantSchemaAccept must agree for
// every entry; the table is the contract that makes the runtime and
// schema non-driftable.
type executionModeParityFixture struct {
	Name                   string
	Mutate                 func(map[string]any)
	WantRuntimeAccept      bool
	WantSchemaAccept       bool
	WantDiagnosticContains string
}

// executionModeParityFixtures is the closed table the schema/runtime
// parity test walks. Every presence category the directive ACT
// required is represented here exactly once.
func executionModeParityFixtures() []executionModeParityFixture {
	return []executionModeParityFixture{
		{
			Name:              "canonical",
			WantRuntimeAccept: true,
			WantSchemaAccept:  true,
		},
		{
			Name:                   "execution-omitted",
			Mutate:                 func(p map[string]any) { delete(p, "execution") },
			WantRuntimeAccept:      false,
			WantSchemaAccept:       false,
			WantDiagnosticContains: "required",
		},
		{
			Name:                   "execution-empty-object",
			Mutate:                 func(p map[string]any) { p["execution"] = map[string]any{} },
			WantRuntimeAccept:      false,
			WantSchemaAccept:       false,
			WantDiagnosticContains: "required",
		},
		{
			Name:                   "execution-mode-empty-string",
			Mutate:                 func(p map[string]any) { p["execution"] = map[string]any{"mode": ""} },
			WantRuntimeAccept:      false,
			WantSchemaAccept:       false,
			WantDiagnosticContains: "empty",
		},
		{
			Name:                   "execution-mode-whitespace",
			Mutate:                 func(p map[string]any) { p["execution"] = map[string]any{"mode": "   "} },
			WantRuntimeAccept:      false,
			WantSchemaAccept:       false,
			WantDiagnosticContains: "whitespace",
		},
		{
			Name:                   "execution-mode-unknown",
			Mutate:                 func(p map[string]any) { p["execution"] = map[string]any{"mode": "parallel"} },
			WantRuntimeAccept:      false,
			WantSchemaAccept:       false,
			WantDiagnosticContains: "not a supported execution mode",
		},
		{
			Name: "execution-mode-uppercase",
			Mutate: func(p map[string]any) {
				p["execution"] = map[string]any{"mode": "SERIAL_FAIL_FAST"}
			},
			WantRuntimeAccept: false,
			WantSchemaAccept:  false,
		},
		{
			Name: "execution-mode-trailing-space",
			Mutate: func(p map[string]any) {
				p["execution"] = map[string]any{"mode": "serial_fail_fast "}
			},
			WantRuntimeAccept: false,
			WantSchemaAccept:  false,
		},
		{
			Name: "execution-mode-number",
			Mutate: func(p map[string]any) {
				p["execution"] = map[string]any{"mode": 1}
			},
			WantRuntimeAccept: false,
			WantSchemaAccept:  false,
		},
		{
			Name: "execution-mode-bool",
			Mutate: func(p map[string]any) {
				p["execution"] = map[string]any{"mode": true}
			},
			WantRuntimeAccept: false,
			WantSchemaAccept:  false,
		},
		{
			Name: "execution-additional-property",
			Mutate: func(p map[string]any) {
				p["execution"] = map[string]any{"mode": string(ExecutionModeSerialFailFast), "extra": true}
			},
			WantRuntimeAccept: false,
			WantSchemaAccept:  false,
		},
		{
			Name: "policy-mode-alias",
			Mutate: func(p map[string]any) {
				p["policy"] = map[string]any{
					"mode":                        string(ExecutionModeSerialFailFast),
					"require_clean_before":        true,
					"require_clean_after":         true,
					"forbid_tracked_full_digests": true,
					"require_diff_check":          true,
				}
			},
			WantRuntimeAccept: false,
			WantSchemaAccept:  false,
		},
		{
			Name: "top-level-mode-alias",
			Mutate: func(p map[string]any) {
				p["mode"] = string(ExecutionModeSerialFailFast)
			},
			WantRuntimeAccept: false,
			WantSchemaAccept:  false,
		},
		{
			Name: "unknown-sibling-property",
			Mutate: func(p map[string]any) {
				p["surprise"] = "nope"
			},
			WantRuntimeAccept: false,
			WantSchemaAccept:  false,
		},
		{
			Name: "policy-false-rejected",
			Mutate: func(p map[string]any) {
				p["policy"] = map[string]any{
					"require_clean_before":        false,
					"require_clean_after":         true,
					"forbid_tracked_full_digests": true,
					"require_diff_check":          true,
				}
			},
			WantRuntimeAccept: false,
			WantSchemaAccept:  false,
		},
	}
}

// Bytes renders the parity fixture's JSON. It composes the canonical
// builder with the optional Mutate closure so every fixture shares
// the same baseline shape.
func (p executionModeParityFixture) Bytes(t *testing.T) []byte {
	t.Helper()
	body := executionModeFixtureBuilder()
	if p.Mutate != nil {
		p.Mutate(body)
	}
	out, err := json.MarshalIndent(body, "", "  ")
	if err != nil {
		t.Fatalf("marshal fixture %q: %v", p.Name, err)
	}
	return out
}
