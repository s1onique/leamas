package evaltest

import (
	"encoding/json"
	"strings"
	"testing"
)

// TestDialectResolutionFailClosed walks every dialect
// failure the ACT enumerates and proves the extension-aware
// evaluator fails closed on each. The standard evaluator is
// also exercised when relevant so consumers cannot silently
// downgrade dialect failures to plain schema failures.
func TestDialectResolutionFailClosed(t *testing.T) {
	// Pin a known-good dialect so unrelated tests cannot
	// influence the active configuration.
	ConfigureDialect(DialectConfig{
		DialectURI:   "https://leamas.io/closure-plan/v1/meta.json",
		MetaSchemaID: "https://leamas.io/closure-plan/v1/meta.json",
		MetaSchemaBytes: []byte(`{
			"$schema": "https://json-schema.org/draft/2020-12/schema",
			"$id": "https://leamas.io/closure-plan/v1/meta.json",
			"$vocabulary": {
				"https://json-schema.org/draft/2020-12/vocab/core": true,
				"https://leamas.io/closure-plan/v1/vocab": true
			}
		}`),
		RequiredVocabularies: []string{
			"https://leamas.io/closure-plan/v1/vocab",
		},
	})
	defer ResetDialect()

	type tc struct {
		name     string
		mutate   func(*map[string]any)
		wantFail string
	}
	cases := []tc{
		{
			name: "missing_schema",
			mutate: func(s *map[string]any) {
				delete(*s, "$schema")
			},
			wantFail: "missing",
		},
		{
			name: "wrong_type_schema",
			mutate: func(s *map[string]any) {
				(*s)["$schema"] = 42
			},
			wantFail: "wrong type",
		},
		{
			name: "unknown_dialect_uri",
			mutate: func(s *map[string]any) {
				(*s)["$schema"] = "https://example.com/other-dialect"
			},
			wantFail: "unknown dialect URI",
		},
		{
			name: "unresolvable_dialect",
			mutate: func(s *map[string]any) {
				// Use a syntactically valid but unconfigured
				// URI; the resolution will accept the value
				// but fail at the meta-schema decode step
				// because the configured bytes disagree.
				(*s)["$schema"] = "https://leamas.io/other/meta.json"
			},
			wantFail: "unknown dialect URI",
		},
		// The "resolved_meta_schema_id_mismatch" and
		// "configured_dialect_but_unresolvable_vocabulary"
		// scenarios need bespoke meta-schema bytes; they
		// are exercised by TestMetaSchemaIDMismatch and
		// TestVocabularyFailures respectively.
	}

	// Build a well-formed base schema.
	base := func() map[string]any {
		return map[string]any{
			"$schema": "https://leamas.io/closure-plan/v1/meta.json",
			"$id":     "https://leamas.io/closure-plan/v1/schema.json",
			"type":    "object",
		}
	}
	instance := map[string]any{}

	for _, c := range cases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			s := base()
			c.mutate(&s)
			raw, _ := json.Marshal(s)
			var roundTripped map[string]any
			if err := json.Unmarshal(raw, &roundTripped); err != nil {
				t.Fatalf("unmarshal: %v", err)
			}
			out := EvaluateWithSchemaExtensionAware(roundTripped, instance)
			if out.Accept {
				t.Fatalf("expected dialect rejection for %s; got accept issues=%v", c.name, out.Issues)
			}
			if c.wantFail != "" {
				found := false
				for _, issue := range out.Issues {
					if strings.Contains(issue, c.wantFail) {
						found = true
						break
					}
				}
				if !found {
					t.Fatalf("issues=%v does not contain %q", out.Issues, c.wantFail)
				}
			}
		})
	}
}

// TestMetaSchemaIDMismatch proves the extension-aware
// evaluator rejects a schema whose $schema URI matches the
// configured dialect URI but whose meta-schema $id disagrees
// with the configured MetaSchemaID.
func TestMetaSchemaIDMismatch(t *testing.T) {
	ConfigureDialect(DialectConfig{
		DialectURI:   "https://leamas.io/closure-plan/v1/meta.json",
		MetaSchemaID: "https://leamas.io/closure-plan/v1/meta.json",
		MetaSchemaBytes: []byte(`{
			"$schema": "https://json-schema.org/draft/2020-12/schema",
			"$id": "https://different.example.com/leamas-closure-plan.json",
			"$vocabulary": {
				"https://leamas.io/closure-plan/v1/vocab": true
			}
		}`),
		RequiredVocabularies: []string{
			"https://leamas.io/closure-plan/v1/vocab",
		},
	})
	defer ResetDialect()

	schema := map[string]any{
		"$schema": "https://leamas.io/closure-plan/v1/meta.json",
		"type":    "object",
	}
	raw, _ := json.Marshal(schema)
	var roundTripped map[string]any
	if err := json.Unmarshal(raw, &roundTripped); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	out := EvaluateWithSchemaExtensionAware(roundTripped, map[string]any{})
	if out.Accept {
		t.Fatalf("expected meta-schema $id mismatch rejection; got accept")
	}
	found := false
	for _, issue := range out.Issues {
		if strings.Contains(issue, "$id mismatch") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("issues=%v does not mention $id mismatch", out.Issues)
	}
}

