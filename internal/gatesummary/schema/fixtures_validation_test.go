// Schema fixture validation tests.
//
// These tests assert that the published v1 and v2 schemas describe the
// wire documents the existing decoder accepts, and that structurally
// invalid documents are rejected by the same schema. The tests are
// driven by the existing testdata/ tree so the schema is guarded by
// the same corpus that the decoder sees.
//
// The validator is the same Draft 2020-12 compiler used by the
// production decoder path (san64/jsonschema/v6). The tests do not
// reach the network; they compile the bytes held by the registry
// subpackage directly.
//
// The schema-vs-decoder matrix distinguishes:
//   - structural invalid: the decoder rejects at the wire boundary
//     and the schema must reject with a structural error.
//   - semantic invalid: the decoder accepts for handoff and the
//     normalizer rejects. The schema intentionally accepts these
//     because the failures are not JSON-Schema-representable.
//   - pre-schema invalid: the decoder rejects before the schema
//     stage (malformed JSON, trailing JSON, decimal schema_version).
//     The schema cannot reject these by definition.
package schema

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	jsonschema "github.com/santhosh-tekuri/jsonschema/v6"
)

// compiledV1 / compiledV2 are the Draft 2020-12 compiled schemas used
// by the fixture tests. They are intentionally package-private so the
// network-aware jsonschema dependency is only present in _test.go.
var (
	compiledV1 *jsonschema.Schema
	compiledV2 *jsonschema.Schema
)

func compileForTest(t *testing.T) {
	t.Helper()
	if compiledV1 != nil && compiledV2 != nil {
		return
	}
	c := jsonschema.NewCompiler()
	c.AssertFormat()
	if err := c.AddResource(SchemaIDV1, jsonRaw(MustBytes(VersionV1))); err != nil {
		t.Fatalf("add v1: %v", err)
	}
	if err := c.AddResource(SchemaIDV2, jsonRaw(MustBytes(VersionV2))); err != nil {
		t.Fatalf("add v2: %v", err)
	}
	v1, err := c.Compile(SchemaIDV1)
	if err != nil {
		t.Fatalf("compile v1: %v", err)
	}
	v2, err := c.Compile(SchemaIDV2)
	if err != nil {
		t.Fatalf("compile v2: %v", err)
	}
	compiledV1 = v1
	compiledV2 = v2
}

// jsonRaw decodes schema bytes into a generic value with UseNumber so
// arbitrary-precision integers survive the decode/validate round.
func jsonRaw(data []byte) any {
	var v any
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.UseNumber()
	if err := dec.Decode(&v); err != nil {
		return nil
	}
	return v
}

// structuralInvalidV1 is the closed set of v1 fixtures the decoder
// rejects at the structural level. The schema must reject each of
// these with a structural validation error.
var structuralInvalidV1 = []string{
	"v1-unknown-field.json",
}

// structuralInvalidV2 is the closed set of v2 fixtures the decoder
// rejects at the structural level. The schema must reject each of
// these with a structural validation error.
//
// Fixtures classified as semantic-only invalid (the decoder accepts
// for handoff and the normalizer rejects) are listed in
// semanticInvalidV2 and are NOT asserted here.
var structuralInvalidV2 = []string{
	"v2-bad-status-enum.json",
	"v2-empty-generated-at.json",
	"v2-invalid-hash.json",
	"v2-invalid-timestamp.json",
	"v2-lower-lifecycle.json",
	"v2-missing-execution-head-oid.json",
	"v2-missing-schema-version.json",
	"v2-negative-duration.json",
	"v2-null-execution-head-oid.json",
	"v2-partial-test-totals.json",
	"v2-schema-version-negative.json",
	"v2-schema-version-string.json",
	"v2-schema-version-zero.json",
	"v2-unknown-field.json",
	"v2-unsupported-version-3.json",
	"v2-uppercase-oid.json",
}

// invalidV2CapturedByPreSchemaEnvelope groups fixtures that the
// decoder rejects before the schema stage (malformed JSON, trailing
// JSON, decimal schema_version). These are structural rejects but
// the schema itself cannot reject them — the pre-schema envelope
// scanner owns that stage.
var invalidV2CapturedByPreSchemaEnvelope = []string{
	"v2-schema-version-decimal.json",
	"v2-trailing-second-value.json",
	"v2-truncated.json",
}

// semanticInvalidV2 is the documented set of v2 fixtures the decoder
// accepts for handoff but the normalizer rejects. The schema
// intentionally accepts these because the failures are not
// JSON-Schema-representable. This table is the contract.
type semanticInvalidEntry struct {
	fixture string
	reason  string
}

