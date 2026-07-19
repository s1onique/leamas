package gatesummary

import (
	"slices"
	"testing"

	jsonschema "github.com/santhosh-tekuri/jsonschema/v6"
	"github.com/santhosh-tekuri/jsonschema/v6/kind"
)

func TestSchemaErrorRequiresRootSchemaWrapper(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		root *jsonschema.ValidationError
	}{
		{name: "nil", root: nil},
		{name: "nil kind", root: &jsonschema.ValidationError{}},
		{name: "non-schema root", root: validationLeaf(&kind.Type{}, nil, v2SchemaID+"#")},
		{name: "empty schema root", root: validationRoot()},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assertDiagnostic(t, translateValidationTree(tc.root), CodeInternal, "/")
		})
	}
}

func TestSchemaErrorTypedNilKindIsInternalWithoutPanic(t *testing.T) {
	t.Parallel()

	var nilRequired *kind.Required
	var nilReference *kind.Reference
	cases := []struct {
		name string
		kind jsonschema.ErrorKind
	}{
		{name: "required", kind: nilRequired},
		{name: "reference", kind: nilReference},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			node := validationLeaf(tc.kind, nil, v2SchemaID+"#")
			assertDiagnostic(t, translateValidationTree(validationRoot(node)),
				CodeInternal, "/")
		})
	}
}

func TestEmptyNestedWrappersAreSchemaViolations(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		kind jsonschema.ErrorKind
	}{
		{name: "schema", kind: &kind.Schema{}},
		{name: "group", kind: &kind.Group{}},
		{name: "reference", kind: &kind.Reference{}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			node := validationLeaf(tc.kind, []string{"checks", "0"}, v2SchemaID+"#/$defs/check")
			assertDiagnostic(t, translateValidationTree(validationRoot(node)),
				CodeSchemaViolation, "/checks/0")
		})
	}
}

func TestTestTotalAnyOfUsesExactStructuredKeywordIdentity(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name      string
		schemaURL string
		wantCode  string
	}{
		{name: "canonical", schemaURL: v2SchemaID + "#/$defs/check", wantCode: CodePartialTestTotals},
		{name: "percent decoded fragment", schemaURL: v2SchemaID + "#/%24defs/check",
			wantCode: CodePartialTestTotals},
		{name: "suffix spoof", schemaURL: "https://example.invalid/gate-summary-v2.schema.json#/$defs/check",
			wantCode: CodeSchemaViolation},
		{name: "wrong object", schemaURL: v2SchemaID + "#/$defs/other",
			wantCode: CodeSchemaViolation},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			node := validationLeaf(&kind.AnyOf{}, []string{"checks", "7"}, tc.schemaURL)
			node.Causes = []*jsonschema.ValidationError{
				validationLeaf(&kind.Required{Missing: []string{"total"}},
					[]string{"checks", "7"}, tc.schemaURL+"/anyOf/0"),
				validationLeaf(&kind.Not{}, []string{"checks", "7"}, tc.schemaURL+"/anyOf/1"),
			}
			ds := translateValidationTree(validationRoot(node))
			assertDiagnostic(t, ds, tc.wantCode, "/checks/7")
		})
	}
}

func TestKeywordIdentityParsesFragmentAndNot(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name       string
		schemaURL  string
		kind       jsonschema.ErrorKind
		wantBase   string
		wantTokens []string
	}{
		{name: "escaped pointer tokens", schemaURL: v2SchemaID + "#/%24defs/a~1b/m~0n",
			kind: &kind.Pattern{}, wantBase: v2SchemaID,
			wantTokens: []string{"$defs", "a/b", "m~n", "pattern"}},
		{name: "not appends missing keyword path", schemaURL: v2SchemaID + "#/$defs/check/anyOf/1",
			kind: &kind.Not{}, wantBase: v2SchemaID,
			wantTokens: []string{"$defs", "check", "anyOf", "1", "not"}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			base, tokens := keywordIdentity(validationLeaf(tc.kind, nil, tc.schemaURL))
			if base != tc.wantBase || !slices.Equal(tokens, tc.wantTokens) {
				t.Fatalf("identity=(%q,%q), want (%q,%q)",
					base, tokens, tc.wantBase, tc.wantTokens)
			}
		})
	}
}

func TestPartialTotalsOwnsAndSuppressesCauseSubtree(t *testing.T) {
	t.Parallel()

	anyOf := validationLeaf(&kind.AnyOf{}, []string{"checks", "0"}, v2SchemaID+"#/$defs/check")
	anyOf.Causes = []*jsonschema.ValidationError{
		validationLeaf(&kind.Required{Missing: []string{"pass_count", "total"}},
			[]string{"checks", "0"}, v2SchemaID+"#/$defs/check/anyOf/0"),
		validationLeaf(&kind.Not{}, []string{"checks", "0"}, v2SchemaID+"#/$defs/check/anyOf/1"),
	}
	assertDiagnostic(t, translateValidationTree(validationRoot(anyOf)),
		CodePartialTestTotals, "/checks/0")
}

func TestSchemaErrorTraversalPreservesWrapperCauseOrder(t *testing.T) {
	t.Parallel()

	group := validationLeaf(&kind.Group{}, nil, v2SchemaID+"#")
	group.Causes = []*jsonschema.ValidationError{
		validationLeaf(&kind.AdditionalProperties{Properties: []string{"z"}}, nil, v2SchemaID+"#"),
		validationLeaf(&kind.Required{Missing: []string{"a"}}, nil, v2SchemaID+"#"),
	}
	ds := translateValidationTree(validationRoot(group))
	if len(ds) != 2 || ds[0].Code != CodeUnknownField || ds[0].Path != "/z" ||
		ds[1].Code != CodeRequiredFieldMissing || ds[1].Path != "/a" {
		t.Fatalf("unexpected precedence/order result: %+v", ds)
	}
}

func TestSchemaErrorHostileNestingDoesNotPanic(t *testing.T) {
	t.Parallel()

	const depth = 20000
	var node *jsonschema.ValidationError = validationLeaf(
		&kind.AllOf{}, []string{"scope_id"}, v2SchemaID+"#/properties/scope_id")
	for i := 0; i < depth; i++ {
		wrapper := validationLeaf(&kind.Group{}, nil, v2SchemaID+"#")
		wrapper.Causes = []*jsonschema.ValidationError{node}
		node = wrapper
	}

	defer func() {
		if recovered := recover(); recovered != nil {
			t.Fatalf("schema translation panicked at depth %d: %v", depth, recovered)
		}
	}()
	assertDiagnostic(t, translateValidationTree(validationRoot(node)),
		CodeSchemaViolation, "/scope_id")
}
