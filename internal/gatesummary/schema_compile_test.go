package gatesummary

import (
	"strings"
	"testing"
)

func TestSchemaBootstrap(t *testing.T) {
	set, err := schemas()
	if err != nil {
		t.Fatalf("schemas bootstrap failed: %v", err)
	}
	if set.v1 == nil {
		t.Fatal("v1 schema not compiled")
	}
	if set.v2 == nil {
		t.Fatal("v2 schema not compiled")
	}
}

func TestSchemaBootstrapConcurrent(t *testing.T) {
	const goroutines = 32
	done := make(chan error, goroutines)
	for i := 0; i < goroutines; i++ {
		go func() {
			_, err := schemas()
			done <- err
		}()
	}
	for i := 0; i < goroutines; i++ {
		if err := <-done; err != nil {
			t.Errorf("concurrent bootstrap: %v", err)
		}
	}
}

func TestFailClosedLoader(t *testing.T) {
	loader := failClosedLoader{}
	if _, err := loader.Load("https://example.com/schema.json"); err == nil {
		t.Fatal("expected fail-closed error for unknown URL")
	}
}

func TestSchemaEmbeddedV1ContainsExpectedFields(t *testing.T) {
	if !strings.Contains(string(v1SchemaJSON), `"const": 1`) {
		t.Fatalf("v1 schema missing const=1 discriminator")
	}
	if !strings.Contains(string(v1SchemaJSON), `"schema_version"`) {
		t.Fatalf("v1 schema missing schema_version field")
	}
}

func TestSchemaEmbeddedV2ContainsExpectedFields(t *testing.T) {
	if !strings.Contains(string(v2SchemaJSON), `"const": 2`) {
		t.Fatalf("v2 schema missing const=2 discriminator")
	}
	if !strings.Contains(string(v2SchemaJSON), `"execution_head_oid"`) {
		t.Fatalf("v2 schema missing execution_head_oid field")
	}
}
