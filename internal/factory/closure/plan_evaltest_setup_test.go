package closure

import (
	"os"
	"testing"

	"github.com/s1onique/leamas/internal/factory/closure/evaltest"
)

// TestMain installs the Leamas Closure Plan v1 dialect into the
// extension-aware evaluator before any test runs. The dialect
// resolution is fail-closed: tests that forget to call
// evaltest.ConfigureDialect see every extension-aware evaluation
// rejected, which is the safer default than silently accepting
// unknown dialects.
//
// ACT-LEAMAS-FACTORY-CLOSE-PLAN-V1-RUN-EXECUTION-FIELDS-
// CONTRACT-PARITY01-CORRECTION16 placed the schema evaluator in
// the internal evaltest package. The closure package remains the
// canonical owner of the dialect identifiers and the embedded
// meta-schema bytes; this TestMain injects them into the
// evaluator at startup so the four-layer parity tests can run.
func TestMain(m *testing.M) {
	cfg := evaltest.DialectConfig{
		DialectURI:   ClosurePlanV1MetaSchemaURI(),
		MetaSchemaID: ClosurePlanV1MetaSchemaURI(),
		// MetaSchemaID is the same as DialectURI for the Closure
		// Plan v1 dialect because the dialect URI IS the meta-schema
		// ID (the meta-schema declares its $id as the same URI).
		MetaSchemaBytes: ClosurePlanV1MetaSchemaBytes(),
		RequiredVocabularies: []string{
			leamasVocabularyURI,
		},
	}
	evaltest.ConfigureDialect(cfg)
	code := m.Run()
	os.Exit(code)
}