var semanticInvalidV2 = []semanticInvalidEntry{
	{"v2-duplicate-check-name.json", "duplicate check name (semantic)"},
	{"v2-fail-exit-zero.json", "overall fail with exit_code 0 (semantic)"},
	{"v2-overall-mismatch.json", "overall status mismatch with check list (semantic)"},
	{"v2-pass-nonzero-exit.json", "overall pass with exit_code 1 (semantic)"},
	{"v2-scope-closed-dirty-after.json", "scope closed with dirty worktree (semantic)"},
	{"v2-skip-nonnull-exit.json", "skip with nonnull exit_code (semantic)"},
	{"v2-test-total-mismatch.json", "test totals arithmetic mismatch (semantic)"},
	{"v2-unavailable-nonnull-exit.json", "unavailable with nonnull exit_code (semantic)"},
}

// fixturesOfVersion returns the list of testdata fixture files used
// for the given schema version. The directory layout is owned by the
// decoder's corpus tests.
func fixturesOfVersion(t *testing.T, version string) (valid, invalid []string) {
	t.Helper()
	dir := filepath.Join("..", "testdata", "valid")
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read valid fixtures: %v", err)
	}
	for _, e := range entries {
		if !e.IsDir() && strings.HasPrefix(e.Name(), "v"+version+"-") && strings.HasSuffix(e.Name(), ".json") {
			valid = append(valid, filepath.Join(dir, e.Name()))
		}
	}
	sort.Strings(valid)
	dir = filepath.Join("..", "testdata", "invalid")
	entries, err = os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read invalid fixtures: %v", err)
	}
	for _, e := range entries {
		if !e.IsDir() && strings.HasPrefix(e.Name(), "v"+version+"-") && strings.HasSuffix(e.Name(), ".json") {
			invalid = append(invalid, filepath.Join(dir, e.Name()))
		}
	}
	sort.Strings(invalid)
	return valid, invalid
}

// TestFixturesV1AcceptsAllValid asserts that every canonical v1
// fixture validates against the published v1 schema.
func TestFixturesV1AcceptsAllValid(t *testing.T) {
	compileForTest(t)
	valid, _ := fixturesOfVersion(t, "1")
	if len(valid) == 0 {
		t.Fatal("no v1 valid fixtures found")
	}
	for _, path := range valid {
		t.Run(filepath.Base(path), func(t *testing.T) {
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("read: %v", err)
			}
			if err := validateBytes(compiledV1, data); err != nil {
				t.Fatalf("valid v1 fixture %s rejected by schema: %v", path, err)
			}
		})
	}
}

// TestFixturesV2AcceptsAllValid asserts that every canonical v2
// fixture validates against the published v2 schema.
func TestFixturesV2AcceptsAllValid(t *testing.T) {
	compileForTest(t)
	valid, _ := fixturesOfVersion(t, "2")
	if len(valid) == 0 {
		t.Fatal("no v2 valid fixtures found")
	}
	for _, path := range valid {
		t.Run(filepath.Base(path), func(t *testing.T) {
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("read: %v", err)
			}
			if err := validateBytes(compiledV2, data); err != nil {
				t.Fatalf("valid v2 fixture %s rejected by schema: %v", path, err)
			}
		})
	}
}

// TestFixturesV1RejectsStructuralInvalid asserts that the v1 schema
// rejects every canonical structurally-invalid v1 fixture.
func TestFixturesV1RejectsStructuralInvalid(t *testing.T) {
	compileForTest(t)
	for _, name := range structuralInvalidV1 {
		t.Run(name, func(t *testing.T) {
			data, err := os.ReadFile(filepath.Join("..", "testdata", "invalid", name))
			if err != nil {
				t.Fatalf("read: %v", err)
			}
			if err := validateBytes(compiledV1, data); err == nil {
				t.Fatalf("invalid v1 fixture %s accepted by schema", name)
			}
		})
	}
}

// TestFixturesV2RejectsStructuralInvalid asserts that the v2 schema
// rejects every canonical structurally-invalid v2 fixture.
func TestFixturesV2RejectsStructuralInvalid(t *testing.T) {
	compileForTest(t)
	for _, name := range structuralInvalidV2 {
		t.Run(name, func(t *testing.T) {
			data, err := os.ReadFile(filepath.Join("..", "testdata", "invalid", name))
			if err != nil {
				t.Fatalf("read: %v", err)
			}
			if err := validateBytes(compiledV2, data); err == nil {
				t.Fatalf("invalid v2 fixture %s accepted by schema", name)
			}
		})
	}
}

