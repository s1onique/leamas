package evaltest

import (
	"bytes"
	"encoding/json"
	"fmt"
)

// DialectConfig carries the dialect identifiers the
// extension-aware evaluator must enforce. The closure package
// owns the canonical values (ClosurePlanV1MetaSchemaURI,
// ClosurePlanV1MetaSchemaBytes); the evaltest package never
// hardcodes them.
//
// The configuration is supplied once at test setup time so the
// extension-aware evaluator never depends on the closure
// package directly. Keeping the dependency direction
// closure → evaltest prevents a circular import.
type DialectConfig struct {
	// DialectURI is the value the extension-aware evaluator
	// expects to find in the schema's "$schema" property.
	DialectURI string
	// MetaSchemaID is the value the evaluator expects to find
	// in the resolved meta-schema's "$id" property.
	MetaSchemaID string
	// MetaSchemaBytes is the raw JSON document of the resolved
	// meta-schema. The evaluator parses it once and caches the
	// result.
	MetaSchemaBytes []byte
	// RequiredVocabularies is the closed set of vocabulary URIs
	// the resolved meta-schema MUST mark as required=true. The
	// Leamas Closure Plan v1 vocabulary is always required.
	RequiredVocabularies []string
}

// activeDialect is the configured dialect for the duration of
// the test binary. ConfigureDialect sets it; ensureDialectAdmitted
// reads it. The zero value (no configuration) makes the
// extension-aware evaluator fail closed on every schema so a
// test that forgot to call ConfigureDialect cannot silently
// accept any schema.
var activeDialect DialectConfig

// ConfigureDialect installs the dialect configuration for the
// current process. Tests call this from TestMain; production
// never reaches this package. The function is not safe for
// concurrent use with parallel evaluator calls; tests should
// call it once at startup before any parallel evaluator use.
func ConfigureDialect(cfg DialectConfig) {
	activeDialect = cfg
}

// ResetDialect clears any previously configured dialect. Tests
// that need a clean dialect state (e.g. dialect-mismatch
// fixtures) call this between cases.
func ResetDialect() {
	activeDialect = DialectConfig{}
}

// ensureDialectAdmitted validates the schema's $schema against
// the configured Leamas dialect, the resolved meta-schema's
// $id, and the required vocabulary set. It returns a non-nil
// error when any check fails; the extension-aware evaluator
// must treat that error as a schema-level rejection.
func ensureDialectAdmitted(schema map[string]any) error {
	if activeDialect.DialectURI == "" {
		return fmt.Errorf("schema dialect: no dialect configured; call evaltest.ConfigureDialect before using EvaluateWithSchemaExtensionAware")
	}
	rawSchema, ok := schema["$schema"]
	if !ok {
		return fmt.Errorf("schema dialect: $schema missing")
	}
	gotSchema, ok := rawSchema.(string)
	if !ok {
		return fmt.Errorf("schema dialect: $schema wrong type %T", rawSchema)
	}
	if gotSchema != activeDialect.DialectURI {
		return fmt.Errorf("schema dialect: unknown dialect URI %q (want %q)", gotSchema, activeDialect.DialectURI)
	}
	metaSchema, err := parseMetaSchema(activeDialect.MetaSchemaBytes)
	if err != nil {
		return fmt.Errorf("schema dialect: meta-schema unparseable: %w", err)
	}
	rawID, ok := metaSchema["$id"]
	if !ok {
		return fmt.Errorf("schema dialect: resolved meta-schema $id missing")
	}
	gotID, ok := rawID.(string)
	if !ok {
		return fmt.Errorf("schema dialect: resolved meta-schema $id wrong type %T", rawID)
	}
	if gotID != activeDialect.MetaSchemaID {
		return fmt.Errorf("schema dialect: resolved meta-schema $id mismatch (got %q, want %q)", gotID, activeDialect.MetaSchemaID)
	}
	vocabRaw, ok := metaSchema["$vocabulary"]
	if !ok {
		return fmt.Errorf("schema dialect: resolved meta-schema $vocabulary missing")
	}
	vocabMap, ok := vocabRaw.(map[string]any)
	if !ok {
		return fmt.Errorf("schema dialect: resolved meta-schema $vocabulary wrong type %T", vocabRaw)
	}
	for _, reqURI := range activeDialect.RequiredVocabularies {
		entry, present := vocabMap[reqURI]
		if !present {
			return fmt.Errorf("schema dialect: required vocabulary %q absent from meta-schema", reqURI)
		}
		enabled, ok := entry.(bool)
		if !ok {
			return fmt.Errorf("schema dialect: required vocabulary %q has wrong type %T", reqURI, entry)
		}
		if !enabled {
			return fmt.Errorf("schema dialect: required vocabulary %q is not enabled", reqURI)
		}
	}
	return nil
}

// parseMetaSchema decodes the meta-schema bytes into a
// map[string]any for dialect checks. json.Decoder with
// UseNumber is used so vocabulary URIs remain stable and the
// evaluator never sees float64 numeric coercion.
func parseMetaSchema(raw []byte) (map[string]any, error) {
	if len(raw) == 0 {
		return nil, fmt.Errorf("meta-schema bytes are empty")
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	var out map[string]any
	if err := dec.Decode(&out); err != nil {
		return nil, err
	}
	return out, nil
}