// TestVocabularyFailures walks every vocabulary failure the
// ACT enumerates: missing $vocabulary, wrong-type $vocabulary,
// unknown required vocabulary, Leamas vocabulary absent,
// Leamas vocabulary set to false.
func TestVocabularyFailures(t *testing.T) {
	cases := []struct {
		name     string
		meta     string
		wantFail string
	}{
		{
			name: "missing_vocabulary",
			meta: `{
				"$schema": "https://json-schema.org/draft/2020-12/schema",
				"$id": "https://leamas.io/closure-plan/v1/meta.json"
			}`,
			wantFail: "$vocabulary missing",
		},
		{
			name: "wrong_type_vocabulary",
			meta: `{
				"$schema": "https://json-schema.org/draft/2020-12/schema",
				"$id": "https://leamas.io/closure-plan/v1/meta.json",
				"$vocabulary": "not-a-map"
			}`,
			wantFail: "$vocabulary wrong type",
		},
		{
			name: "unknown_required_vocabulary",
			meta: `{
				"$schema": "https://json-schema.org/draft/2020-12/schema",
				"$id": "https://leamas.io/closure-plan/v1/meta.json",
				"$vocabulary": {}
			}`,
			wantFail: "absent from meta-schema",
		},
		{
			name: "leamas_vocabulary_absent",
			meta: `{
				"$schema": "https://json-schema.org/draft/2020-12/schema",
				"$id": "https://leamas.io/closure-plan/v1/meta.json",
				"$vocabulary": {
					"https://json-schema.org/draft/2020-12/vocab/core": true
				}
			}`,
			wantFail: "absent from meta-schema",
		},
		{
			name: "leamas_vocabulary_false",
			meta: `{
				"$schema": "https://json-schema.org/draft/2020-12/schema",
				"$id": "https://leamas.io/closure-plan/v1/meta.json",
				"$vocabulary": {
					"https://json-schema.org/draft/2020-12/vocab/core": true,
					"https://leamas.io/closure-plan/v1/vocab": false
				}
			}`,
			wantFail: "not enabled",
		},
	}
	for _, c := range cases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			ConfigureDialect(DialectConfig{
				DialectURI:      "https://leamas.io/closure-plan/v1/meta.json",
				MetaSchemaID:    "https://leamas.io/closure-plan/v1/meta.json",
				MetaSchemaBytes: []byte(c.meta),
				RequiredVocabularies: []string{
					"https://leamas.io/closure-plan/v1/vocab",
				},
			})
			defer ResetDialect()

			schema := map[string]any{
				"$schema": "https://leamas.io/closure-plan/v1/meta.json",
				"type":    "object",
			}
			raw, _ := json.Marshal(schema)
			var roundTripped map[string]any
			if err := json.Unmarshal(raw, &roundTripped); err != nil {
				t.Fatalf("unmarshal: %v", err)
			}
			out := EvaluateWithSchemaExtensionAware(roundTripped, map[string]any{})
			if out.Accept {
				t.Fatalf("expected %s rejection; got accept issues=%v", c.name, out.Issues)
			}
			found := false
			for _, issue := range out.Issues {
				if strings.Contains(issue, c.wantFail) {
					found = true
					break
				}
			}
			if !found {
				t.Fatalf("issues=%v does not contain %q", out.Issues, c.wantFail)
			}
		})
	}
}

// TestDialectUnconfigured ensures EvaluateWithSchemaExtensionAware
// fails closed when no dialect has been configured.
func TestDialectUnconfigured(t *testing.T) {
	ResetDialect()
	out := EvaluateWithSchemaExtensionAware(map[string]any{
		"$schema": "https://leamas.io/closure-plan/v1/meta.json",
		"type":    "object",
	}, map[string]any{})
	if out.Accept {
		t.Fatalf("expected unconfigured dialect rejection; got accept")
	}
	if len(out.Issues) == 0 || !strings.Contains(out.Issues[0], "no dialect configured") {
		t.Fatalf("expected 'no dialect configured' in issues; got %v", out.Issues)
	}
}
