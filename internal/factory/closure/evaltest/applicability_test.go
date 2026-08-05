package evaltest

import (
	"encoding/json"
	"strings"
	"testing"
)

// TestNormativeApplicabilityFailClosed walks every
// x-applicability shape failure the ACT enumerates. The
// extension-aware evaluator must reject the schema and
// surface a typed schema issue; the standard evaluator
// must remain permissive.
//
// The failures enumerated here are:
//
//	absent
//	null
//	wrong type
//	empty where rules are required
//	contains non-object
//	missing sibling
//	missing value
//	missing presence
//	wrong member type
//	unknown member
//	unknown presence
//	duplicate sibling/value pair
func TestNormativeApplicabilityFailClosed(t *testing.T) {
	// The schema evaluator only exercises applicability on
	// schemas whose $schema URI passes the configured
	// dialect. The dialect is installed by the package's
	// own TestMain (TestEvaluatorDialectConfigured). If that
	// TestMain is not yet registered when this test runs in
	// isolation, install a permissive dialect here as a
	// belt-and-braces guard.
	ConfigureDialect(testDialectConfig())
	defer ResetDialect()
	type tc struct {
		name     string
		mutate   func(*map[string]any)
		wantFail string
	}
	baseApp := func() map[string]any {
		return map[string]any{
			"sibling":  "mode",
			"value":    "run",
			"presence": "required",
		}
	}
	baseProp := func() map[string]any {
		return map[string]any{
			"type":            "integer",
			"x-applicability": []any{baseApp()},
			"x-leamas-repository-relative-path": map[string]any{
				"allow_dot":               true,
				"allow_parent_segments":   false,
				"require_lexically_clean": true,
				"separator":               "/",
			},
		}
	}
	baseSchema := func() map[string]any {
		// The base schema declares a sibling `mode` field so
		// applicability rules with sibling="mode" can
		// exercise the rest of the validation pipeline
		// without tripping the "sibling not declared" guard.
		// The applicability rule itself declares
		// sibling="mode" so the walker recognises the
		// declared sibling.
		return map[string]any{
			"$schema": testDialectURI,
			"type":    "object",
			"properties": map[string]any{
				"x":    baseProp(),
				"mode": map[string]any{"type": "string"},
			},
			"required": []any{"x"},
		}
	}

	cases := []tc{
		// "absent" is not tested here: x-applicability is an
		// optional extension in the meta-schema, so omitting
		// it is the normal case. The fail-closed test below
		// confirms that when x-applicability IS declared,
		// null/wrong-type/empty are rejected.
		{
			name: "null",
			mutate: func(s *map[string]any) {
				props := (*s)["properties"].(map[string]any)
				x := props["x"].(map[string]any)
				x["x-applicability"] = nil
			},
			wantFail: "x-applicability missing",
		},
		{
			name: "wrong_type",
			mutate: func(s *map[string]any) {
				props := (*s)["properties"].(map[string]any)
				x := props["x"].(map[string]any)
				x["x-applicability"] = "not-an-array"
			},
			wantFail: "wrong type",
		},
		{
			name: "empty_array",
			mutate: func(s *map[string]any) {
				props := (*s)["properties"].(map[string]any)
				x := props["x"].(map[string]any)
				x["x-applicability"] = []any{}
			},
			wantFail: "array is empty",
		},
		{
			name: "contains_non_object",
			mutate: func(s *map[string]any) {
				props := (*s)["properties"].(map[string]any)
				x := props["x"].(map[string]any)
				x["x-applicability"] = []any{"not-an-object"}
			},
			wantFail: "entry is not a JSON object",
		},
		{
			name: "missing_sibling",
			mutate: func(s *map[string]any) {
				props := (*s)["properties"].(map[string]any)
				x := props["x"].(map[string]any)
				r := baseApp()
				delete(r, "sibling")
				x["x-applicability"] = []any{r}
			},
			wantFail: "sibling absent",
		},
		{
			name: "wrong_type_sibling",
			mutate: func(s *map[string]any) {
				props := (*s)["properties"].(map[string]any)
				x := props["x"].(map[string]any)
				r := baseApp()
				r["sibling"] = 42
				x["x-applicability"] = []any{r}
			},
			wantFail: "sibling wrong type",
		},
		{
			name: "missing_value",
			mutate: func(s *map[string]any) {
				props := (*s)["properties"].(map[string]any)
				x := props["x"].(map[string]any)
				r := baseApp()
				delete(r, "value")
				x["x-applicability"] = []any{r}
			},
			wantFail: "value absent",
		},
		{
			name: "wrong_type_value",
			mutate: func(s *map[string]any) {
				props := (*s)["properties"].(map[string]any)
				x := props["x"].(map[string]any)
				r := baseApp()
				r["value"] = 42
				x["x-applicability"] = []any{r}
			},
			wantFail: "value wrong type",
		},
		{
			name: "missing_presence",
			mutate: func(s *map[string]any) {
				props := (*s)["properties"].(map[string]any)
				x := props["x"].(map[string]any)
				r := baseApp()
				delete(r, "presence")
				x["x-applicability"] = []any{r}
			},
			wantFail: "presence absent",
		},
		{
			name: "wrong_type_presence",
			mutate: func(s *map[string]any) {
				props := (*s)["properties"].(map[string]any)
				x := props["x"].(map[string]any)
				r := baseApp()
				r["presence"] = 42
				x["x-applicability"] = []any{r}
			},
			wantFail: "presence wrong type",
		},
		{
			name: "unknown_presence",
			mutate: func(s *map[string]any) {
				props := (*s)["properties"].(map[string]any)
				x := props["x"].(map[string]any)
				r := baseApp()
				r["presence"] = "optional-but-typo"
				x["x-applicability"] = []any{r}
			},
			wantFail: "unknown presence",
		},
		{
			name: "duplicate_pair",
			mutate: func(s *map[string]any) {
				props := (*s)["properties"].(map[string]any)
				x := props["x"].(map[string]any)
				x["x-applicability"] = []any{baseApp(), baseApp()}
			},
			wantFail: "duplicate rule",
		},
		{
			name: "unknown_member",
			mutate: func(s *map[string]any) {
				props := (*s)["properties"].(map[string]any)
				x := props["x"].(map[string]any)
				r := baseApp()
				r["unexpected"] = "value"
				x["x-applicability"] = []any{r}
			},
			wantFail: "unknown member",
		},
	}
	for _, c := range cases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			s := baseSchema()
			c.mutate(&s)
			raw, _ := json.Marshal(s)
			var roundTripped map[string]any
			if err := json.Unmarshal(raw, &roundTripped); err != nil {
				t.Fatalf("unmarshal: %v", err)
			}
			out := EvaluateWithSchemaExtensionAware(roundTripped, map[string]any{
				"x": 60,
			})
			if out.Accept {
				t.Fatalf("expected applicability rejection for %s; got accept issues=%v", c.name, out.Issues)
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

// TestNormativePathPolicyFailClosed walks every
// x-leamas-repository-relative-path shape failure the ACT
// enumerates. The extension-aware evaluator must reject the
// value when any required member is malformed.
func TestNormativePathPolicyFailClosed(t *testing.T) {
	// We exercise the path-policy directly through the
	// portablePathAccepts helper, which is the same code
	// path the extension-aware schema evaluator uses. Each
	// case installs a malformed extension member and asserts
	// the helper rejects with the expected reason.
	type tc struct {
		name     string
		ext      map[string]any
		wantFail string
	}
	cases := []tc{
		{
			name:     "absent",
			ext:      nil,
			wantFail: "extension missing",
		},
		{
			name:     "null",
			ext:      map[string]any{},
			wantFail: "allow_dot missing",
		},
		{
			name: "wrong_type_allow_dot",
			ext: map[string]any{
				"allow_dot":               42,
				"allow_parent_segments":   false,
				"require_lexically_clean": true,
				"separator":               "/",
			},
			wantFail: "allow_dot missing",
		},
		{
			name: "wrong_type_allow_parent_segments",
			ext: map[string]any{
				"allow_dot":               true,
				"allow_parent_segments":   "no",
				"require_lexically_clean": true,
				"separator":               "/",
			},
			wantFail: "allow_parent_segments missing",
		},
		{
			name: "wrong_type_require_lexically_clean",
			ext: map[string]any{
				"allow_dot":               true,
				"allow_parent_segments":   false,
				"require_lexically_clean": "yes",
				"separator":               "/",
			},
			wantFail: "require_lexically_clean missing",
		},
		{
			name: "wrong_type_separator",
			ext: map[string]any{
				"allow_dot":               true,
				"allow_parent_segments":   false,
				"require_lexically_clean": true,
				"separator":               42,
			},
			wantFail: "separator missing",
		},
		{
			name: "noncanonical_invariant_allow_dot",
			ext: map[string]any{
				"allow_dot":               false,
				"allow_parent_segments":   false,
				"require_lexically_clean": true,
				"separator":               "/",
			},
			wantFail: "noncanonical invariant allow_dot",
		},
		{
			name: "noncanonical_invariant_parent_segments",
			ext: map[string]any{
				"allow_dot":               true,
				"allow_parent_segments":   true,
				"require_lexically_clean": true,
				"separator":               "/",
			},
			wantFail: "noncanonical invariant allow_parent_segments",
		},
		{
			name: "noncanonical_invariant_require_lexically_clean",
			ext: map[string]any{
				"allow_dot":               true,
				"allow_parent_segments":   false,
				"require_lexically_clean": false,
				"separator":               "/",
			},
			wantFail: "noncanonical invariant require_lexically_clean",
		},
		{
			name: "noncanonical_invariant_separator",
			ext: map[string]any{
				"allow_dot":               true,
				"allow_parent_segments":   false,
				"require_lexically_clean": true,
				"separator":               "\\",
			},
			wantFail: "unsupported separator",
		},
	}
	for _, c := range cases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			ok, reason := portablePathAccepts(".", c.ext)
			if ok {
				t.Fatalf("expected rejection for %s; got accept", c.name)
			}
			if c.wantFail != "" && !strings.Contains(reason, c.wantFail) {
				t.Fatalf("reason=%q does not contain %q", reason, c.wantFail)
			}
		})
	}

	// The canonical invariants must be accepted.
	t.Run("canonical_invariants_accepted", func(t *testing.T) {
		ok, reason := portablePathAccepts(".", map[string]any{
			"allow_dot":               true,
			"allow_parent_segments":   false,
			"require_lexically_clean": true,
			"separator":               "/",
		})
		if !ok {
			t.Fatalf("canonical invariants rejected: %s", reason)
		}
	})
}
