package evaltest

import (
	"os"
	"testing"
)

// testDialectURI is the canonical dialect URI used by the
// evaltest package's helper dialect. Tests that build synthetic
// schemas reference this URI so the dialect resolution step
// passes when running in isolation.
const testDialectURI = "https://leamas.io/closure-plan/v1/meta.json"

// testDialectConfig returns a permissive dialect
// configuration that admits any schema carrying the
// testDialectURI. The configuration declares the canonical
// Leamas vocabulary so vocabulary enforcement succeeds.
func testDialectConfig() DialectConfig {
	return DialectConfig{
		DialectURI:   testDialectURI,
		MetaSchemaID: testDialectURI,
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
	}
}

// TestMain installs the test dialect configuration before any
// subtest runs. The dialect is global state but is reset at
// the end of every test that mutates it (TestDialectUnconfigured,
// TestVocabularyFailures, TestMetaSchemaIDMismatch,
// TestDialectResolutionFailClosed) so this baseline
// configuration is restored between runs.
func TestMain(m *testing.M) {
	ConfigureDialect(testDialectConfig())
	code := m.Run()
	ResetDialect()
	os.Exit(code)
}