// TestFixturesV2AcceptsSemanticInvalid asserts that the v2 schema
// accepts the fixtures the decoder accepts for handoff but the
// normalizer rejects.
func TestFixturesV2AcceptsSemanticInvalid(t *testing.T) {
	compileForTest(t)
	for _, entry := range semanticInvalidV2 {
		t.Run(entry.fixture, func(t *testing.T) {
			data, err := os.ReadFile(filepath.Join("..", "testdata", "invalid", entry.fixture))
			if err != nil {
				t.Fatalf("read: %v", err)
			}
			if err := validateBytes(compiledV2, data); err != nil {
				t.Fatalf("semantic-only invalid v2 fixture %s rejected by schema: %v\n"+
					"NOTE: schemas own structural wire shape; semantics are owned by the normalizer.",
					entry.fixture, err)
			}
		})
	}
}

// TestFixturesV2AcceptsPreSchemaInvalid asserts that the v2 schema
// accepts fixtures the pre-schema envelope scanner rejects before
// the schema is invoked.
func TestFixturesV2AcceptsPreSchemaInvalid(t *testing.T) {
	compileForTest(t)
	for _, name := range invalidV2CapturedByPreSchemaEnvelope {
		t.Run(name, func(t *testing.T) {
			// v2-truncated.json is malformed JSON. The JSON decoder
			// fails before the schema validator is invoked, so the
			// schema outcome is not-applicable. The pre-schema envelope
			// scanner rejects this fixture with CodeMalformedJSON per
			// the existing corpus tests in internal/gatesummary
			// (corpus_test.go asserts v2-truncated.json surfaces
			// CodeMalformedJSON). The schema package cannot directly
			// import the parent package, so the binding is documented
			// here and verified by the parent package's corpus tests.
			if name == "v2-truncated.json" {
				t.Skip("v2-truncated.json is malformed JSON; pre-schema envelope rejects with CodeMalformedJSON (see internal/gatesummary/corpus_test.go)")
			}
			data, err := os.ReadFile(filepath.Join("..", "testdata", "invalid", name))
			if err != nil {
				t.Fatalf("read: %v", err)
			}
			if err := validateBytes(compiledV2, data); err != nil {
				t.Fatalf("pre-schema-invalid v2 fixture %s rejected by schema: %v", name, err)
			}
		})
	}
}

// TestFixtureMatrixComplete asserts that every fixture file referenced
// in the structuralInvalid, semanticInvalid, or pre-schema-invalid
// tables actually exists.
func TestFixtureMatrixComplete(t *testing.T) {
	invalidDir := filepath.Join("..", "testdata", "invalid")

	all := append([]string{}, structuralInvalidV1...)
	all = append(all, structuralInvalidV2...)
	all = append(all, invalidV2CapturedByPreSchemaEnvelope...)
	for _, entry := range semanticInvalidV2 {
		all = append(all, entry.fixture)
	}

	seen := map[string]bool{}
	for _, name := range all {
		if seen[name] {
			t.Errorf("fixture %s listed twice in matrix", name)
		}
		seen[name] = true
		path := filepath.Join(invalidDir, name)
		if _, err := os.Stat(path); err != nil {
			t.Errorf("matrix references missing fixture %s: %v", name, err)
		}
	}
}

// newV2WithExitCodeRaw builds a minimal v2 document with the given
// exit_code value as a raw json.Number.
func newV2WithExitCodeRaw(exitCode json.Number) []byte {
	doc := []byte(fmt.Sprintf(`{
		"schema_version": 2,
		"generated_at": "2026-07-19T08:43:26Z",
		"scope_id": "ACT-TEST",
		"scope_status": "CLOSED",
		"scope_disposition": "test",
		"parent_act": "",
		"parent_status": "CLOSED",
		"parent_disposition": "root",
		"overall_status": "pass",
		"overall_disposition": "all good",
		"execution_head_oid": "0123456789abcdef0123456789abcdef01234567",
		"execution_tree_oid": "0123456789abcdef0123456789abcdef01234567",
		"subject_tree_oid": "0123456789abcdef0123456789abcdef01234567",
		"worktree_clean_before": true,
		"worktree_clean_after": true,
		"checks": [
			{
				"name": "test",
				"scope": "ROOT",
				"status": "pass",
				"evidence": "e",
				"detail": "d",
				"extras": {
					"argv": [],
					"exit_code": %s,
					"duration_ms": 0,
					"stdout_sha256": "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
					"stderr_sha256": "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
				}
			}
		]
	}`, exitCode))
	return doc
}

// validateBytes validates the given JSON byte slice against a
// compiled schema.
func validateBytes(sch *jsonschema.Schema, data []byte) error {
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.UseNumber()
	var v any
	if err := dec.Decode(&v); err != nil {
		return err
	}
	return sch.Validate(v)
}

// TestFixtureMatrixComplete asserts that every invalid fixture is
// classified in exactly one closed-set bucket. The fixture corpus

